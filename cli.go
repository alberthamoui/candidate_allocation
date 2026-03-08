package main

import (
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const defaultCLIOptionCount = 5

type cliCandidateSummary struct {
	Total            int
	Selected         int
	SkippedInvalid   int
	SkippedDuplicate int
	DuplicateGroups  int
}

func detectCLIMode(args []string) ([]string, bool) {
	if len(args) == 0 {
		return nil, false
	}

	switch args[0] {
	case "cli", "-cli", "--cli":
		return args[1:], true
	default:
		return nil, false
	}
}

func runCLI(args []string) error {
	flagSet := flag.NewFlagSet("cli", flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)

	filePath := flagSet.String("file", "", "caminho para o arquivo .xlsx")
	optionCount := flagSet.Int("opcoes", defaultCLIOptionCount, "quantidade de opcoes de horario esperadas na aba de candidatos")

	flagSet.Usage = func() {
		fmt.Fprintf(flagSet.Output(), "Uso: %s cli -file arquivo.xlsx [-opcoes %d]\n", os.Args[0], defaultCLIOptionCount)
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*filePath) == "" {
		flagSet.Usage()
		return errors.New("o parametro -file e obrigatorio no modo CLI")
	}
	if *optionCount <= 0 {
		return fmt.Errorf("o parametro -opcoes deve ser maior que zero")
	}

	return executeCLIFlow(*filePath, *optionCount)
}

func executeCLIFlow(filePath string, optionCount int) error {
	SetUp()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("erro ao ler o arquivo %q: %w", filePath, err)
	}

	app := NewApp()

	candidateMapping, err := app.SuggestMapping(data, optionCount)
	if err != nil {
		return fmt.Errorf("erro ao sugerir mapeamento de candidatos: %w", err)
	}

	avaliadorMapping, err := app.SuggestMappingAvaliador()
	if err != nil {
		return fmt.Errorf("erro ao sugerir mapeamento de avaliadores: %w", err)
	}

	restricaoMapping, err := app.SuggestMappingRestricao()
	if err != nil {
		return fmt.Errorf("erro ao sugerir mapeamento de restricoes: %w", err)
	}

	printMapping("candidatos", candidateMapping)
	printMapping("avaliadores", avaliadorMapping)
	printMapping("restricoes", restricaoMapping)

	usuariosResp, err := app.BuildUsuariosWithMapping(candidateMapping)
	if err != nil {
		return fmt.Errorf("erro ao construir candidatos: %w", err)
	}

	restricoes, err := app.BuildRestricoesWithMapping(restricaoMapping)
	if err != nil {
		return fmt.Errorf("erro ao construir restricoes: %w", err)
	}

	candidatos, summary := buildCandidatesForCLI(usuariosResp)
	printCandidateSummary(summary, usuariosResp)
	if len(candidatos) == 0 {
		return errors.New("nenhum candidato valido sobrou para salvar apos filtrar erros e duplicados")
	}

	if err := logic.Save(candidatos); err != nil {
		return fmt.Errorf("erro ao salvar candidatos: %w", err)
	}

	avaliadores, err := app.BuildAvaliadoresWithMapping(avaliadorMapping)
	if err != nil {
		return fmt.Errorf("erro ao construir avaliadores: %w", err)
	}
	fmt.Printf("Avaliadores processados: %d\n", len(avaliadores))

	if err := logic.Save(restricoes); err != nil {
		return fmt.Errorf("erro ao salvar restricoes: %w", err)
	}
	fmt.Printf("Restricoes processadas: %d\n", len(restricoes))

	db, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		return fmt.Errorf("erro ao abrir banco para alocacao: %w", err)
	}
	defer db.Close()

	Alocar(db)
	return nil
}

func printMapping(label string, items []types.MappingItem) {
	fmt.Printf("\n---- MAPEAMENTO %s ----\n", strings.ToUpper(label))
	for _, item := range items {
		columnName := item.NomeColuna
		if strings.TrimSpace(columnName) == "" {
			columnName = "<sem coluna>"
		}
		fmt.Printf("%s <- coluna %d (%s)\n", item.Variavel, item.Indice, columnName)
	}
}

func buildCandidatesForCLI(resp UsuariosResponse) ([]types.Candidato, cliCandidateSummary) {
	summary := cliCandidateSummary{
		Total:           len(resp.Usuarios),
		DuplicateGroups: len(resp.Duplicates),
	}

	duplicateMembers := make(map[int]bool)
	duplicateChosen := make(map[int]bool)

	for _, rawGroup := range resp.Duplicates {
		group := append([]int(nil), rawGroup...)
		sort.Ints(group)

		chosenID := 0
		for _, id := range group {
			duplicateMembers[id] = true
			if chosenID != 0 {
				continue
			}
			if result, ok := resp.Usuarios[id]; ok && len(result.Erros) == 0 {
				chosenID = id
			}
		}

		if chosenID == 0 && len(group) > 0 {
			chosenID = group[0]
		}
		if chosenID != 0 {
			duplicateChosen[chosenID] = true
		}
	}

	ids := make([]int, 0, len(resp.Usuarios))
	for id := range resp.Usuarios {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	candidatos := make([]types.Candidato, 0, len(ids))
	for _, id := range ids {
		result := resp.Usuarios[id]

		if len(result.Erros) > 0 {
			summary.SkippedInvalid++
			continue
		}

		if duplicateMembers[id] && !duplicateChosen[id] {
			summary.SkippedDuplicate++
			continue
		}

		candidatos = append(candidatos, result.Usuario)
	}

	summary.Selected = len(candidatos)
	return candidatos, summary
}

func printCandidateSummary(summary cliCandidateSummary, resp UsuariosResponse) {
	fmt.Printf("\n---- RESUMO CANDIDATOS ----\n")
	fmt.Printf("Total lidos: %d\n", summary.Total)
	fmt.Printf("Selecionados para salvar: %d\n", summary.Selected)
	fmt.Printf("Ignorados por validacao: %d\n", summary.SkippedInvalid)
	fmt.Printf("Ignorados por duplicidade: %d\n", summary.SkippedDuplicate)
	fmt.Printf("Grupos de duplicados detectados: %d\n", summary.DuplicateGroups)
	if len(resp.Duplicates) > 0 {
		fmt.Printf("Duplicados detectados: %v\n", resp.Duplicates)
	}
}
