package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"

	"candidate_alocator/back/allocation"
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
)

type UsuariosResponse = logic.UsuariosResponse

// App struct
type App struct {
	ctx       context.Context
	excelData []byte
	nOpcoes   int
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		fmt.Println("Erro ao preparar banco:", err)
		panic(err)
	}

	db, err := dbpkg.OpenDefault()
	if err != nil {
		fmt.Println("Erro ao abrir banco para limpeza:", err)
		panic(err)
	}
	defer db.Close()

	if err := dbpkg.ClearDatabase(db); err != nil {
		fmt.Println("Erro ao limpar banco:", err)
		panic(err)
	}

	writeWailsSmokeSentinel()
}

// domReady is called after front-end resources have been loaded
func (a App) domReady(ctx context.Context) {
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
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
	ID       int    `json:"id"`
	Nome     string `json:"nome"`
	Semestre int    `json:"semestre"`
	Curso    string `json:"curso"`
}

type UIAvaliador struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Sigla string `json:"sigla"`
}

type UIMesa struct {
	ID          int           `json:"id"`
	Horario     string        `json:"horario"`
	Descricao   string        `json:"descricao"`
	Candidatos  []UICandidate `json:"candidatos"`
	Avaliadores []UIAvaliador `json:"avaliadores"`
}

type UIAllocationResult struct {
	Status      string        `json:"status"`
	Mesas       []UIMesa      `json:"mesas"`
	NaoAlocados []UICandidate `json:"naoAlocados"`
}

func (a *App) GetCriteriaOptions() (map[string][]string, error) {
	db, err := dbpkg.OpenDefault()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT semestre, curso, extras FROM pessoa")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (a *App) RunAllocation(params types.AllocationParams) (UIAllocationResult, error) {
	db, err := dbpkg.OpenDefault()
	if err != nil {
		return UIAllocationResult{}, err
	}
	defer db.Close()

	// 1. Fetch Horarios
	horarios, err := allocation.CarregarHorarios(db)
	if err != nil {
		return UIAllocationResult{}, err
	}

	// 2. Fetch Avaliadores
	avals, err := allocation.CarregarAvaliadores(db)
	if err != nil {
		return UIAllocationResult{}, err
	}

	// 3. Fetch Restricoes (Nposso and PrefiroN)
	restrNposso, err := allocation.CarregarRestricoes(db)
	if err != nil {
		return UIAllocationResult{}, err
	}

	restrPrefiroN := make(map[int]map[int]bool)
	rows, err := db.Query(`SELECT avaliador_id, candidato_id FROM restricoesPrefiroN`)
	if err == nil {
		for rows.Next() {
			var aid, cid int
			if rows.Scan(&aid, &cid) == nil {
				if restrPrefiroN[aid] == nil {
					restrPrefiroN[aid] = make(map[int]bool)
				}
				restrPrefiroN[aid][cid] = true
			}
		}
		rows.Close()
	}

	// 4. Create Groups (mesas) using params
	var solverGroups []types.SolverGroup
	uiMesasMap := make(map[int]UIMesa)

	rand.Seed(time.Now().UnixNano())

	for _, h := range horarios {
		for i := 0; i < params.GruposPorHorario; i++ {
			groupID := h.ID*100 + i

			n := params.AvaliadoresPorGrupo
			if len(avals) < n {
				n = len(avals)
			}

			shuffled := make([]*types.Avaliador, len(avals))
			copy(shuffled, avals)
			rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

			var evalIDs []int
			var uiEvals []UIAvaliador
			for k := 0; k < n; k++ {
				evalIDs = append(evalIDs, shuffled[k].ID)
				uiEvals = append(uiEvals, UIAvaliador{
					ID:    shuffled[k].ID,
					Nome:  shuffled[k].Nome,
					Sigla: shuffled[k].Sigla,
				})
			}

			solverGroups = append(solverGroups, types.SolverGroup{
				ID:            groupID,
				Label:         fmt.Sprintf("%s - Mesa %d", h.Descricao, i+1),
				EvaluatorIDs:  evalIDs,
				MinCandidates: params.MinPessoasPorGrupo,
				MaxCandidates: params.MaxPessoasPorGrupo,
			})

			uiMesasMap[groupID] = UIMesa{
				ID:          groupID,
				Horario:     h.Descricao,
				Descricao:   fmt.Sprintf("Mesa %d", i+1),
				Avaliadores: uiEvals,
				Candidatos:  []UICandidate{},
			}
		}
	}

	// 5. Fetch Candidates (pessoa) and their preferences (disponibilidade)
	prefs, err := allocation.CarregarDisponibilidades(db, horarios)
	if err != nil {
		return UIAllocationResult{}, err
	}

	candRows, err := db.Query("SELECT id, nome, semestre, curso FROM pessoa")
	if err != nil {
		return UIAllocationResult{}, err
	}
	defer candRows.Close()

	var solverCandidates []types.SolverCandidate
	candMap := make(map[int]UICandidate)

	for candRows.Next() {
		var id, semestre int
		var nome, curso string
		if err := candRows.Scan(&id, &nome, &semestre, &curso); err != nil {
			continue
		}

		candMap[id] = UICandidate{
			ID:       id,
			Nome:     nome,
			Semestre: semestre,
			Curso:    curso,
		}

		var preferredGroupIDs []int
		if hIDs, ok := prefs[id]; ok {
			for _, hID := range hIDs {
				for i := 0; i < params.GruposPorHorario; i++ {
					preferredGroupIDs = append(preferredGroupIDs, hID*100+i)
				}
			}
		}

		var forbidden []int
		var avoid []int
		for aid, r := range restrNposso {
			if r[id] {
				forbidden = append(forbidden, aid)
			}
		}
		for aid, r := range restrPrefiroN {
			if r[id] {
				avoid = append(avoid, aid)
			}
		}

		solverCandidates = append(solverCandidates, types.SolverCandidate{
			ID:                id,
			Name:              nome,
			PreferredGroupIDs: preferredGroupIDs,
			Attributes: map[string]string{
				"curso":    curso,
				"semestre": strconv.Itoa(semestre),
			},
			EvaluatorRestrictions: types.SolverCandidateRestrictions{
				ForbiddenEvaluatorIDs: forbidden,
				AvoidEvaluatorIDs:     avoid,
			},
		})
	}

	problem := types.AllocationProblem{
		Candidates: solverCandidates,
		Groups:     solverGroups,
		HardRestrictions: types.SolverHardRestrictions{
			AllCandidatesMustBeAssigned:         false,
			RespectCandidatePreferences:         true,
			EnforceGroupCapacity:                true,
			EnforceForbiddenEvaluators:          true,
			EnforceMinCandidatesOnCompleteState: false,
		},
		SoftRules: types.SolverSoftRules{
			PreferencePenaltyByRank: []int{0, 1, 3, 6, 10},
			AvoidEvaluatorPenalty:   10,
			Criteria:                params.SoftCriteria,
		},
	}

	// 6. Run the new exact solver
	solverOpts := allocation.NormalizeSolverOptions(allocation.SolverOptions{
		WorkerCount:   4,
		ParallelDepth: 2,
	})
	res := allocation.SolveAllocation(problem, solverOpts)

	var uiMesas []UIMesa
	alocadosIDs := make(map[int]bool)

	// Attach candidates to groups based on assignments
	for candID, groupID := range res.Assignments {
		m := uiMesasMap[groupID]
		m.Candidatos = append(m.Candidatos, candMap[candID])
		uiMesasMap[groupID] = m
		alocadosIDs[candID] = true
	}

	// Only include groups that have at least one candidate
	for _, m := range uiMesasMap {
		if len(m.Candidatos) > 0 {
			uiMesas = append(uiMesas, m)
		}
	}

	var naoAlocados []UICandidate
	for cid, c := range candMap {
		if !alocadosIDs[cid] {
			naoAlocados = append(naoAlocados, c)
		}
	}

	status := "Sucesso!"
	if len(naoAlocados) > 0 {
		status = "Alocação Parcial"
	}
	if res.Status == "INFEASIBLE" {
		status = "Impossível (Infeasible)"
	}

	return UIAllocationResult{
		Status:      status,
		Mesas:       uiMesas,
		NaoAlocados: naoAlocados,
	}, nil
}

func writeWailsSmokeSentinel() {
	path := os.Getenv("CANDIDATE_ALLOCATOR_WAILS_SMOKE_FILE")
	if path == "" {
		return
	}

	if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
		fmt.Println("Erro ao escrever sentinel do smoke test do Wails:", err)
	}
}
