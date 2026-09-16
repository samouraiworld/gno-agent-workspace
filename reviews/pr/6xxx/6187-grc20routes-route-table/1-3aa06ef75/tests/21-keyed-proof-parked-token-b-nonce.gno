// Asserts the B-side of BeginKeyedProof's already-parked guard: an allowance
// equal to nonceB sitting on token B before the proof opens must refuse it.
// TestKeyedProofRejectsPreExistingAllowance parks token A only, and no fixture
// parks anything before Begin. Measured at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
// passes at head; with the tokenKeyB disjunct dropped the package suite is still
// `ok` without this file and FAILs with it.
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// # save this file as examples/gno.land/r/nt/grc20routes/v0/keyed_proof_parked_b_test.gno
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # ok
// sed -i '373s/ || allowanceOf(tokenKeyB, prover) == nonceB//' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # FAIL, this test alone
// git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// rm examples/gno.land/r/nt/grc20routes/v0/keyed_proof_parked_b_test.gno

package grc20routes

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

// The mirror of TestKeyedProofRejectsPreExistingAllowance, which parks token A
// and lands on the tokenKeyA disjunct. Without this one, a prover who parked
// nonceB on token B beforehand opens a proof that only needs token A to move
// during it, and one unkeyed Approve then marks the realm keyed.
func TestKeyedProofRejectsPreExistingAllowanceOnTokenB(cur realm, t *testing.T) {
	prover := testutils.TestAddress("staleb")
	keyA := mint(0, cur, "SBA")
	keyB := mint(0, cur, "SBB")
	approveAs("SBB", prover, 666) // parked on token B alone, before the proof opens
	testing.SetOriginCaller(prover)

	// Head and post-fix: the tokenKeyB disjunct at grc20routes.gno:373 refuses it.
	uassert.AbortsWithMessage(t, cur,
		"grc20routes: nonce already parked on the probe; pick another",
		func() { BeginKeyedProof(cross(cur), keyA, keyB, 555, 666) })

	// With that disjunct dropped, BeginKeyedProof returns normally here, so the
	// assertion above is what turns the mutation red.
}
