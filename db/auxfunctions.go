package db

import (
	"database/sql"
	"strings"
)

// Executor é o que as funções abaixo precisam do banco: *sql.DB ou *sql.Tx
// (para gravar tudo numa transação).
type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// AddHorario insere um novo registro em opcoes_horario
func AddHorario(db Executor, opcao string) (int64, error) {
	opcao = strings.TrimSpace(strings.ToLower(opcao))
	res, err := db.Exec(`
		INSERT OR IGNORE INTO opcoes_horario (opcao) VALUES (?)	`, opcao)
	if err != nil {
		return 0, err
	}

	// INSERT OR IGNORE que não inseriu deixa LastInsertId com o id do insert
	// anterior (de outro registro): só confiar nele se inseriu
	if n, err := res.RowsAffected(); err != nil {
		return 0, err
	} else if n > 0 {
		return res.LastInsertId()
	}

	var existing int64
	err = db.QueryRow(`SELECT id FROM opcoes_horario WHERE opcao = ?`, opcao).Scan(&existing)
	return existing, err
}

// AddPessoa insere um novo registro em pessoa
func AddPessoa(db Executor, nome, cpf, numero, emailInsper, emailPessoal string, semestre int, curso string) (int64, error) {
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
func AddDisponibilidade(db Executor, pessoaID, horarioID, preferencia int64) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO disponibilidade (pessoa_id, horario_id, preferencia)
        VALUES (?, ?, ?)
    `, pessoaID, horarioID, preferencia)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func AddAvaliador(db Executor, nome, email, sigla string) (int64, error) {
	res, err := db.Exec(`
		INSERT OR IGNORE INTO avaliador (nome, email, sigla)
		VALUES (?, ?, ?)
	`, nome, email, sigla)
	if err != nil {
		return 0, err
	}

	// ver AddHorario: LastInsertId só vale se inseriu
	if n, err := res.RowsAffected(); err != nil {
		return 0, err
	} else if n > 0 {
		return res.LastInsertId()
	}

	// reaproveita avaliador existente (usa sigla, que é única); se não houver,
	// o nome ou o email repetem os de outro avaliador (sql.ErrNoRows)
	var id int64
	err = db.QueryRow(`SELECT id FROM avaliador WHERE sigla = ?`, sigla).Scan(&id)
	return id, err
}

func AddRestricaoNposso(db Executor, avaliadorID, candidatoID int64) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO restricoesNposso (avaliador_id, candidato_id)
        VALUES (?, ?)
    `, avaliadorID, candidatoID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func AddRestricaoPrefiroN(db Executor, avaliadorID, candidatoID int64) (int64, error) {
	res, err := db.Exec(`
        INSERT INTO restricoesPrefiroN (avaliador_id, candidato_id)
        VALUES (?, ?)
    `, avaliadorID, candidatoID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func GetAvaliadorIDBySigla(db Executor, sigla string) (int64, error) {
	var id int64
	err := db.QueryRow(
		`SELECT id FROM avaliador WHERE sigla = ?`,
		sigla,
	).Scan(&id)
	return id, err
}

func GetPessoaIDByName(db Executor, nome string) (int64, error) {
	var id int64
	err := db.QueryRow(
		`SELECT id FROM pessoa WHERE nome = ?`,
		nome,
	).Scan(&id)
	return id, err
}
