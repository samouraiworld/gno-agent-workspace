# Review: PR [#5916](https://github.com/gnolang/gno/pull/5916)
Event: COMMENT

## Body
Verified on 7e6955997: restoring the old `gasUsed <= 0` guard leaves the stored price stuck at 10 on an empty block, while the fix decays it through [`UpdateGasPrice`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L361) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L361).

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5916-symmetric-gas-price-decay/1-7e6955997/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/sdk/auth/keeper.go:370 [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L370)
Missing test: no test drives [`UpdateGasPrice`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L361) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L361) on an empty block, so the guard relaxation from `gasUsed <= 0` to `gasUsed < 0` has no direct coverage. That guard is the line that changes consensus behavior on empty blocks, yet the [committed regression test](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper_test.go#L189) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper_test.go#L189) reaches only [`calcBlockGasPrice`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper.go#L396) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper.go#L396).

<details><summary>test cases</summary>

Paste into [`tm2/pkg/sdk/auth/keeper_test.go`](https://github.com/gnolang/gno/blob/7e6955997/tm2/pkg/sdk/auth/keeper_test.go#L189) · [↗](../../../../../.worktrees/gno-review-5916/tm2/pkg/sdk/auth/keeper_test.go#L189):

```go
func TestUpdateGasPriceEmptyBlockDecays(t *testing.T) {
	env := setupTestEnv()

	params := DefaultParams()
	params.TargetGasRatio = 70
	params.GasPricesChangeCompressor = 10
	params.InitialGasPrice = std.GasPrice{
		Gas:   1000,
		Price: std.Coin{Amount: 1, Denom: "ugnot"},
	}
	ctx := env.ctx.WithValue(AuthParamsContextKey{}, params)
	// Empty block: bounded meter, zero gas consumed.
	ctx = ctx.WithBlockGasMeter(store.NewGasMeter(ctx.ConsensusParams().Block.MaxGas))

	// Seed a price above the floor.
	env.gk.SetGasPrice(ctx, std.GasPrice{
		Gas:   1000,
		Price: std.Coin{Amount: 10, Denom: "ugnot"},
	})

	env.gk.UpdateGasPrice(ctx)

	require.Equal(t, int64(9), env.gk.LastGasPrice(ctx).Price.Amount,
		"empty block must decay the stored price")
}
```
</details>

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5916 -R gnolang/gno

cat >> tm2/pkg/sdk/auth/keeper_test.go <<'EOF'

func TestUpdateGasPriceEmptyBlockDecays(t *testing.T) {
	env := setupTestEnv()

	params := DefaultParams()
	params.TargetGasRatio = 70
	params.GasPricesChangeCompressor = 10
	params.InitialGasPrice = std.GasPrice{
		Gas:   1000,
		Price: std.Coin{Amount: 1, Denom: "ugnot"},
	}
	ctx := env.ctx.WithValue(AuthParamsContextKey{}, params)
	ctx = ctx.WithBlockGasMeter(store.NewGasMeter(ctx.ConsensusParams().Block.MaxGas))
	env.gk.SetGasPrice(ctx, std.GasPrice{
		Gas:   1000,
		Price: std.Coin{Amount: 10, Denom: "ugnot"},
	})
	env.gk.UpdateGasPrice(ctx)
	require.Equal(t, int64(9), env.gk.LastGasPrice(ctx).Price.Amount,
		"empty block must decay the stored price")
}
EOF

# passes on the fix (empty block decays 10 -> 9)
go test ./tm2/pkg/sdk/auth/ -run TestUpdateGasPriceEmptyBlockDecays -v

# restore the pre-fix empty-block guard: the price now stays stuck at 10
sed -i 's/if gasUsed < 0 {/if gasUsed <= 0 {/' tm2/pkg/sdk/auth/keeper.go
go test ./tm2/pkg/sdk/auth/ -run TestUpdateGasPriceEmptyBlockDecays -v

git checkout -- tm2/pkg/sdk/auth/keeper.go tm2/pkg/sdk/auth/keeper_test.go
```

```
=== RUN   TestUpdateGasPriceEmptyBlockDecays
--- PASS: TestUpdateGasPriceEmptyBlockDecays (0.00s)
ok      github.com/gnolang/gno/tm2/pkg/sdk/auth

# after restoring the old <= 0 guard:
=== RUN   TestUpdateGasPriceEmptyBlockDecays
    keeper_test.go:  expected: 9   actual: 10
        empty block must decay the stored price
--- FAIL: TestUpdateGasPriceEmptyBlockDecays (0.00s)
FAIL
```
</details>
