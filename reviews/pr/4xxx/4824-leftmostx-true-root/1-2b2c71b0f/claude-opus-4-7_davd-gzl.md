# PR #4824: fix(gnovm): make LeftmostX return true root of chains

URL: https://github.com/gnolang/gno/pull/4824
Author: audrenbdb | Base: master | Files: 3 | +120 -6
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-4824 2b2c71b0f` (then `gh -R gnolang/gno pr checkout 4824` inside it)

Verdict: APPROVE — correctness fix to a single-caller AST helper, broader and more accurate heap-escape marking, fully tested, CI green; bug is real but not runtime-observable today (per [@thehowl](https://github.com/gnolang/gno/pull/4824#issuecomment-2799912842) Claude analysis) so merge value is preventing regression rather than fixing a live failure.

## Summary

[`LeftmostX`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/helpers.go#L811-L824) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/helpers.go#L811-L824) walks an AST expression chain to find the root variable used by heap-escape analysis. Before this PR it stopped after one level — `&a.b.c` resolved to `a.b`, not `a`, so the [`*RefExpr` case in `codaHeapDefinesByUse`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/preprocess.go#L3314-L3326) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/preprocess.go#L3314-L3326) failed its `lmx.(*NameExpr)` assertion and never marked `a` as a heap use. The fix replaces the single-step switch with an iterative loop that unwraps `SelectorExpr`, `IndexExpr`, and `SliceExpr` until a non-chain expression is reached. Stopping boundaries (`CallExpr`, `TypeAssertExpr`, `StarExpr`, `RefExpr`) are preserved because they fall through `default`.

The bug is correctness-only: GnoVM's Go GC keeps nested struct values alive through `PointerValue.Base` regardless of `HeapItemValue` promotion, so the missing mark cannot crash today. The closure-capture pass already independently promotes `o` whenever a closure escapes it. So this PR prevents a future regression more than it fixes user-visible behaviour — still worth landing for the same reason the issue was filed (audit finding #4778, Minor).

```
&a.b.c    BEFORE: LeftmostX -> a.b  (SelectorExpr, type-assert fails, a not marked)
          AFTER:  LeftmostX -> a    (NameExpr, a marked HeapUse)

&a[i].b   BEFORE: LeftmostX -> a[i].b (SelectorExpr -> not NameExpr)
          AFTER:  LeftmostX -> a

&a[:].b   BEFORE: LeftmostX -> a[:].b (SliceExpr unhandled in old switch)
          AFTER:  LeftmostX -> a
```

## Glossary

- `LeftmostX` — AST helper that returns the base expression of a selector/index/slice chain. One real caller.
- `codaHeapDefinesByUse` — preprocess pass that marks `NameExpr.Type = HeapUse` for variables whose address is taken via `&...`.
- `HeapDefine`/`HeapUse` — `NameExpr` annotations (`~` prefix in `Preprocessed:` dumps) that force a variable into a `HeapItemValue` cell so closures and external pointers can outlive the stack frame.
- `RefExpr` — AST node for `&X`.

## Fix

Pre-fix `LeftmostX` was a one-shot switch returning `x.X` after seeing the first `SelectorExpr` or `IndexExpr` and had no `SliceExpr` case at all. Post-fix it loops, reassigning `x = tx.X` until it hits a non-chain node, then returns. The load-bearing constraint — that `LeftmostX` must not cross dereference/address/call boundaries — is preserved purely by enumeration: only the three chaining cases descend; everything else (`*StarExpr`, `*RefExpr`, `*CallExpr`, `*TypeAssertExpr`, `*NameExpr`, ...) hits `default` and returns. Doc comment at [`helpers.go:803-810`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/helpers.go#L803-L810) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/helpers.go#L803-L810) is updated to describe the new behaviour and stopping rules. Coverage: nine unit cases in [`helpers_test.go:157-228`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/helpers_test.go#L157-L228) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/helpers_test.go#L157-L228) plus a filetest in [`ref_nested_sel.gno`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/tests/files/ref_nested_sel.gno) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/tests/files/ref_nested_sel.gno) that pins the post-fix `Preprocessed:` dump containing `o<!~VPBlock(1,0)>` (the `~` confirms HeapDefine).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/pkg/gnolang/helpers.go:809`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/helpers.go#L809) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/helpers.go#L809) — doc says "until a non-selector expression is found"; "non-chain" or "non-selector/index/slice" is more precise since the loop also walks index/slice nodes. One-word edit.
- [`gnovm/pkg/gnolang/helpers_test.go:193`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/helpers_test.go#L193) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/helpers_test.go#L193) — equality is done via `toExprTrace`, which has no dedicated `*SliceExpr` case in [`frame.go:193`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/frame.go#L193) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/frame.go#L193) and falls through to `ex.String()`. Works today but the test's notion of "equal" silently shifts if anyone adds a `SliceExpr` case to `toExprTrace`. Either add an explicit `SliceExpr` case there, or switch the test to a structural compare on `reflect.TypeOf(got) == reflect.TypeOf(want)` plus a recursive field walk.
- [`gnovm/tests/files/ref_nested_sel.gno:29`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/tests/files/ref_nested_sel.gno#L29) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/tests/files/ref_nested_sel.gno#L29) — the `// Preprocessed:` dump is one long line; if a future PR changes any unrelated annotation in that printer the diff will be enormous and the bug-relevance (`o<!~VPBlock(1,0)>` vs `o<!VPBlock(1,0)>`) will be hidden. Consider adding a short comment naming the load-bearing tokens (`~` on `o` = HeapDefine, `~` on `gp` references = HeapUse) so the next reviewer doesn't have to grep for `~` characters.

## Missing Tests

- [`gnovm/pkg/gnolang/helpers_test.go:157`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/helpers_test.go#L157) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/helpers_test.go#L157) — no case for a bare `NameExpr` or a `BasicLitExpr` input. Trivial but seals the "default returns input untouched" contract.
  <details><summary>details</summary>

  Add two cases: `LeftmostX(name("a")) == name("a")` and `LeftmostX(&BasicLitExpr{...}) == &BasicLitExpr{...}` (identity, not just trace-equal). Prevents an accidental refactor from introducing nil panics on non-chain input. Cheap.
  </details>

- [`gnovm/pkg/gnolang/helpers_test.go:157`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/helpers_test.go#L157) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/helpers_test.go#L157) — no case for `RefExpr` boundary (`&a`).
  <details><summary>details</summary>

  The PR description lists `RefExpr` as a stopping rule and the implementation depends on it (`*RefExpr` hits `default`), but no test exercises that branch. The single caller in [`preprocess.go:3315`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/preprocess.go#L3315) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/preprocess.go#L3315) already strips the outer `&` (it inspects `n.X`), so it's hard to construct natural Gno that hits this — but the helper is exported and the contract should be verified. One line.
  </details>

- [`gnovm/tests/files/ref_nested_sel.gno`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/tests/files/ref_nested_sel.gno) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/tests/files/ref_nested_sel.gno) — no closure-capture variant.
  <details><summary>details</summary>

  The closure-capture pass is what made this bug invisible in real workloads (per [@thehowl](https://github.com/gnolang/gno/pull/4824#issuecomment-2799912842)). A second filetest with `&a.b.c` taken inside a closure that escapes would document that the two heap-promotion paths now agree on `a`. Not blocking; useful for future regression hunting.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/preprocess.go:3314-3326`](https://github.com/gnolang/gno/blob/2b2c71b0f/gnovm/pkg/gnolang/preprocess.go#L3314-L3326) · [↗](../../../../../.worktrees/gno-review-4824/gnovm/pkg/gnolang/preprocess.go#L3314-L3326) — silent no-op when `lmx` is not a `NameExpr`.
  <details><summary>details</summary>

  After the fix, `LeftmostX` correctly returns the chain root, but the consumer still silently drops anything that isn't a `NameExpr` (e.g. `&f().b` leaves `lmx` as `*CallExpr`). That's the intended behaviour, but a one-line comment — "non-NameExpr roots (call, type-assert, deref) need no heap mark; they are already addressable / freshly allocated" — would prevent a future reader from assuming this is another instance of the bug just fixed.
  </details>

## Questions for Author

- The PR description lists `StarExpr` as a stopping boundary but doesn't justify it with an example. For `&(*p).b.c`, the heap-escape root is whatever `p` points to, not a local — is the assumption that the `*p` deref is itself handled elsewhere (the pointer's target is already heap-allocated by construction)? A one-line note in the PR body would help future readers understand why `StarExpr` is *correctly* a boundary rather than something to also unwrap.
