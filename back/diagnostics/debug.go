//go:build debug || dev

package diagnostics

import (
	"runtime"
	"sync"
	"time"
)

const Enabled = true
const sampleLimit = 128

var sampleMu sync.Mutex
var samples []Sample

// Begin records duration and process-wide allocation deltas, never payloads.
func Begin(operation string, items int) func() {
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		sampleMu.Lock()
		defer sampleMu.Unlock()
		if len(samples) == sampleLimit {
			copy(samples, samples[1:])
			samples = samples[:sampleLimit-1]
		}
		samples = append(samples, Sample{Operation: operation, Items: items, DurationMS: float64(elapsed) / float64(time.Millisecond), AllocatedBytes: after.TotalAlloc - before.TotalAlloc, HeapBytes: after.HeapAlloc})
	}
}

func Snapshot() []Sample {
	sampleMu.Lock()
	defer sampleMu.Unlock()
	return append([]Sample(nil), samples...)
}

func Reset() { sampleMu.Lock(); defer sampleMu.Unlock(); samples = nil }
