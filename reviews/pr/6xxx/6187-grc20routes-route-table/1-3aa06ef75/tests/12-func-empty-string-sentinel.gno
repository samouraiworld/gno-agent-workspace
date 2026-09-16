// Func's doc at grc20routes.gno:492-493 says its "" means "the token does not
// expose that operation to users". On the SourceConvention fallback it means
// "nothing was ever recorded", and the two are the same value with no second
// result to separate them. Measured at head 3aa06ef75 against the real
// r/gnoland/wugnot token, which exports Deposit at wugnot.gno:31 and has no
// route recorded: Func(wugnotKey, "deposit") == "", identical to the "" a realm
// gets for an operation it deliberately declared out. This file PASSES at head
// — pinning the conflation is the finding; the post-fix pair is commented below.
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// cp <this file> examples/gno.land/r/nt/grc20routes/v0/z12_func_sentinel_test.gno
// go build -o /tmp/gno ./gnovm/cmd/gno
// (cd examples && /tmp/gno test -v \
//   -run TestFuncEmptyStringConflatesUnknownWithAbsent ./gno.land/r/nt/grc20routes/v0)
// rm examples/gno.land/r/nt/grc20routes/v0/z12_func_sentinel_test.gno
//
// Observed: --- PASS: TestFuncEmptyStringConflatesUnknownWithAbsent

package grc20routes

import (
	"testing"

	"gno.land/p/nt/uassert/v0"

	"gno.land/r/gnoland/wugnot"
	"gno.land/r/nt/grc20reg/v0"
)

func TestFuncEmptyStringConflatesUnknownWithAbsent(cur realm, t *testing.T) {
	wkey := "gno.land/r/gnoland/wugnot." + wugnot.Token.GetSymbol()
	uassert.True(t, grc20reg.Get(wkey) != nil, "wugnot is a registered token")

	// Nothing in this diff calls Register, so every token deployed today,
	// wugnot included, resolves through the three-op convention fallback.
	_, wsource := Get(wkey)
	uassert.Equal(t, SourceConvention, wsource, "no route is recorded for wugnot")

	// wugnot really does export Deposit(cur realm) at wugnot.gno:31, and the
	// package doc (:69-71) names it as the case a wallet must discover.
	unknown := Func(wkey, "deposit")

	// A realm that DID declare, and deliberately left transfer_from out.
	akey := mint(0, cur, "AONE")
	Register(cross(cur), "AONE", Route{Funcs: []Entry{
		{OpApprove, CanonicalApprove},
		{OpTransfer, CanonicalTransfer},
	}})
	absent := Func(akey, OpTransferFrom)

	// A caller typo against a route that declares every operation it has.
	ckey := mint(0, cur, "CTRE")
	Register(cross(cur), "CTRE", Canonical().With("deposit", "Deposit"))
	typo := Func(ckey, "depsoit")

	// Head: three states, one value, no way to tell them apart at the call site.
	uassert.Equal(t, "", unknown, "wugnot exports Deposit; the route table has no idea")
	uassert.Equal(t, "", absent, "AONE really does not expose transfer_from")
	uassert.Equal(t, "", typo, "CTRE exposes Deposit under the op it declared")
	uassert.True(t, unknown == absent && absent == typo,
		"unknown, absent and misspelled are indistinguishable at the call site")

	// Post-fix (uncomment once Func reports whether it found the entry point,
	// e.g. `func Func(tokenKey, op string) (string, bool)`):
	//
	// _, okUnknown := Func(wkey, "deposit")
	// _, okAbsent := Func(akey, OpTransferFrom)
	// uassert.False(t, okAbsent, "AONE declared its set; transfer_from is not in it")
	// uassert.False(t, okUnknown, "nothing is recorded for wugnot, so nothing is known")
	// name, okApprove := Func(wkey, OpApprove)
	// uassert.True(t, okApprove)
	// uassert.Equal(t, CanonicalApprove, name)
}
