package db

import (
	types "candidate_alocator/back/type"
	"database/sql"
	"fmt"
	"strings"
)

func EnsureAppSchema(db *sql.DB) error {
	statements := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS "opcoes_horario" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"opcao" TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS "disponibilidade" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"pessoa_id" INTEGER NOT NULL,
			"horario_id" INTEGER NOT NULL,
			"preferencia" INTEGER NOT NULL,
			FOREIGN KEY("horario_id") REFERENCES "opcoes_horario"("id"),
			FOREIGN KEY("pessoa_id") REFERENCES "pessoa"("id")
		);`,
		`CREATE TABLE IF NOT EXISTS "restricoesNposso" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"avaliador_id" INTEGER NOT NULL,
			"candidato_id" INTEGER NOT NULL,
			FOREIGN KEY("avaliador_id") REFERENCES "avaliador"("id") ON DELETE CASCADE,
			FOREIGN KEY("candidato_id") REFERENCES "pessoa"("id") ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS "restricoesPrefiroN" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"avaliador_id" INTEGER NOT NULL,
			"candidato_id" INTEGER NOT NULL,
			FOREIGN KEY("avaliador_id") REFERENCES "avaliador"("id") ON DELETE CASCADE,
			FOREIGN KEY("candidato_id") REFERENCES "pessoa"("id") ON DELETE CASCADE
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "idx_disponibilidade_unique"
			ON "disponibilidade" ("pessoa_id", "horario_id", "preferencia");`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("erro ao executar statement %q: %w", stmt, err)
		}
	}

	if err := ensureDynamicTable(db, "avaliador", types.AvaliadorFields()); err != nil {
		return err
	}

	if err := ensureDynamicTable(db, "pessoa", types.CandidateFields()); err != nil {
		return err
	}

	return nil
}

func ClearDatabase(db *sql.DB) error {
	tables := []string{
		"disponibilidade",
		"restricoesNposso",
		"restricoesPrefiroN",
		"opcoes_horario",
		"avaliador",
		"pessoa",
	}

	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)); err != nil {
			return fmt.Errorf("erro ao limpar tabela %s: %w", table, err)
		}
		// Reseta o auto-incremento para cada tabela
		if _, err := db.Exec(fmt.Sprintf(`DELETE FROM sqlite_sequence WHERE name='%s'`, table)); err != nil {
			// Ignora erro se a tabela não tiver auto-incremento
			continue
		}
	}

	return nil
}

func ensureDynamicTable(db *sql.DB, table string, fields []types.FieldSchema) error {
	createStmt := buildCreateTableStatement(table, fields)
	if _, err := db.Exec(createStmt); err != nil {
		return fmt.Errorf("erro ao criar tabela %s: %w", table, err)
	}

	existingColumns, err := getExistingColumns(db, table)
	if err != nil {
		return err
	}

	for _, field := range fields {
		if !field.Persist {
			continue
		}
		if _, ok := existingColumns[field.ColumnName]; ok {
			continue
		}

		alterStmt := fmt.Sprintf(
			`ALTER TABLE "%s" ADD COLUMN "%s" %s`,
			table,
			field.ColumnName,
			field.SQLiteType,
		)
		if _, err := db.Exec(alterStmt); err != nil {
			return fmt.Errorf("erro ao adicionar coluna %s.%s: %w", table, field.ColumnName, err)
		}
	}

	expectedUniqueIndexes := expectedUniqueIndexMap(table, fields)
	existingIndexes, err := getExistingIndexes(db, table)
	if err != nil {
		return err
	}

	for indexName := range existingIndexes {
		if !strings.HasPrefix(indexName, fmt.Sprintf("idx_%s_", table)) || !strings.HasSuffix(indexName, "_unique") {
			continue
		}
		if _, shouldExist := expectedUniqueIndexes[indexName]; shouldExist {
			continue
		}

		dropStmt := fmt.Sprintf(`DROP INDEX IF EXISTS "%s"`, indexName)
		if _, err := db.Exec(dropStmt); err != nil {
			return fmt.Errorf("erro ao remover índice único obsoleto %s: %w", indexName, err)
		}
	}

	for _, field := range fields {
		indexName, ok := expectedUniqueIndexes[uniqueIndexNameForColumn(table, field.ColumnName)]
		if !ok {
			continue
		}
		stmt := fmt.Sprintf(
			`CREATE UNIQUE INDEX IF NOT EXISTS "%s" ON "%s" ("%s") WHERE "%s" IS NOT NULL`,
			indexName,
			table,
			field.ColumnName,
			field.ColumnName,
		)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("erro ao criar índice único %s: %w", indexName, err)
		}
	}

	return nil
}

func expectedUniqueIndexMap(table string, fields []types.FieldSchema) map[string]string {
	indexes := make(map[string]string)
	for _, field := range fields {
		if !field.Persist || !field.Unique {
			continue
		}
		name := uniqueIndexNameForColumn(table, field.ColumnName)
		indexes[name] = name
	}
	return indexes
}

func buildCreateTableStatement(table string, fields []types.FieldSchema) string {
	definitions := []string{`"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT`}
	for _, field := range fields {
		if !field.Persist {
			continue
		}

		definition := fmt.Sprintf(`"%s" %s`, field.ColumnName, field.SQLiteType)
		if field.Required {
			definition += ` NOT NULL`
		}
		definitions = append(definitions, definition)
	}

	return fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS "%s" (%s);`,
		table,
		strings.Join(definitions, ", "),
	)
}

func getExistingColumns(db *sql.DB, table string) (map[string]struct{}, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, table))
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar schema da tabela %s: %w", table, err)
	}
	defer rows.Close()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal interface{}
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return nil, fmt.Errorf("erro ao ler colunas da tabela %s: %w", table, err)
		}
		columns[name] = struct{}{}
	}

	return columns, rows.Err()
}

func getExistingIndexes(db *sql.DB, table string) (map[string]struct{}, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA index_list("%s")`, table))
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar índices da tabela %s: %w", table, err)
	}
	defer rows.Close()

	indexes := make(map[string]struct{})
	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, fmt.Errorf("erro ao ler índices da tabela %s: %w", table, err)
		}
		indexes[name] = struct{}{}
	}

	return indexes, rows.Err()
}

func uniqueIndexNameForColumn(table, column string) string {
	return fmt.Sprintf("idx_%s_%s_unique", table, column)
}
