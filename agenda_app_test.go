package main

import (
	"bytes"
	"candidate_alocator/back/allocation"
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/logic"
	"candidate_alocator/back/operations"
	types "candidate_alocator/back/type"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/xuri/excelize/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func operationsApp(t *testing.T) *App {
	t.Helper()
	a := NewApp()
	a.operationsPath = filepath.Join(t.TempDir(), "operations.db")
	a.synchronousNotices = true
	t.Cleanup(a.clearTransientState)
	p := types.AllocationProblem{Candidates: []types.SolverCandidate{{ID: 1, Name: "Ana", PreferredGroupIDs: []int{1, 2}, PreferenceRankByGroupID: map[int]int{1: 0, 2: 1}, EvaluatorRestrictions: types.SolverCandidateRestrictions{ForbiddenEvaluatorIDs: []int{20}}}, {ID: 2, Name: "Bia", PreferredGroupIDs: []int{2}, PreferenceRankByGroupID: map[int]int{2: 0}}}, Groups: []types.SolverGroup{{ID: 1, Label: "quarta 14-16 grupo 1", ScheduleKey: "quarta 14-16", EvaluatorIDs: []int{10}, MinCandidates: 1, MaxCandidates: 1}, {ID: 2, Label: "quarta 16-18 grupo 2", ScheduleKey: "quarta 16-18", EvaluatorIDs: []int{20}, MinCandidates: 1, MaxCandidates: 1}}, HardRestrictions: types.SolverHardRestrictions{EnforceGroupCapacity: true, EnforceForbiddenEvaluators: true, RespectCandidatePreferences: true}, SoftRules: types.SolverSoftRules{PreferencePenaltyByRank: logic.BaseOptimizationPolicy().PreferencePenaltyByRank}}
	run := allocation.ConfiguredAllocationResult{Problem: p, Config: types.AllocationConfiguration{Normalized: types.NormalizedAllocationInput{Params: types.AllocationParams{AvaliadoresPorGrupo: 1, MinPessoasPorGrupo: 1, MaxPessoasPorGrupo: 1}}}, Candidatos: []types.Candidato{{Nome: "Ana", EmailPessoal: "ana@example.test", Numero: "123456789"}, {Nome: "Bia", EmailPessoal: "bia@example.test"}}, Avaliadores: []types.Avaliador{{ID: 10, Nome: "Eva", Email: "eva@example.test"}, {ID: 20, Nome: "Isa", Email: "isa@example.test"}}, Result: types.SolverResult{Status: "optimal", Assignments: map[int]int{1: 1, 2: 2}}}
	result := buildUIAllocationResult(run)
	result.Mesas[0].Candidatos[0].Telefone = "11999999999"
	if err := a.saveConfiguredAgenda(run, result); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAgendaManualActionsAndLazyMemoryRecovery(t *testing.T) {
	a := operationsApp(t)
	if err := a.startOperations(); err != nil {
		t.Fatal(err)
	}
	r, err := a.EvaluateCandidates([]int{1}, "aprovado")
	if err != nil || r.Mesas[0].Candidatos[0].Avaliacao != "aprovado" {
		t.Fatal(r, err)
	}
	if _, err := a.EvaluateCandidates([]int{1, 99}, "reprovado"); err == nil {
		t.Fatal("unknown ID accepted")
	}
	r, _ = a.GetAgenda()
	if r.Mesas[0].Candidatos[0].Avaliacao != "aprovado" {
		t.Fatal("evaluation not atomic")
	}
	suggestions, err := a.GetMoveSuggestions(1)
	if err != nil || len(suggestions) != 1 || len(suggestions[0].Alerts) == 0 {
		t.Fatal(suggestions, err)
	}
	op, err := a.MoveAgenda(MoveRequest{Kind: "person", CandidateID: 1, DestinationID: 2})
	if err != nil || len(op.Result.Mesas[1].Candidatos) != 2 || len(op.Result.HardViolations) == 0 {
		t.Fatal("manual exception blocked or unreported", op, err)
	}
	_, err = a.MoveAgenda(MoveRequest{Kind: "person", CandidateID: 1, Schedule: "sexta 8-10"})
	if err != nil {
		t.Fatal(err)
	}
	r, _ = a.GetAgenda()
	if len(r.Mesas) != 3 || len(r.Mesas[2].Avaliadores) != 0 || r.Mesas[2].Candidatos[0].Avaliacao != "aprovado" {
		t.Fatal("manual destination or rating lost", r)
	}
	a.evictIdleAgenda(time.Now().Add(agendaIdle * 2))
	if a.allocationResult.Mesas != nil {
		t.Fatal("agenda retained in memory")
	}
	r, err = a.GetAgenda()
	if err != nil || len(r.Mesas) != 3 {
		t.Fatal("lazy restore failed", err)
	}
	a.allocationRunning = true
	if _, err := a.MoveAgenda(MoveRequest{}); err == nil {
		t.Fatal("search can overwrite manual move")
	}
	a.allocationRunning = false
	second := NewApp()
	second.operationsPath = a.operationsPath
	t.Cleanup(second.clearTransientState)
	if err := second.startOperations(); err != nil {
		t.Fatal(err)
	}
	if state := second.GetAllocationRunState(true); !state.HasResult || state.Result == nil || len(state.Result.Mesas) != 3 {
		t.Fatal("restart lost agenda")
	}
}

func TestWholeDayAndExcelUseCurrentAgenda(t *testing.T) {
	a := operationsApp(t)
	op, err := a.MoveAgenda(MoveRequest{Kind: "day", From: "quarta", To: "sexta"})
	if err != nil {
		t.Fatal(err)
	}
	if op.Result.Mesas[0].Horario != "sexta 14-16" || op.Result.Mesas[1].Horario != "sexta 16-18" {
		t.Fatal("intervals changed", op)
	}
	if _, err := a.MoveAgenda(MoveRequest{Kind: "group", GroupID: 1, To: "sábado 10-12"}); err != nil {
		t.Fatal(err)
	}
	encoded, err := a.ExportAgendaExcel()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := base64.StdEncoding.DecodeString(encoded)
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = book.Close() }()
	rows, _ := book.GetRows(book.GetSheetName(0))
	if rows[1][1] != "sábado 10-12" {
		t.Fatal(rows)
	}
	if phoneFromExtras(map[string]string{"WhatsApp": "11999999999"}) == "" {
		t.Fatal("telephone extra not recognized")
	}
}

func TestChannelsMaskSecretsAndStoreDeliveriesSeparately(t *testing.T) {
	a := operationsApp(t)
	cfg := ConfigAvisos{Ativo: true, Mensagem: mensagemPadrao, Email: EmailConfig{Ativo: true, Provedor: "gmail", Remetente: "org@example.test", Senha: "unit-test-placeholder"}}
	if err := a.SaveChannelConfig(cfg); err != nil {
		t.Fatal(err)
	}
	public, _ := a.GetChannelConfig()
	if public.Email.Senha != "" || !public.Email.SenhaDefinida || public.Email.Host != "smtp.gmail.com" {
		t.Fatal("credential exposed or defaults missing", public)
	}
	public.Mensagem = "Olá {{nome}}"
	if err := a.SaveChannelConfig(public); err != nil {
		t.Fatal(err)
	}
	saved, _ := a.noticeConfig()
	if saved.Email.Senha != cfg.Email.Senha {
		t.Fatal("secret lost on message edit")
	}
	a.noticeSender = func(_ ConfigAvisos, d operations.Delivery) error {
		if d.PersonID == 1 {
			return errors.New("test transport failure")
		}
		return nil
	}
	id, err := a.SendAgendaNotices([]int{1}, 0)
	if err != nil {
		t.Fatal(err)
	}
	history, _ := a.GetNoticeHistory(0)
	if len(history.Items) != 1 || history.Items[0].Failed != 1 || history.Items[0].Accepted != 1 {
		t.Fatal(history)
	}
	details, _ := a.GetNoticeDetails(id, 0)
	if len(details.Items) != 2 {
		t.Fatal("candidate and evaluator required")
	}
	states, _ := a.GetNoticeStates()
	if states[1] != "failed" {
		t.Fatal(states)
	}
	a.noticeSender = func(_ ConfigAvisos, _ operations.Delivery) error { return nil }
	next, err := a.RetryNoticeFailures(id)
	if err != nil {
		t.Fatal(err)
	}
	repeated, _ := a.GetNoticeDetails(next, 0)
	if len(repeated.Items) != 1 || repeated.Items[0].Status != "accepted" {
		t.Fatal(repeated)
	}
	if _, err := a.RetryNoticeFailures(id); err == nil {
		t.Fatal("retry duplicated")
	}
	if err := a.DisconnectGoogle(); err != nil {
		t.Fatal(err)
	}
	items := prepareDeliveries(ConfigAvisos{WhatsApp: WhatsConfig{Ativo: true}}, UICandidate{Numero: "123456789"}, UIMesa{})
	if len(items) != 1 || items[0].Destination != "" || items[0].Error == "" {
		t.Fatal("RA treated as WhatsApp")
	}
}

func TestGoogleConsentUsesPKCEAndRejectsInvalidState(t *testing.T) {
	a := operationsApp(t)
	t.Setenv("CANDIDATE_GOOGLE_CLIENT_ID", "")
	if a.GetGoogleConnectionStatus().Available {
		t.Fatal("unconfigured OAuth available")
	}
	if _, err := a.BeginGoogleConnection(); err == nil {
		t.Fatal("missing client accepted")
	}
	t.Setenv("CANDIDATE_GOOGLE_CLIENT_ID", "unit-test-client")
	address, err := a.BeginGoogleConnection()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.googlePending.server.Close() }()
	u, _ := url.Parse(address)
	q := u.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("state") == "" || !strings.Contains(q.Get("scope"), "gmail.send") {
		t.Fatal("OAuth protections absent")
	}
	bad := httptest.NewRecorder()
	a.finishGoogleConnection(a.googlePending, bad, httptest.NewRequest(http.MethodGet, "/google/callback?state=invalid", nil))
	if bad.Code != 400 {
		t.Fatal("invalid state accepted")
	}
	if _, err := a.googleClient(EmailConfig{}); err == nil {
		t.Fatal("unconnected OAuth client allowed")
	}
}

func TestPreviewUsesRealAppAndRejectsUnknownMethods(t *testing.T) {
	a := operationsApp(t)
	router := previewRouter(a, fstest.MapFS{"index.html": {Data: []byte("<html><head></head></html>")}, "asset.js": {Data: []byte("ok")}})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRecorder()
		router.ServeHTTP(r, httptest.NewRequest(method, path, strings.NewReader(body)))
		return r
	}
	if r := request("GET", "/allocation-result", ""); r.Code != 200 || !strings.Contains(r.Body.String(), "bootstrap.js") {
		t.Fatal(r.Body.String())
	}
	if r := request("GET", "/asset.js", ""); r.Body.String() != "ok" {
		t.Fatal(r.Body.String())
	}
	if r := request("POST", "/preview/call/GetAgenda", "[]"); r.Code != 200 || !strings.Contains(r.Body.String(), "Ana") {
		t.Fatal(r.Body.String())
	}
	for _, test := range []struct{ method, body string }{{"shutdown", "[]"}, {"EvaluateCandidates", "[]"}, {"GetAgenda", "invalid"}} {
		if r := request("POST", "/preview/call/"+test.method, test.body); r.Code < 400 {
			t.Fatal("invalid request accepted")
		}
	}
	if err := runAgendaCLI(nil); err == nil {
		t.Fatal("missing command accepted")
	}
	if err := a.SaveChannelConfig(ConfigAvisos{Mensagem: "{{unknown}}"}); err == nil {
		t.Fatal("unknown variable accepted")
	}
}

func TestMessageRenderingAndValidation(t *testing.T) {
	c := UICandidate{Nome: "Ana", EmailPessoal: "ana@example.test", Numero: "123456789", Telefone: "11999999999"}
	m := UIMesa{Horario: "sexta 14-16", Descricao: "Mesa 1"}
	if got := renderMensagem("{{nome}} {{data}} {{horario}}", c, m); got != "Ana sexta 14-16" {
		t.Fatal(got)
	}
	if err := validarMensagem("Olá {{nome}}"); err != nil {
		t.Fatal(err)
	}
	if phone, err := normalizarTelefone(c.Telefone); err != nil || phone != "5511999999999" {
		t.Fatal(phone, err)
	}
	data, _ := json.Marshal(ConfigAvisos{Email: EmailConfig{RefreshToken: "unit-placeholder"}})
	if len(data) == 0 {
		t.Fatal("private storage serialization failed")
	}
}

func TestAutomaticNoticeRequiresExplicitAuthorization(t *testing.T) {
	a := operationsApp(t)
	calls := 0
	a.noticeSender = func(_ ConfigAvisos, _ operations.Delivery) error { calls++; return nil }
	cfg := ConfigAvisos{Ativo: true, Mensagem: mensagemPadrao, Email: EmailConfig{Ativo: true, Provedor: "gmail", Remetente: "org@example.test"}}
	if err := a.SaveChannelConfig(cfg); err != nil {
		t.Fatal(err)
	}
	a.maybeAutoNotify()
	if calls != 0 {
		t.Fatal("sent without consent")
	}
	cfg.EnviarAoAlocar = true
	if err := a.SaveChannelConfig(cfg); err != nil {
		t.Fatal(err)
	}
	a.maybeAutoNotify()
	if calls != 4 {
		t.Fatalf("expected candidates and evaluators, got %d", calls)
	}
}

func TestAgendaCLIReadsAndChangesSameSavedAgenda(t *testing.T) {
	a := operationsApp(t)
	t.Setenv("CANDIDATE_ALLOCATOR_DATA_DIR", filepath.Dir(a.operationsPath))
	for _, args := range [][]string{{"resultado"}, {"avaliar", "-pessoa", "1", "-resultado", "aprovado"}, {"sugestoes", "-pessoa", "1"}, {"historico"}, {"detalhes", "-lote", "1"}, {"canais"}, {"testar-canais"}, {"exportar", "-arquivo", filepath.Join(t.TempDir(), "agenda.xlsx")}} {
		if err := runAgendaCLI(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	snapshot, err := a.readAgenda()
	if err != nil || snapshot.Result.Mesas[0].Candidatos[0].Avaliacao != "aprovado" {
		t.Fatal("CLI changed a separate agenda", err)
	}
	if err := runAgendaCLI([]string{"realocar-dia", "-de", "quarta", "-para", "sexta", "-avisar"}); err == nil {
		t.Fatal("notice setup failure not reflected in exit status")
	}
	snapshot, _ = a.readAgenda()
	if snapshot.Result.Mesas[0].Horario != "sexta 14-16" {
		t.Fatal("notice failure reverted relocation")
	}
	for _, args := range [][]string{{"unknown"}, {"enviar"}, {"previa"}, {"repetir-falhas", "-lote", "99"}, {"exportar"}} {
		if err := runAgendaCLI(args); err == nil {
			t.Fatalf("invalid command succeeded: %v", args)
		}
	}
}

func TestManualAssignmentFromEntirelyUnallocatedResult(t *testing.T) {
	a := operationsApp(t)
	snapshot, err := a.readAgenda()
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Result.NaoAlocados = []UICandidate{snapshot.Result.Mesas[0].Candidatos[0], snapshot.Result.Mesas[1].Candidatos[0]}
	snapshot.Result.Mesas = []UIMesa{}
	snapshot.Run.Result.Assignments = nil
	if err := a.saveConfiguredAgenda(snapshot.Run, snapshot.Result); err != nil {
		t.Fatal(err)
	}
	op, err := a.MoveAgenda(MoveRequest{Kind: "person", CandidateID: 1, Schedule: "sexta 14-16"})
	if err != nil || len(op.Result.NaoAlocados) != 1 || len(op.Result.Mesas) != 1 {
		t.Fatal(op, err)
	}
}

func TestCompletedSearchCanBeEvictedWithoutOpeningResultAgain(t *testing.T) {
	a := operationsApp(t)
	snapshot, err := a.readAgenda()
	if err != nil {
		t.Fatal(err)
	}
	a.allocationRunning = true
	a.allocationRunID = 1
	if !a.completeAllocationRun(1, snapshot.Result) || a.allocationAccess.IsZero() {
		t.Fatal("completed search has no inactivity timestamp")
	}
	a.evictIdleAgenda(time.Now().Add(agendaIdle * 2))
	if a.allocationResult.Mesas != nil {
		t.Fatal("event-delivered result never evicted")
	}
	if result, err := a.GetAgenda(); err != nil || len(result.Mesas) != 2 {
		t.Fatal(result, err)
	}
}

func TestVisualImportKeepsRestrictionsUntilEvaluatorIDsExist(t *testing.T) {
	t.Setenv("CANDIDATE_ALLOCATOR_DATABASE", filepath.Join(t.TempDir(), "import.db"))
	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		t.Fatal(err)
	}
	db, err := dbpkg.OpenDefault()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := dbpkg.InsertStruct(db, "pessoa", types.Candidato{Nome: "Ana", Semestre: "1"}, types.CandidateFields()); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	t.Cleanup(a.clearTransientState)
	if err := a.SaveUsuariosFromMaps([]map[string]interface{}{{"nome": "Ana", "semestre": "1"}}); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveRestricoesFromMaps([]map[string]interface{}{{"candidato": "Ana", "naoPosso": "EV", "prefiroNao": ""}}); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveAvaliadoresFromMaps([]map[string]interface{}{{"nome": "Eva", "email": "eva@example.test", "sigla": "EV"}}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM restricoesNposso").Scan(&count); err != nil || count != 1 {
		t.Fatal("visual import lost evaluator restriction", count, err)
	}
}

func TestDayActionCanSelectWholeDayFromAnIntervalCard(t *testing.T) {
	a := operationsApp(t)
	op, err := a.MoveAgenda(MoveRequest{Kind: "day", Scope: "day", From: "quarta 14-16", To: "sexta"})
	if err != nil {
		t.Fatal(err)
	}
	if op.Result.Mesas[0].Horario != "sexta 14-16" || op.Result.Mesas[1].Horario != "sexta 16-18" {
		t.Fatal("whole-day action only moved one interval")
	}
}
