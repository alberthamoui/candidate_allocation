package db

import (
	"database/sql"
)

// --- removido var Conn *sql.DB e Configure ---

// AddHorario insere um novo registro em opcoes_horario
func AddHorario(db *sql.DB, data, hora, local string) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO opcoes_horario (data, hora, local)
        VALUES (?, ?, ?)
    `, data, hora, local)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
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
