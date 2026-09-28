package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func contar(t *testing.T, s *Session, tabela string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM ` + tabela).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSalvarCandidatosEhTudoOuNada(t *testing.T) {
	s := setupSession()
	cands := []Usuario{
		{Nome: "Ana", CPF: "11111111111", Semestre: "3", Opcoes: []string{"segunda 8-10"}},
		{Nome: "Bia", CPF: "22222222222", Semestre: "3", Opcoes: []string{"segunda 8-10", "terca 10-12"}},
		{Nome: "Caio", CPF: "11111111111", Semestre: "3", Opcoes: []string{"terca 10-12"}}, // CPF repetido
	}
	err := s.SaveUsuarios(cands)
	if err == nil || !strings.Contains(err.Error(), `"Caio"`) {
		t.Fatalf("esperado erro citando o candidato com CPF repetido, obteve %v", err)
	}
	// nada fica gravado pela metade
	for _, tabela := range []string{"pessoa", "disponibilidade", "opcoes_horario"} {
		if n := contar(t, s, tabela); n != 0 {
			t.Errorf("%s tem %d linhas depois do erro; esperado 0", tabela, n)
		}
	}

	// sem o repetido, grava tudo
	if err := s.SaveUsuarios(cands[:2]); err != nil {
		t.Fatal(err)
	}
	if p, d, h := contar(t, s, "pessoa"), contar(t, s, "disponibilidade"), contar(t, s, "opcoes_horario"); p != 2 || d != 3 || h != 2 {
		t.Errorf("gravados: %d pessoas, %d disponibilidades, %d horários; esperado 2, 3, 2", p, d, h)
	}
}

func TestSalvarAvaliadorRepetido(t *testing.T) {
	s := setupSession()
	err := s.SaveAvaliadores([]AvaliadorInfo{
		{Nome: "Ana", Email: "ana@x.com", Sigla: "AN"},
		{Nome: "Ana", Email: "outra@x.com", Sigla: "AN2"}, // nome repetido
	})
	if err == nil || !strings.Contains(err.Error(), "repetido") {
		t.Fatalf("esperado erro de avaliador repetido, obteve %v", err)
	}
	if n := contar(t, s, "avaliador"); n != 0 {
		t.Errorf("%d avaliadores gravados depois do erro; esperado 0", n)
	}
}

func TestRestricoesDesconhecidasSaoIgnoradas(t *testing.T) {
	s := setupSession()
	if err := s.SaveUsuarios([]Usuario{{Nome: "Ana", CPF: "11111111111", Semestre: "3"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAvaliadores([]AvaliadorInfo{{Nome: "Beto", Email: "b@x.com", Sigla: "BT"}}); err != nil {
		t.Fatal(err)
	}
	err := s.SaveRestricoes([]Restricao{
		{Candidato: "Ana", NaoPosso: "BT, XX", PrefiroNao: "BT"}, // XX não existe
		{Candidato: "Ninguém", NaoPosso: "BT"},                   // candidato não existe
	})
	if err != nil {
		t.Fatalf("restrições desconhecidas deveriam ser ignoradas, obteve %v", err)
	}
	if n, p := contar(t, s, "restricoesNposso"), contar(t, s, "restricoesPrefiroN"); n != 1 || p != 1 {
		t.Errorf("gravadas %d 'não posso' e %d 'prefiro não'; esperado 1 e 1", n, p)
	}
}

func TestUploadRejeitaNumeroDeOpcoesInvalido(t *testing.T) {
	router := buildRouter(NewSessionStore(), fstest.MapFS{"index.html": {Data: []byte("ok")}})
	for _, v := range []string{"abc", "0", "11"} {
		buf, err := createMockExcelFile([][]interface{}{{"Nome"}, {"Ana"}})
		if err != nil {
			t.Fatal(err)
		}
		corpo := &strings.Builder{}
		fronteira := "xYz"
		corpo.WriteString("--" + fronteira + "\r\nContent-Disposition: form-data; name=\"nOpcoes\"\r\n\r\n" + v + "\r\n")
		corpo.WriteString("--" + fronteira + "\r\nContent-Disposition: form-data; name=\"file\"; filename=\"a.xlsx\"\r\n\r\n")
		corpo.WriteString(buf.String() + "\r\n--" + fronteira + "--\r\n")
		req := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(corpo.String()))
		req.Header.Set("Content-Type", "multipart/form-data; boundary="+fronteira)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "número de opções inválido") {
			t.Errorf("nOpcoes=%q: status %d, corpo %s", v, rec.Code, rec.Body.String())
		}
	}
}

// Horários que só diferem na caixa viram o mesmo horário, e cada
// disponibilidade aponta para o horário certo.
func TestHorariosComCaixaDiferente(t *testing.T) {
	s := setupSession()
	err := s.SaveUsuarios([]Usuario{
		{Nome: "Ana", CPF: "11111111111", Semestre: "3", Opcoes: []string{"Segunda 8-10", "terca 10-12"}},
		{Nome: "Bia", CPF: "22222222222", Semestre: "3", Opcoes: []string{"segunda 8-10"}},
		{Nome: "Caio", CPF: "33333333333", Semestre: "3", Opcoes: []string{"SEGUNDA 8-10"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := contar(t, s, "opcoes_horario"); n != 2 {
		t.Errorf("%d horários gravados; esperado 2", n)
	}
	rows, err := s.db.Query(`SELECT p.nome, h.opcao FROM disponibilidade d
		JOIN pessoa p ON p.id = d.pessoa_id JOIN opcoes_horario h ON h.id = d.horario_id
		WHERE d.preferencia = 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var nome, opcao string
		if err := rows.Scan(&nome, &opcao); err != nil {
			t.Fatal(err)
		}
		if opcao != "segunda 8-10" {
			t.Errorf("1ª opção de %s aponta para %q; esperado \"segunda 8-10\"", nome, opcao)
		}
	}
}

// A mesma sigla repetida na planilha é o mesmo avaliador: não é erro.
func TestAvaliadorComMesmaSiglaRepetido(t *testing.T) {
	s := setupSession()
	av := AvaliadorInfo{Nome: "Ana", Email: "ana@x.com", Sigla: "AN"}
	if err := s.SaveAvaliadores([]AvaliadorInfo{av, av}); err != nil {
		t.Fatal(err)
	}
	if n := contar(t, s, "avaliador"); n != 1 {
		t.Errorf("%d avaliadores; esperado 1", n)
	}
}
