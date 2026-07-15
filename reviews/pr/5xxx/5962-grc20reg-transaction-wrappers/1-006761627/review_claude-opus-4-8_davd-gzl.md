# PR [#5962](https://github.com/gnolang/gno/pull/5962): feat(grc20reg): add transaction wrappers for registered tokens

URL: https://github.com/gnolang/gno/pull/5962
Author: notJoon | Base: master | Files: 2 | +108 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 006761627 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5962 006761627`

**TL;DR:** grc20reg is a registry that maps a string key to a registered GRC20 token. This PR adds functions that let anyone move, approve, or read a registered token's balances through that key, without importing the token's realm directly.

**Verdict: APPROVE** — caller identity is resolved correctly through the live crossing frame; only the immediate caller's own tokens can move; tests pass; no blocking concerns.

## Summary
Six read wrappers (`GetName`, `GetSymbol`, `GetDecimals`, `TotalSupply`, `BalanceOf`, `Allowance`) and three write wrappers (`Transfer`, `Approve`, `TransferFrom`) are added over `MustGet(tokenKey)`. Each write wrapper asserts `cur.IsCurrent()`, then dispatches through the token's `CallerTeller()`, which resolves the acting account lazily as `cur.Previous().Address()`. Because grc20reg passes its own `cur`, the account debited is always grc20reg's immediate caller: a direct user call spends the user, a realm cross-call spends the calling realm. `Register` also gains the same `cur.IsCurrent()` assertion before it reads `cur.Previous().PkgPath()`.

## Examples
Who gets debited when calling `grc20reg.Transfer(cross(cur), key, to, amt)`:

| Caller into grc20reg | Account debited | Why |
|----------------------|-----------------|-----|
| user Alice (direct maketx) | Alice | `cur.Previous()` is the origin user realm |
| realm `r/demo/vault` (cross-call) | `vault` package address | `cur.Previous()` is the calling realm |
| a spoofed / stale `cur` | none — panics `ErrSpoofedRealm` | `cur.IsCurrent()` is false |

## Glossary
- crossing / `cross`: a call into `func F(cur realm, ...)`; the callee identifies its caller through `cur.Previous()`.
- realm: stateful `r/` package; the `cur realm` builtin threads caller identity, `cur.Previous().Address()` being the unforgeable immediate caller.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None. The new getters carry no doc comments, but neither do the sibling `Get`/`MustGet`/`GetRegistry` in the same file, so this matches the file's existing style.

## Missing Tests
- **[realm-as-owner allowance path is untested]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:59` — realm-path `Approve` and `TransferFrom` are not exercised; only realm-path `Transfer` is.
  <details><summary>details</summary>

  `TestRegistryWrappers` covers user-path `Transfer`/`Approve`/`TransferFrom` and realm-path `Transfer`, plus the insufficient-balance abort. It never sets a realm as the allowance owner, so the branch where `cur.Previous()` is a realm and drives `Approve` then `TransferFrom` is uncovered. The path shares the same `CallerTeller` resolution as the tested cases, so risk is low, but a realm approving a spender that later pulls via `TransferFrom` is the concrete adopter scenario the PR description names (`gov/dao/v3/treasury`). Ready-to-add, assuming the realm holds balance:

  ```gno
  // realm as allowance owner: realm approves carol, carol pulls via TransferFrom
  testing.SetOriginCaller(alice)
  testing.SetRealm(testing.NewCodeRealm(realmCaller))
  Approve(cross(cur), tokenKey, carol, 100)
  uassert.Equal(t, int64(100), Allowance(tokenKey, realmAddr, carol))

  testing.SetRealm(testing.NewUserRealm(carol))
  TransferFrom(cross(cur), tokenKey, realmAddr, bob, 40)
  uassert.Equal(t, int64(60), Allowance(tokenKey, realmAddr, carol))
  ```
  </details>

## Suggestions
None.

## Verified
- Ran the full grc20reg suite green at 006761627, including the new `TestRegistryWrappers`: `gno test -v ./gno.land/r/demo/defi/grc20reg/` from `examples/`.
- Traced caller resolution against the live run: with `cur` = grc20reg's own frame, `CallerTeller().accountFn` returns `cur.Previous().Address()`, so a realm cross-call debits the calling realm and leaves the user untouched. Test confirms: after the realm-path `Transfer(50)`, `realmAddr` drops 200→150 while `alice` stays at 700.
- Confirmed the `Register` hardening is load-bearing: removing the added `assertCurrent(cur)` lets `cur.Previous().PkgPath()` be read from a frame not asserted current, which is the registry-key spoofing gap the assertion closes. Matches the `cur.IsCurrent()`-before-`cur.Previous()` standard.
- Walked the invariant catalog; the only class the diff touches is caller/access control, and it authenticates through `cur.Previous()` (via the teller), not the stack-walking `unsafe` primitives.
- CI: the red `main / test` shard fails on `testdata/storage_deposit_price_change.txtar:37` (a coins-balance match in `gno.land/pkg/integration` against `r/test/storage`), a package this PR does not touch; the parallel `main / test` shard is green. Unrelated to this change.

## Open questions
- The write wrappers assert `cur.IsCurrent()` and then the `fnTeller` methods re-assert `rlm.IsCurrent()` on the same `cur`; the wrapper-level assertion is redundant for those three paths (it remains the sole guard for `Register`). Left as-is: cheap defense-in-depth and keeps the guard uniform across every exported entry point. Not posted.
