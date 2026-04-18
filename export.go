package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

// progressEvent é enviado via SSE durante a execução do algoritmo.
type progressEvent struct {
	Step      string `json:"step"`
	Pct       int    `json:"pct"`
	Tentativa int    `json:"tentativa"`
	Total     int    `json:"total"`
	Score     int    `json:"score"`
}

// RunAlocacao executa o algoritmo na sessão e retorna o resultado.
// emit é chamado com eventos progressEvent e, ao final, com o resultado completo.
func (s *Session) RunAlocacao(emit func(any)) (AlocacaoResponse, error) {
	emit(progressEvent{Step: "Carregando dados...", Pct: 15})

	avals := carregarAvaliadores(s.db)
	hard, soft := carregarRestricoes(s.db)
	horarios := carregarHorarios(s.db)
	prefs := carregarDisponibilidades(s.db, horarios)

	emit(progressEvent{Step: "Iniciando algoritmo...", Pct: 25, Total: NUM_TENTATIVAS * MAX_RESTARTS})

	onProgress := func(tentativa, total, score int) {
		if tentativa%PRINT_QUANTIDADE != 0 {
			return
		}
		var pct int
		if total > 0 {
			pct = 25 + (tentativa*70)/total
		} else {
			if score < 0 {
				pct = 25
			} else {
				pct = 25 + (score*70)/NOTA_MINIMA
			}
		}
		if pct > 95 {
			pct = 95
		}
		emit(progressEvent{Step: "Calculando...", Pct: pct, Tentativa: tentativa, Total: total, Score: score})
	}

	res, mesas := fazerMelhorAlocacaoMesas(horarios, avals, prefs, hard, soft, onProgress)
	emit(progressEvent{Step: "Finalizando...", Pct: 97})

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

	pRows, err := s.db.Query(`SELECT id, nome, email_insper, curso, semestre FROM pessoa`)
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
	s.lastResult = &result
	return result, nil
}

// ExportResultado gera os bytes do arquivo .xlsx com a última alocação.
func (s *Session) ExportResultado() ([]byte, error) {
	if s.lastResult == nil {
		return nil, fmt.Errorf("nenhuma alocação disponível para exportar")
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet1 := "Alocação"
	f.SetSheetName("Sheet1", sheet1)
	for i, h := range []string{"Mesa", "Candidato", "Avaliadores"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet1, cell, h)
	}
	row := 2
	for _, mesa := range s.lastResult.Mesas {
		avStr := strings.Join(mesa.Avaliadores, ", ")
		for _, cand := range mesa.Candidatos {
			f.SetCellValue(sheet1, fmt.Sprintf("A%d", row), mesa.Descricao)
			f.SetCellValue(sheet1, fmt.Sprintf("B%d", row), cand)
			f.SetCellValue(sheet1, fmt.Sprintf("C%d", row), avStr)
			row++
		}
	}

	sheet2 := "Não Alocados"
	f.NewSheet(sheet2)
	for i, h := range []string{"Nome", "Email Institucional", "Curso", "Semestre"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet2, cell, h)
	}
	for i, p := range s.lastResult.NaoAlocadosInfo {
		r := i + 2
		f.SetCellValue(sheet2, fmt.Sprintf("A%d", r), p.Nome)
		f.SetCellValue(sheet2, fmt.Sprintf("B%d", r), p.EmailInsper)
		f.SetCellValue(sheet2, fmt.Sprintf("C%d", r), p.Curso)
		f.SetCellValue(sheet2, fmt.Sprintf("D%d", r), p.Semestre)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
