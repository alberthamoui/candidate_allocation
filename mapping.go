package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// SuggestMapping lê o cabeçalho da primeira aba do Excel e sugere
// um mapeamento automático entre colunas e campos de Usuario.
func (a *App) SuggestMapping(data []byte, quantidade_opcoes int, emailDomain string) ([]MappingItem, error) {
	a.excelData = data
	a.nOpcoes = quantidade_opcoes
	a.emailDomain = emailDomain
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

	variaveisUsuario := getUsuarioFields(quantidade_opcoes)
	fmt.Println("variaveis usuario: ", variaveisUsuario)

	mappingList := make([]string, len(variaveisUsuario))
	for i, usuarioVar := range variaveisUsuario {
		if i < len(header) {
			mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", header[i], i, usuarioVar)
		} else {
			mappingList[i] = fmt.Sprintf("[[null, %d], %q]", i, usuarioVar)
		}
	}
	fmt.Println(mappingList)
	return ProcessMapping(mappingList)
}

// SuggestMappingAvaliador lê o cabeçalho da segunda aba do Excel e sugere
// um mapeamento automático para campos de AvaliadorInfo.
func (a *App) SuggestMappingAvaliador() ([]MappingItem, error) {
	readerData := bytes.NewReader(a.excelData)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(1)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	header := rows[0]

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

// SuggestMappingRestricao lê o cabeçalho da terceira aba do Excel e sugere
// um mapeamento automático para campos de Restricao.
func (a *App) SuggestMappingRestricao() ([]MappingItem, error) {
	readerData := bytes.NewReader(a.excelData)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(2)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	header := rows[0]

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

// ProcessMapping converte cada string JSON "[[nomeColuna,indice],variavel]"
// em um MappingItem. Retorna erro se algum item não for JSON válido.
func ProcessMapping(items []string) ([]MappingItem, error) {
	var result []MappingItem

	for _, item := range items {
		var arr []interface{}
		if err := json.Unmarshal([]byte(item), &arr); err != nil {
			return nil, fmt.Errorf("invalid JSON '%s': %w", item, err)
		}
		if len(arr) != 2 {
			continue
		}

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

		result = append(result, MappingItem{
			NomeColuna: nomeColuna,
			Indice:     int(indiceF),
			Variavel:   variavel,
		})
	}

	return result, nil
}

// BuildUsuariosWithMapping lê a primeira aba do Excel aplicando o mapeamento
// fornecido e retorna usuários validados e índices de duplicatas.
func (a *App) BuildUsuariosWithMapping(mappingItems []MappingItem) (UsuariosResponse, error) {
	data := a.excelData
	nOpcoes := a.nOpcoes
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

	var users []Usuario

	for _, row := range rows[1:] {
		u := Usuario{
			Opcoes: make([]string, nOpcoes),
		}
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
	users_limpo, duplicatedIndices := processData(users, a.emailDomain)

	return UsuariosResponse{Usuarios: users_limpo, Duplicates: duplicatedIndices}, nil
}

// BuildAvaliadoresWithMapping lê a segunda aba do Excel aplicando o mapeamento fornecido.
func (a *App) BuildAvaliadoresWithMapping(mappingItems []MappingItem) ([]AvaliadorInfo, error) {
	if a.excelData == nil {
		return nil, fmt.Errorf("dados do Excel ainda não carregados")
	}

	reader := bytes.NewReader(a.excelData)
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("erro abrindo excel: %w", err)
	}
	defer file.Close()

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

	var avaliadores []AvaliadorInfo

	for _, row := range rows[1:] {
		av := AvaliadorInfo{}
		for _, m := range mappingItems {
			if m.Indice >= len(row) {
				continue
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

		if av.Nome == "" && av.Email == "" && av.Sigla == "" {
			continue
		}
		avaliadores = append(avaliadores, av)
	}

	return avaliadores, nil
}

// BuildRestricoesWithMapping lê a terceira aba do Excel aplicando o mapeamento fornecido.
func (a *App) BuildRestricoesWithMapping(mappingItems []MappingItem) ([]Restricao, error) {
	readerData := bytes.NewReader(a.excelData)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(2)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler excel: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("arquivo sem dados além do header")
	}

	var restricoes []Restricao
	for _, row := range rows[1:] {
		r := Restricao{}
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
