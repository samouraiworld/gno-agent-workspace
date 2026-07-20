# Review: PR [#5975](https://github.com/gnolang/gno/pull/5975)
Posted: https://github.com/gnolang/gno/pull/5975#pullrequestreview-4733409532
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Verified on e3b7a7934: a Go port of the three functions returns the same wrapped values the gno run produces, so the overflow is deterministic across validators rather than a consensus split. An exhaustive sweep of three-way splits for every total up to 120 found no case where the payouts sum above the pool, and dust never exceeded 2 units.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5975-parimutuel-payout-math/1-e3b7a7934/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/p/demo/parimutuel/parimutuel.gno:72-77 [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L72-L77) [posted](https://github.com/gnolang/gno/pull/5975#discussion_r3613027222)
Critical: a winner-takes-all payout wraps to negative at a pool of 3037000500 ugnot, about 3037 GNOT, and returns with no panic; `ImpliedProbabilityBps` wraps the same way at an outcome pool of 922337203685478. The [doc comment](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L68-L71) bounds the individual amounts, but the `a*b` product is the real limit. [`overflow.Mul64p`](https://github.com/gnolang/gno/blob/e3b7a7934/gnovm/stdlibs/math/overflow/overflow_generated.gno#L582-L589) is already used for exactly this in [`p/demo/tokens/grc20`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/tokens/grc20/token.gno#L193-L194).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5975 -R gnolang/gno
cat > examples/gno.land/p/demo/parimutuel/overflow_test.gno <<'EOF'
package parimutuel

import "testing"

func TestPayoutDoesNotWrapAtLargePools(t *testing.T) {
	const pool = int64(4_000_000_000) // 4000 GNOT, in ugnot
	if got := CalculatePayout(pool, pool, pool); got != pool {
		t.Fatalf("winner takes all: got %d, want %d", got, pool)
	}
}

func TestImpliedProbabilityDoesNotWrapAtLargePools(t *testing.T) {
	const pool = int64(1_000_000_000_000_000)
	if got := ImpliedProbabilityBps(pool, pool); got != 10000 {
		t.Fatalf("whole pool on one outcome: got %d bps, want 10000", got)
	}
}
EOF
cd examples && go run ../gnovm/cmd/gno test ./gno.land/p/demo/parimutuel; cd ..
rm examples/gno.land/p/demo/parimutuel/overflow_test.gno
```

```
--- FAIL: TestPayoutDoesNotWrapAtLargePools (0.00s)
winner takes all: got -611686018, want 4000000000
--- FAIL: TestImpliedProbabilityDoesNotWrapAtLargePools (0.00s)
whole pool on one outcome: got -8446 bps, want 10000
FAIL    ./gno.land/p/demo/parimutuel
```
</details>

## examples/gno.land/p/demo/parimutuel/parimutuel.gno:40 [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L40) [posted](https://github.com/gnolang/gno/pull/5975#discussion_r3613027226)
Critical: any outcome holding more than `10000-rakeBps` of the pool panics instead of paying out, so the market cannot be settled. `netPool` goes in as `totalPool`, so [the `winningPool cannot exceed totalPool` guard](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L15-L17) compares a pre-rake quantity against a post-rake one. With a 5% rake on 1000, a winning pool of 951 panics, and [the winner-takes-all case](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L5-L10) panics under any nonzero rake.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5975 -R gnolang/gno
cat > examples/gno.land/p/demo/parimutuel/rake_test.gno <<'EOF'
package parimutuel

import "testing"

func TestRakeWithDominantWinningPool(t *testing.T) {
	// 5% rake on a pool of 1000 leaves 950; 960 of the 1000 sits on the winner.
	if got := CalculatePayoutWithRake(960, 1000, 960, 500); got != 950 {
		t.Fatalf("got %d, want 950", got)
	}
}
EOF
cd examples && go run ../gnovm/cmd/gno test ./gno.land/p/demo/parimutuel; cd ..
rm examples/gno.land/p/demo/parimutuel/rake_test.gno
```

```
panic: parimutuel: winningPool cannot exceed totalPool
Stacktrace:
CalculatePayout<VPBlock(3,0)>(userBet,netPool,winningPool)
    gno.land/p/demo/parimutuel/parimutuel.gno:16
CalculatePayoutWithRake<VPBlock(4,1)>(960,1000,960,500)
    gno.land/p/demo/parimutuel/parimutuel.gno:40
```
</details>

## examples/gno.land/p/demo/parimutuel/parimutuel.gno:36-39 [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L36-L39) [posted](https://github.com/gnolang/gno/pull/5975#discussion_r3613027233)
A negative `totalPool` returns 0 instead of panicking: `netPool` is computed before any validation, so `CalculatePayoutWithRake(100, -1000, 400, 0)` short-circuits at the `netPool <= 0` check while [`CalculatePayout` rejects the same input](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L11). A caller with the sign wrong gets silent zeros everywhere.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5975 -R gnolang/gno
cat > examples/gno.land/p/demo/parimutuel/neg_test.gno <<'EOF'
package parimutuel

import "testing"

func TestNegativeTotalPoolRejected(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("negative totalPool accepted")
		}
	}()
	t.Logf("CalculatePayoutWithRake(100, -1000, 400, 0) = %d", CalculatePayoutWithRake(100, -1000, 400, 0))
}
EOF
cd examples && go run ../gnovm/cmd/gno test ./gno.land/p/demo/parimutuel; cd ..
rm examples/gno.land/p/demo/parimutuel/neg_test.gno
```

```
--- FAIL: TestNegativeTotalPoolRejected (0.00s)
CalculatePayoutWithRake(100, -1000, 400, 0) = 0
negative totalPool accepted
FAIL    ./gno.land/p/demo/parimutuel
```
</details>

## examples/gno.land/p/demo/parimutuel/parimutuel_test.gno:5-10 [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L5-L10) [posted](https://github.com/gnolang/gno/pull/5975#discussion_r3613027240)
Missing test: no case uses a pool large enough for the product in `mulDiv` to leave int64, or a winning pool above `10000-rakeBps` of the total. The largest pool in the suite is 1000, six orders of magnitude below the wrap threshold, and the largest raked `winningPool` is 40% of the total.

<details><summary>test cases</summary>

```go
func TestPayoutDoesNotWrapAtLargePools(t *testing.T) {
	const pool = int64(4_000_000_000) // 4000 GNOT, in ugnot

	if got := CalculatePayout(pool, pool, pool); got != pool {
		t.Fatalf("winner takes all: got %d, want %d", got, pool)
	}
}

func TestImpliedProbabilityDoesNotWrapAtLargePools(t *testing.T) {
	const pool = int64(1_000_000_000_000_000) // 1e9 GNOT, in ugnot

	if got := ImpliedProbabilityBps(pool, pool); got != 10000 {
		t.Fatalf("whole pool on one outcome: got %d bps, want 10000", got)
	}
	if got := ImpliedProbabilityBps(pool/4, pool); got != 2500 {
		t.Fatalf("quarter pool on one outcome: got %d bps, want 2500", got)
	}
}

func TestRakeWithEveryoneOnTheWinner(t *testing.T) {
	// 1% rake, all 1000 staked on the outcome that wins.
	if got := CalculatePayoutWithRake(1000, 1000, 1000, 100); got != 990 {
		t.Fatalf("got %d, want 990", got)
	}
}

func TestRakeWithDominantWinningPool(t *testing.T) {
	// 5% rake leaves 950; a winning pool of 960 is still a valid market.
	if got := CalculatePayoutWithRake(960, 1000, 960, 500); got != 950 {
		t.Fatalf("got %d, want 950", got)
	}
}
```
</details>

## examples/gno.land/p/demo/parimutuel/parimutuel.gno:1 [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L1) [posted](https://github.com/gnolang/gno/pull/5975#discussion_r3613027244)
Nit: the package has no `// Package parimutuel ...` comment, so its generated docs page opens on nothing. Sibling [`p/demo/svg`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/svg/doc.gno#L2) keeps one in a `doc.gno`.

## examples/gno.land/p/demo/parimutuel/parimutuel.gno:73-75 [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L73-L75) [posted](https://github.com/gnolang/gno/pull/5975#discussion_r3613027247)
Nit: the `c == 0` panic is unreachable. All three call sites, [`CalculatePayout`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L12-L14), [`ImpliedProbabilityBps`](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L52-L54) and [the rake path](https://github.com/gnolang/gno/blob/e3b7a7934/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L36), pass a positive `c`.

## examples/gno.land/p/demo/parimutuel/parimutuel_test.gno:84-91 [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel_test.gno#L84-L91) [posted](https://github.com/gnolang/gno/pull/5975#discussion_r3613027252)
Nit: the name says implied probabilities sum to 10000, but every split here divides exactly. `ImpliedProbabilityBps` rounds down, so three outcomes of 1000 in a 3000 pool each return 3333 and sum to 9999.

## examples/gno.land/p/demo/parimutuel/parimutuel.gno:32 [↗](../../../../../.worktrees/gno-review-5975/examples/gno.land/p/demo/parimutuel/parimutuel.gno#L32) [posted](https://github.com/gnolang/gno/pull/5975#discussion_r3613027257)
Suggestion: nothing exported returns the commission itself, only the payout net of it, so every realm routing the house cut re-derives `totalPool * rakeBps / 10000` inline with its own rounding.
