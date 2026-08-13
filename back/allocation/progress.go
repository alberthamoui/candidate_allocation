package allocation

import (
	"math/big"
	"sync"
	"time"
)

const progressReportInterval = 100 * time.Millisecond

// SolverProgress reports how much of the conceptual assignment tree has been
// resolved. A pruned subtree resolves all of its descendant branches at once.
type SolverProgress struct {
	Percent          float64 `json:"percent"`
	BranchesResolved string  `json:"branchesResolved"`
	TotalBranches    string  `json:"totalBranches"`
	BranchesPruned   string  `json:"branchesPruned"`
	NodesVisited     int     `json:"nodesVisited"`
	PrunedSubtrees   int     `json:"prunedSubtrees"`
}

// ProgressCallback receives monotonic snapshots while the solver is running.
type ProgressCallback func(SolverProgress)

type solverProgressTracker struct {
	mu             sync.Mutex
	callback       ProgressCallback
	total          *big.Int
	resolved       *big.Int
	pruned         *big.Int
	nodesVisited   int
	prunedSubtrees int
	lastReport     time.Time
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

	tracker := &solverProgressTracker{
		callback: callback,
		total:    new(big.Int).Set(total),
		resolved: new(big.Int),
		pruned:   new(big.Int),
	}
	tracker.reportLocked(true)
	return tracker, total
}

func (tracker *solverProgressTracker) visitNode() {
	if tracker == nil || tracker.callback == nil {
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.nodesVisited++
	tracker.reportLocked(false)
}

func (tracker *solverProgressTracker) resolve(weight *big.Int, pruned bool) {
	if tracker == nil || tracker.callback == nil || weight == nil || weight.Sign() <= 0 {
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	tracker.resolved.Add(tracker.resolved, weight)
	if tracker.resolved.Cmp(tracker.total) > 0 {
		tracker.resolved.Set(tracker.total)
	}
	if pruned {
		tracker.pruned.Add(tracker.pruned, weight)
		tracker.prunedSubtrees++
	}
	tracker.reportLocked(false)
}

func (tracker *solverProgressTracker) finish() {
	if tracker == nil || tracker.callback == nil {
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.resolved.Set(tracker.total)
	tracker.reportLocked(true)
}

func (tracker *solverProgressTracker) reportLocked(force bool) {
	if tracker.callback == nil {
		return
	}
	now := time.Now()
	if !force && !tracker.lastReport.IsZero() && now.Sub(tracker.lastReport) < progressReportInterval {
		return
	}
	tracker.lastReport = now

	percentage, _ := new(big.Float).
		Quo(new(big.Float).SetInt(tracker.resolved), new(big.Float).SetInt(tracker.total)).
		Float64()
	tracker.callback(SolverProgress{
		Percent:          percentage * 100,
		BranchesResolved: tracker.resolved.String(),
		TotalBranches:    tracker.total.String(),
		BranchesPruned:   tracker.pruned.String(),
		NodesVisited:     tracker.nodesVisited,
		PrunedSubtrees:   tracker.prunedSubtrees,
	})
}

func childBranchWeight(parentWeight *big.Int, optionCount int) *big.Int {
	if parentWeight == nil || optionCount <= 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Quo(new(big.Int).Set(parentWeight), big.NewInt(int64(optionCount)))
}
