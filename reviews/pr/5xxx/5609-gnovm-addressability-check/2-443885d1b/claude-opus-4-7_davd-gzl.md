# PR #5609: fix(gnovm): validate addressability of & operand at preprocess stage

URL: https://github.com/gnolang/gno/pull/5609
Author: aronpark1007 | Base: master | Files: 27 | +135 -26
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5609 443885d1b` (then `gh -R gnolang/gno pr checkout 5609` inside it)

**Verdict: APPROVE** — Closes #5586 for the listed expression classes (binary/unary, type-assert, map index, call result, slice literal) and moves the validation from a runtime crash to a structured preprocess panic; the `*NameExpr → true` branch leaves three pre-existing escape hatches (`&funcName`, `&t.M`, `&s[i]`) but those are not regressions, and the author has explicitly scoped them out for a follow-up.

## Summary

Gno's preprocess stage previously didn't validate the `&` operand's addressability; non-addressable forms either silently produced a usable pointer at runtime (e.g., `&m[k]` for a map, `&i.(S).a` for a type assertion field) or crashed with an opaque `illegal assignment X expression type *gnolang.CallExpr`-style message late in interpretation. This PR adds an `isAddressable(store, last, x)` helper and calls it from `TRANS_LEAVE *RefExpr` (line 2344) plus, for arrays, from `TRANS_LEAVE *SliceExpr` (line 2166). The check is a pure-AST recursion over five node kinds — name, deref, composite literal, index, selector — mirroring Go's spec; everything else falls into `default → false`. 20 existing `addressable_*_err.gno` filetests had their runtime errors replaced with preprocess errors; three new ones (`7c`, `7d`, `9c`) cover `&(a+b)`, `&(-x)`, and `&s.m["a"]`; two more (`6e`, `9d`) cover the new array-slice rejection.

## Glossary

- `isAddressable(store, last, x)` — new preprocess helper at `preprocess.go:4393`; recursive structural walk over `Expr`.
- `*RefExpr` — AST node for `&x`. TRANS_LEAVE fires bottom-up after children are processed.
- `*StarExpr` — AST node for `*p` (deref). Used both as user syntax and as a preprocess-injected wrapper.
- `evalStaticTypeOf` — preprocess type lookup; returns the resolved `Type` for an `Expr` from the surrounding block.
- `ATTR_REF_ELEM_TYPE` — attribute set on `*RefExpr` post-check; consumed by `doOpRef` to build the result pointer's element type.

## Fix

Before: `TRANS_LEAVE *RefExpr` only rejected multi-value calls; everything else flowed through to `doOpRef`, which constructs a `*PointerType{Elt: <ATTR_REF_ELEM_TYPE>}` regardless of whether the operand could have a real address. After: an `isAddressable` gate at [`preprocess.go:2344-2348`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2344-L2348) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2344-L2348) panics with a Go-shaped `"invalid operation: cannot take address of <expr> (value of type <T>)"` message. The check is also reused by [`preprocess.go:2165-2170`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2165-L2170) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2165-L2170) to reject `arr[:]` when the array is unaddressable (e.g., `getArr()[:]`, `m["k"][:]`, `i.(arr1)[:]`). The helper itself, [`preprocess.go:4393-4413`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4393-L4413) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4393-L4413), returns `true` for name/deref/composite, switches on `cx.X.Kind()` for index (Map → false, Slice → true, else recurse), and for selector returns true when the base is a pointer (else recurse). Critically, `*ConstExpr` is not handled — but by the time `TRANS_LEAVE *RefExpr` fires, constant `*NameExpr` references have already been rewritten to `*ConstExpr` at [`preprocess.go:1314-1325`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L1314-L1325) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L1314-L1325), so they fall into `default → false`. Same trick covers pointer-indexed arrays: `getPtr()[0]` is rewritten to `(*getPtr())[0]` at [`preprocess.go:2127-2134`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2127-L2134) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2127-L2134) before `RefExpr` leaves, so the `*StarExpr` arm catches it.

## Critical (must fix)

None.

## Warnings (should fix)

- **[silent acceptance of `&funcName` for user functions]** [@notJoon](https://github.com/gnolang/gno/pull/5609) (raised as scope-out) — [`preprocess.go:4395`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4395) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4395) — `*NameExpr` returns `true` unconditionally; a package-level function name reaches `doOpRef` and a usable `*func()` is constructed and called at runtime.
  <details><summary>details</summary>

  `func bar() {}; p := &bar; (*p)()` runs to completion under PR #5609 — adversarial test [`addressable_gap_funcname.gno`](tests/addressable_gap_funcname.gno) prints `ok` with `// Output:` asserted. Go's typechecker (already wired in for the `// TypeCheckError:` annotation in the same file) rejects it: `invalid operation: cannot take address of bar (value of type func())`. The mechanism is that user functions defined at package level remain `*NameExpr` after preprocess (they aren't const-folded the way uverse names like `println` are at [`preprocess.go:1297-1311`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L1297-L1311) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L1297-L1311)), so the name reaches `isAddressable`'s catch-all `true`. Pre-existing — not introduced by this PR — and acknowledged by the author as out of scope; the issue is that the helper now looks complete, so a follow-up tracking issue would be wise so the gap doesn't decay invisibly. Fix: in the `*NameExpr` arm, check `evalStaticTypeOf(store, last, x).Kind()` and return `false` for `FuncKind` and `TypeKind`.
  </details>

- **[silent acceptance of `&t.M` method value]** [`preprocess.go:4405-4409`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4405-L4409) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4405-L4409) — `*SelectorExpr` recurses on `cx.X` (the receiver) without inspecting the selector itself; taking the address of a bound method value escapes.
  <details><summary>details</summary>

  `t := T{x: 42}; p := &t.M; println((*p)())` prints `42` under PR #5609 — adversarial test [`addressable_gap_method_value.gno`](tests/addressable_gap_method_value.gno) demonstrates. The receiver `t` is addressable (a name), so `isAddressable` returns true for the selector regardless of whether `M` is a field or a method. Go: `invalid operation: cannot take address of t.M (value of type func() int)`. Same root cause as `&funcName` — the static type of the selector result (FuncKind) is never consulted. Fix: after the PointerKind early-return and before recursing, check `evalStaticTypeOf(store, last, cx).Kind()` (the full selector, not the base) and return `false` for `FuncKind`.
  </details>

- **[silent acceptance of `&s[i]` string byte]** [`preprocess.go:4397-4404`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4397-L4404) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4397-L4404) — IndexExpr's `Kind()` switch only handles `MapKind` (false) and `SliceKind` (true); `StringKind` falls through to `isAddressable(cx.X)`, returning true if the string is held in a name.
  <details><summary>details</summary>

  `s := "abc"; p := &s[0]; println(*p)` prints `97` under PR #5609 — adversarial test [`addressable_gap_string_index.gno`](tests/addressable_gap_string_index.gno) demonstrates. Go: `invalid operation: cannot take address of s[0] (value of type byte)`. Note the existing [`addressable_4a_err.gno`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/tests/files/addressable_4a_err.gno) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/tests/files/addressable_4a_err.gno) only carries `// TypeCheckError:`, confirming Gno preprocess has no check today — the PR didn't change this. Fix: add `case StringKind: return false` in the IndexExpr `Kind()` switch.
  </details>

- **[error path nil-deref for `&nil`]** [`preprocess.go:2347`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2347) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2347) — `xt.String()` panics with `nil pointer dereference` when the operand is untyped `nil`.
  <details><summary>details</summary>

  `_ = &nil` produces `runtime error: invalid memory address or nil pointer dereference` in the panic message instead of a clean preprocess error. `evalStaticTypeOf(store, last, n.X)` returns `nil` for untyped `nil`, and the new panic format string at line 2347 calls `xt.String()` unconditionally. Pre-existing pathology — master crashes too, but at a different point (`ATTR_REF_ELEM_TYPE not set during preprocessing` at the attribute-write site) — so this PR moves the crash, doesn't introduce it. Both fail loudly to the test harness, neither produces a user-readable message. Fix: guard `xt == nil` before `xt.String()` (return early with `"invalid operation: cannot take address of nil"` to match Go).
  </details>

## Nits

- [`preprocess.go:4393`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4393) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4393) — `isAddressable` has no doc comment; the Go-spec correspondence and the load-bearing assumption "constant names are already `*ConstExpr` by the time we run" should be stated so a future reader doesn't add `case *ConstExpr` redundantly.

- [`preprocess.go:4397-4404`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4397-L4404) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4397-L4404) — the IndexExpr `Kind()` switch has no `PointerKind` case, which is correct (it'd be unreachable) but the reason is non-obvious: the IndexExpr's X is rewritten to `*StarExpr{X: n.X}` at [`preprocess.go:2127-2134`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2127-L2134) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2127-L2134) when the base is a pointer, so `Kind()` here is never PointerKind. A one-line comment in `isAddressable` ("pointer-to-array index already rewritten to `*StarExpr`") would prevent a future refactor from accidentally regressing.

- Error messages still expose internal representation: `n.X.String()` produces `getPtr<VPBlock(3,2)>()`, `(const-type struct{})`, `(const (1 int))` — pre-existing, not this PR's fault, but every new error string inherits it.

## Missing Tests

- **[follow-up gap, no preprocess assertion]** [`addressable_4a_err.gno`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/tests/files/addressable_4a_err.gno) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/tests/files/addressable_4a_err.gno) — `&str[i]` only has `// TypeCheckError:`, no `// Error:` annotation, confirming Gno preprocess silently accepts it. See `addressable_gap_string_index` finding above.

- **[follow-up gap, no test at all]** `&funcName` — no `addressable_*_err.gno` covers this. See `addressable_gap_funcname` finding above.

- **[follow-up gap, no test at all]** `&t.M` (method value) and `&pkg.Func` (package-qualified function) — no `addressable_*_err.gno` covers either. See `addressable_gap_method_value` finding above.

## Suggestions

- Land this PR as-is, then file a follow-up tracking issue (the author's PR comment already commits to one for `x++`, compound assigns, `m["k"][0] = 9`, etc.) and add `FuncKind`/`TypeKind` exclusion to `*NameExpr` and `StringKind` exclusion to `*IndexExpr` so the helper matches Go-spec addressability rather than "addressable in the cases this PR cared about".

## Questions for Author

- Was `*NameExpr → true` deliberately under-constrained to keep `&someStructTypeVar` working (struct-valued names should be addressable) without paying for an `evalStaticTypeOf` call? A `FuncKind`/`TypeKind` exclusion adds one type lookup per name reference inside a `&` operand — likely cheap, worth confirming.
- The author's PR-thread comment lists `x++`/`getVal() += 1`/`m["k"][0] = 9` as scope-outs for a follow-up. Is there an open issue or just the PR-thread comment? If only the latter, the gap risks decaying invisibly once #5586 is closed.
