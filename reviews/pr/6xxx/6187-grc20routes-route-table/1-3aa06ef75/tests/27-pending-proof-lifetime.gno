// Pins the lifetime of grc20routes' `pending` tree: that a finished proof drops
// its entry, and that an unfinishable one never does. Both pass at the reviewed
// head; the first fails the moment `pending.Remove(pk)` (grc20routes.gno:415) is
// deleted, which the shipped suite and the txtar do not notice.
//
// from a local clone of github.com/gnolang/gno:
//
//	git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
//	cp 27-pending-proof-lifetime.gno examples/gno.land/r/nt/grc20routes/v0/pending_lifetime_test.gno
//	(cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/r/nt/grc20routes/v0)   # both pass
//	sed -i '/^\tpending.Remove(pk)$/d' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
//	(cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/r/nt/grc20routes/v0)   # only the first fails
//	git checkout examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
//	rm examples/gno.land/r/nt/grc20routes/v0/pending_lifetime_test.gno

package grc20routes

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

// The line the suite does not cover: FinishKeyedProof consumes the pending
// entry it just verified.
func TestFinishKeyedProofClearsItsPendingEntry(cur realm, t *testing.T) {
	prover := testutils.TestAddress("clears")
	keyA := mint(0, cur, "CLA")
	keyB := mint(0, cur, "CLB")
	testing.SetOriginCaller(prover)

	BeginKeyedProof(cross(cur), keyA, keyB, 101, 202)
	pk := pendingKey(prover, selfPath)
	urequire.True(t, pending.Has(pk), "BeginKeyedProof opens exactly one entry")

	approveAs("CLA", prover, 101)
	approveAs("CLB", prover, 202)
	FinishKeyedProof(cross(cur), keyA, keyB)

	uassert.False(t, pending.Has(pk),
		"a committed proof leaves nothing behind in the pending tree")
}

// The lifetime the package leaves open: a proof whose second allowance never
// lands cannot be finished, and FinishKeyedProof's success path is the only
// caller of pending.Remove.
func TestAbandonedProofsAccumulate(cur realm, t *testing.T) {
	keyA := mint(0, cur, "ABA")
	keyB := mint(0, cur, "ABB")
	before := pending.Size()

	one := testutils.TestAddress("abandon-one")
	testing.SetOriginCaller(one)
	BeginKeyedProof(cross(cur), keyA, keyB, 501, 502)
	BeginKeyedProof(cross(cur), keyA, keyB, 503, 504) // same key: Set overwrites
	uassert.Equal(t, before+1, pending.Size(),
		"one entry per (prover, realm) pair, not per call")

	two := testutils.TestAddress("abandon-two")
	testing.SetOriginCaller(two)
	BeginKeyedProof(cross(cur), keyA, keyB, 505, 506)
	uassert.Equal(t, before+2, pending.Size(), "a second prover adds a second entry")

	// Neither prover ever moved a token allowance, so neither proof can commit.
	uassert.AbortsWithMessage(t, cur, "grc20routes: "+keyA+" does not carry its nonce",
		func() { FinishKeyedProof(cross(cur), keyA, keyB) })

	// Head: the entry outlives the proof, with no cancel, expiry or eviction
	// entry point to reclaim it.
	uassert.True(t, pending.Has(pendingKey(two, selfPath)),
		"an unfinishable proof stays in realm storage for the life of the realm")
	// Post-fix, once a prover can drop a proof they will not finish:
	// uassert.False(t, pending.Has(pendingKey(two, selfPath)),
	//	"the prover reclaims the entry their abandoned proof opened")
}
