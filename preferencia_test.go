package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLerHorariosLivres(t *testing.T) {
	h, err := lerHorariosLivres("Segunda 14-16\n terça 10-12 ; quarta 8-10,\r\nsegunda 14-16\n\n")
	if err != nil {
		t.Fatal(err)
	}
	if esperado := []string{"segunda 14-16", "terça 10-12", "quarta 8-10"}; !reflect.DeepEqual(h, esperado) {
		t.Errorf("horários = %q; esperado %q", h, esperado)
	}

	if _, err := lerHorariosLivres(" \n , ;"); err == nil {
		t.Error("lista vazia deveria dar erro")
	}
	muitos := make([]string, LIMITE_HORARIOS_LIVRES+1)
	for i := range muitos {
		muitos[i] = "horario " + string(rune('a'+i%26)) + string(rune('a'+i/26))
	}
	if _, err := lerHorariosLivres(strings.Join(muitos, "\n")); err == nil {
		t.Errorf("mais de %d horários deveria dar erro", LIMITE_HORARIOS_LIVRES)
	}
}

func TestCPFOpcionalETelefoneFlexivel(t *testing.T) {
	base := Usuario{Nome: "Ana", Semestre: "3", EmailInsper: "ana@al.insper.edu.br", EmailPessoal: "ana@x.com"}
	casos := []struct {
		cpf, numero         string
		cpfFinal, numFinal  string
		erroCPF, erroNumero bool
	}{
		{cpf: "", numero: "202300001", numFinal: "202300001"}, // sem CPF, RA de 9 dígitos
		{cpf: "123.456.789-01", numero: "(11) 98888-7777", cpfFinal: "12345678901", numFinal: "11988887777"},
		{cpf: "12345678901", numero: "+55 11 98888-7777", cpfFinal: "12345678901", numFinal: "5511988887777"},
		{cpf: "1234", numero: "12345", erroCPF: true, erroNumero: true},                   // curtos demais
		{cpf: "abc.def.ghi-jk", numero: "11 9888A-7777", erroCPF: true, erroNumero: true}, // letras
		{cpf: "", numero: "", erroNumero: true},                                           // telefone continua obrigatório
	}
	for _, c := range casos {
		u := base
		u.CPF, u.Numero = c.cpf, c.numero
		res, _ := processData([]Usuario{u}, "@al.insper.edu.br")
		r := res[1]
		campos := map[string]bool{}
		for _, e := range r.Erros {
			campos[e.Field] = true
		}
		if campos["cpf"] != c.erroCPF || campos["numero"] != c.erroNumero {
			t.Errorf("CPF %q, número %q: erros %v; esperado cpf=%v numero=%v", c.cpf, c.numero, r.Erros, c.erroCPF, c.erroNumero)
		}
		if !c.erroCPF && r.Usuario.CPF != c.cpfFinal {
			t.Errorf("CPF %q virou %q; esperado %q", c.cpf, r.Usuario.CPF, c.cpfFinal)
		}
		if !c.erroNumero && r.Usuario.Numero != c.numFinal {
			t.Errorf("número %q virou %q; esperado %q", c.numero, r.Usuario.Numero, c.numFinal)
		}
	}
}

func TestVariosCandidatosSemCPF(t *testing.T) {
	s := setupSession()
	err := s.SaveUsuarios([]Usuario{
		{Nome: "Ana", Semestre: "3", Opcoes: []string{"segunda 8-10"}},
		{Nome: "Bia", Semestre: "3", Opcoes: []string{"segunda 8-10"}},
		{Nome: "Caio", CPF: "11111111111", Semestre: "3", Opcoes: []string{"segunda 8-10"}},
	})
	if err != nil {
		t.Fatalf("candidatos sem CPF deveriam ser gravados: %v", err)
	}
	var semCPF int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pessoa WHERE cpf IS NULL`).Scan(&semCPF); err != nil {
		t.Fatal(err)
	}
	if semCPF != 2 {
		t.Errorf("%d candidatos com CPF NULL; esperado 2", semCPF)
	}
	// CPF preenchido continua único
	if err := s.SaveUsuarios([]Usuario{{Nome: "Duda", CPF: "11111111111", Semestre: "3"}}); err == nil {
		t.Error("CPF repetido deveria dar erro")
	}
}

// uploadComCampos envia a planilha com os campos de formulário da tela inicial.
func uploadComCampos(t *testing.T, router http.Handler, conteudo []byte, campos map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	fw, err := mw.CreateFormFile("file", "planilha.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(conteudo); err != nil {
		t.Fatal(err)
	}
	for k, v := range campos {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/upload", &corpo)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestUploadSemPreferencia(t *testing.T) {
	planilha, err := os.ReadFile("Excels/teste_oficial.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	store := NewSessionStore()
	router := novoRouter(store)

	rec := uploadComCampos(t, router, planilha, map[string]string{"semPreferencia": "1", "horarios": " \n"})
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "pelo menos um horário") {
		t.Errorf("sem horários: status %d, corpo %s", rec.Code, rec.Body.String())
	}

	rec = uploadComCampos(t, router, planilha, map[string]string{"semPreferencia": "1", "horarios": "segunda 14-16\nterça 10-12"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, corpo %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		SessionID string        `json:"sessionId"`
		Mapping   []MappingItem `json:"mapping"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	for _, m := range resp.Mapping {
		if strings.HasPrefix(m.Variavel, "opcao") {
			t.Errorf("sem preferência, o mapeamento não deveria pedir %q", m.Variavel)
		}
	}
	if h := store.Get(resp.SessionID).horariosLivres; !reflect.DeepEqual(h, []string{"segunda 14-16", "terça 10-12"}) {
		t.Errorf("horários da sessão = %q", h)
	}

	// o modo normal continua igual: pede as colunas de opção
	rec = uploadComCampos(t, router, planilha, map[string]string{"nOpcoes": "5"})
	if !strings.Contains(rec.Body.String(), `"opcao 5"`) {
		t.Errorf("modo normal deveria pedir as 5 opções: %.300s", rec.Body.String())
	}
}

// A planilha oficial tem colunas de opção; sem preferência elas são ignoradas
// e todo candidato pode ir em qualquer um dos horários digitados.
func TestAlocacaoSemPreferencia(t *testing.T) {
	horarios := []string{"segunda 14-16", "terça 10-12", "quarta 16-18", "quinta 8-10", "sexta 14-16"}
	s := setupSession()
	s.horariosLivres = horarios
	preencherSessaoXLSX(t, s, "Excels/teste_oficial.xlsx", 0)

	if n := contar(t, s, "disponibilidade"); n != 98*len(horarios) {
		t.Errorf("%d disponibilidades; esperado %d (cada candidato em todos os horários)", n, 98*len(horarios))
	}

	param := parametrosAlocacaoPadrao()
	previa, err := s.PreviaCapacidade(param)
	if err != nil {
		t.Fatal(err)
	}
	if !previa.SemPreferencia || len(previa.Horarios) != len(horarios) || previa.MaxAlocaveis != 98 {
		t.Errorf("prévia: sem preferência %v, %d horários, cabem %d", previa.SemPreferencia, len(previa.Horarios), previa.MaxAlocaveis)
	}
	for _, h := range previa.Horarios {
		if h.PrimeiraOpcao != 0 || h.Interessados != 98 {
			t.Errorf("%s: 1ª opção %d, interessados %d; esperado 0 e 98", h.Descricao, h.PrimeiraOpcao, h.Interessados)
		}
	}

	// a alocação respeita as regras e nenhum horário custa mais que outro
	d := dadosDaSessao(t, s)
	param.semPreferencia = d.semPreferencia
	res, mesas := fazerMelhorAlocacaoMesas(context.Background(), param, d.horarios, d.avals, d.prefs, d.hard, d.soft, nil, nil)
	validarAlocacao(t, param, res, mesas, d.prefs, d.hard)
	_, pen, _ := pontuarResultado(res, mesas, d.prefs, d.hard, d.soft, true)
	for k, v := range pen {
		if strings.HasPrefix(k, "opcao_") && v != 0 {
			t.Errorf("sem preferência não deveria contar %s (%d)", k, v)
		}
	}
	if res.Alocados != 98 || res.Pontuacao != SCORE_BASE+pen["prefiro_nao"]*PENALIDADE_SOFT {
		t.Errorf("%d alocados, pontuação %d, penalidades %v", res.Alocados, res.Pontuacao, pen)
	}

	r, err := s.RunAlocacao(context.Background(), parametrosAlocacaoPadrao(), func(any) {})
	if err != nil {
		t.Fatal(err)
	}
	if !r.SemPreferencia || r.TotalAlocados != 98 {
		t.Errorf("resultado: sem preferência %v, %d alocados", r.SemPreferencia, r.TotalAlocados)
	}
	for _, m := range r.Mesas {
		if !contem(horarios, m.DiaNome) {
			t.Errorf("mesa em horário não digitado: %q", m.DiaNome)
		}
		for _, c := range m.Candidatos {
			if len(c.Opcoes) != 0 || c.Opcao != 0 {
				t.Errorf("%s: opções %q, opção %d; esperado vazio", c.Nome, c.Opcoes, c.Opcao)
			}
		}
	}
	for _, it := range r.Qualidade {
		if strings.HasPrefix(it.Codigo, "opcao_") {
			t.Errorf("relatório não deveria ter %q sem preferência", it.Codigo)
		}
	}
}

func contem(lista []string, s string) bool {
	for _, x := range lista {
		if x == s {
			return true
		}
	}
	return false
}
