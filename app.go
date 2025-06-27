package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
)

// App struct
type App struct {
	ctx       context.Context
	excelData []byte
	nOpcoes   int
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

func (a *App) SuggestMapping(data []byte, quantidade_opcoes int) ([]MappingItem, error) {
	a.excelData = data
	a.nOpcoes = quantidade_opcoes
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
	mapping_json, err := ProcessMapping(mappingList)
	if err != nil {
		return nil, err
	}
	return mapping_json, nil
}

type MappingItem struct {
	NomeColuna string `json:"nomeColuna"`
	Indice     int    `json:"indice"`
	Variavel   string `json:"variavel"`
}

// ProcessMapping converte cada string JSON "[[nomeColuna,indice],variavel]"
// em um Mapping. Retorna erro se algum item não for JSON válido.
func ProcessMapping(items []string) ([]MappingItem, error) {
	var result []MappingItem

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
		nomeColuna, ok1 := info[0].(string)
		indiceF, ok2 := info[1].(float64)
		variavel, ok3 := arr[1].(string)
		if !ok1 || !ok2 || !ok3 {
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

type UsuariosResponse struct {
	Usuarios   map[int]ValidationResult `json:"usuarios"`
	Duplicates [][]int                  `json:"duplicates"`
}

func (a *App) BuildUsuariosWithMapping(mappingItems []MappingItem) (UsuariosResponse, error) {
	// Abre o arquivo Excel a partir dos dados em []byte
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
func (a *App) Save(usuarios_tratados []Usuario) {
	conn, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fillDb(conn, usuarios_tratados)
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
// 	fmt.Println("\n")
// 	fmt.Println("mapping : ", mapping)
// 	fmt.Println("\n")
// 	usuarios, err := app.BuildUsuariosWithMapping(mapping)
// 	if err != nil {
// 		fmt.Println("Erro ao ler o arquivo:", err)
// 		os.Exit(1)
// 	}
// 	usuarios_filtrados := FilterUniqueUsers(usuarios)
// 	app.Save(usuarios_filtrados)

// 	// out1, _ := json.MarshalIndent(mapping, "", " ")
// 	// out, _ := json.MarshalIndent(usuarios_filtrados, "", "  ")
// 	// out2, _ := json.MarshalIndent(duplicatedIndices, "", "  ")
// 	fmt.Println("usuarios : ", usuarios_filtrados)
// 	fmt.Println("\n")
// 	// fmt.Println(string(out2))

// }
