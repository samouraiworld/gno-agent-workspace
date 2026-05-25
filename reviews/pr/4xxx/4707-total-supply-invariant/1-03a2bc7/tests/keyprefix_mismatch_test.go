// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a local clone of gnolang/gno:
gh pr checkout 4707 -R gnolang/gno && git checkout 03a2bc766
curl -fsSL -o tm2/pkg/sdk/bank/keyprefix_mismatch_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4707-total-supply-invariant/1-03a2bc7/tests/keyprefix_mismatch_test.go
go test -v -run 'TestBalanceChangeInvariantKeyPrefixMismatch' ./tm2/pkg/sdk/bank/
rm tm2/pkg/sdk/bank/keyprefix_mismatch_test.go
*/

package bank

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
)

// TestBalanceChangeInvariantKeyPrefixMismatch demonstrates that the writer
// (addBalanceChanges) and the reader (BalanceChangeInvariant) disagree on the
// storage key. Writes go to plain "balanceIncrease"/"balanceDecrease", reads
// go to storeKey(...) which prepends StoreKeyPrefix="/bk/". As a result the
// invariant never observes a real mismatch — even a one-sided transfer
// (deliberate inflation) is reported as balanced.
func TestBalanceChangeInvariantKeyPrefixMismatch(t *testing.T) {
	env := setupTestEnv()
	ctx := env.ctx

	// Track one denom in TotalSupply.
	params := env.bankk.GetParams(ctx)
	params.TotalSupply = std.NewCoins(std.NewCoin("foo", 1000))
	env.bankk.SetParams(ctx, params)

	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)

	// One-sided "inflation": AddCoins without a matching SubtractCoins.
	// inc=foo:100, dec=empty => invariant SHOULD be broken.
	_, err := env.bankk.AddCoins(ctx, addr, std.NewCoins(std.NewCoin("foo", 100)))
	if err != nil {
		t.Fatalf("AddCoins: %v", err)
	}

	// Confirm counters were written under the unprefixed key.
	store := ctx.Store(env.bankk.key)
	rawInc := store.Get([]byte(balanceIncKey))
	if rawInc == nil {
		t.Fatalf("expected inc bytes under raw key %q, got nil", balanceIncKey)
	}

	// Read under the *prefixed* key that BalanceChangeInvariant uses.
	prefixedInc := store.Get(storeKey(balanceIncKey))
	if prefixedInc != nil {
		t.Fatalf("expected nil under prefixed key (bug presupposes mismatch); got %v", prefixedInc)
	}

	// Run the invariant: it reads the (empty) prefixed key, decides nothing
	// changed, and returns broken=false even though 100foo were minted.
	inv := BalanceChangeInvariant(env.bankk)
	msg, broken := inv(ctx)
	if broken {
		t.Fatalf("expected invariant to silently pass due to key prefix mismatch; msg=%q", msg)
	}
	// IS:     bug — invariant silently passes after one-sided AddCoins.
	// SHOULD: invariant breaks (broken==true) since inc != dec.
}
