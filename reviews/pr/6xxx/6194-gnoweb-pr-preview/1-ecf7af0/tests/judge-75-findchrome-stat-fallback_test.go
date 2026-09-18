package main

// Asserts what findChrome's os.Stat fallback (shots.go:208) actually adds over
// exec.LookPath: only a path that exists and cannot be executed. Measured at
// head ecf7af0f2 — every case below passes there, which is the finding: the
// fallback hands ScreenshotPairs/Screenshot a bin that every chromeShot exec
// then fails on, instead of the empty string that prints one skip line.
//
// Repro from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//   git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//   cp path/to/judge-75-findchrome-stat-fallback_test.go misc/gnopreview/
//   cd misc/gnopreview && go test -run TestJudge75 -v ./...
//   rm judge-75-findchrome-stat-fallback_test.go

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// LookPath already resolves a slashed path when it is runnable, so the Stat
// branch is dead for every executable candidate.
func TestJudge75_LookPathCoversExecutableSlashedPath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Google Chrome")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := exec.LookPath(p); err != nil || got != p {
		t.Fatalf("LookPath(%q) = %q, %v; want the path itself", p, got, err)
	}
	if got := findChrome(p); got != p {
		t.Fatalf("findChrome = %q, want %q", got, p)
	}
}

// The only candidate Stat adds: exists, not a directory, not executable.
// findChrome returns it and every exec on it fails.
func TestJudge75_StatFallbackReturnsNonExecutable(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "chrome-no-x")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath(p); err == nil {
		t.Fatalf("LookPath accepted a non-executable file; the fallback would be dead")
	} else {
		t.Logf("LookPath rejected it: %v", err)
	}

	got := findChrome(p)
	// IS:     head returns the unrunnable path, so callers believe a browser exists.
	if got != p {
		t.Fatalf("findChrome = %q, want %q", got, p)
	}
	// SHOULD: uncomment once the fallback checks executability.
	// if got != "" { t.Fatalf("findChrome = %q, want \"\" (no runnable browser)", got) }

	err := exec.Command(got, "--version").Run()
	if err == nil {
		t.Fatalf("exec of %q unexpectedly succeeded", got)
	}
	t.Logf("every chromeShot on this bin fails: %v", err)
}

// A bare name is worse: Stat resolves it against the working directory, while
// exec.Command re-resolves it on PATH, so the returned value cannot be run at
// all — even when the file in the working directory is executable.
func TestJudge75_StatFallbackReturnsBareNameUnrunnable(t *testing.T) {
	dir := t.TempDir()
	empty := t.TempDir() // a PATH with no browser on it
	if err := os.WriteFile(filepath.Join(dir, "chromium"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", empty)
	t.Setenv("CHROME", "")
	t.Chdir(dir)

	got := findChrome("")
	// IS:     head returns the bare name "chromium" off the working directory.
	if got != "chromium" {
		t.Fatalf("findChrome = %q, want %q", got, "chromium")
	}
	// SHOULD: uncomment once the fallback stops answering from the cwd.
	// if got != "" { t.Fatalf("findChrome = %q, want \"\" (nothing on PATH)", got) }

	err := exec.Command(got, "--version").Run()
	if err == nil {
		t.Fatalf("exec of %q unexpectedly succeeded", got)
	}
	t.Logf("exec.Command re-resolves it on PATH and fails: %v", err)
}
