package allocation

import (
	types "candidate_alocator/back/type"
	"fmt"
)

// ReviewManualAllocation uses the same solver contracts to describe manual
// exceptions. It reports constraints without rejecting an administrator's move.
func ReviewManualAllocation(problem types.AllocationProblem, assignments map[int]int, expectedEvaluators int) types.SolverResult {
	state := types.PartialAllocationState{Assignments: assignments}
	violations := CheckHardConstraints(problem, state)
	seen := map[string]int{}
	members := buildStateGroupMembers(state)
	for _, g := range problem.Groups {
		if len(members[g.ID]) == 0 {
			continue
		}
		if len(g.EvaluatorIDs) < expectedEvaluators {
			violations = append(violations, types.HardConstraintViolation{Code: "missing_evaluators", GroupID: g.ID, Message: fmt.Sprintf("Mesa %d: equipe de avaliadores incompleta", g.ID)})
		}
		for _, id := range g.EvaluatorIDs {
			key := fmt.Sprintf("%s/%d", g.ScheduleKey, id)
			if prior, ok := seen[key]; ok && prior != g.ID {
				violations = append(violations, types.HardConstraintViolation{Code: "evaluator_overlap", GroupID: g.ID, EvaluatorID: id, Message: fmt.Sprintf("Avaliador %d em duas mesas no horário %s", id, g.ScheduleKey)})
			}
			seen[key] = g.ID
		}
	}
	return types.SolverResult{Status: "manual", Assignments: assignments, Score: scoreAssignedAllocation(problem, state), HardViolations: violations}
}
