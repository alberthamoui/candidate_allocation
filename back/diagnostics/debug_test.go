//go:build debug || dev

package diagnostics

import "testing"

func TestBoundedAggregateSamples(t *testing.T) {
	Reset()
	for i := 0; i < sampleLimit+5; i++ {
		Begin("test", i)()
	}
	s := Snapshot()
	if !Enabled || len(s) != sampleLimit || s[0].Items != 5 || s[0].DurationMS < 0 {
		t.Fatal(s)
	}
	s[0].Operation = "changed"
	if Snapshot()[0].Operation != "test" {
		t.Fatal("snapshot aliases internal storage")
	}
	Reset()
	if len(Snapshot()) != 0 {
		t.Fatal("reset failed")
	}
}
