// Candidate 39 -- renderEntry (grc20routes.gno:737) wraps each prefix argument
// in md.EscapeText and places the result inside a markdown code span. EscapeInline's
// set (gnovm/stdlibs/chain/markdown/markdown.go:85) escapes '.', '-', '_', '[', ']',
// '(', ')'; validateSymbol permits '_' and '-' in a symbol. A token symbol
// containing one of those (e.g. Keyed("A_B"), or the proven path, which sets
// Keyed(symbol) verbatim) renders as a literal backslash inside backticks, since
// CommonMark does not interpret backslash escapes inside a code span. A user
// copying the rendered MsgCall argument out of the realm page sends "A\_B", not
// "A_B".
//
// Measured at head 3aa06ef757b87d731a3e554e96c7540a209fd9a3: this test fails
// there with the actual rendered line reading
//   - approve: `Approve(cur, A\_B, spender, amount)`
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// cp 39-render-entry-escapes-code-span.gno examples/gno.land/r/nt/grc20routes/v0/escape_test.gno
// (cd examples && go run ../gnovm/cmd/gno test -v -run 'TestRenderEntryEscapesPrefixIntoLiteralBackslash' ./gno.land/r/nt/grc20routes/v0)
//
// # observed (this run):
// --- FAIL: TestRenderEntryEscapesPrefixIntoLiteralBackslash (0.03s)
// (out contains "Approve(cur, A\_B, spender, amount)", not "Approve(cur, A_B, spender, amount)")
//
// git checkout examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno
// rm examples/gno.land/r/nt/grc20routes/v0/escape_test.gno

package grc20routes

import (
	"strings"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

func TestRenderEntryEscapesPrefixIntoLiteralBackslash(cur realm, t *testing.T) {
	key := mint(0, cur, "A_B")
	Register(cross(cur), "A_B", Keyed("A_B"))

	out := Render(key)

	// What the code should show: the literal, copy-pasteable argument.
	uassert.True(t, strings.Contains(out, "Approve(cur, A_B, spender, amount)"),
		"a user copying the rendered call should see the literal token symbol, not an escaped one")

	// Commented pair, the current (defective) behavior: uncomment once renderEntry
	// stops running md.EscapeText on a value that goes inside a code span.
	// uassert.True(t, strings.Contains(out, "Approve(cur, A\\_B, spender, amount)"))
}
