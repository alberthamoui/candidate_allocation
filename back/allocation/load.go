package allocation

import (
	types "candidate_alocator/back/type"
	"database/sql"
	"fmt"
	"log"
)

func CarregarHorarios(db *sql.DB) (map[int]*types.Horario, error) {
	horarios := make(map[int]*types.Horario)
	rows, err := db.Query(`SELECT id, opcao FROM opcoes_horario`)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar horarios: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var h types.Horario
		if err := rows.Scan(&h.ID, &h.Descricao); err != nil {
			return nil, fmt.Errorf("erro ao ler horario: %w", err)
		}
		h.Candidatos = []int{}
		horarios[h.ID] = &h
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar horarios: %w", err)
	}

	return horarios, nil
}

func CarregarDisponibilidades(db *sql.DB, horarios map[int]*types.Horario) (map[int][]int, error) {
	prefs := make(map[int][]int)
	rows, err := db.Query(`SELECT pessoa_id, horario_id, preferencia FROM disponibilidade ORDER BY pessoa_id, preferencia ASC`)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar disponibilidades: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var pid, hid, pref int
		if err := rows.Scan(&pid, &hid, &pref); err != nil {
			return nil, fmt.Errorf("erro ao ler disponibilidade: %w", err)
		}

		h, ok := horarios[hid]
		if !ok {
			log.Printf("[WARN] horario_id %d não encontrado na tabela de horários. Ignorando.", hid)
			continue
		}

		h.Candidatos = append(h.Candidatos, pid)
		prefs[pid] = append(prefs[pid], hid)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar disponibilidades: %w", err)
	}

	return prefs, nil
}

func CarregarAvaliadores(db *sql.DB) ([]*types.Avaliador, error) {
	rows, err := db.Query(`SELECT id, nome, email FROM avaliador`)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar avaliadores: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var avals []*types.Avaliador
	for rows.Next() {
		var a types.Avaliador
		if err := rows.Scan(&a.ID, &a.Nome, &a.Email); err != nil {
			return nil, fmt.Errorf("erro ao ler avaliador: %w", err)
		}
		avals = append(avals, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar avaliadores: %w", err)
	}

	return avals, nil
}

func CarregarRestricoes(db *sql.DB) (map[int]map[int]bool, error) {
	restr := make(map[int]map[int]bool)

	rows, err := db.Query(`
        SELECT avaliador_id, candidato_id 
        FROM restricoesNposso
    `)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar restricoes NaoPosso: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var aid, cid int
		if err := rows.Scan(&aid, &cid); err != nil {
			return nil, fmt.Errorf("erro ao ler restricao NaoPosso: %w", err)
		}
		if restr[aid] == nil {
			restr[aid] = make(map[int]bool)
		}
		restr[aid][cid] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar restricoes NaoPosso: %w", err)
	}

	rows2, err := db.Query(`
        SELECT avaliador_id, candidato_id 
        FROM restricoesPrefiroN
    `)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar restricoes PrefiroNao: %w", err)
	}
	defer func() { _ = rows2.Close() }()

	for rows2.Next() {
		var aid, cid int
		if err := rows2.Scan(&aid, &cid); err != nil {
			return nil, fmt.Errorf("erro ao ler restricao PrefiroNao: %w", err)
		}
		if restr[aid] == nil {
			restr[aid] = make(map[int]bool)
		}
		restr[aid][cid] = true
	}

	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar restricoes PrefiroNao: %w", err)
	}

	return restr, nil
}
