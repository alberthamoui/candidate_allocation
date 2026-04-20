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

	onProgress := func(globalTentativa, total, score, restart int) {
		if globalTentativa%PRINT_QUANTIDADE != 0 {
			return
		}
		pct := 25 + (globalTentativa*70)/total
		if pct > 95 {
			pct = 95
		}
		step := fmt.Sprintf("Calculando... (rodada %d/%d)", restart, MAX_RESTARTS)
		emit(progressEvent{Step: step, Pct: pct, Tentativa: globalTentativa, Total: total, Score: score})
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

	diaNames := make(map[int]string, len(horarios))
	for _, h := range horarios {
		diaNames[h.ID] = h.Descricao
	}

	var mesaResults []MesaResult
	for _, m := range mesas {
		if len(m.Candidatos) == 0 {
			continue
		}
		mr := MesaResult{ID: m.ID, DiaID: m.DiaID, DiaNome: diaNames[m.DiaID], Descricao: m.Descricao}
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

	// --- Aba "Lista": uma linha por candidato (mesmo formato anterior) ---
	lista := "Lista"
	f.SetSheetName("Sheet1", lista)
	for i, h := range []string{"Mesa", "Candidato", "Avaliadores"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(lista, cell, h)
	}
	row := 2
	for _, mesa := range s.lastResult.Mesas {
		avStr := strings.Join(mesa.Avaliadores, ", ")
		for _, cand := range mesa.Candidatos {
			f.SetCellValue(lista, fmt.Sprintf("A%d", row), mesa.Descricao)
			f.SetCellValue(lista, fmt.Sprintf("B%d", row), cand)
			f.SetCellValue(lista, fmt.Sprintf("C%d", row), avStr)
			row++
		}
	}

	// --- Aba "Alocação": grade 2D agrupada por dia ---
	aloc := "Alocação"
	f.NewSheet(aloc)

	// Agrupar mesas por DiaID, preservando ordem de ID
	type diaGroup struct {
		DiaID   int
		DiaNome string
		Mesas   []MesaResult
	}
	diaOrder := []int{}
	diaMap := map[int]*diaGroup{}
	for _, mr := range s.lastResult.Mesas {
		if _, ok := diaMap[mr.DiaID]; !ok {
			diaOrder = append(diaOrder, mr.DiaID)
			diaMap[mr.DiaID] = &diaGroup{DiaID: mr.DiaID, DiaNome: mr.DiaNome}
		}
		diaMap[mr.DiaID].Mesas = append(diaMap[mr.DiaID].Mesas, mr)
	}

	curRow := 1
	set := func(col, r int, val any) {
		cell, _ := excelize.CoordinatesToCellName(col, r)
		f.SetCellValue(aloc, cell, val)
	}

	for _, diaID := range diaOrder {
		grp := diaMap[diaID]

		// Capitaliza primeira letra do nome do dia
		nome := grp.DiaNome
		if len(nome) > 0 {
			nome = strings.ToUpper(nome[:1]) + nome[1:]
		}
		set(1, curRow, nome)
		curRow++

		mesas := grp.Mesas
		for i := 0; i < len(mesas); i += 2 {
			left := mesas[i]
			hasRight := i+1 < len(mesas)

			// Cabeçalho: "Mesa X" / "Mesa Y"
			set(1, curRow, left.Descricao)
			if hasRight {
				set(7, curRow, mesas[i+1].Descricao)
			}
			curRow++

			// Subcabeçalho
			set(1, curRow, "AVALIADORES")
			set(2, curRow, "CANDIDATOS")
			if hasRight {
				set(7, curRow, "AVALIADORES")
				set(8, curRow, "CANDIDATOS")
			}
			curRow++

			// Linhas de dados
			nRows := len(left.Avaliadores)
			if len(left.Candidatos) > nRows {
				nRows = len(left.Candidatos)
			}
			if hasRight {
				right := mesas[i+1]
				if len(right.Avaliadores) > nRows {
					nRows = len(right.Avaliadores)
				}
				if len(right.Candidatos) > nRows {
					nRows = len(right.Candidatos)
				}
			}
			for k := 0; k < nRows; k++ {
				if k < len(left.Avaliadores) {
					set(1, curRow, left.Avaliadores[k])
				}
				if k < len(left.Candidatos) {
					set(2, curRow, left.Candidatos[k])
				}
				if hasRight {
					right := mesas[i+1]
					if k < len(right.Avaliadores) {
						set(7, curRow, right.Avaliadores[k])
					}
					if k < len(right.Candidatos) {
						set(8, curRow, right.Candidatos[k])
					}
				}
				curRow++
			}

			// Linha em branco entre pares de mesas
			curRow++
		}
	}

	// --- Aba "Não Alocados" ---
	nao := "Não Alocados"
	f.NewSheet(nao)
	for i, h := range []string{"Nome", "Email Institucional", "Curso", "Semestre"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(nao, cell, h)
	}
	for i, p := range s.lastResult.NaoAlocadosInfo {
		r := i + 2
		f.SetCellValue(nao, fmt.Sprintf("A%d", r), p.Nome)
		f.SetCellValue(nao, fmt.Sprintf("B%d", r), p.EmailInsper)
		f.SetCellValue(nao, fmt.Sprintf("C%d", r), p.Curso)
		f.SetCellValue(nao, fmt.Sprintf("D%d", r), p.Semestre)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
