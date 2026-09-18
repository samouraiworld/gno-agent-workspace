// Repro for findChrome in misc/gnopreview/shots.go:204.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout ecf7af0f2
//	cp <this file> misc/gnopreview/chrome_flag_test.go
//	cd misc/gnopreview && go test -run TestFindChrome -v .
//
// findChrome puts the explicit -chrome value at the head of the same candidate
// list as the auto-detected names and takes the first entry that resolves. A
// -chrome path that does not resolve is therefore discarded without a word and
// the run screenshots with whatever browser is on PATH instead.
//
// Observed at ecf7af0f2 (go1.25.9):
//
//	findChrome("/nonexistent/path/to/chrome") = ".../001/chromium"
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindChromeSilentlyIgnoresBadExplicitFlag(t *testing.T) {
	dir := t.TempDir()
	decoy := filepath.Join(dir, "chromium")
	if err := os.WriteFile(decoy, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CHROME", "")

	got := findChrome("/nonexistent/path/to/chrome")
	t.Logf("findChrome(%q) = %q", "/nonexistent/path/to/chrome", got)
	if got == "" {
		t.Fatal("no fallback happened")
	}
	if got != decoy {
		t.Fatalf("got %q, want the PATH decoy %q", got, decoy)
	}
	t.Log("an explicit -chrome that does not resolve is discarded without a word; " +
		"the run screenshots with a different browser than the one asked for")
}
