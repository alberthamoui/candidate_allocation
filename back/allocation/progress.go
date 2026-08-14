package allocation

import (
	"math/big"
	"sync"
	"time"
)

const (
	progressReportInterval = time.Second
	throughputWindow       = time.Minute
)

// SolverProgress reports how much of the conceptual assignment tree has been
// resolved. A pruned subtree resolves all of its descendant branches at once.
type SolverProgress struct {
	Percent                      float64 `json:"percent"`
	BranchesResolved             string  `json:"branchesResolved"`
	TotalBranches                string  `json:"totalBranches"`
	BranchesPruned               string  `json:"branchesPruned"`
	NodesVisited                 int     `json:"nodesVisited"`
	PrunedSubtrees               int     `json:"prunedSubtrees"`
	FirstMinuteComplete          bool    `json:"firstMinuteComplete"`
	FirstMinuteBranchesResolved  string  `json:"firstMinuteBranchesResolved"`
	FirstMinuteNodesVisited      int     `json:"firstMinuteNodesVisited"`
	SecondMinuteComplete         bool    `json:"secondMinuteComplete"`
	SecondMinuteBranchesResolved string  `json:"secondMinuteBranchesResolved"`
	SecondMinuteNodesVisited     int     `json:"secondMinuteNodesVisited"`
}

// ProgressCallback receives monotonic snapshots while the solver is running.
type ProgressCallback func(SolverProgress)

type solverProgressTracker struct {
	mu             sync.Mutex
	emitMu         sync.Mutex
	callback       ProgressCallback
	total          *big.Int
	resolved       *big.Int
	pruned         *big.Int
	nodesVisited   int
	prunedSubtrees int
	startedAt      time.Time
	lastReport     time.Time
	nextSequence   uint64
	lastEmitted    uint64
	firstResolved  *big.Int
	firstNodes     int
	firstComplete  bool
	secondResolved *big.Int
	secondNodes    int
	secondComplete bool
}

type progressEmission struct {
	sequence uint64
	snapshot SolverProgress
}

func newSolverProgressTracker(view solverProblemView, callback ProgressCallback) (*solverProgressTracker, *big.Int) {
	total := big.NewInt(1)
	for _, candidateID := range view.sortedCandidateIDs {
		optionCount := len(view.candidateGroupOptions[candidateID])
		if optionCount == 0 {
			total.SetInt64(1)
			break
		}
		total.Mul(total, big.NewInt(int64(optionCount)))
	}

	startedAt := time.Now()
	tracker := &solverProgressTracker{
		callback:       callback,
		total:          new(big.Int).Set(total),
		resolved:       new(big.Int),
		pruned:         new(big.Int),
		startedAt:      startedAt,
		firstResolved:  new(big.Int),
		secondResolved: new(big.Int),
	}
	tracker.emit(tracker.reportLocked(startedAt, true))
	return tracker, total
}

func (tracker *solverProgressTracker) visitNode() {
	if tracker == nil || tracker.callback == nil {
		return
	}
	tracker.mu.Lock()
	tracker.nodesVisited++
	emission := tracker.reportLocked(time.Now(), false)
	tracker.mu.Unlock()
	tracker.emit(emission)
}

func (tracker *solverProgressTracker) resolve(weight *big.Int, pruned bool) {
	if tracker == nil || tracker.callback == nil || weight == nil || weight.Sign() <= 0 {
		return
	}
	tracker.mu.Lock()
	tracker.resolved.Add(tracker.resolved, weight)
	if tracker.resolved.Cmp(tracker.total) > 0 {
		tracker.resolved.Set(tracker.total)
	}
	if pruned {
		tracker.pruned.Add(tracker.pruned, weight)
		tracker.prunedSubtrees++
	}
	emission := tracker.reportLocked(time.Now(), false)
	tracker.mu.Unlock()
	tracker.emit(emission)
}

func (tracker *solverProgressTracker) finish(completed bool) {
	if tracker == nil || tracker.callback == nil {
		return
	}
	tracker.mu.Lock()
	if completed {
		tracker.resolved.Set(tracker.total)
	}
	emission := tracker.reportLocked(time.Now(), true)
	tracker.mu.Unlock()
	tracker.emit(emission)
}

func (tracker *solverProgressTracker) reportLocked(now time.Time, force bool) *progressEmission {
	if tracker.callback == nil {
		return nil
	}
	tracker.captureThroughputLocked(now)
	if !force && !tracker.lastReport.IsZero() && now.Sub(tracker.lastReport) < progressReportInterval {
		return nil
	}
	tracker.lastReport = now
	tracker.nextSequence++

	percentage, _ := new(big.Float).
		Quo(new(big.Float).SetInt(tracker.resolved), new(big.Float).SetInt(tracker.total)).
		Float64()
	return &progressEmission{sequence: tracker.nextSequence, snapshot: SolverProgress{
		Percent:                      percentage * 100,
		BranchesResolved:             tracker.resolved.String(),
		TotalBranches:                tracker.total.String(),
		BranchesPruned:               tracker.pruned.String(),
		NodesVisited:                 tracker.nodesVisited,
		PrunedSubtrees:               tracker.prunedSubtrees,
		FirstMinuteComplete:          tracker.firstComplete,
		FirstMinuteBranchesResolved:  tracker.firstResolved.String(),
		FirstMinuteNodesVisited:      tracker.firstNodes,
		SecondMinuteComplete:         tracker.secondComplete,
		SecondMinuteBranchesResolved: tracker.secondResolved.String(),
		SecondMinuteNodesVisited:     tracker.secondNodes,
	}}
}

func (tracker *solverProgressTracker) captureThroughputLocked(now time.Time) {
	elapsed := now.Sub(tracker.startedAt)
	if !tracker.firstComplete && elapsed >= throughputWindow {
		tracker.firstResolved.Set(tracker.resolved)
		tracker.firstNodes = tracker.nodesVisited
		tracker.firstComplete = true
	}
	if !tracker.secondComplete && elapsed >= 2*throughputWindow {
		tracker.secondResolved.Sub(tracker.resolved, tracker.firstResolved)
		tracker.secondNodes = tracker.nodesVisited - tracker.firstNodes
		tracker.secondComplete = true
	}
}

// emit never holds the solver-state mutex while crossing the backend/frontend
// boundary. If concurrent workers race to publish, stale snapshots are dropped.
func (tracker *solverProgressTracker) emit(emission *progressEmission) {
	if tracker == nil || tracker.callback == nil || emission == nil {
		return
	}
	tracker.emitMu.Lock()
	defer tracker.emitMu.Unlock()
	if emission.sequence <= tracker.lastEmitted {
		return
	}
	tracker.callback(emission.snapshot)
	tracker.lastEmitted = emission.sequence
}

func childBranchWeight(parentWeight *big.Int, optionCount int) *big.Int {
	if parentWeight == nil || optionCount <= 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Quo(new(big.Int).Set(parentWeight), big.NewInt(int64(optionCount)))
}
