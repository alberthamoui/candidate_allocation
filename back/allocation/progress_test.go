package allocation

import (
	"math/big"
	"testing"
	"time"

	types "candidate_alocator/back/type"
)

func TestProgressCapturesFirstAndSecondMinuteThroughput(t *testing.T) {
	startedAt := time.Date(2026, time.August, 14, 10, 0, 0, 0, time.UTC)
	tracker := &solverProgressTracker{
		callback:       func(SolverProgress) {},
		total:          big.NewInt(1000),
		resolved:       big.NewInt(120),
		pruned:         new(big.Int),
		nodesVisited:   10,
		startedAt:      startedAt,
		firstResolved:  new(big.Int),
		secondResolved: new(big.Int),
	}

	first := tracker.reportLocked(startedAt.Add(throughputWindow), true).snapshot
	if !first.FirstMinuteComplete || first.FirstMinuteBranchesResolved != "120" || first.FirstMinuteNodesVisited != 10 {
		t.Fatalf("unexpected first-minute throughput: %#v", first)
	}
	if first.SecondMinuteComplete {
		t.Fatalf("second minute must still be incomplete: %#v", first)
	}

	tracker.resolved.SetInt64(170)
	tracker.nodesVisited = 18
	second := tracker.reportLocked(startedAt.Add(2*throughputWindow), true).snapshot
	if !second.SecondMinuteComplete || second.SecondMinuteBranchesResolved != "50" || second.SecondMinuteNodesVisited != 8 {
		t.Fatalf("unexpected second-minute throughput: %#v", second)
	}
}

func TestProgressCallbackDoesNotHoldSolverStateMutex(t *testing.T) {
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	visitReturned := make(chan struct{})
	tracker := &solverProgressTracker{
		callback: func(SolverProgress) {
			close(callbackStarted)
			<-releaseCallback
		},
		total:          big.NewInt(10),
		resolved:       new(big.Int),
		pruned:         new(big.Int),
		startedAt:      time.Now(),
		firstResolved:  new(big.Int),
		secondResolved: new(big.Int),
	}

	go func() {
		tracker.visitNode()
		close(visitReturned)
	}()
	<-callbackStarted

	resolveReturned := make(chan struct{})
	go func() {
		tracker.resolve(big.NewInt(1), true)
		close(resolveReturned)
	}()
	select {
	case <-resolveReturned:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("progress callback blocked the solver-state mutex")
	}
	close(releaseCallback)
	<-visitReturned
}

func TestIncumbentCallbackDoesNotBlockBoundChecks(t *testing.T) {
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	updateReturned := make(chan struct{})
	incumbent := &sharedIncumbent{
		taskIndex: 100,
		callback: func(types.SolverResult) {
			close(callbackStarted)
			<-releaseCallback
		},
	}

	go func() {
		incumbent.update(types.SolverResult{Score: types.SoftScoreBreakdown{TotalPenalty: 5}}, 0)
		close(updateReturned)
	}()
	<-callbackStarted

	boundReturned := make(chan struct{})
	go func() {
		_ = incumbent.shouldPrune(5, 0)
		close(boundReturned)
	}()
	select {
	case <-boundReturned:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("incumbent callback blocked solver bound checks")
	}
	close(releaseCallback)
	<-updateReturned
}

func TestIncumbentPublishesAndCountsOnlyStrictScoreImprovements(t *testing.T) {
	callbackScores := make([]int, 0)
	incumbent := &sharedIncumbent{
		taskIndex: 100,
		callback: func(result types.SolverResult) {
			callbackScores = append(callbackScores, result.Score.TotalPenalty)
		},
	}

	if !incumbent.update(types.SolverResult{Assignments: map[int]int{1: 1}, Score: types.SoftScoreBreakdown{TotalPenalty: 10}}, 5) {
		t.Fatal("first feasible solution must be published")
	}
	if incumbent.update(types.SolverResult{Assignments: map[int]int{1: 2}, Score: types.SoftScoreBreakdown{TotalPenalty: 10}}, 2) {
		t.Fatal("equal score must not count as an improvement")
	}
	if got := incumbent.snapshot().Assignments[1]; got != 1 {
		t.Fatalf("equal score must not replace the published allocation, got group %d", got)
	}
	if incumbent.update(types.SolverResult{Score: types.SoftScoreBreakdown{TotalPenalty: 11}}, 1) {
		t.Fatal("worse score must not count as an improvement")
	}
	if !incumbent.update(types.SolverResult{Score: types.SoftScoreBreakdown{TotalPenalty: 9}}, 4) {
		t.Fatal("lower score must be published")
	}
	if len(callbackScores) != 2 || callbackScores[0] != 10 || callbackScores[1] != 9 {
		t.Fatalf("unexpected published scores: %#v", callbackScores)
	}
}
