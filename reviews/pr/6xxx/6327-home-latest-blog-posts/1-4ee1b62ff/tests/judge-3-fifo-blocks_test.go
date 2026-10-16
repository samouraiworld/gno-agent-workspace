package gnoweb

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A FIFO renamed over a served static file: the first resolve blocks in
// os.ReadFile under f.mu and every other resolve waits on f.mu. At 4ee1b62ff
// 0 of 10 concurrent resolves return within 1 s.
func TestZZJudge3FifoBlocksAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home.md")
	os.WriteFile(path, []byte("# Old"), 0o644)
	past := time.Now().Add(-time.Minute)
	os.Chtimes(path, past, past)
	target, err := NewStaticFileAlias(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path+".new", 0o644); err != nil {
		t.Fatal(err)
	}
	os.Rename(path+".new", path)
	done := make(chan string, 10)
	for i := 0; i < 10; i++ {
		go func() { done <- target.resolve().Value }()
	}
	served, timeout := 0, time.After(time.Second)
loop:
	for served < 10 {
		select {
		case <-done:
			served++
		case <-timeout:
			break loop
		}
	}
	t.Logf("served within 1s: %d of 10", served)
}
