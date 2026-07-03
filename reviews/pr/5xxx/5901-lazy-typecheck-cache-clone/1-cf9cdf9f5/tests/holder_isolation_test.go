/* Run: from a gno checkout:
gh pr checkout 5901 -R gnolang/gno && git checkout cf9cdf9f5
curl -fsSL -o gno.land/pkg/sdk/vm/holder_isolation_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5901-lazy-typecheck-cache-clone/1-cf9cdf9f5/tests/holder_isolation_test.go
go test -run 'TestHolder' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/holder_isolation_test.go
*/

// Exercises typeCheckCacheHolder.get() directly (keeper.go:417).
// Proves the three invariants the PR rests on: get() defers the clone,
// clones at most once per holder, and each holder's clone is isolated
// from base and from a sibling holder's clone. types.Package pointers
// used as opaque map values, matching gno.TypeCheckCache's shape.
package vm

import (
	"go/types"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

func TestHolderLazyAndIsolated(t *testing.T) {
	pkgA := types.NewPackage("a", "a")
	base := gno.TypeCheckCache{"a": pkgA}

	// Lazy: constructing the holder does not clone.
	h1 := &typeCheckCacheHolder{base: base}
	if h1.cloned != nil {
		t.Fatal("holder cloned before first get()")
	}

	// First get() clones; the clone carries every base entry. Distinctness
	// from base is proven below by the write that base never sees.
	c1 := h1.get()
	if c1 == nil {
		t.Fatal("get() returned nil clone")
	}
	if c1["a"] != pkgA {
		t.Fatal("clone dropped a base entry")
	}

	// Single-clone-per-holder: a second get() returns the same map, and a
	// write made through the first handle is visible through the second.
	pkgB := types.NewPackage("b", "b")
	c1["b"] = pkgB
	c1b := h1.get()
	if c1b["b"] != pkgB {
		t.Fatal("second get() re-cloned instead of reusing the working copy")
	}

	// Isolation from base: the consumer wrote "b" into the clone; base must
	// not see it. TypeCheckMemPackage writes newly type-checked packages into
	// the passed cache (gotypecheck.go:343), so this is the real risk.
	if _, ok := base["b"]; ok {
		t.Fatal("clone write leaked into shared base cache")
	}

	// Isolation between transactions: a second holder over the same base
	// gets its own clone; h1's write is invisible to it.
	h2 := &typeCheckCacheHolder{base: base}
	c2 := h2.get()
	if _, ok := c2["b"]; ok {
		t.Fatal("one tx's clone write leaked into another tx's clone")
	}
	if c2["a"] != pkgA {
		t.Fatal("second holder's clone dropped a base entry")
	}
}

// Documents the nil-base degradation: maps.Clone(nil) is nil, so a nil base
// makes the nil-sentinel re-clone on every get(). Not reachable in production
// (NewVMKeeper and LoadStdlibCached both set a non-nil base), so this asserts
// the current behavior rather than flagging a bug.
func TestHolderNilBaseReclones(t *testing.T) {
	var nilBase gno.TypeCheckCache // nil map
	h := &typeCheckCacheHolder{base: nilBase}
	if got := h.get(); got != nil {
		t.Fatalf("clone of nil base expected nil, got %v", got)
	}
	// cloned stayed nil, so the sentinel never latches: still nil next call.
	if h.cloned != nil {
		t.Fatal("cloned unexpectedly non-nil after get() over nil base")
	}
}
