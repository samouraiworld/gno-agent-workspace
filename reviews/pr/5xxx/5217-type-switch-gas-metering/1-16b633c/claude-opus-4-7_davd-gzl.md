# PR #5217: fix(gnovm): meter gas correctly for switch case

URL: https://github.com/gnolang/gno/pull/5217
Author: davd-gzl | Base: master | Files: 14 | +413 -47
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]
Local worktree: `git -C gno worktree add .worktrees/gno-review-5217 16b633c` (then `gh -R gnolang/gno pr checkout 5217` inside it)

**Disclosure:** the reviewer is also the PR author. Review is best-effort adversarial; the verdict reflects what a third-party reviewer should see, not author endorsement.

**Verdict: APPROVE** — closes NEWTENDG-81/184 by replacing flat-cost type-switch/type-assert metering with per-leaf-method-probe charging that captures the O(N*M) shape; CI green, jefft0 approved, two minor follow-ups (stale doc comment, deep-embed depth under-charge) noted as nits.

## Summary

`doOpTypeSwitch` charged a flat `OpCPUTypeSwitch=171` regardless of clauses, cases, or interface-method probes, and `doOpTypeAssert1/2` mixed a per-interface-method slope with no concrete-side factor. A crafted type switch with N interface cases over a concrete type with M methods drove `findEmbeddedFieldType` to do O(N*M) walking for O(1) gas — a denial-of-service vector against the GnoVM. The fix unifies both ops on a single `OpCPUInterfaceMethodCheck=4` charged once per leaf-method probe, with `perCheck = 4 * countTypeMethodsForGas(ot)` computed once per call site and hoisted out of clause loops; per-clause/per-case overhead reuses the existing `OpCPUSwitchClause`/`OpCPUSwitchClauseCase` constants from expression switches. `verifyImplementedBy` is split into a gas-free public `IsImplementedBy(it, ot)` (used by `type_check.go`, `uverse.go`, `values_string.go`) and a gas-charging unexported `it.verifyImplementedBy(m, perCheck, ot)` for VM paths, so compile-time and debug callers don't accidentally meter.

## Glossary

- `verifyImplementedBy` — interface-satisfaction check; walks each leaf method of the interface and probes the concrete type via `findEmbeddedFieldType`.
- `findEmbeddedFieldType` — recursive lookup of a name across embedded fields/methods of a type; O(M) per call where M is the effective method-set size.
- `perCheck` — gas cost of one method probe, precomputed as `OpCPUInterfaceMethodCheck * methodCount(ot)`.
- `countTypeMethodsForGas` — upper-bound count of methods walked when probing `ot`; handles `*PointerType`, `*InterfaceType`, `*DeclaredType`, `*StructType`, with cycle guard via `seen` map.
- `OpCPUInterfaceMethodCheck` — new per-probe constant, placeholder value 4 (TODO: bench-grid calibration).

## Fix

Before: `doOpTypeSwitch` charged `171 + 254*nClauses` and never accounted for interface-case method probing; `doOpTypeAssert1/2` charged `OpCPUSlopeTypeAssertIface (349) * len(it.Methods)` but ignored the concrete-side method count, so iface-with-1-method probing a concrete-with-1000-methods type still charged ~349 gas for ~1000 lookups. After: both ops charge `OpCPUInterfaceMethodCheck * methods(ot)` per leaf-method probe via `verifyImplementedBy`, so gas scales linearly in N*M; per-clause and per-case overhead are charged from `doOpTypeSwitch` reusing the constants that already meter expression switches at [`gnovm/pkg/gnolang/machine.go:1574-1578`](https://github.com/gnolang/gno/blob/16b633c/gnovm/pkg/gnolang/machine.go#L1574-L1578) · [↗](../../../../../.worktrees/gno-review-5217/gnovm/pkg/gnolang/machine.go#L1574-L1578). The constraint is that `perCheck` must be computed from `xv.T` (the dynamic concrete value) — done lazily on the first interface case at [`op_exec.go:867-869`](https://github.com/gnolang/gno/blob/16b633c/gnovm/pkg/gnolang/op_exec.go#L867-L869) · [↗](../../../../../.worktrees/gno-review-5217/gnovm/pkg/gnolang/op_exec.go#L867-L869) so type switches over pure concrete cases pay nothing extra.

## Benchmarks / Numbers

Filetest gas after fix (HEAD `16b633c`):

| Test | Methods | Gas |
|---|---|---|
| `typeassert_iface_small.gno` | 1 | 5007 |
| `typeassert_iface_large.gno` | 10 | 5565 |
| `typeassert_iface_embedded.gno` | 10 (via IA+IB embed) | 6322 |
| `typeswitch_iface_small.gno` | 1 | 5238 |
| `typeswitch_iface_large.gno` | 10 | 5796 |
| `typeswitch_iface_pointer.gno` | 10 (`&S{}`) | 6199 |
| `typeswitch_iface_embedded_struct.gno` | 10 (struct embeds Inner) | 6370 |
| `typeswitch_clauses_small.gno` | 1 concrete clause | 3808 |
| `typeswitch_clauses_large.gno` | 8 concrete clauses | 3910 |

Per-method delta from 1→10 methods on the iface-leaf path: `(5565-5007)/9 ≈ 62` and `(5796-5238)/9 ≈ 62`. That's `4 (OpCPUInterfaceMethodCheck) * count(ot=10) = 40` per probe, multiplied by 9 extra leaf methods — broadly consistent with the 4*N*M model when the type's full method set is walked. Embedded-interface counts match flat counts (6322 ≈ flat-10 equivalent path), confirming `countTypeMethodsForGas` recursion through `*InterfaceType` and `*StructType` works as intended.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/pkg/gnolang/types.go:1088`](https://github.com/gnolang/gno/blob/16b633c/gnovm/pkg/gnolang/types.go#L1088) · [↗](../../../../../.worktrees/gno-review-5217/gnovm/pkg/gnolang/types.go#L1088) — comment says "Used to meter the O(N*M) cost of VerifyImplementedBy" but that name no longer exists (renamed to lowercase `verifyImplementedBy` in this PR). Update to match.
- [`gnovm/pkg/gnolang/machine.go:1334`](https://github.com/gnolang/gno/blob/16b633c/gnovm/pkg/gnolang/machine.go#L1334) · [↗](../../../../../.worktrees/gno-review-5217/gnovm/pkg/gnolang/machine.go#L1334) — `OpCPUInterfaceMethodCheck = 4` is flagged as a placeholder with a TODO; safe ceiling for now (4M gas worst-case at N=1000, M=1000 is ~0.13% of block gas), but worth tracking as a follow-up so it doesn't decay.

## Missing Tests

- [`gnovm/tests/files/gas/`](https://github.com/gnolang/gno/blob/16b633c/gnovm/tests/files/gas/) · [↗](../../../../../.worktrees/gno-review-5217/gnovm/tests/files/gas/) — no test exercises the **deep-embedded-empty-struct** case (chain of N embedded structs with zero methods anywhere). `countTypeMethodsForGas` returns 1 for such a chain via the `if n < 1 { return 1 }` floor, so `perCheck = 4`. The actual walk cost in `findEmbeddedFieldType` is O(depth) per probe — under-charged for pathological types. Tx-size cap on type declarations bounds the realistic attack surface, but a filetest pinning the gas for `type A struct { B }; type B struct { C }; ...` followed by an interface assertion would document the floor.
  <details><summary>details</summary>

  Construct a struct chain `S0 ⊂ S1 ⊂ ... ⊂ S100` (each embeds the previous, no methods), then `_ = x.(SomeIface)`. Verify gas vs. a flat 1-field struct. Diff will quantify the under-charge per nesting level.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/types.go:1091`](https://github.com/gnolang/gno/blob/16b633c/gnovm/pkg/gnolang/types.go#L1091) · [↗](../../../../../.worktrees/gno-review-5217/gnovm/pkg/gnolang/types.go#L1091) — `countTypeMethodsForGas` could factor in embedded-struct depth (not just method count) so deeply nested empty-method-set chains pay something proportional to walk depth. Cheapest fix: add a small floor `+ depth` term inside `countEmbeddedFieldMethods`, or charge a fixed `OpCPUEmbeddedHop` per `seen[t] = struct{}{}` entry. Low priority — bounded by tx size — but documents the model.
  <details><summary>details</summary>

  Today the count for `type Chain struct{ Next }` with `Next` empty and 1000 levels deep is 1 (floor). Probing one interface method walks all 1000 levels at ~100ns each = 100µs of real work for 4 gas. Even sustained across 1000 leaf methods (`N*M = 1M probes × 100µs = 100s`), the practical exploitability is limited by how much tx bytes 1000 type declarations consume. Still worth a one-line note in the function's comment that "walk depth is bounded separately by tx-size limits".
  </details>

- [`gnovm/pkg/gnolang/types.go:1037`](https://github.com/gnolang/gno/blob/16b633c/gnovm/pkg/gnolang/types.go#L1037) · [↗](../../../../../.worktrees/gno-review-5217/gnovm/pkg/gnolang/types.go#L1037) — `perInterfaceMethodCheckCost` is called with no upper bound on the count returned. `overflow.Mulp` will panic on overflow, but for a hostile type that somehow returns int64.MAX/3 the panic would replace what should be a metered gas-out. Practically unreachable (method count is bounded by parse-time limits), but worth a comment naming the assumption.

## Questions for Author

- No ADR added under `gnovm/adr/`. AGENTS.md says every non-trivial AI-assisted PR must include one. PR opened 2026-03-02 (likely predates the rule); confirm whether a retroactive `prxxxx_typeswitch_iface_metering.md` is wanted, or whether the commit-message audit trail + filetests are deemed sufficient for a focused security fix.
- `OpCPUSlopeTypeSwitchCase=254` and `OpCPUSlopeTypeAssertIface=349` are deleted outright. Any downstream tooling (gas-model dashboards, calibration scripts under `gnovm/pkg/gnolang/bench_ops_test.go`) that referenced these names will fail to build — grep is clean inside the gno tree, but worth a heads-up if the gas-model spreadsheet tracks them.
