package main

import (
	"context"
	"fmt"
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

// RunAlocacao executa o algoritmo na sessão com os parâmetros dados (já
// validados) e retorna o resultado. emit é chamado com eventos progressEvent.
func (s *Session) RunAlocacao(ctx context.Context, param ParametrosAlocacao, emit func(any)) (AlocacaoResponse, error) {
	emit(progressEvent{Step: "Carregando dados...", Pct: 15})

	d, err := s.carregarDados()
	if err != nil {
		return AlocacaoResponse{}, err
	}
	prefs := d.prefs

	emit(progressEvent{Step: "Iniciando algoritmo...", Pct: 25, Total: recursos.Execucoes * recursos.Iteracoes})

	onProgress := func(feitas, total, score int) {
		pct := 25 + (feitas*70)/total
		if pct > 95 {
			pct = 95
		}
		emit(progressEvent{Step: "Otimizando alocação...", Pct: pct, Tentativa: feitas, Total: total, Score: score})
	}

	res, mesas := fazerMelhorAlocacaoMesas(ctx, param, d.horarios, d.avals, prefs, d.hard, d.soft, d.atributos, onProgress)
	emit(progressEvent{Step: "Finalizando...", Pct: 97})

	mapMesa := make(map[int]*Mesa, len(mesas))
	for _, m := range mesas {
		mapMesa[m.ID] = m
	}
	total := imprimirAlocacaoMesas(res.Alocacao, mapMesa, prefs)
	imprimirMesasPreenchidas(mesas, res.Alocacao, total)

	cands, err := carregarCandidatos(s)
	if err != nil {
		return AlocacaoResponse{}, fmt.Errorf("carregando candidatos: %w", err)
	}
	result := montarResultado(d, cands, res, mesas, param.Criterios)
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

	// celula grava um valor na coluna col e linha lin (a partir de 1); o
	// primeiro erro fica em errCelula e é devolvido no fim.
	var errCelula error
	celula := func(aba string, col, lin int, val any) {
		cell, err := excelize.CoordinatesToCellName(col, lin)
		if err == nil {
			err = f.SetCellValue(aba, cell, val)
		}
		if errCelula == nil && err != nil {
			errCelula = fmt.Errorf("aba %q, célula (%d, %d): %w", aba, col, lin, err)
		}
	}

	// --- Aba "Lista": uma linha por candidato (mesmo formato anterior) ---
	lista := "Lista"
	if err := f.SetSheetName("Sheet1", lista); err != nil {
		return nil, err
	}
	for i, h := range []string{"Mesa", "Candidato", "Avaliadores"} {
		celula(lista, i+1, 1, h)
	}
	row := 2
	for _, mesa := range s.lastResult.Mesas {
		var nomesAv []string
		for _, av := range mesa.Avaliadores {
			nomesAv = append(nomesAv, av.Nome)
		}
		avStr := strings.Join(nomesAv, ", ")
		for _, cand := range mesa.Candidatos {
			celula(lista, 1, row, mesa.Descricao)
			celula(lista, 2, row, cand.Nome)
			celula(lista, 3, row, avStr)
			row++
		}
	}

	// --- Aba "Alocação": grade 2D agrupada por dia ---
	aloc := "Alocação"
	if _, err := f.NewSheet(aloc); err != nil {
		return nil, err
	}

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
	set := func(col, r int, val any) { celula(aloc, col, r, val) }

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
	if _, err := f.NewSheet(nao); err != nil {
		return nil, err
	}
	for i, h := range []string{"Nome", "Email Institucional", "Curso", "Semestre"} {
		celula(nao, i+1, 1, h)
	}
	for i, p := range s.lastResult.NaoAlocadosInfo {
		r := i + 2
		celula(nao, 1, r, p.Nome)
		celula(nao, 2, r, p.EmailInsper)
		celula(nao, 3, r, p.Curso)
		celula(nao, 4, r, p.Semestre)
	}
	if errCelula != nil {
		return nil, errCelula
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
