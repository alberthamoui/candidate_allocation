//go:build !debug && !dev

package main

import (
	"bytes"
	"testing"
)

func TestReleaseHasNoDebugCommand(t *testing.T) {
	t.Setenv("CANDIDATE_DEBUG", "true")
	var output bytes.Buffer
	if err := runDebugCLI([]string{"performance"}, &output); err == nil || output.Len() != 0 {
		t.Fatal("release exposes debug diagnostics")
	}
}
