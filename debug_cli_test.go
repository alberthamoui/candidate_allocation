//go:build debug || dev

package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDebugPerformanceUsesSyntheticIsolatedStorage(t *testing.T) {
	var output bytes.Buffer
	binary := filepath.Join(t.TempDir(), "candidate-debug")
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Fatal("Go CLI must be on PATH for the debug executable test", err)
	}
	// #nosec G204 -- compile this repository into a new test-owned directory with the running Go toolchain.
	if data, err := exec.Command(goBinary, "build", "-tags", "debug", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatal(err, string(data))
	}
	// #nosec G204 -- execute only the synthetic debug binary built immediately above.
	data, err := exec.Command(binary, "debug", "performance", "-sizes", "10", "-runs", "1", "-solver-budget", "50ms").Output()
	if err != nil {
		t.Fatal(err)
	}
	output.Write(data)
	var report struct {
		Cases []performanceCase `json:"cases"`
	}
	if err := json.Unmarshal(output.Bytes(), &report); err != nil || len(report.Cases) != 1 {
		t.Fatal(err, output.String())
	}
	c := report.Cases[0]
	if c.Candidates != 10 || c.Assigned != 10 || c.Violations != 0 || len(c.Samples) < 5 || c.SnapshotBytes == 0 {
		t.Fatal(c)
	}
	if result, err := isolatedPerformanceCase(10, "preferences", time.Second, time.Nanosecond); err != nil || !result.TimedOut {
		t.Fatal(result, err)
	}
	if _, err := measurePerformanceCase(10, "criteria", 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"performance", "-sizes", "0"}, {"performance", "-runs", "0"}, {"performance", "-scenario", "unknown"}, {"performance", "-unknown"}, {"other"}} {
		if err := runDebugCLI(args, &output); err == nil {
			t.Fatal("invalid command accepted", args)
		}
	}
}
