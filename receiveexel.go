package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"regexp"

	"github.com/xuri/excelize/v2"
)

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
func ParseExcelInteractive(path string) ([]Usuario, error) {
	file, err := excelize.OpenFile(path)
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
	fmt.Print("Quantas opções de alocação? ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	nOpcoes, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return nil, fmt.Errorf("número de opções inválido")
	}

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


func processData(data []Usuario) []Usuario {
	cpfVistos := map[string]bool{}
	emailPessoalVistos := map[string]bool{}
	emailInsperVistos := map[string]bool{}
	dataLimpa := []Usuario{}
	erros := []string{}
	var why string

	for _, entrada := range data {
		entrada.CPF = strings.TrimSpace(entrada.CPF)
		entrada.EmailInsper = strings.ToLower(strings.TrimSpace(entrada.EmailInsper))
		entrada.EmailPessoal = strings.ToLower(strings.TrimSpace(entrada.EmailPessoal))
		entrada.Numero = strings.TrimSpace(entrada.Numero)

		// -=-=-=-=-=-=-=- Verifica duplicatas -=-=-=-=-=-=-=-=
		if cpfVistos[entrada.CPF] || emailInsperVistos[entrada.EmailInsper] || emailPessoalVistos[entrada.EmailPessoal] {
			var antiga *Usuario
			for _, d := range dataLimpa {
				if d.CPF == entrada.CPF || d.EmailInsper == entrada.EmailInsper || d.EmailPessoal == entrada.EmailPessoal {
					antiga = &d
					break
				}
			}

			// Remove duplicata antiga
			newDataLimpa := []Usuario{}
			for _, d := range dataLimpa {
				if d.CPF != entrada.CPF && d.EmailInsper != entrada.EmailInsper && d.EmailPessoal != entrada.EmailPessoal {
					newDataLimpa = append(newDataLimpa, d)
				}
			}
			dataLimpa = newDataLimpa

			if entrada.CPF == antiga.CPF {
				why = "CPF"
			} else if entrada.EmailInsper == antiga.EmailInsper {
				why = "Email Insper"
			} else if entrada.EmailPessoal == antiga.EmailPessoal {
				why = "Email Pessoal"
			}
			erros = append(erros, fmt.Sprintf("- Duplicate %+v \n\t- Usuario removida: %+v\n\t- Usuario mantida: %+v", why, entrada, antiga))
			continue
		}

		// -=-=-=-=-=-=-=- Validações -=-=-=-=-=-=-=-=
		if !regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`).MatchString(entrada.EmailPessoal) {
			erros = append(erros, fmt.Sprintf("Email pessoal inválido: %+v", entrada))
		}
		if !regexp.MustCompile(`^[^@]+@al\.insper\.edu\.br$`).MatchString(entrada.EmailInsper) {
			erros = append(erros, fmt.Sprintf("Email Insper inválido: %+v", entrada))
		}
		if !regexp.MustCompile(`^\d{11}$`).MatchString(entrada.CPF) {
			erros = append(erros, fmt.Sprintf("CPF inválido: %+v", entrada))
		}
		if !regexp.MustCompile(`^\d{9}$`).MatchString(entrada.Numero) {
			erros = append(erros, fmt.Sprintf("Número inválido: %+v", entrada))
		}
		if !regexp.MustCompile(`^[1-8]$`).MatchString(entrada.Semestre) {
			erros = append(erros, fmt.Sprintf("Semestre inválido: %+v", entrada))
		}

		// -=-=-=-=-=-=-=- Marca como visto e salva -=-=-=-=-=-=-=-=
		cpfVistos[entrada.CPF] = true
		emailInsperVistos[entrada.EmailInsper] = true
		emailPessoalVistos[entrada.EmailPessoal] = true
		dataLimpa = append(dataLimpa, entrada)
	}

	if len(erros) > 0 {
		fmt.Println(strings.Repeat("----", 25))
		fmt.Println("Erros encontrados:")
		for _, err := range erros {
			fmt.Println(err)
			fmt.Println(strings.Repeat("----", 25))
		}
		return nil
	}

	return dataLimpa
}




func main() {
	path := flag.String("file", "", "caminho para o arquivo .xlsx")
	flag.Parse()
	if *path == "" {
		fmt.Println("Uso: go run main.go -file seu_arquivo.xlsx")
		os.Exit(1)
	}

	usuarios, err := ParseExcelInteractive(*path)

	fmt.Println("Processando dados...")

	usuarios = processData(usuarios)

	fmt.Println("Dados processados com sucesso!")
	fmt.Println(strings.Repeat("----", 25))

	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(usuarios, "", "  ")

	fmt.Println(string(out))
}
