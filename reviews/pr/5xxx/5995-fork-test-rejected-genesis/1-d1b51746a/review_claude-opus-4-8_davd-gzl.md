# PR [#5995](https://github.com/gnolang/gno/pull/5995): fix(gnogenesis): stop fork test from reporting PASS on a rejected genesis

URL: https://github.com/gnolang/gno/pull/5995
Author: davd-gzl | Base: master | Files: 6 | +125 -36
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: d1b51746a (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5995 d1b51746a`
Overview: [visual overview](../overview.html)

**TL;DR:** A tool that checks a genesis file before a chain upgrade said "PASS" on genesis files the chain would actually refuse to start on. The real cause was one layer down: the node had a way to say "I refuse this genesis", and nobody was listening, so it started anyway. This PR makes the node listen.

**Verdict: REQUEST CHANGES** — the abort itself is correctly placed and I could not break it, but it turns an exported constructor into one that cannot boot, and the retry it now sends operators to ignores the field its own error names (2 Warnings, 1 Missing test, 6 Nits, 1 Suggestion).

## Summary

An app tells tm2 "do not start this chain" by setting `ResponseInitChain.Error`. That field is the only channel it has, because [`InitChainSync`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/appconn/app_conn.go#L65) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/appconn/app_conn.go#L65) is a pass-through down to the local client, which returns `(res, nil)` unconditionally. The ABCI handshake never read it, so a node whose genesis was rejected saved the genesis ABCI responses and committed its first block on empty state. `gnogenesis fork test`, which boots an in-memory node to smoke-test a hardfork genesis, inherited the blind spot: its own backstop compared processed txs against deliverable txs, so a genesis with zero deliverable txs evaluated `0 < 0` and the tool printed PASS.

The fix is three lines in [`replay.go:350-352`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L350-L352) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L350-L352): abort the handshake when the field is set. Everything else in the diff is comments, one reworded test assertion message, and two new tests.

That three-line change is a tm2-wide contract change wearing a gnogenesis bugfix's title, and its blast radius is where both Warnings live. Every `loadAppState` rejection is now fatal, including one nobody meant to make fatal, and a newly-failing boot sends operators down a retry path that silently ignores half the genesis file.

```
genesis.json ──► Handshaker.ReplayBlocks ──► InitChainSync ──► baseapp.InitChain ──► gnoland InitChainer
                        │                                                                    │
                        │                                              loadAppState error ───┤
                        │                                                                    ▼
                        │◄──────────────── ResponseInitChain.Error ◄────────────── (was: dropped here)
                        ▼
                  abort the boot          (before: save ABCI responses, commit first block)
```

## Glossary

- ABCI: the app-to-consensus interface; `InitChain` is its genesis-time call.
- app hash: the per-block commitment to application state; before this PR a node that skipped genesis replay still committed blocks, with an app hash computed over empty state.
- genesis tx: a transaction embedded in `GenesisDoc.AppState.Txs`, executed by `InitChainer` before any block.

## Fix

Before, [`ReplayBlocks`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L339-L341) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L339-L341) checked only the Go-level error from `InitChainSync`, which is never non-nil. After, it also checks the response's `Error` field and returns before [`SaveABCIResponses`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L357) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L357). The load-bearing constraint is that no app state survives the abort: `InitChainer` writes into a cache wrap of the commit multistore ([`baseapp.go:263-269`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/sdk/baseapp.go#L263-L269) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/sdk/baseapp.go#L263-L269)) and `Commit` never runs.

## Warnings (should fix)

- **[shipped constructor stops booting]** `gno.land/pkg/gnoland/node_inmemory.go:43` — [`NewDefaultGenesisConfig`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/node_inmemory.go#L43) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/node_inmemory.go#L43) builds `AppState` as a pointer, which `loadAppState` does not match, so the genesis it hands out now fails the boot.
  <details><summary>details</summary>

  [`loadAppState`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L498-L505) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L498-L505) switches on the value type `GnoGenesisState` and on `*GenesisStateRef`; `*GnoGenesisState` falls to `default` and returns [`invalid AppState of type %T`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L504) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L504). Until this PR that error was discarded and the node booted, so the helper's balances, txs and `ChainDomain` were silently dropped rather than applied. The only in-tree caller overwrites `AppState` with a value at [`node.go:717`](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnodev/pkg/dev/node.go#L717) · [↗](../../../../../.worktrees/gno-review-5995/contribs/gnodev/pkg/dev/node.go#L717), which is why CI is green and why nothing in the tree notices. Booting the helper's output with a genesis validator supplied by the caller, as gnodev does, returns `<nil>` at the merge base and `InitChain rejected the genesis: invalid AppState of type *gnoland.GnoGenesisState` at this head. Fix: drop the `&` so the helper produces a genesis that loads, in this PR, since this PR is what makes it fatal.
  </details>

- **[the fix the error message asks for does not work]** `tm2/pkg/bft/node/node.go:1085` — the aborted boot has already persisted the genesis doc, and the retry path re-reads only `AppState` from the file, so correcting the doc-level `InitialHeight` the abort names changes nothing and the node re-aborts forever.
  <details><summary>details</summary>

  [`LoadStateFromDBOrGenesisDocProvider`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/node/node.go#L1058-L1085) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/node/node.go#L1058-L1085) runs at [`node.go:401`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/node/node.go#L401) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/node/node.go#L401), well before the handshake, and persists the doc with `AppState` stripped at [`node.go:1115`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/node/node.go#L1115) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/node/node.go#L1115). On the next boot the stripped-`AppState` branch re-invokes the provider but keeps only [`genDoc.AppState = freshDoc.AppState`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/node/node.go#L1085) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/node/node.go#L1085), and its guard compares only `ChainID` and `AppHash`, so a changed doc-level field passes unnoticed and the database copy wins. Driving that function twice against one database returns `InitialHeight=999` on both calls even when the second provider supplies 100. The abort message names `GenesisDoc.InitialHeight`, which is exactly the field an operator cannot fix, so under any supervisor this is a crash loop escapable only by wiping the data directory. The mechanism predates the PR; what is new is that a rejected genesis now aborts instead of booting, which is what puts an operator on the retry path at all.
  </details>

## Missing Tests

- **[abort untested on the genesis shape it targets]** `tm2/pkg/bft/consensus/replay_test.go:1242` — [`TestHandshakeInitChainError`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay_test.go#L1211) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay_test.go#L1211) covers only `InitialHeight == 1`, so it never exercises the hardfork genesis the PR is motivated by.
  <details><summary>details</summary>

  On `InitialHeight > 1`, [`ReplayBlocks`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L311-L320) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L311-L320) persists the state alignment before `InitChainSync` runs, so the abort leaves `InitialHeight` and `LastBlockHeight` written while no ABCI responses are saved. That write predates this PR and a retry with the same genesis converges, so it is not a defect, but it is the difference between what the test's comment claims and what it checks. A hardfork variant of the same test pins both halves: responses absent, alignment left in place.
  </details>

## Nits

- **[comment describes a check that cannot fire]** `contribs/gnogenesis/internal/fork/test.go:265` — [the comment](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnogenesis/internal/fork/test.go#L265-L269) · [↗](../../../../../.worktrees/gno-review-5995/contribs/gnogenesis/internal/fork/test.go#L265-L269) presents the guard as catching a partial `InitChainer` run, but nothing can still reach `processed < expected`.
  <details><summary>details</summary>

  The decoded `AppState` the fork test hands the node is a value `GnoGenesisState`, which is what the type assertion at [`test.go:108`](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnogenesis/internal/fork/test.go#L108) · [↗](../../../../../.worktrees/gno-review-5995/contribs/gnogenesis/internal/fork/test.go#L108) relies on, so it always takes the in-memory path. That loop at [`app.go:557-560`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L557-L560) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L557-L560) has no break, and `deliverGenesisTx` calls the result handler at [`app.go:866`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L866) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L866) on every path except the `metadata.Failed` skip at [`app.go:833`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L833) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L833), which [`countDeliverableTxs`](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnogenesis/internal/fork/test.go#L342-L350) · [↗](../../../../../.worktrees/gno-review-5995/contribs/gnogenesis/internal/fork/test.go#L342-L350) subtracts from `expected` with the same predicate. The streaming path's mid-loop exits at [`app.go:639-646`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L639-L646) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L639-L646) all return errors, which now abort the boot. Keeping it as a defensive assertion against a future handler-skipping path is reasonable; the comment should say it is unreachable today rather than describe a live failure mode, and the `--verbose` remediation text cannot help with a failure that cannot happen.
  </details>

- **[comment promises a warning the code never logs]** `gno.land/pkg/gnoland/app.go:396` — [the comment](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L396-L397) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L396-L397) says a nil `AppState` is a testing-setup case that gets a warning. It gets `invalid AppState of type <nil>`, and as of this PR that kills the boot.

- **[comment names an example nothing enables]** `tm2/pkg/bft/consensus/replay.go:347` — [the new comment](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L343-L349) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L343-L349) cites `StrictReplay` as the motivating case, but no production code sets it and there is no flag for it, only [`TestInitChainer_StrictReplay`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app_test.go#L2434) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app_test.go#L2434), which calls `InitChainer` directly rather than through the handshake. The reachable triggers are the `loadAppState` preflights. "Nothing used to read the field" is also off by one reader: [`baseapp.InitChain`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/sdk/baseapp.go#L377-L381) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/sdk/baseapp.go#L377-L381) reads it to skip post-init bookkeeping. The handshake was the one ignoring it.

- **[comment credits the log with the only diagnosis]** `gno.land/pkg/gnoland/app.go:402` — [the rewritten comment](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L400-L403) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L400-L403) says the log line is what names the actual cause. After this PR the returned error carries the same string, which is what the new fork test asserts on at [`test_test.go:286`](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnogenesis/internal/fork/test_test.go#L286) · [↗](../../../../../.worktrees/gno-review-5995/contribs/gnogenesis/internal/fork/test_test.go#L286).

- **[test comment claims more than it asserts]** `tm2/pkg/bft/consensus/replay_test.go:1239` — ["Nothing from the rejected genesis may be persisted"](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay_test.go#L1239-L1241) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay_test.go#L1239-L1241) sits above a single `LoadABCIResponses` check, and is false for a hardfork genesis.

- **[hand-rolled helper]** `tm2/pkg/bft/consensus/replay.go:350` — [`res.Error != nil`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L350-L351) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L350-L351) restates [`ResponseBase.IsErr`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/abci/types/types.go#L120-L122) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/abci/types/types.go#L120-L122), and `%s` with `.Error()` can be `%v` with the value.

- `tm2/pkg/bft/consensus/replay_test.go:1237` — [`assert.Contains(t, err.Error(), ...)`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay_test.go#L1236-L1237) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay_test.go#L1236-L1237) where the sibling test in this same PR uses `require.ErrorContains`. Not posted, no linter enforces it and it changes no meaning.

## Suggestions

- **[boot failures split across two mechanisms]** `gno.land/pkg/gnoland/app.go:376` — now that the error field is honoured, the remaining panics on the genesis path give operators a stack trace where a message would do.
  <details><summary>details</summary>

  The [pubkey-type allow-list](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L376-L383) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L376-L383) and [`applyBalance`'s vesting check](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L694) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L694) both panic, alongside the valoper assertion this PR deliberately keeps. A panic escapes `NewInMemoryNode` synchronously, so library embedders like the fork test get an uncaught panic instead of a printable error. Not required here; worth a follow-up so the boot-failure surface is uniform.
  </details>

## Verified

- The original bug reproduces at the merge base: a hardfork genesis with a mismatched `InitialHeight` and zero txs runs `execTest` to `err = <nil>` after printing `PASS: genesis replay completed successfully.`, so the tool exits 0 on a genesis the node rejects.
- The same genesis at the merge base with three txs added does fail, at `delivered 0 of 3 expected txs`, and the pre-PR message named `validateSignerInfo or InitialHeight mismatch` as the likely cause. The zero-tx case is the only one the old guard missed, which is what makes the guard's remaining job empty once the abort lands upstream.
- Removing the [`if res.Error != nil` block](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L350-L352) · [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L350-L352) turns [`TestExecTest_InitialHeightMismatch`](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnogenesis/internal/fork/test_test.go#L241) · [↗](../../../../../.worktrees/gno-review-5995/contribs/gnogenesis/internal/fork/test_test.go#L241) red with "An error is expected but got nil", across the module boundary via the `replace` directive in [`go.mod`](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnogenesis/go.mod#L103) · [↗](../../../../../.worktrees/gno-review-5995/contribs/gnogenesis/go.mod#L103). The same removal turns `TestHandshakeInitChainError` red at its first assertion; relaxing that `require.Error` to `assert.Error` shows the persistence check fails too.
- Booting [`NewDefaultGenesisConfig`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/node_inmemory.go#L36) · [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/node_inmemory.go#L36) through `NewInMemoryNode`, with only a genesis validator added, returns `<nil>` at the merge base and refuses at this head. The helper emits no validators of its own, so without that addition the merge base fails earlier on `validator set is nil in genesis`.
- Dumping the state DB after an aborted boot leaves five tendermint metadata keys and nothing from the genesis: no ABCI responses at height 0, no app or VM state.
- Reusing that same DB for a retry with a corrected genesis boots cleanly to the hardfork height, but only when the correction is to the app-level `InitialHeight`. Driving `LoadStateFromDBOrGenesisDocProvider` twice against one database returns `InitialHeight=999` on both calls when the second provider supplies 100, so a doc-level correction is discarded.
- Green at this head: `tm2/pkg/bft/consensus -run TestHandshake`, `tm2/pkg/bft/node`, `tm2/pkg/bft/state/...`, `tm2/pkg/sdk`, `gno.land/pkg/gnoland`, `gno.land/pkg/integration` including txtar, `gno.land/cmd/gnoland`, and all packages of `contribs/gnodev` and `contribs/gnogenesis`.

## Open questions

- [`tm2/adr/pr5949_remove_secp256k1_validators.md:71-72`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/adr/pr5949_remove_secp256k1_validators.md?plain=1#L71-L72) · [↗](../../../../../.worktrees/gno-review-5995/tm2/adr/pr5949_remove_secp256k1_validators.md#L71-L72) justifies its panic with "`ResponseInitChain.Error` is silently discarded by tm2", which this PR makes false. Not posted: findings cover code only and this skill does not critique ADRs.
- The PR body says a rejected genesis "committed block 1 anyway". For the hardfork genesis it uses as the motivating example the first committed block is `InitialHeight`, not 1. Not posted, it is description wording rather than code.
- The tm2 change is the independently-motivated fix and the gnogenesis verdict bug is one symptom of it. Framing it the other way round is what left the `NewDefaultGenesisConfig` fallout unexamined. Worth deciding whether the description should lead with the tm2 contract change.
