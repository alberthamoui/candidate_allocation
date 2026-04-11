package allocation

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	types "candidate_alocator/back/type"
)

const defaultMaxDebugEvents = 32

type SolverOptions struct {
	WorkerCount    int
	ParallelDepth  int
	MaxDebugEvents int
}

type solverProblemView struct {
	problem                types.AllocationProblem
	candidates             map[int]types.SolverCandidate
	groups                 map[int]types.SolverGroup
	sortedCandidateIDs     []int
	candidateGroupOptions  map[int][]solverGroupOption
	assignmentPenaltyByKey map[int]map[int]int
}

type solverGroupOption struct {
	GroupID            int
	ImmediatePenalty   int
	EvaluatorConflicts int
}

type solverState struct {
	Assignments map[int]int
	GroupCounts map[int]int
}

type preparedTask struct {
	State solverState
}

type buildFrontierContext struct {
	view        solverProblemView
	options     SolverOptions
	tasks       []preparedTask
	metrics     types.SolverMetrics
	debugNotes  []string
	hardWitness []types.HardConstraintViolation
}

type subtreeSolver struct {
	view        solverProblemView
	options     SolverOptions
	best        *types.SolverResult
	metrics     types.SolverMetrics
	debugNotes  []string
	hardWitness []types.HardConstraintViolation
}

type subtreeOutcome struct {
	best        *types.SolverResult
	metrics     types.SolverMetrics
	debugNotes  []string
	hardWitness []types.HardConstraintViolation
}

// SolveAllocation executa a busca exata deterministica com branch and bound.
func SolveAllocation(problem types.AllocationProblem, options SolverOptions) types.SolverResult {
	// problema tem canditados grupos e regras e o options tem os parametros de execução como o paralelimso
	// arruma as opções
	normalizedOptions := NormalizeSolverOptions(options)
	// view basicamente tem dados ja processados, quais grupos esse candidato pode entrar e etc.
	view := buildSolverProblemView(problem)
	// estado inicial do solver com mapa vazio com espaço para os grupos
	root := solverState{
		Assignments: make(map[int]int),
		GroupCounts: make(map[int]int, len(problem.Groups)),
	}

	frontier := buildFrontierContext{
		view:    view,
		options: normalizedOptions,
	}
	// vai podando ramos impossíveis cedo, antes de resolver tudo
	frontier.expand(root, normalizedOptions.ParallelDepth)

	// se tiver tarefas ele vai resolvendo cada uma
	if len(frontier.tasks) == 0 {
		result := buildInfeasibleResult(frontier.hardWitness)
		result.Metrics = frontier.metrics
		result.Metrics.ParallelTasks = 0
		result.DebugNotes = append([]string(nil), frontier.debugNotes...)
		return result
	}

	results := make([]subtreeOutcome, len(frontier.tasks))
	parallelize := normalizedOptions.WorkerCount > 1 && len(frontier.tasks) > 1
	if parallelize {
		semaphore := make(chan struct{}, normalizedOptions.WorkerCount)
		var wg sync.WaitGroup
		for index, task := range frontier.tasks {
			wg.Add(1)
			go func(index int, task preparedTask) {
				defer wg.Done()
				semaphore <- struct{}{}
				results[index] = solvePreparedTask(view, normalizedOptions, task)
				<-semaphore
			}(index, task)
		}
		wg.Wait()
	} else {
		for index, task := range frontier.tasks {
			results[index] = solvePreparedTask(view, normalizedOptions, task)
		}
	}

	finalMetrics := frontier.metrics
	finalMetrics.ParallelTasks = len(frontier.tasks)
	finalNotes := append([]string(nil), frontier.debugNotes...)
	var finalBest *types.SolverResult
	finalWitness := append([]types.HardConstraintViolation(nil), frontier.hardWitness...)

	for _, outcome := range results {
		finalMetrics = mergeSolverMetrics(finalMetrics, outcome.metrics)
		finalNotes = appendLimitedNotes(finalNotes, outcome.debugNotes, normalizedOptions.MaxDebugEvents)
		if finalBest == nil || (outcome.best != nil && isBetterSolverResult(*outcome.best, *finalBest)) {
			if outcome.best != nil {
				copied := cloneSolverResult(*outcome.best)
				finalBest = &copied
			}
		}
		if len(finalWitness) == 0 && len(outcome.hardWitness) > 0 {
			finalWitness = append([]types.HardConstraintViolation(nil), outcome.hardWitness...)
		}
	}

	if finalBest == nil {
		result := buildInfeasibleResult(finalWitness)
		result.Metrics = finalMetrics
		result.DebugNotes = finalNotes
		return result
	}

	finalBest.Metrics = finalMetrics
	finalBest.DebugNotes = finalNotes
	return *finalBest
}

func NormalizeSolverOptions(options SolverOptions) SolverOptions {
	normalized := options
	if normalized.WorkerCount <= 0 {
		normalized.WorkerCount = 1
	}
	if normalized.ParallelDepth < 0 {
		normalized.ParallelDepth = 0
	}
	if normalized.WorkerCount > 1 && normalized.ParallelDepth == 0 {
		normalized.ParallelDepth = 1
	}
	if normalized.MaxDebugEvents <= 0 {
		normalized.MaxDebugEvents = defaultMaxDebugEvents
	}
	return normalized
}

func EstimateLowerBound(problem types.AllocationProblem, state types.PartialAllocationState) int {
	view := buildSolverProblemView(problem)
	return estimateLowerBound(view, state)
}

func buildSolverProblemView(problem types.AllocationProblem) solverProblemView {
	view := solverProblemView{
		problem:                problem,
		candidates:             candidateByID(problem),
		groups:                 groupByID(problem),
		sortedCandidateIDs:     make([]int, 0, len(problem.Candidates)),
		candidateGroupOptions:  make(map[int][]solverGroupOption, len(problem.Candidates)),
		assignmentPenaltyByKey: make(map[int]map[int]int, len(problem.Candidates)),
	}

	for _, candidate := range problem.Candidates {
		view.sortedCandidateIDs = append(view.sortedCandidateIDs, candidate.ID)
		options := make([]solverGroupOption, 0, len(problem.Groups))
		penalties := make(map[int]int, len(problem.Groups))
		for _, group := range problem.Groups {
			if problem.HardRestrictions.RespectCandidatePreferences && !containsInt(candidate.PreferredGroupIDs, group.ID) {
				continue
			}
			if problem.HardRestrictions.EnforceForbiddenEvaluators && groupHasForbiddenEvaluator(candidate, group) {
				continue
			}

			penalty := assignmentPenalty(problem, candidate, group)
			options = append(options, solverGroupOption{
				GroupID:            group.ID,
				ImmediatePenalty:   penalty,
				EvaluatorConflicts: countAvoidEvaluatorConflicts(candidate, group),
			})
			penalties[group.ID] = penalty
		}

		sort.SliceStable(options, func(i, j int) bool {
			if options[i].ImmediatePenalty != options[j].ImmediatePenalty {
				return options[i].ImmediatePenalty < options[j].ImmediatePenalty
			}
			if options[i].EvaluatorConflicts != options[j].EvaluatorConflicts {
				return options[i].EvaluatorConflicts < options[j].EvaluatorConflicts
			}
			return options[i].GroupID < options[j].GroupID
		})

		view.candidateGroupOptions[candidate.ID] = options
		view.assignmentPenaltyByKey[candidate.ID] = penalties
	}

	sort.Ints(view.sortedCandidateIDs)
	return view
}

func solvePreparedTask(view solverProblemView, options SolverOptions, task preparedTask) subtreeOutcome {
	solver := subtreeSolver{
		view:    view,
		options: options,
	}
	solver.resume(task.State)
	return subtreeOutcome{
		best:        solver.best,
		metrics:     solver.metrics,
		debugNotes:  append([]string(nil), solver.debugNotes...),
		hardWitness: append([]types.HardConstraintViolation(nil), solver.hardWitness...),
	}
}

func (builder *buildFrontierContext) expand(state solverState, depth int) {
	builder.metrics.NodesVisited++
	partial := buildPartialAllocationState(state)
	// verifica se estado atual ja viola regra hard
	if violations := pruneViolations(builder.view, state, partial); len(violations) > 0 {
		builder.metrics.NodesPrunedByHard++
		builder.captureWitness(violations)
		builder.note(fmt.Sprintf("hard prune before frontier expansion: %s", violations[0].Code))
		return
	}

	if depth == 0 || IsCompleteState(builder.view.problem, partial) {
		builder.tasks = append(builder.tasks, preparedTask{State: cloneSolverState(state)})
		return
	}

	candidateID, groupIDs := chooseNextCandidate(builder.view, state)
	if candidateID == 0 {
		builder.tasks = append(builder.tasks, preparedTask{State: cloneSolverState(state)})
		return
	}
	if len(groupIDs) == 0 {
		violations := []types.HardConstraintViolation{candidateWithoutGroupViolation(candidateID)}
		builder.metrics.NodesPrunedByHard++
		builder.captureWitness(violations)
		builder.note(fmt.Sprintf("hard prune before frontier expansion: %s", violations[0].Code))
		return
	}

	for _, groupID := range groupIDs {
		builder.expand(assignCandidateToGroup(state, candidateID, groupID), depth-1)
	}
}

func (builder *buildFrontierContext) captureWitness(violations []types.HardConstraintViolation) {
	if len(builder.hardWitness) == 0 && len(violations) > 0 {
		builder.hardWitness = append([]types.HardConstraintViolation(nil), violations...)
	}
}

func (builder *buildFrontierContext) note(message string) {
	builder.debugNotes = appendSingleNote(builder.debugNotes, builder.options.MaxDebugEvents, message)
}

func (solver *subtreeSolver) search(state solverState) {
	solver.metrics.NodesVisited++
	partial := buildPartialAllocationState(state)
	if violations := pruneViolations(solver.view, state, partial); len(violations) > 0 {
		solver.metrics.NodesPrunedByHard++
		solver.captureWitness(violations)
		solver.note(fmt.Sprintf("hard prune: %s", violations[0].Code))
		return
	}
	solver.resume(state)
}

func (solver *subtreeSolver) resume(state solverState) {
	partial := buildPartialAllocationState(state)
	if IsCompleteState(solver.view.problem, partial) {
		solver.metrics.CompleteStates++
		score := ScoreAllocation(solver.view.problem, partial)
		result := types.SolverResult{
			Status:      "optimal",
			Assignments: cloneAssignments(state.Assignments),
			Score:       score,
		}
		if solver.best == nil || isBetterSolverResult(result, *solver.best) {
			solver.metrics.BestUpdates++
			copied := cloneSolverResult(result)
			solver.best = &copied
			solver.note(fmt.Sprintf("best update: score=%d assignments=%s", score.TotalPenalty, formatAssignments(result.Assignments)))
		}
		return
	}

	lowerBound := estimateLowerBound(solver.view, partial)
	if solver.best != nil && lowerBound >= solver.best.Score.TotalPenalty {
		solver.metrics.NodesPrunedByBound++
		solver.note(fmt.Sprintf("bound prune: lower_bound=%d incumbent=%d", lowerBound, solver.best.Score.TotalPenalty))
		return
	}

	candidateID, groupIDs := chooseNextCandidate(solver.view, state)
	if candidateID == 0 {
		solver.metrics.CompleteStates++
		return
	}
	if len(groupIDs) == 0 {
		violations := []types.HardConstraintViolation{candidateWithoutGroupViolation(candidateID)}
		solver.metrics.NodesPrunedByHard++
		solver.captureWitness(violations)
		solver.note(fmt.Sprintf("hard prune: %s", violations[0].Code))
		return
	}

	for _, groupID := range groupIDs {
		solver.search(assignCandidateToGroup(state, candidateID, groupID))
	}
}

func (solver *subtreeSolver) captureWitness(violations []types.HardConstraintViolation) {
	if len(solver.hardWitness) == 0 && len(violations) > 0 {
		solver.hardWitness = append([]types.HardConstraintViolation(nil), violations...)
	}
}

func (solver *subtreeSolver) note(message string) {
	solver.debugNotes = appendSingleNote(solver.debugNotes, solver.options.MaxDebugEvents, message)
}

func pruneViolations(view solverProblemView, state solverState, partial types.PartialAllocationState) []types.HardConstraintViolation {
	if viable, violations := IsStateViable(view.problem, partial); !viable {
		return violations
	}
	return futureHardViolations(view, state)
}

func futureHardViolations(view solverProblemView, state solverState) []types.HardConstraintViolation {
	remainingCandidates := len(view.problem.Candidates) - len(state.Assignments)
	if view.problem.HardRestrictions.AllCandidatesMustBeAssigned && remainingCandidates > remainingCapacity(view, state) {
		return []types.HardConstraintViolation{insufficientCapacityViolation(remainingCandidates, remainingCapacity(view, state))}
	}

	for _, candidateID := range view.sortedCandidateIDs {
		if _, assigned := state.Assignments[candidateID]; assigned {
			continue
		}
		if len(currentFeasibleGroupIDs(view, state, candidateID)) == 0 {
			return []types.HardConstraintViolation{candidateWithoutGroupViolation(candidateID)}
		}
	}

	for _, group := range view.problem.Groups {
		current := state.GroupCounts[group.ID]
		if current == 0 || current >= group.MinCandidates {
			continue
		}
		possible := current + countPotentialCandidatesForGroup(view, state, group.ID)
		if possible < group.MinCandidates {
			return []types.HardConstraintViolation{groupCannotReachMinimumViolation(group.ID, group.MinCandidates, possible)}
		}
	}

	return nil
}

func chooseNextCandidate(view solverProblemView, state solverState) (int, []int) {
	bestCandidateID := 0
	var bestGroups []int
	for _, candidateID := range view.sortedCandidateIDs {
		if _, assigned := state.Assignments[candidateID]; assigned {
			continue
		}

		groupIDs := currentFeasibleGroupIDs(view, state, candidateID)
		if bestCandidateID == 0 || len(groupIDs) < len(bestGroups) || (len(groupIDs) == len(bestGroups) && candidateID < bestCandidateID) {
			bestCandidateID = candidateID
			bestGroups = groupIDs
		}
		if len(groupIDs) == 0 {
			break
		}
	}
	return bestCandidateID, bestGroups
}

func currentFeasibleGroupIDs(view solverProblemView, state solverState, candidateID int) []int {
	options := view.candidateGroupOptions[candidateID]
	groupIDs := make([]int, 0, len(options))
	for _, option := range options {
		if view.problem.HardRestrictions.EnforceGroupCapacity {
			group := view.groups[option.GroupID]
			if state.GroupCounts[option.GroupID] >= group.MaxCandidates {
				continue
			}
		}
		groupIDs = append(groupIDs, option.GroupID)
	}
	return groupIDs
}

func estimateLowerBound(view solverProblemView, state types.PartialAllocationState) int {
	total := 0
	for candidateID, groupID := range state.Assignments {
		total += view.assignmentPenaltyByKey[candidateID][groupID]
	}

	for _, criterion := range view.problem.SoftRules.Criteria {
		switch criterion.Type {
		case types.SoftCriterionMaxValue:
			total += lowerBoundMaxValueCriterion(view, state, criterion)
		case types.SoftCriterionGroupTogether:
			total += lowerBoundGroupTogetherCriterion(view, state, criterion)
		}
	}

	return total
}

func lowerBoundMaxValueCriterion(view solverProblemView, state types.PartialAllocationState, criterion types.SoftCriterion) int {
	if len(criterion.SelectedValues) != 1 {
		return 0
	}
	selectedValue := criterion.SelectedValues[0]
	counts := countSelectedValuesByGroup(view.problem, state, criterion.ColumnKey, criterion.SelectedValues)
	penalty := 0
	for _, group := range view.problem.Groups {
		count := counts[group.ID][selectedValue]
		if count > criterion.Threshold {
			penalty += count - criterion.Threshold
		}
	}
	return penalty
}

func lowerBoundGroupTogetherCriterion(view solverProblemView, state types.PartialAllocationState, criterion types.SoftCriterion) int {
	selectedValues := make(map[string]struct{}, len(criterion.SelectedValues))
	for _, selectedValue := range criterion.SelectedValues {
		selectedValues[selectedValue] = struct{}{}
	}

	groupMembers := buildStateGroupMembers(state)
	groupsWithValues := make(map[int]struct{})
	for groupID, members := range groupMembers {
		for _, candidateID := range members {
			value := view.candidates[candidateID].Attributes[criterion.ColumnKey]
			if _, ok := selectedValues[value]; ok {
				groupsWithValues[groupID] = struct{}{}
				break
			}
		}
	}

	if len(groupsWithValues) <= 1 {
		return 0
	}
	return len(groupsWithValues) - 1
}

func assignCandidateToGroup(state solverState, candidateID, groupID int) solverState {
	child := cloneSolverState(state)
	child.Assignments[candidateID] = groupID
	child.GroupCounts[groupID]++
	return child
}

func cloneSolverState(state solverState) solverState {
	return solverState{
		Assignments: cloneAssignments(state.Assignments),
		GroupCounts: cloneAssignments(state.GroupCounts),
	}
}

func cloneAssignments(assignments map[int]int) map[int]int {
	cloned := make(map[int]int, len(assignments))
	for candidateID, groupID := range assignments {
		cloned[candidateID] = groupID
	}
	return cloned
}

func buildPartialAllocationState(state solverState) types.PartialAllocationState {
	return types.PartialAllocationState{Assignments: state.Assignments}
}

func assignmentPenalty(problem types.AllocationProblem, candidate types.SolverCandidate, group types.SolverGroup) int {
	preferenceRank := indexOfInt(candidate.PreferredGroupIDs, group.ID)
	penalty := preferencePenaltyForRank(problem.SoftRules.PreferencePenaltyByRank, preferenceRank)
	penalty += countAvoidEvaluatorConflicts(candidate, group) * problem.SoftRules.AvoidEvaluatorPenalty
	return penalty
}

func countAvoidEvaluatorConflicts(candidate types.SolverCandidate, group types.SolverGroup) int {
	conflicts := 0
	for _, evaluatorID := range group.EvaluatorIDs {
		if containsInt(candidate.EvaluatorRestrictions.AvoidEvaluatorIDs, evaluatorID) {
			conflicts++
		}
	}
	return conflicts
}

func groupHasForbiddenEvaluator(candidate types.SolverCandidate, group types.SolverGroup) bool {
	for _, evaluatorID := range group.EvaluatorIDs {
		if containsInt(candidate.EvaluatorRestrictions.ForbiddenEvaluatorIDs, evaluatorID) {
			return true
		}
	}
	return false
}

func remainingCapacity(view solverProblemView, state solverState) int {
	remaining := 0
	for _, group := range view.problem.Groups {
		if !view.problem.HardRestrictions.EnforceGroupCapacity {
			return len(view.problem.Candidates)
		}
		remaining += group.MaxCandidates - state.GroupCounts[group.ID]
	}
	return remaining
}

func countPotentialCandidatesForGroup(view solverProblemView, state solverState, groupID int) int {
	count := 0
	for _, candidateID := range view.sortedCandidateIDs {
		if _, assigned := state.Assignments[candidateID]; assigned {
			continue
		}
		if containsInt(currentFeasibleGroupIDs(view, state, candidateID), groupID) {
			count++
		}
	}
	return count
}

func mergeSolverMetrics(left, right types.SolverMetrics) types.SolverMetrics {
	return types.SolverMetrics{
		NodesVisited:       left.NodesVisited + right.NodesVisited,
		CompleteStates:     left.CompleteStates + right.CompleteStates,
		NodesPrunedByHard:  left.NodesPrunedByHard + right.NodesPrunedByHard,
		NodesPrunedByBound: left.NodesPrunedByBound + right.NodesPrunedByBound,
		BestUpdates:        left.BestUpdates + right.BestUpdates,
		ParallelTasks:      left.ParallelTasks + right.ParallelTasks,
	}
}

func isBetterSolverResult(candidate, current types.SolverResult) bool {
	if candidate.Status != "optimal" {
		return false
	}
	if current.Status != "optimal" {
		return true
	}
	if candidate.Score.TotalPenalty != current.Score.TotalPenalty {
		return candidate.Score.TotalPenalty < current.Score.TotalPenalty
	}
	return compareAssignments(candidate.Assignments, current.Assignments) < 0
}

func compareAssignments(left, right map[int]int) int {
	keys := make([]int, 0, len(left))
	for candidateID := range left {
		keys = append(keys, candidateID)
	}
	sort.Ints(keys)
	for _, candidateID := range keys {
		if left[candidateID] < right[candidateID] {
			return -1
		}
		if left[candidateID] > right[candidateID] {
			return 1
		}
	}
	return 0
}

func cloneSolverResult(result types.SolverResult) types.SolverResult {
	result.Assignments = cloneAssignments(result.Assignments)
	result.Score.Components = append([]types.SoftScoreComponent(nil), result.Score.Components...)
	result.HardViolations = append([]types.HardConstraintViolation(nil), result.HardViolations...)
	result.DebugNotes = append([]string(nil), result.DebugNotes...)
	return result
}

func buildInfeasibleResult(violations []types.HardConstraintViolation) types.SolverResult {
	result := types.SolverResult{
		Status:         "infeasible",
		Assignments:    map[int]int{},
		Score:          types.SoftScoreBreakdown{},
		HardViolations: append([]types.HardConstraintViolation(nil), violations...),
		DebugNotes:     []string{},
	}
	if len(violations) == 0 {
		result.RejectionReason = "nenhuma solucao hard valida foi encontrada"
		return result
	}
	result.RejectionReason = violations[0].Message
	return result
}

func appendSingleNote(notes []string, limit int, message string) []string {
	if limit <= 0 || len(notes) >= limit {
		return notes
	}
	return append(notes, message)
}

func appendLimitedNotes(base, extra []string, limit int) []string {
	for _, note := range extra {
		if len(base) >= limit {
			return base
		}
		base = append(base, note)
	}
	return base
}

func formatAssignments(assignments map[int]int) string {
	if len(assignments) == 0 {
		return "{}"
	}
	keys := make([]int, 0, len(assignments))
	for candidateID := range assignments {
		keys = append(keys, candidateID)
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, candidateID := range keys {
		parts = append(parts, fmt.Sprintf("%d->%d", candidateID, assignments[candidateID]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func candidateWithoutGroupViolation(candidateID int) types.HardConstraintViolation {
	return types.HardConstraintViolation{
		Code:        "candidate_without_feasible_group",
		Message:     fmt.Sprintf("candidato %d nao possui grupo viavel restante", candidateID),
		CandidateID: candidateID,
	}
}

func insufficientCapacityViolation(required, available int) types.HardConstraintViolation {
	return types.HardConstraintViolation{
		Code:    "insufficient_remaining_capacity",
		Message: fmt.Sprintf("capacidade restante insuficiente: faltam %d candidatos para %d vagas", required, available),
	}
}

func groupCannotReachMinimumViolation(groupID, minimum, possible int) types.HardConstraintViolation {
	return types.HardConstraintViolation{
		Code:    "group_cannot_reach_min_candidates",
		Message: fmt.Sprintf("grupo %d nao consegue atingir o minimo %d; maximo possivel %d", groupID, minimum, possible),
		GroupID: groupID,
	}
}
