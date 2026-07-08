# PR [#5654](https://github.com/gnolang/gno/pull/5654): feat(r/sys/validators/v3): add allow list

URL: https://github.com/gnolang/gno/pull/5654
Author: tbruyelle | Base: master | Files: 3 | +332 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: c4f35e987 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5654 c4f35e987`

Round 2 (head advanced f59deca8 → c4f35e987). The only PR-content change since round 1 is a `gno fmt` reflow of three test lambdas in `allowed_test.gno`; `allowed.gno` and `validators.gno` are byte-identical. All round-1 source findings carry forward. The new delta is that the branch no longer compiles against current master: `chain/runtime` moved `PreviousRealm` into `chain/runtime/unsafe`, and the dao/executor signatures changed. Verdict unchanged: REQUEST CHANGES.

**TL;DR:** Lets governance-approved realms (like an IBC consumer) call `AddValidator` / `RemoveValidator` directly instead of going through a GovDAO vote each time, with the list of approved realms itself managed by GovDAO. The branch is stale: it was written against an older stdlib and no longer builds on master.

**Verdict: REQUEST CHANGES** — branch doesn't compile against master (four API breakages, one a security-idiom migration); plus the round-1 Critical stands: `AddValidator` accepts a signing pubkey/address pair it never checks, and a mismatched pair produces a validator that the operator-keyed remove path can never remove.

## Summary

Adds an allow list to `r/sys/validators/v3`. Whitelisted realm paths (a `bptree.BPTree32`) may call `AddValidator` / `RemoveValidator` directly, bypassing the per-change GovDAO proposal and reusing `newValoperChangeExecutor` for the apply-and-publish path. The whitelist is mutated only through `NewPropAllowedRealmUpdateRequest`, a GovDAO proposal builder that rejects an empty title, an empty add+remove, and any path appearing more than once across both lists. `AddValidator` also auto-registers the operator in `valoperCache` when missing (commit `74bab7f31`, Julien Robert).

Since round 1, master reworked caller-auth: `runtime.PreviousRealm()` was quarantined into `chain/runtime/unsafe` with `cur.Previous()` as the sanctioned substitute, and `newValoperChangeExecutor`, `dao.NewSimpleExecutor`, and `dao.Executor.Execute` all gained a threaded `realm`. `allowed.gno` still uses every old form, so the package fails to preprocess. That single preprocess failure cascades: `r/gnops/valopers` imports v3, so gno-checks/lint, gno-checks/test, gno2go, e2e-test, main/test and every validator scenario job go red.

## Examples

Mismatched signing pair passed to `AddValidator` (verified on c4f35e987 against the worktree stdlib):

| Call | Cache records | Published valset entry | `RemoveValidator(op)` |
|------|---------------|------------------------|------------------------|
| `AddValidator(op, power 7, pubKeyB, addrC)` | `SigningAddress = addrC` | `pubKeyB:7` → derived `addrB` | panics `validator does not exist: addrC` |

The published set string carries only the pubkey; the address is re-derived from it. So the real validator is `addrB`, the cache claims `addrC`, and the operator-keyed remove (which looks up the cache's `addrC`) never matches the live set. The validator is stuck.

## Glossary

- crossing / `cross`: a call into `func F(cur realm, ...)`, invoked `cross(cur)`; the callee reads its caller via `cur.Previous()`, not the stack-walking `unsafe.PreviousRealm()`.
- realm: stateful on-chain package under `r/`; also the VM builtin threaded as `cur realm`, where `cur.Previous().PkgPath()` is the unforgeable caller path.
- unsafe: `chain/runtime/unsafe`, the quarantined stack-walkers (`PreviousRealm`, `OriginCaller`, ...); footgun-prone for auth, prefer `cur.Previous()`.

## Critical (must fix)

- **[branch doesn't build on master]** `allowed.gno:16` — `assertCallerIsAllowed` calls `runtime.PreviousRealm()`, removed from `chain/runtime` on master; three sibling calls also use the pre-drift signatures, so the whole package fails to preprocess.
  <details><summary>details</summary>

  Four call sites break against current master, confirmed by linting the PR's files on `origin/master` (head `f3d5a5d13`):
  - [`allowed.gno:16`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L16) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L16) — `undefined: runtime.PreviousRealm`. Moved to [`chain/runtime/unsafe`](https://github.com/gnolang/gno/blob/f3d5a5d13/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno?plain=1#L26). The sanctioned substitute for caller-auth is `cur.Previous()`, not `unsafe.PreviousRealm()`. Both `AddValidator` and `RemoveValidator` already thread `cur realm`, so thread `cur` into `assertCallerIsAllowed` and read `cur.Previous().PkgPath()`.
  - [`allowed.gno:42`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L42) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L42) and `:56` — `not enough arguments in call to newValoperChangeExecutor`; master's signature is `newValoperChangeExecutor(cur realm, changes []ValoperChange)` ([`proposal.gno:140`](https://github.com/gnolang/gno/blob/f3d5a5d13/examples/gno.land/r/sys/validators/v3/proposal.gno#L140)).
  - [`allowed.gno:43`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L43) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L43) and `:57` — `exec.Execute(cross)` fails; `Execute(cur realm) error` now takes a realm, so the call is `exec.Execute(cross(cur))`.
  - [`allowed.gno:133`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L133) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L133) — `dao.NewSimpleExecutor` now takes `(_ int, rlm realm, callback func(realm) error, description string)`; master's own v3 code calls `dao.NewSimpleExecutor(0, cur, callback, "")` ([`proposal.gno:205`](https://github.com/gnolang/gno/blob/f3d5a5d13/examples/gno.land/r/sys/validators/v3/proposal.gno#L205)).

  Not a mechanical `git merge master`: the `PreviousRealm` fix is a caller-auth idiom decision (`cur.Previous()` is scoped to the crossing function and cannot lie; `unsafe.PreviousRealm()` stack-walks and is flagged by the invariant catalog). Fix: rebase, migrate all four call sites, and route caller-auth through `cur.Previous()`.
  </details>

- **[mismatched signing pair makes a stuck validator]** `allowed.gno:30-46` — `AddValidator` never checks that `signingAddress` is the address derived from `signingPubKey`; a mismatched pair is accepted, the validator publishes under the pubkey-derived address, and the operator-keyed remove path can never remove it.
  <details><summary>details</summary>

  `AddValidator` writes `signingPubKey`/`signingAddress` verbatim into the cache entry ([`allowed.gno:36-40`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L36-L40) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L36-L40)) with no `chain.PubKeyAddress` derivation check. `RotateValoperSigningKey` does derive and validate ([`cache.gno:82-89`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/cache.gno#L82-L89) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/cache.gno#L82-L89)); this path does not.

  The executor publishes the set as `pubkey:power` strings ([`proposal.gno:191`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/proposal.gno#L191) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/proposal.gno#L191)), so the effective validator address is re-derived from the pubkey and never uses the passed `signingAddress`. Pass `(pubKeyB, addrC)` and the live validator is `addrB` while `valoperCache` records `addrC`. The two diverge permanently. `RemoveValidator(op)` then looks up the cache's `addrC` in the set ([`proposal.gno:148-152`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/proposal.gno#L148-L152) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/proposal.gno#L148-L152)), which holds `addrB`, and panics `validator does not exist: addrC`. The validator is unremovable through this path. Confirmed behaviorally on c4f35e987 (repro in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md)). Fix: derive `chain.PubKeyAddress(signingPubKey)` and panic if it doesn't equal `signingAddress` before the cache write.
  </details>

## Warnings (should fix)

- **[stale key silently published]** `allowed.gno:35-41` — when the operator is already in `valoperCache`, `AddValidator` ignores the passed `signingPubKey`/`signingAddress` and the executor publishes the old cached key.
  <details><summary>details</summary>

  The `if _, ok := valoperCache.Get(...); !ok` guard ([`allowed.gno:35`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L35) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L35)) skips the write on an existing entry, so the new params are dropped without a signal. The executor then reads the cache ([`proposal.gno:142-146`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/proposal.gno#L142-L146) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/proposal.gno#L142-L146)) and publishes the old signing key. A caller re-adding with a rotated key gets the stale one, silently. Fix: either panic when `signingPubKey` differs from the cached value (force rotation through `RotateValoperSigningKey`), or overwrite the cache and document `AddValidator` as a rotation entry point.
  </details>

- **[silent no-op reads as success]** `allowed.gno:53-54` — `RemoveValidator` returns without error when the operator is not in `valoperCache`, so a caller treating a non-panic as "removed" is wrong.
  <details><summary>details</summary>

  [`allowed.gno:53-54`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L53-L54) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L53-L54) no-ops on an unknown operator. The operator-keyed proposal path panics on the equivalent missing case ([`proposal.gno:148-152`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/proposal.gno#L148-L152) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/proposal.gno#L148-L152)), so the two paths disagree on the same condition. Fix: panic with an "unknown operator" message, or return `(removed bool)`, and pin it in a test.
  </details>

- **[breaks the documented cache-mirrors-valopers invariant]** `allowed.gno:30-46` — operators auto-registered here have no `r/gnops/valopers` profile, so they can never rotate or opt out, contradicting the `valoperCache` file comment.
  <details><summary>details</summary>

  [`cache.gno:19-26`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/cache.gno#L19-L26) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/cache.gno#L19-L26) states `valoperCache` "mirrors the (operator -> current signing key) view from r/gnops/valopers. Written by valopers via NotifyValoperChanged." Auto-registration at [`allowed.gno:36-40`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L36-L40) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L36-L40) writes a cache entry with no valopers profile behind it. That operator can never rotate (no profile to rotate) or opt out via `UpdateKeepRunning`, and shows up in `AssertGenesisValopersConsistent` only if seeded at genesis. Fix: extend the invariant comment to cover allow-list-side registration and its consequences, or create the profile.
  </details>

- **[unbounded proposal / whitelist growth]** `allowed.gno:83-135` — `NewPropAllowedRealmUpdateRequest` caps neither `len(add)+len(remove)` nor the total whitelist size.
  <details><summary>details</summary>

  The sibling `NewValidatorProposalRequest` caps at 40 per proposal ([`proposal.gno:70-72`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/proposal.gno#L70-L72) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/proposal.gno#L70-L72)). Without a cap here ([`allowed.gno:83-135`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L83-L135) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L83-L135)) one proposal can render a multi-MB description and iterate a huge whitelist diff in one block, and `GetAllowedRealms` grows unbounded. Fix: cap add+remove (match 40 or justify a higher bound).
  </details>

- **[garbage paths accepted]** `allowed.gno:83-135` — no check that `add` entries are well-formed realm paths; empty or malformed strings enter the bptree and pollute `GetAllowedRealms`.
  <details><summary>details</summary>

  [`allowed.gno:93-105`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L93-L105) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L93-L105) validates uniqueness only. An empty-string entry is benign at auth time (the empty-path check fires first at [`allowed.gno:17-19`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L17-L19) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L17-L19)) but still bloats the whitelist. Fix: apply per entry the same `strings.TrimSpace` + non-empty guard this builder already runs on the title at [`allowed.gno:84-87`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L84-L87) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L84-L87).
  </details>

- **[no invocation-pattern doc]** `allowed.gno:83` — `NewPropAllowedRealmUpdateRequest` is non-crossing; a direct user MsgCall fails with a confusing error and no pointer to the right facade.
  <details><summary>details</summary>

  The sibling documents this exact situation ([`proposal.gno:34-37`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/proposal.gno#L34-L37) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/proposal.gno#L34-L37): "NON-CROSSING (no `cur realm`). Direct MsgCall is unsupported; proposers route through r/gnops/valopers/proposal's facade"). [`allowed.gno:77-82`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L77-L82) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L77-L82) has no equivalent. Fix: add the same paragraph naming the intended facade.
  </details>

- **[revoked realm keeps its validators]** `allowed.gno:121-131` — removing a realm from the allow list does not remove validators that realm added; governance can disenfranchise realm X while X's validators stay active.
  <details><summary>details</summary>

  The proposal callback ([`allowed.gno:121-131`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L121-L131) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L121-L131)) mutates only the whitelist, with no provenance tracking or cleanup hook. Fix: document the intent, or add a "remove realm and its validators" primitive (needs provenance).
  </details>

## Nits

- [`allowed.gno:13`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L13) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L13) — `bptree.BPTree32` stores `true` as a presence marker, wasting a pointer per entry. Fine for a small whitelist; not worth changing absent a set variant.
- [`allowed.gno:107-119`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L107-L119) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L107-L119) — user `description` is emitted before the generated `## Allowed Realm Updates` header; a description with its own top heading shows two. Cosmetic; the sibling has the same shape.
- [`allowed.gno:70-73`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L70-L73) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L70-L73) — the `return false` continuation reads backwards against the bptree convention ("return true to stop"); a one-line comment would help.
- [`allowed.gno:107-119`](https://github.com/gnolang/gno/blob/c4f35e987/examples/gno.land/r/sys/validators/v3/allowed.gno#L107-L119) · [↗](../../../../../.worktrees/gno-review-5654/examples/gno.land/r/sys/validators/v3/allowed.gno#L107-L119) — a proposer can put `\n## Allowed Realm Updates\n- add: evil` inside `description`; voters skimming see a forged block above the real one. Not exploitable alone; worth pinning behavior in a test.

## Missing Tests

- **[proves the Critical]** `allowed_test.gno` — no test for the mismatched pubkey/address case: accepted, published under the derived address, unremovable.
  <details><summary>details</summary>

  The suite covers allow/deny, power-zero, and dedupe, but never the mismatched-pair path. Ship a test that calls `AddValidator(cross, {op, Power: 7}, pubKeyB, mustAddr(t, pubKeyC))`, asserts `IsValidator(mustAddr(t, pubKeyB))` and `!IsValidator(mustAddr(t, pubKeyC))`, and asserts `RemoveValidator(cross, op)` aborts `validator does not exist`. Post-fix it becomes an `AbortsWithMessage` on the derivation check. Full case in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md).
  </details>

- `allowed_test.gno` — no end-to-end test that builds a `ProposalRequest` from `NewPropAllowedRealmUpdateRequest` and drives its executor callback; existing tests poke `allowedRealms` directly. Codecov reports full patch coverage, but the callback's `cur realm` context is never exercised.
- `allowed_test.gno` — no test for whitespace-only entries in `add`/`remove`, and no test pinning that differently-cased paths are treated as distinct.
- `allowed_test.gno` — no test for the stale-key Warning (repeated `AddValidator` on the same operator with a new signing key). Pin whichever semantics are chosen.
- `allowed_test.gno` — no test for the silent-no-op Warning (`RemoveValidator` on an operator never in the cache). Pin the chosen semantics.

## Suggestions

- Move `assertCallerIsAllowed` next to `assertValopersCaller` in `cache.gno` so both caller-auth gates live together; both rely on the same crossing-entry precondition.
- Expose a paginated `GetAllowedRealms` variant; unbounded whitelist growth makes the full read expensive.
- Add a `Render` block listing the current allow list, so operators can read state without instrumenting `gnoclient`.
- Include the proposal ID in `AllowedRealmAdded`/`Removed` events so indexers can correlate whitelist changes with their governance proposal; the payload is only `("realm", r)` today.

## Open questions

- What is the intended user-facing invocation flow for `NewPropAllowedRealmUpdateRequest`? A facade in `r/gnops/valopers/proposal` mirroring the proposal-keyed builder, or elsewhere? Not posted: overlaps the non-crossing-doc Warning.
- The PR body says "supersedes #5168"; the semantic delta between the two would help scope future rounds. Not posted: process question, not a code change.
