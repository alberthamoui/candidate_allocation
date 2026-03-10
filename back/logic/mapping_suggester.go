package logic

import (
	"sort"
	"strings"
	"unicode"

	types "candidate_alocator/back/type"
)

type mappingCandidate struct {
	variableIndex      int
	variable           string
	headerIndex        int
	header             string
	score              int
	exactNormalized    bool
	exactTokenSequence bool
	sharedTokens       int
	tokenCountDelta    int
}

func normalizeMappingName(value string) string {
	return strings.Join(tokenizeMappingName(value), " ")
}

func tokenizeMappingName(value string) []string {
	var builder strings.Builder
	var prev rune

	for _, r := range value {
		switch {
		case r == '_' || r == '-' || unicode.IsPunct(r) || unicode.IsSpace(r):
			builder.WriteRune(' ')
		case unicode.IsUpper(r) && builder.Len() > 0 && (unicode.IsLower(prev) || unicode.IsDigit(prev)):
			builder.WriteRune(' ')
			builder.WriteRune(unicode.ToLower(r))
		case unicode.IsDigit(r) && builder.Len() > 0 && unicode.IsLetter(prev):
			builder.WriteRune(' ')
			builder.WriteRune(r)
		case unicode.IsLetter(r) && builder.Len() > 0 && unicode.IsDigit(prev):
			builder.WriteRune(' ')
			builder.WriteRune(unicode.ToLower(r))
		default:
			builder.WriteRune(unicode.ToLower(r))
		}
		prev = r
	}

	rawTokens := strings.Fields(builder.String())
	var result []string
	for _, t := range rawTokens {
		result = append(result, normalizeNumericToken(t))
	}
	return result
}

func normalizeNumericToken(token string) string {
	switch token {
	case "um", "uma", "primeira", "primeiro", "1a", "1o":
		return "1"
	case "dois", "duas", "segunda", "segundo", "2a", "2o":
		return "2"
	case "tres", "terceira", "terceiro", "3a", "3o":
		return "3"
	case "quatro", "quarta", "quarto", "4a", "4o":
		return "4"
	case "cinco", "quinta", "quinto", "5a", "5o":
		return "5"
	case "seis", "sexta", "sexto", "6a", "6o":
		return "6"
	case "sete", "setima", "setimo", "7a", "7o":
		return "7"
	case "oito", "oitava", "oitavo", "8a", "8o":
		return "8"
	case "nove", "nona", "nono", "9a", "9o":
		return "9"
	case "dez", "decima", "decimo", "10a", "10o":
		return "10"
	}
	return token
}

func mappingSimilarityScore(variable string, header string) int {
	score, _, _, _, _, _ := mappingSimilarityDetails(variable, header)
	return score
}

func mappingSimilarityDetails(variable string, header string) (int, bool, bool, int, int, int) {
	normalizedVariable := normalizeMappingName(variable)
	normalizedHeader := normalizeMappingName(header)

	varTokens := tokenizeMappingName(variable)
	headerTokens := tokenizeMappingName(header)

	varHeaderCompact := strings.ReplaceAll(normalizedHeader, " ", "")
	varVariableCompact := strings.ReplaceAll(normalizedVariable, " ", "")

	exactNormalized := normalizedVariable != "" && (normalizedVariable == normalizedHeader || normalizedVariable == varHeaderCompact || varVariableCompact == normalizedHeader || varVariableCompact == varHeaderCompact)
	exactTokenSequence := slicesEqual(varTokens, headerTokens)
	sharedTokens := countSharedTokens(varTokens, headerTokens)
	allTokensMatch := sharedTokens > 0 && sharedTokens == len(varTokens) && sharedTokens == len(headerTokens)
	tokenCountDelta := abs(len(varTokens) - len(headerTokens))

	score := 0
	switch {
	case exactNormalized || allTokensMatch:
		score += 10_000
	case exactTokenSequence:
		score += 9_000
	default:
		if normalizedVariable != "" && normalizedHeader != "" {
			if strings.Contains(normalizedHeader, normalizedVariable) || strings.Contains(normalizedVariable, normalizedHeader) {
				score += 3_000
			}
		}
		score += sharedTokens * 700
		if sharedTokens > 0 {
			score += 200
		}
	}

	varNumber, hasVarNumber := trailingNumber(varTokens)
	headerNumber, hasHeaderNumber := trailingNumber(headerTokens)
	if hasVarNumber && hasHeaderNumber {
		if varNumber == headerNumber {
			score += 2_500
		} else {
			score -= 2_500
		}
	}

	if tokenCountDelta == 0 {
		score += 100
	} else {
		score -= tokenCountDelta * 10
	}

	if score < 0 {
		score = 0
	}

	return score, exactNormalized, exactTokenSequence, sharedTokens, tokenCountDelta, len(headerTokens)
}

func buildMappingCandidates(headers []string, variables []string) []mappingCandidate {
	candidates := make([]mappingCandidate, 0, len(headers)*len(variables))
	for variableIndex, variable := range variables {
		for headerIndex, header := range headers {
			score, exactNormalized, exactTokenSequence, sharedTokens, tokenCountDelta, _ := mappingSimilarityDetails(variable, header)
			candidates = append(candidates, mappingCandidate{
				variableIndex:      variableIndex,
				variable:           variable,
				headerIndex:        headerIndex,
				header:             header,
				score:              score,
				exactNormalized:    exactNormalized,
				exactTokenSequence: exactTokenSequence,
				sharedTokens:       sharedTokens,
				tokenCountDelta:    tokenCountDelta,
			})
		}
	}

	return candidates
}

func resolveMappingConflicts(headers []string, variables []string, candidates []mappingCandidate) ([]types.MappingItem, map[int]bool) {
	assignments := make([]types.MappingItem, len(variables))
	for i, variable := range variables {
		assignments[i] = unmappedMappingItem(variable, len(headers), i)
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]

		switch {
		case left.score != right.score:
			return left.score > right.score
		case left.exactNormalized != right.exactNormalized:
			return left.exactNormalized
		case left.exactTokenSequence != right.exactTokenSequence:
			return left.exactTokenSequence
		case left.sharedTokens != right.sharedTokens:
			return left.sharedTokens > right.sharedTokens
		case left.tokenCountDelta != right.tokenCountDelta:
			return left.tokenCountDelta < right.tokenCountDelta
		case left.headerIndex != right.headerIndex:
			return left.headerIndex < right.headerIndex
		default:
			return left.variableIndex < right.variableIndex
		}
	})

	usedHeaders := make(map[int]bool, len(headers))
	usedVariables := make(map[int]bool, len(variables))
	assignmentLimit := min(len(headers), len(variables))
	assigned := 0

	threshold := 0.2

	for _, candidate := range candidates {
		if assigned >= assignmentLimit {
			break
		}
		if usedHeaders[candidate.headerIndex] || usedVariables[candidate.variableIndex] {
			continue
		}

		// Check similarity threshold for core variables
		if stringSimilarity(candidate.variable, candidate.header) < threshold {
			continue
		}

		assignments[candidate.variableIndex] = types.MappingItem{
			NomeColuna: candidate.header,
			Indice:     candidate.headerIndex,
			Variavel:   candidate.variable,
		}
		usedHeaders[candidate.headerIndex] = true
		usedVariables[candidate.variableIndex] = true
		assigned++
	}

	return assignments, usedHeaders
}

func suggestMappingByName(headers []string, variables []string) []types.MappingItem {
	if len(headers) == 0 {
		items := make([]types.MappingItem, 0, len(variables))
		for i, variable := range variables {
			items = append(items, unmappedMappingItem(variable, 0, i))
		}
		return items
	}

	candidates := buildMappingCandidates(headers, variables)
	assignments, usedHeaders := resolveMappingConflicts(headers, variables, candidates)

	// Add all unused headers as extra mappings
	for i, header := range headers {
		if usedHeaders[i] {
			continue
		}
		assignments = append(assignments, types.MappingItem{
			NomeColuna: header,
			Indice:     i,
			Variavel:   header, // Suggested as extra with same name as column
		})
	}

	return assignments
}

func stringSimilarity(s1, s2 string) float64 {
	s1 = strings.ToLower(normalizeMappingName(s1))
	s2 = strings.ToLower(normalizeMappingName(s2))
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	d := make([][]int, len(s1)+1)
	for i := range d {
		d[i] = make([]int, len(s2)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, min(d[i][j-1]+1, d[i-1][j-1]+cost))
		}
	}

	distance := d[len(s1)][len(s2)]
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

func unmappedMappingItem(variable string, headerCount int, offset int) types.MappingItem {
	return types.MappingItem{
		NomeColuna: "",
		Indice:     headerCount + offset,
		Variavel:   variable,
	}
}

func countSharedTokens(left []string, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}

	frequencies := make(map[string]int, len(right))
	for _, token := range right {
		frequencies[token]++
	}

	shared := 0
	for _, token := range left {
		if frequencies[token] == 0 {
			continue
		}
		frequencies[token]--
		shared++
	}

	return shared
}

func trailingNumber(tokens []string) (int, bool) {
	if len(tokens) == 0 {
		return 0, false
	}

	last := tokens[len(tokens)-1]
	value := 0
	for _, r := range last {
		if !unicode.IsDigit(r) {
			return 0, false
		}
		value = value*10 + int(r-'0')
	}

	return value, true
}

func slicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
