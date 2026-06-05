# PR #5763: fix(gnovm): allow unsealed *DeclaredType during mutual type-decl recursion

URL: https://github.com/gnolang/gno/pull/5763
Author: ltzmaxwell | Base: master | Files: 2 | +22 -7
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 61fc396e4 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5763 61fc396e4`

**Verdict: REQUEST CHANGES** — the removed `panic("should not happen")` was over-strict, but dropping it converts a loud predefine-time abort into silent structural corruption for the closely related shape `type T2 T1` where `T1` is declared first and carries data fields: `T2`'s base collapses to `struct{}` and any field access on `T2` panics at runtime. The fix suppresses the symptom (the panic) without making the underlying mutual-recursion case actually resolve correctly. Determinism is intact (errors are byte-stable), and the legitimate shapes the PR targets do work — but the PR's own filetest passes only because it never reads a field on `T2`.

## Summary

`tryPredefine` panicked `"should not happen"` whenever a type-decl's RHS resolved to a still-unsealed `*DeclaredType`. That state is legitimate during mutual type-decl recursion, so the panic was wrong. The PR removes it. But removing the guard does not *fix* the mutual case `type T2 T1` (T1 first, T1 carrying its own data fields) — it only changes the failure from a predefine panic to a wrong type. With the guard gone, `T2`'s underlying struct is finalized from the still-empty predefined `T1` and ends up as `struct{}`; reading any field through `T2` then panics with `struct type struct{} has no field <F>`. The PR's filetest (`decltype_mutual.gno`) hides this because it uses `&T2{}` with no fields and never reads one.

```
master (guard present):           PR 61fc396e4 (guard removed):
  type T1 struct{Next *T2; Val int}  type T1 struct{Next *T2; Val int}
  type T2 T1                         type T2 T1
        |                                  |
  predefine T2 from unsealed T1      predefine T2 from unsealed T1
        |                                  |
  panic "should not happen"  <-- loud  compiles; T2.Base = struct{}  <-- silent corruption
                                           |
                                     runtime: "struct type struct{} has no field Val"
```

## Glossary
- `tryPredefine` — preprocessor pass that partially defines package-level names so recursion resolves; `preprocess.go:5361`.
- `DeclaredType` — a named type (`type Foo ...`); holds `Base` (underlying type), `Methods`, and a `sealed` bit; `types.go:1471`.
- `sealed` — one-bit lifecycle marker meaning "Base/Methods filled in"; used for recursion protection in `realm.fillType` and unsealed-equality in `type_check.go`. Not a "do not touch" lock; `types.go:1961`.
- `declareWith` — constructs an unsealed `*DeclaredType` with `Base: baseOf(b)`; `types.go:1517`.
- TRANS_LEAVE TypeDecl — the handler that finalizes a named type: evaluates the RHS, builds a fresh `*DeclaredType`, seals it, copies it back into the predefined pointer; `preprocess.go:3018`.

## Fix

Before: the `*NameExpr` branch of `tryPredefine` looked up the RHS name's slot, and if the resolved type was an unsealed `*DeclaredType` it panicked `"should not happen"` ([`preprocess.go:5493-5499`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/pkg/gnolang/preprocess.go#L5493-L5499) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5493-L5499)). After: it just reads `tv.GetType()` and proceeds. The PR body's two claims hold up: `sealed` is documented as a lifecycle marker, not a slot-lookup precondition ([`types.go:1961-1971`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/pkg/gnolang/types.go#L1961-L1971) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1961-L1971)), and the sites that need a sealed type gate explicitly — `type_check.go` matches unsealed declared types by `PkgPath`+`Name` instead of `TypeID()` ([`type_check.go:440-446`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/pkg/gnolang/type_check.go#L440-L446) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/type_check.go#L440-L446)), and `realm.fillType` uses `sealed` purely for recursion protection ([`realm.go:1788-1799`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/pkg/gnolang/realm.go#L1788-L1799) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/realm.go#L1788-L1799)). The gap is that "the panic was wrong" does not imply "removing it makes the case work" — for the empty-base shape it does not.

## Critical (must fix)

- **[silent type corruption — `type T2 T1` collapses to empty struct]** [`preprocess.go:3066`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/pkg/gnolang/preprocess.go#L3066) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3066) — `type T1 struct{Next *T2; Val int}; type T2 T1` compiles but `T2`'s base is `struct{}`; any field access on `T2` panics at runtime.
  <details><summary>details</summary>

  **Shape:** `type T1 struct{ Next *T2; Val int }` declared first, then `type T2 T1`. Master panics `"should not happen"` at predefine. PR removes that panic; the program now reaches TRANS_LEAVE for `T2`, where `tmp := evalStaticType(store, last, n.Type)` evaluates the `T1` name while `T1`'s own struct body has not yet been copied back into its predefined `*DeclaredType` pointer. `declareWith(..., tmp)` then captures `Base: baseOf(tmp)` = the empty struct, seals it, and writes the empty `T2` back.

  **What you see:** constructing `&T2{Val: 9}` and reading the field panics `struct type struct{} has no field Val`; the whole underlying struct is empty, not just one field. The filetest below asserts the correct output (`9`, `ok`), so it *fails* on `61fc396e4` with that panic and will pass once `T2`'s base resolves to `T1`'s struct. The error string is byte-stable across runs (verified 3×). Copy-paste reproducer in the **Repro** block below.

  The corruption is in `T2` itself, not in the extra `Val` field: even the PR's own `decltype_mutual.gno` shape (`type T1 struct{ Next *T2 }; type T2 T1`, no data field) panics `missing field Next in main.T2` the moment a field is read through a `T2` value rather than a `T1`. The PR's test escapes this only because it declares `var t T1` (reads `.Next` through `T1`) and never sets a field on the `&T2{}` literal.

  **Why it matters:** the PR's stated goal is to make mutual type-decl recursion work. For this shape it does not work — it produces a structurally wrong named type that the user can construct (`&T2{}`) and store, but cannot use. The corruption is silent at compile time and only surfaces as a confusing runtime panic far from the declaration. This is a worse failure mode than the original loud panic. The PR's own filetest `decltype_mutual.gno` does not catch it because `&T2{}` carries no field and `main` never reads a field on `T2`.

  **Determinism note:** behavior is declaration-order-dependent. `type T2 T1` *before* `type T1 struct{...*T2...}` resolves correctly (prints the field); `type T1` first corrupts `T2`. Same logical types, different result by source order. Errors themselves are byte-stable across runs (verified 3×), so this is not a map-iteration nondeterminism — it is order-sensitivity in the predefine/finalize sequence.

  **Fix:** make `type T2 T1` (T1-first, mutual) finalize `T2`'s base from `T1`'s *completed* struct — e.g. defer `T2`'s base resolution until `T1`'s TRANS_LEAVE has copied its body back, or resolve through the predefined `T1` pointer rather than snapshotting `baseOf(tmp)` while `T1` is empty. If correct resolution is out of scope for this PR, the case must still error cleanly (keep a guard that rejects *empty-base* finalization) rather than silently producing `struct{}`. Repro: [`tests/decltype_derived_empty_base.gno`](tests/decltype_derived_empty_base.gno).
  </details>

### Repro

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
cat > gnovm/tests/files/decltype_derived_empty_base.gno <<'EOF'
package main

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

func main() {
	var a T1
	a.Next = &T2{Val: 9}
	println(a.Next.Val)
	println("ok")
}

// Output:
// 9
// ok
EOF
go test -v -run 'TestFiles/decltype_derived_empty_base.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/decltype_derived_empty_base.gno
```

```
--- FAIL: TestFiles/decltype_derived_empty_base.gno (0.00s)
    files_test.go:111: unexpected panic: main/decltype_derived_empty_base.gno:12:15-18: struct type struct{} has no field Val
FAIL
```

The test asserts the correct output (`9`, `ok`), so it FAILS on `61fc396e4` with the panic above — `T2`'s underlying struct is `struct{}`, not `T1`'s — and goes green only once `T2`'s base resolves to `T1`'s struct. Saved with a standalone `/* Run: */` header as [`tests/decltype_derived_empty_base.gno`](tests/decltype_derived_empty_base.gno).

## Warnings (should fix)

- **[test does not exercise the fix it claims to validate]** [`gnovm/tests/files/decltype_mutual.gno:11-13`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/tests/files/decltype_mutual.gno#L11-L13) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L11-L13) — the added filetest passes only because it never reads a field on `T2`.
  <details><summary>details</summary>

  The test constructs `&T2{}` with no field and never accesses a `T2` field, so it sidesteps the empty-base corruption entirely. It proves the panic is gone, not that the type resolves correctly. A regression suite for "mutual type-decl recursion works" should read a data field through the derived type (as `tests/decltype_derived_empty_base.gno` does) and through both ends of the cycle. As written, a future change that re-corrupts `T2.Base` would keep this test green.

  Fix: extend the filetest to declare a data field on `T1` and read it via `T2` (e.g. `var b T2; b.Val = 1; println(b.Val)`), so the assertion actually depends on `T2`'s base being `T1`'s struct.
  </details>

## Nits

- [`gnovm/pkg/gnolang/preprocess.go:5494-5497`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/pkg/gnolang/preprocess.go#L5494-L5497) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5494-L5497) — the new comment asserts "TRANS_LEAVE seals it later via declareWith"; `Seal()` is called once at [`preprocess.go:3068`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/pkg/gnolang/preprocess.go#L3068) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3068), not by `declareWith` (which returns an *unsealed* type, `types.go:1515`). Reword to "TRANS_LEAVE seals it later (`tmp2.Seal()`)" to avoid pointing the next reader at the wrong function.

## Missing Tests

- **[empty-base regression has no guard]** [`gnovm/tests/files/`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/tests/files/) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/) — no filetest pins the `type T2 T1` field-access behavior.
  <details><summary>details</summary>

  No filetest in the PR pins the empty-base shape. [`tests/decltype_derived_empty_base.gno`](tests/decltype_derived_empty_base.gno) asserts the correct output (`9`, `ok`); it fails on `61fc396e4` today and locks in the resolution once the Critical is fixed. Also worth adding: the declaration-order-swapped variant (T2 before T1, which already works) to pin that both orders agree.
  </details>

## Suggestions

- The ENTER-phase `declareWith` at [`preprocess.go:5539`](https://github.com/gnolang/gno/blob/61fc396e4/gnovm/pkg/gnolang/preprocess.go#L5539) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5539) computes a provisional `Base` from the predefine-time `t`; the TRANS_LEAVE handler overwrites it via `*dstT = *tmp2`. If the intent is that the predefine-time base is always discarded, a one-line comment at 5539 noting "Base is provisional; overwritten at TRANS_LEAVE" would save the next reader the trace I had to do.

## Questions for Author

- Is `type T2 T1` (T1-first, T1 carrying data fields, mutual via `*T2`) intended to be supported by this PR, or only the no-field shape in `decltype_mutual.gno`? If the former, the empty-base corruption is a blocker; if the latter, the case still needs a clean error instead of `struct type struct{} has no field`.
- Was the `should not happen` panic ever observed firing in the wild (an issue link), or is this fixing a hypothetical? The motivation would help scope the right fix vs. a narrower guard.
