//go:build ignore
// +build ignore

// Este é um arquivo de exemplo que pode ser executado via: go run example_cli.go -file arquivo.xlsx
// Ele demonstra como usar a API do aplicativo via linha de comando
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	SetUp()

	path := flag.String("file", "", "caminho para o arquivo .xlsx")
	flag.Parse()
	if *path == "" {
		fmt.Println("Uso: go run example_cli.go -file seu_arquivo.xlsx")
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
	if err != nil {
		fmt.Println("Erro ao processar avaliadores:", err)
		os.Exit(1)
	}

	mappingRestricao, err := app.SuggestMappingRestricao()
	if err != nil {
		fmt.Println("Erro ao processar restrições:", err)
		os.Exit(1)
	}

	fmt.Println("\n")
	fmt.Println("mapping candidatos : ", mapping)
	fmt.Println("\n")
	fmt.Println("mapping avaliadores : ", mappingAvaliador)
	fmt.Println("\n")
	fmt.Println("mapping restricao : ", mappingRestricao)
	fmt.Println("\n")

	usuarios, err := app.BuildUsuariosWithMapping(mapping)
	if err != nil {
		fmt.Println("Erro ao ler o arquivo:", err)
		os.Exit(1)
	}
	usuarios_filtrados := FilterUniqueUsers(usuarios)

	avaliadores, err := app.BuildAvaliadoresWithMapping(mappingAvaliador)
	if err != nil {
		fmt.Println("Erro ao ler o arquivo:", err)
		os.Exit(1)
	}
	restricao, err := app.BuildRestricoesWithMapping(mappingRestricao)
	if err != nil {
		fmt.Println("Erro ao ler o arquivo:", err)
		os.Exit(1)
	}
	fmt.Println("\n")
	fmt.Println("usuarios: ", usuarios)
	fmt.Println("\n\n\n")
	fmt.Println("avaliadores: ", avaliadores)
	fmt.Println("\n")
	fmt.Println("Restricao: ", restricao)
	fmt.Println("\n")

	app.Save(usuarios_filtrados)
	app.Save(avaliadores)
	app.Save(restricao)

	// Alocacao
	conn, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		fmt.Println("Erro ao conectar ao banco:", err)
		os.Exit(1)
	}
	defer conn.Close()

	Alocar(conn)

	fmt.Println("\n=== Processamento concluído com sucesso! ===")
}
