# Review: PR [#5999](https://github.com/gnolang/gno/pull/5999)
Event: COMMENT

## Body
The clamp holds up on 28dc8f9fd. A 91584-case sweep of [`calcBlockGasPrice`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L412) against the merge base sorts all 19585 differing cases into the four intended causes and finds no other, and the merge base panics in every one of the 14976 cases where this branch does not. Putting the merge base's panic back in place of the clamp halts the node at full block 1002, and from the cap 407 idle blocks walk the price back to the floor. The same block sequence driven through the full ABCI cycle at a block gas limit of 1000000 produces identical app hashes at every height on both binaries.

One question the diff leaves open: on a chain with [`Block.MaxGas`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/bft/types/params.go#L69-L71) set to -1, master keeps climbing the price while this branch freezes it, so a mixed-version network there forks rather than halts. Nothing in the tree configures -1, so it is a release-note line rather than a code change.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5999-block-gas-price-ratchet/2-28dc8f9fd/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## tm2/pkg/sdk/auth/keeper.go:481-485 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L481-L485)
A chain that reaches the cap rejects every transaction above 1000 gas wanted, whatever fee it offers, and emits nothing an operator can see. The clamp logs nothing, and at the cap the returned price equals the stored one, so [the skip](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L389-L391) returns before [`SetGasPrice`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L347) and [`logTelemetry`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L528) on every later block. The read-path hook is the one surviving signal, and it records [`gp.Gas`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L537-L541) into the [`BlockGasPriceAmount`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/telemetry/metrics/metrics.go#L90-L91) histogram, which is the constant 1000 denominator and not the price.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5999 -R gnolang/gno
cat > tm2/pkg/std/zz_gte_test.go <<'EOF'
package std

import (
	"math"
	"testing"
)

func TestZZGTEAtCap(t *testing.T) {
	block := GasPrice{Gas: 1000, Price: Coin{Amount: math.MaxInt64, Denom: "ugnot"}}
	for _, gasWanted := range []int64{1000, 10_000, 1_000_000} {
		fee := GasPrice{Gas: gasWanted, Price: Coin{Amount: math.MaxInt64, Denom: "ugnot"}}
		ok, err := fee.IsGTE(block)
		t.Logf("gasWanted=%-9d fee=MaxInt64 ugnot -> clears=%v err=%v", gasWanted, ok, err)
	}
}
EOF
go test -v -run TestZZGTEAtCap ./tm2/pkg/std/
rm tm2/pkg/std/zz_gte_test.go
```

```
=== RUN   TestZZGTEAtCap
    zz_gte_test.go:13: gasWanted=1000      fee=MaxInt64 ugnot -> clears=true err=<nil>
    zz_gte_test.go:13: gasWanted=10000     fee=MaxInt64 ugnot -> clears=false err=<nil>
    zz_gte_test.go:13: gasWanted=1000000   fee=MaxInt64 ugnot -> clears=false err=<nil>
--- PASS: TestZZGTEAtCap (0.00s)
```
</details>

## tm2/pkg/sdk/auth/keeper.go:478 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L478)
The floor compares raw `Price.Amount` while a [`GasPrice`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/std/gasprice.go#L11-L14) is a ratio of amount to gas, so a floor configured as 100ugnot/2000gas, 0.05 per gas, settles an idle chain at 100ugnot/1000gas, 0.10 per gas. [The early return above](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L464-L466) goes the other way and rewrites the stored denominator to the initial price's. Both reproduce on master, but this diff owns the floor line now, and [`TestCalcBlockGasPriceFloorAboveOne`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper_test.go#L507) sets `Gas: 1000` on both sides, so the file's only test of a floor above 1 pins the amount contract rather than the price one.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5999 -R gnolang/gno
cat > tm2/pkg/sdk/auth/zz_denom_test.go <<'EOF'
package auth

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestZZDenomFloor(t *testing.T) {
	gk := GasPriceKeeper{}
	params := Params{
		TargetGasRatio:            70,
		GasPricesChangeCompressor: 10,
		InitialGasPrice:           std.GasPrice{Gas: 2000, Price: std.Coin{Amount: 100, Denom: "ugnot"}},
	}
	p := std.GasPrice{Gas: 1000, Price: std.Coin{Amount: 120, Denom: "ugnot"}}
	for range 12 {
		p = gk.calcBlockGasPrice(p, 0, 3_000_000_000, params)
	}
	t.Logf("floor configured 0.0500/gas, settled at %.4f/gas (%v)", float64(p.Price.Amount)/float64(p.Gas), p)
}
EOF
go test -v -run TestZZDenomFloor ./tm2/pkg/sdk/auth/
rm tm2/pkg/sdk/auth/zz_denom_test.go
```

```
=== RUN   TestZZDenomFloor
    zz_denom_test.go:20: floor configured 0.0500/gas, settled at 0.1000/gas (100ugnot/1000gas)
--- PASS: TestZZDenomFloor (0.00s)
```
</details>

## tm2/pkg/sdk/auth/keeper_test.go:204-213 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper_test.go#L204-L213)
Missing test: the descent from the cap, which is what makes the clamp a pause rather than a trap. [`require.Less`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper_test.go#L212) fails only if the first idle block leaves the price at `math.MaxInt64`, so a later change to the floor, the compressor or the min-1 decrement could strand the price near the cap with this still green. Nothing in the file asserts the other half of the assumption, that the decrease branch cannot reach [a clamp](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L481-L485) that hardcodes `math.MaxInt64` for both branches.

<details><summary>test cases</summary>

```go
func TestCalcBlockGasPriceCapDescends(t *testing.T) {
	gk := GasPriceKeeper{}
	const maxGas = int64(3_000_000_000)
	params := Params{
		TargetGasRatio:            70,
		GasPricesChangeCompressor: 10,
		InitialGasPrice:           std.GasPrice{Gas: 1000, Price: std.Coin{Amount: 1, Denom: "ugnot"}},
	}
	p := std.GasPrice{Gas: 1000, Price: std.Coin{Amount: 1, Denom: "ugnot"}}
	for p.Price.Amount != math.MaxInt64 {
		p = gk.calcBlockGasPrice(p, maxGas, maxGas, params)
	}
	require.Equal(t, int64(1000), p.Gas)
	require.Equal(t, "ugnot", p.Price.Denom)

	idle := 0
	for p.Price.Amount != 1 {
		p = gk.calcBlockGasPrice(p, 0, maxGas, params)
		idle++
		require.Less(t, idle, 1000, "the cap is a trap: the price does not return to the floor")
	}
}

// Every decrease result lands in [1, max(last, initial)], so the branch cannot
// reach the clamp even at the largest initial price Params.Validate accepts.
func TestCalcBlockGasPriceDecreaseStaysBounded(t *testing.T) {
	gk := GasPriceKeeper{}
	const maxGas = int64(3_000_000_000)
	for _, last := range []int64{1, 2, 1000, math.MaxInt64 / 2, math.MaxInt64 - 1, math.MaxInt64} {
		for _, initial := range []int64{0, 1, 1000, math.MaxInt64} {
			params := Params{
				TargetGasRatio:            70,
				GasPricesChangeCompressor: 10,
				InitialGasPrice:           std.GasPrice{Gas: 1000, Price: std.Coin{Amount: initial, Denom: "ugnot"}},
			}
			got := gk.calcBlockGasPrice(std.GasPrice{Gas: 1000, Price: std.Coin{Amount: last, Denom: "ugnot"}}, 0, maxGas, params)
			require.LessOrEqual(t, got.Price.Amount, max(last, initial))
			require.GreaterOrEqual(t, got.Price.Amount, int64(1))
		}
	}
}
```

Both are green at 28dc8f9fd. At the merge base the first panics with `The min gas price is out of int64 range` and the second fails on `"0" is not greater than or equal to "1"`, since the floor there is the initial price alone.
</details>

## tm2/pkg/sdk/auth/keeper.go:409 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L409)
Nit: "we floor the result at the initial gas price, or at 1" reads as a choice between the two, and [the code](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L478) takes the larger of them. An initial price of 1000 floors at 1000.

## tm2/pkg/sdk/auth/keeper.go:435-439 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L435-L439)
Nit: the guard fires on the target, not on the limit, so a bounded `MaxGas` of 1 freezes the price too, and every `MaxGas` from 1 to 99 does at a [`TargetGasRatio`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/params.go#L115-L117) of 1. At `MaxGas` -1 the target is -1 and dividing by it is legal, so the real harm there is the ratchet, which this comment does not name.

## tm2/pkg/sdk/auth/keeper.go:460 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper.go#L460)
Nit: the XXX asks whether to cap the increase branch, while the doc comment at [`keeper.go:408-409`](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L408-L409) now says that branch caps at the largest int64 price. The policy ceiling and the representational one are different questions, so the two lines now read as a contradiction.

## tm2/pkg/sdk/auth/keeper_test.go:450 [↗](../../../../../.worktrees/gno-review-5999/tm2/pkg/sdk/auth/keeper_test.go#L450)
Nit: the subtest's two `gasUsed=0` rows pass on master too, where the usage equals the target and [the `Cmp == 0` check](https://github.com/gnolang/gno/blob/28dc8f9fd/tm2/pkg/sdk/auth/keeper.go#L443-L445) returns before either division. The comment says both branches divide by the target, which holds for the other four rows and not for those two.
