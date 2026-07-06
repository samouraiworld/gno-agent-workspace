# PR [#5880](https://github.com/gnolang/gno/pull/5880): docs: add concise AI contract review guide

URL: https://github.com/gnolang/gno/pull/5880
Author: moul | Base: master | Files: 1 | +115 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 (high) | Commit: 26ca914e2 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5880 26ca914e2`

**TL;DR:** Adds `docs/resources/gno-ai-contract-review.md`, a 115-line checklist telling an AI agent the seven highest-yield security bugs to look for when reviewing Gno realm code. It is a short, copy-paste distillation of the longer `gno-security-guide.md`, meant to be applied inline without running tooling.

**Verdict: REQUEST CHANGES** — the seven Quick Checks are directionally correct and each verified against the VM, but the Review Checklist contradicts its own checks twice (line 97 greenlights Check 4's WRONG example, line 101 flags every grc20 token realm as vulnerable) and Check 4's WRONG example does not exhibit the bug it warns about. A doc whose whole value is precise, mechanically-applied checks misfires in these three demonstrable ways; all are small wording fixes.

## Summary
The file maps one-to-one onto §5.1-5.7 of `gno-security-guide.md`: caller identity via `cur realm` (§5.6), `IsUserCall()` for payment guards (§5.5), no exported pointers to mutable state (§5.1), no caller-supplied callbacks under realm authority (§5.3), canonical-type assertion on interface params (§5.4), no stored `realm` values (§5.7), and unexported `/p/`-embedded callback iterators (§5.2). I ran every WRONG/RIGHT pair against the VM at 26ca914e2: all seven checks are correct in direction. The defects are in the examples and the Review Checklist, which an AI reader copies verbatim: the checklist restates two checks more narrowly or more broadly than the checks themselves, and Check 4's WRONG snippet cannot actually write victim state.

## Round note
Round-1 verdict was APPROVE with two findings. This deep multi-lens pass (red-team / blue-team / correctness plus critic and claim-verification gates) confirms both round-1 findings and adds three Warnings and further Nits, and downgrades the verdict to REQUEST CHANGES. Head unchanged (26ca914e2); same round directory.

## Glossary
- crossing function: `func F(cur realm, ...)`; identifies its caller through `cur.Previous()`.
- borrow rule #1: a method declared in realm `/r/V` runs with `m.Realm` borrowed to `/r/V`, so its body writes to `/r/V` state regardless of who called it.
- ephemeral realm: short-lived code realm a `maketx run` executes under; `IsUser()` accepts it, `IsUserCall()` does not.
- canonical-type assert: a nominal type assertion (`IsCanonicalTeller`) that only the declaring package can satisfy; embedding cannot forge it.

## Critical (must fix)
None.

## Warnings (should fix)

- **[the WRONG example can't do the harm the comment claims]** [`gno-ai-contract-review.md:48-56`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L48-L56) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L48) — Check 4's WRONG example `func ApplyHook(fn func()) { fn() }` with the comment "inherits the caller's `m.Realm` and can write to your state" does not exhibit the vulnerability.
  <details><summary>details</summary>

  A bare `func()` supplied by `/p/` code cannot write victim state: it receives no pointer, cannot name any `/r/V` symbol (`/p/`→`/r/` import is a hard preprocess panic, [`preprocess.go:5450`](https://github.com/gnolang/gno/blob/26ca914e2/gnovm/pkg/gnolang/preprocess.go#L5450)), cannot construct a banker (`NewBanker` needs a `realm` handle it doesn't hold, [`banker.gno:89-104`](https://github.com/gnolang/gno/blob/26ca914e2/gnovm/stdlibs/chain/banker/banker.gno#L89-L104)), and cannot spoof victim events (`Emit` keys off the calling frame's package, [`emit_event.go:68`](https://github.com/gnolang/gno/blob/26ca914e2/gnovm/stdlibs/chain/emit_event.go#L68)). The real laundering vector needs the victim to pass a writable pointer to a callback whose parameter type `/p/` code can satisfy; a `/p/`-typed pointer callback succeeds where an `/r/`-declared one is blocked by readonly taint. Worse, the RIGHT example (`func ApplyHook(fn func(*MyState)) { fn(gState) }`) hands out a state pointer the WRONG one never did, so the pair reads as "adding a state pointer is the fix," backwards. The comment also misattributes authority: `m.Realm` stays at *your* realm for the callback ([`gno-security-guide.md:130-137`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L130-L137)), it is not "the caller's". Fix: make WRONG a `/p/`-typed-pointer callback, e.g. `func ApplyHook(fn func(*somelib.Node)) { fn(gNode) }`, keep RIGHT with an `/r/`-declared parameter type, and reword the comment to "runs under your realm's authority."
  </details>

- **[checklist line greenlights the doc's own WRONG example]** [`gno-ai-contract-review.md:97`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L97) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L97) — line 97 flags only a callback "with a `/p/`-typed parameter", but Check 4's WRONG example is `func()` with no parameter, so an AI walking the checklist passes the exact code the doc calls WRONG two screens up.
  <details><summary>details</summary>

  The source is explicit that the signature filter is wrong: "Even `func()` is dangerous" ([`gno-security-guide.md:246`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L246)). The danger keys on the *supplied value* being a top-level `/p/` FuncDecl, not on the parameter type; the defense is retyping the callback parameter with an `/r/`-declared type. Line 97 inverts that into "flag only `/p/`-typed params" and adds "No method" while the example is a plain function. Fix: widen to "no method invokes a caller-supplied func or interface value whose signature `/p/` code can satisfy, including plain `func()`."
  </details>

- **[checklist line flags every grc20 token realm]** [`gno-ai-contract-review.md:101`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L101) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L101) — "Data types holding sensitive state are declared in this realm (`/r/`), not in shared `/p/`", stated unconditionally, marks the source guide's own blessed pattern as a bug.
  <details><summary>details</summary>

  A token realm holds its balances in a `/p/`-declared `grc20.Token` — exactly "sensitive state" in a `/p/` type ([`wugnot.gno:16-17`](https://github.com/gnolang/gno/blob/26ca914e2/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L16-L17)). The source blesses this: grc20 "violates (A) ... but compensates with airtight encapsulation" ([`gno-security-guide.md:176-178`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L176-L178)) and the escape clause "if using `/p/`-declared types (e.g. `grc20.Token`), they're stored in **unexported** package vars" ([`gno-security-guide.md:412`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L412)) is dropped. Line 101 is also the only checklist line with no matching Quick Check, so no example tempers the false positive. An AI applying it flags essentially every token realm on chain. Fix: append the guide's qualifier (unexported storage, no leaked pointers, canonical asserts).
  </details>

- **[Check 3 rule drops slices and maps, the source's own primary example]** [`gno-ai-contract-review.md:96`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L96) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L96) — checklist line 96 and Check 3 both say only "pointer", so a returned internal slice or map header sails through.
  <details><summary>details</summary>

  The guide's anti-pattern 5.1 example is a slice return (`func Users() []*User { return users }`) and its rule text is "Any pointer (slice header, map, struct pointer) returned by a getter" ([`gno-security-guide.md:207-213`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L207-L213)). Check 3's example (lines 37-44) shows only a struct pointer and line 96 says only "pointer", so an AI passes `func Tags() []string` or `func Registry() map[string]*User`. Fix: "No exported function returns a pointer, slice, or map aliasing internal mutable state."
  </details>

- **[relationship table routes the reader to the outdated interrealm doc for a symbol it lacks]** [`gno-ai-contract-review.md:111`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L111) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L111) — the row credits `gno-interrealm.md` with "`cur realm`, `IsCurrent()`, borrow rules", but that file never mentions `IsCurrent()` and self-marks as outdated.
  <details><summary>details</summary>

  `grep -c IsCurrent gno-interrealm.md` returns 0; the symbol is defined and explained in `gno-interrealm-v2.md` (5 hits, [`gno-interrealm-v2.md:378`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-interrealm-v2.md?plain=1#L378)). The v1 header reads "This document is outdated, see `gno-interrealm-v2.md`" ([`gno-interrealm.md:3-5`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-interrealm.md?plain=1#L3-L5)). A routing doc for AI agents sends the reader to the superseded spec for the one symbol it does not contain. Fix: point the row at `gno-interrealm-v2.md`, or add a v2 row.
  </details>

## Nits
- [`gno-ai-contract-review.md:76-77`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L76-L77) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L76) — Check 6 `// WRONG: panics at attach time` on a bare `var savedRealm realm` is doubly imprecise: the bare declaration runs clean, and the panic fires on assignment of a live realm value at transaction finalize, not attach.
  <details><summary>details</summary>

  A package-level `var savedRealm realm` with nothing stored runs to completion; the persistence check trips only when a live realm value is assigned and the object graph is finalized, matching the source ("panics at attachment time or transaction finalize", [`gno-security-guide.md:319-325`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L319-L325)) and the committed filetest [`zrealm_cur_persist_var.gno`](https://github.com/gnolang/gno/blob/26ca914e2/gnovm/tests/files/zrealm_cur_persist_var.gno) (assignment succeeds, `// Error:` at finalize). Confirmed behaviorally: bare var runs clean, assignment errors at finalize with `cannot persist realm value: realm values are ephemeral and tied to a call frame`. Fix: move the WRONG comment onto an assignment line and say "finalize", e.g. `func Save(cur realm) { savedRealm = cur // panics at tx finalize }`.
  </details>

- [`gno-ai-contract-review.md:94`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L94) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L94) — checklist line 94 "call `cur.IsCurrent()`" is satisfied by a no-op `_ = cur.IsCurrent()`, which gives zero protection; the result must gate execution.
  <details><summary>details</summary>

  A stale or stashed realm value returns `IsCurrent()` = false, so the return value has to panic-gate the body, as Check 1 shows at lines 20-21. Verifying only that the code "calls IsCurrent()" passes unguarded code. Fix: reword to "panic unless `cur.IsCurrent()`."
  </details>

- [`gno-ai-contract-review.md:30`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L30) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L30) — Check 2's WRONG `if !IsUser()` calls a free function that does not exist; `IsUser` is a method on a realm value ([`uverse.go:131`](https://github.com/gnolang/gno/blob/26ca914e2/gnovm/pkg/gnolang/uverse.go#L131)), so the snippet is not valid gno and breaks parallelism with the RIGHT line's `cur.Previous().IsUserCall()`. Fix: `if !cur.Previous().IsUser()`.

- [`gno-ai-contract-review.md:61`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L61) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L61) — Check 5's WRONG comment "`Evil{Teller}` embedding bypasses interface checks" is imprecise: embedding bypasses seal/marker checks, not the canonical-type assert, which is nominal and is precisely the RIGHT fix on the next lines.
  <details><summary>details</summary>

  The guide's claim is that embedding "passes any seal/marker check via the embedded methods" ([`gno-security-guide.md:263-265`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L263-L265)); it does not bypass `IsCanonicalTeller`, which works because assertions are nominal ([`grc20/types.gno:171`](https://github.com/gnolang/gno/blob/26ca914e2/examples/gno.land/p/demo/tokens/grc20/types.gno#L171)). As written the comment can be read as "embedding defeats the very assert we recommend." Fix: "embedding passes seal/marker checks."
  </details>

- [`gno-ai-contract-review.md:99`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L99) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L99) — checklist line 99 lists forbidden storage sites as "package-level vars, struct fields, or closure captures" but drops map values ([`gno-security-guide.md:321-323`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L321-L323)) and slices ([`gno-security-guide.md:439`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L439)); literal application misses `map[string]realm`. Fix: add "map values, slice elements."

## Missing Tests
None (docs-only).

## Suggestions
- [`gno-ai-contract-review.md:80-82`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L80-L82) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L80) — Check 6's RIGHT `Save` reads `cur.Previous().Address()` without the `IsCurrent()` guard Check 1 prescribes.
  <details><summary>details</summary>

  Not exploitable for a true crossing function (its `cur` is always the live injected value), but it contradicts Check 1 and teaches a pattern that becomes an unguarded caller read if copied into a non-crossing context. Fix: add the `IsCurrent()` guard to the `Save` example, or note in Check 1 that the guard is needed only when the realm value is caller-passed rather than the live `cur`.
  </details>

- [`gno-ai-contract-review.md:100`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L100) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L100) — checklist line 100 keeps only "unexported" from Check 7, but Check 7's body (line 88) also requires "do not return aliased pointers to them"; a `/p/`-embedded value whose method is promoted and reached through an exported getter pwns with no exported field.
  <details><summary>details</summary>

  Check 7's full rule is unexported AND no aliased-pointer return ([matching `gno-security-guide.md:231-232`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-security-guide.md?plain=1#L231-L232)); line 100 keeps only the first conjunct, rescued only if line 96 is widened per the Check 3 Warning. Fix: "and not reachable via a returned or promoted method."
  </details>

## Open questions
- Coverage: nothing in the 7 checks or 8 checklist lines touches re-entrancy via cross-realm calls, coin-math overflow/underflow, or caller-driven unbounded storage growth. The source security docs are themselves interrealm-scoped, so this is an inherited gap, but line 10 tells the reader these are "the highest-yield issues ... in any realm". A one-line scope sentence would set expectations. Not posted: soft framing point, not a concrete defect in a check.
- Further additions the checklist format fits cheaply (each one line): never import `gno.land/r/tests/vm/test20` (it exports its ledger); never return bound method values of `/p/`-types; never declare an interface method taking `cur realm`. Not posted: deferred scope, no decision the author must make in this PR.
- The relationship table lists `misc/audit-pattern-harness/` ([`gno-ai-contract-review.md:113`](https://github.com/gnolang/gno/blob/26ca914e2/docs/resources/gno-ai-contract-review.md?plain=1#L113)), absent from master; it ships in [#5835](https://github.com/gnolang/gno/pull/5835). Author confirmed #5835 merges first, so the path will resolve. Not an inline change request; a one-line merge-order note goes in the comment Body.
