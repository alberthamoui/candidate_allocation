package main

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

type progressEvent struct {
	Step      string `json:"step"`
	Pct       int    `json:"pct"`
	Tentativa int    `json:"tentativa"`
	Total     int    `json:"total"`
	Score     int    `json:"score"`
}

func (a *App) emitProgress(step string, pct, tentativa, total, score int) {
	wailsrt.EventsEmit(a.ctx, "alocacao:progress", progressEvent{
		Step:      step,
		Pct:       pct,
		Tentativa: tentativa,
		Total:     total,
		Score:     score,
	})
}

// RunAlocacao executa o algoritmo de alocação e retorna o resultado serializável.
func (a *App) RunAlocacao() (AlocacaoResponse, error) {
	a.emitProgress("Conectando ao banco...", 5, 0, 0, 0)
	conn, err := openDB()
	if err != nil {
		return AlocacaoResponse{}, fmt.Errorf("erro ao abrir banco: %w", err)
	}
	defer conn.Close()

	a.emitProgress("Carregando dados...", 15, 0, 0, 0)
	avals := carregarAvaliadores(conn)
	hard, soft := carregarRestricoes(conn)
	horarios := carregarHorarios(conn)
	prefs := carregarDisponibilidades(conn, horarios)

	a.emitProgress("Iniciando algoritmo...", 25, 0, NUM_TENTATIVAS, 0)

	// Emite progresso a cada PRINT_QUANTIDADE iterações para não sobrecarregar o IPC
	onProgress := func(tentativa, total, score int) {
		if tentativa%PRINT_QUANTIDADE != 0 {
			return
		}
		var pct int
		if total > 0 {
			pct = 25 + (tentativa*70)/total
		} else {
			// modo "nota": progresso baseado na proximidade da nota mínima
			if score < 0 {
				pct = 25
			} else {
				pct = 25 + (score*70)/NOTA_MINIMA
			}
		}
		if pct > 95 {
			pct = 95
		}
		a.emitProgress("Calculando...", pct, tentativa, total, score)
	}

	res, mesas := fazerMelhorAlocacaoMesas(horarios, avals, prefs, hard, soft, onProgress)

	a.emitProgress("Finalizando...", 97, 0, 0, 0)

	mapMesa := make(map[int]*Mesa, len(mesas))
	for _, m := range mesas {
		mapMesa[m.ID] = m
	}
	total := imprimirAlocacaoMesas(res.Alocacao, mapMesa, prefs)
	imprimirMesasPreenchidas(mesas, res.Alocacao, total)

	avalNames := make(map[int]string, len(avals))
	for _, av := range avals {
		avalNames[av.ID] = av.Nome
	}

	type pessoaRow struct {
		ID          int
		Nome        string
		EmailInsper string
		Curso       string
		Semestre    int
	}
	var todasPessoas []pessoaRow
	pessoaNames := make(map[int]string)

	pRows, err := conn.Query(`SELECT id, nome, email_insper, curso, semestre FROM pessoa`)
	if err != nil {
		return AlocacaoResponse{}, fmt.Errorf("erro ao carregar candidatos: %w", err)
	}
	for pRows.Next() {
		var p pessoaRow
		if scanErr := pRows.Scan(&p.ID, &p.Nome, &p.EmailInsper, &p.Curso, &p.Semestre); scanErr == nil {
			pessoaNames[p.ID] = p.Nome
			todasPessoas = append(todasPessoas, p)
		}
	}
	pRows.Close()

	var mesaResults []MesaResult
	for _, m := range mesas {
		if len(m.Candidatos) == 0 {
			continue
		}
		mr := MesaResult{ID: m.ID, Descricao: m.Descricao}
		for _, pid := range m.Candidatos {
			name := pessoaNames[pid]
			if name == "" {
				name = fmt.Sprintf("ID %d", pid)
			}
			mr.Candidatos = append(mr.Candidatos, name)
		}
		for _, aid := range m.Avaliadores {
			name := avalNames[aid]
			if name == "" {
				name = fmt.Sprintf("ID %d", aid)
			}
			mr.Avaliadores = append(mr.Avaliadores, name)
		}
		mesaResults = append(mesaResults, mr)
	}
	sort.Slice(mesaResults, func(i, j int) bool {
		return mesaResults[i].ID < mesaResults[j].ID
	})

	alocadosSet := make(map[int]bool, len(res.Alocacao))
	for pid := range res.Alocacao {
		alocadosSet[pid] = true
	}
	var naoAlocados []PessoaInfo
	for _, p := range todasPessoas {
		if !alocadosSet[p.ID] {
			naoAlocados = append(naoAlocados, PessoaInfo{
				ID:          p.ID,
				Nome:        p.Nome,
				EmailInsper: p.EmailInsper,
				Curso:       p.Curso,
				Semestre:    p.Semestre,
			})
		}
	}

	result := AlocacaoResponse{
		Mesas:           mesaResults,
		TotalAlocados:   res.Alocados,
		NaoAlocadosInfo: naoAlocados,
		Pontuacao:       res.Pontuacao,
	}
	a.lastResult = &result
	return result, nil
}

// ExportResultado abre um diálogo de salvar e grava a última alocação em xlsx.
func (a *App) ExportResultado() error {
	if a.lastResult == nil {
		return fmt.Errorf("nenhuma alocação disponível para exportar")
	}

	path, err := wailsrt.SaveFileDialog(a.ctx, wailsrt.SaveDialogOptions{
		Title:           "Salvar Resultado da Alocação",
		DefaultFilename: "alocacao.xlsx",
		Filters: []wailsrt.FileFilter{
			{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"},
		},
	})
	if err != nil {
		return fmt.Errorf("erro ao abrir diálogo: %w", err)
	}
	if path == "" {
		return nil // usuário cancelou
	}

	f := excelize.NewFile()
	defer f.Close()

	// ── Aba 1: Alocação ──────────────────────────────────────────────────
	sheet1 := "Alocação"
	f.SetSheetName("Sheet1", sheet1)
	for i, h := range []string{"Mesa", "Candidato", "Avaliadores"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet1, cell, h)
	}
	row := 2
	for _, mesa := range a.lastResult.Mesas {
		avStr := strings.Join(mesa.Avaliadores, ", ")
		for _, cand := range mesa.Candidatos {
			f.SetCellValue(sheet1, fmt.Sprintf("A%d", row), mesa.Descricao)
			f.SetCellValue(sheet1, fmt.Sprintf("B%d", row), cand)
			f.SetCellValue(sheet1, fmt.Sprintf("C%d", row), avStr)
			row++
		}
	}

	// ── Aba 2: Não Alocados ───────────────────────────────────────────────
	sheet2 := "Não Alocados"
	f.NewSheet(sheet2)
	for i, h := range []string{"Nome", "Email Institucional", "Curso", "Semestre"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet2, cell, h)
	}
	for i, p := range a.lastResult.NaoAlocadosInfo {
		r := i + 2
		f.SetCellValue(sheet2, fmt.Sprintf("A%d", r), p.Nome)
		f.SetCellValue(sheet2, fmt.Sprintf("B%d", r), p.EmailInsper)
		f.SetCellValue(sheet2, fmt.Sprintf("C%d", r), p.Curso)
		f.SetCellValue(sheet2, fmt.Sprintf("D%d", r), p.Semestre)
	}

	return f.SaveAs(path)
}

// openDB abre a conexão com o banco de dados SQLite.
func openDB() (*sql.DB, error) {
	return sql.Open("sqlite3", "./insper.db")
}
