package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"candidate_alocator/back/allocation"
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	allocationProgressEvent = "allocation:progress"
	allocationSolutionEvent = "allocation:solution"
	allocationCompleteEvent = "allocation:complete"
	allocationErrorEvent    = "allocation:error"
)

type UsuariosResponse = logic.UsuariosResponse

// App struct
type App struct {
	ctx                           context.Context
	excelData                     []byte
	nOpcoes                       int
	allocationMu                  sync.Mutex
	allocationCancel              context.CancelFunc
	allocationRunning             bool
	allocationRunID               uint64
	allocationProgress            allocation.SolverProgress
	allocationResult              UIAllocationResult
	allocationHasResult           bool
	allocationError               string
	allocationProgressUnsubscribe func()
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.allocationProgressUnsubscribe = runtime.EventsOn(ctx, allocationProgressEvent, func(_ ...interface{}) {})
	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		fmt.Println("Erro ao preparar banco:", err)
		panic(err)
	}

	db, err := dbpkg.OpenDefault()
	if err != nil {
		fmt.Println("Erro ao abrir banco para limpeza:", err)
		panic(err)
	}
	defer func() { _ = db.Close() }()

	if err := dbpkg.ClearDatabase(db); err != nil {
		fmt.Println("Erro ao limpar banco:", err)
		panic(err)
	}

	writeWailsSmokeSentinel()
}

// beforeClose clears transient state before the app exits.
func (a *App) beforeClose(_ context.Context) (prevent bool) {
	a.clearTransientState()
	return false
}

// shutdown clears transient state and removes the smoke sentinel.
func (a *App) shutdown(_ context.Context) {
	a.clearTransientState()
	removeWailsSmokeSentinel()
}

// O fluxo equivalente via terminal foi movido para cli.go.
// Use: go run . cli -file arquivo.xlsx

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) SuggestMapping(data []byte, quantidadeOpcoes int) ([]types.MappingItem, error) {
	a.excelData = data
	a.nOpcoes = quantidadeOpcoes
	return logic.SuggestMapping(data, quantidadeOpcoes)
}

func (a *App) SuggestMappingAvaliador() ([]types.MappingItem, error) {
	return logic.SuggestMappingAvaliador(a.excelData)
}

func (a *App) SuggestMappingRestricao() ([]types.MappingItem, error) {
	return logic.SuggestMappingRestricao(a.excelData)
}

func (a *App) GetCandidateMappingFieldInfos() []types.MappingFieldInfo {
	return types.CandidateMappingFieldInfos(a.nOpcoes)
}

func (a *App) GetAvaliadorMappingFieldInfos() []types.MappingFieldInfo {
	return types.AvaliadorMappingFieldInfos()
}

func (a *App) GetRestricaoMappingFieldInfos() []types.MappingFieldInfo {
	return types.RestricaoMappingFieldInfos()
}

func ProcessMapping(items []string) ([]types.MappingItem, error) {
	return logic.ProcessMapping(items)
}

func (a *App) BuildUsuariosWithMapping(mappingItems []types.MappingItem) (UsuariosResponse, error) {
	return logic.BuildUsuariosWithMapping(a.excelData, a.nOpcoes, mappingItems)
}

func (a *App) BuildAvaliadoresWithMapping(mappingItems []types.MappingItem) (logic.AvaliadoresResponse, error) {
	return logic.BuildAvaliadoresWithMapping(a.excelData, mappingItems)
}

func (a *App) BuildRestricoesWithMapping(mappingItems []types.MappingItem) ([]types.Restricao, error) {
	return logic.BuildRestricoesWithMapping(a.excelData, mappingItems)
}

func (a *App) SaveRestricoesFromMaps(restricaoMaps []map[string]interface{}) error {
	return logic.SaveRestricoesFromMaps(restricaoMaps)
}

func (a *App) SaveUsuariosFromMaps(candidatoMaps []map[string]interface{}) error {
	return logic.SaveUsuariosFromMaps(candidatoMaps)
}

func (a *App) SaveAvaliadoresFromMaps(avaliadorMaps []map[string]interface{}) error {
	return logic.SaveAvaliadoresFromMaps(avaliadorMaps)
}

func (a *App) DetectUniquePreferenceValues(candidatos []types.Candidato) []types.UniqueValueDetection {
	return logic.DetectUniquePreferenceValues(candidatos)
}

func (a *App) ListCandidateCriterionColumns(candidatos []types.Candidato) []types.CandidateCriterionColumn {
	return logic.ListCandidateCriterionColumns(candidatos)
}

func (a *App) DetectUniqueCandidateColumnValues(candidatos []types.Candidato, columnKey string) ([]types.UniqueValueDetection, error) {
	return logic.DetectUniqueCandidateColumnValues(candidatos, columnKey)
}

func (a *App) DefaultAllocationParams() types.AllocationParams {
	return logic.DefaultAllocationParams()
}

func (a *App) GetWorkflowDefinition() types.WorkflowDefinition {
	return logic.WorkflowDefinition()
}

func (a *App) GetSoftCriterionOptions() []types.SoftCriterionOption {
	return logic.SoftCriterionOptions()
}

func (a *App) NormalizePreferenceScheduleMappings(mappings []types.PreferenceScheduleMapping) []types.PreferenceScheduleMapping {
	return logic.NormalizePreferenceScheduleMappings(mappings)
}

func (a *App) NormalizeSoftCriteria(criteria []types.SoftCriterion) []types.SoftCriterion {
	return logic.NormalizeSoftCriteria(criteria)
}

func (a *App) BuildAllocationConfiguration(
	detections []types.UniqueValueDetection,
	mappings []types.PreferenceScheduleMapping,
	params types.AllocationParams,
	candidatos []types.Candidato,
) (types.AllocationConfiguration, error) {
	return logic.BuildAllocationConfiguration(detections, mappings, params, candidatos)
}

func (a *App) ValidatePreferenceScheduleMappings(mappings []types.PreferenceScheduleMapping) error {
	return logic.ValidatePreferenceScheduleMappings(mappings)
}

func (a *App) ValidateSoftCriteria(criteria []types.SoftCriterion, candidatos []types.Candidato) error {
	return logic.ValidateSoftCriteria(criteria, candidatos)
}

func (a *App) ValidateAllocationParams(params types.AllocationParams, candidatos []types.Candidato) error {
	return logic.ValidateAllocationParams(params, candidatos)
}

func (a *App) CountPossibleAllocationQuantities(params types.AllocationParams, totalPeople int) int {
	return logic.CountPossibleAllocationQuantities(params, totalPeople)
}

func (a *App) CountPossibleAllocationQuantitiesAcrossSchedules(params types.AllocationParams, totalPeople, scheduleCount int) int {
	return logic.CountPossibleAllocationQuantitiesAcrossSchedules(params, totalPeople, scheduleCount)
}

type UICandidate struct {
	ID              int               `json:"id"`
	Nome            string            `json:"nome"`
	Semestre        int               `json:"semestre"`
	Curso           string            `json:"curso"`
	EmailPessoal    string            `json:"emailPessoal"`
	EmailSecundario string            `json:"emailSecundario"`
	Opcoes          []string          `json:"opcoes"`
	NaoPosso        []string          `json:"naoPosso"`
	PrefiroNao      []string          `json:"prefiroNao"`
	Extras          map[string]string `json:"extras"`
}

type UIAvaliador struct {
	ID         int               `json:"id"`
	Nome       string            `json:"nome"`
	Sigla      string            `json:"sigla"`
	Email      string            `json:"email"`
	NaoPosso   []string          `json:"naoPosso"`
	PrefiroNao []string          `json:"prefiroNao"`
	Extras     map[string]string `json:"extras"`
}

type UIMesa struct {
	ID          int           `json:"id"`
	Horario     string        `json:"horario"`
	Descricao   string        `json:"descricao"`
	Candidatos  []UICandidate `json:"candidatos"`
	Avaliadores []UIAvaliador `json:"avaliadores"`
}

type UIAllocationResult struct {
	Status          string                          `json:"status"`
	SolverStatus    string                          `json:"solverStatus"`
	Mesas           []UIMesa                        `json:"mesas"`
	NaoAlocados     []UICandidate                   `json:"naoAlocados"`
	Score           types.SoftScoreBreakdown        `json:"score"`
	Quality         types.AllocationQualityReport   `json:"quality"`
	HardViolations  []types.HardConstraintViolation `json:"hardViolations"`
	RejectionReason string                          `json:"rejectionReason"`
	Metrics         types.SolverMetrics             `json:"metrics"`
	DebugNotes      []string                        `json:"debugNotes"`
}

// UIAllocationRunState is the authoritative snapshot used by the frontend to
// recover if a Wails event is missed or the page is reloaded during a search.
type UIAllocationRunState struct {
	RunID     uint64                    `json:"runId"`
	Running   bool                      `json:"running"`
	HasResult bool                      `json:"hasResult"`
	Progress  allocation.SolverProgress `json:"progress"`
	Result    *UIAllocationResult       `json:"result,omitempty"`
	Error     string                    `json:"error"`
}

// GetAllocationRunState returns the latest backend-owned execution snapshot.
func (a *App) GetAllocationRunState(includeResult bool) UIAllocationRunState {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	state := UIAllocationRunState{
		RunID: a.allocationRunID, Running: a.allocationRunning,
		HasResult: a.allocationHasResult, Progress: a.allocationProgress,
		Error: a.allocationError,
	}
	if includeResult && a.allocationHasResult {
		result := a.allocationResult
		state.Result = &result
	}
	return state
}

func (a *App) GetCriteriaOptions() (map[string][]string, error) {
	db, err := dbpkg.OpenDefault()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query("SELECT semestre, curso, extras FROM pessoa")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	optsMap := make(map[string]map[string]bool)
	optsMap["semestre"] = make(map[string]bool)
	optsMap["curso"] = make(map[string]bool)

	for rows.Next() {
		var semestre int
		var curso string
		var extras sql.NullString
		if err := rows.Scan(&semestre, &curso, &extras); err == nil {
			optsMap["semestre"][strconv.Itoa(semestre)] = true
			if curso != "" {
				optsMap["curso"][curso] = true
			}
			if extras.Valid && extras.String != "" {
				var extMap map[string]interface{}
				if err := json.Unmarshal([]byte(extras.String), &extMap); err == nil {
					for k, v := range extMap {
						if vStr, ok := v.(string); ok && vStr != "" {
							key := "extras." + k
							if optsMap[key] == nil {
								optsMap[key] = make(map[string]bool)
							}
							optsMap[key][vStr] = true
						}
					}
				}
			}
		}
	}

	result := make(map[string][]string)
	for k, vMap := range optsMap {
		var vals []string
		for v := range vMap {
			vals = append(vals, v)
		}
		sort.Strings(vals)
		result[k] = vals
	}

	return result, nil
}

func (a *App) BuildAllocationConfigurationFromDatabase(params types.AllocationParams) (types.AllocationConfiguration, error) {
	db, err := dbpkg.OpenDefault()
	if err != nil {
		return types.AllocationConfiguration{}, err
	}
	defer func() { _ = db.Close() }()

	return allocation.BuildConfigurationFromPersistedData(db, params)
}

func (a *App) RunAllocation(config types.AllocationConfiguration) (UIAllocationResult, error) {
	db, err := dbpkg.OpenDefault()
	if err != nil {
		return UIAllocationResult{}, err
	}
	defer func() { _ = db.Close() }()

	var progress allocation.ProgressCallback
	if a.ctx != nil {
		progress = func(snapshot allocation.SolverProgress) {
			runtime.EventsEmit(a.ctx, allocationProgressEvent, snapshot)
		}
	}
	run, err := allocation.RunConfiguredAllocation(db, config, progress)
	if err != nil {
		return UIAllocationResult{}, err
	}
	return buildUIAllocationResult(run), nil
}

// StartAllocation starts a background exact search. Improving feasible
// solutions are emitted while the final RunAllocation API remains available to
// synchronous callers.
func (a *App) StartAllocation(config types.AllocationConfiguration) error {
	a.allocationMu.Lock()
	if a.allocationRunning {
		a.allocationMu.Unlock()
		return fmt.Errorf("uma alocacao ja esta em execucao")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.allocationRunID++
	runID := a.allocationRunID
	a.allocationCancel = cancel
	a.allocationRunning = true
	a.allocationProgress = allocation.SolverProgress{}
	a.allocationResult = UIAllocationResult{}
	a.allocationHasResult = false
	a.allocationError = ""
	eventContext := a.ctx
	a.allocationMu.Unlock()

	go func() {
		db, err := dbpkg.OpenDefault()
		if err != nil {
			if a.failAllocationRun(runID, err.Error()) && eventContext != nil {
				runtime.EventsEmit(eventContext, allocationErrorEvent, err.Error())
			}
			return
		}
		defer func() { _ = db.Close() }()

		callbacks := allocation.ConfiguredAllocationCallbacks{
			Progress: func(snapshot allocation.SolverProgress) {
				if a.updateAllocationProgress(runID, snapshot) && eventContext != nil {
					runtime.EventsEmit(eventContext, allocationProgressEvent, snapshot)
				}
			},
			Solution: func(snapshot allocation.ConfiguredAllocationResult) {
				uiResult := buildUIAllocationResult(snapshot)
				if a.updateAllocationSolution(runID, uiResult) && eventContext != nil {
					runtime.EventsEmit(eventContext, allocationSolutionEvent, uiResult)
				}
			},
		}
		run, runErr := allocation.RunConfiguredAllocationStreaming(ctx, db, config, callbacks)
		if runErr != nil {
			if a.failAllocationRun(runID, runErr.Error()) && eventContext != nil {
				runtime.EventsEmit(eventContext, allocationErrorEvent, runErr.Error())
			}
			return
		}
		uiResult := buildUIAllocationResult(run)
		if a.completeAllocationRun(runID, uiResult) && eventContext != nil {
			runtime.EventsEmit(eventContext, allocationCompleteEvent, uiResult)
		}
	}()
	return nil
}

// StopAllocation cancels the proof search and keeps the best feasible solution
// already published to the UI.
func (a *App) StopAllocation() bool {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	if !a.allocationRunning || a.allocationCancel == nil {
		return false
	}
	cancel := a.allocationCancel
	a.allocationCancel = nil
	cancel()
	return true
}

func (a *App) finishAllocationRun(runID uint64) {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	if a.allocationRunID != runID {
		return
	}
	a.allocationRunning = false
	a.allocationCancel = nil
}

func (a *App) updateAllocationProgress(runID uint64, progress allocation.SolverProgress) bool {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	if a.allocationRunID != runID || !a.allocationRunning {
		return false
	}
	a.allocationProgress = progress
	return true
}

func (a *App) updateAllocationSolution(runID uint64, result UIAllocationResult) bool {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	if a.allocationRunID != runID || !a.allocationRunning {
		return false
	}
	if a.allocationHasResult && !betterUIAllocationResult(result, a.allocationResult) {
		return false
	}
	a.allocationResult = result
	a.allocationHasResult = true
	return true
}

// betterUIAllocationResult compares provisional results lexicographically:
// first allocate more people, then reduce hard errors, then reduce soft score.
// Equal results are deliberately not republished to the frontend.
func betterUIAllocationResult(candidate, current UIAllocationResult) bool {
	if len(candidate.NaoAlocados) != len(current.NaoAlocados) {
		return len(candidate.NaoAlocados) < len(current.NaoAlocados)
	}
	if len(candidate.HardViolations) != len(current.HardViolations) {
		return len(candidate.HardViolations) < len(current.HardViolations)
	}
	return candidate.Score.TotalPenalty < current.Score.TotalPenalty
}

func (a *App) completeAllocationRun(runID uint64, result UIAllocationResult) bool {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	if a.allocationRunID != runID || !a.allocationRunning {
		return false
	}
	a.allocationRunning = false
	a.allocationCancel = nil
	a.allocationResult = result
	a.allocationHasResult = true
	a.allocationError = ""
	return true
}

func (a *App) failAllocationRun(runID uint64, message string) bool {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	if a.allocationRunID != runID || !a.allocationRunning {
		return false
	}
	a.allocationRunning = false
	a.allocationCancel = nil
	a.allocationError = message
	return true
}

func buildUIAllocationResult(run allocation.ConfiguredAllocationResult) UIAllocationResult {

	evaluatorMap := buildUIEvaluatorMap(run.Avaliadores, run.Problem.Candidates)
	candMap := buildUICandidateMap(run.Candidatos, run.Problem.Candidates, evaluatorMap)
	uiMesasMap := buildUIMesaMap(run.Problem.Groups, evaluatorMap)
	var uiMesas []UIMesa
	alocadosIDs := make(map[int]bool)

	for candID, groupID := range run.Result.Assignments {
		m := uiMesasMap[groupID]
		m.Candidatos = append(m.Candidatos, candMap[candID])
		uiMesasMap[groupID] = m
		alocadosIDs[candID] = true
	}

	for _, m := range uiMesasMap {
		if len(m.Candidatos) > 0 {
			sort.SliceStable(m.Candidatos, func(i, j int) bool {
				return m.Candidatos[i].Nome < m.Candidatos[j].Nome
			})
			uiMesas = append(uiMesas, m)
		}
	}
	sort.SliceStable(uiMesas, func(i, j int) bool {
		return uiMesas[i].ID < uiMesas[j].ID
	})

	var naoAlocados []UICandidate
	for cid, c := range candMap {
		if !alocadosIDs[cid] {
			naoAlocados = append(naoAlocados, c)
		}
	}
	sort.SliceStable(naoAlocados, func(i, j int) bool {
		return naoAlocados[i].Nome < naoAlocados[j].Nome
	})

	status := "Sucesso!"
	if len(naoAlocados) > 0 {
		status = "Alocação Parcial"
	}
	switch run.Result.Status {
	case "infeasible":
		status = "Impossível (Infeasible)"
	case "partial":
		status = "Alocação parcial"
	case "feasible":
		status = "Solução provisória"
	case "cancelled":
		status = "Execução interrompida"
	}

	return UIAllocationResult{
		Status:          status,
		SolverStatus:    run.Result.Status,
		Mesas:           uiMesas,
		NaoAlocados:     naoAlocados,
		Score:           run.Result.Score,
		Quality:         allocation.BuildAllocationQualityReport(run.Problem, run.Result),
		HardViolations:  run.Result.HardViolations,
		RejectionReason: run.Result.RejectionReason,
		Metrics:         run.Result.Metrics,
		DebugNotes:      run.Result.DebugNotes,
	}
}

func buildUICandidateMap(candidatos []types.Candidato, solverCandidates []types.SolverCandidate, evaluators map[int]UIAvaliador) map[int]UICandidate {
	solverByID := make(map[int]types.SolverCandidate, len(solverCandidates))
	for _, candidate := range solverCandidates {
		solverByID[candidate.ID] = candidate
	}

	result := make(map[int]UICandidate, len(candidatos))
	for idx, candidato := range candidatos {
		id := idx + 1
		semestre, _ := strconv.Atoi(strings.TrimSpace(candidato.Semestre))
		solverCandidate := solverByID[id]
		naoPosso := make([]string, 0, len(solverCandidate.EvaluatorRestrictions.ForbiddenEvaluatorIDs))
		for _, evaluatorID := range solverCandidate.EvaluatorRestrictions.ForbiddenEvaluatorIDs {
			if evaluator, ok := evaluators[evaluatorID]; ok {
				naoPosso = append(naoPosso, evaluator.Nome+" ("+evaluator.Sigla+")")
			}
		}
		prefiroNao := make([]string, 0, len(solverCandidate.EvaluatorRestrictions.AvoidEvaluatorIDs))
		for _, evaluatorID := range solverCandidate.EvaluatorRestrictions.AvoidEvaluatorIDs {
			if evaluator, ok := evaluators[evaluatorID]; ok {
				prefiroNao = append(prefiroNao, evaluator.Nome+" ("+evaluator.Sigla+")")
			}
		}
		result[id] = UICandidate{
			ID:              id,
			Nome:            candidato.Nome,
			Semestre:        semestre,
			Curso:           candidato.Curso,
			EmailPessoal:    candidato.EmailPessoal,
			EmailSecundario: candidato.EmailSecundario,
			Opcoes:          append([]string(nil), candidato.Opcoes...),
			NaoPosso:        naoPosso,
			PrefiroNao:      prefiroNao,
			Extras:          nullableExtrasToStrings(candidato.Extras),
		}
	}
	return result
}

func buildUIEvaluatorMap(avaliadores []types.Avaliador, solverCandidates []types.SolverCandidate) map[int]UIAvaliador {
	result := make(map[int]UIAvaliador, len(avaliadores))
	for _, avaliador := range avaliadores {
		result[avaliador.ID] = UIAvaliador{
			ID:     avaliador.ID,
			Nome:   avaliador.Nome,
			Sigla:  avaliador.Sigla,
			Email:  avaliador.Email,
			Extras: nullableExtrasToStrings(avaliador.Extras),
		}
	}
	for _, candidate := range solverCandidates {
		for _, evaluatorID := range candidate.EvaluatorRestrictions.ForbiddenEvaluatorIDs {
			evaluator := result[evaluatorID]
			evaluator.NaoPosso = append(evaluator.NaoPosso, candidate.Name)
			result[evaluatorID] = evaluator
		}
		for _, evaluatorID := range candidate.EvaluatorRestrictions.AvoidEvaluatorIDs {
			evaluator := result[evaluatorID]
			evaluator.PrefiroNao = append(evaluator.PrefiroNao, candidate.Name)
			result[evaluatorID] = evaluator
		}
	}
	return result
}

func nullableExtrasToStrings(extras map[string]*types.NullableString) map[string]string {
	result := make(map[string]string, len(extras))
	for key, value := range extras {
		if value == nil {
			continue
		}
		result[key] = string(*value)
	}
	return result
}

func buildUIMesaMap(groups []types.SolverGroup, evaluators map[int]UIAvaliador) map[int]UIMesa {
	result := make(map[int]UIMesa, len(groups))
	for _, group := range groups {
		uiEvaluators := make([]UIAvaliador, 0, len(group.EvaluatorIDs))
		for _, evaluatorID := range group.EvaluatorIDs {
			if evaluator, ok := evaluators[evaluatorID]; ok {
				uiEvaluators = append(uiEvaluators, evaluator)
			}
		}

		horario, descricao := splitSolverGroupLabel(group.Label)
		result[group.ID] = UIMesa{
			ID:          group.ID,
			Horario:     horario,
			Descricao:   descricao,
			Avaliadores: uiEvaluators,
			Candidatos:  []UICandidate{},
		}
	}
	return result
}

func splitSolverGroupLabel(label string) (string, string) {
	label = strings.TrimSpace(label)
	parts := strings.Split(label, " grupo ")
	if len(parts) != 2 {
		return label, label
	}
	return strings.TrimSpace(parts[0]), "Mesa " + strings.TrimSpace(parts[1])
}

func writeWailsSmokeSentinel() {
	path := os.Getenv("CANDIDATE_ALLOCATOR_WAILS_SMOKE_FILE")
	if path == "" {
		return
	}

	if err := os.WriteFile(filepath.Clean(path), []byte("ok"), 0o600); err != nil {
		fmt.Println("Erro ao escrever sentinel do smoke test do Wails:", err)
	}
}

func removeWailsSmokeSentinel() {
	path := os.Getenv("CANDIDATE_ALLOCATOR_WAILS_SMOKE_FILE")
	if path == "" {
		return
	}

	if err := os.Remove(filepath.Clean(path)); err != nil && !os.IsNotExist(err) {
		fmt.Println("Erro ao remover sentinel do smoke test do Wails:", err)
	}
}

func (a *App) clearTransientState() {
	a.allocationMu.Lock()
	if a.allocationCancel != nil {
		a.allocationCancel()
	}
	a.allocationCancel = nil
	a.allocationRunning = false
	unsubscribe := a.allocationProgressUnsubscribe
	a.allocationProgressUnsubscribe = nil
	a.allocationMu.Unlock()
	if unsubscribe != nil {
		unsubscribe()
	}
	a.ctx = nil
	a.excelData = nil
	a.nOpcoes = 0
}
