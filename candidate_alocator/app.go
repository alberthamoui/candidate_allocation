package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	dbpkg "candidate_alocator/db"

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

func fillDb(db *sql.DB, data []Usuario) {
	// HORARIOS
	idHorarios := map[string]int64{}
	horarios := getHorarios(data)
	for _, horario := range horarios {
		base := strings.Split(horario, " - ")
		hora := base[0]
		date := base[1]
		idHorario, _ := dbpkg.AddHorario(db, date, hora, "None")
		idHorarios[horario] = idHorario
	}
	fmt.Println("Horários inseridos no banco de dados.")

	// CANDIDATOS & DISPONIBILIDADES
	for _, usuario := range data {
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

}

func (a *App) SuggestMapping(data []byte, quantidade_opcoes int) ([]string, error) {
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

func (a *App) BuildUsuariosWithMapping(mappingJSON string) ([]Usuario, error) {
	// Abre o arquivo Excel a partir dos dados em []byte
	data := a.excelData
	nOpcoes := a.nOpcoes
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler excel : %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("arquivo sem dados além do header")
	}

	// Define uma struct auxiliar para armazenar o mapeamento
	type MappingItem struct {
		NomeColuna string `json:"nomeColuna"`
		Indice     int    `json:"indice"`
		Variavel   string `json:"variavel"`
	}

	// Converte cada string do mapping para MappingItem usando unmarshal JSON
	var mappingItems []MappingItem
	if err := json.Unmarshal([]byte(mappingJSON), &mappingItems); err != nil {
		return nil, fmt.Errorf("erro ao parsear mapping JSON: %w", err)
	}

	var users []Usuario
	// Processa as linhas do Excel, ignorando o header (linha 0)
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
	users_limpo := processData(users)
	conn, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("Salvando dados no banco de dados...")
	fillDb(conn, users_limpo)
	fmt.Println("Dados salvos com sucesso!")
	return users_limpo, nil
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
// 	usuarios, err := app.BuildUsuariosWithMapping(mapping, 5)
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
