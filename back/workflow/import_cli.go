package workflow

import (
	"bufio"
	"candidate_alocator/back/allocation"
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
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
	candidateMapping, err = getInteractiveMapping("Candidatos", data, 0, candidateMapping)
	if err != nil {
		return err
	}
	usuariosResp, err := logic.BuildUsuariosWithMapping(data, optionCount, candidateMapping)
	if err != nil {
		return fmt.Errorf("erro ao construir candidatos: %w", err)
	}

	candidatos, summary := buildCandidatesForCLI(usuariosResp)
	printCandidateSummary(summary, usuariosResp)
	if len(candidatos) == 0 {
		return errors.New("nenhum candidato valido sobrou para salvar apos filtrar erros e duplicados")
	}

	if err := logic.Save(candidatos); err != nil {
		return fmt.Errorf("erro ao salvar candidatos: %w", err)
	}

	restricaoMapping, err := logic.SuggestMappingRestricao(data)
	if err != nil {
		return fmt.Errorf("erro ao sugerir mapeamento de restricoes: %w", err)
	}
	restricaoMapping, err = getInteractiveMapping("Restricoes", data, 2, restricaoMapping)
	if err != nil {
		return err
	}
	restricoes, err := logic.BuildRestricoesWithMapping(data, restricaoMapping)
	if err != nil {
		return fmt.Errorf("erro ao construir restricoes: %w", err)
	}

	avaliadorMapping, err := logic.SuggestMappingAvaliador(data)
	if err != nil {
		return fmt.Errorf("erro ao sugerir mapeamento de avaliadores: %w", err)
	}
	avaliadorMapping, err = getInteractiveMapping("Avaliadores", data, 1, avaliadorMapping)
	if err != nil {
		return err
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

func getInteractiveMapping(label string, data []byte, sheetIndex int, currentMapping []types.MappingItem) ([]types.MappingItem, error) {
	fmt.Printf("\n---- CONFIGURAÇÃO DE MAPEAMENTO: %s ----\n", strings.ToUpper(label))

	rows, err := logic.GetRowsFromSheet(data, sheetIndex)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("aba %s sem dados", label)
	}
	header := rows[0]

	fmt.Println("Colunas detectadas no Excel:")
	for i, col := range header {
		fmt.Printf("[%d] %s | ", i, col)
		if (i+1)%4 == 0 {
			fmt.Println()
		}
	}
	fmt.Println()

	fmt.Println("Sugestão de mapeamento atual:")
	for _, item := range currentMapping {
		colName := item.NomeColuna
		if colName == "" {
			colName = "<não mapeado>"
		}
		fmt.Printf("  %s -> [%d] %s\n", item.Variavel, item.Indice, colName)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nDeseja personalizar este mapeamento? (s/N): ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "s" {
		// Se não quiser personalizar, removemos os itens que não têm coluna associada
		// (conforme a regra de que só o que estiver explicitamente mapeado será importado)
		var filtered []types.MappingItem
		for _, item := range currentMapping {
			if item.NomeColuna != "" {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	}

	var newMapping []types.MappingItem
	for _, item := range currentMapping {
		fmt.Printf("Campo '%s' [atual: %d (%s)]. Novo índice (Enter p/ manter, -1 p/ ignorar): ",
			item.Variavel, item.Indice, item.NomeColuna)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			if item.NomeColuna != "" {
				newMapping = append(newMapping, item)
			}
			continue
		}

		idx, err := strconv.Atoi(input)
		if err != nil || idx < 0 {
			fmt.Printf("Campo '%s' ignorado.\n", item.Variavel)
			continue
		}

		if idx < len(header) {
			newMapping = append(newMapping, types.MappingItem{
				NomeColuna: header[idx],
				Indice:     idx,
				Variavel:   item.Variavel,
			})
		} else {
			fmt.Println("Índice inválido, campo ignorado.")
		}
	}

	for {
		fmt.Print("\nDeseja adicionar um campo extra? (s/N): ")
		extraAns, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(extraAns)) != "s" {
			break
		}

		fmt.Print("Índice da coluna no Excel: ")
		idxStr, _ := reader.ReadString('\n')
		idx, err := strconv.Atoi(strings.TrimSpace(idxStr))
		if err != nil || idx < 0 || idx >= len(header) {
			fmt.Println("Índice inválido.")
			continue
		}

		fmt.Print("Nome para este campo (chave no banco): ")
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)
		if key == "" {
			key = header[idx]
		}

		newMapping = append(newMapping, types.MappingItem{
			NomeColuna: header[idx],
			Indice:     idx,
			Variavel:   key,
		})
	}

	return newMapping, nil
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
