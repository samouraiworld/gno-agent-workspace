# PR #4008: fix(gnovm): Verify prefix in `Address.IsValid()` to prevent invalid addresses

URL: https://github.com/gnolang/gno/pull/4008
Author: notJoon | Base: master | Files: 2 | +6 -2
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `32f0c57e` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4008 32f0c57e`

**Verdict: NEEDS DISCUSSION** — fix is already on master via the native `IsValid` impl + `GetFromBech32`; PR diff doesn't apply (files removed by the std-split refactor); only the test case for non-`g` prefix is still worth salvaging.

## Summary
PR was opened in March 2025 to add a `"g"` prefix check to `Address.IsValid()` after an OpenZeppelin finding that bech32-valid but non-Gno addresses (e.g. `bc1...`) silently passed validation. Since then the std-split refactor (PR #4040) moved `gnovm/stdlibs/std/crypto.gno` → `gnovm/stdlibs/chain/address.gno` and reimplemented `address.IsValid()` natively in `uverse.go`, routing through `crypto.AddressFromBech32` → `GetFromBech32`, which already rejects mismatched HRPs. The patch as-written no longer applies to the files it targets, and the behavioral fix is already live; what is *not* on master is the non-`g`-prefix regression test, which is the only piece worth carrying forward.

## Glossary
- **HRP** — human-readable prefix portion of a bech32 string (here `"g"`).
- **std-split** — refactor (PR #4040, Sep 2025) that broke `gnovm/stdlibs/std/` into `chain/`, `chain/banker/`, `chain/runtime/`, etc.; `Address` became the predeclared `address` type with a native `IsValid` method.
- **GetFromBech32** — `tm2/pkg/crypto/bech32.go` helper that decodes bech32 and enforces `hrp == prefix`.

## Fix
Before this PR's diff (on its now-stale base): [`gnovm/stdlibs/std/crypto.gno:18-21`](https://github.com/gnolang/gno/blob/32f0c57e/gnovm/stdlibs/std/crypto.gno#L18-L21) · [↗](../../../../../.worktrees/gno-review-4008/gnovm/stdlibs/std/crypto.gno#L18-L21) called `DecodeBech32(a)` and returned `ok` — any bech32-valid string was accepted regardless of HRP. The PR changes those lines to return `prefix == bech32AddrPrefix && ok` and adds one test case `bc156tlrfxxxelwrmvu0v986psjln9ry60ef34yp2 → false` at [`gnovm/stdlibs/std/crypto_test.gno:19-20`](https://github.com/gnolang/gno/blob/32f0c57e/gnovm/stdlibs/std/crypto_test.gno#L19-L20) · [↗](../../../../../.worktrees/gno-review-4008/gnovm/stdlibs/std/crypto_test.gno#L19-L20). On current master the equivalent check sits in [`tm2/pkg/crypto/bech32.go:57-59`](../../../../../gno/tm2/pkg/crypto/bech32.go#L57-L59) (`hrp != prefix` → error) reached from [`uverse.go:1001-1017`](../../../../../gno/gnovm/pkg/gnolang/uverse.go#L1001-L1017) where `defNativeMethod("address", "IsValid", ...)` calls `crypto.AddressFromBech32`. Net: behavior is correct on master without this PR; only the regression test case for a non-`g` HRP is genuinely new.

## Critical (must fix)
- **[stale — diff doesn't apply, fix already in master]** [`gnovm/stdlibs/std/crypto.gno:18-21`](https://github.com/gnolang/gno/blob/32f0c57e/gnovm/stdlibs/std/crypto.gno#L18-L21) · [↗](../../../../../.worktrees/gno-review-4008/gnovm/stdlibs/std/crypto.gno#L18-L21) — patched file was removed by the std-split refactor; the prefix check is now enforced natively.
  <details><summary>details</summary>

  PR HEAD (`32f0c57e`, last merge of master Sep 4 2025) still targets `gnovm/stdlibs/std/crypto.gno`, which no longer exists on master after PR #4040 (`feat(stdlibs)!: std split`). The `Address` type became the predeclared `address` (lowercase) with a native `IsValid` defined at [`gnovm/pkg/gnolang/uverse.go:1001-1017`](../../../../../gno/gnovm/pkg/gnolang/uverse.go#L1001-L1017), which calls `crypto.AddressFromBech32`. That in turn goes through [`tm2/pkg/crypto/bech32.go:19-26`](../../../../../gno/tm2/pkg/crypto/bech32.go#L19-L26) → [`GetFromBech32`](../../../../../gno/tm2/pkg/crypto/bech32.go#L47-L61) which checks `hrp != prefix`. I confirmed empirically by writing a chain-package test on current master that asserts `IsValid` returns false for `cosmos1...` and `bc1...` style prefixes — passes without this PR. Fix: rebase is non-trivial because the diff target is gone; recommend closing the PR (or repurposing it to add the missing regression tests, see Missing Tests below).
  </details>

## Warnings (should fix)
- **[design feedback now moot]** [`gnovm/pkg/gnolang/uverse.go:1001-1017`](../../../../../gno/gnovm/pkg/gnolang/uverse.go#L1001-L1017) — zivkovicmilos' "keep `IsValid` bech32-only, add a `Prefix` method" suggestion no longer matches the codebase shape.
  <details><summary>details</summary>

  The review thread debated whether `IsValid` should validate the prefix (moul, notJoon) or stay bech32-only with a separate `Prefix()` method (zivkovicmilos). On current master, `IsValid` is a native method that *already* enforces the prefix via `AddressFromBech32`, and the `address` type is predeclared (no exportable surface to hang a `Prefix` method off cleanly without expanding the native ABI). Any further design debate should happen against the native definition, not the gno-side helper this PR was editing. Fix: re-anchor the discussion on `uverse.go:1001` or close the design thread.
  </details>

- **[no test in tm2 crypto bech32 suite]** [`tm2/pkg/crypto/bech32_test.go:17-22`](../../../../../gno/tm2/pkg/crypto/bech32_test.go#L17-L22) — `invalidStrs` only varies the data portion; nothing in the test file asserts a wrong HRP is rejected.
  <details><summary>details</summary>

  All entries in `invalidStrs` start with `crypto.Bech32AddrPrefix() + ...`, so they exercise data-side decoding errors only. The prefix-mismatch branch at [`bech32.go:57-59`](../../../../../gno/tm2/pkg/crypto/bech32.go#L57-L59) — the load-bearing check for the original OpenZeppelin finding — has no direct regression coverage. davd-gzl flagged this on the PR thread. Fix: add `_, err := crypto.AddressFromBech32("bc1...")` / `"cosmos1..."` assertions to `bech32_test.go`. I wrote one and it passes on master (see Missing Tests for the file).
  </details>

## Nits
None.

## Missing Tests
- **[regression coverage gap, gno side]** [`gnovm/stdlibs/chain/address_test.gno:19`](../../../../../gno/gnovm/stdlibs/chain/address_test.gno#L19) — the `bc156tlrfxxxelwrmvu0v986psjln9ry60ef34yp2 → false` case from this PR was not carried into the post-split test file.
  <details><summary>details</summary>

  The std-split kept the canonical-`g` happy paths and the bech32 data-charset failure cases but dropped the non-`g` HRP case the PR explicitly added. Without it, a future regression that re-introduces the OpenZeppelin bug (e.g. someone routing `IsValid` back through a prefix-agnostic decoder) would slip past CI. Fix: add the non-`g` cases — see test file at `tests/adv_address_prefix_test.gno` in this review directory.
  </details>

- **[regression coverage gap, tm2 side]** [`tm2/pkg/crypto/bech32_test.go`](../../../../../gno/tm2/pkg/crypto/bech32_test.go) — no test asserts that `AddressFromBech32` rejects a non-`g` HRP.
  <details><summary>details</summary>

  `IsValid`'s native impl depends on this rejection; the assertion belongs in this file. See `tests/adv_bech32_prefix_test.go` in this review directory — verified passing on master.
  </details>

## Suggestions
- `PR thread` — close PR #4008 and open a small follow-up that adds only the two missing regression tests (chain + tm2/crypto).
  <details><summary>details</summary>

  The PR's behavioral fix is already in master; rebasing onto the std-split is more churn than just porting the two test cases. A clean test-only PR is easier to review and lets this thread close with credit to the original finding.
  </details>

## Questions for Author
- Are you willing to close this and submit a test-only follow-up (chain `address_test.gno` + `tm2/pkg/crypto/bech32_test.go`), given the behavioral fix is now live via the native `IsValid`?
- Was the `IsGnoAddress` / `Prefix()` direction (thehowl / leohhhn / zivkovicmilos) ever escalated to an issue or ADR? If yes, link it; if not, the design thread should restart against `uverse.go:1001`, not this diff.
