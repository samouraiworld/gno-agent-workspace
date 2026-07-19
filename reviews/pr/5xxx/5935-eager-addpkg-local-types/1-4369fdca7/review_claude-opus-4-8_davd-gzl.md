# PR [#5935](https://github.com/gnolang/gno/pull/5935): fix(gnovm): persist function-local declared types eagerly at addpkg (alt to save-time walk)

URL: https://github.com/gnolang/gno/pull/5935
Author: ltzmaxwell | Base: master | Files: 11 | +869 -4
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 4369fdca7 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5935 4369fdca7`

**TL;DR:** In gno, a type declared inside a function body (`type S struct{...}`) was never written to the chain's type store, so storing a value of that type in realm state saved a pointer to a record that does not exist; after a node restart every read of that value crashed and the realm was permanently unreadable. This PR writes all such types once, at package upload time, instead of walking every object on every save as the earlier attempt did.

**Verdict: NEEDS DISCUSSION** — the enumeration is complete and the restart tests genuinely reproduce, but the PR drops the save-time walk without a migration, so packages uploaded before it merges stay broken, and the guard meant to catch a missed route is compiled out everywhere CI runs (2 Warnings, 1 Missing test, 1 Suggestion).

## Summary
Package-level declared types are `SetType`'d at addpkg; function-local types were not. Any saved `TypedValue` whose type is function-local serialized as `RefType{"pkg[loc].Name"}`, a dangling pointer into the type store `/t/`. The live process never noticed because the type sits in `cacheTypes`; after restart, loading the object hits `GetType` → miss → `panic("unexpected type with id ...")`. This PR replaces [#5894](https://github.com/gnolang/gno/pull/5894)'s per-save walk with a single AST pass at upload time: [`saveFuncLocalTypes`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L917-L943) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L917-L943) transcribes the package fileset and `SetType`s every non-alias `*TypeDecl` whose type is a function-local `DeclaredType`. Persisting the type once with the deployer, rather than on-demand at whichever transaction first escapes a value, is the whole point of the change.

The soundness argument holds up: local `DeclaredType`s are materialized at preprocess time, so the type object the walk stores is the same pointer `doOpTypeDecl` later assigns at runtime ([`op_decl.go:66-73`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/op_decl.go#L66-L73) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/op_decl.go#L66-L73)), and the walk reuses the same `Transcribe` traversal the preprocessor itself uses, so any `TypeDecl` the preprocessor reached is reachable here. Legal placements are narrow: `declareWith` accepts only `PackageNode`/`FileNode`/`FuncDecl`/`FuncLitExpr` parents ([`types.go:1526-1537`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/types.go#L1526-L1537) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/types.go#L1526-L1537)), so a type declared in a nested block is rejected at preprocess time and no `ParentLoc` collision between two same-named types in one function is reachable.

```
addpkg (runMemPackage save=true):
  runFileDecls (preprocess)  →  saveFuncLocalTypes  →  FinalizeRealmTransaction  →  init()  →  resave
                                 └─ SetType(S) per *TypeDecl        └─ var-initializer values persist here
later tx:
  Bind() persists S-typed value → RefType{pkg[loc].S} → /t/ record already written  ✓
```

## Examples
| gno on chain | before | after |
|---|---|---|
| `type S struct{T}; X = S{...}` (interface var) | reload panics `unexpected type with id ...S` | reads back |
| `type S struct{T}` in a `p/` package, value stored in a realm | reload panics | `p/` addpkg wrote the record |
| package uploaded before this merges, value escapes after | reload panics | still panics, no record ever written |

## Glossary
- function-local type: `DeclaredType` declared in a function body; TypeID is `pkg[loc].Name`.
- addpkg: the `maketx addpkg` upload transaction, where types are `SetType`'d.
- RefType: placeholder a persist-copy stores for a type; `RefType{ID}` → `/t/<ID>` record, resolved on reload.
- persist-copy: ref-collapsed object copy (`copyValueWithRefs`) amino-marshaled on `SetObject`.
- TypeID: canonical type identity; changing an already-persisted TypeID is consensus-breaking.
- storage deposit: per-realm refundable charge for on-chain storage bytes, via `processStorageDeposit`.

## Fix
`saveNewPackageValuesAndTypes` gains one call at [`machine.go:874`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L874) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L874), placed above the realm finalization so a file-level var initializer already holding a local-typed value finds its type written. `copyTypeWithRefs` now copies `ParentLoc` ([`realm.go:1575`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/realm.go#L1575) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/realm.go#L1575)) so the stored record lands under the TypeID that values reference. The `debugAssert` invariant in `SetObject` ([`store.go:618-626`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/store.go#L618-L626) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/store.go#L618-L626)) now falls back to a raw backend key probe ([`store.go:930-940`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/store.go#L930-L940) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/store.go#L930-L940)), because addpkg-persisted types are absent from a later transaction's fresh `cacheTypes`.

## Critical (must fix)
None.

## Warnings (should fix)
- **[already-deployed packages stay broken]** `gnovm/pkg/gnolang/machine.go:917` — the only route that writes a function-local type is the addpkg walk, so packages uploaded before this merges never get a record and values of their local types still persist as dangling refs.
  <details><summary>details</summary>

  Repo-wide there are three `SetType` call sites: [`machine.go:896`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L896) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L896) and [`store.go:392`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/store.go#L392) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/store.go#L392) both iterate package block values, which hold only package-level types, and [`machine.go:940`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L940) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L940) inside the new walk. That walk runs only from `saveNewPackageValuesAndTypes`, reached only via `runMemPackage` with `save=true` ([`machine.go:430-435`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L430-L435) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L430-L435)), which on chain is addpkg and stdlib load. Nothing re-runs it for a package already in the store. [#5894](https://github.com/gnolang/gno/pull/5894)'s save-time walk healed those packages on their next save; deleting it trades that away. `examples/` itself is unaffected: outside `_test.gno` files and `examples/quarantined/`, the only two matches for a function-local type declaration are the `stringer` interfaces at [`uassert.gno:403`](https://github.com/gnolang/gno/blob/4369fdca7/examples/gno.land/p/nt/uassert/v0/uassert.gno#L403) · [↗](../../../../../.worktrees/gno-review-5935/examples/gno.land/p/nt/uassert/v0/uassert.gno#L403) and [`uassert.gno:537`](https://github.com/gnolang/gno/blob/4369fdca7/examples/gno.land/p/nt/uassert/v0/uassert.gno#L537) · [↗](../../../../../.worktrees/gno-review-5935/examples/gno.land/p/nt/uassert/v0/uassert.gno#L537), both inside `/* */` block comments the parser never turns into a `*TypeDecl`. The exposure is user-deployed packages on any chain that keeps its state across this merge. Fix: decide in this PR whether the target chain is fresh, and if not, ship the migration or keep the save-time walk as a transitional backstop.
  </details>

- **[the safety net never runs]** `gnovm/pkg/gnolang/store.go:618` — the invariant that catches a missed enumeration route is behind the `debugAssert` build tag, and no CI workflow builds with it.
  <details><summary>details</summary>

  With the save-time walk gone, correctness rests entirely on the addpkg walk enumerating every function-local type, and the only thing that detects a gap is `assertNoDanglingLocalTypeRef` in `SetObject`. `-tags debugAssert` appears in exactly one place in the repo, the [`test.debugAssert` target](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/Makefile#L116-L119) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/Makefile#L116-L119); grepping `.github/workflows/` for `debugAssert` returns nothing, so no job invokes it. The guard is therefore absent from production builds and from every CI run, and the three `zrealm_localtype*.gno` filetests that exist to exercise it pass identically with and without the tag. Fix: run the localtype filetests under `-tags debugAssert` in CI so the invariant is load-bearing.
  </details>

## Nits
None.

## Missing Tests
- **[call ordering can regress silently]** `gnovm/pkg/gnolang/machine.go:870-874` — no test covers a file-level var initializer holding a local-typed value, so moving the walk below the realm finalization leaves every committed test green.
  <details><summary>details</summary>

  The comment at [`machine.go:870-873`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L870-L873) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L870-L873) states the constraint: the walk must precede `FinalizeRealmTransaction` because a file-level var initializer may already hold a local-typed value at addpkg-save time. Every committed test assigns inside `main` or inside an explicitly called `Bind`, so none of them exercises it. Moving the call below the `IsRealm` block leaves the three `zrealm_localtype*.gno` filetests green under `-tags debugAssert` and all three `restart_local_type*.txtar` green, while a package whose `var X = mk()` returns a local-typed value panics `dangling function-local type ref gno.land/r/test[...].S in persisted value`. The same file also covers the `init()` half, which persists via `resavePackageValues` after the walk. Fix: add the filetest at [`zrealm_localtype3.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5935-eager-addpkg-local-types/1-4369fdca7/tests/zrealm_localtype3.gno) · [↗](tests/zrealm_localtype3.gno); it passes at 4369fdca7 and fails if the call moves.
  </details>

## Suggestions
- **[unpriced state growth per package]** `gnovm/pkg/gnolang/machine.go:930-943` — every function-local type in a package now gets a `/t/` record whether or not a value of it can ever escape, and type records are outside the storage-deposit accounting.
  <details><summary>details</summary>

  The walk `SetType`s on the `*TypeDecl` alone, with no reachability test, so a type declared in a function that no caller can make escape gets a record all the same, as does a local interface type, whose values are always persisted under their concrete type. `SetType` charges amino-encode gas and writes straight to `baseStore` ([`store.go:855-869`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/store.go#L855-L869) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/store.go#L855-L869)) without touching `realmStorageDiffs`, which `processStorageDeposit` reads via [`RealmStorageDiffs`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/store.go#L1277-L1279) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/store.go#L1277-L1279) and which only `SetObject` feeds. Package-level types have always behaved this way, so the class is pre-existing; what changes is that the local-type share goes from "types that actually escaped" to "all of them". Record size stays roughly linear in source size, so this is store bloat rather than an amplification vector. No change requested; flagging for whoever prices type storage next.
  </details>

## Verified
Verified on 4369fdca7 (checks the test suite does not run):

- **Revert-proof:** commenting out the `m.saveFuncLocalTypes(pv)` call at [`machine.go:874`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L874) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L874) makes `restart_local_type.txtar` fail after the node restart with `unexpected type with id gno.land/r/test/lt2[gno.land/r/test/lt2/lt2.gno:11:1-14:2].S`, the permanently-unreadable-state panic. The eager walk is what carries the fix, not the `ParentLoc` copy alone.
- **Ordering is untested:** relocating the same call below the `IsRealm` block leaves all three `zrealm_localtype*.gno` filetests green under `-tags debugAssert` and all three `restart_local_type*.txtar` green; only the added [`zrealm_localtype3.gno`](tests/zrealm_localtype3.gno) probe panics. See the Missing test finding.
- **Enumeration reaches closure-nested declarations:** the walk uses the same `Transcribe` pass the preprocessor uses, and `restart_local_type3.txtar` exercises a type declared inside a `FuncLitExpr` re-declared in a post-restart transaction; it passes.
- **Nested-block declarations are unreachable:** a `type S` inside an `if` body is rejected at preprocess time with `expected type expr but got *gnolang.IfCaseStmt` from [`declareWith`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/types.go#L1536) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/types.go#L1536), so two same-named local types in one function cannot collide on a shared `ParentLoc`.
- Green at 4369fdca7: `restart_local_type{,2,3}.txtar`, `zrealm_localtype{0,1,2}.gno` with and without `-tags debugAssert`, and `./gno.land/pkg/sdk/vm/...` including the gas fixtures.

## Open questions
- `ParentLoc` folds the declaring function's full line/column span into the TypeID, so on-chain type identity is now pinned to source spans; a parser change that shifts a span would invalidate stored records. Immutable on-chain source makes it inert, and the same property holds for the competing design. Not posted: no action available in this PR.
- `zrealm_localtype0.gno`'s golden pins the literal span `:12:1-16:2`, so editing lines above `main` in that file rewrites the expected TypeID. Intentional, since pinning the on-the-wire `RefType` shape is the test's job. Not posted: no change needed.
