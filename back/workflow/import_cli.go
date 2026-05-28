package workflow

import (
	"bufio"
	"candidate_alocator/back/allocation"
	dbpkg "candidate_alocator/back/db"
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	defer func() { _ = db.Close() }()

	if err := dbpkg.ClearDatabase(db); err != nil {
		fmt.Println("Erro ao limpar banco:", err)
		panic(err)
	}

	// #nosec G304 - CLI input path is user-provided and handled as a local file.
	data, err := os.ReadFile(filepath.Clean(filePath))
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
	config, err := collectAllocationSetup(os.Stdin, os.Stdout, candidatos)
	if err != nil {
		return err
	}
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar configuracao em JSON: %w", err)
	}
	fmt.Println(string(configJSON))

	allocationResult, err := allocation.RunConfiguredAllocation(db, config)
	if err != nil {
		return fmt.Errorf("erro ao executar alocacao configurada: %w", err)
	}
	resultJSON, err := json.MarshalIndent(allocationResult.Result, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar resultado da alocacao em JSON: %w", err)
	}
	fmt.Println(string(resultJSON))

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

func collectAllocationSetup(in io.Reader, out io.Writer, candidatos []types.Candidato) (types.AllocationConfiguration, error) {
	reader := bufio.NewReader(in)
	detections := logic.DetectUniquePreferenceValues(candidatos)

	_, _ = fmt.Fprintln(out, "\n---- PASSO 5: MAPEAMENTO DE PREFERENCIAS ----")
	if len(detections) == 0 {
		_, _ = fmt.Fprintln(out, "Nenhuma preferencia foi detectada nos candidatos.")
	}

	mappings := make([]types.PreferenceScheduleMapping, 0, len(detections))
	for _, detection := range detections {
		_, _ = fmt.Fprintf(out, "\nPreferencia detectada: %s (%s, %d ocorrencias)\n",
			detection.ValorOriginal, detection.ValorNormalizado, detection.Ocorrencias)

		dia, err := promptLineWithDefault(reader, out, "Dia real", "segunda")
		if err != nil {
			return types.AllocationConfiguration{}, err
		}
		hora, err := promptLineWithDefault(reader, out, "Hora real", "08:00")
		if err != nil {
			return types.AllocationConfiguration{}, err
		}

		mappings = append(mappings, types.PreferenceScheduleMapping{
			ValorPreferencia: detection.ValorNormalizado,
			Dia:              dia,
			Hora:             hora,
		})
	}

	if err := logic.ValidatePreferenceScheduleMappings(mappings); err != nil {
		return types.AllocationConfiguration{}, err
	}

	defaults := logic.DefaultAllocationParams()
	params := defaults

	_, _ = fmt.Fprintln(out, "\n---- PASSO 6: PARAMETROS DE ALOCACAO ----")
	_, _ = fmt.Fprintf(out, "Grupos por horario [%d]: ", defaults.GruposPorHorario)
	if value, err := readOptionalLine(reader); err != nil {
		return types.AllocationConfiguration{}, err
	} else if value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return types.AllocationConfiguration{}, fmt.Errorf("grupos por horario invalido: %w", err)
		}
		params.GruposPorHorario = parsed
	}

	_, _ = fmt.Fprintf(out, "Minimo de pessoas por grupo [%d]: ", defaults.MinPessoasPorGrupo)
	if value, err := readOptionalLine(reader); err != nil {
		return types.AllocationConfiguration{}, err
	} else if value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return types.AllocationConfiguration{}, fmt.Errorf("minimo de pessoas por grupo invalido: %w", err)
		}
		params.MinPessoasPorGrupo = parsed
	}

	_, _ = fmt.Fprintf(out, "Maximo de pessoas por grupo [%d]: ", defaults.MaxPessoasPorGrupo)
	if value, err := readOptionalLine(reader); err != nil {
		return types.AllocationConfiguration{}, err
	} else if value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return types.AllocationConfiguration{}, fmt.Errorf("maximo de pessoas por grupo invalido: %w", err)
		}
		params.MaxPessoasPorGrupo = parsed
	}

	_, _ = fmt.Fprintf(out, "Avaliadores por grupo [%d]: ", defaults.AvaliadoresPorGrupo)
	if value, err := readOptionalLine(reader); err != nil {
		return types.AllocationConfiguration{}, err
	} else if value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return types.AllocationConfiguration{}, fmt.Errorf("avaliadores por grupo invalido: %w", err)
		}
		params.AvaliadoresPorGrupo = parsed
	}

	criteria, err := collectSoftCriteria(reader, out, candidatos)
	if err != nil {
		return types.AllocationConfiguration{}, err
	}
	params.SoftCriteria = criteria

	config, err := logic.BuildAllocationConfiguration(detections, mappings, params, candidatos)
	if err != nil {
		return config, err
	}

	_, _ = fmt.Fprintln(out, "\n---- RESUMO DA CONFIGURACAO ----")
	printAllocationQuantities(out, config.Normalized.Params, len(candidatos), len(config.Normalized.PreferenceMappings))

	return config, nil
}

type softCriterionOption struct {
	Type        types.SoftCriterionType
	Label       string
	Description string
}

func collectSoftCriteria(reader *bufio.Reader, out io.Writer, candidatos []types.Candidato) ([]types.SoftCriterion, error) {
	options := []softCriterionOption{
		{Type: types.SoftCriterionMinValue, Label: "Minimo por valor unico", Description: "Se existe o valor selecionado no grupo, garanta pelo menos N"},
		{Type: types.SoftCriterionAtLeastOneEach, Label: "Pelo menos 1 de cada valor", Description: "Garanta representacao para os valores selecionados"},
		{Type: types.SoftCriterionBalancedDistribution, Label: "Distribuicao equilibrada", Description: "Espalhe os valores selecionados entre os grupos"},
		{Type: types.SoftCriterionGroupTogether, Label: "Agrupamento", Description: "Prefira manter os valores selecionados juntos"},
		{Type: types.SoftCriterionMaxValue, Label: "Maximo por valor unico", Description: "Limite o numero de pessoas do valor selecionado por grupo"},
	}
	columns := logic.ListCandidateCriterionColumns(candidatos)
	if len(columns) == 0 {
		return []types.SoftCriterion{}, nil
	}

	criteria := make([]types.SoftCriterion, 0)
	_, _ = fmt.Fprintln(out, "\nConfiguracao de criterios soft:")
	for {
		addMore, err := promptYesNo(reader, out, "Deseja adicionar um criterio soft? (s/N): ")
		if err != nil {
			return nil, err
		}
		if !addMore {
			return criteria, nil
		}

		_, _ = fmt.Fprintln(out, "\nTipos de criterio disponiveis:")
		for i, option := range options {
			_, _ = fmt.Fprintf(out, "[%d] %s - %s\n", i+1, option.Label, option.Description)
		}
		optionIndex, err := promptChoiceIndex(reader, out, "Tipo do criterio: ", len(options))
		if err != nil {
			return nil, err
		}
		selectedOption := options[optionIndex]

		_, _ = fmt.Fprintln(out, "\nColunas elegiveis:")
		for i, column := range columns {
			_, _ = fmt.Fprintf(out, "[%d] %s\n", i+1, column.Label)
		}
		columnIndex, err := promptChoiceIndex(reader, out, "Coluna do criterio: ", len(columns))
		if err != nil {
			return nil, err
		}
		selectedColumn := columns[columnIndex]

		detections, err := logic.DetectUniqueCandidateColumnValues(candidatos, selectedColumn.Key)
		if err != nil {
			return nil, err
		}
		if len(detections) == 0 {
			return nil, fmt.Errorf("a coluna %q nao possui valores unicos selecionaveis", selectedColumn.Key)
		}

		_, _ = fmt.Fprintf(out, "\nValores unicos de %s:\n", selectedColumn.Label)
		for i, detection := range detections {
			_, _ = fmt.Fprintf(out, "[%d] %s (%d ocorrencias)\n", i+1, detection.ValorOriginal, detection.Ocorrencias)
		}

		var selectedValues []string
		switch selectedOption.Type {
		case types.SoftCriterionMinValue, types.SoftCriterionMaxValue:
			valueIndex, err := promptChoiceIndex(reader, out, "Valor selecionado: ", len(detections))
			if err != nil {
				return nil, err
			}
			selectedValues = []string{detections[valueIndex].ValorNormalizado}
		default:
			selectedIndices, err := promptMultiChoiceIndices(reader, out, "Valores selecionados (ex.: 1,3): ", len(detections))
			if err != nil {
				return nil, err
			}
			selectedValues = make([]string, 0, len(selectedIndices))
			for _, index := range selectedIndices {
				selectedValues = append(selectedValues, detections[index].ValorNormalizado)
			}
		}

		criterion := types.SoftCriterion{
			Type:           selectedOption.Type,
			ColumnKey:      selectedColumn.Key,
			SelectedValues: selectedValues,
		}

		if selectedOption.Type == types.SoftCriterionMinValue || selectedOption.Type == types.SoftCriterionMaxValue {
			threshold, err := promptPositiveInt(reader, out, "Numero N: ")
			if err != nil {
				return nil, err
			}
			criterion.Threshold = threshold
		}

		criterion = logic.NormalizeSoftCriteria([]types.SoftCriterion{criterion})[0]
		if err := logic.ValidateSoftCriteria([]types.SoftCriterion{criterion}, candidatos); err != nil {
			return nil, err
		}

		_, _ = fmt.Fprintf(out, "Resumo do criterio: %s\n", formatSoftCriterion(criterion))
		confirm, err := promptYesNo(reader, out, "Confirmar criterio? (s/N): ")
		if err != nil {
			return nil, err
		}
		if confirm {
			criteria = append(criteria, criterion)
		}
	}
}

func formatSoftCriterion(criterion types.SoftCriterion) string {
	return logic.DescribeSoftCriterion(criterion)
}

func printAllocationQuantities(out io.Writer, params types.AllocationParams, totalPeople, scheduleCount int) {
	count := logic.CountPossibleAllocationQuantitiesAcrossSchedules(params, totalPeople, scheduleCount)

	_, _ = fmt.Fprintln(out, "\n---- QUANTIDADES POSSIVEIS DE ALOCACAO ----")
	if count == 0 {
		_, _ = fmt.Fprintf(out, "Nenhuma distribuicao valida para %d candidatos e %d horarios com os parametros atuais.\n", totalPeople, scheduleCount)
		return
	}

	_, _ = fmt.Fprintf(out, "Quantidade total de alocacoes distintas: %d\n", count)
}

func promptYesNo(reader *bufio.Reader, out io.Writer, label string) (bool, error) {
	_, _ = fmt.Fprint(out, label)
	value, err := readOptionalLine(reader)
	if err != nil {
		return false, err
	}
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "s" || value == "sim" || value == "y" || value == "yes", nil
}

func promptChoiceIndex(reader *bufio.Reader, out io.Writer, label string, max int) (int, error) {
	for {
		_, _ = fmt.Fprint(out, label)
		value, err := readOptionalLine(reader)
		if err != nil {
			return 0, err
		}
		index, err := strconv.Atoi(value)
		if err == nil && index >= 1 && index <= max {
			return index - 1, nil
		}
		_, _ = fmt.Fprintln(out, "Escolha invalida.")
	}
}

func promptMultiChoiceIndices(reader *bufio.Reader, out io.Writer, label string, max int) ([]int, error) {
	for {
		_, _ = fmt.Fprint(out, label)
		value, err := readOptionalLine(reader)
		if err != nil {
			return nil, err
		}
		parts := strings.Split(value, ",")
		indices := make([]int, 0, len(parts))
		seen := make(map[int]struct{})
		valid := true
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			index, err := strconv.Atoi(part)
			if err != nil || index < 1 || index > max {
				valid = false
				break
			}
			normalizedIndex := index - 1
			if _, ok := seen[normalizedIndex]; ok {
				continue
			}
			seen[normalizedIndex] = struct{}{}
			indices = append(indices, normalizedIndex)
		}
		if valid && len(indices) > 0 {
			return indices, nil
		}
		_, _ = fmt.Fprintln(out, "Selecao invalida.")
	}
}

func promptPositiveInt(reader *bufio.Reader, out io.Writer, label string) (int, error) {
	for {
		_, _ = fmt.Fprint(out, label)
		value, err := readOptionalLine(reader)
		if err != nil {
			return 0, err
		}
		parsed, err := strconv.Atoi(value)
		if err == nil && parsed > 0 {
			return parsed, nil
		}
		_, _ = fmt.Fprintln(out, "Numero invalido.")
	}
}

func promptLineWithDefault(reader *bufio.Reader, out io.Writer, label, defaultValue string) (string, error) {
	_, _ = fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
	value, err := readOptionalLine(reader)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return defaultValue, nil
		}
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func promptRequiredLine(reader *bufio.Reader, out io.Writer, label string) (string, error) {
	for {
		_, _ = fmt.Fprint(out, label)
		value, err := readOptionalLine(reader)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
		_, _ = fmt.Fprintln(out, "Valor obrigatorio.")
	}
}

func readOptionalLine(reader *bufio.Reader) (string, error) {
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	trimmed := strings.TrimSpace(value)
	if errors.Is(err, io.EOF) && trimmed == "" {
		return "", io.EOF
	}
	return trimmed, nil
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
