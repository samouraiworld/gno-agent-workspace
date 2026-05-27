# PR #5489: feat(gnoland): genesis tx metadata + chain replay at initial height

URL: https://github.com/gnolang/gno/pull/5489
Author: moul | Base: master | Files: 22 | +1097 -25
Reviewed by: davd-gzl | Model: claude-opus-4.7 | Commit: `d9360e2` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5489 d9360e2`

**Verdict: REQUEST CHANGES** — protocol mechanism is sound and well-tested, but the chain-ID override is fail-open (piux2): a non-empty `metadata.ChainID` not in `PastChainIDs` silently falls back to the current chain ID instead of rejecting the tx, and `ValidateGenState` does not check `PastChainIDs` (empty strings, overlong IDs) or `GnoGenesisState.InitialHeight`. `GnoGenesisState.InitialHeight` is informational-only and silently desyncs from the load-bearing `GenesisDoc.InitialHeight`.

## Summary

Adds protocol-level support for hardfork genesis replay: each genesis tx may carry a `GnoTxMetadata{BlockHeight, Timestamp, ChainID}`; when `BlockHeight > 0`, the ante handler runs in normal mode (real sig verification, account numbers, sequences) instead of genesis mode, and the per-tx `ChainID` overrides the context chain ID if it is in a new `GnoGenesisState.PastChainIDs` allowlist. A new `GenesisDoc.InitialHeight` (tm2) lets a new chain resume block production at the height the old chain halted — this required threading "first block at any height when store is empty" through eight tm2 sites (consensus state.go × 3, blockchain/reactor.go, bft/types/block.go, state/state.go, state/validation.go, state/execution.go, store/store.go, sdk/baseapp.go). The PR is the protocol mechanism only; the hardfork tooling and deployment scripts live in #5411.

```
                    Genesis txs                              Block production
                    ┌──────────────────┐                     ┌──────────────────┐
old chain halt ───► │ tx1 BH=42 CID=A  │  InitChain ───────► │ block 100 (gen.) │ ──► block 101 ...
                    │ tx2 BH=43 CID=A  │  → ante (full sig)  │  empty commit    │     normal commit
                    │ tx3 BH=99 CID=B  │  → CID override     │  GenesisTime     │     MedianTime
                    └──────────────────┘    if CID∈[A,B]     └──────────────────┘
                                                       ▲
                              GenesisDoc.InitialHeight=100 → Handshaker sets state.LastBlockHeight=99
                              GnoGenesisState.InitialHeight=100 → log line only (informational)
```

## Glossary

- `GnoTxMetadata` — per-tx envelope: `Timestamp`, `BlockHeight`, `ChainID`.
- `PastChainIDs` — `GnoGenesisState.[]string` allowlist of chain IDs accepted for the override.
- `ctxFn` — `sdk.ContextFn` applied per-tx in `InitChainer.loadAppState` before `baseApp.Deliver`.
- `Handshaker` — tm2 component that runs `InitChain` and reconciles state/store heights at startup.
- `isGenesis` — ante-handler flag derived from `ctx.BlockHeight() == 0`; gates `accNum`/`seq` in the sign payload.
- `state.LastBlockID.IsZero()` — fresh-genesis detector used by the new code instead of `Height == 1`.

## Fix

Before: `InitChainer` ran every tx in genesis mode (`accNum=0`, `seq=0`, optional sig skip, auto account creation, infinite gas) at header height 0; the chain always started at height 1; tm2 used `Height == 1` to detect genesis blocks. After: txs with `metadata.BlockHeight > 0` go through the full ante handler (real sigs, real account numbers/sequences); their context header is rewritten per-tx ([`gno.land/pkg/gnoland/app.go:412-435`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L412-L435) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L412-L435)); chain ID is overridden if `metadata.ChainID ∈ PastChainIDs`; tm2 detects genesis via `LastBlockID.IsZero()` (or `Height == 1`) so a chain can start at any `InitialHeight > 1` with an empty block store. The load-bearing constraint everywhere in tm2: `if store.Height() == 0 { treat as genesis }` — replaces the old `Height == 1`/`Height > 1` heuristics ([`tm2/pkg/bft/types/block.go:74`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/types/block.go#L74) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/types/block.go#L74), [`tm2/pkg/bft/state/validation.go:99`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/state/validation.go#L99) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/state/validation.go#L99), [`tm2/pkg/bft/state/execution.go:271-294`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/state/execution.go#L271-L294) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/state/execution.go#L271-L294), [`tm2/pkg/bft/store/store.go:168-175`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/store/store.go#L168-L175) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/store/store.go#L168-L175)).

## Critical (must fix)

- **[chain ID override fails open]** [@piux2](https://github.com/gnolang/gno/pull/5489#issuecomment-4239043826) [`gno.land/pkg/gnoland/app.go:429`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L429) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L429) — A non-empty `metadata.ChainID` not in `PastChainIDs` silently falls back to verification under the current chain ID instead of rejecting the tx.
  <details><summary>details</summary>

  The override condition is `metadata.BlockHeight > 0 && metadata.ChainID != "" && isPastChainID(state.PastChainIDs, metadata.ChainID)`. When the first two hold but the third does not, `ctx.WithChainID` is never called and the tx is verified under `req.ChainID` (the current chain's ID, set in [`tm2/pkg/sdk/baseapp.go:326`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/sdk/baseapp.go#L326) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/sdk/baseapp.go#L326)). The ADR claim "Unrecognised chain IDs are never silently overridden — they fail as expected" ([`gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md:67`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L67) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L67)) is true in the narrow sense that the *override* doesn't happen, but the *tx* is still accepted if it was signed against the current chain ID. The existing test [`gno.land/pkg/gnoland/app_test.go:1571-1638`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app_test.go#L1571-L1638) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app_test.go#L1571-L1638) encodes this permissive behavior: `metadata.ChainID = "unknown-chain"`, `PastChainIDs` empty, tx signed with `chainID = "new-chain"` (the current chain) — and `InitChain` succeeds. For a hardfork replay invariant, a tx declaring `metadata.ChainID = "unknown"` is a malformed/lying tx; accepting it under the current chain ID lets a malformed genesis pass review unnoticed. Fix: when `metadata.BlockHeight > 0 && metadata.ChainID != "" && !isPastChainID(...)`, return a hard error from `loadAppState` (entire genesis rejected) instead of silently falling through. Invert the test to assert this failure mode, per piux2's suggestion.
  </details>

- **[no validation of new genesis fields]** [`gno.land/pkg/gnoland/genesis.go:264`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/genesis.go#L264) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/genesis.go#L264) — `ValidateGenState` accepts `PastChainIDs` containing `""`, duplicates, or strings > `MaxChainIDLen`; accepts negative `InitialHeight`.
  <details><summary>details</summary>

  Compare the strictness on `GenesisDoc.ChainID` in tm2 ([`tm2/pkg/bft/types/genesis.go:79-86`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/types/genesis.go#L79-L86) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/types/genesis.go#L79-L86) — empty rejected, `len > MaxChainIDLen` rejected) and `InitialHeight` ([`tm2/pkg/bft/types/genesis.go:94`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/types/genesis.go#L94) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/types/genesis.go#L94) — negative rejected). The same per-string checks must apply to `PastChainIDs` entries, otherwise a typo (`""`, accidental whitespace, or a chain ID copy-pasted with quotes) silently disables the override path for that entry and the tx falls into the fail-open case above. Empty-string entries are especially dangerous because `metadata.ChainID != ""` in the override guard would still need to hold, but a tx with `metadata.ChainID == ""` would skip the override anyway — so an empty entry is dead weight, but a typo'd entry is silently broken. Fix: extend `ValidateGenState` to check `state.PastChainIDs` (each non-empty, each ≤ `MaxChainIDLen`, deduplicated) and `state.InitialHeight >= 0`. Wire it where `GnoGenesisState` is parsed (genesis JSON ingestion) so the chain refuses to boot with a malformed allowlist.
  </details>

## Warnings (should fix)

- **[InitialHeight desync footgun]** [`gno.land/pkg/gnoland/types.go:131`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/types.go#L131) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/types.go#L131) — `GnoGenesisState.InitialHeight` is informational only; the operator must also set `GenesisDoc.InitialHeight` or the chain silently starts at height 1 with stale logs claiming otherwise.
  <details><summary>details</summary>

  The ADR explicitly notes the field is "Informational field for tooling … not read by the app during InitChain" ([`gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md:39-42`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L39-L42) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L39-L42)). The only consumer is the log line at [`gno.land/pkg/gnoland/app.go:456-460`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L456-L460) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L456-L460) saying "chain will start from initial height" — which is a lie if `GenesisDoc.InitialHeight` was left at 0. Two state-of-the-world fields meaning the same thing, only one of which is load-bearing, is a classic split-brain. Fix options: (a) drop `GnoGenesisState.InitialHeight` entirely and read `req.InitialHeight` from `abci.RequestInitChain` if/when ABCI gains it; (b) make `loadAppState` assert `state.InitialHeight == genDoc.InitialHeight` (requires plumbing); (c) at minimum, change the log line to read the actual `state.LastBlockHeight` and document the wiring requirement loudly in the field comment. Option (a) is cleanest — one source of truth.
  </details>

- **[replay accounts must align with old chain]** [`gno.land/pkg/gnoland/app.go:142-156`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L142-L156) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L142-L156) — Auto-account-creation is skipped for `metadata.BlockHeight > 0` txs; replay txs require accounts pre-funded with matching `AccountNumber` and `Sequence`, otherwise sig verification fails with `UnauthorizedError`.
  <details><summary>details</summary>

  When the ante handler computes `isGenesis := ctx.BlockHeight() == 0` ([`tm2/pkg/sdk/auth/ante.go:115`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/sdk/auth/ante.go#L115) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/sdk/auth/ante.go#L115)), a `BlockHeight > 0` tx skips the `isGenesis` branch in `GetSignBytes` and uses the live `accNum`/`seq` from the account ([`tm2/pkg/sdk/auth/ante.go:422-441`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/sdk/auth/ante.go#L422-L441) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/sdk/auth/ante.go#L422-L441)). The author hit this himself in [`commit b4e87a7`](https://github.com/gnolang/gno/pull/5489/commits/b4e87a78748cede3e385346ceb34b8efd8e74094) ("Two keys with BlockHeight>0 caused UnauthorizedError because the second key got accNum=1 but was signed with accNum=0"). The ADR's "Open items" section flags this ([`gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md:95-100`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L95-L100) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L95-L100)) and the multi-chain test at [`gno.land/pkg/gnoland/app_test.go:1640-1734`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app_test.go#L1640-L1734) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app_test.go#L1640-L1734) only works because both txs use one key. There is no protocol enforcement: a malformed genesis with mismatched account numbers will silently produce per-tx failures whose `res.IsErr()` logs land in stderr ([`gno.land/pkg/gnoland/app.go:438-445`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L438-L445) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L438-L445)) but the InitChainer keeps going. Fix: when `GenesisTxResultHandler == PanicOnFailingTxResultHandler` is set (production path, see [`gno.land/pkg/gnoland/app.go:243`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L243) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L243)) this surfaces as a panic — that's OK. But the dev/test path uses `NoopGenesisTxResultHandler`, which would hide a corrupt replay. At minimum, document the account-ordering contract in the ADR's "Decision" section (not "Open items"), and emit an explicit `Logger().Error("hardfork replay tx failed sig verification — account ordering may be wrong")` when a `BlockHeight > 0` tx errors. The genesis-assembly tool (gnolang/tx-archive#70) will need to honour the ordering invariant.
  </details>

- **[fail-open block height override]** [`gno.land/pkg/gnoland/app.go:419-421`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L419-L421) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L419-L421) — `metadata.BlockHeight` is applied without checking that it is ≤ `state.InitialHeight` or `GenesisDoc.InitialHeight - 1`; a malicious or buggy genesis can replay a tx claiming a future block height.
  <details><summary>details</summary>

  The header rewrite `header.Height = metadata.BlockHeight` happens unconditionally for any `BlockHeight > 0`. There is no bound check. A realm querying `std.ChainHeight()` during replay will see whatever the metadata says — including values larger than `InitialHeight`, which would let a replay tx see a height "from the future" relative to the chain it's now part of. For honest tx-archive output this never happens, but the genesis is a trusted input that should still be sanity-checked. Fix: in `loadAppState`, when `state.InitialHeight > 0`, assert `metadata.BlockHeight < state.InitialHeight` (or against `GenesisDoc.InitialHeight` once that is properly threaded — see the InitialHeight desync warning above). Combine with the missing `ValidateGenState` checks.
  </details>

- **[silent over-permissive override for empty metadata.ChainID]** [`gno.land/pkg/gnoland/app.go:429`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L429) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L429) — If `metadata.BlockHeight > 0` but `metadata.ChainID == ""`, the tx runs through the full ante handler with the *current* chain ID. For a true hardfork replay tx, `ChainID == ""` is a tx-archive bug; the code accepts it.
  <details><summary>details</summary>

  This is the same fail-open shape as the piux2 critical, narrower scope. The replay invariant is "every historical tx declares its origin chain"; an empty `ChainID` violates it. The ADR's per-tx ChainID is the whole point of the design ([`gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md:72-81`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L72-L81) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L72-L81)). Fix: same hard error as the critical — if `BlockHeight > 0`, require `ChainID != ""` AND `ChainID ∈ PastChainIDs`. Two separate guards collapsing into one rule: "historical tx ⇒ allowlisted origin chain ID, no exceptions."
  </details>

- **[tm2 fresh-genesis detector relies on field-zero overload]** [`tm2/pkg/bft/state/validation.go:99`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/state/validation.go#L99) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/state/validation.go#L99) — `isGenesisBlock := state.LastBlockID.IsZero()` is the new genesis detector; if a future code path ever zeroes `LastBlockID` mid-chain (block-store corruption recovery, snapshot import bug), validation silently treats the next block as genesis and skips commit verification.
  <details><summary>details</summary>

  Same pattern appears in [`tm2/pkg/bft/types/block.go:74`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/types/block.go#L74) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/types/block.go#L74) (`b.Height == 1 || b.Header.LastBlockID.IsZero()`) and [`tm2/pkg/bft/state/state.go:134`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/state/state.go#L134) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/state/state.go#L134) (`commit.BlockID.IsZero()`). The old `Height == 1` rule was a single, structural invariant — height 1 only exists once per chain lifetime. The new `IsZero()` rule is a *value* invariant — true at genesis, but a zero-init bug elsewhere could make it true mid-chain. There is no positive marker on the state struct ("this is the first block after a genesis-from-height-N") that's checked independently. Fix: add a derived check that pairs `IsZero()` with `state.LastBlockHeight == 0 || ... InitialHeight context`. Less risky alternative: leave the heuristic but add a test asserting that `IsZero()` only holds while `blockStore.Height() == 0` — fuzz it. As written the code is correct, but the safety net for "future maintainers must not zero LastBlockID" is just a code comment.
  </details>

## Nits

- [`gno.land/pkg/gnoland/types.go:140`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/types.go#L140) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/types.go#L140) — `GnoTxMetadata.Timestamp` lacks `omitempty` while `BlockHeight` and `ChainID` have it; inconsistent and bloats every genesis tx with `"timestamp":0`.
- [`gno.land/pkg/gnoland/app.go:476`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L476) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L476) — `isPastChainID` is a 1-line wrapper over `slices.Contains`; inline it at the call site or drop the helper. Per [@tbruyelle's review suggestion](https://github.com/gnolang/gno/pull/5489#discussion_r2380000000) the author already simplified to `slices.Contains` — the helper is dead weight now.
- [`gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md:80`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L80) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/adr/pr5489_genesis_tx_metadata_initial_height.md#L80) — "State-level override" alternative description conflates `OriginalChainID` (rejected) with the chosen `PastChainIDs + per-tx ChainID`. Reads as if both alternatives have the same shape; sharpen the contrast.
- [`tm2/pkg/bft/consensus/replay.go:336`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/consensus/replay.go#L336) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/consensus/replay.go#L336) — `if h.genDoc.InitialHeight > 1` ignores `InitialHeight == 1`; treat as equivalent to default (no-op). Comment that this is intentional, otherwise a future reader wonders about the off-by-one.
- [`gno.land/pkg/gnoland/app.go:357-364`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L357-L364) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L357-L364) — "Chain upgrade genesis replay" log fires only when `len(state.PastChainIDs) > 0`. A replay with `InitialHeight > 0` but empty `PastChainIDs` (single-chain rename, no historical txs) doesn't log. Trigger on either field non-zero.

## Missing Tests

- **[fail-open chain ID]** [`gno.land/pkg/gnoland/app_test.go:1571`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app_test.go#L1571) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app_test.go#L1571) — Invert this test once the fix lands: assert `InitChain` rejects a tx with `BlockHeight > 0 && ChainID = "unknown" && "unknown" ∉ PastChainIDs`.
  <details><summary>details</summary>

  Currently the test signs the tx with the current chain ID and expects success, which silently encodes the buggy behavior. After the fix the test should sign with `"unknown-chain"` (or anything), and assert the genesis is rejected with an explicit allowlist error. Same shape for the "empty `metadata.ChainID` with `BlockHeight > 0`" case — missing entirely.
  </details>

- **[malformed PastChainIDs]** [`gno.land/pkg/gnoland/genesis.go:264`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/genesis.go#L264) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/genesis.go#L264) — No tests cover `ValidateGenState` rejecting empty/oversized/duplicated entries in `PastChainIDs` or negative `InitialHeight`.
  <details><summary>details</summary>

  Mirror the existing `genesis_test.go` shape: a table with `name`, `state`, `wantErr`. Cases: `PastChainIDs: [""]`, `PastChainIDs: ["x", "x"]`, `PastChainIDs: [strings.Repeat("a", MaxChainIDLen+1)]`, `InitialHeight: -1`.
  </details>

- **[InitialHeight desync]** [`gno.land/pkg/gnoland/node_initial_height_test.go:34-50`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/node_initial_height_test.go#L34-L50) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/node_initial_height_test.go#L34-L50) — The full-boot test only sets `GenesisDoc.InitialHeight`; it doesn't assert what happens when `GnoGenesisState.InitialHeight ≠ GenesisDoc.InitialHeight`.
  <details><summary>details</summary>

  Add a test pair: `GenesisDoc.InitialHeight=100, GnoGenesisState.InitialHeight=42` — chain still starts at 100, log line is misleading. Either assert the misleading-log status quo (current behavior) or (preferred) assert the new validation that requires the two to match.
  </details>

- **[bounded BlockHeight in metadata]** [`gno.land/pkg/gnoland/app.go:419`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L419) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L419) — No test covers `metadata.BlockHeight > GenesisDoc.InitialHeight`.
  <details><summary>details</summary>

  Should be rejected at InitChain. Currently silently accepted; a replay tx claims a future height.
  </details>

## Suggestions

- [`gno.land/pkg/gnoland/app.go:412-435`](https://github.com/gnolang/gno/blob/d9360e2/gno.land/pkg/gnoland/app.go#L412-L435) · [↗](../../../../../.worktrees/gno-review-5489/gno.land/pkg/gnoland/app.go#L412-L435) — Extract the `ctxFn` builder into a named function `buildHistoricalTxContextFn(metadata, pastChainIDs)` for readability and direct testability.
  <details><summary>details</summary>

  Currently the closure builds context, captures `state.PastChainIDs` from outer scope, and is only exercised via the full `InitChain` path. A standalone function lets unit tests assert the chain-ID override matrix (16 combinations: 2 BlockHeight × 2 ChainID-empty × 2 ChainID-in-allowlist × 2 Timestamp) without spinning up an app. Would have caught the piux2 fail-open earlier.
  </details>

- [`tm2/pkg/bft/consensus/replay.go:336`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/consensus/replay.go#L336) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/consensus/replay.go#L336) — Validate `h.genDoc.InitialHeight` against `state.LastBlockHeight` to detect a chain that has already produced blocks but is now being told to "start at InitialHeight" — that should be a hard error, not silently ignored by the `stateBlockHeight == 0` guard.
  <details><summary>details</summary>

  As written, if a node restarts with a modified genesis containing a different `InitialHeight`, the `stateBlockHeight == 0` guard at [`replay.go:318`](https://github.com/gnolang/gno/blob/d9360e2/tm2/pkg/bft/consensus/replay.go#L318) · [↗](../../../../../.worktrees/gno-review-5489/tm2/pkg/bft/consensus/replay.go#L318) means the change is silently ignored. That's the correct behavior (you can't retroactively rebase a chain), but it should be loud — `if h.genDoc.InitialHeight > 1 && state.LastBlockHeight > h.genDoc.InitialHeight { logger.Warn("genesis InitialHeight is in the past relative to current state, ignoring") }` or stricter.
  </details>

## Questions for Author

- Is the fail-open behavior (tx with `metadata.ChainID = "unknown"` accepted under current chain ID) intentional, or just an artifact of the test setup? piux2's read and mine agree it should fail-closed; the ADR also reads fail-closed.
- Does `gnolang/tx-archive#70` already populate `metadata.ChainID` for all exported txs, or are some left empty (legacy export format)? If some are empty, the empty-ChainID question becomes a backward-compat issue, not just a fail-open bug.
- The auto-account-creation skip when `BlockHeight > 0` requires `state.Balances` to include every signer with the matching `AccountNumber`. Does the planned hardfork tooling (#5411) enforce balance ordering, or is the operator on the hook?
- Was `GnoGenesisState.InitialHeight` kept (instead of dropping it) for forward compat with a future ABCI `RequestInitChain.InitialHeight`? If yes, document that in the ADR's "Decision" section.
- Why is the genesis-detector heuristic split across `Height == 1` (block.go), `LastBlockID.IsZero()` (validation.go, state.go), `LastCommit.Size() == 0` (execution.go), and `store.Height() == 0` (store.go, reactor.go, baseapp.go, state.go)? Each site picks a different invariant; a unified `state.IsAtGenesis()` helper would centralize the rule and prevent future skew.
