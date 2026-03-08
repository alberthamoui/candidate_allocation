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

	"github.com/xuri/excelize/v2"
)

var (
	reEmailPessoal    = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
	reEmailSecundario = regexp.MustCompile(`^[^@]+@al\.insper\.edu\.br$`)
	reCPF             = regexp.MustCompile(`^\d{11}$`)
	reNumero          = regexp.MustCompile(`^\d{9}$`)
	reSemestre        = regexp.MustCompile(`^([1-9]|10)$`)
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
	Usuarios        map[int]ValidationResult `json:"usuarios"`
	Duplicates      [][]int                  `json:"duplicates"`
	DuplicateFields []string                 `json:"duplicateFields"`
}

// SuggestMapping lê a aba de candidatos e monta uma sugestão inicial de
// mapeamento entre colunas do Excel e campos esperados pela aplicação.
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

// SuggestMappingAvaliador faz a mesma sugestão de mapeamento, mas usando a aba
// de avaliadores e os campos definidos no tipo AvaliadorInfo.
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

// SuggestMappingRestricao sugere o mapeamento da aba de restrições com base nos
// campos declarados no tipo Restricao.
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

// suggestMappingForSheet concentra a lógica genérica de sugestão de mapping
// para qualquer aba a partir do índice da planilha e da lista de variáveis.
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

// BuildUsuariosWithMapping percorre a aba de candidatos, aplica o mapping
// definido na interface e devolve os candidatos já validados e agrupados.
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
		u, err := buildCandidateFromRow(row, nOpcoes, mappingItems)
		if err != nil {
			return UsuariosResponse{}, err
		}
		users = append(users, u)
	}
	users_limpo, duplicatedIndices := processData(users)

	return UsuariosResponse{
		Usuarios:        users_limpo,
		Duplicates:      duplicatedIndices,
		DuplicateFields: types.CandidateDuplicateFieldNames(),
	}, nil
}

// BuildAvaliadoresWithMapping monta os avaliadores a partir da segunda aba,
// ignora linhas vazias e já persiste o resultado no banco.
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
		av, err := buildStructFromRow[types.AvaliadorInfo](row, mappingItems)
		if err != nil {
			return nil, err
		}

		// ignora linhas totalmente vazias
		if isStructZeroValue(av) {
			continue
		}
		avaliadores = append(avaliadores, av)
	}
	Save(avaliadores)
	return avaliadores, nil
}

// BuildRestricoesWithMapping converte a aba de restrições em structs prontos
// para uso posterior no salvamento e na etapa de alocação.
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
		r, err := buildStructFromRow[types.Restricao](row, mappingItems)
		if err != nil {
			return nil, err
		}
		restricoes = append(restricoes, r)
	}

	return restricoes, nil
}

// SaveRestricoesFromMaps converte o payload genérico vindo do frontend para a
// struct de domínio e delega o salvamento para a função Save.
func SaveRestricoesFromMaps(restricaoMaps []map[string]interface{}) error {
	restricoes := make([]types.Restricao, 0, len(restricaoMaps))
	for _, m := range restricaoMaps {
		restricao, err := decodeMapToStruct[types.Restricao](m)
		if err != nil {
			return err
		}
		restricoes = append(restricoes, restricao)
	}

	return Save(restricoes)
}

// SaveUsuariosFromMaps converte o payload editado no frontend em candidatos e
// usa o fluxo padrão de persistência da aplicação.
func SaveUsuariosFromMaps(candidatoMaps []map[string]interface{}) error {
	candidatos := make([]types.Candidato, 0, len(candidatoMaps))
	for _, userMap := range candidatoMaps {
		candidato, err := decodeMapToStruct[types.Candidato](userMap)
		if err != nil {
			return err
		}
		candidatos = append(candidatos, candidato)
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

// getRowsFromSheet abre uma planilha em memória e retorna todas as linhas da
// aba indicada, validando se o índice realmente existe.
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

// getUsuarioFields lista os campos mapeáveis do tipo Candidato e expande o
// campo virtual de opções conforme a quantidade pedida.
func getUsuarioFields(quantidadeOpcoes int) []string {
	var fields []string
	for _, tag := range types.JSONFieldNames(types.Candidato{}) {
		if tag == "extras" {
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

// getAvaliadorFields devolve os nomes JSON dos campos que podem ser mapeados
// para avaliadores.
func getAvaliadorFields() []string {
	var fields []string
	for _, tag := range types.JSONFieldNames(types.AvaliadorInfo{}) {
		if tag == "extras" {
			continue
		}
		fields = append(fields, tag)
	}
	return fields
}

// getRestricaoFields devolve os nomes JSON usados no mapeamento de restrições.
func getRestricaoFields() []string {
	return types.RestricaoFieldNames()
}

// processData aplica validações básicas nos candidatos e encontra duplicidades
// por CPF e e-mails para alimentar a etapa de revisão manual.
func processData(data []types.Candidato) (map[int]ValidationResult, [][]int) {
	resultados := make(map[int]ValidationResult)

	for idx, entrada := range data {
		var errs []ErrorEntry

		entrada.CPF = strings.TrimSpace(entrada.CPF)
		entrada.EmailSecundario = strings.ToLower(strings.TrimSpace(entrada.EmailSecundario))
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
		if !reEmailSecundario.MatchString(entrada.EmailSecundario) {
			errs = append(errs, ErrorEntry{Field: 7, Msg: "email_secundario inválido"})
			entrada.EmailSecundario = ""
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
		for _, fieldName := range types.CandidateDuplicateFieldNames() {
			rawValue := candidateFieldValue(usr, fieldName)
			if rawValue == "" {
				continue
			}
			key := fieldName + ":" + rawValue
			valueIndices[key] = append(valueIndices[key], idx)
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

func candidateFieldValue(candidate types.Candidato, fieldName string) string {
	for _, field := range types.CandidateFields() {
		if field.JSONName != fieldName || field.Kind != reflect.String {
			continue
		}

		value := reflect.ValueOf(candidate).Field(field.Index)
		if !value.IsValid() {
			return ""
		}
		return strings.TrimSpace(value.String())
	}

	return ""
}

// getHorarios coleta todas as opções de horário presentes nos candidatos sem
// repetir valores, para depois persisti-las na tabela de horários.
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

// normalizaOpcao padroniza um horário removendo espaços extras e convertendo o
// texto para minúsculas antes de comparar ou salvar.
func normalizaOpcao(op string) string {
	return strings.TrimSpace(strings.ToLower(op))
}

// fillDb recebe um conjunto de dados de domínio e executa as inserções
// necessárias nas tabelas relacionadas do banco.
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
			id, err := dbpkg.InsertStruct(db, "pessoa", usuario, types.CandidateFields())
			if err != nil {
				fmt.Printf("Erro ao adicionar candidato %s: %v\n", usuario.Nome, err)
				continue
			}
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
			if _, err := dbpkg.InsertStruct(db, "avaliador", a, types.AvaliadorFields()); err != nil {
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

// Save abre a conexão padrão do projeto e delega a persistência para fillDb de
// acordo com o tipo concreto recebido.
func Save(data interface{}) error {
	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		return err
	}

	conn, err := dbpkg.OpenDefault()
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

// buildCandidateFromRow aplica o mapping de uma linha do Excel a um candidato,
// incluindo o preenchimento das opções em posições ordenadas.
func buildCandidateFromRow(row []string, nOpcoes int, mappingItems []types.MappingItem) (types.Candidato, error) {
	record := make(map[string]interface{})
	record["opcoes"] = make([]string, nOpcoes)

	for _, mItem := range mappingItems {
		if mItem.Indice >= len(row) {
			continue
		}

		cell := strings.TrimSpace(row[mItem.Indice])
		if strings.HasPrefix(mItem.Variavel, "opcao") {
			parts := strings.Split(mItem.Variavel, " ")
			if len(parts) == 2 {
				optionNum, err := strconv.Atoi(parts[1])
				if err == nil && optionNum > 0 && optionNum <= nOpcoes {
					record["opcoes"].([]string)[optionNum-1] = cell
				}
			}
			continue
		}

		record[mItem.Variavel] = cell
	}

	return decodeMapToStruct[types.Candidato](record)
}

// buildStructFromRow monta qualquer struct baseada em tags JSON a partir de uma
// linha da planilha e do mapping configurado.
func buildStructFromRow[T any](row []string, mappingItems []types.MappingItem) (T, error) {
	record := make(map[string]interface{})
	for _, mapping := range mappingItems {
		if mapping.Indice >= len(row) {
			continue
		}
		record[mapping.Variavel] = strings.TrimSpace(row[mapping.Indice])
	}

	return decodeMapToStruct[T](record)
}

// decodeMapToStruct usa JSON como ponte para popular uma struct tipada a partir
// de um map genérico vindo do Excel ou do frontend.
func decodeMapToStruct[T any](record map[string]interface{}) (T, error) {
	var target T

	payload, err := json.Marshal(record)
	if err != nil {
		return target, err
	}

	if err := json.Unmarshal(payload, &target); err != nil {
		return target, err
	}

	return target, nil
}

// isStructZeroValue verifica se todos os campos da struct ainda estão com seus
// valores zero, o que ajuda a ignorar linhas vazias.
func isStructZeroValue[T any](value T) bool {
	return reflect.DeepEqual(value, *new(T))
}
