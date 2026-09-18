// b5-refactor-comment-golden_test.go — characterization test for misc/gnopreview
// comment.go at gnolang/gno PR 6194, head ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// comment.go renders the sticky PR comment and the snapshot landing page. The
// refactor candidates on that file (drop the `direct` map, one-parameter
// `link`, `isSeed` as a *Plan method, one `dir` local in Index, the p.Empty()
// guard above the two prologue writes) must not move a single byte of either
// rendering. This test pins every branch of Comment (realms-only, gnoweb-only,
// both, before/after pairs, empty plan) across three baseURL shapes plus Index,
// and fails if any output changes.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/
//	cd misc/gnopreview && go test -run TestRefactorGolden ./...   # PASS at head
//	# apply the refactor, run it again: still PASS == behaviour preserved.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// wantDigest is the SHA-256 of the renderings below, measured at head.
const wantDigest = "bba94764c9cf0495c1a90e4ff5baa5d2c37beace99c5a53d3cec88ee8059fd9a"

func refactorGoldenText() string {
	plans := []*Plan{
		{},
		{
			ChangedRealms: []string{"gno.land/r/x/leaf"},
			ChangedPkgs:   []string{"gno.land/p/x/base/v0"},
			Realms:        []string{"gno.land/r/x/leaf", "gno.land/r/x/other"},
			Dropped:       3,
		},
		{
			Gnoweb: true,
			Realms: []string{"gno.land/r/gnoland/home", "gno.land/r/gnoland/blog"},
			Shots: []Shot{
				{File: "_shots/home.png", Label: "Home"},
				{File: "_shots/blog.png", Label: "Blog"},
				{File: "_shots/help.png", Label: "Help"},
			},
		},
		{
			Gnoweb:        true,
			ChangedRealms: []string{"gno.land/r/x/leaf"},
			ChangedPkgs:   []string{"gno.land/p/x/base/v0"},
			Realms:        []string{"gno.land/r/x/leaf", "gno.land/r/gnoland/home", "gno.land/r/x/other"},
			Shots:         []Shot{{File: "_shots/home.png", Label: "Home"}},
		},
		{
			ChangedRealms: []string{"gno.land/r/x/leaf", "gno.land/r/x/fresh"},
			Realms:        []string{"gno.land/r/x/fresh", "gno.land/r/x/leaf"},
			Pairs: []ShotPair{
				{Realm: "gno.land/r/x/leaf", Before: "_shots/r-x-leaf-before.png", After: "_shots/r-x-leaf-after.png", URL: "r/x/leaf/"},
				{Realm: "gno.land/r/x/fresh", After: "_shots/r-x-fresh-after.png", URL: "r/x/fresh/", New: true},
			},
		},
	}
	var b strings.Builder
	for i, p := range plans {
		for _, base := range []string{"https://example.test/pr-7/", "https://example.test/pr-7", ""} {
			fmt.Fprintf(&b, "=== comment %d base=%q\n%s\n", i, base, Comment(p, base, "7"))
		}
	}
	c := &Crawler{}
	c.pages = map[string]*page{"/r/x/leaf": nil, "/r/gnoland/home": nil}
	idx := &Plan{Realms: []string{"gno.land/r/x/leaf", "gno.land/r/gnoland/home", "gno.land/r/x/missing"}}
	fmt.Fprintf(&b, "=== index\n%s\n", Index(idx, c))
	return b.String()
}

func TestRefactorGolden(t *testing.T) {
	got := refactorGoldenText()
	sum := sha256.Sum256([]byte(got))
	if d := hex.EncodeToString(sum[:]); d != wantDigest {
		t.Errorf("rendering changed: digest %s, want %s\n---\n%s", d, wantDigest, got)
	}
}
