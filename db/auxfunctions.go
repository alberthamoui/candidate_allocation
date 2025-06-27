package db

import (
	"database/sql"
	"strings"
)

// --- removido var Conn *sql.DB e Configure ---

// AddHorario insere um novo registro em opcoes_horario
func AddHorario(db *sql.DB, opcao string) (int64, error) {
	opcao = strings.TrimSpace(strings.ToLower(opcao))
	res, err := db.Exec(`
		INSERT OR IGNORE INTO opcoes_horario (opcao) VALUES (?)	`, opcao)
	if err != nil { return 0, err }

	id, _ := res.LastInsertId()
	if id != 0 { return id, nil } // inseriu agora

	var existing int64
	err = db.QueryRow(`SELECT id FROM opcoes_horario WHERE opcao = ?`, opcao).Scan(&existing)
	return existing, err
}

// AddPessoa insere um novo registro em pessoa
func AddPessoa(db *sql.DB, nome, cpf, numero, emailInsper, emailPessoal string, semestre int, curso string) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO pessoa (nome,cpf, numero, email_insper, email_pessoal,  semestre, curso)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, nome, cpf, numero, emailInsper, emailPessoal, semestre, curso)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AddDisponibilidade insere um vínculo em disponibilidade
func AddDisponibilidade(db *sql.DB, pessoaID, horarioID, preferencia int64) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO disponibilidade (pessoa_id, horario_id, preferencia)
        VALUES (?, ?, ?)
    `, pessoaID, horarioID, preferencia)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}


func AddAvaliador(db *sql.DB, nome, email, sigla string) (int64, error) {
	res, err := db.Exec(`
		INSERT OR IGNORE INTO avaliador (nome, email, sigla)
		VALUES (?, ?, ?)
	`, nome, email, sigla)
	if err != nil {
		return 0, err
	}

	id, _ := res.LastInsertId()
	if id != 0 {
		return id, nil // inserido agora
	}

	// reaproveita avaliador existente (usa sigla, que é única)
	err = db.QueryRow(`SELECT id FROM avaliador WHERE sigla = ?`, sigla).Scan(&id)
	return id, err
}
