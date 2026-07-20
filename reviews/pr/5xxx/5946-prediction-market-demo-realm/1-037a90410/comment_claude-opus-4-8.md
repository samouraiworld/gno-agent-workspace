# Review: PR [#5946](https://github.com/gnolang/gno/pull/5946)
Posted: https://github.com/gnolang/gno/pull/5946#pullrequestreview-4733431140
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Verified on 037a90410: with the `avl.Get` arity fixed so the package builds, a claim at 5,000,000,000 ugnot per side panics inside the banker send, and succeeds once the division comes first. Go returns the same wrapped values, so this is `int64` semantics, not a GnoVM divergence.

[`ClaimWinnings`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L119-L153) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L119) is the only path that moves coins out of the realm, so any case it refuses strands the escrow for good.

Realms under `examples/` are [pre-deployed to gno.land testnets](https://github.com/gnolang/gno/blob/037a90410/examples/README.md?plain=1#L7-L9) · [↗](../../../../../.worktrees/gno-review-5946/examples/README.md#L7), so this holds real testnet ugnot once merged.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5946-prediction-market-demo-realm/1-037a90410/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:74 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L74) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613044977)
Critical: [`avl.Tree.Get`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/p/nt/avl/v0/tree.gno#L58) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/p/nt/avl/v0/tree.gno#L58) returns a single value, so this assignment and the four others at lines [107](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L107) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L107), [121](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L121) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L121), [14](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L14) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L14) and [31](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L31) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L31) fail to typecheck. The four tests the description lists have never run, and [`ci / examples`](https://github.com/gnolang/gno/blob/037a90410/.github/workflows/ci-dir-examples.yml#L7-L10) · [↗](../../../../../.worktrees/gno-review-5946/.github/workflows/ci-dir-examples.yml#L7) will fail once it is released to run.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5946 -R gnolang/gno
cd examples/gno.land/r/demo/predictionmarket
go run ../../../../../gnovm/cmd/gno test -v .
```

```
predictionmarket.gno:74:17: assignment mismatch: 2 variables but markets.Get returns 1 value (code=gnoTypeCheckError)
predictionmarket.gno:107:17: assignment mismatch: 2 variables but markets.Get returns 1 value (code=gnoTypeCheckError)
predictionmarket.gno:121:17: assignment mismatch: 2 variables but markets.Get returns 1 value (code=gnoTypeCheckError)
FAIL    . 	0.62s
FAIL
FAIL: 0 build errors, 1 test errors
```
</details>

## examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:143 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L143) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613044980)
Critical: `b.amount * totalPool` wraps `int64` before the division, so the payout goes negative and the [banker send](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L152) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L152) aborts. The limit is on the product, not on either amount, so 5,000,000,000 ugnot on each of two outcomes already crosses it and every bettor in that market is locked out for good. [`math/overflow`](https://github.com/gnolang/gno/blob/037a90410/gnovm/stdlibs/math/overflow/overflow_generated.gno#L583-L589) · [↗](../../../../../.worktrees/gno-review-5946/gnovm/stdlibs/math/overflow/overflow_generated.gno#L583) turns the wrap into a panic, as [`p/demo/tokens/grc20`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/p/demo/tokens/grc20/token.gno#L193-L194) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/p/demo/tokens/grc20/token.gno#L193) does.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5946 -R gnolang/gno
D=examples/gno.land/r/demo/predictionmarket

# the package does not build yet; apply the avl.Get arity fix first
sed -i 's/raw, exists := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/if !exists {/if raw == nil {/' $D/predictionmarket.gno
sed -i 's/raw, exists := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/raw, _ := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/if !exists {/if raw == nil {/' $D/predictionmarket_test.gno

cat > $D/payout_probe_test.gno <<'EOF'
package predictionmarket

import (
	"chain"
	"chain/banker"
	"strconv"
	"testing"
)

func TestClaimPayoutDoesNotWrap(cur realm, t *testing.T) {
	rlm := chain.PackageAddress("gno.land/r/demo/predictionmarket")
	testing.IssueCoins(rlm, chain.Coins{{"ugnot", 1_000_000_000_000}})
	winner := cur.Previous().Address()
	before := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot")

	// 5000 GNOT on each side: the sole Home bettor takes the whole 10000 GNOT.
	const stake = int64(5_000_000_000)
	id := CreateMarket(cur, "overflow probe")
	m := markets.Get(strconv.Itoa(id)).(*Market)
	m.pools[OutcomeHome] = stake
	m.pools[OutcomeAway] = stake
	m.bets = append(m.bets, &Bet{bettor: winner, outcome: OutcomeHome, amount: stake})
	m.resolved = true
	m.winner = OutcomeHome
	ClaimWinnings(cur, id)

	got := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot") - before
	if got != 2*stake {
		t.Fatalf("payout: got %d, want %d", got, 2*stake)
	}
}
EOF

cd $D && go run ../../../../../gnovm/cmd/gno test -v -run TestClaimPayoutDoesNotWrap . 2>&1 | head -3
cd - && rm $D/payout_probe_test.gno && git checkout $D
```

```
=== RUN   TestClaimPayoutDoesNotWrap
panic: invalid result:  + -1068046444ugnot = -1068046444ugnot: non-positive coin amount: -1068046444
```
</details>

## examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:130-133 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L130) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613044989)
Critical: when the declared outcome drew no bets, every claim panics here and the escrow stays in the realm for good. [`ResolveMarket`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L104-L106) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L104) accepts any outcome in 0..2 without checking that anyone backed it, so a draw between two bettors split across Home and Away hits this on the first try. A market the admin never resolves ends the same way.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5946 -R gnolang/gno
D=examples/gno.land/r/demo/predictionmarket

# the package does not build yet; apply the avl.Get arity fix first
sed -i 's/raw, exists := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/if !exists {/if raw == nil {/' $D/predictionmarket.gno
sed -i 's/raw, exists := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/raw, _ := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/if !exists {/if raw == nil {/' $D/predictionmarket_test.gno

cat > $D/empty_pool_probe_test.gno <<'EOF'
package predictionmarket

import (
	"strconv"
	"testing"
)

func TestClaimWithEmptyWinningPool(cur realm, t *testing.T) {
	bettor := cur.Previous().Address()
	id := CreateMarket(cur, "empty pool probe")
	m := markets.Get(strconv.Itoa(id)).(*Market)
	m.pools[OutcomeHome] = 1_000_000
	m.pools[OutcomeAway] = 1_000_000
	m.bets = append(m.bets, &Bet{bettor: bettor, outcome: OutcomeHome, amount: 1_000_000})
	m.resolved = true
	m.winner = OutcomeDraw // nobody backed Draw

	defer func() { t.Fatalf("claim panicked: %v; the 2000000ugnot escrow has no exit", recover()) }()
	ClaimWinnings(cur, id)
}
EOF

cd $D && go run ../../../../../gnovm/cmd/gno test -v -run TestClaimWithEmptyWinningPool . 2>&1 | grep -E 'RUN|claim panicked|FAIL'
cd - && rm $D/empty_pool_probe_test.gno && git checkout $D
```

```
=== RUN   TestClaimWithEmptyWinningPool
claim panicked: no winning pool; the 2000000ugnot escrow has no exit
--- FAIL: TestClaimWithEmptyWinningPool (0.00s)
FAIL    . 	1.22s
FAIL
```
</details>

## examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:99-103 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L99) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613044993)
Nothing stops the admin from betting, then resolving their own outcome and taking the pot: [`PlaceBet`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L65-L72) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L65) admits the admin like any account, and there is no betting cutoff, oracle or dispute window. The package carries no doc comment saying a bettor is trusting the admin completely.

## examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno:53-62 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno#L53) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613044995)
Missing test: nothing places a bet or completes a claim, so both the overflow and the empty-winning-pool lock reach the chain unchallenged. The admin gate is also vacuous under `gno test`, where `unsafe.OriginCaller()` and `cur.Previous().Address()` are both the empty address.

<details><summary>test cases</summary>

`PlaceBet` cannot be driven from an in-package test, because [`AssertOriginCall`](https://github.com/gnolang/gno/blob/037a90410/gnovm/stdlibs/chain/runtime/native.go#L9-L27) · [↗](../../../../../.worktrees/gno-review-5946/gnovm/stdlibs/chain/runtime/native.go#L9) requires the first frame to be a message call; a txtar integration test is the right home for the bet path. The payout assertions seed the market directly. Red at 037a90410 with the arity fix applied, green once the multiplication stays in range.

```go
package predictionmarket

import (
	"chain"
	"chain/banker"
	"strconv"
	"testing"
)

const realmPath = "gno.land/r/demo/predictionmarket"

// seedResolved builds a resolved market directly, bypassing PlaceBet's
// origin-call guard, which no in-package test can satisfy.
func seedResolved(cur realm, bettor address, stake int64) int {
	id := CreateMarket(cur, "overflow probe")
	m := markets.Get(strconv.Itoa(id)).(*Market)
	m.pools[OutcomeHome] = stake
	m.pools[OutcomeAway] = stake
	m.bets = append(m.bets, &Bet{bettor: bettor, outcome: OutcomeHome, amount: stake})
	m.resolved = true
	m.winner = OutcomeHome
	return id
}

func TestClaimPayoutDoesNotWrap(cur realm, t *testing.T) {
	testing.IssueCoins(chain.PackageAddress(realmPath), chain.Coins{{"ugnot", 1_000_000_000_000}})
	winner := cur.Previous().Address()
	before := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot")

	// 5000 GNOT on each side: the sole Home bettor takes the whole 10000 GNOT.
	const stake = int64(5_000_000_000)
	ClaimWinnings(cur, seedResolved(cur, winner, stake))

	got := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot") - before
	if got != 2*stake {
		t.Fatalf("payout: got %d, want %d", got, 2*stake)
	}
}

func TestClaimPayoutSmallStakeUnchanged(cur realm, t *testing.T) {
	testing.IssueCoins(chain.PackageAddress(realmPath), chain.Coins{{"ugnot", 1_000_000_000_000}})
	winner := cur.Previous().Address()
	before := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot")

	const stake = int64(1_000_000)
	ClaimWinnings(cur, seedResolved(cur, winner, stake))

	got := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot") - before
	if got != 2*stake {
		t.Fatalf("payout: got %d, want %d", got, 2*stake)
	}
}
```
</details>

## examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:155 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L155) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613045002)
Nit: `Render` ignores its `path` argument, so a single market page, a bettor's open positions and a claimable balance are all unreachable from the web view.

## examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:61 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L61) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613045006)
Nit: the tree key is the decimal id and [`Iterate`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/p/nt/avl/v0/tree.gno#L89) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/p/nt/avl/v0/tree.gno#L89) walks keys as strings, so `Render` lists market 10 before market 2.

## examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:44-47 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L44) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613045012)
Suggestion: `admin` is fixed at deploy with no transfer path, so a lost key leaves every open market unresolvable and its escrow stranded. [`p/nt/ownable/v0`](https://github.com/gnolang/gno/blob/037a90410/examples/gno.land/p/nt/ownable/v0/ownable.gno#L1) · [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/p/nt/ownable/v0/ownable.gno#L1) is the usual shape in `examples/`.

## examples/gno.land/r/demo/predictionmarket/predictionmarket.gno:70-72 [↗](../../../../../.worktrees/gno-review-5946/examples/gno.land/r/demo/predictionmarket/predictionmarket.gno#L70) [posted](https://github.com/gnolang/gno/pull/5946#discussion_r3613045017)
Suggestion: the accepted outcomes are hardcoded here, so a market cannot declare itself binary and a bettor can stake on Draw in a contest that cannot draw.
