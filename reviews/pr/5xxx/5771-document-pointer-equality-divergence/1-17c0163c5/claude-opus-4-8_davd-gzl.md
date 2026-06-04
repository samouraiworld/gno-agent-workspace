# PR #5771: docs(gnovm): document pointer equality divergence for zero-sized types

URL: https://github.com/gnolang/gno/pull/5771
Author: ltzmaxwell | Base: master | Files: 5 | +65 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `17c0163c5` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5771 17c0163c5`
Related: [PR #5708](../../5708-zerobase-zero-sized-types/2-e94b9079/claude-opus-4-7_davd-gzl.md) — the abandoned "implement zerobase folding" approach this PR supersedes by documenting the divergence instead.

**Verdict: APPROVE** — purely additive docs + cross-reference comments. Every row of both the Gno and gc-Go tables verified empirically against Gno master and gc-Go (go1.26.3); the code comments match the source they point at. No bugs, no behavior change, no ADR needed (docs-only).

## Summary

Documents that Gno's pointer `==` is uniform `(Base, Index)` identity for every element type: `new(T)` and `&CompositeLit{}` each mint a fresh `*HeapItemValue`, so two distinct allocations always differ in `Base` and compare `false`, including for zero-sized `T`. gc-Go diverges: zero-sized allocations that escape to the heap fold onto `runtime.zerobase` (so `new(struct{}) == new(struct{})` can be `true`), and `&a[i]`/`&s.f` on zero-sized elements collapse to the base address. The PR adds a "Pointer equality for zero-sized types" section to `go-gno-compatibility.md`, a note to `gno-memory-model.md`, and cross-reference comments at `PointerValue`, `doOpRef`, and the `new` builtin.

This resolves the round-2 `NEEDS DISCUSSION` on #5708: rather than replicate `runtime.zerobase` (which broke pointer identity across tx boundaries and leaked HIV ownership across realms), the team keeps uniform identity and documents the gc-Go difference. `grep` confirms no `zerobase` cache survives in master — [`uverse.go:1164`](https://github.com/gnolang/gno/blob/17c0163c5/gnovm/pkg/gnolang/uverse.go#L1164) · [↗](../../../../../.worktrees/gno-review-5771/gnovm/pkg/gnolang/uverse.go#L1164) and [`values.go:190`](https://github.com/gnolang/gno/blob/17c0163c5/gnovm/pkg/gnolang/values.go#L190) · [↗](../../../../../.worktrees/gno-review-5771/gnovm/pkg/gnolang/values.go#L190) are the only mentions, both in this PR's comments.

## Verification

The load-bearing content is the two comparison tables. I confirmed every row.

Gno (filetest, all rows match the documented `// Output`):

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5771 -R gnolang/gno
cat > gnovm/tests/files/zz_ptreq.gno <<'EOF'
package main

type T struct{}

func main() {
	println("new==new:", new(T) == new(T))
	var x, y T
	println("&x==&y:", &x == &y)
	var a [10]T
	println("&a0==&a1:", &a[0] == &a[1])
	var s struct{ a, b T }
	println("&sa==&sb:", &s.a == &s.b)
	println("&x==&x:", &x == &x)
	println("&a3==&a3:", &a[3] == &a[3])
	p := &x
	q := p
	println("p==q:", p == q)
}

// Output:
// new==new: false
// &x==&y: false
// &a0==&a1: false
// &sa==&sb: false
// &x==&x: true
// &a3==&a3: true
// p==q: true
EOF
go test -run 'TestFiles/zz_ptreq.gno$' -test.short ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zz_ptreq.gno
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.040s
```

gc-Go (go1.26.3), confirming the escape-dependence and offset-arithmetic claims:

```
new==new inplace: false      # non-escaping → distinct (doc: unspecified*, false here)
new==new escaped: true       # both escape → runtime.zerobase (doc: true only when escaped)
&x==&y inplace: false
&x==&y escaped: true
&a0==&a1: true               # offset arithmetic, unconditional
&sa==&sb: true               # offset arithmetic, unconditional
&x==&x: true
&a3==&a3: true
```

The `new(T) == new(T)` row is the one most docs get wrong (commonly stated as unconditionally `true`); this PR correctly marks it escape-dependent and footnotes why. The `*` footnote distinguishing escape-folding (heap rows) from offset-collapse (array/struct rows) is accurate: the offset rows are address-identical regardless of allocation site.

The `values.go` comment's reduction — whole-struct `lv.V == rv.V` "reduces to same `Base` + same `Index`" — is sound: the struct `==` also compares the `TV` pointer, but `TV` is canonical per `(Base, Index)`, confirmed by the `&a[3]==&a[3]` and `p==q` rows returning `true` (a non-canonical `TV` would make same-slot pointers compare unequal). `PointerKind` falls through to `return lv.V == rv.V` at [`op_binary.go:567`](https://github.com/gnolang/gno/blob/17c0163c5/gnovm/pkg/gnolang/op_binary.go#L567) · [↗](../../../../../.worktrees/gno-review-5771/gnovm/pkg/gnolang/op_binary.go#L567); the `new` builtin mints a fresh `HeapItem` per call at [`uverse.go:1180`](https://github.com/gnolang/gno/blob/17c0163c5/gnovm/pkg/gnolang/uverse.go#L1180) · [↗](../../../../../.worktrees/gno-review-5771/gnovm/pkg/gnolang/uverse.go#L1180). Both comments match.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`go-gno-compatibility.md:88`](https://github.com/gnolang/gno/blob/17c0163c5/docs/resources/go-gno-compatibility.md#L88) · [↗](../../../../../.worktrees/gno-review-5771/docs/resources/go-gno-compatibility.md#L88) — the `new(T) == new(T)` row tags gc-Go as `unspecified*` while the next row (`&x == &y`) is also `unspecified*` but its parenthetical says "true only when escaped" vs. the first row's "true only when both escape → runtime.zerobase". The two are the same mechanism; using identical phrasing on both rows (or just `escaped` on the second) would remove the momentary "are these different?" pause. Cosmetic.

## Missing Tests

None required for a docs PR. Optional but high-value: the verification filetest above pins every documented Gno row as executable `// Output`, so a future change to pointer-equality semantics would fail loudly instead of silently drifting from this doc. Worth landing alongside the docs if the author wants the table guarded.

## Suggestions

- The new `go-gno-compatibility.md` section links to `gno-memory-model.md` and vice versa, and both code comments point to `values.go`/the compat doc. Consider also linking the compat section back to the Go spec's exact wording inline where "unspecified" first appears (it currently only appears in the Rationale block at the bottom) so a reader landing mid-section sees the spec basis without scrolling. Minor.

## Questions for Author

None.
