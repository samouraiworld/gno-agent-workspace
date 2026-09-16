// Asserts the proven route prefix for a token a realm registers while operating
// under a sub identity whose subpath contains a dot. The registry key is
// "gno.land/r/nt/grc20routes/v0#a.b.SUBTK"; Get returns Prefix ["b.SUBTK"]
// instead of the symbol "SUBTK", and JSON publishes it. Passes at the reviewed
// head as written; uncomment the "fixed" line and it fails there.
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// cp 36-subrealm-dot-prefix_test.gno examples/gno.land/r/nt/grc20routes/v0/
// (cd examples && go run ../gnovm/cmd/gno test -v -run TestProvenPrefixFromDotSubpath ./gno.land/r/nt/grc20routes/v0)
// rm examples/gno.land/r/nt/grc20routes/v0/36-subrealm-dot-prefix_test.gno

package grc20routes

import (
	"strings"
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestProvenPrefixFromDotSubpath(cur realm, t *testing.T) {
	prover := testutils.TestAddress("subdot")

	// chain.isValidSubpath allows "." inside a segment, so "a.b" is a legal
	// subpath and a realm may register its tokens under that identity.
	sub := cur.Sub("a.b")
	keyA := mint(0, sub, "SUBTK") // keys in grc20reg as "<selfPath>#a.b.SUBTK"
	keyB := mint(0, cur, "SUBB")  // same host realm, primary identity

	urequire.Equal(t, selfPath+"#a.b.SUBTK", keyA)
	urequire.Equal(t, selfPath, PkgPath(keyA), "the MsgCall destination is the host realm")

	// The proven path emits a prefix the declared path refuses outright.
	uassert.AbortsWithMessage(t, cur, "grc20routes: invalid symbol character: .",
		func() { Register(cross(cur), "b.SUBTK", Keyed("b.SUBTK")) })

	testing.SetOriginCaller(prover)
	BeginKeyedProof(cross(cur), keyA, keyB, 777, 888)
	approveAs("SUBTK", prover, 777)
	approveAs("SUBB", prover, 888)
	FinishKeyedProof(cross(cur), keyA, keyB)

	r, source := Get(keyA)
	urequire.Equal(t, SourceProven, source)
	urequire.Equal(t, 1, len(r.Prefix))

	// fqname.Parse cuts at the FIRST dot after the last slash, so the subpath
	// tail lands in the prefix the wallet sends as the leading MsgCall argument.
	uassert.Equal(t, "b.SUBTK", r.Prefix[0], "head: the subpath tail is in the prefix")
	// uassert.Equal(t, "SUBTK", r.Prefix[0], "fixed: the prefix is the symbol")

	// The same value reaches ABCI consumers.
	uassert.True(t, strings.Contains(JSON(keyA), `"prefix":["b.SUBTK"]`),
		"head: JSON publishes the malformed prefix")
}
