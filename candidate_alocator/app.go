package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
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

type colInfo struct {
	nome          string
	optIndexOpcao int
}

// findDuplicateMappingIndex retorna o índice de um mapeamento duplicado em mappings,
// ignorando o índice indicado por ignoreColumnIndex. Para "opcao", compara também o índice da opção.
func findDuplicateMappingIndex(mappingTipo string, optionNumber int, ignoreColumnIndex int, mappings []colInfo) int {
	for idx, mapping := range mappings {
		if idx == ignoreColumnIndex {
			continue
		}
		if mappingTipo == "opcao" {
			if mapping.nome == "opcao" && mapping.optIndexOpcao == optionNumber {
				return idx
			}
		} else {
			if mapping.nome == mappingTipo {
				return idx
			}
		}
	}
	return -1
}

// promptUserForColumn exibe o prompt para configurar o mapeamento de uma coluna e retorna o tipo escolhido e, se for o caso, o número da opção.
func promptUserForColumn(reader *bufio.Reader, colName string) (string, int) {
	fmt.Printf("Coluna %q : ", colName)
	input, _ := reader.ReadString('\n')
	chosen := strings.ToLower(strings.TrimSpace(input))
	optIdx := 0
	if chosen == "opcao" {
		fmt.Print("Digite o número da opção: ")
		optInput, _ := reader.ReadString('\n')
		optIdx, _ = strconv.Atoi(strings.TrimSpace(optInput))
	}
	return chosen, optIdx
}

// askOverrideOrReselect pergunta se o usuário deseja sobrepor o mapeamento duplicado ou reescolher para a coluna atual.
func askOverrideOrReselect(reader *bufio.Reader, existingMapping colInfo, currentCol string) string {
	fmt.Printf("Ja existe uma coluna marcada como %q deseja sobrepor com a coluna %q", existingMapping.nome, currentCol)
	if existingMapping.nome == "opcao" {
		fmt.Printf(" com opção %d", existingMapping.optIndexOpcao)
	}
	fmt.Println(". Deseja sobrepor? (s para sobrepor, qualquer outra tecla para reescolher)")
	choice, _ := reader.ReadString('\n')
	choice = strings.ToLower(strings.TrimSpace(choice))
	if choice == "s" {
		return "override"
	}
	return "reselect"
}

// ParseExcelInteractive lê o arquivo Excel e solicita interativamente o mapeamento de cada coluna.
func (a *App) ParseExcelInteractive(data []byte, nOpcoes int) ([]Usuario, error) {
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
		return nil, fmt.Errorf("arquivo sem dados")
	}

	reader := bufio.NewReader(os.Stdin)

	header := rows[0]
	lista_colunas := make([]colInfo, len(header))
	validTypes := map[string]bool{
		"timestamp":     true,
		"nome":          true,
		"cpf":           true,
		"numero":        true,
		"semestre":      true,
		"curso":         true,
		"email_insper":  true,
		"email_pessoal": true,
		"opcao":         true,
		"none":          true,
	}

	// Para cada coluna, solicita o mapeamento com verificação de duplicidade.
	fmt.Println("Escolha com os seguintes tipos: timestamp, nome, cpf, numero, semestre, curso, email_insper, email_pessoal, opcao ou none")
	for i, colName := range header {
		for {

			chosenType, chosenOptIndex := promptUserForColumn(reader, colName)
			if !validTypes[chosenType] {
				fmt.Println("Inválido. Escolha: timestamp, nome, cpf, numero, semestre, curso, email_insper, email_pessoal, opcao ou none")
				continue
			}
			dupIdx := findDuplicateMappingIndex(chosenType, chosenOptIndex, i, lista_colunas)

			if dupIdx == -1 {
				// se não for sai
				lista_colunas[i].nome = chosenType
				lista_colunas[i].optIndexOpcao = chosenOptIndex
				break
			}
			decision := askOverrideOrReselect(reader, lista_colunas[dupIdx], colName)
			if decision == "override" {

				// Atribui o mapeamento atual.
				lista_colunas[i].nome = chosenType
				lista_colunas[i].optIndexOpcao = chosenOptIndex

				colName = header[dupIdx]
				i = dupIdx
			} else {
				// Reselect: repete o loop para a coluna atual.
				continue
			}
		}
	}

	// Monta slice de usuários
	var users []Usuario
	for _, row := range rows[1:] {
		u := Usuario{
			Opcoes: make([]string, nOpcoes),
		}
		for idx, cell := range row {
			if idx >= len(lista_colunas) {
				continue
			}
			ci := lista_colunas[idx]
			switch ci.nome {
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
			case "opcao":
				if ci.optIndexOpcao-1 < len(u.Opcoes) && ci.optIndexOpcao-1 >= 0 {
					u.Opcoes[ci.optIndexOpcao-1] = cell
				}
			}
		}
		users = append(users, u)
	}
	return users, nil
}
