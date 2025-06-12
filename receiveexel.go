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
	field    string // “timestamp”, “nome”, “cpf”, “numero”, “semestre”, “curso”, “email_insper”, “email_pessoal” ou “opcao”
	optIndex int    // de 1 até nOpcoes, apenas para “opcao”
}

func ParseExcelInteractive(path string) ([]Usuario, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("arquivo sem dados")
	}

	reader := bufio.NewReader(os.Stdin)
	// 1) Pergunta quantas opções
	fmt.Print("Quantas opções de alocação? ")
	in, _ := reader.ReadString('\n')
	nOpcoes, _ := strconv.Atoi(strings.TrimSpace(in))

	header := rows[0]
	colMap := make([]colInfo, len(header))

	// 2) Para cada coluna, pergunta o tipo (só valores válidos)
	fmt.Println("Agora, para cada coluna, escolha entre:")
	fmt.Println("  timestamp, nome, cpf, numero, semestre, curso, email_insper, email_pessoal, opcao ou none")
	for i, h := range header {
		var typ string
		for {
			fmt.Printf("Coluna %d (%q): ", i+1, h)
			in, _ := reader.ReadString('\n')
			typ = strings.ToLower(strings.TrimSpace(in))
			if typ == "timestamp" || typ == "nome" || typ == "cpf" ||
				typ == "numero" || typ == "semestre" || typ == "curso" ||
				typ == "email_insper" || typ == "email_pessoal" ||
				typ == "opcao" || typ == "none" {
				break
			}
			fmt.Println("Inválido. Escolha: timestamp, nome, cpf, numero, semestre, curso, email_insper, email_pessoal, opcao ou none")
		}

		ci := colInfo{field: typ}
		if typ == "opcao" {
			// pergunta índice (1..nOpcoes)
			for {
				fmt.Printf("  Qual o índice da opção (1..%d)? ", nOpcoes)
				in2, _ := reader.ReadString('\n')
				idx, err := strconv.Atoi(strings.TrimSpace(in2))
				if err == nil && idx >= 1 && idx <= nOpcoes {
					ci.optIndex = idx
					break
				}
			}
		}
		colMap[i] = ci
	}

	// 3) Monta slice de usuários
	var users []Usuario
	for _, row := range rows[1:] {
		u := Usuario{
			Opcoes: make([]string, nOpcoes),
		}
		for idx, cell := range row {
			ci := colMap[idx]
			switch ci.field {
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
				u.Opcoes[ci.optIndex-1] = cell
			}
		}
		users = append(users, u)
	}
	return users, nil
}

func main() {
	path := flag.String("file", "", "caminho para o arquivo .xlsx")
	flag.Parse()
	if *path == "" {
		fmt.Println("Uso: go run main.go -file seu_arquivo.xlsx")
		os.Exit(1)
	}

	usuarios, err := ParseExcelInteractive(*path)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(usuarios, "", "  ")
	fmt.Println(string(out))
}
