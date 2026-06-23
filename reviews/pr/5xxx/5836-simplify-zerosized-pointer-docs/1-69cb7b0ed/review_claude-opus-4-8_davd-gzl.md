# PR #5836: docs(gnovm): simplify zero-sized pointer equality docs

URL: https://github.com/gnolang/gno/pull/5836
Author: davd-gzl | Base: master | Files: 5 | +19 -58
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `69cb7b0ed` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5836 69cb7b0ed`

**TL;DR:** Follow-up to #5771. That PR documented why Gno and Go disagree on whether two pointers to distinct zero-sized variables (like `&struct{}{}`) are equal, with a detailed comparison table in the compatibility doc. This PR shrinks that table to a one-line pointer and moves the short explanation into the memory-model doc, and rewords the three cross-reference code comments. No behavior changes.

**Verdict: APPROVE** — net docs simplification, technically accurate; the three code comments still match the code they point at and the rewritten example reproduces on `69cb7b0ed`. One open editorial thread: [@ltzmaxwell](https://github.com/gnolang/gno/pull/5836#discussion_r2479446073) (the #5771 author) wants the detailed compatibility section kept rather than collapsed; that is a placement-of-detail call between contributors, not a code defect, and is the only thing to settle before merge.

## Summary
#5771 added a "Pointer equality for zero-sized types" section to `go-gno-compatibility.md` (a Gno-vs-gc-Go comparison table plus a rationale citing the Go spec), a short note in `gno-memory-model.md`, and cross-reference comments at `PointerValue`, `doOpRef`, and the `new` builtin. This PR keeps the compatibility section as a two-line summary that links to the memory model, expands the memory-model note from a back-reference into the self-contained explanation with a runnable example, and rewords the three code comments to drop the `runtime.zerobase` framing while still pointing at `PointerValue`. The behavioral content is unchanged from #5771: Gno pointer `==` is uniform `(Base, Index)` identity, so two distinct zero-sized allocations always compare `false`.

## What the PR drops
The previous compatibility section carried the full gc-Go divergence detail: the escape-dependent `runtime.zerobase` folding (heap-escaped zero-sized allocations collapse to one shared address), the offset-arithmetic collapse (`&a[i]` / `&s.f` on zero-sized elements address-identical), and the Go-spec citation that makes the divergence "unspecified" rather than a Gno bug. The new text states only that Gno is "never equal" where "Go may report them equal" and no longer says when or why gc-Go differs. This is the substance of the open reviewer thread (see Open questions) — the simplification is defensible for a user-facing doc, but the dropped detail was the part that justified Gno's choice as spec-compliant rather than arbitrary.

## Verification

The only behavioral claim the new docs assert is the memory-model example. It reproduces on `69cb7b0ed`, and the simplified "equal only when they point to the same variable" wording holds at the edges (aliased same variable `true`, distinct array elements `false`, same element twice `true`):

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5836 -R gnolang/gno
cat > gnovm/tests/files/zz_ptreq_5836.gno <<'EOF'
package main

func main() {
	a, b := struct{}{}, struct{}{}
	println("&a==&b:", &a == &b)        // doc example: false in Gno
	var x int
	println("&x==&x:", &x == &x)        // same variable: true
	var arr [3]int
	println("&arr0==&arr1:", &arr[0] == &arr[1]) // distinct elements: false
	println("&arr1==&arr1:", &arr[1] == &arr[1]) // same element: true
}

// Output:
// &a==&b: false
// &x==&x: true
// &arr0==&arr1: false
// &arr1==&arr1: true
EOF
go test -run 'TestFiles/zz_ptreq_5836.gno$' -test.short ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zz_ptreq_5836.gno
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.038s
```

The three reworded code comments each still describe the code they annotate, confirmed by reading the source at the PR head:

- [`values.go:187-189`](https://github.com/gnolang/gno/blob/69cb7b0ed/gnovm/pkg/gnolang/values.go#L187-L189) · [↗](../../../../../.worktrees/gno-review-5836/gnovm/pkg/gnolang/values.go#L187) — "Equality (isEql) reduces to (Base, Index)". `PointerKind` falls through to whole-struct `lv.V == rv.V` in `isEql`; since `TV` is canonical per `(Base, Index)`, this reduces to same Base + same Index, as the #5771 review confirmed empirically.
- [`op_expressions.go:198-199`](https://github.com/gnolang/gno/blob/69cb7b0ed/gnovm/pkg/gnolang/op_expressions.go#L198-L199) · [↗](../../../../../.worktrees/gno-review-5836/gnovm/pkg/gnolang/op_expressions.go#L198) — "No size-dependent path here". `doOpRef` takes the element type from `ATTR_REF_ELEM_TYPE` and builds the pointer with no branch on element size; the body confirms it.
- [`uverse.go:1169-1171`](https://github.com/gnolang/gno/blob/69cb7b0ed/gnovm/pkg/gnolang/uverse.go#L1169-L1171) · [↗](../../../../../.worktrees/gno-review-5836/gnovm/pkg/gnolang/uverse.go#L1169) — "new(T) allocates a fresh *HeapItemValue per call". The builtin calls `NewHeapItem` then returns a `PointerValue{Base: hi, Index: 0}`, so each call yields a distinct Base.

This PR changes only comments and prose: every changed line in all three Go files is a comment line (diffed against the PR's parent `e2f91d6da`), so no test or invariant is in scope beyond the doc accuracy checked above.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

None.

## Missing Tests

None required. As with #5771, the pointer-equality rows are not pinned by an executable filetest in the tree, so a future semantics change would silently drift from these docs. The #5771 review already flagged this as optional-but-high-value; this PR doesn't change that posture.

## Suggestions

None.

## Open questions

- [@ltzmaxwell](https://github.com/gnolang/gno/pull/5836#discussion_r2479446073) (the #5771 author) argues for keeping the detailed compatibility section: it explains the memory-model design (why Gno differs from gc-Go while staying Go-spec-compliant), and readers typically land on the compatibility doc first and follow the link for depth. The counter-position (this PR's) is that the compatibility doc should stay user-facing and short, with the depth in the memory model. Both are reasonable; it's a doc-placement judgment for the two contributors to settle. Not posted as a blocking finding because nothing in the code or the retained prose is wrong — surfaced here so the resolution is visible. Possible middle ground: keep the one-line compatibility summary but restore the gc-Go divergence detail (escape-folding + offset-collapse + spec citation) in the memory-model section it now points to, so no information is lost, just relocated.
