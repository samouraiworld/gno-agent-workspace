// solo-finder-1: a static alias file replaced by one gnoweb cannot read keeps
// serving the old page with nothing returned or logged; a restart then fails.
// Also: concurrent resolves during renames, under -race (race-clean).
//
// Repro from a plain clone, as a non-root user (root reads a 0000 file):
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6327/head && git checkout --detach 4ee1b62ff5fa327ac9817b4b0d6f08251765a354
//   cp <this file> gno.land/pkg/gnoweb/zz_solofinder1_test.go
//   go test ./gno.land/pkg/gnoweb -run 'TestZZSoloFinder1' -race -count=1 -v
//
// Observed at 4ee1b62ff:
//   request 0: Value="# Old" Kind=1   (x3)
//   restart on the same file: err=open .../home.md: permission denied
//   --- PASS: TestZZSoloFinder1ConcurrentResolve (0.01s)   (no DATA RACE)
//   go test ./gno.land/pkg/gnoweb/ -race -count=1  ->  ok 63.166s
// Merge base f915442bf: NewStaticFileAlias/ResolveAlias do not exist; the file
// was read once in parseAliases, whose read error fails gnoweb at boot.
package gnoweb_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Concurrent requests resolving one static alias while gnoalias.sh-style
// renames replace the file underneath. Run with -race.
func TestZZSoloFinder1ConcurrentResolve(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home.md")
	require.NoError(t, os.WriteFile(path, []byte("# v0"), 0o644))
	target, err := gnoweb.NewStaticFileAlias(path)
	require.NoError(t, err)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				v := gnoweb.ResolveAlias(target).Value
				if len(v) < 3 || v[:3] != "# v" {
					t.Errorf("torn page: %q", v)
					return
				}
			}
		}()
	}
	for i := 1; i <= 200; i++ {
		next := fmt.Sprintf("%s.%d", path, i)
		require.NoError(t, os.WriteFile(next, []byte(fmt.Sprintf("# v%d", i)), 0o644))
		require.NoError(t, os.Rename(next, path))
	}
	close(stop)
	wg.Wait()
	// The last write is under 2 s old, so the next resolve re-reads it.
	assert.Equal(t, "# v200", gnoweb.ResolveAlias(target).Value)
}

// A file replaced by one gnoweb cannot read: the old page keeps being served
// with nothing returned or logged, and a restart on the same file then fails.
func TestZZSoloFinder1UnreadableFileIsSilent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a 0000 file")
	}
	path := filepath.Join(t.TempDir(), "home.md")
	require.NoError(t, os.WriteFile(path, []byte("# Old"), 0o644))
	target, err := gnoweb.NewStaticFileAlias(path)
	require.NoError(t, err)

	next := path + ".new"
	require.NoError(t, os.WriteFile(next, []byte("# New, longer"), 0o000))
	earlier := time.Now().Add(-time.Minute)
	require.NoError(t, os.Chtimes(next, earlier, earlier))
	require.NoError(t, os.Rename(next, path))

	for i := 0; i < 3; i++ {
		got := gnoweb.ResolveAlias(target)
		t.Logf("request %d: Value=%q Kind=%v", i, got.Value, got.Kind)
		assert.Equal(t, "# Old", got.Value)
	}
	_, err = gnoweb.NewStaticFileAlias(path)
	t.Logf("restart on the same file: err=%v", err)
	assert.Error(t, err)
}
