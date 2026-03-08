//go:build !darwin && !linux

package main

import (
	"os/exec"
	"testing"
)

func configureProcessGroup(cmd *exec.Cmd) {
}

func killProcessTree(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
