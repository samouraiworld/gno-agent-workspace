# PR #4951: chore: provide a better out-of-gas error UX

URL: https://github.com/gnolang/gno/pull/4951
Author: D4ryl00 | Base: master | Files: 14 | +501 -46
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `44a8373` (stale — +36 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4951 44a8373`

**Verdict: REQUEST CHANGES** — fix the misleading `suggested gas-wanted` value emitted on server-side OOG (computed on partial pre-panic gas) and the unconditional `simulate with consensus maximum` hint on DeliverTx panics; rest of the PR is a solid UX rework.

## Summary

Reworks gas simulation and out-of-gas (OOG) UX. Client (`gnokey`) now fetches the chain's consensus `Block.MaxGas` (in a goroutine, parallel with the account query), rewrites `tx.Fee.GasWanted` to that ceiling when running `--simulate test/only`, and either reports `gasUsed` (and a +5% suggestion) on success or synthesizes a client-side OOG error explaining the original `gas-wanted` was too low. Server side, the two OOG recover paths (`runTx` in [`baseapp.go:778-792`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/sdk/baseapp.go#L778-L792) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/sdk/baseapp.go#L778-L792) and the ante handler in [`ante.go:80-91`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/sdk/auth/ante.go#L80-L91) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/sdk/auth/ante.go#L80-L91)) are unified behind a shared [`store.OutOfGasLog`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/store/types/gas.go#L132) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/store/types/gas.go#L132) helper that distinguishes "exceeds tx's gas wanted" from "exceeds max block gas" and optionally appends a "simulate with consensus maximum" hint. Failed-tx CLI output now prints the same metrics block (gas, storage, events, info, hash) as success.

```
gnokey maketx call …
              │
              ├─ async: fetchConsensusMaxGas → maxGasCh
              ├─ sync : QueryHandler(auth/accounts/<addr>)
              │
              ↓
        resolveMaxGas (drains maxGasCh)
              │
              ↓
   SimulateTx with tx.GasWanted := maxGas  (rewritten only when user-gas < maxGas)
              │
       ┌──────┴──────┐
       │             │
   server OK     server OOG
       │             │
  gasUsed > orig?    │
       │             │
   client OOG    keep server error
   (no hint)     (server-rendered message)
       │             │
       └──── appendSuggestedGasWanted (gasUsed + 5%)   ← BUG when server OOG'd: gasUsed is partial
              ↓
       broadcast or return
```

## Glossary

- `simulateMaxGas` — `BroadcastCfg.simulateMaxGas`, the consensus `Block.MaxGas` the client uses as the ceiling for the rewritten simulation tx. 0 = "unknown / fetch failed", -1 = "chain has no gas limit" (falls back to `MaxInt64`).
- `rewritten` — second return of `buildSimulationTxBytes`; true when the client substituted `simulateMaxGas` for the user's `GasWanted` before sending to `.app/simulate`.
- `withSimulateHint` — last arg of `OutOfGasLog`; appends "simulate with consensus maximum (X) to get real transaction usage" when true and `maxGas > 0` and `gasWanted < maxGas`.
- `appendSuggestedGasWanted` — writes `suggested gas-wanted (gas used + 5%): N` into `DeliverTx.Info`, computed via `suggestedGasWanted(gasUsed) = gasUsed + ceil(gasUsed/20)`.

## Fix

Before: simulation ran with the user's `GasWanted`, so an underestimated value caused the server to OOG mid-execution and surface a `gasUsed` that was partial (gas-until-panic). The user kept bumping `--gas-wanted` and re-broadcasting blindly. Two recover paths (`baseapp.runTx`, `auth.NewAnteHandler`) each formatted their own OOG strings with different shapes.

After: when `Simulate != skip`, the client fetches `consensus.Block.MaxGas` concurrently with the account query and rewrites the simulated tx's `GasWanted` to that ceiling (or `MaxInt64` if `maxGas == -1`). Successful simulation returns the real `gasUsed`; the client annotates the response with a `+5%` suggestion and (for `--simulate only`) a fee estimate. If `gasUsed > originalGasWanted`, the client synthesizes its own `std.OutOfGasError` whose log explains the original `gas-wanted` was too low. Server-side, both recover paths now route through [`store.OutOfGasLog`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/store/types/gas.go#L132) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/store/types/gas.go#L132) for a single canonical message.

## Critical (must fix)

- **[suggested gas misleads when server OOG'd]** [`tm2/pkg/crypto/keys/client/broadcast.go:156`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L156) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L156) — `appendSuggestedGasWanted` runs on `gasUsed` even when the server panicked mid-execution, producing a `suggested gas-wanted = partial_gas + 5%` that is a known-wrong number.
  <details><summary>details</summary>

  When `rewritten=true` and the rewritten simulation OOG'd server-side (the tx actually needs more than the consensus `Block.MaxGas`), `res.DeliverTx.Error != nil` and `res.DeliverTx.GasUsed` is the gas consumed before the panic. The branch at [`broadcast.go:144`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L144) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L144) is skipped (the server error is preserved), but `appendSuggestedGasWanted(res)` still runs at [`broadcast.go:156`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L156) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L156). The user is told `suggested gas-wanted (gas used + 5%): N` where `N` is partial — exactly the same UX bug this PR set out to fix, just shifted from `Data` into `Info`. Same shape applies when `simulateMaxGas == 0` (consensus fetch failed): `rewritten=false`, server runs simulation with the user's small `GasWanted`, OOG's, and the client still suggests `partial_gas + 5%`. @thehowl flagged the conceptual case in [PR thread on baseapp.go](https://github.com/gnolang/gno/pull/4951#discussion_r2733114878) ("we don't actually know what the suggested value should be"); the client now reintroduces the same trap. Fix: gate `appendSuggestedGasWanted` on `res.DeliverTx.Error == nil` (success only), or pass an explicit flag indicating the run completed.
  </details>

## Warnings (should fix)

- **[wrong hint on DeliverTx panic]** [`tm2/pkg/sdk/baseapp.go:784`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/sdk/baseapp.go#L784) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/sdk/baseapp.go#L784) — `OutOfGasLog(..., withSimulateHint=true)` is called regardless of `mode`, so real on-chain DeliverTx OOG'd txs now tell users "simulate with consensus maximum to get real transaction usage" even though the user already simulated and broadcast.
  <details><summary>details</summary>

  A common DeliverTx OOG happens when CheckTx passed (or was bypassed by `--simulate skip`) but state shifted between blocks and the tx genuinely needs more gas. Telling the operator/relayer "go simulate" in the on-chain `result.Log` is misleading — they're past simulation, and the on-chain log ends up in block events, indexers, and explorers. The hint is useful for simulation-mode OOG (which is what `app.Simulate` triggers) but not for DeliverTx. Same applies to the ante handler at [`ante.go:86`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/sdk/auth/ante.go#L86) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/sdk/auth/ante.go#L86). Fix: thread the run mode into the recover and only set `withSimulateHint=true` when `mode == RunTxModeSimulate`.
  </details>

- **[tx-size cost inflation on rewritten sim]** [`tm2/pkg/crypto/keys/client/broadcast.go:189-191`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L189-L191) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L189-L191) — rewriting `GasWanted` to consensus max (`MaxInt64` in the fallback) re-marshals the tx with a larger varint, charging extra `TxSizeCostPerByte` gas in the ante; reported `gasUsed` and the `+5%` suggestion are biased high vs the real tx.
  <details><summary>details</summary>

  The author acknowledged this in [a PR comment](https://github.com/gnolang/gno/pull/4951#issuecomment-2682447831) ("if rewritten GasWanted changes encoded tx size by 1 byte, you get a +10 gas delta") and accepted it as the cost of moving the gas-cap decision to the client. The delta is small (10s of gas per added varint byte; the `TxSizeCostPerByte` default is 10) — but the suggestion is supposed to be the value the user should set for the real tx, and the real tx will encode a *smaller* `GasWanted` than what was simulated, lowering txSize cost. The bias is benign for `simulateMaxGas` in the millions, but for the `simulationMaxGasFallback = MaxInt64` path (chain reports `Block.MaxGas == -1`), the delta is a handful of varint bytes — still bounded, but worth a one-line comment in the docstring of [`buildSimulationTxBytes`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L178) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L178) so a future reader chasing a "why does simulate report N+10 gas?" report finds the answer. Fix: either subtract the delta from the suggestion (precise but fragile), or document the bias and move on.
  </details>

- **[silent fallback to user gas-wanted when fetch fails]** [`tm2/pkg/crypto/keys/client/maketx.go:283-287`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/maketx.go#L283-L287) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/maketx.go#L283-L287) — when `fetchConsensusMaxGas` errors, `resolveMaxGas` prints a `warning:` to stderr and returns 0, so `buildSimulationTxBytes` keeps the user's `GasWanted` for simulation; with an underestimated `gas-wanted` this falls straight into the partial-gas bug in the previous finding.
  <details><summary>details</summary>

  The warning is appropriate, but the downstream behavior changes from "simulation gives a usable number" to "simulation OOGs with an unreliable suggestion" without telling the user that the suggested-gas-wanted line is now bogus. Two viable fixes: (a) when `simulateMaxGas == 0`, refuse to emit a `suggested gas-wanted` and instead say `cannot suggest gas-wanted: consensus max gas unknown, retry with -simulate skip or fix RPC connectivity`; (b) keep the warning but mark the suggestion as "(unreliable: based on partial gas)". Either path is better than the current silent-but-wrong number.
  </details>

- **[suggestion overflow on huge gasUsed]** [`tm2/pkg/crypto/keys/client/broadcast.go:199-205`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L199-L205) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L199-L205) — `overflow.Addp` panics on overflow; for `gasUsed` near `MaxInt64`, `Addp(gasUsed, margin)` will panic and kill the CLI mid-broadcast.
  <details><summary>details</summary>

  Reachable only if a chain runs with a huge `Block.MaxGas` (or `-1` and the user simulates a runaway tx), so the practical risk is small. But the function is called unconditionally on every successful simulation, and `Addp` is the more conservative cousin specifically meant to *panic* on overflow — here we want clamping. @thehowl proposed [`margin++` over the `if margin == MaxInt64 { ... }` rewrite](https://github.com/gnolang/gno/pull/4951#discussion_r2873894412); the author's safer formula avoids one overflow but leaves the final `Addp` exposed. Fix: clamp at `MaxInt64` (`if margin > MaxInt64 - gasUsed { margin = MaxInt64 - gasUsed }`) and drop the panic-prone `Addp`.
  </details>

## Nits

- [`tm2/pkg/crypto/keys/client/broadcast.go:37`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L37) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L37) — `simulationMaxGasFallback = MaxInt64` is a package-private constant; a one-line comment noting it's only reached when `Block.MaxGas == -1` would help the next reader.
- [`tm2/pkg/crypto/keys/client/broadcast.go:209`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L209) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L209) — `"suggested gas-wanted (gas used + 5%): %d"` uses a literal `5%`; if you ever tune the margin, this string drifts from the constant (currently the 5% lives in the `gasUsed/20` ratio at line 200, unnamed). Extract `const suggestedGasMarginPercent = 5` and reuse in both spots.
- [`tm2/pkg/crypto/keys/client/maketx.go:284`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/maketx.go#L284) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/maketx.go#L284) — `"warning: could not fetch consensus max gas, simulated gas limit will be unbounded"` says "unbounded" but the actual behavior is "uses user's `--gas-wanted` unchanged". Misleading. Suggest `"warning: could not fetch consensus max gas; simulation will run with user-provided --gas-wanted (may OOG with partial gas reported): %v"`.
- [`tm2/pkg/store/types/gas.go:132`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/store/types/gas.go#L132) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/store/types/gas.go#L132) — the `withSimulateHint bool` last-arg pattern is fine for two callers, but if the mode-aware fix in the Warnings section lands, consider promoting it to an enum (`OOGContext{Simulate, Deliver, Check}`) — bool args at call sites are easy to flip wrong.

## Missing Tests

- **[no test for server-OOG + suggested gas]** [`tm2/pkg/crypto/keys/client/broadcast.go:156`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L156) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L156) — the existing `TestAppendSuggestedGasWanted` only covers success-path `GasUsed = 100`; there's no test asserting what happens when `DeliverTx.Error != nil`.
  <details><summary>details</summary>

  Add a test that constructs a `ctypes.ResultBroadcastTxCommit` with `DeliverTx.Error = abci.StringError("out of gas")` and `GasUsed = 47` (partial) and asserts the post-fix behavior: `suggested gas-wanted` is NOT appended (or is annotated as unreliable). Pairs with the Critical finding above.
  </details>

- **[no DeliverTx vs Simulate hint coverage]** [`tm2/pkg/sdk/baseapp_test.go:1066-1146`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/sdk/baseapp_test.go#L1066-L1146) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/sdk/baseapp_test.go#L1066-L1146) — `TestOOGLogBeyondGasWanted` asserts the hint is present on a DeliverTx OOG, locking in the wrong behavior described in the Warnings section.
  <details><summary>details</summary>

  Once the mode-aware fix lands, this test should split: DeliverTx OOG → no hint; Simulate OOG → hint present. The current test would block the fix because it explicitly asserts `"simulate with consensus maximum (200) to get real transaction usage"` is in the DeliverTx log.
  </details>

## Suggestions

- [`tm2/pkg/crypto/keys/client/broadcast.go:142-149`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/broadcast.go#L142-L149) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/broadcast.go#L142-L149) — the synthesis of a client-side OOG when `rewritten && gasUsed > originalGasWanted` is clean, but the synthesized `res.DeliverTx.Error` is set via `abci.ABCIErrorOrStringError(std.ErrOutOfGas(log))` while the server-side OOG uses `ABCIError(std.ErrOutOfGas(log))` (see [`baseapp.go:785`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/sdk/baseapp.go#L785) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/sdk/baseapp.go#L785)). Worth checking if both produce identical wire shapes — diverging error types here will confuse downstream consumers (indexers, error-classification logic).
- [`gno.land/pkg/keyscli/root.go:77-97`](https://github.com/gnolang/gno/blob/44a8373/gno.land/pkg/keyscli/root.go#L77-L97) · [↗](../../../../../.worktrees/gno-review-4951/gno.land/pkg/keyscli/root.go#L77-L97) — `PrintTxMetrics` is now called from both success and failure paths, but the success branch in `PrintTxInfo` still calls `io.Println("OK!")` *before* the metrics block, while failures print metrics *before* the error trace. Consistent ordering (header → metrics → status/error) would scan better.

## Questions for Author

- The new `fetchConsensusMaxGas` goroutine in [`maketx.go:137-144`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/maketx.go#L137-L144) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/maketx.go#L137-L144) shares no `context.Context` with the surrounding `SignAndBroadcastHandler` — if the user Ctrl-C's during the account query, does the consensus query leak past process exit? (Likely fine because the channel is buffered, but worth confirming the RPC client honors a shutdown.)
- For chains with `Block.MaxGas == -1` (no limit), the simulation tx is rewritten to `GasWanted = MaxInt64`. Does any existing ante check reject `GasWanted > some_internal_cap` before the gas meter is installed? If so, the simulation would fail unrelated to gas. The added test [`TestBuildSimulationTxBytesUsesFallbackWhenConsensusMaxGasUndefined`](https://github.com/gnolang/gno/blob/44a8373/tm2/pkg/crypto/keys/client/maketx_test.go#L80) · [↗](../../../../../.worktrees/gno-review-4951/tm2/pkg/crypto/keys/client/maketx_test.go#L80) only covers the byte-level rewrite, not the server-side acceptance.
