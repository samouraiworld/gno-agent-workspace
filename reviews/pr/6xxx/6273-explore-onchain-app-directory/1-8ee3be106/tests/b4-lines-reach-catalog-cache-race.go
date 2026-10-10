// Repro (gnolang/gno#6273 at 8ee3be106, cache.go, invariant catalog class
// "Global mutable state & concurrency"): concurrent renders over one Handler
// share every cached *homeResponse / *buildResponse / *pageResponse pointer,
// so any render path writing into a cached value is a data race on it.
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6273/head && git checkout --detach 8ee3be106cd932e3156197c4ce3a080520493eb9
//   cp <this file> gno.land/pkg/gnoweb/feature/store/zz_b4_cache_race_test.go
//   go test -race -count=1 -run 'TestB4CacheConcurrentRenders' ./gno.land/pkg/gnoweb/feature/store/
//   rm gno.land/pkg/gnoweb/feature/store/zz_b4_cache_race_test.go
//
// A clean run (ok, no "WARNING: DATA RACE") means the render paths only read
// the shared cached values and the mutex covers the entry map. The final
// assertion pins coalescing: 64 goroutines x 40 renders over three cache keys
// (home, build, category/defi/1) cost exactly three realm queries, which
// TestCacheCoalesces (three sequential renders) does not pin.

package store

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestB4CacheConcurrentRenders(t *testing.T) {
	listings := []string{trustedApp, communityApp, hostileApp, mismatched}
	home := homeBody(listings...)
	build := `{"version":1,"shelves":[{"title":"Packages","slugs":["blog","pixels","spoof","gnoswap"]}],` +
		`"builders":{"top":[{"namespace":"gnoland","listings":1,"stars":12}],"new":[]},` +
		`"listings":[` + trustedApp + `,` + communityApp + `]}`
	page := `{"version":1,"category":{"key":"defi","label":"DeFi","count":4},"page":1,"pages":1,` +
		`"listings":[` + trustedApp + `,` + communityApp + `,` + hostileApp + `]}`
	h, c := newTestHandler(t, map[string]string{
		"api/v1/home":            home,
		"api/v1/build":           build,
		"api/v1/category/defi/1": page,
	})

	targets := []string{"", ":build", ":c/defi"}
	var wg sync.WaitGroup
	for g := range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 40 {
				out := render(t, h, targets[(g+i)%len(targets)])
				if out == "" {
					t.Errorf("empty render for %q", targets[(g+i)%len(targets)])
					return
				}
			}
		}()
	}
	wg.Wait()
	t.Logf("realm calls: %d", c.calls.Load())
	assert.Equal(t, int32(3), c.calls.Load(), "one realm query per cache key")
}
