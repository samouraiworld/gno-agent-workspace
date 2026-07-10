# PR [#5924](https://github.com/gnolang/gno/pull/5924): fix(gnogenesis,gnoland): surface silent genesis-replay failures

URL: https://github.com/gnolang/gno/pull/5924
Author: aeddi | Base: master | Files: 3 | +83 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: f79f1ae62 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5924 f79f1ae62`

**TL;DR:** A hardfork genesis whose `InitChainer` aborts before replaying transactions used to boot silently: the node printed "Completed ABCI Handshake" and `gnogenesis fork test` printed PASS, with only an empty appHash to hint that genesis state never loaded. This PR makes both fail loudly.

**Verdict: APPROVE** — pure observability, no happy-path behavior change; one non-blocking test gap (the `fork test` guard has no end-to-end coverage) and one minor display Suggestion.

## Summary
Two independent surfacing fixes for the same class of silent failure. In `gno.land/pkg/gnoland/app.go`, `InitChainer` now logs the `loadAppState` error before returning it, because tendermint's handshake discards `ResponseInitChain.Error` and an operator otherwise saw a clean-looking boot with an empty appHash. In `contribs/gnogenesis/internal/fork/test.go`, `fork test` now compares the number of transactions the result handler actually processed against the number of deliverable transactions in the genesis, and fails when fewer were delivered, rather than printing PASS over a chain with zero or partial genesis state. `countDeliverableTxs` mirrors `deliverGenesisTx`'s skip of `Metadata.Failed` entries so an all-failed genesis (expected 0) still passes trivially.

## Fix
Before, a `loadAppState` error propagated only through `ResponseInitChain.Error`, which tm2 drops, so the failure was invisible at every layer. After, the error is logged at [`app.go:380`](https://github.com/gnolang/gno/blob/f79f1ae62/gno.land/pkg/gnoland/app.go#L380) · [↗](../../../../../.worktrees/gno-review-5924/gno.land/pkg/gnoland/app.go#L380) before the same error value is returned. Separately, `fork test` gains a post-replay guard at [`test.go:282-292`](https://github.com/gnolang/gno/blob/f79f1ae62/contribs/gnogenesis/internal/fork/test.go#L282-L292) · [↗](../../../../../.worktrees/gno-review-5924/contribs/gnogenesis/internal/fork/test.go#L282-L292): `processed < countDeliverableTxs(appState.Txs)` returns a FAIL error pointing the operator at `gnoland start --log-level debug`. The load-bearing constraint is that the deliverable count exactly complements the result-handler firing condition, so the guard cannot false-positive a healthy replay.

## Glossary
- InitChainer — the ABCI `InitChain` handler that replays genesis txs into fresh state.
- appHash — the state-root hash committed after genesis; empty when replay never ran.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- **[core guard has no end-to-end test]** `contribs/gnogenesis/internal/fork/test.go:283` — the new incomplete-replay guard, the PR's headline behavior, is only covered indirectly through its `countDeliverableTxs` helper unit test; no test drives `execTest` to the failure.
  <details><summary>details</summary>

  `TestCountDeliverableTxs` at [`test_test.go:215`](https://github.com/gnolang/gno/blob/f79f1ae62/contribs/gnogenesis/internal/fork/test_test.go#L215) · [↗](../../../../../.worktrees/gno-review-5924/contribs/gnogenesis/internal/fork/test_test.go#L215) pins the counting contract, and `TestExecTest_EmptyGenesis` / `TestExecTest_HardforkGenesis` cover the happy path, but nothing asserts that a broken genesis makes `execTest` return the new "genesis replay delivered N of M" error instead of PASS. A regression that silently dropped the guard would keep every existing test green.

  A genesis whose `GnoGenesisState.InitialHeight` mismatches the `GenesisDoc` makes [`applyInMemoryAppState`](https://github.com/gnolang/gno/blob/f79f1ae62/gno.land/pkg/gnoland/app.go#L494-L499) · [↗](../../../../../.worktrees/gno-review-5924/gno.land/pkg/gnoland/app.go#L494-L499) return before the tx loop; the node still boots to Ready, so the guard is the only thing that catches it. I ran the ready-to-add test below at f79f1ae62: it passes (guard present); it fails against master before the guard commit. Fix: add [`incomplete_replay_guard_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5924-surface-genesis-replay-failures/1-f79f1ae62/tests/incomplete_replay_guard_test.go) · [↗](tests/incomplete_replay_guard_test.go) to the `fork` package.
  </details>

## Suggestions
- **[processed/total line reads as a failure on a valid replay]** `contribs/gnogenesis/internal/fork/test.go:262` — the "Txs processed: %d / %d" line uses `len(appState.Txs)` as the denominator, so a healthy genesis carrying `Metadata.Failed` txs prints e.g. "3 / 5" right before "PASS".
  <details><summary>details</summary>

  The guard correctly measures against `countDeliverableTxs` (=3 here), but the human-readable line at [`test.go:262`](https://github.com/gnolang/gno/blob/f79f1ae62/contribs/gnogenesis/internal/fork/test.go#L262) · [↗](../../../../../.worktrees/gno-review-5924/contribs/gnogenesis/internal/fork/test.go#L262) still divides by `len(appState.Txs)`, which includes the intentionally-skipped `Metadata.Failed` entries. On a genesis with source-failed txs a fully successful replay reads "Txs processed: 3 / 5", the same shape as a partial failure. Since the PR's goal is honest replay status, printing the deliverable count as the denominator (or showing skipped-failed separately) removes the false alarm. Fix: divide by `countDeliverableTxs(appState.Txs)`, or print the skipped-failed count on its own line.
  </details>

## Verified
- The target scenario boots to Ready and the guard catches it: a genesis with one deliverable tx and a mismatched `InitialHeight` makes `InitChainer` return a `ResponseInitChain.Error` before the tx loop; the in-memory node still reaches `Ready()`, `processed` stays 0, and `execTest` returns `FAIL: genesis replay delivered 0 of 1 expected txs`. Confirmed by running [`incomplete_replay_guard_test.go`](tests/incomplete_replay_guard_test.go) in the `fork` package (~4s, PASS at f79f1ae62).
- `countDeliverableTxs` is the exact complement of the result-handler firing condition: [`deliverGenesisTx`](https://github.com/gnolang/gno/blob/f79f1ae62/gno.land/pkg/gnoland/app.go#L792-L800) · [↗](../../../../../.worktrees/gno-review-5924/gno.land/pkg/gnoland/app.go#L792-L800) returns before `cfg.GenesisTxResultHandler` only for `Metadata.Failed` txs; every other tx reaches the handler at [`app.go:813`](https://github.com/gnolang/gno/blob/f79f1ae62/gno.land/pkg/gnoland/app.go#L813) · [↗](../../../../../.worktrees/gno-review-5924/gno.land/pkg/gnoland/app.go#L813). So `processed == expected` on any complete replay and the guard cannot false-positive.
- Tests run green at f79f1ae62: `TestCountDeliverableTxs`, `TestTestCfg_FlagDefaults`, `TestExecTest_MissingGenesis`, `TestExecTest_InvalidGenesis` in `contribs/gnogenesis/internal/fork`.

## Open questions
None.
