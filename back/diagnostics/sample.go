// Package diagnostics exposes aggregate timings only in debug/dev builds.
package diagnostics

type Sample struct {
	Operation      string  `json:"operation"`
	Items          int     `json:"items"`
	DurationMS     float64 `json:"durationMs"`
	AllocatedBytes uint64  `json:"allocatedBytes"`
	HeapBytes      uint64  `json:"heapBytes"`
}
