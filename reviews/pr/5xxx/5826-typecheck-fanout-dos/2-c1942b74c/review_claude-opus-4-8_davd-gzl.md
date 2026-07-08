# PR [#5826](https://github.com/gnolang/gno/pull/5826): fix(gnovm): bound type-expansion fan-out before go/types validType

URL: https://github.com/gnolang/gno/pull/5826
Author: ltzmaxwell | Base: master | Files: 9 | +1079 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `c1942b74c` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5826 c1942b74c`

Round 2. Head advanced `088ce87` → `c1942b74c` (+8 commits, rebased). All three round-1 findings are fixed and reject on the deploy path (direct, generic, interface-union, and cross-package fan-out), and the round-1 STEP-label nit is addressed. A full re-review pass surfaced a distinct consensus DoS the per-type budget leaves open: the guard bounds each named type but never the *sum* across types, and `go/types` runs `validType` once per type with the cross-package/#65711 cache disabled, so many individually-under-budget types stall the unmetered deploy type-check. Verdict stays REQUEST CHANGES on that new finding.

**TL;DR:** Deploying a Gno package runs Go's type checker on it, unmetered, before any gas is charged. Go's checker has a known trap: certain ~40-line type definitions make it do exponential work and effectively hang, freezing a block-producing node. This PR adds a fast pre-check that counts how much work each type would cause and rejects the package if any single type is too much, and separately rejects the go1.18 type shapes (generics, interface type-sets) that drive the same blowup.

**Verdict: REQUEST CHANGES** — the exponential single-type fan-out the PR targets is fully fixed, but the per-type budget does not bound aggregate `validType` work, so a ~450KB package built from ~16k individually-under-budget types stalls the unmetered deploy type-check ~30s (measured), a consensus DoS on the same path ([typecheck_bound.go:291-298](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L291-L298) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L291)).

## Summary
`TypeCheckMemPackage` runs `go/types` on every deployed package at `MsgAddPackage` / `MsgRun`, before gas meters the CPU ([keeper.go:647](https://github.com/gnolang/gno/blob/c1942b74c/gno.land/pkg/sdk/vm/keeper.go#L647) · [↗](../../../../../.worktrees/gno-review-5826/gno.land/pkg/sdk/vm/keeper.go#L647)). `go/types`' `validType` walk follows value-containment edges, does not memoize (disabled for golang/go#65711), and is exponential on a "doubling" chain: a ~40-type package costs ~2⁴⁰ node visits and hangs. The PR adds two pre-passes before `go/types.Check` ([gotypecheck.go:450-470](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/gotypecheck.go#L450-L470) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/gotypecheck.go#L450)): `checkNoGenerics` rejects go1.18 generics and interface type-set unions/approximations syntactically, and `checkTypeExpansionBoundImports` computes the same node count with memoization (linear), follows value-containment across import boundaries, and rejects any single type over a 100_000-node budget. All three round-1 shapes now reject. What remains: the budget is applied per named type only, and since `validType` runs once per type with no cross-call cache, total type-check cost is (number of types) × (per-type cost). A package of ~16k types each scored ~57k (all under budget) makes `validType` visit ~10⁹ nodes and stalls the deploy path ~30s, with the guard accepting the package.

## Examples
| deployed package | round-1 at `088ce87` | round-2 at `c1942b74c` |
|---|---|---|
| `type T14 struct{ a,b [0]T13 }` doubling chain | rejected (denial-of-service) | rejected: `T14 … 114684 nodes > 100000` |
| `type A_n struct{ x W[A_{n-1}] }`, `W[P] struct{ a,b [0]P }` | accepted → `validType` hangs | rejected: `generic type declarations are not supported` |
| `type I_n interface{ [0]I_{n-1} \| [1]I_{n-1} }` | accepted → `validType` hangs | rejected: `interface type unions are not supported` |
| `p1` embedding imported `p0.T` four times, over budget | accepted → node hangs on deploy | rejected: `T … 229366 nodes > 100000` |
| depth-13 chain (`T13`≈57k, under budget) + 16k siblings `type S_i struct{ x T13 }` | (not tested) | **accepted → deploy type-check ~30s** |

## Glossary
- **`validType`** — `go/types` pass that rejects infinitely-expanding value types; the exponential, unmemoized walk this PR defends against. Called once per named type declaration.
- **value-containment edge** — a type embedding another by value (struct field, array element, interface embedded/type-set term, named RHS); pointers/slices/maps/chans/funcs break it. The only edges `validType` recurses through.
- **fan-out / doubling chain** — each level references the previous one twice by value, so node count doubles per level: the worst case for an unmemoized walk.
- **per-type budget** — the guard's 100_000-node cap, applied to each named type in isolation.
- **aggregate cost** — the sum of every named type's `validType` node count, i.e. the total work `go/types` does for the package. The guard does not bound this.

## Fix (as landed)
`checkNoGenerics` ([typecheck_bound.go:374-428](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L374-L428) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L374)) rejects type parameters on a type or func declaration plus interface type-set unions (`|`) and approximations (`~`); bare and `;`-separated type-set terms are counted by the bound instead. `checkTypeExpansionBoundImports` ([typecheck_bound.go:282-306](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L282-L306) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L282)) follows value-containment into already-deployed dependencies via `expansionPkgResolver` ([typecheck_bound.go:310-340](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L310-L340) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L310)), treating stdlib as a bounded leaf, sharing a `memoizingGetter` ([gotypecheck.go:112-133](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/gotypecheck.go#L112-L133) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/gotypecheck.go#L112)) with the importer so a dependency is read once. Budget dropped 1_000_000 → 100_000 ([typecheck_bound.go:57](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L57) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L57)); measured honest max across 357 packages / 877 named types is 35.

## Edge-set coverage
`cost()` ([typecheck_bound.go:213-269](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L213-L269) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L213)) recurses through exactly the edges `validType` recurses through, verified against `$GOROOT/src/go/types/validtype.go`:

| `validType` case | recurses into | `cost()` arm | covered |
|---|---|---|---|
| `*Array` | element | `ArrayType`, `Len != nil` | counted |
| `*Struct` | fields (× field-name multiplicity) | `StructType` | counted |
| `*Interface` | embeddeds (incl. bare/`;`-separated type-set terms) | `InterfaceType` | counted |
| `*Named` | RHS / underlying, cross-package | `Ident` → `namedCost`, `SelectorExpr` | counted |
| `*Union` (go1.18) | union terms | — | rejected by `checkNoGenerics` |
| `*TypeParam` (go1.18) | substituted type argument | — | rejected by `checkNoGenerics` |
| pointer/slice/map/chan/func | (not recursed) | `StarExpr`/slice/`MapType`/`ChanType`/`FuncType` | leaf `1` |

The per-*type* modeling is faithful. The gap is that the guard scores each type against the budget in isolation and never sums.

## Benchmarks / Numbers
Aggregate `validType` cost is linear in the number of under-budget types, all accepted by the guard. Measured on the real `TypeCheckMemPackage` deploy path at `c1942b74c` (depth-13 shared chain + N siblings `type S_i struct{ x T13 }`, no imports, so pure `go/types` CPU):

| package | source | guard verdict | `TypeCheckMemPackage` wall-clock |
|---|---|---|---|
| chain only (N=0) | 15 lines | accepted | 0.004s |
| N=500 | 515 lines | accepted | 0.87s |
| N=1500 | 1515 lines | accepted | 2.6s |
| N=3000 | 3015 lines | accepted | 5.2s |
| N=8000 | 8015 lines (~224KB) | accepted | 14.9s |
| N=16000 | 16015 lines (~448KB) | accepted | 29.6s |

Slope ~1.74ms per sibling type. `MaxBlockTxBytes` is 1MB ([params.go:20-21](https://github.com/gnolang/gno/blob/c1942b74c/tm2/pkg/bft/types/params.go?plain=1#L20-L21) · [↗](../../../../../.worktrees/gno-review-5826/tm2/pkg/bft/types/params.go#L20)), so a single valid tx holds ~36k such lines → ~60s+ of unmetered type-check, well past a block-production budget.

## Critical (must fix)
- **[same consensus DoS, reached by adding types instead of one big type]** `gnovm/pkg/gnolang/typecheck_bound.go:291-298` — The budget caps each named type's `validType` cost but never the sum across types; `go/types` runs `validType` once per type with no cross-call cache, so ~16k under-budget types (a ~450KB package the guard accepts) stall the unmetered deploy type-check ~30s.
  <details><summary>details</summary>

  `checkTypeExpansionBoundImports` loops over every named type, computes its cost `v`, and rejects only if that single `v > typeExpansionBudget` ([typecheck_bound.go:291-298](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L291-L298) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L291)). It never accumulates. `go/types` calls `check.validType(t)` once per type declaration (`decl.go`), and the "exit early if already valid" cache is commented out for golang/go#65711 — the same non-memoization this PR's header cites. So each type's walk is independent: a single deep-but-under-budget type `T13` (cost ≈ 57340, accepted) referenced once by each of N sibling types costs `N × 57340` in `validType`, unbounded in N. Measured on the real `TypeCheckMemPackage` path: N=16000 (~448KB, guard accepts) = 29.6s of pure `go/types` CPU, linear at ~1.74ms/type; a full 1MB tx exceeds a minute. The CPU is unmetered ([keeper.go:647](https://github.com/gnolang/gno/blob/c1942b74c/gno.land/pkg/sdk/vm/keeper.go#L647) · [↗](../../../../../.worktrees/gno-review-5826/gno.land/pkg/sdk/vm/keeper.go#L647)) — the package imports nothing, so there are no store reads to charge — so this is the same block-producer stall the PR set out to close, reached by aggregation rather than a single exponential type. The guard already computes each per-type cost with memoization, so the sum is available in the same pass. Fix: also reject when the total `validType` cost across the package's types exceeds a global budget, not only when a single type does. Repro and observed timings in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md).
  </details>

## Warnings (should fix)
None.

## Nits
None. The round-1 nit (STEP 3 → STEP 3.5 → STEP 3 label jump) is addressed: the pre-type-check guards are labeled STEP 3 consistently ([gotypecheck.go:450](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/gotypecheck.go#L450) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/gotypecheck.go#L450)).

## Missing Tests
- **[aggregate cost untested]** `gnovm/pkg/gnolang/typecheck_bound_test.go:1` — The suite covers single-type value, pointer, slice, function-scope, multi-element interface, generic, union, and cross-package shapes, all per-type; none asserts a bound on the sum across many under-budget types.
  <details><summary>details</summary>

  A unit row that builds `buildSiblingAttack(N)` (a shared under-budget deep chain plus N siblings) and asserts `checkTypeExpansionBound` rejects it would be the regression guard once the Critical is fixed. Today the guard accepts every N, so the row must be written against the post-fix aggregate bound. The behavioral proof (guard accepts, `TypeCheckMemPackage` takes ~30s at N=16000) is in the Critical's repro.
  </details>

## Suggestions
None.

## Verified
- Round-1 fixes hold on the deploy path: the four committed txtars (`addpkg_typecheck_fanout`, `_generic`, `_imported`, `_union`) all pass at `c1942b74c` — each drop-in package that hung the node at `088ce87` is now rejected before `go/types` runs.
- `checkNoGenerics` has no bypass I could find: a generic alias (`type A[P any] = ...`) is rejected via `TypeSpec.TypeParams`, and a union nested inside a struct field (`struct{ x interface{ int | string } }`) is rejected via `ast.Inspect` reaching the nested `InterfaceType`. Bare and `;`-separated interface type-set terms are deliberately not rejected and are counted by `cost()` instead (`interface{ [0]I0; [1]I0 }` at depth 30 rejects via the bound).
- Aggregate DoS reproduced on the real `TypeCheckMemPackage` path: guard returns nil for N ∈ {0,500,1500,3000,8000,16000} while type-check time rises linearly to 29.6s at N=16000 (~448KB), all under the 1MB tx cap. Package has no imports, so the entire time is unmetered `go/types` CPU.
- Green at `c1942b74c`: `TestCheckTypeExpansionBound`, `TestCheckNoGenerics`, `TestCheckTypeExpansionBoundImports`, `TestCheckTypeExpansionBoundLinearTime`, `TestFiles/type_param0`.

## Open questions
- The author frames the "lots of valid source" case as #5892's job (per-byte gas). A flat per-byte charge priced for honest packages would under-price this attack: the sibling shape is ~2000+ `validType` nodes per source byte, so honest-rate per-byte gas does not cover a ~450KB package that costs ~10⁹ node visits. Whether to add an aggregate node-count bound here or defer to #5892 with that density accounted for is the author's call. Not posted verbatim as a separate comment; folded into the Critical's fix framing so the author sees it when deciding.
- The `*ast.IndexExpr` arm still scores a generic instantiation by its base type alone ([typecheck_bound.go:262-265](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L262-L265) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L262)). Dead-safe today: `checkNoGenerics` rejects every generic declaration at deploy, so no importable generic type can exist to instantiate; the MAINTENANCE note at [typecheck_bound.go:48-56](https://github.com/gnolang/gno/blob/c1942b74c/gnovm/pkg/gnolang/typecheck_bound.go#L48-L56) · [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L48) already flags it must be counted, not rejected, if Gno ever accepts generics. Not posted: nothing to change today.
