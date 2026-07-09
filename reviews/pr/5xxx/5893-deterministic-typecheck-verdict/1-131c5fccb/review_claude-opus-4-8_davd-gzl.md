# PR [#5893](https://github.com/gnolang/gno/pull/5893): fix(gnovm): make the consensus type-check verdict and error deterministic

URL: https://github.com/gnolang/gno/pull/5893
Author: jaekwon | Base: master | Files: 43 | +829 -132
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 131c5fccb (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5893 131c5fccb`

**TL;DR:** When the chain type-checks a submitted package, the accept/reject decision and the rejection text both feed consensus, so every node must agree. This PR makes them agree regardless of which Go toolchain each node's binary was built with: it pins the accepted Go language version, and it replaces the toolchain-dependent error text in the hashed transaction result with a fixed placeholder while keeping the full detail on the unhashed log the user sees.

**Verdict: APPROVE** — closes both identified fork axes with a committed test pinning each; the only residual (go/types accept/reject can still drift across toolchains at a fixed language version) is inherent to using the host type-checker and left as an open question, not a blocker.

## Scope note

This is the third of a stacked trio. The diff against master carries the whole stack: mempackage split ([#5891](https://github.com/gnolang/gno/pull/5891): `store.go`, `keeper.go`, `machine.go`, `debugger.go`) and preprocess-gas metering ([#5892](https://github.com/gnolang/gno/pull/5892): `params.go`, `genesis.go`, gas txtars). Those belong to their own PRs and are reviewed there. This review covers PR #5893's own delta: the two determinism commits (`349755959`, `60ace8868`) plus moul's merge-cleanup follow-ups (`2f8da6e7f`, `c1546d6d9`, `131c5fccb`). The stacked files were read for determinism/consensus red flags; none attributable to this PR.

## Summary

Two independent ways two honest validators could fork on the same submitted package, no attacker needed. First, the consensus type-checker built its `go/types` config with no `GoVersion`, so version-gated syntax (e.g. `for range 10`, a go1.22 feature) was accepted or rejected according to whichever Go the validator binary was compiled with. Second, on rejection the raw `go/types`/`go/parser` diagnostic strings were amino-encoded into `TypeCheckError.Errors` and hashed into the block's results hash, and that text drifts in wording, count, and order across toolchains, so two nodes correctly rejecting the same package could still commit different result hashes. The fix pins `GoVersion` to `go1.18` (a syntax-acceptance floor only, the shim's generics being the reason it can't go lower) and collapses `TypeCheckError` to an empty sentinel, moving the full diagnostics onto the unhashed `Result.Log` trace.

## Examples

| A user submits | Before | After |
|---|---|---|
| `for range 10` (range-over-int, go1.22) | accepted on a go1.22+ build, rejected on older → fork | rejected on every build (pinned go1.18) |
| a loop body closing over the loop variable | accepted | accepted (its go1.22 per-iteration semantics live in the interpreter, not this gate) |
| any type-check failure | raw go/types text hashed into the result → wording/count/order forks the hash | fixed sentinel hashed; full text stays on the unhashed log |

## Glossary

- type-check: go/types-based validation of gno source (`TypeCheckMemPackage`), distinct from preprocessing.
- results hash: per-block Merkle commitment to every tx's `ABCIResult`; toolchain-dependent bytes in `ABCIResult.Error` fork the chain.
- app hash: per-block commitment to application state; divergence halts the chain.
- gnobuiltins: synthetic packages injected only for type-checking, never run on chain.
- Amino: gno's deterministic serialization codec; here it encodes the hashed error.

## Fix

The pin lands in the shared `types.Config` at [`gotypecheck.go:178-189`](https://github.com/gnolang/gno/blob/131c5fccb/gnovm/pkg/gnolang/gotypecheck.go#L178-L189) · [↗](../../../../../.worktrees/gno-review-5893/gnovm/pkg/gnolang/gotypecheck.go#L178), so it applies to the target package and every imported one. `TypeCheckError` drops its `Errors []string` field and becomes `struct{ abciError }` at [`errors.go:27-35`](https://github.com/gnolang/gno/blob/131c5fccb/gno.land/pkg/sdk/vm/errors.go#L27-L35) · [↗](../../../../../.worktrees/gno-review-5893/gno.land/pkg/sdk/vm/errors.go#L27) with a constant `Error()` string; `ErrTypeCheck` now wraps the joined messages as the error's msg trace around that empty sentinel at [`errors.go:81-91`](https://github.com/gnolang/gno/blob/131c5fccb/gno.land/pkg/sdk/vm/errors.go#L81-L91) · [↗](../../../../../.worktrees/gno-review-5893/gno.land/pkg/sdk/vm/errors.go#L81). The proto field and its hand-generated amino marshal/unmarshal are removed ([`vm.proto:49-50`](https://github.com/gnolang/gno/blob/131c5fccb/gno.land/pkg/sdk/vm/vm.proto#L49-L50) · [↗](../../../../../.worktrees/gno-review-5893/gno.land/pkg/sdk/vm/vm.proto#L49), `pb3_gen.go`). Both keeper call sites, [`keeper.go:704`](https://github.com/gnolang/gno/blob/131c5fccb/gno.land/pkg/sdk/vm/keeper.go#L704) · [↗](../../../../../.worktrees/gno-review-5893/gno.land/pkg/sdk/vm/keeper.go#L704) (AddPackage) and [`keeper.go:1090`](https://github.com/gnolang/gno/blob/131c5fccb/gno.land/pkg/sdk/vm/keeper.go#L1090) · [↗](../../../../../.worktrees/gno-review-5893/gno.land/pkg/sdk/vm/keeper.go#L1090) (Run), feed through this one wrapper, so both `go/types` and `go/parser` diagnostics are coarsened.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- `gno.land/pkg/sdk/vm/params.go:26-38` — the default-const block is now mixed-typed: `minWriteDepth100Default` and `iterNextCostFlatDefault` are untyped while `minGetReadDepth100Default`, `minSetReadDepth100Default`, and `preprocessGasPerByteDefault` are `int64(...)`. Each is correct for its use (the untyped ones only feed `NewParams`; `preprocessGasPerByteDefault` needs the type because `params_test.go` compares it against an `int64` field with `assert.Equal`, which is type-sensitive), so this is cosmetic drift from the two merge-cleanup commits, not a defect. Not posted.

## Missing Tests
None. Coverage is thorough: the GoVersion pin test asserts range-over-int rejected and loop-closure accepted at [`gotypecheck_test.go:450`](https://github.com/gnolang/gno/blob/131c5fccb/gnovm/pkg/gnolang/gotypecheck_test.go#L450) · [↗](../../../../../.worktrees/gno-review-5893/gnovm/pkg/gnolang/gotypecheck_test.go#L450); the hashed-error determinism test asserts two different failures encode to identical `ABCIResult.Bytes()` while the full detail survives on `%#v` at [`errors_test.go:22`](https://github.com/gnolang/gno/blob/131c5fccb/gno.land/pkg/sdk/vm/errors_test.go#L22) · [↗](../../../../../.worktrees/gno-review-5893/gno.land/pkg/sdk/vm/errors_test.go#L22); `range9.gno`/`range12.gno` add `TypeCheckError:` directives; `err_metadata.txtar:15` pins the sentinel in the hashed error and the full parser diagnostic in the log.

## Suggestions
None.

## Verified
- Revert-proof of the GoVersion pin (CI runs only with the pin, so this is CI-invisible): commenting out `GoVersion: "go1.18"` and re-running `TestTypeCheckMemPackage_GoVersionPinned` on a go1.26 toolchain makes `for range 10` type-check with no error (`An error is expected but got nil`), i.e. this build accepts what the pinned checker rejects — the exact build-dependent verdict divergence the PR removes. File restored after.
- Determinism chain traced end to end: `ErrTypeCheck` → `errors.Wrap(TypeCheckError{}, msgs)` captures a stacktrace and puts the joined diagnostics on the msg trace; `sdk.ABCIError` → `ABCIErrorOrStringError` calls `errors.Cause`, which unwraps to the bare `TypeCheckError{}` data, so only the empty sentinel is amino-encoded into `ABCIResult.Error`, and `ABCIResults.Hash` merkle-hashes that into `LastResultsHash` ([`results.go:14-23`](https://github.com/gnolang/gno/blob/131c5fccb/tm2/pkg/bft/types/results.go#L14-L23) · [↗](../../../../../.worktrees/gno-review-5893/tm2/pkg/bft/types/results.go#L14)). No diagnostic-dependent bytes reach the hashed path.
- No consumer reads the removed `TypeCheckError.Errors` field: the only references are the two keeper call sites (via `ErrTypeCheck`) and the amino registration. Filetests read a separate `RunResult.TypeCheckError` string populated directly from `gno.TypeCheckMemPackage`, so `TypeCheckError:` directives are unaffected.
- Green at 131c5fccb: `TestTypeCheckMemPackage_GoVersionPinned`, `TestErrTypeCheckCoarseHashedError`, `TestFiles/range9.gno`, `TestFiles/range12.gno`, `TestTestdata/err_metadata`, and the vm package's `TypeCheck|Params|ImportTestDepGas|Preprocess` subset.

## Open questions
- The pin fixes the version-gated subset of the accept/reject verdict, but `go/types` acceptance at a fixed `GoVersion` can still differ across Go toolchain releases (implementation bug fixes, tightened/loosened checks on non-version-gated code). Under the PR's own heterogeneous-binary threat model that residual is not closed; the only full fix is vendoring a pinned `go/types`, which is large. In practice validators run identical release binaries, so it is defense-in-depth; noted for maintainer awareness, no change asked in this PR.
