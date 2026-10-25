package logic

import (
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/diagnostics"
	types "candidate_alocator/back/type"
	"database/sql"
	"fmt"
	"strings"
	"unicode"
)

// ImportData is the complete reviewed import, committed as one unit.
type ImportData struct {
	Candidatos  []types.Candidato
	Avaliadores []types.Avaliador
	Restricoes  []types.Restricao
}

func DecodeImportMaps(candidates, evaluators, restrictions []map[string]interface{}) (ImportData, error) {
	var data ImportData
	for i, row := range candidates {
		v, err := decodeMapToStruct[types.Candidato](row)
		if err != nil {
			return data, fmt.Errorf("candidatos, registro %d: %w", i+1, err)
		}
		data.Candidatos = append(data.Candidatos, v)
	}
	for i, row := range evaluators {
		v, err := decodeMapToStruct[types.Avaliador](row)
		if err != nil {
			return data, fmt.Errorf("avaliadores, registro %d: %w", i+1, err)
		}
		data.Avaliadores = append(data.Avaliadores, v)
	}
	for i, row := range restrictions {
		v, err := decodeMapToStruct[types.Restricao](row)
		if err != nil {
			return data, fmt.Errorf("restrições, registro %d: %w", i+1, err)
		}
		data.Restricoes = append(data.Restricoes, v)
	}
	return data, nil
}

// ReplaceImport preserves the previous database if any reviewed row fails.
func ReplaceImport(db *sql.DB, data ImportData) error {
	defer diagnostics.Begin("import.persist", len(data.Candidatos))()
	if len(data.Candidatos) == 0 {
		return fmt.Errorf("a importação precisa de ao menos um candidato")
	}
	return saveTransaction(db, true, data.Candidatos, data.Avaliadores, data.Restricoes)
}

func Save(data interface{}) error {
	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		return err
	}
	db, err := dbpkg.OpenDefault()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	return saveTransaction(db, false, data)
}

func saveTransaction(db *sql.DB, replace bool, batches ...any) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("não foi possível iniciar a gravação: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if replace {
		if err := dbpkg.ClearDatabase(tx); err != nil {
			return err
		}
	}
	for _, batch := range batches {
		if err := fillDb(tx, batch); err != nil {
			return fmt.Errorf("nenhum dado desta operação foi gravado: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("não foi possível confirmar a gravação: %w", err)
	}
	return nil
}

func fillDb(db dbpkg.Connection, data interface{}) error {
	switch rows := data.(type) {
	case []types.Candidato:
		ids := map[string]int64{}
		for i, row := range rows {
			row.Nome = strings.TrimSpace(row.Nome)
			if row.Nome == "" {
				return fmt.Errorf("candidatos, registro %d: nome obrigatório", i+1)
			}
			id, err := dbpkg.InsertStruct(db, "pessoa", row, types.CandidateFields())
			if err != nil {
				return fmt.Errorf("candidatos, registro %d: %w", i+1, err)
			}
			for rank, raw := range row.Opcoes {
				option := normalizaOpcao(raw)
				if option == "" {
					continue
				}
				scheduleID, ok := ids[option]
				if !ok {
					scheduleID, err = dbpkg.AddHorario(db, option)
					if err != nil {
						return fmt.Errorf("candidatos, registro %d, preferência %d: %w", i+1, rank+1, err)
					}
					ids[option] = scheduleID
				}
				if _, err := dbpkg.AddDisponibilidade(db, id, scheduleID, int64(rank+1)); err != nil {
					return fmt.Errorf("candidatos, registro %d, preferência %d: %w", i+1, rank+1, err)
				}
			}
		}
	case []types.Avaliador:
		for i, row := range rows {
			row.Nome, row.Sigla, row.Email = strings.TrimSpace(row.Nome), strings.TrimSpace(row.Sigla), strings.TrimSpace(row.Email)
			if row.Nome == "" || row.Sigla == "" || row.Email == "" {
				return fmt.Errorf("avaliadores, registro %d: preencha nome, sigla e e-mail", i+1)
			}
			if _, err := dbpkg.InsertStruct(db, "avaliador", row, types.AvaliadorFields()); err != nil {
				return fmt.Errorf("avaliadores, registro %d: %w", i+1, err)
			}
		}
	case []types.Restricao:
		for i, row := range rows {
			if strings.TrimSpace(row.Candidato+row.NaoPosso+row.PrefiroNao) == "" {
				continue
			}
			id, err := dbpkg.GetPessoaIDByName(db, row.Candidato)
			if err != nil {
				return fmt.Errorf("restrições, registro %d: %w", i+1, err)
			}
			for _, kind := range []struct {
				name, value string
				add         func(dbpkg.Connection, int64, int64) (int64, error)
			}{
				{"Não posso", row.NaoPosso, dbpkg.AddRestricaoNposso},
				{"Prefiro não", row.PrefiroNao, dbpkg.AddRestricaoPrefiroN},
			} {
				seen := map[string]bool{}
				for _, code := range strings.FieldsFunc(kind.value, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
					if seen[code] {
						continue
					}
					seen[code] = true
					evaluatorID, err := dbpkg.GetAvaliadorIDBySigla(db, code)
					if err != nil {
						return fmt.Errorf("restrições, registro %d, %s: sigla %q não encontrada; revise os avaliadores: %w", i+1, kind.name, code, err)
					}
					if _, err := kind.add(db, evaluatorID, id); err != nil {
						return fmt.Errorf("restrições, registro %d, %s: %w", i+1, kind.name, err)
					}
				}
			}
		}
	default:
		return fmt.Errorf("tipo de dado não suportado para gravação: %T", data)
	}
	return nil
}
