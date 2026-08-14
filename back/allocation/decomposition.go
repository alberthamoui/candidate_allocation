package allocation

import (
	"math/big"
	"sort"
	"sync"
	"time"

	types "candidate_alocator/back/type"
)

type solverComponent struct {
	candidateIDs []int
	groupIDs     []int
}

type aggregateProgress struct {
	mu             sync.Mutex
	callback       ProgressCallback
	startedAt      time.Time
	totals         []*big.Int
	resolved       []*big.Int
	pruned         []*big.Int
	nodes          []int
	subtrees       []int
	firstResolved  *big.Int
	firstNodes     int
	firstComplete  bool
	secondResolved *big.Int
	secondNodes    int
	secondComplete bool
}

// SolveAllocation solves every independent component exactly and combines
// their proven optima. Coupled additional criteria keep the problem unified.
func SolveAllocation(problem types.AllocationProblem, options SolverOptions) types.SolverResult {
	if options.Incumbent != nil {
		return solveConnectedAllocation(problem, options)
	}
	components := independentSolverComponents(problem)
	if len(components) <= 1 {
		return solveConnectedAllocation(problem, options)
	}

	progress := newAggregateProgress(problem, components, options.Progress)
	combinedAssignments := make(map[int]int, len(problem.Candidates))
	combinedMetrics := types.SolverMetrics{}
	combinedNotes := make([]string, 0)
	for componentIndex, component := range components {
		componentProblem := buildComponentProblem(problem, component)
		componentOptions := options
		componentOptions.Progress = progress.callbackFor(componentIndex)
		result := solveConnectedAllocation(componentProblem, componentOptions)
		combinedMetrics = mergeSolverMetrics(combinedMetrics, result.Metrics)
		combinedNotes = appendLimitedNotes(combinedNotes, result.DebugNotes, NormalizeSolverOptions(options).MaxDebugEvents)
		if result.Status != "optimal" {
			result.Metrics = combinedMetrics
			result.DebugNotes = combinedNotes
			return result
		}
		for candidateID, groupID := range result.Assignments {
			combinedAssignments[candidateID] = groupID
		}
	}

	partial := types.PartialAllocationState{Assignments: combinedAssignments}
	return types.SolverResult{
		Status:      "optimal",
		Assignments: combinedAssignments,
		Score:       ScoreAllocation(problem, partial),
		Metrics:     combinedMetrics,
		DebugNotes:  combinedNotes,
	}
}

func independentSolverComponents(problem types.AllocationProblem) []solverComponent {
	if hasCoupledSoftCriteria(problem.SoftRules.Criteria) || len(problem.Candidates) == 0 || !problem.HardRestrictions.RespectCandidatePreferences {
		return []solverComponent{{}}
	}
	groupToCandidates := make(map[int][]int, len(problem.Groups))
	candidateToGroups := make(map[int][]int, len(problem.Candidates))
	for _, candidate := range problem.Candidates {
		for _, groupID := range candidate.PreferredGroupIDs {
			candidateToGroups[candidate.ID] = append(candidateToGroups[candidate.ID], groupID)
			groupToCandidates[groupID] = append(groupToCandidates[groupID], candidate.ID)
		}
	}

	visitedCandidates := make(map[int]bool, len(problem.Candidates))
	components := make([]solverComponent, 0)
	for _, candidate := range problem.Candidates {
		if visitedCandidates[candidate.ID] {
			continue
		}
		component := solverComponent{}
		candidateQueue := []int{candidate.ID}
		visitedGroups := make(map[int]bool)
		visitedCandidates[candidate.ID] = true
		for len(candidateQueue) > 0 {
			candidateID := candidateQueue[0]
			candidateQueue = candidateQueue[1:]
			component.candidateIDs = append(component.candidateIDs, candidateID)
			for _, groupID := range candidateToGroups[candidateID] {
				if visitedGroups[groupID] {
					continue
				}
				visitedGroups[groupID] = true
				component.groupIDs = append(component.groupIDs, groupID)
				for _, nextCandidateID := range groupToCandidates[groupID] {
					if !visitedCandidates[nextCandidateID] {
						visitedCandidates[nextCandidateID] = true
						candidateQueue = append(candidateQueue, nextCandidateID)
					}
				}
			}
		}
		sort.Ints(component.candidateIDs)
		sort.Ints(component.groupIDs)
		components = append(components, component)
	}
	return components
}

func hasCoupledSoftCriteria(criteria []types.SoftCriterion) bool {
	for _, criterion := range criteria {
		if criterion.Type == types.SoftCriterionBalancedDistribution || criterion.Type == types.SoftCriterionGroupTogether {
			return true
		}
	}
	return false
}

func buildComponentProblem(problem types.AllocationProblem, component solverComponent) types.AllocationProblem {
	candidateSet := make(map[int]bool, len(component.candidateIDs))
	groupSet := make(map[int]bool, len(component.groupIDs))
	for _, candidateID := range component.candidateIDs {
		candidateSet[candidateID] = true
	}
	for _, groupID := range component.groupIDs {
		groupSet[groupID] = true
	}
	componentProblem := problem
	componentProblem.Candidates = nil
	componentProblem.Groups = nil
	for _, candidate := range problem.Candidates {
		if candidateSet[candidate.ID] {
			componentProblem.Candidates = append(componentProblem.Candidates, candidate)
		}
	}
	for _, group := range problem.Groups {
		if groupSet[group.ID] {
			componentProblem.Groups = append(componentProblem.Groups, group)
		}
	}
	return componentProblem
}

func newAggregateProgress(problem types.AllocationProblem, components []solverComponent, callback ProgressCallback) *aggregateProgress {
	aggregate := &aggregateProgress{
		callback: callback, totals: make([]*big.Int, len(components)), resolved: make([]*big.Int, len(components)),
		pruned: make([]*big.Int, len(components)), nodes: make([]int, len(components)), subtrees: make([]int, len(components)),
		startedAt: time.Now(), firstResolved: new(big.Int), secondResolved: new(big.Int),
	}
	for index, component := range components {
		view := buildSolverProblemView(buildComponentProblem(problem, component))
		total := big.NewInt(1)
		for _, candidateID := range view.sortedCandidateIDs {
			count := len(view.candidateGroupOptions[candidateID])
			if count == 0 {
				count = 1
			}
			total.Mul(total, big.NewInt(int64(count)))
		}
		aggregate.totals[index] = total
		aggregate.resolved[index] = new(big.Int)
		aggregate.pruned[index] = new(big.Int)
	}
	aggregate.emitLocked()
	return aggregate
}

func (aggregate *aggregateProgress) callbackFor(index int) ProgressCallback {
	if aggregate == nil || aggregate.callback == nil {
		return nil
	}
	return func(snapshot SolverProgress) {
		aggregate.mu.Lock()
		defer aggregate.mu.Unlock()
		aggregate.resolved[index] = parseProgressInteger(snapshot.BranchesResolved)
		aggregate.pruned[index] = parseProgressInteger(snapshot.BranchesPruned)
		aggregate.nodes[index] = snapshot.NodesVisited
		aggregate.subtrees[index] = snapshot.PrunedSubtrees
		aggregate.emitLocked()
	}
}

func (aggregate *aggregateProgress) emitLocked() {
	if aggregate.callback == nil {
		return
	}
	total := new(big.Int)
	resolved := new(big.Int)
	pruned := new(big.Int)
	nodes := 0
	subtrees := 0
	for index := range aggregate.totals {
		total.Add(total, aggregate.totals[index])
		resolved.Add(resolved, aggregate.resolved[index])
		pruned.Add(pruned, aggregate.pruned[index])
		nodes += aggregate.nodes[index]
		subtrees += aggregate.subtrees[index]
	}
	elapsed := time.Since(aggregate.startedAt)
	if !aggregate.firstComplete && elapsed >= throughputWindow {
		aggregate.firstResolved.Set(resolved)
		aggregate.firstNodes = nodes
		aggregate.firstComplete = true
	}
	if !aggregate.secondComplete && elapsed >= 2*throughputWindow {
		aggregate.secondResolved.Sub(resolved, aggregate.firstResolved)
		aggregate.secondNodes = nodes - aggregate.firstNodes
		aggregate.secondComplete = true
	}
	percentage, _ := new(big.Float).Quo(new(big.Float).SetInt(resolved), new(big.Float).SetInt(total)).Float64()
	aggregate.callback(SolverProgress{
		Percent: percentage * 100, BranchesResolved: resolved.String(), TotalBranches: total.String(),
		BranchesPruned: pruned.String(), NodesVisited: nodes, PrunedSubtrees: subtrees,
		FirstMinuteComplete: aggregate.firstComplete, FirstMinuteBranchesResolved: aggregate.firstResolved.String(),
		FirstMinuteNodesVisited: aggregate.firstNodes, SecondMinuteComplete: aggregate.secondComplete,
		SecondMinuteBranchesResolved: aggregate.secondResolved.String(), SecondMinuteNodesVisited: aggregate.secondNodes,
	})
}

func parseProgressInteger(value string) *big.Int {
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return new(big.Int)
	}
	return parsed
}
