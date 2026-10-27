package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOperacaoELoteRollbackJuntos(t *testing.T) {
	s := sessaoComResultado()
	s.lastResult.Mesas[0].Avaliadores = nil
	defer fecharBanco(s)
	before, _ := s.ResultadoAtual()
	if _, err := s.db.Exec(`CREATE TRIGGER recusar_lote BEFORE INSERT ON lotes BEGIN SELECT RAISE(FAIL,'disco indisponível');END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RealocarMesa(1, "sexta 10-12", true); err == nil {
		t.Fatal("erro de gravação ignorado")
	}
	after, _ := s.ResultadoAtual()
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if !bytes.Equal(a, b) {
		t.Fatal("agenda alterada apesar da falha do lote")
	}
	if n := contar(t, s, "lotes"); n != 0 {
		t.Fatalf("lote parcial: %d", n)
	}
}
func TestEnvioIndividualNaoRegravaSnapshot(t *testing.T) {
	s := sessaoComResultado()
	s.lastResult.Mesas[0].Avaliadores = nil
	defer fecharBanco(s)
	s.avisos.WhatsApp.Ativo = false
	s.avisos.enviarEmail = func(EmailConfig, string, string, string) error { return nil }
	// O snapshot pode estar indisponível depois de autorizar o lote: o retorno
	// de cada provedor deve ser salvo na tabela de entregas, independentemente.
	s.avisos.enviarEmail = func(EmailConfig, string, string, string) error {
		if _, err := s.db.Exec(`CREATE TRIGGER IF NOT EXISTS recusar_estado BEFORE INSERT ON estado_sessao BEGIN SELECT RAISE(FAIL,'snapshot bloqueado');END`); err != nil {
			return err
		}
		return nil
	}
	op, err := s.EnviarAvisos(pedidoEnvio{Pessoa: "Ana"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if op.Envio.Enviados != 1 || op.Envio.Falhas != 0 {
		t.Fatalf("retorno não salvo: %+v", op.Envio)
	}
}
func TestIncertoNaoRepeteEHistoricoTemPaginacao(t *testing.T) {
	s := sessaoComResultado()
	s.lastResult.Mesas[0].Avaliadores = nil
	defer fecharBanco(s)
	s.avisos.WhatsApp.Ativo = false
	s.avisos.enviarEmail = func(EmailConfig, string, string, string) error {
		return envioIncerto{errors.New("conexão perdida após transmissão")}
	}
	op, err := s.EnviarAvisos(pedidoEnvio{Pessoa: "Ana"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if op.Envio.Incertos != 1 || erroEnvio(op) == nil {
		t.Fatalf("incerto tratado como sucesso: %+v", op)
	}
	if _, err := s.RepetirFalhas(op.LoteID, false); err == nil {
		t.Fatal("incerto foi repetido")
	}
	s.avisos.enviarEmail = func(EmailConfig, string, string, string) error { return nil }
	for i := 0; i < 4; i++ {
		if _, err := s.EnviarAvisos(pedidoEnvio{Pessoa: "Bia"}, false); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.HistoricoPagina(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Lotes) != 2 || p.Proximo == 0 || len(p.Lotes[0].Avisos) != 0 {
		t.Fatalf("histórico sem limite ou com detalhes: %+v", p)
	}
	next, err := s.HistoricoPagina(p.Proximo, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Lotes) != 2 || next.Lotes[0].ID == p.Lotes[0].ID {
		t.Fatal("cursor repetiu página")
	}
}
func TestEvictRecuperaAgendaEApenasEnvioInterrompidoFicaIncerto(t *testing.T) {
	store, err := NewPersistentSessionStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	id, s, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	s.lastResult = resultadoExemplo()
	s.lastResult.Mesas[0].Avaliadores = nil
	s.avisos = ConfigAvisos{Ativo: true, Email: EmailConfig{Ativo: true}}
	s.mu.Lock()
	l := s.prepararLoteLocked([]int{1})
	s.novoLote = &l
	err = s.persistirLocked()
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("UPDATE entregas SET status='enviando' WHERE lote=?", l.ID); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	s.updatedAt = time.Now().Add(-time.Hour)
	s.busy.Add(1)
	store.evictLocked(time.Now(), false)
	if store.sessions[id] != s {
		t.Fatal("sessão ativa descarregada")
	}
	s.busy.Add(-1)
	store.evictLocked(time.Now(), false)
	if store.sessions[id] != nil {
		t.Fatal("sessão inativa retida na memória")
	}
	store.mu.Unlock()
	loaded := store.Get(id)
	if loaded == nil {
		t.Fatal("agenda perdida após descarregar")
	}
	r, err := loaded.ResultadoAtual()
	if err != nil || r.TotalAlocados != 3 {
		t.Fatalf("agenda não restaurada: %+v %v", r, err)
	}
	lote, err := loaded.Lote(l.ID)
	if err != nil || lote.Incertos != 1 || lote.Pendentes != 0 {
		t.Fatalf("retomada reenviou algo incerto: %+v %v", lote, err)
	}
}
func TestCLIETelaCompartilhamTodasOperacoes(t *testing.T) {
	store := NewSessionStore()
	defer store.Close()
	id, s, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	s.lastResult = resultadoExemplo()
	s.lastResult.Mesas[0].Avaliadores = nil
	s.avisos = ConfigAvisos{Ativo: true, Email: EmailConfig{Ativo: true}}
	var recipients int
	s.avisos.enviarEmail = func(EmailConfig, string, string, string) error { recipients++; return nil }
	srv := httptest.NewServer(novoRouter(store))
	defer srv.Close()
	run := func(args ...string) {
		t.Helper()
		args = append(args, "-servidor", srv.URL, "-sessao", id)
		if err := executarCLI(args); err != nil {
			t.Fatal(err)
		}
	}
	run("avaliar", "-id", "1", "-resultado", "passou")
	run("realocar-pessoa", "-pessoa", "1", "-mesa", "2")
	run("realocar-mesa", "-mesa", "2", "-para", "28/10/2026 10:00-12:00")
	run("realocar-dia", "-de", "28/10/2026", "-para", "29/10/2026")
	run("enviar", "-pessoa", "1")
	if recipients != 1 {
		t.Fatalf("envio não foi específico: %d", recipients)
	}
	dest := filepath.Join(t.TempDir(), "agenda.xlsx")
	run("exportar", "-saida", dest)
	if info, err := os.Stat(dest); err != nil || info.Size() < 1000 {
		t.Fatalf("Excel inválido: %v", err)
	}
	resp, err := http.Get(srv.URL + "/api/resultado?sessionId=" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var r AlocacaoResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	if acharNoResultado(&r, "Ana").Avaliacao != "aprovado" || diaDe(&r, "Ana") != "29/10/2026 10:00-12:00" {
		t.Fatalf("app vê outro estado: %+v", r)
	}
	s.avisos.enviarEmail = func(EmailConfig, string, string, string) error { return fmt.Errorf("falha simulada") }
	if err := executarCLI([]string{"enviar", "-pessoa", "1", "-servidor", srv.URL, "-sessao", id}); err == nil {
		t.Fatal("CLI retornou sucesso para falha de envio")
	}
}
func TestMapeamentoNegativoECpfAusente(t *testing.T) {
	s := setupSession()
	defer fecharBanco(s)
	b, err := createMockExcelFile([][]interface{}{{"Nome"}, {"Ana"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SuggestMapping(b.Bytes(), 0, "@exemplo.test"); err != nil {
		t.Fatal(err)
	}
	u, err := s.BuildUsuariosWithMapping([]MappingItem{{Variavel: "nome", Indice: 0}, {Variavel: "cpf", Indice: -1}})
	if err != nil || u.Usuarios[1].Usuario.CPF != "" {
		t.Fatalf("campo opcional falhou: %+v %v", u, err)
	}
}
func TestDataInteiraNaoAceitaDuplicarIntervalosEConflitosSobrepostos(t *testing.T) {
	s := sessaoComResultado()
	s.lastResult.Mesas[0].Avaliadores = nil
	defer fecharBanco(s)
	if _, err := s.RealocarDia("quarta", "sexta 12-14", false); err == nil {
		t.Fatal("dia inteiro com intervalo de destino ambíguo aceito")
	}
	s.lastResult.Mesas[0].DiaNome = "28/10/2026 10:00-12:00"
	s.lastResult.Mesas[1].DiaNome = "28/10/2026 11:00-13:00"
	s.lastResult.Mesas[0].Avaliadores = []AvaliadorResultado{{ID: 10, Nome: "Prof"}}
	s.lastResult.Mesas[1].Avaliadores = s.lastResult.Mesas[0].Avaliadores
	op, err := s.RealocarMesa(2, "28/10/2026 11:00-13:00", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(op.Resultado.Alertas, " "), "sobrepostos") {
		t.Fatal("conflito de intervalos não sinalizado")
	}
	if ordemHorario("29/10/2026 10:00-12:00") < ordemHorario("28/10/2026 12:00-14:00") {
		t.Fatal("datas fora de ordem")
	}
}
