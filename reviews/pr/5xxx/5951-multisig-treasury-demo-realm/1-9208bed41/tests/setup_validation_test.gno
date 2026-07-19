/* Run: from a gno checkout:
gh pr checkout 5951 -R gnolang/gno && git checkout 9208bed41
curl -fsSL -o examples/gno.land/r/demo/multisig/setup_validation_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5951-multisig-treasury-demo-realm/1-9208bed41/tests/setup_validation_test.gno
cd examples/gno.land/r/demo/multisig && mv multisig_test.gno /tmp/ && gno test -v . ; mv /tmp/multisig_test.gno .
rm setup_validation_test.gno
*/

// Setup takes three owner addresses and hardcodes ownerCount to 3
// without checking they are distinct, and Propose stores the payout
// amount without checking its sign. At 9208bed41 every rejection below
// is missing, so the first assertion fails.
// Setup is one-shot and package state is shared across test files, so
// this file has to replace multisig_test.gno rather than sit beside it:
// the shipped suite spends the single Setup call in its first test.
package multisig_test

import (
	"testing"

	"gno.land/r/demo/multisig"
)

const (
	ownerA = address("g1owner1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	ownerB = address("g1owner2xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	ownerC = address("g1owner3xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	payee  = address("g1recipientxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
)

func mustPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic: %s", what)
		}
	}()
	fn()
}

func TestSetupAndProposeValidation(cur realm, t *testing.T) {
	// Three copies of one address leave a single owner who can approve
	// once, so a 2-of-3 treasury can never reach its own threshold.
	mustPanic(t, "duplicate owners", func() {
		multisig.Setup(cross(cur), ownerA, ownerA, ownerA, 2)
	})

	// Two distinct owners cannot satisfy a threshold of 3.
	mustPanic(t, "threshold above distinct owner count", func() {
		multisig.Setup(cross(cur), ownerA, ownerA, ownerB, 3)
	})

	multisig.Setup(cross(cur), ownerA, ownerB, ownerC, 2)

	// The bank keeper refuses a non-positive send, so these proposals
	// are dead on arrival. Owners should not spend approvals on them.
	testing.SetRealm(testing.NewUserRealm(ownerA))
	mustPanic(t, "negative amount", func() {
		multisig.Propose(cross(cur), "drain", payee, -500)
	})
	mustPanic(t, "zero amount", func() {
		multisig.Propose(cross(cur), "noop", payee, 0)
	})
}
