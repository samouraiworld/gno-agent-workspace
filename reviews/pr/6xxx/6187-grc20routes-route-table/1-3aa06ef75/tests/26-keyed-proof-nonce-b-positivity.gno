// Asserts the B-side of BeginKeyedProof's positivity guard: a zero or negative
// nonceB must refuse the proof. TestKeyedProofRejectsMismatchedInputs passes
// (0, 2), so only the nonceA disjunct is reached, and every other call in the
// suite and in grc20routes_keyed_proof.txtar uses positive nonces.
// Measured at 3aa06ef757b87d731a3e554e96c7540a209fd9a3: passes at head; with the
// nonceB disjunct dropped the package suite is still `ok` without this file and
// FAILs with it.
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// # save this file as examples/gno.land/r/nt/grc20routes/v0/keyed_proof_nonce_b_test.gno
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # ok
// sed -i '367s/ || nonceB <= 0//' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # FAIL, this test alone
// git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// rm examples/gno.land/r/nt/grc20routes/v0/keyed_proof_nonce_b_test.gno

package grc20routes

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

// The mirror of the (0, 2) case in TestKeyedProofRejectsMismatchedInputs, which
// lands on the nonceA disjunct. Without this one a nonceB of zero — the value
// every untouched allowance already holds — or a negative one reaches the stored
// proof, and FinishKeyedProof then asks grc20 for an allowance it cannot hold.
func TestKeyedProofRejectsNonPositiveNonceB(cur realm, t *testing.T) {
	prover := testutils.TestAddress("npos")
	keyA := mint(0, cur, "NPA")
	keyB := mint(0, cur, "NPB")
	testing.SetOriginCaller(prover)

	// Head and post-fix: the nonceB disjunct at grc20routes.gno:367 refuses both.
	// Zero is checked before the already-parked guard, which would otherwise
	// catch it with a different message.
	uassert.AbortsWithMessage(t, cur, "grc20routes: nonces must be positive",
		func() { BeginKeyedProof(cross(cur), keyA, keyB, 2, 0) })

	// Negative: nothing downstream of the positivity guard looks at it, so with
	// the disjunct dropped BeginKeyedProof returns normally here and parks a
	// proof whose keyB leg no Approve can ever satisfy.
	uassert.AbortsWithMessage(t, cur, "grc20routes: nonces must be positive",
		func() { BeginKeyedProof(cross(cur), keyA, keyB, 2, -1) })
}
