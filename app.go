package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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

type AvaliadorInfo struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Sigla string `json:"sigla"`
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

func (a *App) SuggestMappingAvaliador() ([]MappingItem, error) {
	readerData := bytes.NewReader(a.excelData)
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

	// Lista de possíveis variáveis para avaliador (ajuste conforme necessário)
	variaveisAvaliador := []string{"Avaliador", "Email", "Sigla"}
	mappingList := make([]string, len(header))
	variaveisAvaliadorIdx := 0
	for i, colName := range header {
		var avaliadorVar string
		if variaveisAvaliadorIdx < len(variaveisAvaliador) {
			avaliadorVar = variaveisAvaliador[variaveisAvaliadorIdx]
			variaveisAvaliadorIdx++
		} else {
			avaliadorVar = "none"
		}
		mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", colName, i, avaliadorVar)
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

func (a *App) BuildAvaliadoresWithMapping(mappingItems []MappingItem) ([]AvaliadorInfo, error) {
	if a.excelData == nil {
		return nil, fmt.Errorf("dados do Excel ainda não carregados")
	}

	// abre planilha a partir do []byte salvo em a.excelData
	reader := bytes.NewReader(a.excelData)
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

	var avaliadores []AvaliadorInfo

	// percorre linhas (ignorando cabeçalho)
	for _, row := range rows[1:] {
		av := AvaliadorInfo{}
		for _, m := range mappingItems {
			if m.Indice >= len(row) {
				continue // coluna vazia nesta linha
			}
			val := strings.TrimSpace(row[m.Indice])

			switch strings.ToLower(m.Variavel) {
			case "avaliador":
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

	return avaliadores, nil
}



func (a *App) Save(data interface{}) {
	conn, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()


	switch v := data.(type) {
	case []Usuario:
		fillDb(conn, data)
	case []AvaliadorInfo:
		fillDb(conn, data)
	default:
		fmt.Println("Tipo de dado não suportado em fillDb", v)
	}
}


func main() {
	path := flag.String("file", "", "caminho para o arquivo .xlsx")
	flag.Parse()
	if *path == "" {
		fmt.Println("Uso: go run main.go -file seu_arquivo.xlsx")
		os.Exit(1)
	}

	data, err := os.ReadFile(*path)
	if err != nil {
		fmt.Println("Erro ao ler o arquivo:", err)
		os.Exit(1)
	}


	app := NewApp()
	mapping, err := app.SuggestMapping(data, 5)
	if err != nil {
		fmt.Println("Erro ao sugerir mapeamento:", err)
		os.Exit(1)
	}

	mappingAvaliador, err := app.SuggestMappingAvaliador()

	fmt.Println("\n")
	fmt.Println("mapping candidatos : ", mapping)
	fmt.Println("\n")
	fmt.Println("mapping avaliadores : ", mappingAvaliador)


	usuarios, err := app.BuildUsuariosWithMapping(mapping)
	if err != nil {
		fmt.Println("Erro ao ler o arquivo:", err)
		os.Exit(1)
	}
	usuarios_filtrados := FilterUniqueUsers(usuarios)

	avaliadores, err := app.BuildAvaliadoresWithMapping(mappingAvaliador)

	fmt.Println("\n")
	fmt.Println("usuarios: ", usuarios)
	fmt.Println("\n\n\n")
	fmt.Println("avaliadores: ", avaliadores)

	app.Save(usuarios_filtrados)

	app.Save(avaliadores)

	// Alocacao
	// Alocar(conn)


	// out1, _ := json.MarshalIndent(mapping, "", " ")
	// out, _ := json.MarshalIndent(usuarios_filtrados, "", "  ")
	// out2, _ := json.MarshalIndent(duplicatedIndices, "", "  ")
	// fmt.Println("usuarios : ", usuarios_filtrados)
	fmt.Println("\n")
	// fmt.Println(string(out2))

}
