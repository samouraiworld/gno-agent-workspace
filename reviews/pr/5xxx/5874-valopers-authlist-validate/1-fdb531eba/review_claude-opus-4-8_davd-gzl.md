# PR [#5874](https://github.com/gnolang/gno/pull/5874): fix(valopers): validate auth-list members, sanitize description, reject negative min fee

URL: https://github.com/gnolang/gno/pull/5874
Author: davd-gzl | Base: master | Files: 5 | +87 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: fdb531eba (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5874 fdb531eba`

**TL;DR:** Three input-validation hardenings in the `r/gnops/valopers` realm: reject malformed bech32 addresses before they reach the auth list, run the operator-supplied profile description through a markdown sanitizer before rendering it, and reject a negative minimum-fee value before it is cast to an unsigned integer.

**Verdict: APPROVE** — three self-contained hardenings, each with a matching edge-case test; no auth, payment, or state-write regression found.

## Summary
Three operator-controlled inputs previously reached state or rendering without full validation. `AddToAuthList`/`DeleteFromAuthList` now run `validateBech32(member)` up front, so empty or bad-checksum member addresses panic with `ErrInvalidAddress` instead of landing in the auth list. `Valoper.Render` now pipes the operator-controlled `Description` through `sanitize.Block`, neutralizing structural markdown injection (forged headings, HTML blocks, link-reference definitions) that would otherwise break into the profile page or the govdao proposal that embeds `Render`. `ProposeNewMinFeeProposalRequest` rejects a negative `newMinFee` before the `int64`→`uint64` cast that would otherwise wrap it to a near-`MaxUint64` register fee.

## Examples
Observed `Valoper.Render()` output by `Description` input (newlines shown as `\n`), on fdb531eba:

| Description input | Rendered body after the `## Moniker` heading |
|---|---|
| `hello world` | `## Mon\n\nhello world\n\n- Operator Address:` |
| `# FORGEDHEADING\nbody` | `## Mon\n\n\# FORGEDHEADING\nbody\n\n- Operator Address:` |
| `[a]: https://x.example` (sanitizes to empty) | `## Mon\n- Operator Address:` |
| `""` (empty) | `## Mon\n- Operator Address:` |

The forged `#` is escaped to `\#` and renders as literal text, not a heading. The sanitize-to-empty and empty cases both terminate the heading line with a single `\n`, so the operator-address chrome never merges into the heading.

## Glossary
- ephemeral realm: a `maketx run` code realm; `IsUser()` accepts it, `IsUserCall()` does not. Not in play here — valopers gates on `unsafe.OriginCaller()==addr`, not `IsUser()`.
- realm: the `cur realm` threaded parameter; `cur.Previous().Address()` is the unforgeable caller identity used by the auth list.

## Fix
`AddToAuthList`/`DeleteFromAuthList` prepend a `validateBech32(member)` guard at [`valopers.gno:92`](https://github.com/gnolang/gno/blob/fdb531eba/examples/gno.land/r/gnops/valopers/valopers.gno#L92) · [↗](../../../../../.worktrees/gno-review-5874/examples/gno.land/r/gnops/valopers/valopers.gno#L92) and [`valopers.gno:102`](https://github.com/gnolang/gno/blob/fdb531eba/examples/gno.land/r/gnops/valopers/valopers.gno#L102) · [↗](../../../../../.worktrees/gno-review-5874/examples/gno.land/r/gnops/valopers/valopers.gno#L102). `Valoper.Render` keys the description branch on the sanitized result at [`valopers.gno:472-476`](https://github.com/gnolang/gno/blob/fdb531eba/examples/gno.land/r/gnops/valopers/valopers.gno#L472-L476) · [↗](../../../../../.worktrees/gno-review-5874/examples/gno.land/r/gnops/valopers/valopers.gno#L472), emitting a bare `\n` when the body sanitizes to empty so the heading line stays terminated. `ProposeNewMinFeeProposalRequest` panics on a negative value at [`proposal.gno:92-94`](https://github.com/gnolang/gno/blob/fdb531eba/examples/gno.land/r/gnops/valopers/proposal/proposal.gno#L92-L94) · [↗](../../../../../.worktrees/gno-review-5874/examples/gno.land/r/gnops/valopers/proposal/proposal.gno#L92) before the `uint64(newMinFee)` cast.

## Invariant catalog
- Determinism: `sanitize.Block` is a pure O(n) transform; auth list is a bptree; no map iteration, wall clock, or randomness. Clean.
- Realm state safety: every new guard panics before any state write (`validateBech32` before `GetByAddr`/`v.Auth().AddToAuthList`; the negative check before `NewSysParamUint64PropRequest`). No partial write, no re-entrancy, `Render` is read-only. Clean.
- Caller & access control: the auth checks (`rlm.IsCurrent()` + `a.OwnedBy(rlm.Previous().Address())`) live in `authorizable.AddToAuthList`/`DeleteFromAuthList` and are unchanged; the PR only prepends input validation. Verified the unchanged success/unauthorized paths still pass (`TestValopers_UpdateAuthMembers`). Clean; ordering note below.
- Coin & banker: the negative-min-fee rejection removes the one `int64`→`uint64` wrap path; the `Register`/`UpdateSigningKey` fee guards are untouched. No overflow/underflow introduced. Clean.
- Global mutable state: no new package-level mutable var; `sanitize`'s charset bitmaps are `init()`-set and read-only. Clean.
- Error & panic handling: `.gno` panics for user-facing rejection, sentinel `errors.New` values. Clean.
- Gas / Storage deposit / VM-fault / VM-semantics / Type-check: not touched (realm-level `.gno` change, no VM internals; new `sanitize/v0` import resolves and CI lint is green).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. The added tests cover the load-bearing edges: empty and bad-checksum member addresses for both auth-list entry points, a forged-heading description that must survive as prose but not as a live heading, a description that sanitizes to empty (lone link-reference definition) that must keep the heading terminated, and a negative min fee. The `z_2_filetest.gno` golden was updated for the new blank-line envelope.

## Suggestions
None.

## Open questions
- `validateBech32(member)` runs before the auth check inside `authorizable`, so an unauthorized caller who passes a malformed member address gets `ErrInvalidAddress` rather than `ErrNotSuperuser`. No state is written on either path and the address is caller-supplied, so nothing sensitive leaks; the reordering is cosmetic. Not worth a posted comment.
