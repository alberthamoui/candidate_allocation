package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xuri/excelize/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRealocacaoManualLivreESugestoesOrdenadas(t *testing.T) {
	s := sessaoComResultado()
	s.avisos.Ativo = false
	s.parametros = parametrosAlocacaoPadrao()
	s.parametros.MinPessoasPorMesa = 1
	s.parametros.AvaliadoresPorMesa = 1
	s.lastResult.Mesas[0].Candidatos[0].NaoPosso = []string{"Proibido"}
	s.lastResult.Mesas[1].Avaliadores = []AvaliadorResultado{{ID: 11, Nome: "Proibido"}}
	s.lastResult.Mesas = append(s.lastResult.Mesas, MesaResult{ID: 3, DiaNome: "sexta 8-10", Descricao: "sexta 8-10 - mesa 1", Avaliadores: []AvaliadorResultado{{ID: 12, Nome: "Permitido"}}, Candidatos: []CandidatoResultado{}})
	antes := clonarResultado(s.lastResult)
	sugs, err := s.Sugestoes("Ana")
	if err != nil {
		t.Fatal(err)
	}
	if len(sugs) != 2 || sugs[0].MesaID != 3 || len(sugs[1].Alertas) == 0 {
		t.Fatalf("ranking incorreto: %+v", sugs)
	}
	a, _ := json.Marshal(antes)
	b, _ := json.Marshal(s.lastResult)
	if !bytes.Equal(a, b) {
		t.Fatal("sugestões alteraram a agenda")
	}
	op, err := s.RealocarPessoa("Ana", "", 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(op.Resultado.Alertas) == 0 {
		t.Fatal("restrição manual aceita sem alerta")
	}
	op, err = s.RealocarPessoa("Ana", "sábado 10-12", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if diaDe(op.Resultado, "Ana") != "sábado 10-12" {
		t.Fatalf("destino livre não aceito: %s", diaDe(op.Resultado, "Ana"))
	}
}
func TestDiaInteiroPreservaIntervalosEExcelAtualizado(t *testing.T) {
	s := sessaoComResultado()
	s.lastResult.Mesas[1].DiaNome = "quarta 16-18"
	s.lastResult.Mesas[1].Descricao = "quarta 16-18 - mesa 2"
	s.lastResult.Mesas[1].DiaID = 1
	if _, err := s.RealocarDia("quarta", "sexta", false); err != nil {
		t.Fatal(err)
	}
	if diaDe(s.lastResult, "Ana") != "sexta 14-16" || diaDe(s.lastResult, "Caio") != "sexta 16-18" {
		t.Fatal("intervalos do dia não preservados")
	}
	b, err := s.ExportResultado()
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows("Alocação")
	if err != nil {
		t.Fatal(err)
	}
	headings := map[string]bool{}
	for _, r := range rows {
		if len(r) > 0 {
			headings[r[0]] = true
		}
	}
	if !headings["Sexta 14-16"] || !headings["Sexta 16-18"] {
		t.Fatalf("Excel mistura horários: %v", headings)
	}
}
func TestPersistenciaRecuperaDadosEAvaliacoes(t *testing.T) {
	dir := t.TempDir()
	store, err := NewPersistentSessionStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	id, s, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("Excels/base_exemplo.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	mapping, err := s.SuggestMapping(data, 5, "@al.insper.edu.br")
	if err != nil {
		t.Fatal(err)
	}
	users, err := s.BuildUsuariosWithMapping(mapping)
	if err != nil {
		t.Fatal(err)
	}
	cands := []Usuario{}
	for _, u := range users.Usuarios {
		cands = append(cands, u.Usuario)
	}
	if err := s.SaveUsuarios(cands); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.lastResult = resultadoExemplo()
	s.mu.Unlock()
	if _, err := s.Avaliar("Ana", 0, "passou"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RealocarMesa(1, "sexta 14-16", false); err != nil {
		t.Fatal(err)
	}
	cfg := s.avisos
	cfg.Email.Senha = "senha-ficticia"
	if err := s.ConfigurarAvisos(cfg); err != nil {
		t.Fatal(err)
	}
	store2, err := NewPersistentSessionStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded := store2.Get(id)
	if loaded == nil {
		t.Fatal("sessão não recuperada")
	}
	defer fecharBanco(loaded)
	defer fecharBanco(s)
	r, err := loaded.ResultadoAtual()
	if err != nil {
		t.Fatal(err)
	}
	if acharNoResultado(r, "Ana").Avaliacao != "aprovado" || diaDe(r, "Ana") != "sexta 14-16" {
		t.Fatal("avaliação ou mudança perdidas")
	}
	var n int
	if err := loaded.db.QueryRow("SELECT count(*) FROM pessoa").Scan(&n); err != nil || n != 2 {
		t.Fatalf("candidatos perdidos: %d %v", n, err)
	}
	cfg2 := loaded.ConfigAtual()
	if cfg2.Email.Senha != "" || !cfg2.Email.SenhaDefinida {
		t.Fatal("API expõe segredo ou perdeu a configuração")
	}
	info, err := os.Stat(filepath.Join(dir, id+".db"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatal("arquivo de sessão sem proteção")
	}
}
func TestConcorrenciaOperacoesESnapshots(t *testing.T) {
	s := sessaoComResultado()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				switch i {
				case 0:
					_, _ = s.Avaliar("Ana", 0, "passou")
				case 1:
					_, _ = s.RealocarMesa(1, fmt.Sprint("sexta ", j), false)
				case 2:
					r, _ := s.ResultadoAtual()
					_, _ = json.Marshal(r)
				case 3:
					_, _ = s.ExportResultado()
				}
			}
		}(i)
	}
	wg.Wait()
}
func TestAvisosIncluemAvaliadorERepetemSomenteFalhas(t *testing.T) {
	s := sessaoComResultado()
	s.avisos.WhatsApp.Ativo = false
	s.lastResult.Mesas[0].Avaliadores[0].Email = "prof@exemplo.test"
	falhar := true
	destinos := []string{}
	s.avisos.enviarEmail = func(_ EmailConfig, para, _, _ string) error {
		destinos = append(destinos, para)
		if para == "ana@al.insper.edu.br" && falhar {
			return fmt.Errorf("falha simulada")
		}
		return nil
	}
	op, err := s.RealocarDia("quarta 14-16", "sexta 14-16", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(destinos) != 3 || op.Envio.Falhas != 1 || op.Envio.Enviados != 2 {
		t.Fatalf("avisos: %+v %v", op.Envio, destinos)
	}
	falhar = false
	destinos = nil
	retry, err := s.RepetirFalhas(op.LoteID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(destinos) != 1 || destinos[0] != "ana@al.insper.edu.br" || retry.Envio.Falhas != 0 {
		t.Fatalf("repetiu envios bem sucedidos: %v", destinos)
	}
}
func TestWhatsappNaoUsaMatriculaEDefineDDI(t *testing.T) {
	cfg := ConfigAvisos{Ativo: true, WhatsApp: WhatsConfig{Ativo: true}}
	c := CandidatoResultado{Nome: "Ana", Numero: "123456789"}
	m := MesaResult{DiaNome: "21 de outubro 14-16"}
	av := prepararAvisos(cfg, c, m)
	if av[0].Erro == "" {
		t.Fatal("matrícula aceita como telefone")
	}
	c.Telefone = "(11) 98888-7777"
	av = prepararAvisos(cfg, c, m)
	if av[0].Destino != "5511988887777" || av[0].Erro != "" {
		t.Fatalf("telefone: %+v", av)
	}
	if text := renderMensagem("{{data}} às {{horario}}", c, m); text != "21 de outubro às 14-16" {
		t.Fatal(text)
	}
}
func TestAvaliacaoLoteAtomica(t *testing.T) {
	s := sessaoComResultado()
	_, err := s.AvaliarLote([]itemAvaliacao{{ID: 1, Resultado: "passou"}, {ID: 999, Resultado: "passou"}})
	if err == nil {
		t.Fatal("id inválido aceito")
	}
	r, _ := s.ResultadoAtual()
	if acharNoResultado(r, "Ana").Avaliacao != "" {
		t.Fatal("lote parcialmente aplicado")
	}
}
func TestCliRemotaAtualizaMesmaSessao(t *testing.T) {
	store := NewSessionStore()
	id, s, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	s.lastResult = resultadoExemplo()
	srv := httptest.NewServer(novoRouter(store))
	defer srv.Close()
	args := []string{"avaliar", "-servidor", srv.URL, "-sessao", id, "-id", "1", "-resultado", "aprovado"}
	if err := executarCLI(args); err != nil {
		t.Fatal(err)
	}
	r, _ := s.ResultadoAtual()
	if acharNoResultado(r, "Ana").Avaliacao != "aprovado" {
		t.Fatal("CLI mudou outro estado")
	}
	if err := executarCLI([]string{"sugestoes", "-servidor", srv.URL, "-sessao", id, "-pessoa", "Ana"}); err != nil {
		t.Fatal(err)
	}
}
func TestConfigAvisosRedigeEPreservaSegredos(t *testing.T) {
	s := sessaoComResultado()
	cfg := s.avisos
	cfg.Email.Senha = "senha-ficticia"
	cfg.WhatsApp.Token = "token-ficticio"
	if err := s.ConfigurarAvisos(cfg); err != nil {
		t.Fatal(err)
	}
	safe := s.ConfigAtual()
	safe.Mensagem = "Oi {{nome}}"
	if err := s.ConfigurarAvisos(safe); err != nil {
		t.Fatal(err)
	}
	if s.avisos.Email.Senha != "senha-ficticia" || s.avisos.WhatsApp.Token != "token-ficticio" {
		t.Fatal("salvar mensagem apaga credenciais")
	}
	b, _ := json.Marshal(s.ConfigAtual())
	if strings.Contains(string(b), "fictic") {
		t.Fatal("segredos expostos")
	}
}
func TestAPICanaisErroSemSegredos(t *testing.T) {
	s := sessaoComResultado()
	cfg := s.avisos
	cfg.WhatsApp.Provedor = "webhook"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "token-secreto", http.StatusBadRequest) }))
	defer srv.Close()
	cfg.WhatsApp.URL = srv.URL
	err := enviarWhatsAppVariaveis(cfg.WhatsApp, "5511999999999", "texto", nil)
	if err == nil || strings.Contains(err.Error(), "token-secreto") {
		t.Fatalf("erro inseguro: %v", err)
	}
}

type transporteTeste func(*http.Request) (*http.Response, error)

func (f transporteTeste) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestMetaEnviaModeloAprovadoEValidaResposta(t *testing.T) {
	client := &http.Client{Transport: transporteTeste(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "graph.facebook.com" || r.Header.Get("Authorization") != "Bearer token-ficticio" {
			t.Fatal("provedor/auth incorretos")
		}
		var payload struct {
			Product  string `json:"messaging_product"`
			To       string `json:"to"`
			Type     string `json:"type"`
			Template struct {
				Name       string `json:"name"`
				Components []struct {
					Parameters []map[string]string `json:"parameters"`
				} `json:"components"`
			} `json:"template"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Product != "whatsapp" || payload.To != "5511999999999" || payload.Type != "template" || payload.Template.Name != "entrevista" || payload.Template.Components[0].Parameters[0]["text"] != "Ana" {
			t.Fatalf("payload: %+v", payload)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"messages":[{"id":"wamid.123"}]}`)), Header: make(http.Header)}, nil
	})}
	cfg := WhatsConfig{Provedor: "meta", PhoneID: "1234", Token: "token-ficticio", Template: "entrevista", VariaveisTemplate: []string{"nome"}, httpClient: client}
	if err := enviarWhatsAppVariaveis(cfg, "5511999999999", "texto", map[string]string{"nome": "Ana"}); err != nil {
		t.Fatal(err)
	}
}
func TestTelefoneInternacionalPreservaDDI(t *testing.T) {
	n, err := normalizarTelefone("+33 1 23 45 67 89")
	if err != nil || n != "33123456789" {
		t.Fatalf("telefone internacional: %s %v", n, err)
	}
}

func TestEnvioAsyncLiberaAgendaERepeticaoIdempotente(t *testing.T) {
	s := sessaoComResultado()
	s.avisos.WhatsApp.Ativo = false
	iniciado := make(chan struct{}, 10)
	continuar := make(chan struct{})
	s.avisos.enviarEmail = func(_ EmailConfig, _, _, _ string) error {
		iniciado <- struct{}{}
		<-continuar
		return fmt.Errorf("falhou")
	}
	op, err := s.operar(s.enviarAvisosTodosLocked, true)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-iniciado:
	case <-time.After(2 * time.Second):
		t.Fatal("envio não começou")
	}
	if _, err := s.Avaliar("Ana", 0, "passou"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.operar(s.enviarAvisosTodosLocked, true); err == nil {
		t.Fatal("envio duplicado aceito enquanto primeiro envia")
	}
	close(continuar)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		l, _ := s.Lote(op.LoteID)
		if l.Status != "enviando" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	retry, err := s.RepetirFalhas(op.LoteID, false)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Envio.Falhas == 0 {
		t.Fatal("teste deveria conservar falhas")
	}
	if _, err := s.RepetirFalhas(op.LoteID, false); err == nil {
		t.Fatal("mesmas falhas encaminhadas duas vezes")
	}
}
