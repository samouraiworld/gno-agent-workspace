# PR #5388: fix(bank): add missing return after address parse error in `queryBalance`

URL: https://github.com/gnolang/gno/pull/5388
Author: notJoon | Base: master | Files: 2 | +25 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7

Verdict: APPROVE — minimal one-line fix restoring the missing `return` plus a regression test that empirically fails pre-fix and passes post-fix; no callers depend on the broken fall-through behavior.

## Summary

`queryBalance` in `tm2/pkg/sdk/bank/handler.go` set `res.Error` on a bech32 parse failure but did not `return`, so execution fell through to `bh.bank.GetCoins(ctx, addr)` with the zero-value `crypto.Address{}` and overwrote the response with the zero-address balance. The ABCI response then carried both `Error` and `Data` populated — a contract violation, with a minor information-leak surface (any coins held by the zero address would be exposed via any malformed-address query). The fix is a single `return` statement. Bug landed in 2021 (commit `36fff1370`, refactor from `amino.UnmarshalJSON` to `crypto.AddressFromBech32`); previous review by @jefft0 already APPROVED.

## Fix

Before, the `if err != nil` branch built a `sdk.ABCIResponseQueryFromError` into `res` and continued; `GetCoins` then ran on the zero address and `res.Data = bz` clobbered the response just before the final `return` (see [`tm2/pkg/sdk/bank/handler.go:121-136`](../../../../../.worktrees/gno-review-5388/tm2/pkg/sdk/bank/handler.go#L121-L136) in the pre-fix state shown in commit `d1c1c9504~1`). After, the added [`return`](../../../../../.worktrees/gno-review-5388/tm2/pkg/sdk/bank/handler.go#L124) short-circuits — `Error` set, `Data` empty, matching every other error branch in the file. The companion test [`TestQueryBalanceInvalidAddress`](../../../../../.worktrees/gno-review-5388/tm2/pkg/sdk/bank/handler_test.go#L58-L79) funds the zero address with a sentinel coin (`secret`, 999) and queries with `"notavalidaddress"`, asserting `res.Error != nil` and `res.Data` empty — a clean regression guard against future copy-paste regressions of the same shape.

## Empirical confirmation

Reverted [`tm2/pkg/sdk/bank/handler.go`](../../../../../.worktrees/gno-review-5388/tm2/pkg/sdk/bank/handler.go) to the pre-fix state (`git checkout d1c1c9504~1 -- tm2/pkg/sdk/bank/handler.go`) and ran the new test:

```
--- FAIL: TestQueryBalanceInvalidAddress (0.00s)
    Error: Should be empty, but was [34 57 57 57 115 101 99 114 101 116 34]
    Messages: invalid address should not return any balance data
```

Bytes `34 57 57 57 115 101 99 114 101 116 34` decode to `"999secret"` — confirming the zero-address coins leak through `res.Data` while `res.Error` is also set. Restoring the fix makes the test pass.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`tm2/pkg/sdk/bank/handler_test.go:65`](../../../../../.worktrees/gno-review-5388/tm2/pkg/sdk/bank/handler_test.go#L65) — `crypto.Address{}` comment "all-zero address" is fine but the test name `TestQueryBalanceInvalidAddress` slightly undersells the assertion: the test specifically guards against the cross-talk between a malformed query and the zero-address account. `TestQueryBalanceInvalidAddressDoesNotLeakZeroAddress` or a one-line comment naming the failure mode would age better.
- PR description typo: "execution falls through ro `GetConis`" → "to `GetCoins`". Cosmetic.

## Missing Tests

None. The added test is exactly the right shape — funding the zero address with a sentinel makes the bug observable; without funding, the pre-fix code would still return empty Data (zero-address has no coins) and the test would have been silent.

## Suggestions

None.

## Questions for Author

None.
