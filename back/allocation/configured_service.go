package allocation

import (
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/diagnostics"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
)

// ConfiguredAllocationResult contains the normalized solver execution plus the
// domain records used to build the problem.
type ConfiguredAllocationResult struct {
	Config      types.AllocationConfiguration `json:"config"`
	Problem     types.AllocationProblem       `json:"problem"`
	Result      types.SolverResult            `json:"result"`
	Candidatos  []types.Candidato             `json:"candidatos"`
	Avaliadores []types.Avaliador             `json:"avaliadores"`
}

type ConfiguredAllocationCallbacks struct {
	Progress ProgressCallback
	Solution func(ConfiguredAllocationResult)
}

// RunConfiguredAllocation executes the official configured allocation path and
// optionally streams progress snapshots from the exact solver.
func RunConfiguredAllocation(db *sql.DB, config types.AllocationConfiguration, progress ProgressCallback) (ConfiguredAllocationResult, error) {
	return RunConfiguredAllocationStreaming(context.Background(), db, config, ConfiguredAllocationCallbacks{Progress: progress})
}

// RunConfiguredAllocationStreaming publishes improving feasible solutions and
// honors cancellation while preserving the best incumbent found so far.
func RunConfiguredAllocationStreaming(
	ctx context.Context,
	db *sql.DB,
	config types.AllocationConfiguration,
	callbacks ConfiguredAllocationCallbacks,
) (ConfiguredAllocationResult, error) {
	defer diagnostics.Begin("allocation.total", 0)()
	if db == nil {
		return ConfiguredAllocationResult{}, errors.New("conexao do banco nao pode ser nil")
	}
	if config.Diagnostics.HasErrors {
		return ConfiguredAllocationResult{}, errors.New("configuracao de alocacao possui erros de validacao")
	}

	data, err := LoadConfiguredAllocationData(db)
	if err != nil {
		return ConfiguredAllocationResult{}, err
	}

	problem, err := BuildAllocationProblem(config, data.Candidatos, data.Avaliadores, data.Restricoes)
	if err != nil {
		return ConfiguredAllocationResult{}, err
	}

	options := NormalizeSolverOptions(SolverOptions{
		WorkerCount:   4,
		ParallelDepth: 2,
		Progress:      callbacks.Progress,
		Context:       ctx,
	})
	if callbacks.Solution != nil {
		options.Incumbent = func(solution types.SolverResult) {
			snapshotConfig := config
			snapshotConfig.Result = types.AllocationExecutionResult{Status: solution.Status, Notes: append([]string(nil), solution.DebugNotes...)}
			callbacks.Solution(ConfiguredAllocationResult{
				Config: snapshotConfig, Problem: problem, Result: solution,
				Candidatos: data.Candidatos, Avaliadores: data.Avaliadores,
			})
		}
	}
	endSolve := diagnostics.Begin("allocation.solve", len(problem.Candidates))
	result := SolveAllocation(problem, options)
	endSolve()

	config.Result = types.AllocationExecutionResult{
		Status: result.Status,
		Notes:  append([]string(nil), result.DebugNotes...),
	}

	return ConfiguredAllocationResult{
		Config:      config,
		Problem:     problem,
		Result:      result,
		Candidatos:  data.Candidatos,
		Avaliadores: data.Avaliadores,
	}, nil
}

// ConfiguredAllocationData is the persisted domain input required by the
// AllocationConfiguration -> AllocationProblem adapter.
type ConfiguredAllocationData struct {
	Candidatos  []types.Candidato
	Avaliadores []types.Avaliador
	Restricoes  []types.Restricao
}

// LoadConfiguredAllocationData loads the domain data persisted by the import
// workflow in a shape that can be consumed by BuildAllocationProblem.
func LoadConfiguredAllocationData(db *sql.DB) (ConfiguredAllocationData, error) {
	defer diagnostics.Begin("allocation.load", 0)()
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ConfiguredAllocationData{}, err
	}
	defer func() { _ = tx.Rollback() }()
	candidatos, err := loadConfiguredCandidates(tx)
	if err != nil {
		return ConfiguredAllocationData{}, err
	}
	avaliadores, err := loadConfiguredEvaluators(tx)
	if err != nil {
		return ConfiguredAllocationData{}, err
	}
	restricoes, err := loadConfiguredRestrictions(tx)
	if err != nil {
		return ConfiguredAllocationData{}, err
	}

	if err := tx.Commit(); err != nil {
		return ConfiguredAllocationData{}, err
	}
	return ConfiguredAllocationData{
		Candidatos:  candidatos,
		Avaliadores: avaliadores,
		Restricoes:  restricoes,
	}, nil
}

// BuildConfigurationFromPersistedData creates the Wails execution config from
// the data already saved in the database.
func BuildConfigurationFromPersistedData(db *sql.DB, params types.AllocationParams) (types.AllocationConfiguration, error) {
	data, err := LoadConfiguredAllocationData(db)
	if err != nil {
		return types.AllocationConfiguration{}, err
	}

	detections := logic.DetectUniquePreferenceValues(data.Candidatos)
	if params.SemPreferencia {
		candidates := append([]types.Candidato(nil), data.Candidatos...)
		for i := range candidates {
			candidates[i].Opcoes = append([]string(nil), params.HorariosLivres...)
		}
		detections = logic.DetectUniquePreferenceValues(candidates)
	}
	mappings := params.ScheduleMappings
	if mappings == nil {
		mappings = logic.SuggestScheduleMappings(detections)
	}
	if err := logic.ValidateScheduleCoverage(detections, mappings); err != nil {
		return types.AllocationConfiguration{}, err
	}
	for i, mapping := range mappings {
		day, hour, err := logic.CanonicalSchedule(mapping.Dia, mapping.Hora)
		if err != nil {
			return types.AllocationConfiguration{}, fmt.Errorf("preferência %q: %w", mapping.ValorPreferencia, err)
		}
		// Do not mutate the caller's editable mappings.
		if i == 0 {
			mappings = append([]types.PreferenceScheduleMapping(nil), mappings...)
		}
		mappings[i].Dia, mappings[i].Hora = day, hour
	}

	return logic.BuildAllocationConfiguration(detections, mappings, params, data.Candidatos)
}

func loadConfiguredCandidates(db dbpkg.Connection) ([]types.Candidato, error) {
	rows, err := db.Query(`SELECT id, timestamp, nome, cpf, numero, CAST(semestre AS TEXT), curso, email_secundario, email_pessoal, extras FROM pessoa ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar candidatos configurados: %w", err)
	}
	defer func() { _ = rows.Close() }()

	candidatos := make([]types.Candidato, 0)
	candidateIDs := make([]int, 0)
	for rows.Next() {
		var (
			id     int
			extras sql.NullString
			item   types.Candidato
		)
		if err := rows.Scan(
			&id,
			&item.Timestamp,
			&item.Nome,
			&item.CPF,
			&item.Numero,
			&item.Semestre,
			&item.Curso,
			&item.EmailSecundario,
			&item.EmailPessoal,
			&extras,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler candidato configurado: %w", err)
		}
		if extras.Valid && strings.TrimSpace(extras.String) != "" {
			if err := json.Unmarshal([]byte(extras.String), &item.Extras); err != nil {
				return nil, fmt.Errorf("erro ao decodificar extras do candidato %q: %w", item.Nome, err)
			}
		}
		candidatos = append(candidatos, item)
		candidateIDs = append(candidateIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar candidatos configurados: %w", err)
	}

	preferences, err := loadCandidatePreferences(db)
	if err != nil {
		return nil, err
	}
	for idx, id := range candidateIDs {
		candidatos[idx].Opcoes = append([]string(nil), preferences[id]...)
	}

	return candidatos, nil
}

func loadCandidatePreferences(db dbpkg.Connection) (map[int][]string, error) {
	rows, err := db.Query(`
		SELECT d.pessoa_id, h.opcao
		FROM disponibilidade d
		JOIN opcoes_horario h ON h.id = d.horario_id
		ORDER BY d.pessoa_id, d.preferencia ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar preferencias dos candidatos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int][]string)
	for rows.Next() {
		var (
			candidateID int
			option      string
		)
		if err := rows.Scan(&candidateID, &option); err != nil {
			return nil, fmt.Errorf("erro ao ler preferencia de candidato: %w", err)
		}
		result[candidateID] = append(result[candidateID], option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar preferencias dos candidatos: %w", err)
	}

	return result, nil
}

func loadConfiguredEvaluators(db dbpkg.Connection) ([]types.Avaliador, error) {
	rows, err := db.Query(`SELECT id, nome, email, sigla, extras FROM avaliador ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar avaliadores configurados: %w", err)
	}
	defer func() { _ = rows.Close() }()

	avaliadores := make([]types.Avaliador, 0)
	for rows.Next() {
		var (
			extras sql.NullString
			item   types.Avaliador
		)
		if err := rows.Scan(&item.ID, &item.Nome, &item.Email, &item.Sigla, &extras); err != nil {
			return nil, fmt.Errorf("erro ao ler avaliador configurado: %w", err)
		}
		if extras.Valid && strings.TrimSpace(extras.String) != "" {
			if err := json.Unmarshal([]byte(extras.String), &item.Extras); err != nil {
				return nil, fmt.Errorf("erro ao decodificar extras do avaliador %q: %w", item.Nome, err)
			}
		}
		avaliadores = append(avaliadores, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar avaliadores configurados: %w", err)
	}

	return avaliadores, nil
}

func loadConfiguredRestrictions(db dbpkg.Connection) ([]types.Restricao, error) {
	restrictionsByCandidate := make(map[string]*types.Restricao)

	if err := appendRestrictionRows(db, restrictionsByCandidate, "restricoesNposso"); err != nil {
		return nil, err
	}
	if err := appendRestrictionRows(db, restrictionsByCandidate, "restricoesPrefiroN"); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(restrictionsByCandidate))
	for name := range restrictionsByCandidate {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]types.Restricao, 0, len(names))
	for _, name := range names {
		result = append(result, *restrictionsByCandidate[name])
	}
	return result, nil
}

func appendRestrictionRows(db dbpkg.Connection, byCandidate map[string]*types.Restricao, table string) error {
	targetField := "NaoPosso"
	if table == "restricoesPrefiroN" {
		targetField = "PrefiroNao"
	}

	// #nosec G201 - table is selected by the caller from a fixed internal set.
	rows, err := db.Query(fmt.Sprintf(`
		SELECT p.nome, a.sigla
		FROM %s r
		JOIN pessoa p ON p.id = r.candidato_id
		JOIN avaliador a ON a.id = r.avaliador_id
		ORDER BY p.nome, a.sigla
	`, table))
	if err != nil {
		return fmt.Errorf("erro ao carregar %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var candidateName, evaluatorSigla string
		if err := rows.Scan(&candidateName, &evaluatorSigla); err != nil {
			return fmt.Errorf("erro ao ler %s: %w", table, err)
		}
		restriction := byCandidate[candidateName]
		if restriction == nil {
			restriction = &types.Restricao{Candidato: candidateName}
			byCandidate[candidateName] = restriction
		}
		if targetField == "NaoPosso" {
			restriction.NaoPosso = appendCSVValue(restriction.NaoPosso, evaluatorSigla)
		} else {
			restriction.PrefiroNao = appendCSVValue(restriction.PrefiroNao, evaluatorSigla)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("erro ao iterar %s: %w", table, err)
	}

	return nil
}

func appendCSVValue(current, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return current
	}
	if strings.TrimSpace(current) == "" {
		return value
	}
	return current + ", " + value
}
