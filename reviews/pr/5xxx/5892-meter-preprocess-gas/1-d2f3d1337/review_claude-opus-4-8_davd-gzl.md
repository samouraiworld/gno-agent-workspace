# PR [#5892](https://github.com/gnolang/gno/pull/5892): feat(gno.land): meter type-check+preprocess gas at AddPackage and Run

URL: https://github.com/gnolang/gno/pull/5892
Author: jaekwon | Base: pr1-mempackage-split | Files: 31 | +218 -58
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: d2f3d1337 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5892 d2f3d1337`

**TL;DR:** Deploying (`MsgAddPackage`) or running (`MsgRun`) Gno code forces every validator to type-check and preprocess the submitted source, and today neither step charges gas, so a large submission buys a lot of free validator CPU. This PR adds a governable per-byte charge on the submitted `.gno` bytes, taken up front before the work runs, so an oversized package is rejected by the gas meter instead of consuming CPU for free.

**Verdict: APPROVE** — the charge is deterministic, taken before the work it bills, and legacy state is handled on every read/validate/init path; the two acknowledged limitations (super-linear worst case, uncharged dependency bytes) are correctly scoped out.

Stacked on [#5891](https://github.com/gnolang/gno/pull/5891) (base `pr1-mempackage-split`); this review covers only 5892's own diff. Ship needs a chain relaunch: the non-zero default serializes into genesis vm params, moving the app hash.

## Summary
A new governable vm param `PreprocessGasPerByte` (default 1250, proto field 14) is charged per `.gno` source byte at `AddPackage` and `Run`, before `TypeCheckMemPackage` and the later preprocess pass, both otherwise unmetered. The byte base counts prod, `_test`, and `_filetest` files because the type-check pass reads all of them. The default is validated positive and capped at 100000; a stored zero is treated as pre-field legacy state and defaulted to 1250 on every read (`GetParams`), validate (`ValidateGenesis`), and init (`InitGenesis`) path, so the charge stays active and an unrelated param update can't trip whole-struct re-validation. A drive-by replaces the prod-less-package guard's allocate-a-filtered-copy check with a per-file scan.

## Glossary
- addpkg: the `maketx addpkg` transaction that uploads a package or realm.
- amino: gno's deterministic serialization codec; the new field 14 encodes into genesis vm params state.
- app hash: per-block Merkle commitment to app state; a non-zero default param shifts it, requiring relaunch.
- gas: metered CPU/memory cost; consensus-relevant, so any change to it is a behavior change.
- MemPackage: in-memory set of a package's source files; the unit charged, type-checked, and run.
- preprocess: the static pass resolving names, types, and blocks before execution.
- type-check: go/types-based validation of gno source, distinct from preprocessing.

## Fix
Before: `AddPackage`/`Run` ran the go/types check and the VM preprocess pass with zero gas charged, linear-in-bytes free validator CPU. After: [`keeper.go:590-598`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/keeper.go#L590-L598) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L590) sums the `.gno` byte lengths and consumes `PreprocessGasPerByte * bytes` via `overflow.Mulp`, called at [`keeper.go:699-700`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/keeper.go#L699-L700) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L699) and [`keeper.go:1081`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/keeper.go#L1081) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L1081) before each type-check. The load-bearing constraint is that a zero stored value can only mean pre-field legacy state, since [`Validate`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/params.go#L181-L187) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/params.go#L181) rejects zero on every write path, so [`applyLegacyDefaults`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/params.go#L232-L237) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/params.go#L232) can safely map zero to the default without ambiguity.

## Benchmarks / Numbers
| Fixture | pre-charge | this PR | delta | notes |
|---|---|---|---|---|
| addpkg usea/useb (`addpkg_import_testdep_gas`) | 3113401 | 3218401 | +105000 | = 1250 × 84 source bytes; usea == useb equality preserved |
| addpkg bar (`restart_gas`) | 2780229 | 2937729 | +157500 | survives node restart |
| addpkg hello simulate (`gnokey_gasfee`) | 2756609 | 2850359 | +93750 | simulate estimate tracks the charge |

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- **[cap guard has no test]** `gno.land/pkg/sdk/vm/params_test.go:288` — [`Validate`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/params.go#L185-L187) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/params.go#L185)'s upper bound (`PreprocessGasPerByte > 100000`) is exercised by no test.
  <details><summary>details</summary>

  The zero and negative paths are covered: `TestGetParamsDefaultsPreprocessGasPerByte` proves a stored zero defaults, and `TestGenesisToleratesLegacyPreprocessGasPerByte` proves an explicit `-1` is rejected. The `> 100000` cap is untested. Low value: the sibling `IterNextCostFlat` cap is likewise untested, so this only closes a pre-existing pattern gap. Kept review-only; not posted.
  </details>

## Suggestions
None.

## Verified
- Revert-proof of the charge: commenting out the `AddPackagePreprocess` `chargePreprocessGas` call at [`keeper.go:700`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/keeper.go#L700) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L700) drops usea in `addpkg_import_testdep_gas.txtar` from 3218401 back to exactly 3113401, the pre-charge baseline named in that fixture's comment. The +105000 delta is 1250 × 84 source bytes, confirming the charge is applied at AddPackage and follows the per-byte formula.
- The charge is order-independent: `chargePreprocessGas` sums `len(f.Body)` over `.gno` files, so file order does not affect the total, and it adds the same constant to usea and useb, preserving PR1's `usea == useb` split-equality regression guard (both pin 3218401).
- `hasProdGnoFile` at [`keeper.go:606`](https://github.com/gnolang/gno/blob/d2f3d1337/gno.land/pkg/sdk/vm/keeper.go#L606) · [↗](../../../../../.worktrees/gno-review-5892/gno.land/pkg/sdk/vm/keeper.go#L606) is behavior-equivalent to the replaced `MPFProd.FilterMemPackage(memPkg).IsEmpty()`: `IsEmpty()` counts only `.gno` files (`IsEmptyOf(".gno")`), and `FilterGno` keeps a `.gno` file iff it is not `_test.gno`/`_filetest.gno`, so both reject exactly when no prod `.gno` file exists; the non-`.gno` files that `FilterMemPackage` copies never entered the old count.
- Tests green at d2f3d1337: `go test ./gno.land/pkg/sdk/vm` (params string, legacy-default, genesis-tolerance, WillSetParam-exhaustive, apphash-crossrealm38 with the new default-param genesis hash) and the re-pinned `addpkg_import_testdep_gas` / `restart_gas` / `gnokey_gasfee` integration fixtures.

## Open questions
- Per-byte pricing bounds the large-submission DoS but not a small package whose type-check cost is super-linear in bytes (deep generic instantiation, huge constant expressions). Author flagged this explicitly as out of scope and governable; no in-PR decision needed, so not posted.
- The charge prices the package's own bytes only; imported dependency source that gets re-go-type-checked is not charged. Author states the mid-term plan is to drop go-type-checking of dependencies, with a per-dep-byte term as a possible follow-up. Not posted.
