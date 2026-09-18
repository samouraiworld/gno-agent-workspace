// Pins the shape Comment produces when baseURL is empty — the default of
// -base-url, so every local `gnopreview comment` run takes it.
// Measured at ecf7af0f2: the four `base == ""` guards (comment.go:30, 100, 131,
// 150) are decided independently and none of them is covered; rewriting the one
// at 30 to `return ""` leaves `go test ./...` green.
//
// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_judge221_test.go
//	cd misc/gnopreview && go test -run TestCommentWithoutBaseURL -v .
//	# green at this head; red once any of the four guards changes shape.
//	rm zz_judge221_test.go
package main

import (
	"strings"
	"testing"
)

func TestCommentWithoutBaseURL(t *testing.T) {
	t.Run("gnoweb mode still lists every realm", func(t *testing.T) {
		p := &Plan{Gnoweb: true, Realms: gnowebSeedRealms,
			Shots: []Shot{{File: "home.png", Label: "home"}}}
		got := Comment(p, "", "7")
		// The godoc promises the comment "still lists what was rendered".
		for _, r := range gnowebSeedRealms {
			if !strings.Contains(got, "- `"+r+"`\n") {
				t.Errorf("realm %q missing from the no-base comment:\n%s", r, got)
			}
		}
		// Nothing may emit a root-relative src/href: GitHub resolves one
		// against github.com, and the body is posted by a privileged job.
		for _, frag := range []string{`src="/`, `href="/`, "](/"} {
			if strings.Contains(got, frag) {
				t.Errorf("no-base comment carries a root-relative link %q:\n%s", frag, got)
			}
		}
		// shotGrid has no URL to embed against, so it contributes nothing.
		if strings.Contains(got, "<img") {
			t.Errorf("no-base comment embeds an image:\n%s", got)
		}
	})

	t.Run("both mode keeps its headings", func(t *testing.T) {
		p := &Plan{Gnoweb: true,
			ChangedRealms: []string{"gno.land/r/demo/foo"},
			Realms:        []string{"gno.land/r/demo/foo", "gno.land/r/demo/bar"},
			ChangedPkgs:   []string{"gno.land/p/demo/x"},
			Pairs:         []ShotPair{{Realm: "gno.land/r/demo/foo", URL: "u", After: "a.png", New: true}}}
		got := Comment(p, "", "7")
		for _, want := range []string{"**Changed realms (1)**", "- `gno.land/r/demo/foo`\n", "- `gno.land/r/demo/bar`\n"} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q:\n%s", want, got)
			}
		}
		if strings.Contains(got, "<img") || strings.Contains(got, `href="/`) {
			t.Errorf("no-base comment carries an image or a root-relative link:\n%s", got)
		}
	})
}
