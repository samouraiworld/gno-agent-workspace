# Review: PR [#5908](https://github.com/gnolang/gno/pull/5908)
Event: REQUEST_CHANGES

## Body
Both failures share one cause: this PR enlarges [`grc20`](https://github.com/gnolang/gno/blob/288cdb044/examples/gno.land/p/demo/tokens/grc20/token.gno#L27) [↗](../../../../../.worktrees/gno-review-5908/examples/gno.land/p/demo/tokens/grc20/token.gno#L27), which loads into the gov/dao/v3 genesis graph through [`v3/treasury`](https://github.com/gnolang/gno/blob/288cdb044/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L6) [↗](../../../../../.worktrees/gno-review-5908/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L6), so exact gas and storage-deposit goldens shift. Verified on 288cdb044: the failing integration tests pass on the parent d1489dc5c and fail on this head, so the drift is from this change, not a pre-existing flake.

One design question, not blocking: was the breaking signature deliberate over keeping `NewToken(0, rlm, name, symbol, decimals)` with `id` defaulting to `symbol` and adding a separate id-aware constructor? That decouples the identifier without breaking every downstream `grc20` caller.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5908 -R gnolang/gno
go test ./gno.land/pkg/integration/ \
  -run 'TestTestdata/(storage_deposit_price_change|govdao_proposal_change_law)$'
```

```
--- FAIL: TestTestdata/govdao_proposal_change_law (9.55s)
    gas used (41006774) exceeds tx's gas wanted (41000000) during operation: simulation
    FAIL: testdata/govdao_proposal_change_law.txtar:16: unexpected "gnokey" command failure: out of gas error
--- FAIL: TestTestdata/storage_deposit_price_change (10.26s)
        "coins": "9999853978400ugnot",
    FAIL: testdata/storage_deposit_price_change.txtar:72: no match for `"coins": "9999854` found in stdout
FAIL
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5908-decouple-token-id-symbol/1-288cdb044/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/integration/testdata/govdao_proposal_change_law.txtar:16 [↗](../../../../../.worktrees/gno-review-5908/gno.land/pkg/integration/testdata/govdao_proposal_change_law.txtar#L16)
Creating the change-law proposal now uses 41,006,774 gas, over this line's `-gas-wanted 41_000_000`, so the run fails out-of-gas. The enlarged `grc20` in govdao's genesis graph raises preprocessing gas here. Raise the gas budget above the new cost.

## gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar:72 [↗](../../../../../.worktrees/gno-review-5908/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L72)
The final-balance golden `"coins": "9999854` no longer matches: the account ends with 9,999,853,978,400ugnot, 21,600 lower. The larger `grc20` source raises genesis storage and gas in this test's graph, so this golden needs the new value.
