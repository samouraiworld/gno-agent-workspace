# PR #5609: fix(gnovm): validate addressability of & operand at preprocess stage

URL: https://github.com/gnolang/gno/pull/5609
Author: aronpark1007 | Base: master | Files: 27 | +135 -26
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `443885d1b` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5609 443885d1b`

**TL;DR:** Gno used to let you write `&expr` for things that have no real address (a map element, a type-assertion result, a temporary). This PR adds a preprocess-stage check that rejects most of those with a clean Go-style error instead of a late runtime crash.

**Verdict: REQUEST CHANGES** — Two blockers: the branch conflicts with master (not mergeable), and the new `isAddressable` helper treats a composite literal as addressable in every position, so `&[3]int{…}[0]`, `&struct{…}{}.x`, and `[3]int{…}[:]` still reach runtime where Go rejects them. [@thehowl](https://github.com/gnolang/gno/pull/5609#pullrequestreview-4389750223) listed these plus `&s[i]` (string) and `&t.M` (method value) on 2026-05-29; the head has not moved since, so none are addressed. The cleanup direction is right and notJoon approved an earlier read; this is about finishing the helper and rebasing, not redirecting.

Not superseded: 5609 is the only open PR for #5586. The RefExpr work it builds on (#5474, #5491) is already merged; #5480 was the closed alternative. No competing addressability PR is open.

## Summary

Gno's preprocess stage never validated the `&` operand's addressability: non-addressable forms either silently produced a usable pointer at runtime (`&m[k]`, `&i.(S).a`) or crashed late with an opaque `illegal assignment ... *gnolang.CallExpr` message. The PR adds `isAddressable(store, last, x)` and calls it from `TRANS_LEAVE *RefExpr` and, for arrays, from `TRANS_LEAVE *SliceExpr`. The check is a pure-AST recursion mirroring Go's spec; everything unhandled falls into `default → false`. It correctly closes the binary/unary, type-assert, map-index, call-result, and named-array-slice classes. Six classes still flow through to runtime that Go rejects at compile time: composite-literal selector/index/slice bases, string-byte index, method value, function name, untyped `nil` (which crashes the preprocessor), and pointer-receiver method calls on non-addressable receivers (the check is never wired into that site).

## Glossary

- `isAddressable(store, last, x)` — new preprocess helper at `preprocess.go:4393`; recursive structural walk over `Expr`.
- `*RefExpr` — AST node for `&x`. TRANS_LEAVE fires bottom-up after children are processed.
- `*CompositeLitExpr` — AST node for a composite literal (`T{...}`).
- `evalStaticTypeOf` — preprocess type lookup; returns the resolved `Type` for an `Expr` from the surrounding block.

## Fix

`TRANS_LEAVE *RefExpr` previously rejected only multi-value calls; everything else flowed to `doOpRef`, which builds a `*PointerType` regardless of whether the operand has an address. The PR gates that path with `isAddressable` at [`preprocess.go:2344-2348`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2344-L2348) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2344-L2348) and reuses it from the slice path at [`preprocess.go:2165-2170`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2165-L2170) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2165-L2170) to reject `arr[:]` over an unaddressable array. The helper itself, [`preprocess.go:4393-4413`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4393-L4413) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4393-L4413), returns `true` for name/deref/composite, switches on `cx.X.Kind()` for index (Map → false, Slice → true, else recurse), and for selector returns true on a pointer base (else recurse). The load-bearing trick: by the time `*RefExpr` leaves, constant names are already `*ConstExpr` and pointer-indexed arrays are already rewritten to `*StarExpr`, so both land on the right arm.

## Critical (must fix)

None.

## Warnings (should fix)

- **[branch conflicts with master, not mergeable]** [PR #5609](https://github.com/gnolang/gno/pull/5609) — `gh pr view` reports `mergeable: CONFLICTING`, `mergeStateStatus: DIRTY`.
  <details><summary>details</summary>

  The branch must be merged/rebased onto master before it can land. Per the prior round and morgan's note, the conflict is mechanical (the `RefExpr`/`SliceExpr` structure it hooks into is unchanged upstream), not a logic clash. Fix: rebase on master and resolve.
  </details>

- **[composite literal accepted as address base, so `&[3]int{…}[0]` / `&struct{…}{}.x` / `[3]int{…}[:]` reach runtime]** [@thehowl](https://github.com/gnolang/gno/pull/5609#pullrequestreview-4389750223) — [`preprocess.go:4395`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4395) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4395) — `*CompositeLitExpr` returns `true` unconditionally, but a composite literal is addressable in Go only as the direct operand of `&` (`&T{}`), not as the base of a selector, index, or slice.
  <details><summary>details</summary>

  When the literal is a selector/index/slice base, the `*SelectorExpr` and `*IndexExpr` arms recurse into it and the slice arm calls `isAddressable` on it; all three then report `true`. Verified on 443885d1b: `&[3]int{1,2,3}[0]` prints `1`, `&struct{ x int }{}.x` prints `0`, `[3]int{1,2,3}[:]` runs with `len 3` — Go rejects all three (`cannot take address of …` / `cannot slice unaddressable value`). Adversarial filetests [`addressable_gap_complit_index.gno`](tests/addressable_gap_complit_index.gno), [`addressable_gap_complit_selector.gno`](tests/addressable_gap_complit_selector.gno), [`addressable_gap_complit_slice.gno`](tests/addressable_gap_complit_slice.gno) assert the current `// Output:` and carry the post-fix `// TypeCheckError:`. Fix: treat a composite literal as addressable only as the direct `&` operand; a `*CompositeLitExpr` reached by recursion (selector/index/slice base) is not addressable.
  </details>

- **[pointer-method call on a non-addressable receiver skips the check]** [`preprocess.go:2404`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2404) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2404) — calling a pointer-receiver method on a value auto-takes the receiver's address, but the synthesized `&RefExpr` is marked `setPreprocessed`, so the `TRANS_LEAVE *RefExpr` gate never runs `isAddressable` on it.
  <details><summary>details</summary>

  The receiver-address synthesis at [`preprocess.go:2404`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2404) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2404) and [`preprocess.go:2432`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2432) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2432) quotes the Go spec's "If x is addressable" precondition in its comment but never enforces it. Verified on 443885d1b: `m["k"].Inc()` (pointer method, map element) and `T{1}.Inc()` (composite literal) both run; the map case even mutates the stored element (prints `2`). Go rejects both: `cannot call pointer method Inc on T`. Adversarial filetest [`addressable_gap_ptr_method_recv.gno`](tests/addressable_gap_ptr_method_recv.gno). This is the same check missing at a third site, not a separate bug class. Fix: gate the synthesis on `isAddressable(store, last, n.X)` at 2404 and 2432.
  </details>

- **[`&s[i]` on a string accepted]** [@thehowl](https://github.com/gnolang/gno/pull/5609#pullrequestreview-4389750223) — [`preprocess.go:4397-4404`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4397-L4404) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4397-L4404) — the IndexExpr `Kind()` switch handles `MapKind` (false) and `SliceKind` (true) only; `StringKind` falls through to `isAddressable(cx.X)`, true for a name.
  <details><summary>details</summary>

  Verified on 443885d1b: `s := "abc"; p := &s[0]; println(*p)` prints `97`. Go: `invalid operation: cannot take address of s[0] (value of type byte)`. Adversarial filetest [`addressable_gap_string_index.gno`](tests/addressable_gap_string_index.gno). Fix: add `case StringKind: return false` to the IndexExpr `Kind()` switch.
  </details>

- **[`&t.M` method value accepted]** [@thehowl](https://github.com/gnolang/gno/pull/5609#pullrequestreview-4389750223) — [`preprocess.go:4405-4409`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4405-L4409) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4405-L4409) — `*SelectorExpr` recurses on the receiver without inspecting the selector itself, so a bound method value is treated like a field.
  <details><summary>details</summary>

  Verified on 443885d1b: `t := T{x:42}; p := &t.M; println((*p)())` prints `42`. Go: `invalid operation: cannot take address of t.M (value of type func() int)`. Adversarial filetest [`addressable_gap_method_value.gno`](tests/addressable_gap_method_value.gno). Fix: when the selector's static type is `FuncKind`, return `false`.
  </details>

- **[`&funcName` accepted for user functions]** [`preprocess.go:4395`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4395) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4395) — `*NameExpr` returns `true` unconditionally, so a package-level function name builds a usable `*func()`. Fix: return `false` for `FuncKind`/`TypeKind`.
  <details><summary>details</summary>

  `p := &bar` (filetest [`addressable_gap_funcname.gno`](tests/addressable_gap_funcname.gno), full [repro](comment_claude-opus-4-8.md)):

  ```
  $ gno run  # func bar(){}; p := &bar; _ = p; println("ok")
  ok
  # Go: invalid operation: cannot take address of bar (value of type func())
  ```
  </details>

- **[`&nil` crashes the preprocessor with nil-deref]** [`preprocess.go:2347`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L2347) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2347) — `evalStaticTypeOf` returns nil for untyped `nil` and `xt.String()` dereferences it, so the error path crashes instead of erroring cleanly. Pre-existing (master also crashes, elsewhere). Fix: guard `xt == nil` and return `cannot take address of nil`.
  <details><summary>details</summary>

  `_ = &nil` (full [repro](comment_claude-opus-4-8.md)):

  ```
  panic: runtime error: invalid memory address or nil pointer dereference
      --- preprocess stack ---
      stack 2: func main() { _<VPInvalid(0)> = &((const (undefined))) }
  ```
  </details>

## Nits

- [`preprocess.go:4393`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/preprocess.go#L4393) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4393) — `isAddressable` has no doc comment; state the Go-spec correspondence and the load-bearing assumption "constant names are already `*ConstExpr` and pointer-indexed arrays already `*StarExpr` by the time this runs" so a future reader doesn't add a redundant `case *ConstExpr`.
- Error messages still expose internal representation (`getPtr<VPBlock(3,2)>()`, `(const-type struct{})`) — pre-existing, but every new error string inherits it.

## Missing Tests

- **[composite-literal address bases]** no `addressable_*_err.gno` covers `&[3]int{…}[0]`, `&struct{…}{}.x`, or `[3]int{…}[:]`. See the composite-literal Warning; adversarial filetests written.
- **[string index / method value / funcName]** carried from round 2 — no `_err.gno` asserts a preprocess rejection for any. Adversarial filetests written for each.
- **[pointer-method on non-addressable receiver]** no `addressable_*_err.gno` covers `m["k"].Inc()` / `T{1}.Inc()`. See the pointer-method Warning; adversarial filetest [`addressable_gap_ptr_method_recv.gno`](tests/addressable_gap_ptr_method_recv.gno) written.

## Suggestions

- Land the cleanup, but close the five Go-rejected classes in this PR (per morgan's table) rather than deferring, since the helper now looks complete and the gap would decay invisibly once #5586 is marked fixed. The fixes are local to `isAddressable`: composite-literal-only-as-direct-operand, `StringKind → false`, selector/name `FuncKind`/`TypeKind → false`.

- Reuse the existing lvalue classifier. [`type_check.go:1019`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/type_check.go#L1019) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/type_check.go#L1019) already has `assertValidAssignLhs`, which walks the same node kinds to decide assignment-target validity and already guards `xt != nil` and rejects a `StringKind` index — the two cases `isAddressable` regresses on. The two aren't interchangeable (a map element is assignable but not addressable; a composite literal is addressable via `&T{}` but not assignable), but `isAddressable` should adopt those two arms verbatim and the pair should live beside each other; the `// TODO: star, addressable` at [`type_check.go:33`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/type_check.go#L33) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/type_check.go#L33) marks the type-checker as the intended home for this check.

## Open questions

- Was `*NameExpr → true` left broad to avoid an `evalStaticTypeOf` call per name inside a `&` operand? The `FuncKind`/`TypeKind` exclusion adds one type lookup; likely cheap. Not posted — design confirmation, no behavior decision required in-band.
- Is there a tracking issue for the author's other scope-outs (`x++`, compound assigns, `m["k"][0] = 9`)? Only the PR thread mentions them; risk of invisible decay once #5586 closes. Not posted unless the author marks #5586 fixed.
