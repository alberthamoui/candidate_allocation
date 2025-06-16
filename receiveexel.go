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
	field    string // "timestamp", "nome", "cpf", "numero", "semestre", "curso", "email_insper", "email_pessoal", "opcao" ou "none"
	optIndex int    // de 1 até nOpcoes, apenas para "opcao"
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

	// Função interna para validar mapeamento duplicado (exceto coluna "opcao" com índice diferente)
	findDuplicate := func(t string, idx int, exclude int) int {
		for j, ci := range colMap {
			if j == exclude {
				continue
			}
			if t == "opcao" {
				if ci.field == "opcao" && ci.optIndex == idx {
					return j
				}
			} else {
				if ci.field == t {
					return j
				}
			}
		}
		return -1
	}

	// 2) Para cada coluna, pergunta o tipo e trata duplicidade
	fmt.Println("Agora, para cada coluna, escolha entre:")
	fmt.Println("  timestamp, nome, cpf, numero, semestre, curso, email_insper, email_pessoal, opcao ou none")
	for i, colHeader := range header {
		var chosenType string
		var chosenOptIndex int

		for {
			fmt.Printf("Coluna %q: ", colHeader)
			in, _ = reader.ReadString('\n')
			typ := strings.ToLower(strings.TrimSpace(in))
			if !validTypes[typ] {
				fmt.Println("Inválido. Escolha: timestamp, nome, cpf, numero, semestre, curso, email_insper, email_pessoal, opcao ou none")
				continue
			}

			if typ == "opcao" {
				var idx int
				for {
					fmt.Printf("  Qual o índice da opção (1..%d)? ", nOpcoes)
					in, _ = reader.ReadString('\n')
					tmp, err := strconv.Atoi(strings.TrimSpace(in))
					if err != nil || tmp < 1 || tmp > nOpcoes {
						continue
					}
					idx = tmp
					dup := findDuplicate("opcao", idx, -1)
					if dup != -1 {
						fmt.Printf("Já existe mapeamento para 'opcao' com índice %d na coluna %q. Deseja substituir? (s/n): ", idx, header[dup])
						resp, _ := reader.ReadString('\n')
						resp = strings.ToLower(strings.TrimSpace(resp))
						if resp == "s" {
							// Se o mapeamento antigo for também "opcao", permite remapear para outro índice
							for {
								fmt.Printf("Para qual campo deseja remapear a coluna %q? ", header[dup])
								newField, _ := reader.ReadString('\n')
								newField = strings.ToLower(strings.TrimSpace(newField))
								// Se o usuário desejar manter como opcao, permita escolher um novo índice
								if newField == "opcao" {
									var newIdx int
									for {
										fmt.Printf("  Qual o novo índice da opção para a coluna %q? ", header[dup])
										in, _ := reader.ReadString('\n')
										tmp, err := strconv.Atoi(strings.TrimSpace(in))
										if err != nil || tmp < 1 || tmp > nOpcoes {
											continue
										}
										newIdx = tmp
										if findDuplicate("opcao", newIdx, dup) != -1 {
											fmt.Println("Esse índice já está mapeado, tente outro.")
											continue
										}
										break
									}
									colMap[dup] = colInfo{field: "opcao", optIndex: newIdx}
									break
								} else if !validTypes[newField] || newField == "opcao" {
									// Se não for "opcao" (caso inválido, repetir)
									fmt.Println("Inválido. Digite um campo válido (exceto opcao) ou digite 'opcao' para remapear com novo índice.")
									continue
								} else {
									if findDuplicate(newField, 0, dup) != -1 {
										fmt.Println("Esse campo já está mapeado em outra coluna, tente outro.")
										continue
									}
									colMap[dup] = colInfo{field: newField}
									break
								}
							}
							chosenType = "opcao"
							chosenOptIndex = idx
							break
						} else {
							fmt.Println("Por favor, escolha outro índice para esta coluna.")
							continue
						}
					} else {
						chosenType = "opcao"
						chosenOptIndex = idx
						break
					}
				}
			} else if typ != "none" {
				dup := findDuplicate(typ, 0, -1)
				if dup != -1 {
					fmt.Printf("Já existe mapeamento para '%s' na coluna %q. Deseja substituir? (s/n): ", typ, header[dup])
					resp, _ := reader.ReadString('\n')
					resp = strings.ToLower(strings.TrimSpace(resp))
					if resp == "s" {
						for {
							fmt.Printf("Para qual campo deseja remapear a coluna %q? ", header[dup])
							newField, _ := reader.ReadString('\n')
							newField = strings.ToLower(strings.TrimSpace(newField))
							if newField == "opcao" {
								fmt.Println("Utilize 'opcao' apenas na definição original informando o índice.")
								continue
							}
							if !validTypes[newField] {
								fmt.Println("Inválido. Digite um campo válido (exceto opcao).")
								continue
							}
							if findDuplicate(newField, 0, dup) != -1 {
								fmt.Println("Esse campo já está mapeado em outra coluna, tente outro.")
								continue
							}
							colMap[dup] = colInfo{field: newField}
							break
						}
						chosenType = typ
					} else {
						fmt.Println("Por favor, escolha outro campo para esta coluna.")
						continue
					}
				} else {
					chosenType = typ
				}
			} else {
				chosenType = typ
			}
			break
		}
		colMap[i] = colInfo{field: chosenType, optIndex: chosenOptIndex}
	}

	// 3) Monta slice de usuários
	var users []Usuario
	for _, row := range rows[1:] {
		u := Usuario{
			Opcoes: make([]string, nOpcoes),
		}
		for idx, cell := range row {
			if idx >= len(colMap) {
				continue
			}
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
