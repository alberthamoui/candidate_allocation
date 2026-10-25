//go:build !debug && !dev

package diagnostics

import "testing"

func TestReleaseCannotEnablePerformanceSampling(t *testing.T) {
	t.Setenv("CANDIDATE_DEBUG", "true")
	Begin("test", 1)()
	Reset()
	if Enabled || Snapshot() != nil {
		t.Fatal("release enabled debug instrumentation")
	}
}
