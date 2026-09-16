// Candidate #49 — opTail gives no arity for an open-set operation.
//
// from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
//   cp -r examples /tmp/probe && cp 49-optail-open-set-no-arity.gno \
//     /tmp/probe/gno.land/r/nt/grc20routes/v0/zzz_probe49_test.gno
//   (cd /tmp/probe && gno test -v -run TestProbe49 ./gno.land/r/nt/grc20routes/v0)
//
// Asserts: Register()ing an open-set op ("deposit", the PR doc's own wugnot
// example) and reading it back through Render (renderToken) gives a line with
// the generic ", …" tail rather than an argument list, unlike the three
// canonical ops on the same route. Fails at the reviewed head — because the
// current code IS the bug, this is the passing-now assertion; there is no
// fixed-state twin to comment in since opTail has no argument table to draw
// deposit's arity from.
package grc20routes

import (
	"strings"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

func TestProbe49_OpenOpRenderHasNoArity(cur realm, t *testing.T) {
	key := mint(0, cur, "PROB")

	Register(cross(cur), "PROB", Canonical().With("deposit", "Deposit"))

	out := renderToken(key)
	t.Log(out)

	// Current (buggy) state: deposit's tail is a bare ellipsis, no arity.
	uassert.True(t, strings.Contains(out, "deposit: `Deposit(cur, …)`"),
		"expected the generic ellipsis tail for the open operation, per opTail's default case")

	// What the open-set doc comment (Route.Funcs, lines 217-223) promises and
	// does not deliver: an arity for deposit the way approve gets one.
	// Uncomment once opTail (or an Entry.Arity field) covers declared ops:
	// uassert.True(t, strings.Contains(out, "deposit: `Deposit(cur, amount)`"),
	// 	"deposit's own declared shape should render, not a placeholder")
}
