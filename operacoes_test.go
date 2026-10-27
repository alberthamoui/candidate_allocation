package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xuri/excelize/v2"
)

func resultadoExemplo() *AlocacaoResponse {
	return &AlocacaoResponse{
		TotalAlocados: 3,
		Mesas: []MesaResult{
			{
				ID: 1, DiaID: 1, DiaNome: "quarta 14-16", Descricao: "quarta 14-16 - mesa 1",
				Candidatos: []CandidatoResultado{
					{ID: 1, Nome: "Ana", EmailInsper: "ana@al.insper.edu.br", Numero: "11999999999", Curso: "Eco"},
					{ID: 2, Nome: "Bia", EmailInsper: "bia@al.insper.edu.br", Numero: "11888888888"},
				},
				Avaliadores: []AvaliadorResultado{{ID: 10, Nome: "Prof"}},
			},
			{
				ID: 2, DiaID: 2, DiaNome: "quinta 10-12", Descricao: "quinta 10-12 - mesa 1",
				Candidatos: []CandidatoResultado{
					{ID: 3, Nome: "Caio", EmailPessoal: "caio@gmail.com", Numero: "11777777777"},
				},
			},
		},
		NaoAlocadosInfo: []CandidatoResultado{
			{ID: 4, Nome: "Duda", EmailInsper: "duda@al.insper.edu.br", Numero: "11666666666"},
		},
	}
}

func sessaoComResultado() *Session {
	s := setupSession()
	s.lastResult = resultadoExemplo()
	s.usarPreferencia = true
	s.avaliacoes = map[int]string{}
	for i := range s.lastResult.Mesas {
		for j := range s.lastResult.Mesas[i].Candidatos {
			c := &s.lastResult.Mesas[i].Candidatos[j]
			c.Telefone = c.Numero
		}
	}
	s.avisos = ConfigAvisos{Ativo: true, Mensagem: "Olá {{nome}}, sua entrevista é em {{data}} às {{horario}} na {{mesa}}.", Email: EmailConfig{Ativo: true, Host: "smtp.exemplo.com", Remetente: "org@exemplo.com"}, WhatsApp: WhatsConfig{Ativo: true, URL: "https://exemplo.test/zap"}}
	return s
}

func TestCPFOpcional(t *testing.T) {
	s := setupSession()
	err := s.SaveUsuarios([]Usuario{
		{Nome: "Ana", Semestre: "3", Opcoes: []string{"segunda 8-10"}},
		{Nome: "Bia", CPF: "   ", Semestre: "3", Opcoes: []string{"segunda 8-10"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := contar(t, s, "pessoa"); n != 2 {
		t.Fatalf("%d pessoas gravadas; esperado 2 sem CPF", n)
	}

	resultados, _ := processData([]Candidato{{Nome: "Ana", CPF: ""}, {Nome: "Bia", CPF: "123"}}, "@al.insper.edu.br")
	if temCampo(resultados[1].Erros, "cpf") {
		t.Errorf("CPF vazio deveria passar, erros: %+v", resultados[1].Erros)
	}
	if !temCampo(resultados[2].Erros, "cpf") {
		t.Errorf("CPF preenchido e curto deveria falhar, erros: %+v", resultados[2].Erros)
	}
}

func TestNumeroAceitaTelefoneERA(t *testing.T) {
	resultados, _ := processData([]Candidato{
		{Nome: "Ana", Numero: "(11) 98888-7777"},
		{Nome: "Bia", Numero: "123456789"},
		{Nome: "Caio", Numero: "123"},
	}, "@al.insper.edu.br")
	if temCampo(resultados[1].Erros, "numero") || resultados[1].Usuario.Numero != "11988887777" {
		t.Errorf("telefone formatado deveria valer, erros: %+v numero %q", resultados[1].Erros, resultados[1].Usuario.Numero)
	}
	if temCampo(resultados[2].Erros, "numero") || resultados[2].Usuario.Numero != "123456789" {
		t.Errorf("RA de 9 dígitos deveria valer, erros: %+v", resultados[2].Erros)
	}
	if !temCampo(resultados[3].Erros, "numero") {
		t.Errorf("número curto deveria falhar, erros: %+v", resultados[3].Erros)
	}
}

func temCampo(erros []ErrorEntry, campo string) bool {
	for _, e := range erros {
		if e.Field == campo {
			return true
		}
	}
	return false
}

func TestMensagemComVariaveis(t *testing.T) {
	msg := renderMensagem("Oi {{nome}}, dia {{data}} às {{horario}} ({{mesa}}).",
		CandidatoResultado{Nome: "Ana", EmailInsper: "ana@x.com"},
		MesaResult{DiaNome: "quarta 14-16", Descricao: "quarta 14-16 - mesa 1"})
	if msg != "Oi Ana, dia quarta às 14-16 (quarta 14-16 - mesa 1)." {
		t.Fatalf("mensagem: %q", msg)
	}
}

func TestRealocarDiaAvisa(t *testing.T) {
	s := sessaoComResultado()
	var emails, zaps []string
	s.avisos.enviarEmail = func(_ EmailConfig, para, _, corpo string) error {
		emails = append(emails, para+"|"+corpo)
		return nil
	}
	s.avisos.enviarWhats = func(_ WhatsConfig, para, corpo string) error {
		zaps = append(zaps, para+"|"+corpo)
		return nil
	}
	op, err := s.RealocarDia("Quarta 14-16", "sexta 9-11", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(emails) != 2 || len(zaps) != 2 {
		t.Fatalf("avisos: %d e-mails e %d whatsapps; esperado 2 e 2\n%+v", len(emails), len(zaps), op.Avisos)
	}
	if !strings.Contains(emails[0], "em sexta às 9-11") {
		t.Fatalf("e-mail não usou a data nova: %s", emails[0])
	}
	for _, m := range s.lastResult.Mesas {
		if m.ID == 1 && m.DiaNome != "sexta 9-11" {
			t.Fatalf("mesa 1 ficou em %q", m.DiaNome)
		}
		if m.ID == 2 && m.DiaNome != "quinta 10-12" {
			t.Fatalf("mesa 2 foi movida: %q", m.DiaNome)
		}
	}
}

func TestRealocarPessoaEAvaliar(t *testing.T) {
	s := sessaoComResultado()
	if _, err := s.RealocarPessoa("Duda", "quarta 14-16", 0, false); err != nil {
		t.Fatal(err)
	}
	if len(s.lastResult.NaoAlocadosInfo) != 0 {
		t.Fatalf("Duda ainda está sem mesa: %+v", s.lastResult.NaoAlocadosInfo)
	}
	if s.lastResult.TotalAlocados != 4 {
		t.Fatalf("total alocados %d; esperado 4", s.lastResult.TotalAlocados)
	}
	mesa := s.lastResult.Mesas[indiceMesa(s.lastResult, 1)]
	if len(mesa.Candidatos) != 3 {
		t.Fatalf("mesa 1 tem %d candidatos", len(mesa.Candidatos))
	}

	if _, err := s.RealocarPessoa("Ana", "sexta 8-10", 0, false); err != nil {
		t.Fatal(err)
	}
	if i := indiceMesaPorDia(s.lastResult, "sexta 8-10"); i < 0 || len(s.lastResult.Mesas[i].Candidatos) != 1 {
		t.Fatal("Ana não foi para uma mesa nova na sexta")
	}
	if _, err := s.Avaliar("Ana", 0, "passou"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Avaliar("", 3, "não passou"); err != nil {
		t.Fatal(err)
	}
	ana := acharNoResultado(s.lastResult, "Ana")
	caio := acharNoResultado(s.lastResult, "Caio")
	if ana.Avaliacao != "aprovado" || caio.Avaliacao != "reprovado" {
		t.Fatalf("avaliações ana=%q caio=%q", ana.Avaliacao, caio.Avaliacao)
	}

	arq, err := s.ExportResultado()
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(arq))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	linhas, err := f.GetRows("Lista")
	if err != nil {
		t.Fatal(err)
	}
	var viuAna bool
	for _, linha := range linhas {
		if len(linha) >= 7 && linha[1] == "Ana" {
			viuAna = true
			if linha[6] != "passou" || !strings.Contains(linha[3], "sexta") {
				t.Fatalf("linha da Ana: %+v", linha)
			}
		}
	}
	if !viuAna {
		t.Fatalf("Ana não está na planilha: %+v", linhas)
	}
}

func TestSemPreferenciaCustoZero(t *testing.T) {
	d := dadosAlocacao{
		atributos: map[int]map[string]string{1: {"curso": "Eco"}, 2: {"curso": "Dir"}},
		hard:      map[int]map[int]bool{},
		soft:      map[int]map[int]bool{},
	}
	d = aplicarSemPreferencia(d, []string{"Quarta 14-16, quinta 10-12"})
	if len(d.horarios) != 2 {
		t.Fatalf("%d horários", len(d.horarios))
	}
	param := parametrosAlocacaoPadrao()
	param.SemPreferencia = true
	param.AvaliadoresPorMesa = 1
	param.MinPessoasPorMesa = 1
	p := montarProblema(param, d.horarios, []*Avaliador{{ID: 1, Nome: "Prof"}}, d.prefs, d.hard, d.soft)
	if p.nCand != 2 || p.nSlot != 2 {
		t.Fatalf("instância %d candidatos, %d horários", p.nCand, p.nSlot)
	}
	for _, custo := range p.custoPref {
		if custo != 0 {
			t.Fatalf("custo de horário %d; sem preferência deveria ser 0", custo)
		}
	}
}

func TestUploadSemPreferenciaAceitaZeroOpcoes(t *testing.T) {
	router := buildRouter(NewSessionStore(), fstest.MapFS{"index.html": {Data: []byte("ok")}})
	buf, err := createMockExcelFile([][]interface{}{{"Nome", "Email Institucional"}, {"Ana", "ana@al.insper.edu.br"}})
	if err != nil {
		t.Fatal(err)
	}
	corpo := &strings.Builder{}
	fronteira := "xYz"
	corpo.WriteString("--" + fronteira + "\r\nContent-Disposition: form-data; name=\"usarPreferencia\"\r\n\r\nfalse\r\n")
	corpo.WriteString("--" + fronteira + "\r\nContent-Disposition: form-data; name=\"nOpcoes\"\r\n\r\n0\r\n")
	corpo.WriteString("--" + fronteira + "\r\nContent-Disposition: form-data; name=\"file\"; filename=\"a.xlsx\"\r\n\r\n")
	corpo.WriteString(buf.String() + "\r\n--" + fronteira + "--\r\n")
	req := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(corpo.String()))
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+fronteira)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, corpo %s", rec.Code, rec.Body.String())
	}
}

func indiceMesaPorDia(r *AlocacaoResponse, dia string) int {
	for i, m := range r.Mesas {
		if m.DiaNome == dia {
			return i
		}
	}
	return -1
}

func acharNoResultado(r *AlocacaoResponse, nome string) CandidatoResultado {
	for _, m := range r.Mesas {
		for _, c := range m.Candidatos {
			if c.Nome == nome {
				return c
			}
		}
	}
	for _, c := range r.NaoAlocadosInfo {
		if c.Nome == nome {
			return c
		}
	}
	return CandidatoResultado{}
}

func diaDe(r *AlocacaoResponse, nome string) string {
	for _, m := range r.Mesas {
		for _, c := range m.Candidatos {
			if c.Nome == nome {
				return m.DiaNome
			}
		}
	}
	return ""
}
