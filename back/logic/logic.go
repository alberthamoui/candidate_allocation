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
	"sort"
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

type AvaliadorValidationResult struct {
	Erros     []ErrorEntry    `json:"erros"`
	Avaliador types.Avaliador `json:"avaliador"`
}

type AvaliadoresResponse struct {
	Avaliadores     map[int]AvaliadorValidationResult `json:"avaliadores"`
	Duplicates      [][]int                           `json:"duplicates"`
	DuplicateFields []string                          `json:"duplicateFields"`
}

type UsuariosResponse struct {
	Usuarios        map[int]ValidationResult `json:"usuarios"`
	Duplicates      [][]int                  `json:"duplicates"`
	DuplicateFields []string                 `json:"duplicateFields"`
}

// SuggestMapping lê a aba de candidatos e sugere um mapeamento inicial com
// base na similaridade entre os nomes das colunas e os campos esperados.
func SuggestMapping(data []byte, quantidadeOpcoes int) ([]types.MappingItem, error) {
	return SuggestMappingForSheet(data, 0, GetUsuarioFields(quantidadeOpcoes))
}

// SuggestMappingAvaliador faz a sugestão de mapeamento da aba de avaliadores
// usando a similaridade com os campos do tipo Avaliador.
func SuggestMappingAvaliador(data []byte) ([]types.MappingItem, error) {
	return SuggestMappingForSheet(data, 1, GetAvaliadorFields())
}

// SuggestMappingRestricao sugere o mapeamento da aba de restrições usando a
// similaridade entre cabeçalhos e campos do tipo Restricao.
func SuggestMappingRestricao(data []byte) ([]types.MappingItem, error) {
	return SuggestMappingForSheet(data, 2, GetRestricaoFields())
}

// suggestMappingForSheet concentra a lógica genérica de sugestão de mapping
// para qualquer aba a partir do índice da planilha e da lista de variáveis.
func SuggestMappingForSheet(data []byte, sheetIndex int, variables []string) ([]types.MappingItem, error) {
	rows, err := GetRowsFromSheet(data, sheetIndex)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}

	header := rows[0]
	return suggestMappingByName(header, variables), nil
}

// BuildUsuariosWithMapping percorre a aba de candidatos, aplica o mapping
// definido na interface e devolve os candidatos já validados e agrupados.
func BuildUsuariosWithMapping(data []byte, nOpcoes int, mappingItems []types.MappingItem) (UsuariosResponse, error) {
	if err := validateMappingItems(mappingItems); err != nil {
		return UsuariosResponse{}, err
	}
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

	header := rows[0]
	for _, row := range rows[1:] {
		u, err := buildCandidateFromRow(row, header, nOpcoes, mappingItems)
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
// ignora linhas vazias e retorna a resposta com validação e duplicados.
func BuildAvaliadoresWithMapping(data []byte, mappingItems []types.MappingItem) (AvaliadoresResponse, error) {
	if err := validateMappingItems(mappingItems); err != nil {
		return AvaliadoresResponse{}, err
	}

	if data == nil {
		return AvaliadoresResponse{}, fmt.Errorf("dados do Excel ainda não carregados")
	}

	// abre planilha a partir do []byte salvo em a.excelData
	reader := bytes.NewReader(data)
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return AvaliadoresResponse{}, fmt.Errorf("erro abrindo excel: %w", err)
	}
	defer file.Close()

	// 2ª aba (índice 1) onde estão os avaliadores
	sheet := file.GetSheetName(1)
	if sheet == "" {
		return AvaliadoresResponse{}, fmt.Errorf("arquivo não possui uma segunda aba com avaliadores")
	}

	rows, err := file.GetRows(sheet)
	if err != nil {
		return AvaliadoresResponse{}, fmt.Errorf("erro lendo aba de avaliadores: %w", err)
	}
	if len(rows) < 2 {
		return AvaliadoresResponse{}, fmt.Errorf("aba de avaliadores não contém dados além do cabeçalho")
	}

	var avaliadores []types.Avaliador

	// percorre linhas (ignorando cabeçalho)
	header := rows[0]
	for _, row := range rows[1:] {
		av, err := buildStructFromRowWithExtras[types.Avaliador](row, header, mappingItems)
		if err != nil {
			return AvaliadoresResponse{}, err
		}

		// ignora linhas totalmente vazias
		if isStructZeroValue(av) {
			continue
		}
		avaliadores = append(avaliadores, av)
	}

	avaliadoresLimpo, duplicatedIndices := processAvaliadores(avaliadores)

	return AvaliadoresResponse{
		Avaliadores:     avaliadoresLimpo,
		Duplicates:      duplicatedIndices,
		DuplicateFields: types.AvaliadorDuplicateFieldNames(),
	}, nil
}

// BuildRestricoesWithMapping converte a aba de restrições em structs prontos
// para uso posterior no salvamento e na etapa de alocação.
func BuildRestricoesWithMapping(data []byte, mappingItems []types.MappingItem) ([]types.Restricao, error) {
	if err := validateMappingItems(mappingItems); err != nil {
		return nil, err
	}
	rows, err := GetRowsFromSheet(data, 2)
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
func GetRowsFromSheet(data []byte, sheetIndex int) ([][]string, error) {
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
func GetUsuarioFields(quantidadeOpcoes int) []string {
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
func GetAvaliadorFields() []string {
	var fields []string
	for _, tag := range types.JSONFieldNames(types.Avaliador{}) {
		if tag == "extras" || tag == "id" {
			continue
		}
		fields = append(fields, tag)
	}
	return fields
}

// getRestricaoFields devolve os nomes JSON usados no mapeamento de restrições.
func GetRestricaoFields() []string {
	return types.RestricaoFieldNames()
}

// SaveAvaliadoresFromMaps converte o payload editado no frontend em avaliadores e
// usa o fluxo padrão de persistência da aplicação.
func SaveAvaliadoresFromMaps(avaliadorMaps []map[string]interface{}) error {
	avaliadores := make([]types.Avaliador, 0, len(avaliadorMaps))
	for _, m := range avaliadorMaps {
		avaliador, err := decodeMapToStruct[types.Avaliador](m)
		if err != nil {
			return err
		}
		avaliadores = append(avaliadores, avaliador)
	}

	return Save(avaliadores)
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

	duplicateFields := types.CandidateDuplicateFieldNames()
	duplicatedIndices := findDuplicatesGeneric(len(resultados), duplicateFields, func(id int, field string) string {
		return candidateFieldValue(resultados[id].Usuario, field)
	})

	return resultados, duplicatedIndices
}

func processAvaliadores(data []types.Avaliador) (map[int]AvaliadorValidationResult, [][]int) {
	resultados := make(map[int]AvaliadorValidationResult)

	for idx, entrada := range data {
		var errs []ErrorEntry

		entrada.Nome = strings.TrimSpace(entrada.Nome)
		entrada.Email = strings.ToLower(strings.TrimSpace(entrada.Email))
		entrada.Sigla = strings.TrimSpace(entrada.Sigla)

		if entrada.Nome == "" {
			errs = append(errs, ErrorEntry{Field: 0, Msg: "nome é obrigatório"})
		}
		if entrada.Email == "" {
			errs = append(errs, ErrorEntry{Field: 1, Msg: "email é obrigatório"})
		}
		if entrada.Sigla == "" {
			errs = append(errs, ErrorEntry{Field: 2, Msg: "sigla é obrigatória"})
		}

		resultados[idx+1] = AvaliadorValidationResult{
			Erros:     errs,
			Avaliador: entrada,
		}
	}

	duplicateFields := types.AvaliadorDuplicateFieldNames()
	duplicatedIndices := findDuplicatesGeneric(len(resultados), duplicateFields, func(id int, field string) string {
		return avaliadorFieldValue(resultados[id].Avaliador, field)
	})

	return resultados, duplicatedIndices
}

func findDuplicatesGeneric(n int, duplicateFieldNames []string, getFieldValue func(int, string) string) [][]int {
	valueIndices := make(map[string][]int)
	for i := 1; i <= n; i++ {
		for _, fieldName := range duplicateFieldNames {
			rawValue := strings.TrimSpace(getFieldValue(i, fieldName))
			if rawValue == "" || strings.ToLower(rawValue) == "null" {
				continue
			}
			key := fieldName + ":" + rawValue
			valueIndices[key] = append(valueIndices[key], i)
		}
	}

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

	return duplicatedIndices
}

func avaliadorFieldValue(avaliador types.Avaliador, fieldName string) string {
	for _, field := range types.AvaliadorFields() {
		if field.JSONName != fieldName || field.Kind != reflect.String {
			continue
		}

		value := reflect.ValueOf(avaliador).Field(field.Index)
		if !value.IsValid() {
			return ""
		}
		return strings.TrimSpace(value.String())
	}

	return ""
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
	sort.Strings(horariosUnicos)
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
	case []types.Avaliador:
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
	case []types.Candidato, []types.Avaliador, []types.Restricao:
		fillDb(conn, data)
	default:
		log.Printf("Tipo de dado não suportado em fillDb: %T", data)
	}

	return nil
}

// buildCandidateFromRow aplica o mapping de uma linha do Excel a um candidato,
// incluindo o preenchimento das opções em posições ordenadas e o residual em extras.
func buildCandidateFromRow(row []string, header []string, nOpcoes int, mappingItems []types.MappingItem) (types.Candidato, error) {
	record := make(map[string]interface{})
	record["opcoes"] = make([]string, nOpcoes)
	extrasFromMapping := make(map[string]string)
	coreFields := make(map[string]bool)
	for _, tag := range types.JSONFieldNames(types.Candidato{}) {
		coreFields[tag] = true
	}

	for _, mItem := range mappingItems {
		if mItem.Indice < 0 || mItem.Indice >= len(row) {
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

		if mItem.Variavel == "extras" {
			columnName := headerNameAt(header, mItem.Indice)
			key := extraColumnKey(columnName, mItem.Indice, extrasFromMapping)
			extrasFromMapping[key] = cell
			continue
		}

		if coreFields[mItem.Variavel] {
			record[mItem.Variavel] = cell
		} else if mItem.Variavel != "" {
			extrasFromMapping[mItem.Variavel] = cell
		}
	}

	if len(extrasFromMapping) > 0 {
		record["extras"] = extrasFromMapping
	}

	return decodeMapToStruct[types.Candidato](record)
}

// buildStructFromRow monta qualquer struct baseada em tags JSON a partir de uma
// linha da planilha e do mapping configurado.
func buildStructFromRow[T any](row []string, mappingItems []types.MappingItem) (T, error) {
	var zero T
	if mappingIncludesExtras(mappingItems) && !typeSupportsExtrasField[T]() {
		return zero, fmt.Errorf("mapeamento para extras só é permitido para structs com campo \"extras\"")
	}

	record := make(map[string]interface{})
	for _, mapping := range mappingItems {
		if mapping.Indice < 0 || mapping.Indice >= len(row) || mapping.Variavel == "" {
			continue
		}
		record[mapping.Variavel] = strings.TrimSpace(row[mapping.Indice])
	}

	return decodeMapToStruct[T](record)
}

// buildStructFromRowWithExtras monta uma struct genérica e anexa colunas não
// consumidas pelo mapping ao campo extras, quando ele existir no tipo alvo.
func buildStructFromRowWithExtras[T any](row []string, header []string, mappingItems []types.MappingItem) (T, error) {
	record := make(map[string]interface{})
	extras := make(map[string]string)

	coreFields := make(map[string]bool)
	for _, tag := range types.JSONFieldNames(*new(T)) {
		coreFields[tag] = true
	}

	for _, mapping := range mappingItems {
		if mapping.Indice < 0 || mapping.Indice >= len(row) {
			continue
		}
		value := strings.TrimSpace(row[mapping.Indice])
		if mapping.Variavel == "extras" {
			columnName := headerNameAt(header, mapping.Indice)
			key := extraColumnKey(columnName, mapping.Indice, extras)
			extras[key] = value
			continue
		}
		if coreFields[mapping.Variavel] {
			record[mapping.Variavel] = value
		} else if mapping.Variavel != "" {
			extras[mapping.Variavel] = value
		}
	}

	if len(extras) > 0 {
		record["extras"] = extras
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

func headerNameAt(header []string, index int) string {
	if index >= 0 && index < len(header) {
		return header[index]
	}
	return ""
}

func mappingIncludesExtras(mappingItems []types.MappingItem) bool {
	for _, mapping := range mappingItems {
		if mapping.Variavel == "extras" {
			return true
		}
	}
	return false
}

func typeSupportsExtrasField[T any]() bool {
	for _, field := range types.JSONFieldNames(*new(T)) {
		if field == "extras" {
			return true
		}
	}
	return false
}

func validateMappingItems(mappingItems []types.MappingItem) error {
	// Permitimos índices negativos, eles serão ignorados durante o processamento
	return nil
}

func collectUnusedColumnExtras(row []string, header []string, mappingItems []types.MappingItem) map[string]string {
	usedColumns := make(map[int]bool, len(mappingItems))
	for _, item := range mappingItems {
		if item.Indice >= 0 && item.Indice < len(header) {
			usedColumns[item.Indice] = true
		}
	}

	extras := make(map[string]string)
	for index, columnName := range header {
		if usedColumns[index] || index >= len(row) {
			continue
		}

		value := strings.TrimSpace(row[index])
		if value == "" {
			continue
		}

		key := extraColumnKey(columnName, index, extras)
		extras[key] = value
	}

	if len(extras) == 0 {
		return nil
	}
	return extras
}

func extraColumnKey(columnName string, index int, existing map[string]string) string {
	base := strings.TrimSpace(columnName)
	if base == "" {
		base = fmt.Sprintf("coluna_%d", index)
	}
	if _, exists := existing[base]; !exists {
		return base
	}
	return fmt.Sprintf("%s_%d", base, index)
}
