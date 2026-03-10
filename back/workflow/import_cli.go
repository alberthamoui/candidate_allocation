package workflow

import (
	"candidate_alocator/back/allocation"
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type cliCandidateSummary struct {
	Total            int
	Selected         int
	SkippedInvalid   int
	SkippedDuplicate int
	DuplicateGroups  int
}

func RunCLI(ctx context.Context, filePath string, optionCount int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		return err
	}
	db, err := dbpkg.OpenDefault()
	if err != nil {
		fmt.Println("Erro ao abrir banco para limpeza:", err)
		panic(err)
	}
	defer db.Close()

	if err := dbpkg.ClearDatabase(db); err != nil {
		fmt.Println("Erro ao limpar banco:", err)
		panic(err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("erro ao ler o arquivo %q: %w", filePath, err)
	}

	candidateMapping, err := logic.SuggestMapping(data, optionCount)
	if err != nil {
		return fmt.Errorf("erro ao sugerir mapeamento de candidatos: %w", err)
	}

	avaliadorMapping, err := logic.SuggestMappingAvaliador(data)
	if err != nil {
		return fmt.Errorf("erro ao sugerir mapeamento de avaliadores: %w", err)
	}

	restricaoMapping, err := logic.SuggestMappingRestricao(data)
	if err != nil {
		return fmt.Errorf("erro ao sugerir mapeamento de restricoes: %w", err)
	}

	printMapping("candidatos", candidateMapping)
	printMapping("avaliadores", avaliadorMapping)
	printMapping("restricoes", restricaoMapping)

	usuariosResp, err := logic.BuildUsuariosWithMapping(data, optionCount, candidateMapping)
	if err != nil {
		return fmt.Errorf("erro ao construir candidatos: %w", err)
	}

	restricoes, err := logic.BuildRestricoesWithMapping(data, restricaoMapping)
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

	avaliadoresResp, err := logic.BuildAvaliadoresWithMapping(data, avaliadorMapping)
	if err != nil {
		return fmt.Errorf("erro ao construir avaliadores: %w", err)
	}

	avaliadores := buildAvaliadoresForCLI(avaliadoresResp)
	if err := logic.Save(avaliadores); err != nil {
		return fmt.Errorf("erro ao salvar avaliador: %w", err)
	}

	fmt.Printf("Avaliadores processados: %d\n", len(avaliadores))

	if err := logic.Save(restricoes); err != nil {
		return fmt.Errorf("erro ao salvar restricoes: %w", err)
	}
	fmt.Printf("Restricoes processadas: %d\n", len(restricoes))

	if err := allocation.Run(db); err != nil {
		return fmt.Errorf("erro ao executar alocacao: %w", err)
	}

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

func buildCandidatesForCLI(resp logic.UsuariosResponse) ([]types.Candidato, cliCandidateSummary) {
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

func printCandidateSummary(summary cliCandidateSummary, resp logic.UsuariosResponse) {
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

func buildAvaliadoresForCLI(resp logic.AvaliadoresResponse) []types.Avaliador {
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
			if result, ok := resp.Avaliadores[id]; ok && len(result.Erros) == 0 {
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

	ids := make([]int, 0, len(resp.Avaliadores))
	for id := range resp.Avaliadores {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	avaliadores := make([]types.Avaliador, 0, len(ids))
	for _, id := range ids {
		result := resp.Avaliadores[id]

		if len(result.Erros) > 0 {
			continue
		}

		if duplicateMembers[id] && !duplicateChosen[id] {
			continue
		}

		avaliadores = append(avaliadores, result.Avaliador)
	}

	return avaliadores
}
