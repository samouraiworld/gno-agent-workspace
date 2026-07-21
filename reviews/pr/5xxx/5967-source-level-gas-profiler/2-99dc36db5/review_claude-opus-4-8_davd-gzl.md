# PR [#5967](https://github.com/gnolang/gno/pull/5967): feat(gnovm): source-level gas profiler ("gas pprof")

URL: https://github.com/gnolang/gno/pull/5967
Author: omarsy | Base: master | Files: 45 | +3209 -54
Reviewed by: davd-gzl | Model: claude-opus-4-8 (high, deep) | Commit: 99dc36db5 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5967 99dc36db5`

Round 2. Head advanced aeb88c9eb → 99dc36db5 (+2 commits, no rebase, no master merge): `39eaa8102` adds `.app/profiletx` to gnodev's lazy-load interceptor, `99dc36db5` renames `maketx -profile` to `-gasprofile` and prints that the tx was not broadcast. All three round-1 findings are resolved and re-verified live; the profiler engine (`gnovm/pkg/gasprof`, `gnovm/pkg/gnolang`, `gno.land/pkg/sdk/vm`, `tm2/pkg/sdk`) is byte-identical to round 1, so its verification carries.

**TL;DR:** Adds a tool that shows where a gno transaction's gas went, function by function, in your own gno source. It reads the gas the VM already charges, attributes each charge to the gno function that caused it, and writes a standard pprof file you open with `go tool pprof` (top functions, flame graphs, per-line). It is off by default, and when on it never changes the gas charged or the execution result.

**Verdict: APPROVE** — the interceptor gap that broke first-use is fixed and confirmed live on a fresh node; the flag now matches `gno test -gasprofile` and says when it skips the broadcast. Only an optional hardening Suggestion and no-change nits remain.

## Summary
A `GasMeter` decorator reads every gas charge (CPU, alloc, store I/O, amino, refunds) and an incremental call-tree cursor, driven by the machine's frame push/pop, attributes each charge to the current gno function; the tree is emitted as a multi-dimension pprof profile. Two surfaces drive it: `gno test -gasprofile=<file>` (unit tests and filetests) and a dev-only `.app/profiletx` ABCI query reachable via `gnokey maketx -gasprofile`, `gnoclient.ProfileTx`, and gnodev. The engine is a new self-contained package (`gnovm/pkg/gasprof`, stdlib + `tm2/pkg/store` only, no import cycle). The consensus gas path measures unchanged when profiling is off: the machine hooks are `nil`-guarded and short-circuit before evaluating `IsCall()`, and the `// Gas:` filetests report the same gas at this head as at the merge-base.

## Fix
`handleQuery` in [`contribs/gnodev/pkg/proxy/path_interceptor.go:316`](https://github.com/gnolang/gno/blob/99dc36db5/contribs/gnodev/pkg/proxy/path_interceptor.go#L316) · [↗](../../../../../.worktrees/gno-review-5967/contribs/gnodev/pkg/proxy/path_interceptor.go#L316) now matches `".app/simulate", ".app/profiletx"` in one case, so both amino-encoded-tx query paths trigger the same lazy package load. On the CLI, the flag and its config field became `-gasprofile`/`GasProfile`, and the success path prints that the transaction was not broadcast. Rejecting `-gasprofile` together with `-broadcast` was correctly not chosen: `-broadcast` defaults to true, so a guard would reject every documented invocation, which [`maketx_test.go:189-205`](https://github.com/gnolang/gno/blob/99dc36db5/tm2/pkg/crypto/keys/client/maketx_test.go#L189-L205) · [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/crypto/keys/client/maketx_test.go#L189-L205) pins by parsing through `RegisterFlags` rather than a struct literal.

## Verified
- **The first-use failure is gone (live, decisive).** Booted gnodev from this branch and ran `gnokey maketx call -gasprofile` against `gno.land/r/demo/defi/foo20` on a fresh node with the package never loaded and no prior `.app/simulate`. It now reports `ok` and writes a full tree: `foo20.Faucet` 48.28% flat, with `grc20.PrivateLedger.Mint` and `balanceOf` beneath it. The proxy log contains zero `unhandled: ".app/profiletx"` lines, against one at round 1's sha. Same numbers as round 1's post-load run, so the fix restores the correct profile rather than a different one.
- **The not-broadcast notice prints.** The same run emitted `transaction was NOT broadcast (-gasprofile profiles instead of sending)`, and the integration txtar asserts both that line and the absence of `TX HASH`.
- **Rename is complete.** No `"profile"` flag string, `cfg.Profile`, or `-profile` reference survives in Go sources, docs, the ADR, or the txtar; the only remaining `--profile-gas` mentions are Aptos prior-art. Help text now points at `gno test -gasprofile`.
- **Gas is measurably unchanged vs the merge-base.** The `gnovm/tests/files/gas/*.gno` filetests assert exact gas via `// Gas:` directives, and the PR modifies none of those goldens. Ran them at this head and at the merge-base `959cefd91`: the observed value is identical on both sides (`slice_alloc` reports 70971242 each time). This is a measurement of the charged amount, not an inference from the `nil`-guards, and it covers only the paths those filetests exercise. Nine of those filetests fail on both sides with identical failing sets, so the failures are pre-existing on master and unrelated to this PR; CI is green on both, pointing at local Go version drift reddening the goldens.
- **Engine untouched.** `git diff aeb88c9eb..HEAD` reports no change under `gnovm/pkg/gasprof`, `gnovm/pkg/gnolang`, `gno.land/pkg/sdk/vm`, or `tm2/pkg/sdk`, so round 1's reconciliation, observation-only, dev-only-gating, cursor-exactness, and pprof-validity results still hold at this sha.
- Green at `99dc36db5`: `contribs/gnodev/pkg/proxy` (including the two new interceptor tests), `tm2/pkg/crypto/keys/client`, `gnovm/pkg/gasprof`, and the `maketx_gasprofile.txtar` integration test. CI is green except the manual merge-requirements bot.

## Critical (must fix)
None.

## Warnings (should fix)
None. Round 1's interceptor Warning is fixed at [`path_interceptor.go:316`](https://github.com/gnolang/gno/blob/99dc36db5/contribs/gnodev/pkg/proxy/path_interceptor.go#L316) · [↗](../../../../../.worktrees/gno-review-5967/contribs/gnodev/pkg/proxy/path_interceptor.go#L316) and covered by [`path_interceptor_internal_test.go:19-32`](https://github.com/gnolang/gno/blob/99dc36db5/contribs/gnodev/pkg/proxy/path_interceptor_internal_test.go#L19-L32) · [↗](../../../../../.worktrees/gno-review-5967/contribs/gnodev/pkg/proxy/path_interceptor_internal_test.go#L19-L32), which asserts both query paths yield the same loaded package.

## Suggestions
- **[invariant can decay invisibly]** [`gnovm/pkg/gasprof/gasprof.go:135`](https://github.com/gnolang/gno/blob/99dc36db5/gnovm/pkg/gasprof/gasprof.go#L135) · [↗](../../../../../.worktrees/gno-review-5967/gnovm/pkg/gasprof/gasprof.go#L135) — no debug assert ties the cursor depth to the machine's call-frame count.
  <details><summary>details</summary>

  The cursor-exactness invariant (`len(stack) == numGasCallFrames()+1`) is load-bearing but self-healing on failure: a desync never panics and never breaks reconciliation (the tree sum is unchanged, gas just lands on the wrong node), so no test would catch it. A future refactor that adds a frame-truncation path bypassing `Enter`/`Pop`/`SyncDepth` would mis-attribute silently. A `if debug { … }` assert at the `Enter`/`Pop` hooks ([`machine.go:2461`](https://github.com/gnolang/gno/blob/99dc36db5/gnovm/pkg/gnolang/machine.go#L2461) · [↗](../../../../../.worktrees/gno-review-5967/gnovm/pkg/gnolang/machine.go#L2461)) comparing the two would make such a regression fail loudly. Current code is correct; this is hardening. Carried from round 1, not posted.
  </details>

## Nits
- **[comment precision]** [`tm2/pkg/sdk/auth/ante.go:514`](https://github.com/gnolang/gno/blob/99dc36db5/tm2/pkg/sdk/auth/ante.go#L514) · [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/sdk/auth/ante.go#L514) — "simulation is still metered" is true past the genesis block, but the pre-first-commit `Simulate` fallback runs at height 0 and hits the infinite-meter branch. The comment's own "only genesis" carve-out already covers it. No change needed; not posted.
- **[discarded RPC on error]** [`tm2/pkg/crypto/keys/client/maketx.go:295`](https://github.com/gnolang/gno/blob/99dc36db5/tm2/pkg/crypto/keys/client/maketx.go#L295) · [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/crypto/keys/client/maketx.go#L295) — the consensus-max-gas goroutine launches before `signTx`, so a sign-setup failure fires one `fetchConsensusMaxGas` RPC whose result is discarded. Buffered channel, no leak. No change needed; not posted.
- **[documented edge]** [`gnovm/pkg/gasprof/gasprof.go:205`](https://github.com/gnolang/gno/blob/99dc36db5/gnovm/pkg/gasprof/gasprof.go#L205) · [↗](../../../../../.worktrees/gno-review-5967/gnovm/pkg/gasprof/gasprof.go#L205) — `ConsumeGas` records before delegating, so on int64 overflow (which panics before mutating) the profile over-counts by one charge. Requires ~9.2e18 gas, terminal and unreachable; already noted in the code comment. No change needed; not posted.
- **[test-only output]** [`gnovm/pkg/gasprof/gasprof.go:281`](https://github.com/gnolang/gno/blob/99dc36db5/gnovm/pkg/gasprof/gasprof.go#L281) · [↗](../../../../../.worktrees/gno-review-5967/gnovm/pkg/gasprof/gasprof.go#L281) — `flatTotal()` can go net-negative when a refund fires at a different cursor node than the original write. Only `WriteFolded` uses it, and only tests call that; shipped `WritePprof` keeps refund as its own non-negative index. No change needed; not posted.
- **[attribution granularity]** [`gnovm/pkg/gnolang/machine.go:1526`](https://github.com/gnolang/gno/blob/99dc36db5/gnovm/pkg/gnolang/machine.go#L1526) · [↗](../../../../../.worktrees/gno-review-5967/gnovm/pkg/gnolang/machine.go#L1526) — two anonymous functions on the same source line share `pkg.(anonymous)`+file+line and merge into one profile node. A granularity limit, not a correctness issue. No change needed; not posted.

## Missing Tests
None. Round 1's gap (no coverage of the `-gasprofile` plus default `-broadcast` invocation) is closed by [`maketx_test.go:189-205`](https://github.com/gnolang/gno/blob/99dc36db5/tm2/pkg/crypto/keys/client/maketx_test.go#L189-L205) · [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/crypto/keys/client/maketx_test.go#L189-L205), and the interceptor fix carries both an internal unit test and an end-to-end proxy subtest.

## Open questions
None.
