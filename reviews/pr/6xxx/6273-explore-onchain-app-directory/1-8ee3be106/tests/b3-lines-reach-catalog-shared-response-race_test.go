// Repro, from a plain clone of gnolang/gno at 8ee3be106cd932e3156197c4ce3a080520493eb9:
//
//	git clone https://github.com/gnolang/gno && cd gno && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//	cp <this file> gno.land/pkg/gnoweb/feature/store/zz_b3_shared_race_test.go
//	go test -count=1 -race -run TestB3ConcurrentRendersShareCachedResponses -v ./gno.land/pkg/gnoweb/feature/store/
//
// Catalog class: global mutable state and concurrency. home(), build() and
// paged() hand every concurrent request the same cached, sanitised response
// pointer; this checks that no view builder writes through it, under -race.
package store

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
)

func TestB3ConcurrentRendersShareCachedResponses(t *testing.T) {
	ls := strings.Join([]string{trustedApp, communityApp,
		listingJSON("two", "nym-acmex123", kindApp, "Two", `"earned":true`),
		listingJSON("three", "nym-acmex123", kindApp, "Three", `"earned":true`),
		listingJSON("four", "nym-acmex123", kindApp, "Four", `"earned":true`),
	}, ",")
	home := `{"version":1,"height":5000,"time":1759538400,"pulse":{"listed_7d":2,"stars_7d":3},` +
		`"categories":[{"key":"defi","label":"DeFi","count":30}],` +
		`"shelves":[{"title":"New","pinned":true,"slugs":["blog","pixels","two","three","four"],"more":[{"key":"latest","title":"New","count":30}]},` +
		`{"title":"Trending","slugs":["two","three","four","pixels"],"stars_7d":[9,4,3,1],"more":[{"key":"trending","title":"Trending","count":4}]}],` +
		`"activity":[{"kind":"listed","slug":"two"}],"listings":[` + ls + `]}`
	pkg := listingJSON("mylib", "nym-acmex123", kindPackage, "My lib", "")
	h, _ := newTestHandler(t, map[string]string{
		"api/v1/home":            home,
		"api/v1/build":           `{"version":1,"shelves":[{"title":"New to build with","slugs":["mylib","mylib","mylib","mylib"]}],"builders":{"top":[{"namespace":"acme","listings":3,"stars":40}],"new":[]},"listings":[` + pkg + `]}`,
		"api/v1/category/defi/2": `{"version":1,"category":{"key":"defi","label":"DeFi","count":30},"page":2,"pages":2,"listings":[` + communityApp + `]}`,
		"api/v1/list/latest/2":   `{"version":1,"list":{"key":"latest","title":"New","count":30},"page":2,"pages":2,"listings":[` + communityApp + `]}`,
	})
	var wg sync.WaitGroup
	var mu sync.Mutex
	rendered := map[string]int{}
	for range 8 {
		for _, target := range []string{"", ":c/defi?page=2", ":latest?page=2", ":build"} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 25 {
					u, err := weburl.Parse("/r/gnoland/store" + target)
					if err != nil {
						t.Error(err)
						return
					}
					view, _ := h.View(context.Background(), u)
					if view == nil {
						continue
					}
					var buf bytes.Buffer
					if err := view.Render(&buf); err != nil {
						t.Error(err)
						return
					}
					mu.Lock()
					rendered[target]++
					mu.Unlock()
				}
			}()
		}
	}
	wg.Wait()
	t.Logf("rendered per target: %v", rendered)
	for _, target := range []string{"", ":c/defi?page=2", ":latest?page=2", ":build"} {
		if rendered[target] == 0 {
			t.Errorf("%q never rendered: the race check did not reach its view", target)
		}
	}
}
