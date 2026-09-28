package main

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// comRecursos troca os recursos globais durante o teste.
func comRecursos(t *testing.T, r Recursos) {
	t.Helper()
	antes := recursos
	recursos = r
	t.Cleanup(func() { recursos = antes })
}

func TestLerCpuMax(t *testing.T) {
	casos := map[string]float64{
		"10000 100000":  0.1, // Render gratuito
		"50000 100000":  0.5,
		"200000 100000": 2,
		"max 100000":    0, // sem cota
		"-1 100000":     0, // cgroup v1 sem cota
		"":              0,
		"abc":           0,
	}
	for entrada, want := range casos {
		if got := lerCpuMax(entrada); got != want {
			t.Errorf("lerCpuMax(%q) = %v, esperado %v", entrada, got, want)
		}
	}
}

func TestConfigurarRecursos(t *testing.T) {
	semEnv := func(string) string { return "" }
	comRecursos(t, Recursos{CPUs: 8, Execucoes: SA_EXECUCOES, Iteracoes: SA_ITERACOES, AlocacoesSimultaneas: 2, MaxSessoes: 200, MaxUploadMB: 10})

	// Render gratuito: 0,1 CPU → uma busca e uma alocação por vez
	r, err := configurarRecursos(semEnv, 0.1)
	if err != nil || r.CPUs != 0.1 || r.Execucoes != 1 || r.AlocacoesSimultaneas != 1 || r.Iteracoes != SA_ITERACOES {
		t.Errorf("0,1 CPU: %+v, %v", r, err)
	}
	// sem cota numa máquina de 8 núcleos: como antes
	r, _ = configurarRecursos(semEnv, 0)
	if r.CPUs != 8 || r.Execucoes != SA_EXECUCOES || r.AlocacoesSimultaneas != 1 {
		t.Errorf("sem cota: %+v", r)
	}
	// 2 CPUs: duas buscas por alocação
	r, _ = configurarRecursos(semEnv, 2)
	if r.Execucoes != 2 || r.AlocacoesSimultaneas != 1 {
		t.Errorf("2 CPUs: %+v", r)
	}

	// variáveis de ambiente mandam
	env := map[string]string{"ALOCACAO_EXECUCOES": "3", "ALOCACAO_ITERACOES": "50000", "ALOCACOES_SIMULTANEAS": "2", "MAX_SESSOES": "10", "MAX_UPLOAD_MB": "5"}
	r, err = configurarRecursos(func(k string) string { return env[k] }, 0.1)
	if err != nil || r.Execucoes != 3 || r.Iteracoes != 50_000 || r.AlocacoesSimultaneas != 2 || r.MaxSessoes != 10 || r.MaxUploadMB != 5 {
		t.Errorf("com variáveis: %+v, %v", r, err)
	}
	for _, ruim := range []string{"abc", "0", "999999999"} {
		if _, err := configurarRecursos(func(k string) string {
			if k == "ALOCACAO_ITERACOES" {
				return ruim
			}
			return ""
		}, 0); err == nil {
			t.Errorf("ALOCACAO_ITERACOES=%q deveria dar erro", ruim)
		}
	}
}

func novoRouter(store *SessionStore) http.Handler {
	return buildRouter(store, fstest.MapFS{"index.html": {Data: []byte("ok")}})
}

func TestAlocacaoCanceladaParaLogo(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	d := dadosDaSessao(t, s)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar() // o usuário já saiu

	inicio := time.Now()
	res, mesas := fazerMelhorAlocacaoMesas(ctx, parametrosAlocacaoPadrao(), d.horarios, d.avals, d.prefs, d.hard, d.soft, nil, nil)
	if dur := time.Since(inicio); dur > 300*time.Millisecond {
		t.Errorf("alocação cancelada levou %v", dur)
	}
	// ainda devolve uma alocação válida (a da solução inicial)
	validarAlocacao(t, parametrosAlocacaoPadrao(), res, mesas, d.prefs, d.hard)
}

// Com uma vaga só, a segunda alocação espera a vez e desiste se o usuário sair.
func TestFilaDeAlocacoes(t *testing.T) {
	r := recursos
	r.AlocacoesSimultaneas = 1
	comRecursos(t, r)
	store := NewSessionStore()
	store.sessions["s"] = carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	router := novoRouter(store)

	store.vagasAlocacao <- struct{}{} // outra alocação ocupando a vaga
	ctx, cancelar := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancelar()
	req := httptest.NewRequest(http.MethodGet, "/api/alocar?sessionId=s", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if corpo := rec.Body.String(); !strings.Contains(corpo, "Aguardando outra alocação terminar") || strings.Contains(corpo, `"done"`) {
		t.Errorf("esperado só o aviso de espera, corpo: %s", corpo)
	}

	// vaga liberada: roda até o fim e devolve a vaga
	<-store.vagasAlocacao
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/alocar?sessionId=s", nil))
	if !strings.Contains(rec.Body.String(), `"done":true`) {
		t.Errorf("alocação não terminou: %.300s", rec.Body.String())
	}
	if len(store.vagasAlocacao) != 0 {
		t.Error("a vaga não foi devolvida")
	}
}

func TestAlocacaoSemSessaoAvisaQueExpirou(t *testing.T) {
	rec := httptest.NewRecorder()
	novoRouter(NewSessionStore()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/alocar?sessionId=nao-existe", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "sessão expirou") {
		t.Errorf("status %d, corpo %s", rec.Code, rec.Body.String())
	}
}

func upload(t *testing.T, router http.Handler, conteudo []byte) *httptest.ResponseRecorder {
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
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/upload", &corpo)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestLimitesDeUploadESessoes(t *testing.T) {
	r := recursos
	r.MaxUploadMB, r.MaxSessoes = 1, 1
	comRecursos(t, r)
	store := NewSessionStore()
	router := novoRouter(store)

	if rec := upload(t, router, make([]byte, 2<<20)); rec.Code != http.StatusRequestEntityTooLarge || !strings.Contains(rec.Body.String(), "limite de 1 MB") {
		t.Errorf("upload grande: status %d, corpo %s", rec.Code, rec.Body.String())
	}

	planilha, err := createMockExcelFile([][]interface{}{{"Nome"}, {"Ana"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec := upload(t, router, planilha.Bytes()); rec.Code != http.StatusOK {
		t.Fatalf("primeiro upload: status %d, corpo %s", rec.Code, rec.Body.String())
	}
	if rec := upload(t, router, planilha.Bytes()); rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "muitas sessões") {
		t.Errorf("com o servidor cheio: status %d, corpo %s", rec.Code, rec.Body.String())
	}
}
