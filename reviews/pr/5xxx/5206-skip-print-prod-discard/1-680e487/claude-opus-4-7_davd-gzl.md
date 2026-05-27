# PR #5206: feat(gnovm): skip print/println in production discard-output mode

URL: https://github.com/gnolang/gno/pull/5206
Author: omarsy | Base: master | Files: 11 | +85 -63
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5206 680e487` (then `gh -R gnolang/gno pr checkout 5206` inside it)

Verdict: NEEDS DISCUSSION — fast-path is safe in current production wiring, but breaks `MsgRun` DeliverTx.Data semantics (RPC/CLI regression flagged by [@thehowl](https://github.com/gnolang/gno/pull/5206#pullrequestreview-3070000000)), creates a latent consensus-divergence trap if any validator sets `VMOutput`, and silently removes the gas charge for `print`/`println` in production. Decide whether to land it as-is, narrow it to format-only skipping while still capturing into a per-tx buffer (preserves DeliverTx.Data + gas), or take the broader scalar-only redesign thehowl proposes.

## Summary

Adds an early-return in the `print`/`println` uverse natives when `m.Output == io.Discard && !bm.NativeEnabled` so production (where the keeper passes `vm.Output == nil`, which `NewMachineWithOptions` coerces to `io.Discard`) skips all formatting work. The keeper's `Run` is simplified: the per-transaction `bytes.Buffer` is removed and the machine writes directly to `vm.Output`. Net effect in production: `print`/`println` are no-ops at zero gas cost; `MsgRun`'s `DeliverTx.Data` is now always `""`. Integration txtars are migrated from `println(...)` + `stdout '...'` assertions to in-script `if got != want { panic(...) }` checks.

## Glossary

- `vm.Output` — `io.Writer` field on `VMKeeper`, set from `AppOptions.VMOutput`. `gnoland` binary never sets it (nil); `gnodev` sets `os.Stdout`.
- `m.Output` — machine output sink. `NewMachineWithOptions` coerces `nil` to `io.Discard` ([`machine.go:114-117`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/machine.go#L114-L117) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/machine.go#L114-L117)).
- `bm.NativeEnabled` — compile-time `const` from build tag `benchmarkingnative`; `false` in production, `true` in bench builds.
- `uversePrint` — gas-charging path that formats args and writes to `m.Output` ([`uverse.go:1224-1231`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/uverse.go#L1224-L1231) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/uverse.go#L1224-L1231)).
- `MsgRun` / `DeliverTx.Data` — `MsgRun` previously surfaced the captured stdout buffer as the tx response data, exposed over RPC and printed by `gnokey maketx run`.

## Fix

Before: production `print`/`println` always ran `formatUverseOutput` + `consumeGas` then wrote to a `bytes.Buffer` (captured per-tx in `keeper.Run`) which became `DeliverTx.Data`. After: when the sink is `io.Discard` and the bench build tag is off, the native returns immediately at [`uverse.go:917-921`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/uverse.go#L917-L921) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/uverse.go#L917-L921) and [`uverse.go:944-948`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/uverse.go#L944-L948) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/uverse.go#L944-L948); `keeper.Run` no longer holds a buffer and `res` returns the zero `""` ([`keeper.go:905-940`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/sdk/vm/keeper.go#L905-L940) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/sdk/vm/keeper.go#L905-L940)). The load-bearing gate is the exact identity check `m.Output == io.Discard`: only the default-`nil`-coerced sink triggers the skip, and any explicit writer (test bench, gnodev) bypasses it.

## Critical (must fix)

- **[MsgRun DeliverTx.Data becomes empty without doc/changelog]** [@thehowl](https://github.com/gnolang/gno/pull/5206#pullrequestreview-2725816432) [`keeper.go:905-940`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/sdk/vm/keeper.go#L905-L940) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/sdk/vm/keeper.go#L905-L940) — RPC contract change: `broadcast_tx_commit` `data` field for `MsgRun` is now always `""`.
  <details><summary>details</summary>

  `keeper.Run` previously returned `buf.String()` which the handler wrote to `res.Data` ([`handler.go:60-66`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/sdk/vm/handler.go#L60-L66) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/sdk/vm/handler.go#L60-L66)). That value is exposed over RPC (`/broadcast_tx_commit`) and printed by `gnokey maketx run` at [`broadcast.go:91`](https://github.com/gnolang/gno/blob/680e487/tm2/pkg/crypto/keys/client/broadcast.go#L91) · [↗](../../../../../.worktrees/gno-review-5206/tm2/pkg/crypto/keys/client/broadcast.go#L91) via `io.Println(string(res.DeliverTx.Data))`. With this PR, that line always renders empty in production — the `main: --- hello from foo ---` lines disappear from CLI output, and any RPC consumer that parses MsgRun output (e.g. simulate-based reads, as thehowl describes) gets nothing. The txtar deletions in [`adduser.txtar:13`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/integration/testdata/adduser.txtar#L13) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/integration/testdata/adduser.txtar#L13), [`loadpkg_work.txtar:11`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/integration/testdata/loadpkg_work.txtar#L11) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/integration/testdata/loadpkg_work.txtar#L11), [`maketx_run.txtar:10`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/integration/testdata/maketx_run.txtar#L10) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/integration/testdata/maketx_run.txtar#L10), [`maketx_run_send.txtar:9`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/integration/testdata/maketx_run_send.txtar#L9) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/integration/testdata/maketx_run_send.txtar#L9), and [`prevrealm.txtar:63-91`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/integration/testdata/prevrealm.txtar#L63-L91) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/integration/testdata/prevrealm.txtar#L63-L91) confirm the regression — every `stdout 'main: ---'` / `stdout ${ADDR}` assertion is removed. Fix: either (a) preserve a per-tx buffer behind `m.Output` (write to `io.MultiWriter(buf, vm.Output)` only when needed, so DeliverTx.Data stays populated and the skip remains for true-`Discard` standalone callers), (b) document this as an intentional breaking change in the PR body + changelog and audit downstream consumers (gnoclient docs, indexers, faucet), or (c) take thehowl's scalar-only redesign which keeps println functional but bounded.
  </details>

## Warnings (should fix)

- **[gas accounting silently changes across environments]** [`uverse.go:917-921`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/uverse.go#L917-L921) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/uverse.go#L917-L921) — same `print` call costs `1376 + len(output)/10` gas under `gnodev` (Output=os.Stdout) and **0 gas** under `gnoland` (Output=Discard).
  <details><summary>details</summary>

  `consumeGas(m, NativeCPUUversePrintInit)` and the per-byte charge in [`uverse.go:1225-1227`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/uverse.go#L1225-L1227) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/uverse.go#L1225-L1227) are bypassed entirely when the early-return fires. Practical consequences: (1) gas estimation done locally against `gnodev` over-estimates production cost, so users overpay; (2) realms whose gas profile depends on `println` calls behave differently across local-dev and prod; (3) if any validator/full-node operator ever sets `VMOutput` (e.g. for log forwarding), that node will charge gas while peers don't — `GasConsumed` diverges, breaking deterministic block replay. There's no code guard preventing operators from setting `VMOutput`. Fix: make the skip semantically separate from gas — always `consumeGas` for `print`/`println` based on the would-be output size (compute via `formatUverseOutput(m, arg0, newline)` then discard the bytes), and only skip the actual `m.Output.Write`. That keeps gas deterministic regardless of operator wiring while still saving the write syscall. Alternative: hard-code production behavior by removing the `vm.Output` knob and forcing `Discard` consensus-side.
  </details>

- **[`vm.Output` plumbed unguarded to all keeper paths]** [`keeper.go:133,389,628,714,911,931,1153`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/sdk/vm/keeper.go#L905-L940) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/sdk/vm/keeper.go#L905-L940) — every machine construction inherits the operator-configurable `vm.Output`, so the consensus-divergence surface is wide, not just `Run`.
  <details><summary>details</summary>

  The early-return depends on the exact identity `m.Output == io.Discard`. Any operator who wires `VMOutput` to a non-discard writer (capture-for-logs, log-forwarding to a sidecar, debugging on a sentry) flips the gas accounting on that node only. Today no production binary does this, but the door is open and there's no comment or runtime check warning against it. Fix: either remove the `VMOutput` field from `AppOptions` (gnoland never uses it) and keep it as a gnodev-only concept, or add a comment + lint pointing maintainers at the determinism contract. If the field stays, consider asserting `vm.Output == io.Discard || vm.Output == nil` at consensus path entry and panicking otherwise on chain-id matching mainnet/testnet.
  </details>

- **[print/panic asymmetry preserved]** [@thehowl](https://github.com/gnolang/gno/pull/5206#pullrequestreview-2776892712) [`op_call.go:544`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/op_call.go#L544) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/op_call.go#L544) — PR addresses `print`/`println` but `panic` still calls `TypedValue.Sprint` uncharged on arbitrary values.
  <details><summary>details</summary>

  thehowl's review observes that the same unbounded-formatting concern applies to `panic`: `makeUnhandledPanicError` calls `last.Sprint(m)` on every exception in the chain ([`op_call.go:539-549`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/op_call.go#L539-L549) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/op_call.go#L539-L549)) without gas charging. A malicious realm can `panic(largeStruct)` to consume CPU at zero gas — outside this PR's scope, but if the framing is "remove the attack vector via discard-output", `panic` keeps the door open. The scalar-only redesign thehowl proposes (println/panic accept only ints/strings/Stringer/error, render `<TYPE value>` otherwise, charge gas on panic) fixes both at once and preserves `MsgRun` semantics. Worth a design call before landing this PR — if the broader fix is on the roadmap, this PR may be premature.
  </details>

## Nits

- [`prevrealm.txtar:110-112`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/integration/testdata/prevrealm.txtar#L110-L112) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/integration/testdata/prevrealm.txtar#L110-L112) — `Addr(cur realm) string` is added to `r/foo` and never called by any of the run scripts. Dead code.
- [`prevrealm.txtar:176-183`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/integration/testdata/prevrealm.txtar#L176-L183) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/integration/testdata/prevrealm.txtar#L176-L183) — `run/bar-b.gno` still uses `println(bar.B())` but the calling test case (`12`) is commented out. Either delete the file or migrate it to the panic-on-mismatch shape for consistency.
- [`uverse.go:918-920`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/uverse.go#L918-L920) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/uverse.go#L918-L920) — comment says "skip print formatting work to avoid unnecessary runtime/gas overhead" but the actual effect is also to skip the gas charge itself. Phrase it as "skip formatting AND gas accounting" so the next reader doesn't miss the consensus implication.
- The duplicated check + identical body in `print` and `println` defNatives could share a helper; not worth holding the PR for.

## Missing Tests

- **[no test for production-mode gas behavior]** [`gnovm/pkg/gnolang/uverse_test.go`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/) — there's no test that asserts `consumeGas` is (or isn't) called when `m.Output == io.Discard`.
  <details><summary>details</summary>

  The PR adds [`keeper_test.go:899-906`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/sdk/vm/keeper_test.go#L899-L906) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/sdk/vm/keeper_test.go#L899-L906) which exercises the `vm.Output != nil` path only. The production path (`vm.Output == nil`, machine sees `io.Discard`) is untested at the unit level. A regression in [`uverse.go:917`](https://github.com/gnolang/gno/blob/680e487/gnovm/pkg/gnolang/uverse.go#L917) · [↗](../../../../../.worktrees/gno-review-5206/gnovm/pkg/gnolang/uverse.go#L917) — e.g. someone changing the identity check to `m.Output != nil` — would silently re-enable formatting without test coverage. Fix: add a focused test that constructs a Machine with `Output: nil`, runs `println("x")` and asserts (a) gas consumed is 0 (or whatever the post-design value is), (b) no panic, (c) `m.Output` is unchanged. Mirror for `Output: io.Discard`.
  </details>

- **[no test for `Run` returning empty string]** [`keeper_test.go`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/sdk/vm/keeper_test.go) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/sdk/vm/keeper_test.go) — `Run` returning `""` in the default config is now load-bearing for the RPC contract and should be explicitly asserted.
  <details><summary>details</summary>

  Add a test that runs `MsgRun` with `vm.Output == nil`, then asserts `res == ""` even when the script calls `println`. Currently the only test ([`keeper_test.go:903`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/sdk/vm/keeper_test.go#L903) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/sdk/vm/keeper_test.go#L903)) sets `env.vmk.Output = &out` and asserts `res == ""`, but that's incidental (always empty post-PR), not a verified contract for the production path.
  </details>

## Suggestions

- [`keeper.go:905-940`](https://github.com/gnolang/gno/blob/680e487/gno.land/pkg/sdk/vm/keeper.go#L905-L940) · [↗](../../../../../.worktrees/gno-review-5206/gno.land/pkg/sdk/vm/keeper.go#L905-L940) — consider keeping a per-tx `bytes.Buffer` and only feeding it (not skipping print) so `DeliverTx.Data` continues to surface println output for `gnokey maketx run` while production gas charging is preserved.
  <details><summary>details</summary>

  Minimal alternative: revert `keeper.go` to the previous `output = io.MultiWriter(buf, vm.Output)` shape, drop the `uverse.go` early-return, and instead introduce a hard cap on `consumeGas` output length (or fail closed) so the "unbounded formatting" worry is addressed without losing user-visible behavior. This addresses the security framing without breaking the RPC contract.
  </details>

## Questions for Author

- Was the `MsgRun.DeliverTx.Data` regression intentional? If yes, can the PR body call it out as a breaking change and link the changelog/migration note for RPC consumers?
- Did you consider thehowl's scalar-only redesign for both `print` and `panic`? Is this PR meant to land first as a stop-gap, or as the final answer?
- The early-return identity-checks `io.Discard`. Why is that preferable to a typed flag on `Machine` (e.g. `m.DiscardOutput bool`) that doesn't depend on pointer-equality with `io.Discard`?
- Should `VMOutput` be removed from `AppOptions` entirely (since the production binary never sets it) to close the latent consensus-divergence risk, or kept for gnodev-style flexibility with an explicit determinism comment?
