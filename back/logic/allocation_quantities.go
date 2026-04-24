package logic

import types "candidate_alocator/back/type"

// CountPossibleAllocationQuantities conta quantas alocações distintas existem
// quando os grupos são rotulados e as pessoas são distintas.
func countWaysForScheduleNoEmpty(G, min, max, people int) int {
	var generate func(partsLeft, currentSum, minSizeAllowed int, currentSizes []int) int
	generate = func(partsLeft, currentSum, minSizeAllowed int, currentSizes []int) int {
		if partsLeft == 0 {
			if currentSum == people {
				return calculateWays(people, currentSizes)
			}
			return 0
		}

		totalWays := 0
		for size := minSizeAllowed; size <= max; size++ {
			if size < min {
				continue
			}
			if currentSum+size > people {
				break
			}

			currentSizes = append(currentSizes, size)
			totalWays += generate(partsLeft-1, currentSum+size, size, currentSizes)
			currentSizes = currentSizes[:len(currentSizes)-1]
		}
		return totalWays
	}

	return generate(G, 0, min, make([]int, 0, G))
}

func countWaysForScheduleEmpty(G, min, max, people int) int {
	var generate func(partsLeft, currentSum, minSizeAllowed int, currentSizes []int) int
	generate = func(partsLeft, currentSum, minSizeAllowed int, currentSizes []int) int {
		if partsLeft == 0 {
			if currentSum == people {
				return calculateWays(people, currentSizes)
			}
			return 0
		}

		totalWays := 0
		for size := minSizeAllowed; size <= max; size++ {
			if size > 0 && size < min {
				continue
			}
			if currentSum+size > people {
				break
			}

			currentSizes = append(currentSizes, size)
			totalWays += generate(partsLeft-1, currentSum+size, size, currentSizes)
			currentSizes = currentSizes[:len(currentSizes)-1]
		}
		return totalWays
	}

	return generate(G, 0, 0, make([]int, 0, G))
}

func calculateWays(totalPeople int, sizes []int) int {
	ways := 1
	remaining := totalPeople
	counts := make(map[int]int)
	for _, s := range sizes {
		ways *= binomialCoefficient(remaining, s)
		remaining -= s
		if s > 0 {
			counts[s]++
		}
	}
	for _, c := range counts {
		ways /= factorial(c)
	}
	return ways
}

func factorial(n int) int {
	res := 1
	for i := 2; i <= n; i++ {
		res *= i
	}
	return res
}

func CountPossibleAllocationQuantities(params types.AllocationParams, totalPeople int) int {
	if totalPeople < 0 {
		return 0
	}
	if params.GruposPorHorario <= 0 || params.MinPessoasPorGrupo <= 0 || params.MaxPessoasPorGrupo <= 0 {
		return 0
	}
	if params.MinPessoasPorGrupo > params.MaxPessoasPorGrupo {
		return 0
	}

	minTotal := params.GruposPorHorario * params.MinPessoasPorGrupo
	maxTotal := params.GruposPorHorario * params.MaxPessoasPorGrupo
	if totalPeople < minTotal {
		return 0
	}

	if totalPeople <= maxTotal {
		return countWaysForScheduleNoEmpty(params.GruposPorHorario, params.MinPessoasPorGrupo, params.MaxPessoasPorGrupo, totalPeople)
	}

	total := 0
	for assigned := minTotal; assigned <= maxTotal; assigned++ {
		ways := binomialCoefficient(totalPeople, assigned) * countWaysForScheduleNoEmpty(params.GruposPorHorario, params.MinPessoasPorGrupo, params.MaxPessoasPorGrupo, assigned)
		total += ways
	}
	return total
}

func CountPossibleAllocationQuantitiesAcrossSchedules(params types.AllocationParams, totalPeople, scheduleCount int) int {
	if totalPeople < 0 || scheduleCount <= 0 {
		return 0
	}
	if params.GruposPorHorario <= 0 || params.MinPessoasPorGrupo <= 0 || params.MaxPessoasPorGrupo <= 0 {
		return 0
	}
	if params.MinPessoasPorGrupo > params.MaxPessoasPorGrupo {
		return 0
	}

	totalSlots := params.GruposPorHorario * scheduleCount
	if totalSlots <= 0 || totalPeople > totalSlots*params.MaxPessoasPorGrupo {
		return 0
	}

	waysForSchedule := make([]int, totalPeople+1)
	for p := 0; p <= totalPeople; p++ {
		waysForSchedule[p] = countWaysForScheduleEmpty(params.GruposPorHorario, params.MinPessoasPorGrupo, params.MaxPessoasPorGrupo, p)
	}

	dp := make([]int, totalPeople+1)
	dp[0] = 1

	for s := 1; s <= scheduleCount; s++ {
		nextDp := make([]int, totalPeople+1)
		for p := 0; p <= totalPeople; p++ {
			for k := 0; k <= p; k++ {
				ways := dp[p-k] * waysForSchedule[k] * binomialCoefficient(p, k)
				nextDp[p] += ways
			}
		}
		dp = nextDp
	}

	return dp[totalPeople]
}

// CalculatePossibleAllocationQuantities devolve todas as distribuições
// ordenadas de pessoas entre os grupos, respeitando os limites mínimo e máximo.
func CalculatePossibleAllocationQuantities(params types.AllocationParams, totalPeople int) [][]int {
	if totalPeople < 0 {
		return nil
	}
	if params.GruposPorHorario <= 0 || params.MinPessoasPorGrupo <= 0 || params.MaxPessoasPorGrupo <= 0 {
		return nil
	}
	if params.MinPessoasPorGrupo > params.MaxPessoasPorGrupo {
		return nil
	}

	minTotal := params.GruposPorHorario * params.MinPessoasPorGrupo
	maxTotal := params.GruposPorHorario * params.MaxPessoasPorGrupo
	if totalPeople < minTotal || totalPeople > maxTotal {
		return nil
	}

	return enumeratePossibleAllocationQuantities(params.GruposPorHorario, params.MinPessoasPorGrupo, params.MaxPessoasPorGrupo, totalPeople)
}

// CalculatePossibleAllocationQuantitiesAcrossSchedules devolve todas as
// distribuições ordenadas considerando todos os grupos disponíveis em todos os
// horários.
func CalculatePossibleAllocationQuantitiesAcrossSchedules(params types.AllocationParams, totalPeople, scheduleCount int) [][]int {
	if totalPeople < 0 || scheduleCount <= 0 {
		return nil
	}
	if params.GruposPorHorario <= 0 || params.MinPessoasPorGrupo <= 0 || params.MaxPessoasPorGrupo <= 0 {
		return nil
	}
	if params.MinPessoasPorGrupo > params.MaxPessoasPorGrupo {
		return nil
	}

	totalSlots := params.GruposPorHorario * scheduleCount
	if totalSlots <= 0 || totalPeople > totalSlots*params.MaxPessoasPorGrupo {
		return nil
	}

	return enumeratePossibleAllocationQuantities(totalSlots, params.MinPessoasPorGrupo, params.MaxPessoasPorGrupo, totalPeople)
}

func enumeratePossibleAllocationQuantities(groupsLeft, minPerGroup, maxPerGroup, remaining int) [][]int {
	result := make([][]int, 0)
	enumerateAllocationQuantities(groupsLeft, minPerGroup, maxPerGroup, remaining, make([]int, 0, groupsLeft), &result)
	return result
}

func enumerateAllocationQuantities(groupsLeft, minPerGroup, maxPerGroup, remaining int, current []int, result *[][]int) {
	if groupsLeft == 0 {
		if remaining == 0 {
			allocation := append([]int(nil), current...)
			*result = append(*result, allocation)
		}
		return
	}

	if groupsLeft == 1 {
		if remaining >= minPerGroup && remaining <= maxPerGroup {
			allocation := append(append([]int(nil), current...), remaining)
			*result = append(*result, allocation)
		}
		return
	}

	minValue := maxInt(minPerGroup, remaining-(groupsLeft-1)*maxPerGroup)
	maxValue := minInt(maxPerGroup, remaining-(groupsLeft-1)*minPerGroup)
	for peopleInGroup := minValue; peopleInGroup <= maxValue; peopleInGroup++ {
		current = append(current, peopleInGroup)
		enumerateAllocationQuantities(groupsLeft-1, minPerGroup, maxPerGroup, remaining-peopleInGroup, current, result)
		current = current[:len(current)-1]
	}
}

func binomialCoefficient(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n-k {
		k = n - k
	}

	result := 1
	for i := 1; i <= k; i++ {
		result = result * (n - k + i) / i
	}
	return result
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
