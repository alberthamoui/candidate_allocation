//go:build ignore
// +build ignore

// Executa o fluxo completo do backend via linha de comando, sem precisar do Wails.
//
// Uso:
//
// Terminal (Linux/Mac):
// go run $(ls *.go | grep -v _test.go | grep -v '^main\.go$' | tr '\n' ' ') -file ./Execelteste/base_exemplo.xlsx
// 
// Terminal (Windows):
// go run example_cli.go -file ./Execelteste/base_exemplo.xlsx -opcoes 5 -domain @al.insper.edu.br
//
// powerShell:
// go run example_cli.go app.go models.go mapping.go export.go alocate.go processa.go setup.go -file ./Execelteste/base_exemplo.xlsx
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	filePath := flag.String("file", "", "caminho para o arquivo .xlsx")
	nOpcoes := flag.Int("opcoes", 5, "número de opções de horário por candidato")
	domain := flag.String("domain", "@al.insper.edu.br", "domínio do email institucional")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Uso: go run example_cli.go -file arquivo.xlsx [-opcoes 5] [-domain @al.insper.edu.br]")
		os.Exit(1)
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Println("Erro ao ler o arquivo:", err)
		os.Exit(1)
	}

	// ── Inicializa banco e app ────────────────────────────────────────────
	SetUp()
	app := NewApp()

	// ── Mapeamento ────────────────────────────────────────────────────────
	mapping, err := app.SuggestMapping(data, *nOpcoes, *domain)
	if err != nil {
		fmt.Println("Erro ao sugerir mapeamento de candidatos:", err)
		os.Exit(1)
	}
	mappingAvaliador, err := app.SuggestMappingAvaliador()
	if err != nil {
		fmt.Println("Erro ao sugerir mapeamento de avaliadores:", err)
		os.Exit(1)
	}
	mappingRestricao, err := app.SuggestMappingRestricao()
	if err != nil {
		fmt.Println("Erro ao sugerir mapeamento de restrições:", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Mapeamento candidatos ===")
	for _, m := range mapping {
		fmt.Printf("  coluna %q (idx %d) → %s\n", m.NomeColuna, m.Indice, m.Variavel)
	}
	fmt.Println("\n=== Mapeamento avaliadores ===")
	for _, m := range mappingAvaliador {
		fmt.Printf("  coluna %q (idx %d) → %s\n", m.NomeColuna, m.Indice, m.Variavel)
	}
	fmt.Println("\n=== Mapeamento restrições ===")
	for _, m := range mappingRestricao {
		fmt.Printf("  coluna %q (idx %d) → %s\n", m.NomeColuna, m.Indice, m.Variavel)
	}

	// ── Constrói dados ────────────────────────────────────────────────────
	usuariosResp, err := app.BuildUsuariosWithMapping(mapping)
	if err != nil {
		fmt.Println("Erro ao construir candidatos:", err)
		os.Exit(1)
	}
	avaliadores, err := app.BuildAvaliadoresWithMapping(mappingAvaliador)
	if err != nil {
		fmt.Println("Erro ao construir avaliadores:", err)
		os.Exit(1)
	}
	restricoes, err := app.BuildRestricoesWithMapping(mappingRestricao)
	if err != nil {
		fmt.Println("Erro ao construir restrições:", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Candidatos (%d) ===\n", len(usuariosResp.Usuarios))
	for idx, vr := range usuariosResp.Usuarios {
		erroStr := ""
		if len(vr.Erros) > 0 {
			erroStr = fmt.Sprintf(" [ERROS: %v]", vr.Erros)
		}
		fmt.Printf("  [%d] %s%s\n", idx, vr.Usuario.Nome, erroStr)
	}
	if len(usuariosResp.Duplicates) > 0 {
		fmt.Println("\n  Duplicatas detectadas:", usuariosResp.Duplicates)
	}

	fmt.Printf("\n=== Avaliadores (%d) ===\n", len(avaliadores))
	for _, a := range avaliadores {
		fmt.Printf("  %s (%s) – sigla: %s\n", a.Nome, a.Email, a.Sigla)
	}

	fmt.Printf("\n=== Restrições (%d) ===\n", len(restricoes))
	for _, r := range restricoes {
		fmt.Printf("  %s | NaoPosso: %q | PrefiroNao: %q\n", r.Candidato, r.NaoPosso, r.PrefiroNao)
	}

	// ── Salva no banco ────────────────────────────────────────────────────
	candidatosUnicos := FilterUniqueUsers(usuariosResp)
	if err := app.SaveUsuarios(candidatosUnicos); err != nil {
		fmt.Println("Erro ao salvar candidatos:", err)
		os.Exit(1)
	}
	if err := app.SaveAvaliadores(avaliadores); err != nil {
		fmt.Println("Erro ao salvar avaliadores:", err)
		os.Exit(1)
	}
	if err := app.SaveRestricoes(restricoes); err != nil {
		fmt.Println("Erro ao salvar restrições:", err)
		os.Exit(1)
	}

	// ── Alocação ──────────────────────────────────────────────────────────
	conn, err := sql.Open("sqlite3", "./base.db")
	if err != nil {
		fmt.Println("Erro ao conectar ao banco:", err)
		os.Exit(1)
	}
	defer conn.Close()

	Alocar(conn)

	fmt.Println("\n=== Processamento concluído com sucesso! ===")
}
