// What this measures: misc/gnopreview/README.md calls -max-pages (400) a cap
// that "backstops the rest", and the ADR lists "At most 25 realms and 400
// pages" under Bounds. Crawler.Run does not truncate at that number: it returns
// an error, which fails `gnopreview render`, which fails the "pr / preview" job
// on the contributor's pull request. No preview, no comment, one red check.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_pagecap_test.go
//	cd misc/gnopreview && go test ./... -run PageCap -v
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Seeds() emits four URLs for one realm (/r/x/y, $source, $help, /r/x), so a
// cap of 2 is reached with the queue still full — the same shape a 26th realm
// or a link-rich realm set reaches against the shipped cap of 400.
func TestPageCapAbortsTheCrawl(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head></head><body><a href="/r/x/y:p/a">a</a></body></html>`)
	}))
	defer srv.Close()

	capped := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, MaxPages: 2, Live: "https://gno.land"}
	err := capped.Run()
	if err == nil {
		t.Fatalf("Run() with MaxPages=2 returned nil: the cap truncated as documented (captured %d pages)", len(capped.pages))
	}
	t.Logf("Run() with MaxPages=2 -> %v (captured %d pages, the rest dropped with the job)", err, len(capped.pages))

	// Counterfactual: the fixture itself is fine, so the error is the cap.
	uncapped := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, MaxPages: 0, Live: "https://gno.land"}
	if err := uncapped.Run(); err != nil {
		t.Fatalf("Run() with no cap failed, so the fixture is at fault, not the cap: %v", err)
	}
	t.Logf("Run() with MaxPages=0 -> nil, %d pages captured", len(uncapped.pages))
}
