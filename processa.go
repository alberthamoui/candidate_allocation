package main

import (
	dbpkg "candidate_alocator/db"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	reEmailPessoal = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
	reEmailInsper  = regexp.MustCompile(`^[^@]+@al\.insper\.edu\.br$`)
	reCPF          = regexp.MustCompile(`^\d{11}$`)
	reNumero       = regexp.MustCompile(`^\d{9}$`)
	reSemestre     = regexp.MustCompile(`^([1-9]|10)$`)
)

func getUsuarioFields(quantidade_opcoes int) []string {
	t := reflect.TypeOf(Usuario{})
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
	Field int    `json:"field"`
	Msg   string `json:"msg"`
}
type ValidationResult struct {
	Erros   []ErrorEntry `json:"erros"`
	Usuario Usuario      `json:"usuario"`
}

// limpa dados

func processData(data []Usuario) (map[int]ValidationResult, [][]int) {
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
			errs = append(errs, ErrorEntry{3, "cpf inválido"})
		}
		if !reNumero.MatchString(entrada.Numero) {
			errs = append(errs, ErrorEntry{4, "numero inválido"})
		}
		if !reSemestre.MatchString(entrada.Semestre) {
			errs = append(errs, ErrorEntry{5, "semestre inválido"})
			entrada.Semestre = ""
		}
		if !reEmailInsper.MatchString(entrada.EmailInsper) {
			errs = append(errs, ErrorEntry{7, "email_insper inválido"})
			entrada.EmailInsper = ""
		}
		if !reEmailPessoal.MatchString(entrada.EmailPessoal) {
			errs = append(errs, ErrorEntry{8, "email_pessoal inválido"})
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

func getHorarios(data []Usuario) []string {
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
func FilterUniqueUsers(resp UsuariosResponse) []Usuario {
	// marca todos os índices duplicados, exceto o primeiro de cada grupo
	skip := make(map[int]struct{})
	for _, grp := range resp.Duplicates {
		for i, idx := range grp {
			if i > 0 {
				skip[idx] = struct{}{}
			}
		}
	}
	// coleta os usuários mantendo a ordem arbitrária do map
	var out []Usuario
	for idx, vr := range resp.Usuarios {
		if _, isDup := skip[idx]; isDup {
			continue
		}
		out = append(out, vr.Usuario)
	}
	return out
}

func normalizaOpcao(op string) string {
	return strings.TrimSpace(strings.ToLower(op))
}

func fillDb(db *sql.DB, data interface{}) {
	switch v := data.(type) {
	case []Usuario:
		// HORARIOS
		idHorarios := map[string]int64{}
		horarios := getHorarios(v)
		for _, horario := range horarios {
			opcao := normalizaOpcao(horario)
			if opcao == "" {
				continue // não grava lixo
			}
			idHorario, _ := dbpkg.AddHorario(db, opcao)
			idHorarios[opcao] = idHorario
		}

		fmt.Println("Horários inseridos no banco de dados.")

		// CANDIDATOS & DISPONIBILIDADES
		for _, usuario := range v {
			semestreInt, _ := strconv.Atoi(usuario.Semestre)
			id, _ := dbpkg.AddPessoa(db, usuario.Nome, usuario.CPF, usuario.Numero, usuario.EmailInsper, usuario.EmailPessoal, semestreInt, usuario.Curso)
			fmt.Printf("Adicionando usuário: %s (ID: %d)\n", usuario.Nome, id)
			count := 0
			for _, opcao := range usuario.Opcoes {
				count++
				fmt.Printf("Adicionando disponibilidade para usuário %s (ID: %d) - Horário: %s (ID HORARIO: %d)\n", usuario.Nome, id, opcao, idHorarios[opcao])
				dbpkg.AddDisponibilidade(db, id, idHorarios[opcao], int64(count))
			}
		}
	case []AvaliadorInfo:
		for _, a := range v {
			id, err := dbpkg.AddAvaliador(db, a.Nome, a.Email, a.Sigla)
			if err != nil {
				fmt.Printf("Erro ao adicionar avaliador %s: %v\n", a.Nome, err)
			} else {
				fmt.Printf("Avaliador %s adicionado com ID %d\n", a.Nome, id)
			}
		}
	case []Restricao:
		for _, restricao := range v {
			CandidatoId, err := dbpkg.GetPessoaIDByName(db, restricao.Candidato)
			if err != nil {
				fmt.Printf("Erro ao  pegar o id do candidato %s: %v\n", restricao.Candidato, err)
			}
			id, err := dbpkg.AddRestricao(db, CandidatoId, restricao.NaoPosso, restricao.PrefiroNao)
			if err != nil {
				fmt.Printf("Erro ao adicionar restrição para candidato %d: %v\n", CandidatoId, err)
			} else {
				fmt.Printf("Restrição adicionada para candidato %d com ID %d\n", CandidatoId, id)
			}
		}

	default:
		fmt.Println("Tipo de dado não suportado em fillDb")
	}
}
