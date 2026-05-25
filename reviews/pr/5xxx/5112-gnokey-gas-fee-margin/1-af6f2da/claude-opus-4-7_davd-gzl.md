# PR #5112: chore(gnokey): Add simulate optional parameter -gas-fee-margin

URL: https://github.com/gnolang/gno/pull/5112
Author: jefft0 | Base: master | Files: 3 | +36 -5
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — Small, backwards-compatible flag; default `5` preserves the old hard-coded behaviour; covered by txtar including the parse-error paths. Already approved by `notJoon`, `Villaquiranm`, `moul`.

## Summary

`gnokey ... -simulate only` previously baked in a 5% buffer on top of the simulated fee. This PR exposes that buffer as `-gas-fee-margin` (uint64, default 5), so a caller can pick 0 (no buffer) or e.g. 10 (more headroom against a rising gas price between simulate and broadcast). The hard-coded `5` is replaced by `gasFeeMargin` passed through `BroadcastCfg` into `estimateGasFee`. `Uint64Var` rejects non-numeric and negative inputs at parse time, so no extra validation is needed. Only consumed when `Simulate == SimulateOnly`; ignored otherwise, as documented in the flag help.

## Fix

Before: `feeBuffer := overflow.Mulp(fee, 5) / 100` in [`broadcast.go:163`](../../../../../.worktrees/gno-review-5112/tm2/pkg/crypto/keys/client/broadcast.go#L163). After: `feeBuffer := overflow.Mulp(fee, int64(gasFeeMargin)) / 100` in [`broadcast.go:165`](../../../../../.worktrees/gno-review-5112/tm2/pkg/crypto/keys/client/broadcast.go#L165), with the value plumbed via `MakeTxCfg.GasFeeMargin` ([`maketx.go:28`](../../../../../.worktrees/gno-review-5112/tm2/pkg/crypto/keys/client/maketx.go#L28)) → `BroadcastCfg.GasFeeMargin` ([`broadcast.go:31`](../../../../../.worktrees/gno-review-5112/tm2/pkg/crypto/keys/client/broadcast.go#L31)) → `estimateGasFee` ([`broadcast.go:147`](../../../../../.worktrees/gno-review-5112/tm2/pkg/crypto/keys/client/broadcast.go#L147)). The default of `5` makes the change behaviour-preserving for every existing caller.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`tm2/pkg/crypto/keys/client/broadcast.go:165`](../../../../../.worktrees/gno-review-5112/tm2/pkg/crypto/keys/client/broadcast.go#L165) — `int64(gasFeeMargin)` narrows a `uint64` parsed by `Uint64Var`. Any value `>= 2^63` wraps to negative and (a) flips the sign of `feeBuffer`, then (b) `overflow.Addp(fee, feeBuffer)` panics, or (c) `Mulp` panics on the negative product. Pure CLI nuisance — no realistic operator types a margin of nine quintillion — but a `uint64` flag whose only consumer takes `int64` is a small smell. Either declare the flag with `Int64Var` (already rejects negatives via parse) or clamp/validate in `MakeTxCfg.Validate()`.
- [`tm2/pkg/crypto/keys/client/maketx.go:113-118`](../../../../../.worktrees/gno-review-5112/tm2/pkg/crypto/keys/client/maketx.go#L113-L118) — help text says "only useful with `-simulate only`" but the flag is silently ignored under `-simulate test`/`-simulate skip`. Consider either emitting a warning when set together with a non-`only` mode, or just rewording the help to "ignored unless `-simulate only`". Minor.

## Missing Tests

- [`gno.land/pkg/integration/testdata/simulate_gas.txtar:15-25`](../../../../../.worktrees/gno-review-5112/gno.land/pkg/integration/testdata/simulate_gas.txtar#L15-L25) — txtar covers `-gas-fee-margin=0`, `=10`, default `=5`, `=zzz`, `=-10`. No assertion that the flag is in fact ignored under `-simulate test` (i.e. `gas fee:` line absent or unchanged). Not blocking.

## Suggestions

- [`tm2/pkg/crypto/keys/client/broadcast.go:31`](../../../../../.worktrees/gno-review-5112/tm2/pkg/crypto/keys/client/broadcast.go#L31) — `BroadcastCfg.GasFeeMargin` is exported but `BroadcastCfg.tx`/`testSimulate` are not. Field is only consumed within the package; lowercase would match the existing "set by SignAndBroadcastHandler" convention used right above for `testSimulate`. Cosmetic.

## Questions for Author

- Any reason `Uint64Var` was chosen over `Int64Var`? `Int64Var` already rejects negatives at parse time (the test at [`simulate_gas.txtar:32-33`](../../../../../.worktrees/gno-review-5112/gno.land/pkg/integration/testdata/simulate_gas.txtar#L32-L33) would still pass) and would eliminate the `int64(...)` cast.
