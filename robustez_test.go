package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// Um erro de banco numa sessão falha só as requisições dela: o servidor
// continua de pé e as outras sessões seguem funcionando. (Com log.Fatal, este
// teste derrubaria o próprio processo de teste.)
func TestErroDeBancoNaoDerrubaOServidor(t *testing.T) {
	store := NewSessionStore()
	router := buildRouter(store, fstest.MapFS{"index.html": {Data: []byte("ok")}})

	quebrada := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	saudavel := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	store.sessions["quebrada"] = quebrada
	store.sessions["saudavel"] = saudavel

	// simula um bug de schema: uma tabela que o carregamento lê some
	if _, err := quebrada.db.Exec(`DROP TABLE restricoesPrefiroN`); err != nil {
		t.Fatal(err)
	}

	pedir := func(url, sessao string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("X-Session-Id", sessao)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// prévia da sessão quebrada: 500 com a causa
	rec := pedir("/api/capacidade", "quebrada")
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "carregando restrições") {
		t.Errorf("prévia da sessão quebrada: status %d, corpo %s", rec.Code, rec.Body.String())
	}

	// alocação da sessão quebrada: evento de erro no stream
	rec = pedir("/api/alocar?sessionId=quebrada", "")
	if !strings.Contains(rec.Body.String(), `"error"`) || !strings.Contains(rec.Body.String(), "carregando restrições") {
		t.Errorf("alocação da sessão quebrada deveria mandar um evento de erro, corpo: %s", rec.Body.String())
	}

	// a outra sessão segue normal
	if rec := pedir("/api/capacidade", "saudavel"); rec.Code != http.StatusOK {
		t.Errorf("prévia da sessão saudável: status %d, corpo %s", rec.Code, rec.Body.String())
	}
	rec = pedir("/api/alocar?sessionId=saudavel", "")
	if !strings.Contains(rec.Body.String(), `"done":true`) {
		t.Errorf("alocação da sessão saudável não terminou, corpo: %.300s", rec.Body.String())
	}
}

func TestSetupConnDevolveErro(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := setupConn(db); err == nil {
		t.Error("esperado erro ao criar o schema num banco fechado")
	}
}
