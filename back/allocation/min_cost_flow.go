package allocation

import (
	"container/heap"
	"math"
)

type flowEdge struct {
	to       int
	reverse  int
	capacity int
	cost     int
}

type flowQueueItem struct {
	node     int
	distance int
}

type flowPriorityQueue []flowQueueItem

func (queue flowPriorityQueue) Len() int { return len(queue) }
func (queue flowPriorityQueue) Less(i, j int) bool {
	if queue[i].distance != queue[j].distance {
		return queue[i].distance < queue[j].distance
	}
	return queue[i].node < queue[j].node
}
func (queue flowPriorityQueue) Swap(i, j int)   { queue[i], queue[j] = queue[j], queue[i] }
func (queue *flowPriorityQueue) Push(value any) { *queue = append(*queue, value.(flowQueueItem)) }
func (queue *flowPriorityQueue) Pop() any {
	old := *queue
	value := old[len(old)-1]
	*queue = old[:len(old)-1]
	return value
}

func addFlowEdge(graph [][]flowEdge, from, to, capacity, cost int) {
	forward := flowEdge{to: to, reverse: len(graph[to]), capacity: capacity, cost: cost}
	backward := flowEdge{to: from, reverse: len(graph[from]), capacity: 0, cost: -cost}
	graph[from] = append(graph[from], forward)
	graph[to] = append(graph[to], backward)
}

func initialMinCostAssignment(view solverProblemView, state *solverState) (map[int]int, bool) {
	_, assignment, feasible := minCostCompletion(view, state)
	if !feasible {
		return nil, false
	}
	for candidateIndex, groupID := range state.Assignments {
		if groupID != 0 {
			assignment[view.sortedCandidateIDs[candidateIndex]] = groupID
		}
	}
	if repairMinimumGroupSizes(view, assignment) {
		return assignment, true
	}
	return nil, false
}

func repairMinimumGroupSizes(view solverProblemView, assignment map[int]int) bool {
	counts := make(map[int]int, len(view.sortedGroupIDs))
	for _, groupID := range assignment {
		counts[groupID]++
	}
	for _, targetGroupID := range view.sortedGroupIDs {
		target := view.groups[targetGroupID]
		for counts[targetGroupID] > 0 && counts[targetGroupID] < target.MinCandidates {
			bestCandidateID := 0
			bestSourceGroupID := 0
			bestDelta := math.MaxInt
			for _, candidateID := range view.sortedCandidateIDs {
				sourceGroupID := assignment[candidateID]
				if sourceGroupID == targetGroupID {
					continue
				}
				if _, allowed := view.assignmentPenaltyByKey[candidateID][targetGroupID]; !allowed {
					continue
				}
				sourceRemaining := counts[sourceGroupID] - 1
				sourceMinimum := view.groups[sourceGroupID].MinCandidates
				if sourceRemaining != 0 && sourceRemaining < sourceMinimum {
					continue
				}
				delta := view.assignmentPenaltyByKey[candidateID][targetGroupID] - view.assignmentPenaltyByKey[candidateID][sourceGroupID]
				if delta < bestDelta || (delta == bestDelta && candidateID < bestCandidateID) {
					bestCandidateID, bestSourceGroupID, bestDelta = candidateID, sourceGroupID, delta
				}
			}
			if bestCandidateID == 0 {
				return false
			}
			assignment[bestCandidateID] = targetGroupID
			counts[bestSourceGroupID]--
			counts[targetGroupID]++
		}
	}
	return true
}

func minCostCompletion(view solverProblemView, state *solverState) (int, map[int]int, bool) {
	cost, assignment, complete := minCostAssignment(view, state)
	if !complete {
		return 0, nil, false
	}
	return cost, assignment, true
}

// maximumCardinalityMinCostAssignment assigns as many candidates as possible
// while preserving every per-assignment hard rule and group capacity. It is a
// deterministic fallback for problems where a complete hard-valid allocation
// does not exist.
func maximumCardinalityMinCostAssignment(view solverProblemView, state *solverState) map[int]int {
	_, assignment, _ := minCostAssignment(view, state)
	for candidateIndex, groupID := range state.Assignments {
		if groupID != 0 {
			assignment[view.sortedCandidateIDs[candidateIndex]] = groupID
		}
	}
	repairOrDropUnderfilledGroups(view, assignment)
	return assignment
}

func minCostAssignment(view solverProblemView, state *solverState) (int, map[int]int, bool) {
	remainingCandidates := make([]int, 0, len(view.sortedCandidateIDs)-state.AssignedCount)
	for candidateIndex, candidateID := range view.sortedCandidateIDs {
		if state.Assignments[candidateIndex] == 0 {
			remainingCandidates = append(remainingCandidates, candidateID)
		}
	}
	if len(remainingCandidates) == 0 {
		return 0, map[int]int{}, true
	}

	source := 0
	candidateOffset := 1
	groupOffset := candidateOffset + len(remainingCandidates)
	sink := groupOffset + len(view.sortedGroupIDs)
	graph := make([][]flowEdge, sink+1)

	for candidatePosition, candidateID := range remainingCandidates {
		candidateNode := candidateOffset + candidatePosition
		addFlowEdge(graph, source, candidateNode, 1, 0)
		for _, option := range view.candidateGroupOptions[candidateID] {
			if view.problem.HardRestrictions.EnforceGroupCapacity && state.GroupCounts[option.GroupIndex] >= view.groups[option.GroupID].MaxCandidates {
				continue
			}
			addFlowEdge(graph, candidateNode, groupOffset+option.GroupIndex, 1, option.ImmediatePenalty)
		}
	}
	for groupIndex, groupID := range view.sortedGroupIDs {
		capacity := len(remainingCandidates)
		if view.problem.HardRestrictions.EnforceGroupCapacity {
			capacity = view.groups[groupID].MaxCandidates - state.GroupCounts[groupIndex]
		}
		if capacity > 0 {
			addFlowEdge(graph, groupOffset+groupIndex, sink, capacity, 0)
		}
	}

	flow, cost := runMinCostFlow(graph, source, sink, len(remainingCandidates))
	assignment := make(map[int]int, len(remainingCandidates))
	for candidatePosition, candidateID := range remainingCandidates {
		candidateNode := candidateOffset + candidatePosition
		for _, edge := range graph[candidateNode] {
			if edge.to < groupOffset || edge.to >= sink || edge.capacity != 0 {
				continue
			}
			assignment[candidateID] = view.sortedGroupIDs[edge.to-groupOffset]
			break
		}
	}
	return cost, assignment, flow == len(remainingCandidates) && len(assignment) == len(remainingCandidates)
}

// repairOrDropUnderfilledGroups keeps every visible fallback table valid. It
// first tries to fill an under-minimum table with legal moves; if that cannot
// be done, its members become unallocated instead of being shown in an invalid
// table.
func repairOrDropUnderfilledGroups(view solverProblemView, assignment map[int]int) {
	if !view.problem.HardRestrictions.EnforceMinCandidatesOnCompleteState {
		return
	}
	counts := make(map[int]int, len(view.sortedGroupIDs))
	for _, groupID := range assignment {
		counts[groupID]++
	}
	for _, targetGroupID := range view.sortedGroupIDs {
		target := view.groups[targetGroupID]
		for counts[targetGroupID] > 0 && counts[targetGroupID] < target.MinCandidates {
			bestCandidateID := 0
			bestSourceGroupID := 0
			bestDelta := math.MaxInt
			for _, candidateID := range view.sortedCandidateIDs {
				sourceGroupID, assigned := assignment[candidateID]
				if !assigned || sourceGroupID == targetGroupID {
					continue
				}
				if _, allowed := view.assignmentPenaltyByKey[candidateID][targetGroupID]; !allowed {
					continue
				}
				sourceRemaining := counts[sourceGroupID] - 1
				sourceMinimum := view.groups[sourceGroupID].MinCandidates
				if sourceRemaining != 0 && sourceRemaining < sourceMinimum {
					continue
				}
				delta := view.assignmentPenaltyByKey[candidateID][targetGroupID] - view.assignmentPenaltyByKey[candidateID][sourceGroupID]
				if delta < bestDelta || (delta == bestDelta && (bestCandidateID == 0 || candidateID < bestCandidateID)) {
					bestCandidateID, bestSourceGroupID, bestDelta = candidateID, sourceGroupID, delta
				}
			}
			if bestCandidateID == 0 {
				for candidateID, groupID := range assignment {
					if groupID == targetGroupID {
						delete(assignment, candidateID)
					}
				}
				counts[targetGroupID] = 0
				break
			}
			assignment[bestCandidateID] = targetGroupID
			counts[bestSourceGroupID]--
			counts[targetGroupID]++
		}
	}
}

func runMinCostFlow(graph [][]flowEdge, source, sink, requiredFlow int) (int, int) {
	const infinity = math.MaxInt / 4
	potential := make([]int, len(graph))
	flow := 0
	cost := 0
	for flow < requiredFlow {
		distance := make([]int, len(graph))
		previousNode := make([]int, len(graph))
		previousEdge := make([]int, len(graph))
		for index := range distance {
			distance[index] = infinity
			previousNode[index] = -1
		}
		distance[source] = 0
		queue := &flowPriorityQueue{{node: source, distance: 0}}
		heap.Init(queue)
		for queue.Len() > 0 {
			item := heap.Pop(queue).(flowQueueItem)
			if item.distance != distance[item.node] {
				continue
			}
			for edgeIndex, edge := range graph[item.node] {
				if edge.capacity <= 0 {
					continue
				}
				nextDistance := item.distance + edge.cost + potential[item.node] - potential[edge.to]
				if nextDistance >= distance[edge.to] {
					continue
				}
				distance[edge.to] = nextDistance
				previousNode[edge.to] = item.node
				previousEdge[edge.to] = edgeIndex
				heap.Push(queue, flowQueueItem{node: edge.to, distance: nextDistance})
			}
		}
		if previousNode[sink] == -1 {
			break
		}
		for node := range potential {
			if distance[node] < infinity {
				potential[node] += distance[node]
			}
		}
		for node := sink; node != source; node = previousNode[node] {
			from := previousNode[node]
			edgeIndex := previousEdge[node]
			reverse := graph[from][edgeIndex].reverse
			graph[from][edgeIndex].capacity--
			graph[node][reverse].capacity++
		}
		flow++
		cost += potential[sink]
	}
	return flow, cost
}
