package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// --- removido var Conn *sql.DB e Configure ---

// AddHorario insere um novo registro em opcoes_horario
func AddHorario(db Connection, opcao string) (int64, error) {
	opcao = strings.TrimSpace(strings.ToLower(opcao))
	var existing int64
	err := db.QueryRow(`SELECT id FROM opcoes_horario WHERE opcao = ? ORDER BY id LIMIT 1`, opcao).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := db.Exec(`INSERT INTO opcoes_horario (opcao) VALUES (?)`, opcao)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AddDisponibilidade insere um vínculo em disponibilidade
func AddDisponibilidade(db Connection, pessoaID, horarioID, preferencia int64) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO disponibilidade (pessoa_id, horario_id, preferencia)
        VALUES (?, ?, ?)
    `, pessoaID, horarioID, preferencia)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func AddRestricaoNposso(db Connection, avaliadorID, candidatoID int64) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO restricoesNposso (avaliador_id, candidato_id)
        VALUES (?, ?)
    `, avaliadorID, candidatoID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func AddRestricaoPrefiroN(db Connection, avaliadorID, candidatoID int64) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO restricoesPrefiroN (avaliador_id, candidato_id)
        VALUES (?, ?)
    `, avaliadorID, candidatoID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func GetAvaliadorIDBySigla(db Connection, sigla string) (int64, error) {
	var id int64
	err := db.QueryRow(
		`SELECT id FROM avaliador WHERE sigla = ?`,
		sigla,
	).Scan(&id)
	return id, err
}

func GetPessoaIDByName(db Connection, nome string) (int64, error) {
	var id, count int64
	err := db.QueryRow(`SELECT COALESCE(MIN(id), 0), COUNT(*) FROM pessoa WHERE nome = ?`, strings.TrimSpace(nome)).Scan(&id, &count)
	if err != nil {
		return 0, err
	}
	if count != 1 {
		return 0, fmt.Errorf("candidato %q possui %d correspondências; corrija o nome na restrição", nome, count)
	}
	return id, nil
}
