# PR #5754: fix(gnolang): allow local type declarations in block statements

URL: https://github.com/gnolang/gno/pull/5754
Author: davd-gzl | Base: master | Files: 2 | +150 -4
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 0b36de2c0 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5754 0b36de2c0`

**Verdict: APPROVE** — minimal, behavior-preserving fix for a real preprocess panic; the new `default` branch keeps the existing func-local behavior and extends it to block parents, TypeID disambiguation is sound (block spans are unique), and the only untested line is a documented unreachable guard. No correctness, determinism, or state-safety concerns.

## Summary
Declaring a named type inside any block other than a function body (`{ type T int }`, a `for`/`if`/`switch` block) panicked during preprocess with `expected type expr but got *gnolang.BlockStmt`. [`declareWith`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/types.go#L1517) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/types.go#L1517) only computed a disambiguating `ParentLoc` for `*FuncDecl`/`*FuncLitExpr` parents and `panic`ed on every other block node. The fix replaces the panic with a `default` branch that reads the parent block's location (already assigned by `setNodeLocations`), so two same-named local types in sibling blocks get distinct `TypeID`s instead of crashing. Adds `bigmap.gno` (Go's `test/bigmap.go`) as a regression filetest.

## Glossary
- `declareWith` — builds an unsealed `*DeclaredType` for a type decl, stamping `ParentLoc` for disambiguation.
- `ParentLoc` — `Location` folded into a declared type's `TypeID` so identically-named local types in different scopes don't collide.
- `setNodeLocations` — preprocess pass that stamps every `BlockNode` with a unique `Location` (from its source `Span`) before predefine runs.
- `TypeID` — string identity of a type; for local types it is `pkgPath[parentLoc].name`.

## Fix
Before: the `switch parent.(type)` had a `case *FuncDecl, *FuncLitExpr` setting `ploc = parent.GetLocation()` and a `default` that panicked. Block-local types have a `*BlockStmt`/`*ForStmt`/`*IfCaseStmt`/... parent, so they hit the panic. After: the func cases are folded into `default`, which reads `parent.GetLocation()` for any non-package/file block node and guards against a zero location with a loud panic. See [`types.go:1517-1542`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/types.go#L1517-L1542) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/types.go#L1517-L1542). The load-bearing constraint: `setNodeLocations` ([`preprocess.go:742`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/preprocess.go#L742) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/preprocess.go#L742)) runs before predefine and stamps every block with a `Location` built from its full `Span` (start+end line/col, [`preprocess.go:6237-6242`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/preprocess.go#L6237-L6242) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/preprocess.go#L6237-L6242)), so sibling blocks always have distinct, non-zero locations.

## Correctness notes
- Func-local types are unchanged: previously `case *FuncDecl, *FuncLitExpr` set `ploc = parent.GetLocation()`; now they fall into `default`, which does the same `parent.GetLocation()`. Same value, same `TypeID`. Verified the only two callers pass the immediate enclosing block as `parent` ([`preprocess.go:3066`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/preprocess.go#L3066) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/preprocess.go#L3066), [`preprocess.go:5542`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/preprocess.go#L5542) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/preprocess.go#L5542)).
- `ParentLoc` feeds `TypeID` only through `DeclaredTypeID` ([`types.go:2011-2017`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/types.go#L2011-L2017) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/types.go#L2011-L2017)) and `String()` ([`types.go:2019-2025`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/types.go#L2019-L2025) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/types.go#L2019-L2025)). A non-zero block location switches both to the `pkgPath[loc].name` form, which is exactly the disambiguation wanted. No other consumer of `ParentLoc` exists in the package.
- Determinism: `Location` is derived from source span, deterministic across runs. No new map iteration or pointer-identity dependence introduced.

## Verified behavior
Block-local declarations across `for`, sibling, and nested-shadowing scopes preprocess without panic and keep distinct identities. Reproduced with the PR's own test plus three ad-hoc filetests (sibling `T int`/`T string`, per-iteration `for` type, nested shadowing) — all pass on the PR head.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5754 -R gnolang/gno
cat > gnovm/tests/files/zz_localtype_probe.gno <<'EOF'
package main

func main() {
	for i := 0; i < 2; i++ {
		type L [1]byte
		m := make(map[L]int)
		m[L{byte(i)}] = i
		if m[L{byte(i)}] != i {
			panic("bad")
		}
	}
	var a, b interface{}
	{ type T int; a = T(1) }
	{ type T string; b = T("x") }
	_, okA := a.(int)
	_, okB := b.(string)
	println("loop-ok", okA, okB)
}

// Output:
// loop-ok false false
EOF
go test -run 'TestFiles/zz_localtype_probe.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zz_localtype_probe.gno
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.153s
```

(`okA`/`okB` are `false`: the named local `T` is distinct from `int`/`string`, confirming the local types are not collapsed onto their underlying types and the two sibling `T`s do not collide.)

## CI
Two red checks, both unrelated to this diff:
- `Merge Requirements` — the github-bot gate ("pending initial approval"), not a code failure. Reviewer [@Villaquiranm](https://github.com/gnolang/gno/pull/5754) already left an APPROVED review after self-resolving an initial question.
- `scenario-01-four-validators-reset-three` — a consensus/e2e scenario unrelated to a GnoVM preprocess change; flaky in this run while scenarios 02-18 all pass.

The local `TestFiles -test.short` run shows 10 failures (`redeclaration3/4`, `redeclaration_global1`, `switch13`, `type41`, `types/{add,and,eql_0b4,eql_0f0,or}_f0`), all type-checker error-message format diffs that reproduce identically on clean `master` (`4c9de5225`). Pre-existing, not introduced here.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`gnovm/tests/files/bigmap.gno:1-37`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/tests/files/bigmap.gno#L1-L37) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/tests/files/bigmap.gno#L1-L37) — the first half (`seq`/`cmp` + `map[int][1000]byte`) exercises large map values, orthogonal to the block-local-type bug in #5662. It is a faithful port of Go's corpus and adds coverage, so keeping it whole is fine; just noting the regression target is only the nine `{ type T ...; type V ... }` blocks.

## Missing Tests
- **[defensive branch uncovered]** [`gnovm/pkg/gnolang/types.go:1528-1532`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/types.go#L1528-L1532) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/types.go#L1528-L1532) — the `if ploc.IsZero()` panic is the line Codecov flags (33% patch coverage).
  <details><summary>details</summary>

  The branch is documented as unreachable: `setNodeLocations` guarantees every block node has a non-zero location before predefine runs, and `SaveBlockNodes` ([`preprocess.go:6277`](https://github.com/gnolang/gno/blob/0b36de2c0/gnovm/pkg/gnolang/preprocess.go#L6277) · [↗](../../../../../.worktrees/gno-review-5754/gnovm/pkg/gnolang/preprocess.go#L6277)) panics independently if any block location is zero. There is no clean way to construct a zero-location block from a `.gno` filetest, so the gap is acceptable for a fail-loud guard. No action required.
  </details>

## Suggestions
None.

## Questions for Author
None.
