# PR [#5944](https://github.com/gnolang/gno/pull/5944): feat(gnogenesis): add -skip-signature-check to genesis verify

URL: https://github.com/gnolang/gno/pull/5944
Author: aeddi | Base: master | Files: 2 | +172 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 859dcb88c (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5944 859dcb88c`

**TL;DR:** `gnogenesis verify` gets a `-skip-signature-check` flag so it can validate genesis files whose tx signatures intentionally don't verify: admin caller-overrides patched after signing, and valoper-seed placeholder signatures with no key material. The same commit fixes a nil-pointer panic when a tx signature carries no public key.

**Verdict: APPROVE** — correct, well-tested, no blockers. One optional regression-guard test suggested.

## Summary
Two commits. The first adds a `signer.PubKey == nil` guard before `VerifyBytes`, turning a nil-pointer panic into an `invalid tx signature` error for txs whose signatures carry no public key. The second adds the opt-in `-skip-signature-check` flag, which `continue`s past the per-tx signature loop only; `ValidateBasic`, `ValidateGenState`, balance validation, and the hardfork valoper-coverage check all still run. This mirrors the node's `--skip-genesis-sig-verification`, which already lets these genesis txs through at boot.

## Glossary
- **valoper-seed** — `gnogenesis fork valoper-seed`, an offline generator that emits `valopers.Register` txs; it has no key material, so it fills placeholder (zero-value) signatures.
- **caller-override** — an admin-gated genesis call signed with a build key, then patched to set `msg.caller` to the admin address; the signature no longer matches the tx body by design.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- **[flag could silently widen past signatures]** [`contribs/gnogenesis/internal/verify/verify.go:96-98`](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/verify/verify.go#L96-L98) · [↗](../../../../../.worktrees/gno-review-5944/contribs/gnogenesis/internal/verify/verify.go#L96) — no test asserts a non-signature check still fires with `-skip-signature-check` set.
  <details><summary>details</summary>

  The two skip subtests only prove the signature check is bypassed; none proves a later check still runs, which is the flag's whole contract ("every other check still runs"). Since the flag disables signature verification, a future refactor that turned the `continue` into a broader short-circuit is exactly the dangerous direction, and the current tests wouldn't catch it. A subtest that sets the flag on a hardfork genesis with an uncovered validator still gets `errUncoveredGenesisValidator`: verified green as-is, and red when the `continue` is changed to `return nil`. Fix: add the guard subtest in [`tests/skip_check_runs_later_checks.go.txt`](tests/skip_check_runs_later_checks.go.txt).
  </details>

## Suggestions
None.

## Verified
- Revert-proof of the panic fix: removing the `signer.PubKey == nil` guard makes the `missing signer public key` subtest panic with `runtime error: invalid memory address or nil pointer dereference` at the `VerifyBytes` call ([`verify.go:123`](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/verify/verify.go#L123) · [↗](../../../../../.worktrees/gno-review-5944/contribs/gnogenesis/internal/verify/verify.go#L123)); with the guard it returns `errInvalidTxSignature`.
- The panic path is reachable from real producers: [`buildRegisterTx` at valoper_seed.go:372](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/fork/valoper_seed.go#L372) · [↗](../../../../../.worktrees/gno-review-5944/contribs/gnogenesis/internal/fork/valoper_seed.go#L372) and [addpkg.go:124](https://github.com/gnolang/gno/blob/859dcb88c/contribs/gnogenesis/internal/fork/addpkg.go#L124) · [↗](../../../../../.worktrees/gno-review-5944/contribs/gnogenesis/internal/fork/addpkg.go#L124) both emit `make([]std.Signature, len(...))`, i.e. zero-value signatures with a nil `PubKey`.
- The `GetSignatures()[0]` access and the new guard rest on `ValidateBasic`: it rejects `len(Signatures) == 0` and `len != len(GetSigners())` ([`tm2/pkg/std/tx.go:53-58`](https://github.com/gnolang/gno/blob/859dcb88c/tm2/pkg/std/tx.go#L53-L58) · [↗](../../../../../.worktrees/gno-review-5944/tm2/pkg/std/tx.go#L53)), so index 0 can't panic and a zero-value placeholder slice passes both checks and reaches the guard.
- Suggested guard subtest run green with the flag and red when the `continue` is replaced by `return nil` (coverage check skipped).
- All `TestGenesis_Verify` subtests green at 859dcb88c (`go test ./internal/verify/` in `contribs/gnogenesis`).

## Open questions
- Flag name `-skip-signature-check` differs from the node's `--skip-genesis-sig-verification`. The help text cross-references the node flag, so discoverability is covered; not worth a change. Not posted.
