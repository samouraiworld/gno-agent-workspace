// Repro for: the stop() returned by startGnodev sends SIGTERM and then waits
// forever, with no SIGKILL escalation and no deadline, so a gnodev that does
// not exit on SIGTERM hangs the render step; none of the three preview
// workflows sets timeout-minutes, so the job runs to the 360-minute default.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_stophang_test.go
//	cd misc/gnopreview && go test -run TestStopHangs -v .
package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStopHangsOnAGnodevThatIgnoresSIGTERM(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	fake := filepath.Join(dir, "fake-gnodev")
	script := "#!/bin/sh\ntrap '' TERM\necho $$ > " + pidFile + "\nwhile true; do sleep 0.2; done\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config{out: dir, gnodev: fake, port: 18899}
	stop, _, err := startGnodev(cfg, dir, nil, cfg.port, "gnodev.log")
	if err != nil {
		t.Fatal(err)
	}

	var pid int
	for range 50 {
		if b, err := os.ReadFile(pidFile); err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
			if pid > 0 {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("fake gnodev never started")
	}

	done := make(chan struct{})
	go func() { stop(); close(done) }()

	select {
	case <-done:
		t.Fatal("stop() returned; SIGTERM was not ignored")
	case <-time.After(3 * time.Second):
		t.Log("stop() still blocked 3s after SIGTERM: no escalation, no deadline")
	}

	_ = syscall.Kill(-pid, syscall.SIGKILL)
	select {
	case <-done:
		t.Log("stop() returned only once the process was SIGKILLed from outside")
	case <-time.After(5 * time.Second):
		t.Fatal("stop() still blocked after SIGKILL")
	}
}
