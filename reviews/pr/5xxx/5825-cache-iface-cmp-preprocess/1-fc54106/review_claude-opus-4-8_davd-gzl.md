# PR #5825: perf(gnovm): cache interface-comparison verdict at preprocess (ATTR_IFACE_CMP)

URL: https://github.com/gnolang/gno/pull/5825
Author: ltzmaxwell | Base: master | Files: 9 | +313 -26
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: fc54106 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5825 fc54106`

**TL;DR:** Comparing two interface values with `==`/`!=` must panic when their boxed dynamic type is uncomparable (e.g. a slice). That rule needs a static "is an operand interface-typed?" verdict, which a sibling PR recomputed on every comparison. This PR computes it once during compilation and stores it on the syntax node, so the hot loop just reads a flag.

**Verdict: APPROVE** — pure perf refactor of a verdict introduced in #5713; behavior is unchanged and the new cold-reload test has real teeth. Only a stale comment in one benchmark fixture.

## Summary

For `==`/`!=` on interface-typed operands, Go panics if the dynamic type is uncomparable, and that static verdict cannot be derived from the runtime operand bytes alone (`var s []int; s == nil` is legal, `any(s) == any(s)` panics, both reach `isEql` with `{T: []int, V: nil}`). #5713 carried the verdict by recomputing it per evaluation: two `ATTR_TYPEOF_VALUE` map lookups plus a `baseOf` on each `doOpEql`/`doOpNeq`. This PR computes it once at preprocess from the already-resolved operand static types and caches it as `ATTR_IFACE_CMP`, dropping the hot path to a single map lookup (~2.3% over feature-off vs #5713's ~8.5% on the concrete path, per the PR's benchstat). The same attribute folds in the `switch`-tag path in `op_exec`, retiring the per-eval helpers `isInterfaceCmp` and `hasInterfaceStaticType`.

## Glossary

- **`ATTR_IFACE_CMP`** — new attribute set on a surviving `==`/`!=` `BinaryExpr` or a non-type-switch `SwitchStmt` whose operand/tag is statically interface-typed.
- **`isEql`** — the runtime equality kernel; its `viaIface` arg gates the uncomparable-dynamic-type panic.
- **preprocess** — the compile/typecheck pass over the AST; runs once per package, re-runs on every node restart (packages are re-preprocessed on load).
- **`ATTR_TYPEOF_VALUE`** — existing per-expr cached static type, also preprocess-derived and non-persisted; the lifecycle `ATTR_IFACE_CMP` piggybacks on.

## Fix

The verdict moves from runtime to preprocess. In `preprocess.go`'s `*BinaryExpr` `TRANS_LEAVE`, after all const-fold and shift cases have already returned, the surviving `EQL`/`NEQ` nodes get `ATTR_IFACE_CMP=true` when either operand's static type (`lt`/`rt`, resolved at the top of the case) is an interface, at [`preprocess.go:1546-1552`](https://github.com/gnolang/gno/blob/fc54106ed40af4865db37f2985458462f0049f1d/gnovm/pkg/gnolang/preprocess.go#L1546-L1552) · [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/preprocess.go#L1546). The `*SwitchStmt` case mirrors it for a non-type-switch with an interface tag at [`preprocess.go:2972-2981`](https://github.com/gnolang/gno/blob/fc54106ed40af4865db37f2985458462f0049f1d/gnovm/pkg/gnolang/preprocess.go#L2972-L2981) · [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/preprocess.go#L2972). The three runtime readers ([`op_binary.go:92`](https://github.com/gnolang/gno/blob/fc54106ed40af4865db37f2985458462f0049f1d/gnovm/pkg/gnolang/op_binary.go#L92) · [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/op_binary.go#L92), [`:109`](https://github.com/gnolang/gno/blob/fc54106ed40af4865db37f2985458462f0049f1d/gnovm/pkg/gnolang/op_binary.go#L109) · [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/op_binary.go#L109), [`op_exec.go:986`](https://github.com/gnolang/gno/blob/fc54106ed40af4865db37f2985458462f0049f1d/gnovm/pkg/gnolang/op_exec.go#L986) · [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/op_exec.go#L986)) replace the helper calls with `GetAttribute(ATTR_IFACE_CMP) == true`. The single preprocess-time predicate `isInterfaceStaticType(Type)` lives in [`types.go:2612-2620`](https://github.com/gnolang/gno/blob/fc54106ed40af4865db37f2985458462f0049f1d/gnovm/pkg/gnolang/types.go#L2612-L2620) · [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/types.go#L2612).

## Benchmarks / Numbers

From the PR's ADR (Apple M3, n=15, benchstat, program-level loop), overhead vs feature-off:

| path | #5713 (prior) | attr (this PR) |
|---|---|---|
| concrete `==` | +8.5% | +2.3% |
| interface `==` | +5.6% | +2.2% |

Not independently re-measured; the absolute correctness and zero-orphan claims below are what I verified.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- `benchdata/cmp_iface.gno:5` — comment names `isInterfaceCmp` and `OpEqlIface`, neither of which exists after this PR.
  <details><summary>details</summary>

  The comment describes the exercised path as "the `isInterfaceCmp`/`OpEqlIface` path". `isInterfaceCmp` is removed by this very PR ([`op_binary.go`](https://github.com/gnolang/gno/blob/fc54106ed40af4865db37f2985458462f0049f1d/gnovm/pkg/gnolang/op_binary.go) · [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/op_binary.go), helper deleted), and `OpEqlIface` is the opcode design the ADR explicitly abandoned, so it never existed in the tree. Confirmed: `grep -rn 'OpEqlIface\|OpNeqIface\|isInterfaceCmp' gnovm/ --include='*.go'` returns zero matches outside the ADR. The fixture itself is correct and runs. Fix: name the live path (the `ATTR_IFACE_CMP` / `isEql` interface-comparison path).
  </details>

## Missing Tests

None. The new [`op_binary_ifacecmp_reload_test.go`](https://github.com/gnolang/gno/blob/fc54106ed40af4865db37f2985458462f0049f1d/gnovm/pkg/gnolang/op_binary_ifacecmp_reload_test.go) · [↗](../../../../../.worktrees/gno-review-5825/gnovm/pkg/gnolang/op_binary_ifacecmp_reload_test.go) closes the only new risk this PR introduces (a non-persisted attribute that must be re-derived on node restart), and the 18 pre-existing `cmp_uncomp_*` filetests cover the behavioral surface.

## Suggestions

None.

## Verification

Verified on fc54106, from the `.worktrees/gno-review-5825` worktree:

- **Behavioral parity on the tricky boundaries (not just the happy path).** A custom filetest comparing an interface against untyped `nil` (legal, no panic) and a function-call operand returning an uncomparable interface (`get() == get()` where `get` returns `interface{}` holding `[]int`) reproduces the exact old behavior: `a == nil` → `false`, the call-operand comparison panics and is recovered, `UNREACHABLE` never prints. This is the case where the new code could have diverged: it snapshots `lt`/`rt` *before* `checkOrConvertType` rewrites the operands, whereas the old per-eval path read the type *after*. The two agree because in the `EQL`/`NEQ` branches `checkOrConvertType` only ever converts *untyped* operands (an untyped const/nil), and an untyped operand neither is nor becomes an interface, so interface-ness is invariant across the rewrite.
- **The cold-reload test has teeth.** Commenting out the `BinaryExpr` `ATTR_IFACE_CMP` set makes `TestBinaryExprIfaceCmp_SurvivesColdReload` fail ("comparing uncomparable []int via interface must panic after cold reload" — the panic no longer fires because `viaIface` defaults to false). The test genuinely guards the re-preprocess-on-restart assumption, not a tautology.
- **Zero orphaned references.** Both removed helpers (`isInterfaceCmp`, `hasInterfaceStaticType`) have no remaining callers; `ATTR_IFACE_CMP` is written at exactly the two preprocess sites and read at exactly the three runtime sites; `isInterfaceStaticType` has only its two preprocess callers. Full call graph consistent.
- **The two new benchmark fixtures are wired in and execute.** `BenchmarkBenchdata` auto-discovers `./benchdata` via `os.ReadDir`, so `cmp_concrete.gno`/`cmp_iface.gno` need no registration; both run clean under the VM at `-benchtime=1x`.

Suites: `Gas` and `TestTestdata` (txtar) green; all 18 `cmp_uncomp_*` filetests green; `gnovm/pkg/gnolang` non-Files unit tests green; gofmt/go vet clean. The `Files -short` suite reports 10 failures locally, but they are **pre-existing on origin/master** (a `go/types` toolchain drift in type-check error wording, e.g. `i (local variable) is not a type` vs `i is not a type`); the failing set is byte-identical between base and PR (`diff` empty) and is unrelated to comparison logic. PR CI is fully green.

## Open questions

None.
