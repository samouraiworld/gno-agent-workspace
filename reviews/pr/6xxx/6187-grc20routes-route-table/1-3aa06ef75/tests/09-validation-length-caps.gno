// Pins the five length caps in grc20routes validation: maxEntries (16, :540),
// maxPrefixArg (128, :562), maxSymbolLen (64, :587), maxOpLen (32, :604) and
// maxIdentLen (64, :616). TestValidationRejectsUnsafeValues has eleven cases and
// none is over a length cap. Measured at head 3aa06ef75: deleting any one of the
// five leaves the package suite green 5/5; with this file it goes red 5/5.
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// cp 09-validation-length-caps.gno examples/gno.land/r/nt/grc20routes/v0/caps_test.gno
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)
//
// # the mutation showing the gap, one guard at a time (540 562 587 604 616):
// G=examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// for N in 540 562 587 604 616; do
//   git checkout "$G" && sed -i "${N},$((N+2))d" "$G"
//   (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)
// done
// git checkout "$G" && rm examples/gno.land/r/nt/grc20routes/v0/caps_test.gno

package grc20routes

import (
	"strings"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

// entries builds n distinct valid entries, so only the maxEntries guard can fire.
func entries(n int) []Entry {
	es := make([]Entry, n)
	for i := 0; i < n; i++ {
		es[i] = Entry{"op_" + strings.Repeat("x", i), "Fn" + strings.Repeat("X", i)}
	}
	return es
}

func TestValidationRejectsOverlongValues(cur realm, t *testing.T) {
	mint(0, cur, "CAPS")

	// Each value is one byte over its constant: 17>16, 33>32, 65>64, 129>128.
	longOp := strings.Repeat("a", maxOpLen+1)
	longFunc := "A" + strings.Repeat("a", maxIdentLen) // exported, so only length can fail
	longSymbol := strings.Repeat("A", maxSymbolLen+1)
	longPrefix := strings.Repeat("a", maxPrefixArg+1)

	cases := []struct {
		name   string
		symbol string
		route  Route
		msg    string
	}{
		{
			"one entry over maxEntries",
			"CAPS",
			Route{Funcs: entries(maxEntries + 1)},
			"grc20routes: too many entry points",
		},
		{
			"operation name over maxOpLen",
			"CAPS",
			Route{Funcs: []Entry{{longOp, "Approve"}}},
			"grc20routes: operation name too long: " + longOp,
		},
		{
			"function name over maxIdentLen",
			"CAPS",
			Route{Funcs: []Entry{{OpApprove, longFunc}}},
			"grc20routes: approve name too long",
		},
		{
			// Shadowed by the grc20reg gate in Register, which refuses any symbol
			// over grc20.MaxSymbolLen (11) anyway. Pins the message, not a bound.
			"symbol over maxSymbolLen",
			longSymbol,
			Route{Funcs: []Entry{{OpApprove, "Approve"}}},
			"grc20routes: symbol too long",
		},
		{
			"prefix argument over maxPrefixArg",
			"CAPS",
			Route{Funcs: []Entry{{OpApprove, "Approve"}}, Prefix: []string{longPrefix}},
			"grc20routes: prefix argument too long",
		},
	}

	for _, tc := range cases {
		r, sym := tc.route, tc.symbol
		uassert.AbortsWithMessage(t, cur, tc.msg,
			func() { Register(cross(cur), sym, r) }, tc.name)
	}
}

// TestValidationAcceptsValuesAtTheCap is the other half: the caps are inclusive,
// so a route sitting exactly on every one of them registers.
func TestValidationAcceptsValuesAtTheCap(cur realm, t *testing.T) {
	mint(0, cur, "ATCAP")

	es := entries(maxEntries)
	es[0] = Entry{strings.Repeat("a", maxOpLen), "A" + strings.Repeat("a", maxIdentLen-1)}
	r := Route{Funcs: es, Prefix: []string{strings.Repeat("a", maxPrefixArg)}}

	uassert.NotAborts(t, cur, func() { Register(cross(cur), "ATCAP", r) },
		"a route exactly on every cap must register")
}
