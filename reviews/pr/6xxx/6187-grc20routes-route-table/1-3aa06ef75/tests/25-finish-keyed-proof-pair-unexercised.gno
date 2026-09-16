// Asserts FinishKeyedProof's pair check at grc20routes.gno:401: a prover who
// opened (A,B) must be refused when finishing with a different pair at the same
// realm. Every Finish in the suite and in the txtar passes exactly the keys its
// Begin got. Measured at 3aa06ef757b87d731a3e554e96c7540a209fd9a3: passes at
// head; with lines 401-403 deleted the package suite is `ok` 20/20 and
// grc20routes_keyed_proof.txtar PASSes, and only this file turns that red.
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// # the integration fixture first, with the guard gone and this file absent:
// sed -i '401,403d' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20routes_keyed_proof' -p 1 -parallel 1   # ok
// git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// # then the package suite, with this file saved as
// # examples/gno.land/r/nt/grc20routes/v0/keyed_proof_pair_test.gno
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # ok
// sed -i '401,403d' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # FAIL, this test alone
// git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// rm examples/gno.land/r/nt/grc20routes/v0/keyed_proof_pair_test.gno

package grc20routes

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

// The pending entry is keyed by (prover, realm path), not by the token pair, so
// any registered token of that realm reaches it. tokenKeyB reaches nothing else
// at all: drop the check and it is an argument FinishKeyedProof never reads.
func TestFinishKeyedProofRejectsADifferentPair(cur realm, t *testing.T) {
	prover := testutils.TestAddress("pair")
	keyA := mint(0, cur, "FPA")
	keyB := mint(0, cur, "FPB")
	keyC := mint(0, cur, "FPC") // a third token on the same realm, never in the proof
	testing.SetOriginCaller(prover)

	BeginKeyedProof(cross(cur), keyA, keyB, 1111, 2222)
	approveAs("FPA", prover, 1111)
	approveAs("FPB", prover, 2222)

	// Head and post-fix: the pair check at grc20routes.gno:401 refuses both.
	// With it gone each call commits the (A,B) proof instead, since the nonce
	// checks below it read p.keyA and p.keyB, not the arguments.
	uassert.AbortsWithMessage(t, cur,
		"grc20routes: tokens do not match the proof in progress",
		func() { FinishKeyedProof(cross(cur), keyA, keyC) }) // B-side: the ignored argument
	uassert.AbortsWithMessage(t, cur,
		"grc20routes: tokens do not match the proof in progress",
		func() { FinishKeyedProof(cross(cur), keyC, keyB) }) // A-side: same realm, same pending key

	// The pair that was opened still finishes, so the two aborts above are the
	// guard and not a proof this test broke.
	FinishKeyedProof(cross(cur), keyA, keyB)
	uassert.True(t, IsKeyedRealm(selfPath))
}
