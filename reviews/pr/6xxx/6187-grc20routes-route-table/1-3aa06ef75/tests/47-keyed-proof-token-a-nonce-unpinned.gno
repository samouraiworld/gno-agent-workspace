// Asserts the token-A half of FinishKeyedProof's load-bearing pair: token B
// carries its nonce, token A does not, so the proof must be refused. The
// package's 20 tests and grc20routes_keyed_proof.txtar only ever reach the
// token-B branch. Measured at 3aa06ef757b87d731a3e554e96c7540a209fd9a3: passes
// at head; with grc20routes.gno:408-410 deleted the suite is still `ok` (20/20)
// without this file and FAILs with it.
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// # save this file as examples/gno.land/r/nt/grc20routes/v0/keyed_proof_side_a_test.gno
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # ok
// sed -i '408,410d' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno        # drop the token-A nonce check
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # FAIL, this test alone
// git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// rm examples/gno.land/r/nt/grc20routes/v0/keyed_proof_side_a_test.gno

package grc20routes

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

// The mirror of TestKeyedProofFailsWhenOnlyOneTokenMoves, which moves token A
// and so lands on the keyB branch at grc20routes.gno:411. This one moves token
// B alone and lands on the keyA branch at grc20routes.gno:408, the branch no
// test in the diff reaches.
func TestKeyedProofFailsWhenOnlyTokenAIsMissing(cur realm, t *testing.T) {
	prover := testutils.TestAddress("sideA")
	keyA := mint(0, cur, "SAA")
	keyB := mint(0, cur, "SAB")
	testing.SetOriginCaller(prover)

	BeginKeyedProof(cross(cur), keyA, keyB, 333, 444)
	approveAs("SAB", prover, 444) // only token B: the side the suite never covers

	// Head, and any correct fix: the token-A check refuses the proof.
	uassert.AbortsWithMessage(t, cur,
		"grc20routes: "+keyA+" does not carry its nonce",
		func() { FinishKeyedProof(cross(cur), keyA, keyB) })

	// With grc20routes.gno:408-410 deleted FinishKeyedProof returns normally
	// here and provenKeyed[selfPath] is set off one token's movement, which is
	// what makes the line above turn that mutation red:
	// uassert.True(t, IsKeyedRealm(selfPath))   // the mutated, wrong behaviour
}
