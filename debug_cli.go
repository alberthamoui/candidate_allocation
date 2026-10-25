//go:build debug || dev

package main

import (
	"candidate_alocator/back/allocation"
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/diagnostics"
	"candidate_alocator/back/logic"
	"candidate_alocator/back/operations"
	types "candidate_alocator/back/type"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type performanceCase struct {
	TimedOut      bool                 `json:"timedOut"`
	WallMS        float64              `json:"wallMs"`
	Note          string               `json:"note,omitempty"`
	Candidates    int                  `json:"candidates"`
	Scenario      string               `json:"scenario"`
	Run           int                  `json:"run"`
	Status        string               `json:"solverStatus"`
	Assigned      int                  `json:"assigned"`
	Violations    int                  `json:"violations"`
	Nodes         int64                `json:"nodes"`
	WorkbookBytes int                  `json:"workbookBytes"`
	SnapshotBytes int                  `json:"snapshotBytes"`
	Samples       []diagnostics.Sample `json:"samples"`
}

func runDebugCLI(args []string, output io.Writer) error {
	if len(args) == 4 && args[0] == "_performance-case" {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || n > 10000 {
			return fmt.Errorf("volume inválido")
		}
		budget, err := time.ParseDuration(args[3])
		if err != nil || budget < 50*time.Millisecond || budget > time.Minute || (args[2] != "preferences" && args[2] != "criteria") {
			return fmt.Errorf("cenário inválido")
		}
		result, err := measurePerformanceCase(n, args[2], budget)
		if err != nil {
			return err
		}
		return json.NewEncoder(output).Encode(result)
	}
	if len(args) == 0 || args[0] != "performance" {
		return fmt.Errorf("use: debug performance -sizes 100,1000,5000 -runs 3 -solver-budget 2s -scenario preferences|criteria")
	}
	f := flag.NewFlagSet("debug performance", flag.ContinueOnError)
	f.SetOutput(output)
	sizes := f.String("sizes", "100,1000,5000", "quantidades de candidatos sintéticos")
	runs := f.Int("runs", 3, "repetições por volume")
	budget := f.Duration("solver-budget", 2*time.Second, "prazo cooperativo da busca exata; preparo é medido separadamente")
	caseTimeout := f.Duration("case-timeout", 30*time.Second, "limite total por cenário isolado, incluindo preparo")
	scenario := f.String("scenario", "preferences", "preferences ou criteria")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if *caseTimeout < time.Second || *caseTimeout > 10*time.Minute || f.NArg() != 0 || *runs < 1 || *runs > 10 || *budget < 50*time.Millisecond || *budget > time.Minute || (*scenario != "preferences" && *scenario != "criteria") {
		return fmt.Errorf("parâmetros de desempenho inválidos")
	}
	counts := []int{}
	for _, raw := range strings.Split(*sizes, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || n < 1 || n > 10000 {
			return fmt.Errorf("volume inválido: use números entre 1 e 10000")
		}
		counts = append(counts, n)
	}
	report := struct {
		OS          string            `json:"os"`
		Arch        string            `json:"arch"`
		Go          string            `json:"go"`
		CPUs        int               `json:"cpus"`
		Budget      string            `json:"solverBudget"`
		CaseTimeout string            `json:"caseTimeout"`
		Cases       []performanceCase `json:"cases"`
	}{OS: runtime.GOOS, Arch: runtime.GOARCH, Go: runtime.Version(), CPUs: runtime.NumCPU(), Budget: budget.String(), CaseTimeout: caseTimeout.String()}
	for _, n := range counts {
		for i := 1; i <= *runs; i++ {
			result, err := isolatedPerformanceCase(n, *scenario, *budget, *caseTimeout)
			if err != nil {
				return fmt.Errorf("volume %d, repetição %d: %w", n, i, err)
			}
			result.Run = i
			report.Cases = append(report.Cases, result)
		}
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func measurePerformanceCase(n int, scenario string, budget time.Duration) (performanceCase, error) {
	result := performanceCase{Candidates: n, Scenario: scenario}
	dir, err := os.MkdirTemp(os.Getenv("CANDIDATE_PERF_WORK_DIR"), "candidate-performance-")
	if err != nil {
		return result, err
	}
	// #nosec G703 -- dir was created by MkdirTemp; only that new synthetic directory is removed.
	defer func() { _ = os.RemoveAll(dir) }()
	diagnostics.Reset()
	book, err := performanceWorkbook(n)
	if err != nil {
		return result, err
	}
	result.WorkbookBytes = len(book)
	mapping, err := logic.SuggestMapping(book, 2)
	if err != nil {
		return result, err
	}
	parsed, err := logic.BuildUsuariosWithMapping(book, 2, mapping)
	if err != nil {
		return result, err
	}
	data := logic.ImportData{Candidatos: make([]types.Candidato, 0, n), Avaliadores: []types.Avaliador{{Nome: "Avaliador sintético", Email: "evaluator@example.invalid", Sigla: "EV"}}}
	for i := 1; i <= n; i++ {
		row := parsed.Usuarios[i]
		if len(row.Erros) > 0 {
			return result, fmt.Errorf("fixture sintética inválida no registro %d", i)
		}
		data.Candidatos = append(data.Candidatos, row.Usuario)
	}
	db, err := sql.Open("sqlite3", filepath.Join(dir, "import.db")+"?_foreign_keys=on")
	if err != nil {
		return result, err
	}
	defer func() { _ = db.Close() }()
	if err := dbpkg.EnsureAppSchema(db); err != nil {
		return result, err
	}
	if err := logic.ReplaceImport(db, data); err != nil {
		return result, err
	}
	params := types.AllocationParams{GruposPorHorario: (n + 39) / 40, MinPessoasPorGrupo: 1, MaxPessoasPorGrupo: 8, AvaliadoresPorGrupo: 1}
	if scenario == "criteria" {
		params.SoftCriteria = []types.SoftCriterion{{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "curso", SelectedValues: []string{"ADM", "ECO", "ENG"}}}
	}
	config, err := allocation.BuildConfigurationFromPersistedData(db, params)
	if err != nil {
		return result, err
	}
	if parent := os.Getenv("CANDIDATE_PERF_WORK_DIR"); parent != "" {
		path := filepath.Join(parent, "samples.json")
		data, err := json.Marshal(diagnostics.Snapshot())
		if err != nil {
			return result, err
		}
		// #nosec G304 G703 -- fixed sample filename in the private directory created by the parent diagnostic process.
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return result, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	run, err := allocation.RunConfiguredAllocationStreaming(ctx, db, config, allocation.ConfiguredAllocationCallbacks{})
	if err != nil {
		return result, err
	}
	result.Status, result.Assigned, result.Violations = run.Result.Status, len(run.Result.Assignments), len(run.Result.HardViolations)
	result.Nodes = int64(run.Result.Metrics.NodesVisited)
	ui := buildUIAllocationResult(run)
	payload, err := json.Marshal(agendaSnapshot{Run: run, Result: ui})
	if err != nil {
		return result, err
	}
	result.SnapshotBytes = len(payload)
	store, err := operations.New(filepath.Join(dir, "operations.db"), time.Minute, 2)
	if err != nil {
		return result, err
	}
	if err := store.Save("benchmark", payload); err != nil {
		store.Close()
		return result, err
	}
	store.Close()
	store, err = operations.New(filepath.Join(dir, "operations.db"), time.Minute, 2)
	if err != nil {
		return result, err
	}
	defer store.Close()
	if _, err := store.Load("benchmark"); err != nil {
		return result, err
	}
	if _, err := store.Load("benchmark"); err != nil {
		return result, err
	}
	// Notification records are synthetic; no transport or real account is used.
	items := make([]operations.Delivery, n)
	for i := range items {
		items[i] = operations.Delivery{PersonID: i + 1, Channel: "email", Destination: "synthetic@example.invalid", Message: "synthetic"}
	}
	if _, err := store.CreateBatch("benchmark", items); err != nil {
		return result, err
	}
	end := diagnostics.Begin("history.summary", n)
	_, err = store.History("benchmark", 0, 10)
	end()
	if err != nil {
		return result, err
	}
	result.Samples = diagnostics.Snapshot()
	return result, nil
}

func performanceWorkbook(n int) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	header := []any{"Nome", "CPF", "Numero", "Semestre", "Curso", "Email secundario", "Email pessoal", "Opcao 1", "Opcao 2"}
	if err := f.SetSheetRow("Sheet1", "A1", &header); err != nil {
		return nil, err
	}
	slots := []string{"segunda 09-11", "terça 09-11", "quarta 09-11", "quinta 09-11", "sexta 09-11"}
	for i := 0; i < n; i++ {
		row := []any{fmt.Sprintf("Candidato %05d", i), "", fmt.Sprintf("%09d", i+1), "2", []string{"ADM", "ECO", "ENG"}[i%3], fmt.Sprintf("s%d@al.insper.edu.br", i), fmt.Sprintf("s%d@example.invalid", i), slots[i%5], slots[(i+1)%5]}
		if err := f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", i+2), &row); err != nil {
			return nil, err
		}
	}
	b, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Each benchmark runs in its own process so even non-cooperative initial solver
// preparation cannot hang diagnostics or retain its allocations after timeout.
func isolatedPerformanceCase(n int, scenario string, budget, limit time.Duration) (performanceCase, error) {
	result := performanceCase{Candidates: n, Scenario: scenario}
	dir, err := os.MkdirTemp("", "candidate-performance-parent-")
	if err != nil {
		return result, err
	}
	// #nosec G703 -- dir was created by MkdirTemp; only that new synthetic directory is removed.
	defer func() { _ = os.RemoveAll(dir) }()
	samplePath := filepath.Join(dir, "samples.json")
	executable, err := os.Executable()
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	start := time.Now()
	command := exec.CommandContext(ctx, executable, "debug", "_performance-case", strconv.Itoa(n), scenario, budget.String()) // #nosec G204 -- only this application's executable and validated synthetic parameters.
	command.Env = append(os.Environ(), "CANDIDATE_PERF_WORK_DIR="+dir)
	data, err := command.Output()
	wall := float64(time.Since(start)) / float64(time.Millisecond)
	if ctx.Err() != nil {
		if partial, readErr := os.ReadFile(samplePath); readErr == nil { // #nosec G304 -- application-created private diagnostic file.
			_ = json.Unmarshal(partial, &result.Samples)
		}
		result.Status = "timeout"
		result.TimedOut = true
		result.WallMS = wall
		result.Note = "O preparo/busca excedeu o limite total; o processo isolado foi encerrado. Contagens e amostras incompletas não representam um resultado de alocação."
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("cenário isolado falhou: %w", err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	result.WallMS = wall
	return result, nil
}
