# PR [#5880](https://github.com/gnolang/gno/pull/5880): docs: add concise AI contract review guide

URL: https://github.com/gnolang/gno/pull/5880
Author: moul | Base: master | Files: 1 | +115 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 (high) | Commit: 26ca914e2 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5880 26ca914e2`

**TL;DR:** Adds `docs/resources/gno-ai-contract-review.md`, a 115-line checklist telling an AI agent the seven highest-yield security bugs to look for when reviewing Gno realm code. It is a short, copy-paste distillation of the longer `gno-security-guide.md`, meant to be applied inline without running tooling.

**Verdict: APPROVE** — every one of the seven checks is a faithful, technically-correct distillation of `gno-security-guide.md` §5, and all cited symbols (`IsCurrent`, `Previous`, `IsUserCall`, `grc20.IsCanonicalTeller`) exist and behave as shown. Two doc-accuracy nits: the Check 6 panic timing is mislabeled, and the `IsCurrent()` guard that Check 1 prescribes is dropped from Check 6's example. One merge-order note only (the harness path lands with #5835), folded into the comment Body rather than posted inline.

## Summary
The file maps one-to-one onto §5.1-5.7 of `gno-security-guide.md`: caller identity via `cur realm` (§5.6), `IsUserCall()` for payment guards (§5.5), no exported pointers to mutable state (§5.1), no caller-supplied callbacks under realm authority (§5.3), canonical-type assertion on interface params (§5.4), no stored `realm` values (§5.7), and unexported `/p/`-embedded callback iterators (§5.2). I verified each WRONG/RIGHT pair against the VM and the source guide. The security substance is correct. Findings are documentation polish, not security errors.

## Glossary
- crossing function: `func F(cur realm, ...)`; identifies its caller through `cur.Previous()`.
- ephemeral realm: short-lived code realm a `maketx run` executes under; `IsUser()` accepts it, `IsUserCall()` does not.
- readonly taint: foreign-read values carry a sticky read-only bit; writes panic.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[panic fires later than the comment says]** [`gno-ai-contract-review.md:74-77`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L74-L77) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L74) — the `// WRONG: panics at attach time` comment on the bare `var savedRealm realm` is doubly imprecise.
  <details><summary>details</summary>

  A bare package-level `var savedRealm realm` with nothing stored into it runs clean, it never panics. The panic fires only when a live realm value is assigned, and it fires at transaction finalize, not "attach time". The persistence check trips on the realm's backing `*HeapItemValue` when the object graph is finalized, matching the source guide's own wording ("panics at attachment time or transaction finalize", [`gno-security-guide.md:319-325`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L319-L325)) and the committed filetest [`zrealm_cur_persist_struct.gno`](https://github.com/gnolang/gno/blob/26ca914e2/gnovm/tests/files/zrealm_cur_persist_struct.gno) (assignment succeeds, `// Error:` at finalize). Fix: move the WRONG comment onto an assignment line and say "finalize", e.g. `func Save(cur realm) { savedRealm = cur // panics at tx finalize }`.
  </details>

## Missing Tests
None (docs-only).

## Suggestions
- **[Check 1 prescribes a guard Check 6's example drops]** [`gno-ai-contract-review.md:80-82`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L80-L82) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L80) — Check 1 makes `if !cur.IsCurrent() { panic(...) }` the standard before reading caller identity, but Check 6's RIGHT example reads `cur.Previous().Address()` in `Save` without it.
  <details><summary>details</summary>

  Real realms guard caller-identity reads with `IsCurrent()` when the realm value could be stale or stashed (`examples/gno.land/r/gov/dao/v3/impl/impl.gno:51`, `examples/gno.land/r/sys/users/store.gno:223`). A reader who copies Check 6 verbatim gets an unguarded caller read, contradicting the pattern the same doc set two checks earlier. Fix: add the `IsCurrent()` guard to the `Save` example, or note in Check 1 that the guard is needed only when the realm value is caller-passed rather than the live `cur`. No security bug in the example as written; internal consistency only.
  </details>

## Open questions
- The relationship table lists `misc/audit-pattern-harness/` ([`gno-ai-contract-review.md:113`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L113)), absent from master; it ships in [#5835](https://github.com/gnolang/gno/pull/5835). Author says #5835 merges first, so the path will resolve. Not posted as an inline change request; a one-line merge-order note goes in the comment.md Body instead.
