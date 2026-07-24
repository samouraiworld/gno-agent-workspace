# PR [#5995](https://github.com/gnolang/gno/pull/5995): fix(gnogenesis): stop fork test from reporting PASS on a rejected genesis

URL: https://github.com/gnolang/gno/pull/5995
Author: davd-gzl | Base: master | Files: 11 | +425 -43
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep, inline) | Commit: 7435cfe94 (latest) | Round: 2
Local worktree: `.worktrees/gno-review-5995` on `fix/gnogenesis-fork-test-verdict`

**TL;DR:** The gnogenesis fork test replays a genesis through an in-memory node and reports whether it boots. On a hardfork genesis with zero deliverable transactions it printed PASS even when the node had actually rejected the genesis, because its only completeness check compared "transactions processed" against "transactions expected" and both were zero. The fix makes the node abort on an InitChain rejection so the error propagates up to the test, and narrows the guard to the case it can still catch. A follow-up commit in this round reverts a mistaken change to the genesis-reset gate.

**Verdict: APPROVE** — the bug is reproduced (a rejected genesis returned nil from `execTest`), the full rejection-to-test error path is verified end to end, and a bogus intermediate commit was reverted with its two salvageable parts kept (0 Critical, 0 Warning).

## Summary

`execTest` in [`contribs/gnogenesis/internal/fork/test.go`](https://github.com/gnolang/gno/blob/7435cfe94/contribs/gnogenesis/internal/fork/test.go) guarded an incomplete InitChainer run by comparing `processed` against `countDeliverableTxs(appState.Txs)`. On a genesis with no deliverable txs that is `0 < 0`, which never fires. The failures the guard named, an InitialHeight mismatch and a rejected SignerInfo, are raised in `applyInMemoryAppState` before the tx loop and are independent of the tx count, so a hardfork genesis with mismatched InitialHeight and zero txs printed "PASS: genesis replay completed successfully" and exited 0.

The fix routes those failures through node startup: `InitChainer` reports them via `ResponseInitChain.Error`, the handshake now honours that field ([`replay.go:351`](https://github.com/gnolang/gno/blob/7435cfe94/tm2/pkg/bft/consensus/replay.go#L351)), so `NewInMemoryNode` returns the error and `execTest` never reaches the guard. The guard keeps its narrower job of catching a tx loop that exits early.

## Round-2 delta

This round added `revert(tm2/node): keep the genesis reset keyed on the applied-genesis marker` (7435cfe94). An intermediate commit had swapped the genesis-reset gate from `genesisApplied` (which reads `LoadABCIResponses(stateDB, 0)`) to a committed-block check, on the stated grounds that the old gate read the block store and could desync from the state record. That premise is false: `genesisApplied` reads the state DB, the same database the reset writes, so losing the block store leaves those responses intact and the described failure cannot occur. The swap also widened the reset window past a successful InitChain. The revert restores the state-DB gate and keeps the two parts of that commit that stand on their own: the validate-before-write ordering in `resetToGenesisDoc`, and the `%w` wrap errorlint requires.

## Glossary

- fork test: `gnogenesis fork test`, which replays a genesis through an in-memory node to check it boots before a hardfork.
- InitChainer: the ABCI callback that applies genesis state at height 0; it signals rejection through `ResponseInitChain.Error`.
- deliverable tx: a genesis tx that reaches the InitChain delivery loop; `metadata.Failed` txs are skipped and excluded from the count.

## Fix

The core change is [`replay.go:349-352`](https://github.com/gnolang/gno/blob/7435cfe94/tm2/pkg/bft/consensus/replay.go#L349-L352): the handshake now aborts when `ResponseInitChain.Error` is set, wrapped with `%w` since `abci.Error` satisfies `error` and `IsErr()` guards nil. The guard at [`test.go:274-282`](https://github.com/gnolang/gno/blob/7435cfe94/contribs/gnogenesis/internal/fork/test.go#L274-L282) keeps the tx-count comparison but its comment now states it cannot stand alone and defers the pre-loop failures to the abort. The reset-gate revert at [`node.go`](https://github.com/gnolang/gno/blob/7435cfe94/tm2/pkg/bft/node/node.go) restores `genesisApplied` and its doc.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[retained guard is unreachable today]** `contribs/gnogenesis/internal/fork/test.go:274` — after the abort change, `processed < expected` can no longer fire, because `metadata.Failed` txs return before the handler and every other tx calls it, so `processed == expected` always.
  <details><summary>details</summary>

  The comment already says it is kept "as a tripwire for a future path that skips the handler", which is honest and defensible for a cheap assertion. Left as-is deliberately; noting it so a future reader does not mistake it for live coverage.
  </details>

## Missing Tests

None. [`TestExecTest_InitialHeightMismatch`](https://github.com/gnolang/gno/blob/7435cfe94/contribs/gnogenesis/internal/fork/test_test.go#L245) drives the exact degenerate case (zero-tx hardfork genesis with a mismatched InitialHeight) and asserts `execTest` returns an error containing "InitialHeight mismatch". The node-side reset behaviour is pinned by `TestE2ERejectThenCorrectedGenesisBoots` and `TestGenesisResetKeepsGoodDocOnInvalidFile`.

## Suggestions

None.

## Verified

- The bug reproduces. Disabling the InitChain abort (`if false && res.IsErr()`) and running `TestExecTest_InitialHeightMismatch -count=1` makes `execTest` return nil on a genesis the node rejected: "An error is expected but got nil". `minimalAppState()` carries `Txs: []`, so `expected == 0` and the old guard evaluated `0 < 0`. Restoring the abort turns it green.
- The full rejection-to-test path holds, traced rather than assumed: `applyInMemoryAppState` returns the mismatch error at [`app.go:512`](https://github.com/gnolang/gno/blob/7435cfe94/gno.land/pkg/gnoland/app.go#L512), before the tx loop at line 555; `loadAppState` wraps it into `ResponseInitChain.Error` at [`app.go:405`](https://github.com/gnolang/gno/blob/7435cfe94/gno.land/pkg/gnoland/app.go#L405); the handshake aborts on it at `replay.go:351`; `NewInMemoryNode` returns it; `execTest` surfaces it at [`test.go:224`](https://github.com/gnolang/gno/blob/7435cfe94/contribs/gnogenesis/internal/fork/test.go#L224). `validateSignerInfo` sits at `app.go:529`, also pre-loop, so both named failures are covered.
- The revert is justified, not cosmetic. `genesisApplied` reads `LoadABCIResponses(stateDB, 0)` from the state DB, so the block-store-desync failure the reverted commit described is unreachable. The two salvageable parts were kept and each revert-tested: reordering `saveGenesisDoc` after `MakeGenesisState` makes `TestGenesisResetKeepsGoodDocOnInvalidFile` fail when undone, and the `%w` wrap is required by errorlint.
- The retained guard is unreachable in the current code: `metadata.Failed` txs return early at `deliverGenesisTx` without the handler, every other tx calls it, and `countDeliverableTxs` excludes exactly the former, so `processed == expected` always.
- Green at 7435cfe94: `go test ./tm2/pkg/bft/node/ ./tm2/pkg/bft/consensus/ ./gno.land/pkg/gnoland/`, and `go test ./...` in the `contribs/gnogenesis` module (a separate module; root-level `./contribs/...` fails with "directory prefix does not contain main module"). `gofmt -l` clean.

## Open questions

- A stray `config/addrbook.json` was written into the worktree root during full-suite runs. Not reproducible from any package this PR touches, so pre-existing test hygiene rather than a defect here. Not posted.
