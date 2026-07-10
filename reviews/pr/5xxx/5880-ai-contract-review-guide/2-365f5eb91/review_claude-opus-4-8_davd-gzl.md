# PR [#5880](https://github.com/gnolang/gno/pull/5880): docs: add concise AI contract review guide

URL: https://github.com/gnolang/gno/pull/5880
Author: moul | Base: master | Files: 3 | +303 -1
Reviewed by: davd-gzl | Model: claude-opus-4-8 (high) | Commit: 365f5eb91 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5880 365f5eb91`

**TL;DR:** Grows `docs/resources/gno-ai-contract-review.md` from 7 to 10 checklist cases (adds `/p/`-type mutation methods returned as pointers, `unsafe.PreviousRealm()`, and unsanitized `Render` input), extends `gno-security-guide.md` with matching §5.1a/§5.1b/§5.8 sections, and folds the same rules into `AGENTS.md`. Still a copy-paste distillation an AI applies inline without tooling.

**Verdict: REQUEST CHANGES** — every round-1 finding still stands (the Review Checklist restates its Quick Checks and drifts: line 194 greenlights Check 4's WRONG example, line 198 flags every grc20 token realm, Check 4's WRONG snippet can't write victim state), and the new content adds one more of the same class: the unsafe rule flags any import of `chain/runtime/unsafe`, but that package also exports `OriginCaller()`/`OriginSend()`, the tx-origin primitives the stdlib itself blesses. All fixes are small wording changes.

## Round note
Round 2. Head advanced 26ca914e2 → 365f5eb91 (two commits: cases 8-10 added to the checklist, §5.1a/§5.1b/§5.8 added to the security guide, four lines added to `AGENTS.md`). Patch-ids differ, so a full round. All five round-1 Warnings, five Nits, and two Suggestions carry unchanged, re-anchored to the shifted lines. New this round: one Warning (unsafe rule over-broadens past caller-identity) and one Nit (`avl.Tree` field names in §5.1a are wrong). Verdict unchanged: REQUEST CHANGES.

The three new checks are directionally correct: `unsafe.PreviousRealm()` exists at [`unsafe.gno:26`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L26) and skips the `cur.IsCurrent()` frame check; `sanitize.InlineText` exists at [`sanitize.gno:345`](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L345); Check 8's borrow-rule-#2 write-commit path matches the guide's own [§2.2](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L54) and [§5.1](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L216).

## Summary
The file maps one-to-one onto §5.1-5.8 of `gno-security-guide.md`. The seven original checks are unchanged from round 1; the three additions cover: a `/p/`-type stored as a realm-owned pointer whose exported methods (`avl.Tree.Set`) mutate under borrow rule #2 even when all fields are unexported (§5.1a/§5.1b), `chain/runtime/unsafe.PreviousRealm()` used in a crossing function that already holds a `cur realm` (§5.8), and unsanitized user input written into `Render` output (Check 10, no security-guide section). The defects remain in the examples and the Review Checklist, which an AI reader copies verbatim: of the seven checklist lines flagged here, five (188, 191, 194, 196, 197) restate their check more narrowly than the source and two (189, 198) more broadly. Check 4's WRONG snippet also cannot actually write victim state, and the new unsafe rule flags legitimate tx-origin usage.

## Glossary
- crossing function: `func F(cur realm, ...)`; identifies its caller through `cur.Previous()`.
- borrow rule #2: a `/p/`-declared method invoked on a receiver stored in realm `/r/V` borrows `m.Realm` to `/r/V` for the call, so the method body writes `/r/V` state regardless of who called it.
- ephemeral realm: short-lived code realm a `maketx run` executes under; `IsUser()` accepts it, `IsUserCall()` does not.
- canonical-type assert: a nominal type assertion (`IsCanonicalTeller`) that only the declaring package can satisfy; embedding cannot forge it.
- tx-origin identity: the transaction's originating account/realm, obtained via `unsafe.OriginCaller()`/`OriginSend()`; distinct from the immediate caller `cur.Previous()` gives.

## Critical (must fix)
None.

## Warnings (should fix)

- **[the WRONG example can't do the harm the comment claims]** [`gno-ai-contract-review.md:48-56`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L48-L56) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L48) — Check 4's WRONG example `func ApplyHook(fn func()) { fn() }` with the comment "inherits the caller's `m.Realm` and can write to your state" does not exhibit the vulnerability.
  <details><summary>details</summary>

  A bare `func()` supplied by `/p/` code cannot write victim state: it receives no pointer, cannot name any `/r/V` symbol (`/p/`→`/r/` import is a hard preprocess panic, [`preprocess.go:5450`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/pkg/gnolang/preprocess.go#L5450)), cannot construct a banker, and cannot spoof victim events (`Emit` keys off the calling frame's package, [`emit_event.go:68`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/stdlibs/chain/emit_event.go#L68)). The real laundering vector needs the victim to pass a writable pointer to a callback whose parameter type `/p/` code can satisfy; a `/p/`-typed pointer callback succeeds where an `/r/`-declared one is blocked by readonly taint. Worse, the RIGHT example (`func ApplyHook(fn func(*MyState)) { fn(gState) }`) hands out a state pointer the WRONG one never did, so the pair reads as "adding a state pointer is the fix," backwards. The comment also misattributes authority: `m.Realm` stays at *your* realm for the callback ([`gno-security-guide.md:130-137`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L130-L137)), it is not "the caller's". Fix: make WRONG a `/p/`-typed-pointer callback, e.g. `func ApplyHook(fn func(*somelib.Node)) { fn(gNode) }`, keep RIGHT with an `/r/`-declared parameter type, and reword the comment to "runs under your realm's authority."
  </details>

- **[checklist line greenlights the doc's own WRONG example]** [`gno-ai-contract-review.md:194`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L194) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L194) — line 194 flags only a callback "with a `/p/`-typed parameter", but Check 4's WRONG example is `func()` with no parameter, so an AI walking the checklist passes the exact code the doc calls WRONG.
  <details><summary>details</summary>

  The source is explicit that the signature filter is wrong: "Even `func()` is dangerous" ([`gno-security-guide.md:297`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L297)). The danger keys on the *supplied value* being a top-level `/p/` FuncDecl, not on the parameter type; the defense is retyping the callback parameter with an `/r/`-declared type. Line 194 inverts that into "flag only `/p/`-typed params" while the example is a plain function. Fix: widen to "no method invokes a caller-supplied func or interface value whose signature `/p/` code can satisfy, including plain `func()`."
  </details>

- **[checklist line flags every grc20 token realm]** [`gno-ai-contract-review.md:198`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L198) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L198) — "Data types holding sensitive state are declared in this realm (`/r/`), not in shared `/p/`", stated unconditionally, marks the source guide's own blessed pattern as a bug.
  <details><summary>details</summary>

  A token realm holds its balances in a `/p/`-declared `grc20.Token` — exactly "sensitive state" in a `/p/` type ([`wugnot.gno:16-17`](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L16-L17)). The source blesses this: grc20 "violates (A) ... but compensates with airtight encapsulation" ([`gno-security-guide.md:176-178`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L176-L178)) and the escape clause "if using `/p/`-declared types (e.g. `grc20.Token`), they're stored in **unexported** package vars" ([`gno-security-guide.md:495-497`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L495-L497)) is dropped. Line 198 is also the only checklist line with no matching Quick Check, so no example tempers the false positive. An AI applying it flags essentially every token realm on chain. Fix: append the guide's qualifier (unexported storage, no leaked pointers, canonical asserts).
  </details>

- **[Check 3 rule drops slices and maps, the source's own primary example]** [`gno-ai-contract-review.md:191`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L191) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L191) — checklist line 191 and Check 3 both say only "pointer", so a returned internal slice or map header sails through.
  <details><summary>details</summary>

  The guide's anti-pattern 5.1 example is a slice return (`func Users() []*User { return users }`) and its rule text is "Any pointer (slice header, map, struct pointer) returned by a getter" ([`gno-security-guide.md:207-213`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L207-L213)). Check 3's example (lines 38-44) shows only a struct pointer and line 191 says only "pointer", so an AI passes `func Tags() []string` or `func Registry() map[string]*User`. Fix: "No exported function returns a pointer, slice, or map aliasing internal mutable state."
  </details>

- **[relationship table routes the reader to the outdated interrealm doc for a symbol it lacks]** [`gno-ai-contract-review.md:209`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L209) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L209) — the row credits `gno-interrealm.md` with "`cur realm`, `IsCurrent()`, borrow rules", but that file never mentions `IsCurrent()` and self-marks as outdated.
  <details><summary>details</summary>

  `grep -c IsCurrent gno-interrealm.md` returns 0; the symbol is defined and explained in `gno-interrealm-v2.md` ([`gno-interrealm-v2.md:378`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-interrealm-v2.md?plain=1#L378)). The v1 header reads "This document is outdated, see `gno-interrealm-v2.md`" ([`gno-interrealm.md:3-5`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-interrealm.md?plain=1#L3-L5)). A routing doc for AI agents sends the reader to the superseded spec for the one symbol it does not contain. Fix: point the row at `gno-interrealm-v2.md`, or add a v2 row.
  </details>

- **[unsafe rule flags legitimate tx-origin usage]** [`gno-ai-contract-review.md:189`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L189) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L189) — checklist line 189 and Check 9's rule (line 161) flag *any* import of `chain/runtime/unsafe` alongside `cur realm`, but that package also exports `OriginCaller()`/`OriginSend()`, tx-origin primitives with no `cur` substitute.
  <details><summary>details</summary>

  `chain/runtime/unsafe` exports four symbols: `PreviousRealm`, `CurrentRealm`, `OriginCaller`, `OriginSend` ([`unsafe.gno:26-64`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L26-L64)). Check 9 correctly targets `PreviousRealm()` for caller identity, which `cur.Previous()` replaces. But the package doc explicitly reserves `OriginCaller()`/`OriginSend()` "for cases where you intentionally want tx-level identity (event emission, fee attribution) and pair them with `runtime.AssertOriginCall()`" ([`unsafe.gno:9-12`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L9-L12)); `cur` gives the immediate caller, not the tx origin, so there is no substitute. A crossing realm that emits events with `unsafe.OriginCaller()` legitimately imports the package. Line 189's "No import ... alongside `cur realm`", Check 9's line 161 "Flag any import", and the security guide's "Delete the `chain/runtime/unsafe` import" ([`gno-security-guide.md:407-413`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L407-L413)) plus the `AGENTS.md` "red flag" line all over-broaden from `PreviousRealm()`/`CurrentRealm()` to the whole package. Fix: scope the rule to `PreviousRealm()`/`CurrentRealm()` used for caller identity, not any import of the package.
  </details>

## Nits
- [`gno-ai-contract-review.md:76-77`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L76-L77) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L76) — Check 6 `// WRONG: panics at attach time` on a bare `var savedRealm realm` is doubly imprecise: the bare declaration runs clean, and the panic fires on assignment of a live realm value at transaction finalize, not attach.
  <details><summary>details</summary>

  A package-level `var savedRealm realm` with nothing stored runs to completion; the persistence check trips only when a live realm value is assigned and the object graph is finalized, matching the source ("panics at attachment time or transaction finalize", [`gno-security-guide.md:372-375`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L372-L375)) and the committed filetest [`zrealm_cur_persist_var.gno`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/tests/files/zrealm_cur_persist_var.gno). Confirmed behaviorally in round 1: bare var runs clean, assignment errors at finalize with `cannot persist realm value: realm values are ephemeral and tied to a call frame`. Fix: move the WRONG comment onto an assignment line and say "finalize", e.g. `func Save(cur realm) { savedRealm = cur // panics at tx finalize }`.
  </details>

- [`gno-ai-contract-review.md:188`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L188) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L188) — checklist line 188 "call `cur.IsCurrent()`" is satisfied by a no-op `_ = cur.IsCurrent()`, which gives zero protection; the result must gate execution.
  <details><summary>details</summary>

  A stale or stashed realm value returns `IsCurrent()` = false, so the return value has to panic-gate the body, as Check 1 shows at lines 20-21. Verifying only that the code "calls IsCurrent()" passes unguarded code. Fix: reword to "panic unless `cur.IsCurrent()`."
  </details>

- [`gno-ai-contract-review.md:30`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L30) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L30) — Check 2's WRONG `if !IsUser()` calls a free function that does not exist; `IsUser` is a method on a realm value ([`uverse.go:131`](https://github.com/gnolang/gno/blob/365f5eb91/gnovm/pkg/gnolang/uverse.go#L131)), so the snippet is not valid gno and breaks parallelism with the RIGHT line's `cur.Previous().IsUserCall()`. Fix: `if !cur.Previous().IsUser()`.

- [`gno-ai-contract-review.md:61`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L61) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L61) — Check 5's WRONG comment "`Evil{Teller}` embedding bypasses interface checks" is imprecise: embedding bypasses seal/marker checks, not the canonical-type assert, which is nominal and is precisely the RIGHT fix on the next lines.
  <details><summary>details</summary>

  The guide's claim is that embedding "passes any seal/marker check via the embedded methods" ([`gno-security-guide.md:315-316`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L315-L316)); it does not bypass `IsCanonicalTeller`, which works because assertions are nominal ([`grc20/types.gno:171`](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/p/demo/tokens/grc20/types.gno#L171)). As written the comment can be read as "embedding defeats the very assert we recommend." Fix: "embedding passes seal/marker checks."
  </details>

- [`gno-ai-contract-review.md:196`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L196) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L196) — checklist line 196 lists forbidden storage sites as "package-level vars, struct fields, or closure captures" but drops map values (§5.7, [`gno-security-guide.md:372-374`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L372-L374)) and slices (§8, [`gno-security-guide.md:523-524`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L523-L524)); literal application misses `map[string]realm`. Fix: add "map values, slice elements."

- [`gno-security-guide.md:230`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L230) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-security-guide.md#L230) — §5.1a says "`avl.Tree` ... All fields (`root`, `size`) are unexported", but the real struct is `type Tree struct { node *Node }` ([`avl/v0/tree.gno:27`](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/p/nt/avl/v0/tree.gno#L27)); there is no `root` or `size` field.
  <details><summary>details</summary>

  The point (all fields unexported, exported mutators `Set`/`Remove`) holds — the field is `node`, and `Set`/`Remove` exist ([`avl/v0/tree.gno:65-73`](https://github.com/gnolang/gno/blob/365f5eb91/examples/gno.land/p/nt/avl/v0/tree.gno#L65-L73)). Only the named fields are fabricated. In a doc built on precise examples, wrong field names invite a reader to distrust the rest. Fix: `type Tree struct { node *Node }`, or drop the parenthetical.
  </details>

## Missing Tests
None (docs-only).

## Suggestions
- [`gno-ai-contract-review.md:80-82`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L80-L82) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L80) — Check 6's RIGHT `Save` reads `cur.Previous().Address()` without the `IsCurrent()` guard Check 1 prescribes.
  <details><summary>details</summary>

  Not exploitable for a true crossing function (its `cur` is always the live injected value), but it contradicts Check 1 and teaches a pattern that becomes an unguarded caller read if copied into a non-crossing context. Fix: add the `IsCurrent()` guard to the `Save` example, or note in Check 1 that the guard is needed only when the realm value is caller-passed rather than the live `cur`.
  </details>

- [`gno-ai-contract-review.md:197`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L197) · [↗](../../../../../.worktrees/gno-review-5880/docs/resources/gno-ai-contract-review.md#L197) — checklist line 197 keeps only "unexported" from Check 7, but Check 7's body (line 88) also requires "do not return aliased pointers to them"; an unexported `/p/`-type field handed out by an exported getter satisfies line 197 and leaks the same handle.
  <details><summary>details</summary>

  Check 7's full rule is unexported AND no aliased-pointer return ([matching §5.2's rule, `gno-security-guide.md:280-283`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-security-guide.md?plain=1#L280-L283)); line 197 keeps only the first conjunct, rescued only if line 191 is widened per the Check 3 Warning.

  ```go
  type Registry struct {
      tree *avl.Tree // unexported: line 197 passes
  }

  // the alias escapes anyway; Iterate then runs a caller-supplied
  // /p/ callback under this realm's authority
  func (r *Registry) Tree() *avl.Tree { return r.tree }
  ```

  Fix: "and not returned by any exported function or method."
  </details>

## Open questions
- Coverage: nothing in the 10 checks touches re-entrancy via cross-realm calls, coin-math overflow/underflow, or caller-driven unbounded storage growth. The source security docs are themselves interrealm-scoped, so this is an inherited gap, but line 10 tells the reader these are "the highest-yield issues ... in any realm". A one-line scope sentence would set expectations. Not posted: soft framing point, not a concrete defect in a check.
- The relationship table's `misc/audit-pattern-harness/` row ([`gno-ai-contract-review.md:211`](https://github.com/gnolang/gno/blob/365f5eb91/docs/resources/gno-ai-contract-review.md?plain=1#L211)) points at a path absent from master; it ships in [#5835](https://github.com/gnolang/gno/pull/5835). Round 2 dropped the inline `#5835` link the round-1 table carried, so the reader now has no pointer to where the path comes from. Author confirmed #5835 merges first, so the path resolves after that. Not posted: merge-order thread state the author already knows.
