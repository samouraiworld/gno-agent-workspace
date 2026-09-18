package main

// b10-refactor-r2: the Empty() assertion in TestBuildPlan's table restates the
// Mode() assertion beside it; it cannot fail on its own, for any Plan.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/b10_empty_test.go
//	cd misc/gnopreview && go test -run 'TestB10EmptyRestatesMode' -v ./...
//
// The 16-cell enumeration below is green: Empty() == (Mode() == "none") holds
// for every combination of the four fields Mode() reads, so plan_test.go's
//
//	if got.Empty() != (tc.wantMode == "none") { ... }
//
// is true by construction whenever the Mode assertion above it passes.

import "testing"

func TestB10EmptyRestatesMode(t *testing.T) {
	t.Parallel()
	some := []string{"gno.land/r/x/a"}
	for i := range 16 {
		p := &Plan{Gnoweb: i&1 != 0}
		if i&2 != 0 {
			p.ChangedRealms = some
		}
		if i&4 != 0 {
			p.ChangedPkgs = some
		}
		if i&8 != 0 {
			p.Realms = some
		}
		if got, want := p.Empty(), p.Mode() == "none"; got != want {
			t.Errorf("cell %d: Empty()=%v Mode()=%q", i, got, p.Mode())
		}
	}
}
