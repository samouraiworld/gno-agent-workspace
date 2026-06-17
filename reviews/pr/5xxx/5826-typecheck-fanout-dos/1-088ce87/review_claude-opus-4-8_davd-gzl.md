# PR #5826: fix(gnovm): bound type-expansion fan-out before go/types validType

URL: https://github.com/gnolang/gno/pull/5826
Author: ltzmaxwell | Base: master | Files: 4 | +407 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `088ce87` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5826 088ce87`

**TL;DR:** Deploying a Gno package on gno.land runs Go's type checker on it, unmetered, before any gas is charged. Go's checker has a known trap: certain ~40-line type definitions make it do exponential work and effectively hang, which would freeze a block-producing node. This PR adds a fast pre-check that counts how much work each type would cause and rejects the package if it is too much.

**Verdict: REQUEST CHANGES** — the value-containment fan-out case is correctly fixed, but the same exponential `validType` DoS is still reachable through a generic-instantiation fan-out, which the guard counts as a constant and lets through ([typecheck_bound.go:143-146](https://github.com/gnolang/gno/blob/088ce87/gnovm/pkg/gnolang/typecheck_bound.go#L143-L146) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L143)).

## Summary
`TypeCheckMemPackage` runs `go/types` on every deployed package at `MsgAddPackage` / `MsgRun` time, before metering ([keeper.go:647](https://github.com/gnolang/gno/blob/088ce87/gno.land/pkg/sdk/vm/keeper.go#L647) · [↗](../../../../../.worktrees/gno-review-5826/gno.land/pkg/sdk/vm/keeper.go#L647)). `go/types`' `validType` walk follows value-containment edges, does not memoize (disabled for golang/go#65711), and is exponential on a "doubling" chain (`type Tn struct{ a, b [0]T{n-1} }`): a ~40-type package costs ~2⁴⁰ node visits and hangs the checker, a consensus DoS. The PR adds `checkTypeExpansionBound`, a STEP 3.5 pre-pass that computes the same node count *with* memoization (so it is linear) and rejects any package whose count exceeds 1_000_000. The guard models `validType`'s edge set well for the direct-value case and is correctly linear, but its `*ast.IndexExpr` arm drops the type argument, so a fan-out routed through a generic instantiation is undercounted and reopens the same hang.

```
 honest type            value fan-out (FIXED)      generic fan-out (STILL HANGS)
 T = struct{a,b int}    Tn = struct{a,b [0]T{n-1}}  An = struct{ x W[A{n-1}] }
 cost grows linearly    cost doubles per level      W[X] counted as cost(W) only,
 → accepted             → guard rejects (1.8M>1M)   constant → guard accepts → validType 2^n
```

## Glossary
- **`validType`** — `go/types` pass that rejects infinitely-expanding value types; the exponential, unmemoized walk this PR defends against.
- **value-containment edge** — a type embedding another *by value* (struct field, array element, named RHS); pointers/slices/maps/chans break it. The only edges `validType` recurses through.
- **fan-out / doubling chain** — each level references the previous one twice by value, so node count doubles per level: the worst case for an unmemoized walk.
- **STEP 3.5** — the new guard's slot in `typeCheckMemPackage`, after Go parsing (STEP 3), before `go/types.Check` (STEP 4).

## Fix
The new file walks the parsed AST and, for every named type, sums the nodes `validType` would visit, memoizing per RHS node so the guard itself is linear even when the type it models is exponential (`namedCost` / `cost` in [typecheck_bound.go:66-150](https://github.com/gnolang/gno/blob/088ce87/gnovm/pkg/gnolang/typecheck_bound.go#L66-L150) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L66)). `cost` recurses only through the edges `validType` recurses through (array element, struct fields × field-name multiplicity, interface embeddeds, named RHS) and treats pointers, slices, maps, chans, funcs, and imported types as leaves, mirroring the toolchain. Counts saturate at `MaxUint64` via `overflow.Add`/`Mul` so the arithmetic can't wrap, and a per-loop early exit at the budget keeps the guard cheap on honest deep types. The earliest-declared offender is reported for deterministic, consensus-safe error text.

## Benchmarks / Numbers
Verified on 088ce87. Guard cost is linear in depth; `validType` (real pipeline) is exponential.

| chain depth | guard (`checkTypeExpansionBound`) | `go/types` validType, value fan-out (real `TypeCheckMemPackage`) |
|---|---|---|
| 100 | 0.11 ms | — |
| 1000 | 1.2 ms | — |
| 5000 | 4.4 ms | — |
| 20 | — | rejected by guard pre-fix path / ~1.4 s unbounded |
| 40 | — | rejected: "T18 expands to ≥1_835_004 nodes > 1_000_000" |

| generic fan-out depth (real `TypeCheckMemPackage`, this PR applied) | result |
|---|---|
| 18 | 0.34 s |
| 20 | 1.4 s |
| 22 | 5.9 s |
| 24 | hang (>20 s) |

## Critical (must fix)
- **[same DoS slips through a generic type]** `gnovm/pkg/gnolang/typecheck_bound.go:143-146` — A value fan-out routed through a generic instantiation (`W[A{n-1}]`) is counted as `cost(W)`, a constant, so the guard accepts it and `go/types` validType still hangs the unmetered deploy path.
  <details><summary>details</summary>

  The `*ast.IndexExpr` / `*ast.IndexListExpr` arms return `cost(t.X)` — the cost of the *base* generic type — and discard the type argument. But `validType` substitutes the argument into the base's body and recurses through it, so when the argument is the doubling chain, each level still doubles. With `type W[P any] struct{ a, b [0]P }` and `type An struct{ x W[A{n-1}] }`, the guard scores every `An` at a small constant (`P` is unresolved → leaf → `cost(W)` ≈ 5) and never reaches the budget, while `validType` visits ~2ⁿ nodes. Through the real `TypeCheckMemPackage` (the exact call `keeper.go:647` makes at `MsgAddPackage`, mode `TCLatestStrict`), a depth-24 package hangs past 20s and a depth-40 package is the same consensus DoS the PR set out to close. Gno does accept user-defined generic types through this path (a trivial `W[int]` type-checks cleanly), so the vector is live, not hypothetical. The comment on line 144 ("generic instantiation: bound by the base type") states the load-bearing assumption that is false: the base does not bound the cost when the type argument drives the expansion, and golang/go#65711 — cited in this file's own header as the reason `validType` does not memoize — is itself the generic case. Fix: make the guard reject (or conservatively over-count) generic instantiations whose type arguments can drive value-containment fan-out, rather than scoring them by the base type alone, so a generic doubling chain hits the budget like the direct-value one. Repro and the desired post-fix assertion: [generic_fanout_dos_test.go](https://github.com/gnolang/gno/blob/088ce87/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/tests/generic_fanout_dos_test.go) · [↗](../../../../../.worktrees/gno-review-5826/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/tests/generic_fanout_dos_test.go) (carried inline in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md)).
  </details>

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- **[generic vector untested]** `gnovm/pkg/gnolang/typecheck_bound_test.go:1` — The suite covers value, pointer, slice, and function-scope fan-out but has no generic-instantiation case, the one shape the guard mishandles.
  <details><summary>details</summary>

  Adding a generic doubling chain (`type W[P any] struct{ a, b [0]P }`; `type An struct{ x W[A{n-1}] }`) as a `wantErr: true` row would have caught the Critical above. The adversarial artifact [generic_fanout_dos_test.go](https://github.com/gnolang/gno/blob/088ce87/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/tests/generic_fanout_dos_test.go) · [↗](../../../../../.worktrees/gno-review-5826/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/tests/generic_fanout_dos_test.go) is the package-level (real `TypeCheckMemPackage`) form; a unit-level `checkTypeExpansionBound` row in this file would be the natural regression guard once the Critical is fixed.
  </details>

## Suggestions
- `gnovm/pkg/gnolang/typecheck_bound.go:43` — Once the generic gap is closed, consider a deploy-path integration test (a `.txtar` sibling of `addpkg_typecheck_fanout.txtar`) for the generic shape, so the consensus boundary itself is covered, not only the unit function.
  <details><summary>details</summary>

  The existing `addpkg_typecheck_fanout.txtar` proves the value case is rejected end-to-end at `gnokey maketx addpkg`. A generic-shape twin would lock the full path. Deferred to after the fix so it asserts the post-fix `denial-of-service` rejection rather than encoding a hang. Confirmed behaviorally: the generic shape currently hangs the real `TypeCheckMemPackage`, so a txtar written now would have to encode a timeout, not a clean stderr match.
  </details>

## Open questions
- Interface type-set unions (`interface{ [0]T0 | [1]T0 }`) are modeled (`*ast.InterfaceType` sums embedded `f.Type`), but a top-level union term written as `*ast.BinaryExpr` (`A | B`) hits `cost`'s default → 1. Did not find a value-containment vector through unions that `validType` walks but the guard undercounts (union terms are only reachable as interface elements, which the struct/array value path does not recurse into by value), so not posted; worth a second look when fixing the generic arm since both are "the argument/term drives expansion" shapes.
