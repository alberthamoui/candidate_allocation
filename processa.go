package main

import (
	dbpkg "candidate_alocator/db"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	reEmailPessoal = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
	reCPF          = regexp.MustCompile(`^\d{11}$`)
	reNumero       = regexp.MustCompile(`^\d{9}$`)
	reSemestre     = regexp.MustCompile(`^([1-9]|10)$`)
)

func getUsuarioFields(quantidade_opcoes int) []string {
	t := reflect.TypeOf(Candidato{})
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag != "" && tag != "-" {
			if tag == "opcoes" {
				for j := 1; j <= quantidade_opcoes; j++ {
					fields = append(fields, fmt.Sprintf("opcao %d", j))
				}
			} else {
				fields = append(fields, tag)
			}
		}
	}
	return fields
}

func getAvaliadorFields() []string {
	t := reflect.TypeOf(AvaliadorInfo{})
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	// preenche com "none" ou trunca para chegar em quantidade_opcoes
	return fields
}

func getRestricaoFields() []string {
	t := reflect.TypeOf(Restricao{})
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	return fields
}

type ErrorEntry struct {
	Field string `json:"field"`
	Msg   string `json:"msg"`
}
type ValidationResult struct {
	Erros   []ErrorEntry `json:"erros"`
	Usuario Candidato    `json:"usuario"`
}

// limpa dados

func processData(data []Candidato, emailDomain string) (map[int]ValidationResult, [][]int) {
	// Compila regex do email institucional dinamicamente com base no domínio configurado.
	// Escapa o domínio para uso seguro em regex (ex: "@al.insper.edu.br" → "@al\.insper\.edu\.br").
	escapedDomain := regexp.QuoteMeta(emailDomain)
	reEmailInstitucional := regexp.MustCompile(`^[^@]+` + escapedDomain + `$`)

	resultados := make(map[int]ValidationResult)

	// 1) validação e registro de TODAS as entradas (com ou sem erros)
	for idx, entrada := range data {
		var errs []ErrorEntry

		// --- trims básicos ---
		entrada.CPF = strings.TrimSpace(entrada.CPF)
		entrada.EmailInsper = strings.ToLower(strings.TrimSpace(entrada.EmailInsper))
		entrada.EmailPessoal = strings.ToLower(strings.TrimSpace(entrada.EmailPessoal))
		entrada.Numero = strings.TrimSpace(entrada.Numero)

		// --- validações por regex (e zera o campo quando inválido) ---
		if !reCPF.MatchString(entrada.CPF) {
			errs = append(errs, ErrorEntry{"cpf", "cpf inválido"})
		}
		if !reNumero.MatchString(entrada.Numero) {
			errs = append(errs, ErrorEntry{"numero", "numero inválido"})
		}
		if !reSemestre.MatchString(entrada.Semestre) {
			errs = append(errs, ErrorEntry{"semestre", "semestre inválido"})
			entrada.Semestre = ""
		}
		if !reEmailInstitucional.MatchString(entrada.EmailInsper) {
			errs = append(errs, ErrorEntry{"email_insper", fmt.Sprintf("email institucional inválido (esperado: ...%s)", emailDomain)})
			entrada.EmailInsper = ""
		}
		if !reEmailPessoal.MatchString(entrada.EmailPessoal) {
			errs = append(errs, ErrorEntry{"email_pessoal", "email_pessoal inválido"})
			entrada.EmailPessoal = ""
		}

		// armazena o resultado independentemente de erros
		resultados[idx+1] = ValidationResult{
			Erros:   errs,
			Usuario: entrada,
		}
	}

	// 2) checa duplicatas sobre TODAS as entradas processadas
	valueIndices := make(map[string][]int)
	for idx, resultado := range resultados {
		usr := resultado.Usuario
		if usr.CPF != "" {
			valueIndices["cpf:"+usr.CPF] = append(valueIndices["cpf:"+usr.CPF], idx)
		}
		if usr.EmailInsper != "" {
			valueIndices["email_insper:"+usr.EmailInsper] = append(valueIndices["email_insper:"+usr.EmailInsper], idx)
		}
		if usr.EmailPessoal != "" {
			valueIndices["email_pessoal:"+usr.EmailPessoal] = append(valueIndices["email_pessoal:"+usr.EmailPessoal], idx)
		}
	}

	// Union-Find para agrupar índices duplicados
	n := len(resultados)
	parent := make([]int, n+1)
	for i := 1; i <= n; i++ {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for _, indices := range valueIndices {
		if len(indices) > 1 {
			base := indices[0]
			for _, i := range indices[1:] {
				union(base, i)
			}
		}
	}
	groups := make(map[int][]int)
	for i := 1; i <= n; i++ {
		groups[find(i)] = append(groups[find(i)], i)
	}

	duplicatedIndices := make([][]int, 0)

	for _, g := range groups {
		if len(g) > 1 {
			duplicatedIndices = append(duplicatedIndices, g)
		}
	}
	return resultados, duplicatedIndices
}

func getHorarios(data []Candidato) []string {
	horariosMap := make(map[string]bool)
	for _, usuario := range data {
		for _, opcao := range usuario.Opcoes {
			horariosMap[opcao] = true
		}
	}

	horariosUnicos := []string{}
	for horario := range horariosMap {
		horariosUnicos = append(horariosUnicos, horario)
	}

	return horariosUnicos
}

func normalizaOpcao(op string) string {
	return strings.TrimSpace(strings.ToLower(op))
}

// fillDb grava candidatos, avaliadores ou restrições numa transação: se um
// registro falhar, nada é gravado e o erro volta para quem chamou.
// Restrições que citam candidato ou sigla inexistentes são ignoradas (com
// log), como antes; erros do banco não.
func fillDb(db *sql.DB, data interface{}) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback() // o erro que importa é o que causou o rollback
			return
		}
		err = tx.Commit()
	}()

	switch v := data.(type) {
	case []Candidato:
		return gravarCandidatos(tx, v)
	case []AvaliadorInfo:
		return gravarAvaliadores(tx, v)
	case []Restricao:
		return gravarRestricoes(tx, v)
	default:
		return fmt.Errorf("tipo de dado não suportado em fillDb: %T", data)
	}
}

func gravarCandidatos(tx *sql.Tx, candidatos []Candidato) error {
	idHorarios := map[string]int64{}
	for _, horario := range getHorarios(candidatos) {
		opcao := normalizaOpcao(horario)
		if _, visto := idHorarios[opcao]; visto || opcao == "" {
			continue // não grava lixo nem o mesmo horário escrito de outro jeito
		}
		id, err := dbpkg.AddHorario(tx, opcao)
		if err != nil {
			return fmt.Errorf("horário %q: %w", opcao, err)
		}
		idHorarios[opcao] = id
	}

	for _, usuario := range candidatos {
		// semestre inválido já aparece como erro na revisão; aqui vira 0
		semestreInt, _ := strconv.Atoi(usuario.Semestre)
		id, err := dbpkg.AddPessoa(tx, usuario.Nome, usuario.CPF, usuario.Numero, usuario.EmailInsper, usuario.EmailPessoal, semestreInt, usuario.Curso)
		if err != nil {
			return fmt.Errorf("candidato %q (CPF %q) não foi salvo: %w", usuario.Nome, usuario.CPF, err)
		}
		preferencia := int64(0)
		for _, opcao := range usuario.Opcoes {
			norm := normalizaOpcao(opcao)
			if norm == "" {
				continue
			}
			preferencia++
			if _, err := dbpkg.AddDisponibilidade(tx, id, idHorarios[norm], preferencia); err != nil {
				return fmt.Errorf("horário %q do candidato %q: %w", norm, usuario.Nome, err)
			}
		}
	}
	log.Printf("%d candidatos e %d horários gravados", len(candidatos), len(idHorarios))
	return nil
}

func gravarAvaliadores(tx *sql.Tx, avaliadores []AvaliadorInfo) error {
	for _, a := range avaliadores {
		if _, err := dbpkg.AddAvaliador(tx, a.Nome, a.Email, a.Sigla); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// INSERT OR IGNORE não inseriu e não há avaliador com essa sigla:
				// nome ou email repetem os de outro avaliador
				return fmt.Errorf("avaliador %q (sigla %q) não foi salvo: nome ou email repetido", a.Nome, a.Sigla)
			}
			return fmt.Errorf("avaliador %q (sigla %q) não foi salvo: %w", a.Nome, a.Sigla, err)
		}
	}
	log.Printf("%d avaliadores gravados", len(avaliadores))
	return nil
}

func gravarRestricoes(tx *sql.Tx, restricoes []Restricao) error {
	siglas := func(s string) []string {
		return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	}
	gravadas := 0
	for _, restricao := range restricoes {
		candidatoID, err := dbpkg.GetPessoaIDByName(tx, restricao.Candidato)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[WARN] restrição ignorada: candidato %q não encontrado", restricao.Candidato)
			continue
		} else if err != nil {
			return fmt.Errorf("restrição do candidato %q: %w", restricao.Candidato, err)
		}

		tipos := []struct {
			siglas  string
			inserir func(dbpkg.Executor, int64, int64) (int64, error)
		}{
			{restricao.NaoPosso, dbpkg.AddRestricaoNposso},
			{restricao.PrefiroNao, dbpkg.AddRestricaoPrefiroN},
		}
		for _, tipo := range tipos {
			for _, sigla := range siglas(tipo.siglas) {
				avalID, err := dbpkg.GetAvaliadorIDBySigla(tx, sigla)
				if errors.Is(err, sql.ErrNoRows) {
					log.Printf("[WARN] restrição ignorada: avaliador %q (candidato %q) não encontrado", sigla, restricao.Candidato)
					continue
				} else if err != nil {
					return fmt.Errorf("restrição do candidato %q com %q: %w", restricao.Candidato, sigla, err)
				}
				if _, err := tipo.inserir(tx, avalID, candidatoID); err != nil {
					return fmt.Errorf("restrição do candidato %q com %q: %w", restricao.Candidato, sigla, err)
				}
				gravadas++
			}
		}
	}
	log.Printf("%d restrições gravadas", gravadas)
	return nil
}
