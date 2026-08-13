package allocation

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"

	types "candidate_alocator/back/type"
)

const (
	defaultMaxDebugEvents = 32
	defaultFlowBoundDepth = 2
)

type SolverOptions struct {
	WorkerCount    int
	ParallelDepth  int
	MaxDebugEvents int
	FlowBoundDepth int
	Progress       ProgressCallback
}

type solverProblemView struct {
	problem                types.AllocationProblem
	candidates             map[int]types.SolverCandidate
	groups                 map[int]types.SolverGroup
	sortedCandidateIDs     []int
	sortedGroupIDs         []int
	candidateIndexByID     map[int]int
	groupIndexByID         map[int]int
	candidateGroupOptions  map[int][]solverGroupOption
	assignmentPenaltyByKey map[int]map[int]int
	groupSymmetryClass     map[int]int
}

type solverGroupOption struct {
	GroupID            int
	GroupIndex         int
	ImmediatePenalty   int
	EvaluatorConflicts int
}

type solverState struct {
	Assignments   []int
	GroupCounts   []int
	AssignedCount int
	BasePenalty   int
}

type preparedTask struct {
	State  solverState
	Weight *big.Int
}

type sharedIncumbent struct {
	mu        sync.RWMutex
	best      *types.SolverResult
	taskIndex int
}

type buildFrontierContext struct {
	view        solverProblemView
	options     SolverOptions
	tasks       []preparedTask
	metrics     types.SolverMetrics
	debugNotes  []string
	hardWitness []types.HardConstraintViolation
	progress    *solverProgressTracker
	incumbent   *sharedIncumbent
}

type subtreeSolver struct {
	view        solverProblemView
	options     SolverOptions
	taskIndex   int
	metrics     types.SolverMetrics
	debugNotes  []string
	hardWitness []types.HardConstraintViolation
	progress    *solverProgressTracker
	incumbent   *sharedIncumbent
}

type subtreeOutcome struct {
	metrics     types.SolverMetrics
	debugNotes  []string
	hardWitness []types.HardConstraintViolation
}

func solveConnectedAllocation(problem types.AllocationProblem, options SolverOptions) types.SolverResult {
	normalizedOptions := NormalizeSolverOptions(options)
	view := buildSolverProblemView(problem)
	progress, rootWeight := newSolverProgressTracker(view, normalizedOptions.Progress)
	defer progress.finish()

	root := newSolverState(view)
	incumbent := &sharedIncumbent{taskIndex: int(^uint(0) >> 1)}
	frontier := buildFrontierContext{
		view:      view,
		options:   normalizedOptions,
		progress:  progress,
		incumbent: incumbent,
	}

	if assignment, feasible := initialMinCostAssignment(view, &root); feasible {
		partial := types.PartialAllocationState{Assignments: assignment}
		if viable, _ := IsStateViable(problem, partial); viable && IsCompleteState(problem, partial) {
			result := types.SolverResult{Status: "optimal", Assignments: assignment, Score: ScoreAllocation(problem, partial)}
			if incumbent.update(result, incumbent.taskIndex) {
				frontier.metrics.BestUpdates++
				frontier.note(fmt.Sprintf("initial min-cost incumbent: score=%d", result.Score.TotalPenalty))
			}
		}
	}

	frontier.expand(&root, normalizedOptions.ParallelDepth, rootWeight)
	if len(frontier.tasks) == 0 {
		if best := incumbent.snapshot(); best != nil {
			best.Metrics = frontier.metrics
			best.DebugNotes = append([]string(nil), frontier.debugNotes...)
			return *best
		}
		result := buildInfeasibleResult(frontier.hardWitness)
		result.Metrics = frontier.metrics
		result.DebugNotes = append([]string(nil), frontier.debugNotes...)
		return result
	}

	results := make([]subtreeOutcome, len(frontier.tasks))
	parallelize := normalizedOptions.WorkerCount > 1 && len(frontier.tasks) > 1
	if parallelize {
		jobs := make(chan int)
		workerCount := normalizedOptions.WorkerCount
		if workerCount > len(frontier.tasks) {
			workerCount = len(frontier.tasks)
		}
		var wg sync.WaitGroup
		for worker := 0; worker < workerCount; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for index := range jobs {
					results[index] = solvePreparedTask(view, normalizedOptions, frontier.tasks[index], index, progress, incumbent)
				}
			}()
		}
		for index := range frontier.tasks {
			jobs <- index
		}
		close(jobs)
		wg.Wait()
	} else {
		for index, task := range frontier.tasks {
			results[index] = solvePreparedTask(view, normalizedOptions, task, index, progress, incumbent)
		}
	}

	finalMetrics := frontier.metrics
	finalMetrics.ParallelTasks = len(frontier.tasks)
	finalNotes := append([]string(nil), frontier.debugNotes...)
	finalWitness := append([]types.HardConstraintViolation(nil), frontier.hardWitness...)
	for _, outcome := range results {
		finalMetrics = mergeSolverMetrics(finalMetrics, outcome.metrics)
		finalNotes = appendLimitedNotes(finalNotes, outcome.debugNotes, normalizedOptions.MaxDebugEvents)
		if len(finalWitness) == 0 && len(outcome.hardWitness) > 0 {
			finalWitness = append([]types.HardConstraintViolation(nil), outcome.hardWitness...)
		}
	}

	best := incumbent.snapshot()
	if best == nil {
		result := buildInfeasibleResult(finalWitness)
		result.Metrics = finalMetrics
		result.DebugNotes = finalNotes
		return result
	}
	best.Metrics = finalMetrics
	best.DebugNotes = finalNotes
	return *best
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
	if normalized.FlowBoundDepth <= 0 {
		normalized.FlowBoundDepth = defaultFlowBoundDepth
	}
	return normalized
}

// EstimateLowerBound returns a safe optimistic score for a partial state.
func EstimateLowerBound(problem types.AllocationProblem, partial types.PartialAllocationState) int {
	view := buildSolverProblemView(problem)
	state := stateFromPartial(view, partial)
	bound, feasible := simpleLowerBound(view, &state)
	if !feasible {
		return invalidStatePenalty
	}
	return bound
}

func buildSolverProblemView(problem types.AllocationProblem) solverProblemView {
	view := solverProblemView{
		problem:                problem,
		candidates:             candidateByID(problem),
		groups:                 groupByID(problem),
		candidateIndexByID:     make(map[int]int, len(problem.Candidates)),
		groupIndexByID:         make(map[int]int, len(problem.Groups)),
		candidateGroupOptions:  make(map[int][]solverGroupOption, len(problem.Candidates)),
		assignmentPenaltyByKey: make(map[int]map[int]int, len(problem.Candidates)),
		groupSymmetryClass:     make(map[int]int, len(problem.Groups)),
	}
	for _, candidate := range problem.Candidates {
		view.sortedCandidateIDs = append(view.sortedCandidateIDs, candidate.ID)
	}
	for _, group := range problem.Groups {
		view.sortedGroupIDs = append(view.sortedGroupIDs, group.ID)
	}
	sort.Ints(view.sortedCandidateIDs)
	sort.Ints(view.sortedGroupIDs)
	for index, candidateID := range view.sortedCandidateIDs {
		view.candidateIndexByID[candidateID] = index
	}
	for index, groupID := range view.sortedGroupIDs {
		view.groupIndexByID[groupID] = index
	}

	for _, candidateID := range view.sortedCandidateIDs {
		candidate := view.candidates[candidateID]
		options := make([]solverGroupOption, 0, len(problem.Groups))
		penalties := make(map[int]int, len(problem.Groups))
		for _, groupID := range view.sortedGroupIDs {
			group := view.groups[groupID]
			if problem.HardRestrictions.RespectCandidatePreferences && !containsInt(candidate.PreferredGroupIDs, group.ID) {
				continue
			}
			if problem.HardRestrictions.EnforceForbiddenEvaluators && groupHasForbiddenEvaluator(candidate, group) {
				continue
			}
			penalty := assignmentPenalty(problem, candidate, group)
			options = append(options, solverGroupOption{
				GroupID: group.ID, GroupIndex: view.groupIndexByID[group.ID], ImmediatePenalty: penalty,
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
		view.candidateGroupOptions[candidateID] = options
		view.assignmentPenaltyByKey[candidateID] = penalties
	}
	view.groupSymmetryClass = buildGroupSymmetryClasses(view)
	return view
}

func newSolverState(view solverProblemView) solverState {
	return solverState{Assignments: make([]int, len(view.sortedCandidateIDs)), GroupCounts: make([]int, len(view.sortedGroupIDs))}
}

func stateFromPartial(view solverProblemView, partial types.PartialAllocationState) solverState {
	state := newSolverState(view)
	for candidateID, groupID := range partial.Assignments {
		candidateIndex, candidateOK := view.candidateIndexByID[candidateID]
		groupIndex, groupOK := view.groupIndexByID[groupID]
		if !candidateOK || !groupOK {
			continue
		}
		state.Assignments[candidateIndex] = groupID
		state.GroupCounts[groupIndex]++
		state.AssignedCount++
		state.BasePenalty += view.assignmentPenaltyByKey[candidateID][groupID]
	}
	return state
}

func solvePreparedTask(view solverProblemView, options SolverOptions, task preparedTask, taskIndex int, progress *solverProgressTracker, incumbent *sharedIncumbent) subtreeOutcome {
	solver := subtreeSolver{view: view, options: options, taskIndex: taskIndex, progress: progress, incumbent: incumbent}
	solver.resume(&task.State, task.Weight)
	return subtreeOutcome{metrics: solver.metrics, debugNotes: append([]string(nil), solver.debugNotes...), hardWitness: append([]types.HardConstraintViolation(nil), solver.hardWitness...)}
}

func (builder *buildFrontierContext) expand(state *solverState, depth int, weight *big.Int) {
	builder.metrics.NodesVisited++
	builder.progress.visitNode()
	if violation := futureHardViolation(builder.view, state); violation != nil {
		builder.metrics.NodesPrunedByHard++
		builder.captureWitness([]types.HardConstraintViolation{*violation})
		builder.note("hard prune before frontier expansion: " + violation.Code)
		builder.progress.resolve(weight, true)
		return
	}
	if depth == 0 || state.AssignedCount == len(builder.view.sortedCandidateIDs) {
		builder.tasks = append(builder.tasks, preparedTask{State: cloneSolverState(*state), Weight: new(big.Int).Set(weight)})
		return
	}

	candidateID, options, symmetrySkipped := chooseNextCandidate(builder.view, state)
	builder.metrics.BranchesSkippedBySymmetry += symmetrySkipped
	if candidateID == 0 || len(options) == 0 {
		violation := candidateWithoutGroupViolation(candidateID)
		builder.metrics.NodesPrunedByHard++
		builder.captureWitness([]types.HardConstraintViolation{violation})
		builder.progress.resolve(weight, true)
		return
	}
	childWeight := builder.resolveSkippedBranches(candidateID, len(options), weight)
	for _, option := range options {
		applyAssignment(builder.view, state, candidateID, option)
		builder.expand(state, depth-1, childWeight)
		undoAssignment(builder.view, state, candidateID, option)
	}
}

func (builder *buildFrontierContext) resolveSkippedBranches(candidateID, feasibleCount int, weight *big.Int) *big.Int {
	staticCount := len(builder.view.candidateGroupOptions[candidateID])
	childWeight := childBranchWeight(weight, staticCount)
	if skipped := staticCount - feasibleCount; skipped > 0 {
		builder.progress.resolve(new(big.Int).Mul(new(big.Int).Set(childWeight), big.NewInt(int64(skipped))), true)
	}
	return childWeight
}

func (solver *subtreeSolver) search(state *solverState, weight *big.Int) {
	solver.metrics.NodesVisited++
	solver.progress.visitNode()
	if violation := futureHardViolation(solver.view, state); violation != nil {
		solver.metrics.NodesPrunedByHard++
		solver.captureWitness([]types.HardConstraintViolation{*violation})
		solver.note("hard prune: " + violation.Code)
		solver.progress.resolve(weight, true)
		return
	}
	solver.resume(state, weight)
}

func (solver *subtreeSolver) resume(state *solverState, weight *big.Int) {
	if state.AssignedCount == len(solver.view.sortedCandidateIDs) {
		solver.evaluateCompleteState(state, weight)
		return
	}

	lowerBound, feasible := simpleLowerBound(solver.view, state)
	if !feasible {
		solver.metrics.NodesPrunedByHard++
		solver.progress.resolve(weight, true)
		return
	}
	if solver.incumbent.shouldPrune(lowerBound, solver.taskIndex) {
		solver.metrics.NodesPrunedByBound++
		solver.note(fmt.Sprintf("bound prune: lower_bound=%d", lowerBound))
		solver.progress.resolve(weight, true)
		return
	}
	flowUsed := state.AssignedCount <= solver.options.FlowBoundDepth || len(solver.view.sortedCandidateIDs)-state.AssignedCount <= 8
	if incumbentScore, exists := solver.incumbent.score(); exists && lowerBound <= incumbentScore {
		flowUsed = true
	}
	if flowUsed {
		flowCost, _, flowFeasible := minCostCompletion(solver.view, state)
		if !flowFeasible {
			solver.metrics.NodesPrunedByFlow++
			solver.progress.resolve(weight, true)
			return
		}
		criterionBound := criterionLowerBound(solver.view, state)
		if flowBound := state.BasePenalty + flowCost + criterionBound; flowBound > lowerBound {
			lowerBound = flowBound
		}
	}
	if solver.incumbent.shouldPrune(lowerBound, solver.taskIndex) {
		if flowUsed {
			solver.metrics.NodesPrunedByFlow++
		} else {
			solver.metrics.NodesPrunedByBound++
		}
		solver.note(fmt.Sprintf("bound prune: lower_bound=%d", lowerBound))
		solver.progress.resolve(weight, true)
		return
	}

	candidateID, options, symmetrySkipped := chooseNextCandidate(solver.view, state)
	solver.metrics.BranchesSkippedBySymmetry += symmetrySkipped
	if candidateID == 0 || len(options) == 0 {
		violation := candidateWithoutGroupViolation(candidateID)
		solver.metrics.NodesPrunedByHard++
		solver.captureWitness([]types.HardConstraintViolation{violation})
		solver.progress.resolve(weight, true)
		return
	}
	staticCount := len(solver.view.candidateGroupOptions[candidateID])
	childWeight := childBranchWeight(weight, staticCount)
	if skipped := staticCount - len(options); skipped > 0 {
		solver.progress.resolve(new(big.Int).Mul(new(big.Int).Set(childWeight), big.NewInt(int64(skipped))), true)
	}
	for _, option := range options {
		applyAssignment(solver.view, state, candidateID, option)
		solver.search(state, childWeight)
		undoAssignment(solver.view, state, candidateID, option)
	}
}

func (solver *subtreeSolver) evaluateCompleteState(state *solverState, weight *big.Int) {
	solver.metrics.CompleteStates++
	assignments := assignmentsMap(solver.view, state)
	partial := types.PartialAllocationState{Assignments: assignments}
	if viable, violations := IsStateViable(solver.view.problem, partial); !viable {
		solver.metrics.NodesPrunedByHard++
		solver.captureWitness(violations)
		solver.progress.resolve(weight, true)
		return
	}
	result := types.SolverResult{Status: "optimal", Assignments: assignments, Score: ScoreAllocation(solver.view.problem, partial)}
	if solver.incumbent.update(result, solver.taskIndex) {
		solver.metrics.BestUpdates++
		solver.note(fmt.Sprintf("best update: score=%d assignments=%s", result.Score.TotalPenalty, formatAssignments(assignments)))
	}
	solver.progress.resolve(weight, false)
}

func simpleLowerBound(view solverProblemView, state *solverState) (int, bool) {
	total := state.BasePenalty
	for candidateIndex, candidateID := range view.sortedCandidateIDs {
		if state.Assignments[candidateIndex] != 0 {
			continue
		}
		options, _ := currentFeasibleOptions(view, state, candidateID, false)
		if len(options) == 0 {
			return 0, false
		}
		minimum := options[0].ImmediatePenalty
		for _, option := range options[1:] {
			if option.ImmediatePenalty < minimum {
				minimum = option.ImmediatePenalty
			}
		}
		total += minimum
	}
	return total + criterionLowerBound(view, state), true
}

func criterionLowerBound(view solverProblemView, state *solverState) int {
	if len(view.problem.SoftRules.Criteria) == 0 {
		return 0
	}
	partial := types.PartialAllocationState{Assignments: assignmentsMap(view, state)}
	total := 0
	for _, criterion := range view.problem.SoftRules.Criteria {
		switch criterion.Type {
		case types.SoftCriterionMaxValue:
			total += lowerBoundMaxValueCriterion(view, partial, criterion)
		case types.SoftCriterionGroupTogether:
			total += lowerBoundGroupTogetherCriterion(view, partial, criterion)
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
		if count := counts[group.ID][selectedValue]; count > criterion.Threshold {
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
	groupsWithValues := make(map[int]struct{})
	for groupID, members := range buildStateGroupMembers(state) {
		for _, candidateID := range members {
			if _, ok := selectedValues[view.candidates[candidateID].Attributes[criterion.ColumnKey]]; ok {
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

func futureHardViolation(view solverProblemView, state *solverState) *types.HardConstraintViolation {
	remaining := len(view.sortedCandidateIDs) - state.AssignedCount
	if view.problem.HardRestrictions.AllCandidatesMustBeAssigned && remaining > remainingCapacity(view, state) {
		violation := insufficientCapacityViolation(remaining, remainingCapacity(view, state))
		return &violation
	}
	for candidateIndex, candidateID := range view.sortedCandidateIDs {
		if state.Assignments[candidateIndex] != 0 {
			continue
		}
		if options, _ := currentFeasibleOptions(view, state, candidateID, false); len(options) == 0 {
			violation := candidateWithoutGroupViolation(candidateID)
			return &violation
		}
	}
	for groupIndex, groupID := range view.sortedGroupIDs {
		group := view.groups[groupID]
		current := state.GroupCounts[groupIndex]
		if current == 0 || current >= group.MinCandidates {
			continue
		}
		possible := current + countPotentialCandidatesForGroup(view, state, groupID)
		if possible < group.MinCandidates {
			violation := groupCannotReachMinimumViolation(groupID, group.MinCandidates, possible)
			return &violation
		}
	}
	return nil
}

func chooseNextCandidate(view solverProblemView, state *solverState) (int, []solverGroupOption, int) {
	bestCandidateID := 0
	var bestOptions []solverGroupOption
	bestSymmetrySkipped := 0
	for candidateIndex, candidateID := range view.sortedCandidateIDs {
		if state.Assignments[candidateIndex] != 0 {
			continue
		}
		options, symmetrySkipped := currentFeasibleOptions(view, state, candidateID, true)
		if bestCandidateID == 0 || len(options) < len(bestOptions) || (len(options) == len(bestOptions) && candidateID < bestCandidateID) {
			bestCandidateID, bestOptions, bestSymmetrySkipped = candidateID, options, symmetrySkipped
		}
		if len(options) == 0 {
			break
		}
	}
	return bestCandidateID, bestOptions, bestSymmetrySkipped
}

func currentFeasibleOptions(view solverProblemView, state *solverState, candidateID int, breakSymmetry bool) ([]solverGroupOption, int) {
	options := view.candidateGroupOptions[candidateID]
	result := make([]solverGroupOption, 0, len(options))
	seenSymmetryState := make(map[[2]int]struct{})
	skipped := 0
	for _, option := range options {
		if view.problem.HardRestrictions.EnforceGroupCapacity && state.GroupCounts[option.GroupIndex] >= view.groups[option.GroupID].MaxCandidates {
			continue
		}
		if breakSymmetry {
			key := [2]int{view.groupSymmetryClass[option.GroupID], state.GroupCounts[option.GroupIndex]}
			if _, exists := seenSymmetryState[key]; exists {
				skipped++
				continue
			}
			seenSymmetryState[key] = struct{}{}
		}
		result = append(result, option)
	}
	return result, skipped
}

func buildGroupSymmetryClasses(view solverProblemView) map[int]int {
	classes := make(map[int]int, len(view.sortedGroupIDs))
	signatureClass := make(map[string]int)
	for _, groupID := range view.sortedGroupIDs {
		group := view.groups[groupID]
		parts := []string{group.ScheduleKey, fmt.Sprint(group.MinCandidates), fmt.Sprint(group.MaxCandidates), fmt.Sprint(group.EvaluatorIDs)}
		for _, candidateID := range view.sortedCandidateIDs {
			penalty, allowed := view.assignmentPenaltyByKey[candidateID][groupID]
			parts = append(parts, fmt.Sprintf("%t:%d", allowed, penalty))
		}
		signature := strings.Join(parts, "|")
		class, ok := signatureClass[signature]
		if !ok {
			class = len(signatureClass) + 1
			signatureClass[signature] = class
		}
		classes[groupID] = class
	}
	return classes
}

func applyAssignment(view solverProblemView, state *solverState, candidateID int, option solverGroupOption) {
	state.Assignments[view.candidateIndexByID[candidateID]] = option.GroupID
	state.GroupCounts[option.GroupIndex]++
	state.AssignedCount++
	state.BasePenalty += option.ImmediatePenalty
}

func undoAssignment(view solverProblemView, state *solverState, candidateID int, option solverGroupOption) {
	state.BasePenalty -= option.ImmediatePenalty
	state.AssignedCount--
	state.GroupCounts[option.GroupIndex]--
	state.Assignments[view.candidateIndexByID[candidateID]] = 0
}

func cloneSolverState(state solverState) solverState {
	return solverState{Assignments: append([]int(nil), state.Assignments...), GroupCounts: append([]int(nil), state.GroupCounts...), AssignedCount: state.AssignedCount, BasePenalty: state.BasePenalty}
}

func assignmentsMap(view solverProblemView, state *solverState) map[int]int {
	assignments := make(map[int]int, state.AssignedCount)
	for candidateIndex, groupID := range state.Assignments {
		if groupID != 0 {
			assignments[view.sortedCandidateIDs[candidateIndex]] = groupID
		}
	}
	return assignments
}

func assignmentPenalty(problem types.AllocationProblem, candidate types.SolverCandidate, group types.SolverGroup) int {
	rank := candidatePreferenceRank(candidate, group.ID)
	penalty := preferencePenaltyForRank(problem.SoftRules.PreferencePenaltyByRank, rank)
	return penalty + countAvoidEvaluatorConflicts(candidate, group)*problem.SoftRules.AvoidEvaluatorPenalty
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

func remainingCapacity(view solverProblemView, state *solverState) int {
	if !view.problem.HardRestrictions.EnforceGroupCapacity {
		return len(view.sortedCandidateIDs)
	}
	remaining := 0
	for groupIndex, groupID := range view.sortedGroupIDs {
		remaining += view.groups[groupID].MaxCandidates - state.GroupCounts[groupIndex]
	}
	return remaining
}

func countPotentialCandidatesForGroup(view solverProblemView, state *solverState, groupID int) int {
	count := 0
	groupIndex := view.groupIndexByID[groupID]
	if state.GroupCounts[groupIndex] >= view.groups[groupID].MaxCandidates {
		return 0
	}
	for candidateIndex, candidateID := range view.sortedCandidateIDs {
		if state.Assignments[candidateIndex] == 0 {
			if _, allowed := view.assignmentPenaltyByKey[candidateID][groupID]; allowed {
				count++
			}
		}
	}
	return count
}

func (incumbent *sharedIncumbent) update(result types.SolverResult, taskIndex int) bool {
	incumbent.mu.Lock()
	defer incumbent.mu.Unlock()
	if incumbent.best != nil {
		if result.Score.TotalPenalty > incumbent.best.Score.TotalPenalty {
			return false
		}
		if result.Score.TotalPenalty == incumbent.best.Score.TotalPenalty && taskIndex >= incumbent.taskIndex {
			return false
		}
	}
	copied := cloneSolverResult(result)
	incumbent.best = &copied
	incumbent.taskIndex = taskIndex
	return true
}

func (incumbent *sharedIncumbent) shouldPrune(lowerBound, taskIndex int) bool {
	incumbent.mu.RLock()
	defer incumbent.mu.RUnlock()
	if incumbent.best == nil {
		return false
	}
	if lowerBound > incumbent.best.Score.TotalPenalty {
		return true
	}
	return lowerBound == incumbent.best.Score.TotalPenalty && incumbent.taskIndex <= taskIndex
}

func (incumbent *sharedIncumbent) snapshot() *types.SolverResult {
	incumbent.mu.RLock()
	defer incumbent.mu.RUnlock()
	if incumbent.best == nil {
		return nil
	}
	copied := cloneSolverResult(*incumbent.best)
	return &copied
}

func (incumbent *sharedIncumbent) score() (int, bool) {
	incumbent.mu.RLock()
	defer incumbent.mu.RUnlock()
	if incumbent.best == nil {
		return 0, false
	}
	return incumbent.best.Score.TotalPenalty, true
}

func mergeSolverMetrics(left, right types.SolverMetrics) types.SolverMetrics {
	return types.SolverMetrics{
		NodesVisited: left.NodesVisited + right.NodesVisited, CompleteStates: left.CompleteStates + right.CompleteStates,
		NodesPrunedByHard: left.NodesPrunedByHard + right.NodesPrunedByHard, NodesPrunedByBound: left.NodesPrunedByBound + right.NodesPrunedByBound,
		NodesPrunedByFlow: left.NodesPrunedByFlow + right.NodesPrunedByFlow, BranchesSkippedBySymmetry: left.BranchesSkippedBySymmetry + right.BranchesSkippedBySymmetry,
		BestUpdates: left.BestUpdates + right.BestUpdates, ParallelTasks: left.ParallelTasks + right.ParallelTasks,
	}
}

func cloneAssignments(assignments map[int]int) map[int]int {
	cloned := make(map[int]int, len(assignments))
	for candidateID, groupID := range assignments {
		cloned[candidateID] = groupID
	}
	return cloned
}

func cloneSolverResult(result types.SolverResult) types.SolverResult {
	result.Assignments = cloneAssignments(result.Assignments)
	result.Score.Components = append([]types.SoftScoreComponent(nil), result.Score.Components...)
	result.HardViolations = append([]types.HardConstraintViolation(nil), result.HardViolations...)
	result.DebugNotes = append([]string(nil), result.DebugNotes...)
	return result
}

func buildInfeasibleResult(violations []types.HardConstraintViolation) types.SolverResult {
	result := types.SolverResult{Status: "infeasible", Assignments: map[int]int{}, HardViolations: append([]types.HardConstraintViolation(nil), violations...), DebugNotes: []string{}}
	if len(violations) == 0 {
		result.RejectionReason = "nenhuma solucao hard valida foi encontrada"
	} else {
		result.RejectionReason = violations[0].Message
	}
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

func (builder *buildFrontierContext) captureWitness(violations []types.HardConstraintViolation) {
	if len(builder.hardWitness) == 0 && len(violations) > 0 {
		builder.hardWitness = append([]types.HardConstraintViolation(nil), violations...)
	}
}

func (builder *buildFrontierContext) note(message string) {
	builder.debugNotes = appendSingleNote(builder.debugNotes, builder.options.MaxDebugEvents, message)
}

func (solver *subtreeSolver) captureWitness(violations []types.HardConstraintViolation) {
	if len(solver.hardWitness) == 0 && len(violations) > 0 {
		solver.hardWitness = append([]types.HardConstraintViolation(nil), violations...)
	}
}

func (solver *subtreeSolver) note(message string) {
	solver.debugNotes = appendSingleNote(solver.debugNotes, solver.options.MaxDebugEvents, message)
}

func candidateWithoutGroupViolation(candidateID int) types.HardConstraintViolation {
	return types.HardConstraintViolation{Code: "candidate_without_feasible_group", Message: fmt.Sprintf("candidato %d nao possui grupo viavel restante", candidateID), CandidateID: candidateID}
}

func insufficientCapacityViolation(required, available int) types.HardConstraintViolation {
	return types.HardConstraintViolation{Code: "insufficient_remaining_capacity", Message: fmt.Sprintf("capacidade restante insuficiente: faltam %d candidatos para %d vagas", required, available)}
}

func groupCannotReachMinimumViolation(groupID, minimum, possible int) types.HardConstraintViolation {
	return types.HardConstraintViolation{Code: "group_cannot_reach_min_candidates", Message: fmt.Sprintf("grupo %d nao consegue atingir o minimo %d; maximo possivel %d", groupID, minimum, possible), GroupID: groupID}
}
