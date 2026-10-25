//go:build !debug && !dev

package diagnostics

const Enabled = false

func Begin(_ string, _ int) func() { return func() {} }
func Snapshot() []Sample           { return nil }
func Reset()                       {}
