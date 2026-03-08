package logic

import (
	"bytes"
	dbpkg "candidate_alocator/back/db"
	types "candidate_alocator/back/type"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
)

var (
	reEmailPessoal = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
	reEmailInsper  = regexp.MustCompile(`^[^@]+@al\.insper\.edu\.br$`)
	reCPF          = regexp.MustCompile(`^\d{11}$`)
	reNumero       = regexp.MustCompile(`^\d{9}$`)
	reSemestre     = regexp.MustCompile(`^([1-9]|10)$`)
)

type ErrorEntry struct {
	Field int    `json:"field"`
	Msg   string `json:"msg"`
}

type ValidationResult struct {
	Erros   []ErrorEntry    `json:"erros"`
	Usuario types.Candidato `json:"usuario"`
}

type UsuariosResponse struct {
	Usuarios   map[int]ValidationResult `json:"usuarios"`
	Duplicates [][]int                  `json:"duplicates"`
}

func SuggestMapping(data []byte, quantidadeOpcoes int) ([]types.MappingItem, error) {
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)

	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	header := rows[0]
	// fmt.Println("header : ", header)

	// Lista de possíveis variáveis da struct Usuario (em minúsculo)
	variaveisUsuario := getUsuarioFields(quantidadeOpcoes)
	fmt.Println("variaveis usuario: ", variaveisUsuario)

	// Alocação aleatória das variáveis para cada coluna
	mappingList := make([]string, len(variaveisUsuario))
	for i, usuarioVar := range variaveisUsuario {
		if i < len(header) {
			mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", header[i], i, usuarioVar)
		} else {
			mappingList[i] = fmt.Sprintf("[[null, %d], %q]", i, usuarioVar)
		}
	}
	fmt.Println(mappingList)
	mapping_json, err := ProcessMapping(mappingList)
	if err != nil {
		return nil, err
	}
	return mapping_json, nil
}

func SuggestMappingAvaliador(data []byte) ([]types.MappingItem, error) {
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(1) // Avaliador
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	header := rows[0]

	// gerar dinamicamente as variáveis de avaliador
	variaveisAvaliador := getAvaliadorFields()

	fmt.Println("variaveis avaliador : ", variaveisAvaliador)
	mappingList := make([]string, len(variaveisAvaliador))
	for i, v := range variaveisAvaliador {
		if i < len(header) {
			mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", header[i], i, v)
		} else {
			mappingList[i] = fmt.Sprintf("[[null, %d], %q]", i, v)
		}
	}
	return ProcessMapping(mappingList)
}

func SuggestMappingRestricao(data []byte) ([]types.MappingItem, error) {
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(2) // Restrição
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	header := rows[0]

	// gerar dinamicamente as variáveis de restrição
	variaveisRestricao := getRestricaoFields()
	mappingList := make([]string, len(variaveisRestricao))
	for i, v := range variaveisRestricao {
		if i < len(header) {
			mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", header[i], i, v)
		} else {
			mappingList[i] = fmt.Sprintf("[[null, %d], %q]", i, v)
		}
	}
	return ProcessMapping(mappingList)
}

func suggestMappingForSheet(data []byte, sheetIndex int, variables []string) ([]types.MappingItem, error) {
	rows, err := getRowsFromSheet(data, sheetIndex)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}

	header := rows[0]
	mappingList := make([]string, len(variables))
	for i, variable := range variables {
		if i < len(header) {
			mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", header[i], i, variable)
		} else {
			mappingList[i] = fmt.Sprintf("[[null, %d], %q]", i, variable)
		}
	}

	return ProcessMapping(mappingList)
}

func BuildUsuariosWithMapping(data []byte, nOpcoes int, mappingItems []types.MappingItem) (UsuariosResponse, error) {
	// Abre o arquivo Excel a partir dos dados em []byte
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return UsuariosResponse{}, err
	}
	defer file.Close()

	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return UsuariosResponse{}, fmt.Errorf("erro ao ler excel : %w", err)
	}
	if len(rows) < 2 {
		return UsuariosResponse{}, fmt.Errorf("arquivo sem dados além do header")
	}

	var users []types.Candidato

	for _, row := range rows[1:] {
		u := types.Candidato{
			Opcoes: make([]string, nOpcoes),
		}
		// Para cada mapping, pega o conteúdo da coluna correspondente e atribui
		for _, mItem := range mappingItems {
			if mItem.Indice >= len(row) {
				continue
			}
			cell := row[mItem.Indice]
			switch mItem.Variavel {
			case "timestamp":
				u.Timestamp = cell
			case "nome":
				u.Nome = cell
			case "cpf":
				u.CPF = cell
			case "numero":
				u.Numero = cell
			case "semestre":
				u.Semestre = cell
			case "curso":
				u.Curso = cell
			case "email_insper":
				u.EmailInsper = cell
			case "email_pessoal":
				u.EmailPessoal = cell
			default:
				// Se for uma opção, o mItem.UserField deve estar no formato "opcao X"
				if strings.HasPrefix(mItem.Variavel, "opcao") {
					parts := strings.Split(mItem.Variavel, " ")
					if len(parts) == 2 {
						optionNum, err := strconv.Atoi(parts[1])
						if err == nil && optionNum > 0 && optionNum <= nOpcoes {
							u.Opcoes[optionNum-1] = cell
						}
					}
				}
			}
		}
		users = append(users, u)
	}
	users_limpo, duplicatedIndices := processData(users)

	return UsuariosResponse{Usuarios: users_limpo, Duplicates: duplicatedIndices}, nil
}

func BuildAvaliadoresWithMapping(data []byte, mappingItems []types.MappingItem) ([]types.AvaliadorInfo, error) {

	if data == nil {
		return nil, fmt.Errorf("dados do Excel ainda não carregados")
	}

	// abre planilha a partir do []byte salvo em a.excelData
	reader := bytes.NewReader(data)
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("erro abrindo excel: %w", err)
	}
	defer file.Close()

	// 2ª aba (índice 1) onde estão os avaliadores
	sheet := file.GetSheetName(1)
	if sheet == "" {
		return nil, fmt.Errorf("arquivo não possui uma segunda aba com avaliadores")
	}

	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("erro lendo aba de avaliadores: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("aba de avaliadores não contém dados além do cabeçalho")
	}

	var avaliadores []types.AvaliadorInfo

	// percorre linhas (ignorando cabeçalho)
	for _, row := range rows[1:] {
		av := types.AvaliadorInfo{}
		for _, m := range mappingItems {
			if m.Indice >= len(row) {
				continue // coluna vazia nesta linha
			}
			val := strings.TrimSpace(row[m.Indice])

			switch strings.ToLower(m.Variavel) {
			case "nome":
				av.Nome = val
			case "email":
				av.Email = val
			case "sigla":
				av.Sigla = val
			}
		}

		// ignora linhas totalmente vazias
		if av.Nome == "" && av.Email == "" && av.Sigla == "" {
			continue
		}
		avaliadores = append(avaliadores, av)
	}
	Save(avaliadores)
	return avaliadores, nil
}

func BuildRestricoesWithMapping(data []byte, mappingItems []types.MappingItem) ([]types.Restricao, error) {
	rows, err := getRowsFromSheet(data, 2)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler excel: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("arquivo sem dados além do header")
	}

	var restricoes []types.Restricao
	for _, row := range rows[1:] {
		r := types.Restricao{}
		for _, m := range mappingItems {
			if m.Indice >= len(row) {
				continue
			}

			cell := row[m.Indice]
			switch m.Variavel {
			case "candidato":
				r.Candidato = cell
			case "naoPosso":
				r.NaoPosso = cell
			case "prefiroNao":
				r.PrefiroNao = cell
			}
		}
		restricoes = append(restricoes, r)
	}

	return restricoes, nil
}

func SaveRestricoesFromMaps(restricaoMaps []map[string]interface{}) error {
	var restricoes []types.Restricao
	for _, m := range restricaoMaps {
		restricoes = append(restricoes, types.Restricao{
			Candidato:  getStringFromMap(m, "candidato"),
			NaoPosso:   getStringFromMap(m, "naoPosso"),
			PrefiroNao: getStringFromMap(m, "prefiroNao"),
		})
	}

	return Save(restricoes)
}

func SaveUsuariosFromMaps(candidatoMaps []map[string]interface{}) error {
	var candidatos []types.Candidato
	for _, userMap := range candidatoMaps {
		candidatos = append(candidatos, types.Candidato{
			Timestamp:    getStringFromMap(userMap, "timestamp"),
			Nome:         getStringFromMap(userMap, "nome"),
			CPF:          getStringFromMap(userMap, "cpf"),
			Numero:       getStringFromMap(userMap, "numero"),
			Semestre:     getStringFromMap(userMap, "semestre"),
			Curso:        getStringFromMap(userMap, "curso"),
			EmailInsper:  getStringFromMap(userMap, "email_insper"),
			EmailPessoal: getStringFromMap(userMap, "email_pessoal"),
			Opcoes:       getSliceFromMap(userMap, "opcoes"),
		})
	}

	return Save(candidatos)
}

// ProcessMapping converte cada string JSON "[[nomeColuna,indice],variavel]"
// em um Mapping. Retorna erro se algum item não for JSON válido.
func ProcessMapping(items []string) ([]types.MappingItem, error) {
	var result []types.MappingItem

	for _, item := range items {
		// decodifica o JSON em um slice genérico
		var arr []interface{}
		if err := json.Unmarshal([]byte(item), &arr); err != nil {
			return nil, fmt.Errorf("invalid JSON '%s': %w", item, err)
		}
		if len(arr) != 2 {
			continue
		}

		// arr[0] => [nomeColuna, indice]
		info, ok := arr[0].([]interface{})
		if !ok || len(info) != 2 {
			continue
		}
		var nomeColuna string
		if info[0] != nil {
			s, ok := info[0].(string)
			if !ok {
				continue
			}
			nomeColuna = s
		}

		indiceF, ok2 := info[1].(float64)
		variavel, ok3 := arr[1].(string)
		if !ok2 || !ok3 {
			continue
		}

		result = append(result, types.MappingItem{
			NomeColuna: nomeColuna,
			Indice:     int(indiceF),
			Variavel:   variavel,
		})
	}

	return result, nil
}

func FilterUniqueUsers(resp UsuariosResponse) []types.Candidato {
	skip := make(map[int]struct{})
	for _, grp := range resp.Duplicates {
		for i, idx := range grp {
			if i > 0 {
				skip[idx] = struct{}{}
			}
		}
	}

	var out []types.Candidato
	for idx, vr := range resp.Usuarios {
		if _, isDup := skip[idx]; isDup {
			continue
		}
		out = append(out, vr.Usuario)
	}

	return out
}

func getRowsFromSheet(data []byte, sheetIndex int) ([][]string, error) {
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(sheetIndex)
	if sheet == "" {
		return nil, fmt.Errorf("arquivo nao possui a aba esperada no indice %d", sheetIndex)
	}

	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func getUsuarioFields(quantidadeOpcoes int) []string {
	t := reflect.TypeOf(types.Candidato{})
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		if tag == "opcoes" {
			for j := 1; j <= quantidadeOpcoes; j++ {
				fields = append(fields, fmt.Sprintf("opcao %d", j))
			}
			continue
		}
		fields = append(fields, tag)
	}
	return fields
}

func getAvaliadorFields() []string {
	t := reflect.TypeOf(types.AvaliadorInfo{})
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	return fields
}

func getRestricaoFields() []string {
	t := reflect.TypeOf(types.Restricao{})
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	return fields
}

func processData(data []types.Candidato) (map[int]ValidationResult, [][]int) {
	resultados := make(map[int]ValidationResult)

	for idx, entrada := range data {
		var errs []ErrorEntry

		entrada.CPF = strings.TrimSpace(entrada.CPF)
		entrada.EmailInsper = strings.ToLower(strings.TrimSpace(entrada.EmailInsper))
		entrada.EmailPessoal = strings.ToLower(strings.TrimSpace(entrada.EmailPessoal))
		entrada.Numero = strings.TrimSpace(entrada.Numero)

		if !reCPF.MatchString(entrada.CPF) {
			errs = append(errs, ErrorEntry{Field: 3, Msg: "cpf inválido"})
		}
		if !reNumero.MatchString(entrada.Numero) {
			errs = append(errs, ErrorEntry{Field: 4, Msg: "numero inválido"})
		}
		if !reSemestre.MatchString(entrada.Semestre) {
			errs = append(errs, ErrorEntry{Field: 5, Msg: "semestre inválido"})
			entrada.Semestre = ""
		}
		if !reEmailInsper.MatchString(entrada.EmailInsper) {
			errs = append(errs, ErrorEntry{Field: 7, Msg: "email_insper inválido"})
			entrada.EmailInsper = ""
		}
		if !reEmailPessoal.MatchString(entrada.EmailPessoal) {
			errs = append(errs, ErrorEntry{Field: 8, Msg: "email_pessoal inválido"})
			entrada.EmailPessoal = ""
		}

		resultados[idx+1] = ValidationResult{
			Erros:   errs,
			Usuario: entrada,
		}
	}

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
		if len(indices) <= 1 {
			continue
		}
		base := indices[0]
		for _, i := range indices[1:] {
			union(base, i)
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

func getHorarios(data []types.Candidato) []string {
	horariosMap := make(map[string]bool)
	for _, usuario := range data {
		for _, opcao := range usuario.Opcoes {
			horariosMap[opcao] = true
		}
	}

	var horariosUnicos []string
	for horario := range horariosMap {
		horariosUnicos = append(horariosUnicos, horario)
	}
	return horariosUnicos
}

func normalizaOpcao(op string) string {
	return strings.TrimSpace(strings.ToLower(op))
}

func fillDb(db *sql.DB, data interface{}) {
	switch v := data.(type) {
	case []types.Candidato:
		idHorarios := map[string]int64{}
		for _, horario := range getHorarios(v) {
			opcao := normalizaOpcao(horario)
			if opcao == "" {
				continue
			}
			idHorario, _ := dbpkg.AddHorario(db, opcao)
			idHorarios[opcao] = idHorario
		}

		for _, usuario := range v {
			semestreInt, _ := strconv.Atoi(usuario.Semestre)
			id, _ := dbpkg.AddPessoa(db, usuario.Nome, usuario.CPF, usuario.Numero, usuario.EmailInsper, usuario.EmailPessoal, semestreInt, usuario.Curso)
			for idx, opcao := range usuario.Opcoes {
				opcaoNormalizada := normalizaOpcao(opcao)
				if opcaoNormalizada == "" {
					continue
				}
				horarioID, ok := idHorarios[opcaoNormalizada]
				if !ok || horarioID == 0 {
					continue
				}
				dbpkg.AddDisponibilidade(db, id, horarioID, int64(idx+1))
			}
		}
	case []types.AvaliadorInfo:
		for _, a := range v {
			if _, err := dbpkg.AddAvaliador(db, a.Nome, a.Email, a.Sigla); err != nil {
				fmt.Printf("Erro ao adicionar avaliador %s: %v\n", a.Nome, err)
			}
		}
	case []types.Restricao:
		for _, restricao := range v {
			candidatoID, err := dbpkg.GetPessoaIDByName(db, restricao.Candidato)
			if err != nil {
				fmt.Printf("Erro ao pegar o id do candidato %s: %v\n", restricao.Candidato, err)
				continue
			}

			if restricao.NaoPosso != "" {
				parts := strings.FieldsFunc(restricao.NaoPosso, func(r rune) bool {
					return r == ',' || unicode.IsSpace(r)
				})
				for _, sig := range parts {
					sigla := strings.TrimSpace(sig)
					avalID, err := dbpkg.GetAvaliadorIDBySigla(db, sigla)
					if err != nil {
						fmt.Printf("Erro no GetAvaliadorID (%s): %v\n", sigla, err)
						continue
					}
					if _, err := dbpkg.AddRestricaoNposso(db, avalID, candidatoID); err != nil {
						fmt.Printf("Erro ao inserir NaoPosso [%s] p/ candidato %d: %v\n", sigla, candidatoID, err)
					}
				}
			}

			if restricao.PrefiroNao != "" {
				parts := strings.FieldsFunc(restricao.PrefiroNao, func(r rune) bool {
					return r == ',' || unicode.IsSpace(r)
				})
				for _, sig := range parts {
					sigla := strings.TrimSpace(sig)
					avalID, err := dbpkg.GetAvaliadorIDBySigla(db, sigla)
					if err != nil {
						fmt.Printf("Erro no GetAvaliadorID (%s): %v\n", sigla, err)
						continue
					}
					if _, err := dbpkg.AddRestricaoPrefiroN(db, avalID, candidatoID); err != nil {
						fmt.Printf("Erro ao inserir PrefiroNao [%s] p/ candidato %d: %v\n", sigla, candidatoID, err)
					}
				}
			}
		}
	default:
		fmt.Println("Tipo de dado não suportado em fillDb")
	}
}

func Save(data interface{}) error {
	conn, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		return err
	}
	defer conn.Close()

	switch data.(type) {
	case []types.Candidato, []types.AvaliadorInfo, []types.Restricao:
		fillDb(conn, data)
	default:
		log.Printf("Tipo de dado não suportado em fillDb: %T", data)
	}

	return nil
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getSliceFromMap(m map[string]interface{}, key string) []string {
	if val, ok := m[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			var result []string
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}
