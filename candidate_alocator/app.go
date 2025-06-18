package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx
}

// domReady is called after front-end resources have been loaded
func (a App) domReady(ctx context.Context) {
	// Add your action here
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	// Perform your teardown here
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

type Usuario struct {
	Timestamp    string   `json:"timestamp"`
	Nome         string   `json:"nome"`
	CPF          string   `json:"cpf"`
	Numero       string   `json:"numero"`
	Semestre     string   `json:"semestre"`
	Curso        string   `json:"curso"`
	EmailInsper  string   `json:"email_insper"`
	EmailPessoal string   `json:"email_pessoal"`
	Opcoes       []string `json:"opcoes"`
}

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

func (a *App) SuggestMapping(data []byte, quantidade_opcoes int) ([]string, error) {
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

	// Lista de possíveis variáveis da struct Usuario (em minúsculo)
	variaveisUsuario := getUsuarioFields(quantidade_opcoes)

	// Alocação aleatória das variáveis para cada coluna
	mappingList := make([]string, len(header))
	variaveisUsuarioIdx := 0
	for i, colName := range header {
		var usuarioVar string
		if variaveisUsuarioIdx < len(variaveisUsuario) {
			usuarioVar = variaveisUsuario[variaveisUsuarioIdx]
			variaveisUsuarioIdx++
		} else {
			usuarioVar = "none"
		}
		// Formato: [[nome na row, indice da row], variavel do usuario escolhida]
		mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", colName, i, usuarioVar)
	}

	return mappingList, nil
}

// BuildUsuariosWithMapping monta o slice de usuários utilizando a alocação informada.
// O parâmetro mapping é uma lista de strings no formato:
//
//	[[ "Nome da Coluna", índice ], "campo_do_usuario"]
//
// Exemplo:
//
//	[["TimeStamp", 0], "timestamp"]
//	[["Primeira Opção", 8], "opcao 1"]
func (a *App) BuildUsuariosWithMapping(data []byte, mapping []string, nOpcoes int) ([]Usuario, error) {
	// Abre o arquivo Excel a partir dos dados em []byte
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
	if len(rows) < 2 {
		return nil, fmt.Errorf("arquivo sem dados além do header")
	}

	// Define uma struct auxiliar para armazenar o mapeamento
	type MappingItem struct {
		HeaderName string
		ColIndex   int
		UserField  string
	}

	// Converte cada string do mapping para MappingItem usando unmarshal JSON
	var mappingItems []MappingItem
	for _, mStr := range mapping {
		// O mapping deve ter o formato: [[ "Header", índice ], "userField"]
		var item []interface{}
		if err := json.Unmarshal([]byte(mStr), &item); err != nil {
			return nil, fmt.Errorf("erro ao parsear mapping: %v", err)
		}
		if len(item) != 2 {
			return nil, fmt.Errorf("formato de mapping inválido")
		}
		// O primeiro elemento deve ser um array: ["Header", índice]
		headerPart, ok := item[0].([]interface{})
		if !ok || len(headerPart) != 2 {
			return nil, fmt.Errorf("formato de header inválido no mapping")
		}
		headerName, ok := headerPart[0].(string)
		if !ok {
			return nil, fmt.Errorf("header name inválido")
		}
		colIndexFloat, ok := headerPart[1].(float64)
		if !ok {
			return nil, fmt.Errorf("col index inválido")
		}
		colIndex := int(colIndexFloat)
		// O segundo elemento é a variável do usuário
		userField, ok := item[1].(string)
		if !ok {
			return nil, fmt.Errorf("user field inválido")
		}
		mappingItems = append(mappingItems, MappingItem{
			HeaderName: headerName,
			ColIndex:   colIndex,
			UserField:  userField,
		})
	}

	var users []Usuario
	// Processa as linhas do Excel, ignorando o header (linha 0)
	for _, row := range rows[1:] {
		u := Usuario{
			Opcoes: make([]string, nOpcoes),
		}
		// Para cada mapping, pega o conteúdo da coluna correspondente e atribui
		for _, mItem := range mappingItems {
			if mItem.ColIndex >= len(row) {
				continue
			}
			cell := row[mItem.ColIndex]
			switch mItem.UserField {
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
				if strings.HasPrefix(mItem.UserField, "opcao") {
					parts := strings.Split(mItem.UserField, " ")
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

	return users, nil
}

// func main() {
// 	path := flag.String("file", "", "caminho para o arquivo .xlsx")
// 	flag.Parse()
// 	if *path == "" {
// 		fmt.Println("Uso: go run main.go -file seu_arquivo.xlsx")
// 		os.Exit(1)
// 	}

// 	data, err := os.ReadFile(*path)
// 	if err != nil {
// 		fmt.Println("Erro ao ler o arquivo:", err)
// 		os.Exit(1)
// 	}

// 	app := NewApp()
// 	mapping, err := app.SuggestMapping(data, 5)
// 	if err != nil {
// 		fmt.Println("Erro ao sugerir mapeamento:", err)
// 		os.Exit(1)
// 	}
// 	usuarios, err := app.BuildUsuariosWithMapping(data, mapping, 5)
// 	if err != nil {
// 		fmt.Println("Erro ao montar os usuários:", err)
// 		os.Exit(1)
// 	}
// 	out1, _ := json.MarshalIndent(mapping, "", " ")
// 	out, _ := json.MarshalIndent(usuarios, "", "  ")
// 	fmt.Println(string(out1))
// 	fmt.Println("\n")
// 	fmt.Println(string(out))
// }
