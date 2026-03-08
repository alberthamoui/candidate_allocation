package main

import (
	"reflect"
	"testing"
)

func TestDetectCLIMode(t *testing.T) {
	args, ok := detectCLIMode([]string{"cli", "-file", "input.xlsx"})
	if !ok {
		t.Fatal("expected cli mode to be detected")
	}

	expected := []string{"-file", "input.xlsx"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("unexpected cli args: got %#v want %#v", args, expected)
	}
}

func TestDetectCLIModeReturnsFalseForDesktopMode(t *testing.T) {
	args, ok := detectCLIMode([]string{"serve"})
	if ok {
		t.Fatal("expected non-cli args to skip cli mode")
	}
	if args != nil {
		t.Fatalf("expected nil args for non-cli mode, got %#v", args)
	}
}
