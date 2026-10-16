package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
)

// ==================================================
// =========== CARREGAMENTO DE DADOS DB ============
// ==================================================

// dadosAlocacao é tudo o que a alocação lê do banco da sessão.
type dadosAlocacao struct {
	horarios   map[int]*Horario
	prefs      map[int][]int // pessoa_id → horários em ordem de preferência
	avals      []*Avaliador
	hard, soft map[int]map[int]bool      // avaliador_id → candidato_id → restrição
	atributos  map[int]map[string]string // pessoa_id → coluna (curso, semestre) → valor
}

// carregarDados lê do banco da sessão os dados da alocação. Um erro aqui
// falha só a requisição desta sessão, nunca o servidor.
func (s *Session) carregarDados() (dadosAlocacao, error) {
	var d dadosAlocacao
	var err error
	if d.horarios, err = carregarHorarios(s.db); err != nil {
		return d, fmt.Errorf("carregando horários: %w", err)
	}
	if d.prefs, err = carregarDisponibilidades(s.db, d.horarios); err != nil {
		return d, fmt.Errorf("carregando disponibilidades: %w", err)
	}
	if d.avals, err = carregarAvaliadores(s.db); err != nil {
		return d, fmt.Errorf("carregando avaliadores: %w", err)
	}
	if d.hard, d.soft, err = carregarRestricoes(s.db); err != nil {
		return d, fmt.Errorf("carregando restrições: %w", err)
	}
	if d.atributos, err = carregarAtributos(s.db); err != nil {
		return d, fmt.Errorf("carregando curso e semestre: %w", err)
	}
	return d, nil
}

// carregarAtributos lê as colunas que os critérios adicionais podem usar.
func carregarAtributos(db *sql.DB) (map[int]map[string]string, error) {
	rows, err := db.Query(`SELECT id, curso, semestre FROM pessoa`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	atributos := make(map[int]map[string]string)
	for rows.Next() {
		var id, semestre int
		var curso string
		if err := rows.Scan(&id, &curso, &semestre); err != nil {
			return nil, err
		}
		sem := ""
		if semestre > 0 {
			sem = strconv.Itoa(semestre)
		}
		atributos[id] = map[string]string{COLUNA_CURSO: curso, COLUNA_SEMESTRE: sem}
	}
	return atributos, rows.Err()
}

func carregarHorarios(db *sql.DB) (map[int]*Horario, error) {
	horarios := make(map[int]*Horario)
	rows, err := db.Query(`SELECT id, opcao FROM opcoes_horario`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var h Horario
		if err := rows.Scan(&h.ID, &h.Descricao); err != nil {
			return nil, err
		}
		h.Candidatos = []int{}
		horarios[h.ID] = &h
	}
	return horarios, rows.Err()
}

func carregarDisponibilidades(db *sql.DB, horarios map[int]*Horario) (map[int][]int, error) {
	prefs := make(map[int][]int)
	rows, err := db.Query(`SELECT pessoa_id, horario_id, preferencia FROM disponibilidade ORDER BY pessoa_id, preferencia ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pid, hid, pref int
		if err := rows.Scan(&pid, &hid, &pref); err != nil {
			return nil, err
		}

		h, ok := horarios[hid]
		if !ok {
			log.Printf("[WARN] horario_id %d não encontrado na tabela de horários. Ignorando.", hid)
			continue
		}

		h.Candidatos = append(h.Candidatos, pid)
		prefs[pid] = append(prefs[pid], hid)
	}
	return prefs, rows.Err()
}

func carregarAvaliadores(db *sql.DB) ([]*Avaliador, error) {
	rows, err := db.Query(`SELECT id, nome, email, sigla FROM avaliador`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var avals []*Avaliador
	for rows.Next() {
		var a Avaliador
		if err := rows.Scan(&a.ID, &a.Nome, &a.Email, &a.Sigla); err != nil {
			return nil, err
		}
		avals = append(avals, &a)
	}
	return avals, rows.Err()
}

func carregarRestricoes(db *sql.DB) (hard, soft map[int]map[int]bool, err error) {
	loadInto := func(query string) (map[int]map[int]bool, error) {
		m := make(map[int]map[int]bool)
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var aid, cid int
			if err := rows.Scan(&aid, &cid); err != nil {
				return nil, err
			}
			if m[aid] == nil {
				m[aid] = make(map[int]bool)
			}
			m[aid][cid] = true
		}
		return m, rows.Err()
	}

	if hard, err = loadInto(`SELECT avaliador_id, candidato_id FROM restricoesNposso`); err != nil {
		return nil, nil, err
	}
	if soft, err = loadInto(`SELECT avaliador_id, candidato_id FROM restricoesPrefiroN`); err != nil {
		return nil, nil, err
	}
	return hard, soft, nil
}
