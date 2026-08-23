# PR [#6058](https://github.com/gnolang/gno/pull/6058): fix(gnovm): let switch/if cases shadow their init statement's names

URL: https://github.com/gnolang/gno/pull/6058
Author: thehowl | Base: master | Files: 17 | +615 -76
Reviewed by: davd-gzl | Model: claude-opus-5 (high, deep) | Commit: 391840aa7 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6058 391840aa7`

## Overview

A `switch` or an `if` may declare a name in its init statement, and Go lets the body of a case
declare that name again, hiding the outer one for the rest of the body. GnoVM did not. Neither
statement gets a runtime block of its own: the VM makes one block for the whole statement and every
case runs inside it, so each case's name table opens with a copy of the init's names. A
redeclaration in the case body found its name already in that table and wrote over the copy, which
killed the preprocessor when the two types differed and silently replaced the outer value when they
matched. The branch gives the redeclaration a second entry for the same name and reads a name back
from the last entry, so the copy and the shadow hold their own values. A use of the name before the
redeclaration still finds the outer one, because the second entry carries no type until the
declaration is reached.

**Verdict: REQUEST CHANGES** — the shadowing fix matches Go on every shape measured against `go run`,
and a `const` shadow is the one that does not: a use of the outer name before it yields a zero value
where the merge base refused to compile the program, which [#6060](https://github.com/gnolang/gno/pull/6060)
settles and this branch alone does not, so the merge order needs saying (2 warnings, 3 missing
tests, 3 suggestions, 2 nits).

## Verify first

- [`gnovm/pkg/gnolang/nodes.go:2341`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2341) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2341): the second slot is appended with `isConst` carried through, and `Consts` is keyed by name. Drop [`switch58.gno`](tests/switch58.gno) into `gnovm/tests/files/` and confirm the first line prints `1`.
- [`gnovm/pkg/gnolang/op_exec.go:736`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/op_exec.go#L736) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/op_exec.go#L736): the truncation decides what every later clause reads. Run [`fallthrough-alloc_test.go`](tests/fallthrough-alloc_test.go) and confirm the accounted bytes are flat across chain lengths.
- [`gnovm/pkg/gnolang/nodes.go:2003-2011`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2003-L2011) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2003): revert the map build to first-wins and confirm something reddens.

## Summary

`StaticBlock.Reserve` may now append a second slot for a name the case block already holds, when the
existing slot is one of the leading names copied in from the `IfStmt` or `SwitchStmt`, and
`GetLocalIndex` reads a name back from the last slot instead of the first. `numFauxCopiedNames` is
the boundary between the two regions, and a path below it resolves to the parent node, which keeps
heap marking and `ATTR_HEAP_USES` on the right block. The branch also carries
[#6056](https://github.com/gnolang/gno/pull/6056) as its first commit, byte-identical to the version
merged as 0cf310707, and one hunk beyond it: `ExpandWith` now assigns `b.Source` above its
equal-size early return.

## Fix

`Reserve` used to no-op whenever the name was already present, so a `v := "s"` in a case body
retargeted the copy of the switch's own `v`. It now compares the reserving declaration against the
slot's `NameSource` on `(Origin, Type, Index)` and appends when they differ, which keeps the copy
loop idempotent across the re-preprocessing stdlib nodes take. The load-bearing constraint is that
the shadow's slot stays untyped until `preprocess1` reaches its declaration:
[`fillNameExprPath`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L5813-L5817) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/preprocess.go#L5813) already reads "reserved but not yet typed" as not defined here and walks up, which is what makes a
use before the shadow find the outer name.

## Benchmarks / Numbers

Bytes accounted by the allocator across `RunMain`, for a switch of N clauses chained by
`fallthrough`, each declaring four locals. Harness and full output in
[`tests/fallthrough-alloc.md`](tests/fallthrough-alloc.md).

| Clauses | `391840aa7` | `0cf310707` master | `754780601` merge base |
|---|---|---|---|
| 1 | 2760 | 2760 | 2760 |
| 2 | 2920 | 2920 | 2760 |
| 10 | 4200 | 4200 | 2760 |
| 50 | 10600 | 10600 | 2760 |

Gas for the 50-clause case: 153270, 153270, 150379. VM cycles are identical on all three.

## Warnings (should fix)

- **[correctness]** `gnovm/pkg/gnolang/nodes.go:2341` — a `const` shadow makes a use of the outer name before it evaluate to a zero value, where the merge base rejected the program.
  <details><summary>details</summary>

  `defineNew` appends the shadowed name to [`Consts`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2495-L2497) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2495), which
  [`getLocalIsConst`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L1916-L1918) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L1916) reads with
  `slices.Contains`, so the name is const for the copy slot too and a use textually before the
  declaration const-folds against a slot that carries no value. `go run` prints `1` and `c`; the head
  prints `0` and `c`; the merge base answers `StaticBlock.Define2(v) cannot change const status`. The
  same divergence appears in an `if` branch, an `else` branch, a clause reached by `fallthrough`, and
  through a closure called before the shadow. The name-keyed lookup predates the branch, and an
  ordinary nested block prints `0` at the merge base as well, which is what the ADR's parity argument
  rests on. What the branch changes is that a case body reaches it at all, and reaches it without a
  diagnostic. [#6060](https://github.com/gnolang/gno/pull/6060) replaces the name-keyed const checks
  with `GetIsConstAt` on the NameExpr's resolved path, and its body names this branch as where the
  bug was found; merging 6830e2549 into this head reports no conflict and turns
  [`tests/switch58.gno`](tests/switch58.gno) green with `switch52.gno` still passing. Refusing the
  append here when `isConst` is set restores the merge base's rejection and reddens `switch52.gno`,
  which the branch itself adds, so position-sensitive resolution is the only shape that satisfies
  both. Fix: land [#6060](https://github.com/gnolang/gno/pull/6060) first, or say in the body that
  this one carries the divergence until it does. Measurements in
  [`tests/const-shadow.md`](tests/const-shadow.md) and
  [`tests/const-shadow-with-6060.md`](tests/const-shadow-with-6060.md).
  </details>

- **[gas]** `gnovm/pkg/gnolang/op_exec.go:736` — each `fallthrough` charges allocation gas for the target clause's whole name count on a slice that is only re-sliced within its existing capacity.
  <details><summary>details</summary>

  The truncation lowers `len(b.Values)` to the switch's own name count, so
  [`ExpandWith`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/values.go#L3149-L3151) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/values.go#L3149) computes `newNames` against that rather than against the previous clause's count and calls
  `AllocateBlockItems` for the target clause's whole set. On that path
  [`growBlockValues`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/values.go#L2879-L2881) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/values.go#L2879) re-slices inside the existing capacity, so nothing is allocated for the charge, and
  [`Allocate`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/alloc.go#L327-L349) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/alloc.go#L327) both charges gas and counts toward `maxBytes`, which drives the GC callback. Accounted bytes grow
  with the chain length while the footprint does not, per the table above. `cap(b.Values)` does not
  move, so `GetShallowSize` and the storage deposit are unaffected. This does not block the branch:
  the line arrives through [#6056](https://github.com/gnolang/gno/pull/6056) and the head and current
  master columns are identical. Fix: skip `AllocateBlockItems` when `growBlockValues` re-slices within
  capacity, or charge only growth past the block's high-water mark.
  </details>

## Missing Tests

- **[coverage]** `gnovm/tests/files/switch53.gno` — no filetest reaches a type switch, and the branch rewrites where the type switch variable is defined.
  <details><summary>details</summary>

  `defineSwitchVar` in [`preprocess.go:1004-1010`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L1004-L1010) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/preprocess.go#L1004) replaces three `last.Define` calls with a write to the slot at `numFauxCopiedNames()`, and the
  `NameSource` comparison in `Reserve` names the type switch variable's call site as the reason it
  compares by declaration rather than by pointer. Nothing in `switch47.gno` through `switch53.gno` or
  `if9.gno` constructs one. [`tests/typeswitch11.gno`](tests/typeswitch11.gno) covers a clause body
  shadowing the type switch's init name and the type switch variable itself shadowing one; it panics
  at the merge base with `cannot change .T; was string, new int` and passes at the head, and its
  `// Output:` is `go run`'s.
  </details>

- **[coverage]** `gnovm/pkg/gnolang/nodes.go:2003` — the switch from first-wins to last-wins in the map build has no test, and a filetest cannot pin it.
  <details><summary>details</summary>

  A case block only reaches the map branch past `nameIndexThreshold`, 32, and an end-to-end fixture
  passes under both semantics: with the map reverted to first-wins, `Reserve` sees the wrong index,
  appends a third slot for the same name and restores self-consistency, so program output never
  moves. `GetLocalIndex` itself does diverge. The unit test in
  [`tests/nodes_test-last-wins.patch`](tests/nodes_test-last-wins.patch) asserts the two branches
  agree at widths 20 and 40; it passes at the head and fails on the revert with
  `width 40: want the shadow's slot`.
  </details>

- **[coverage]** `gnovm/tests/files/if9.gno` — four shapes the new filetests leave out, each of which panics at the merge base.
  <details><summary>details</summary>

  `if9.gno` covers the two branches of one `if`; an `else if` opens its own faux block with its own
  init and nothing reaches it. `switch49.gno` reaches a `default` but only reads the init name there.
  Every existing fixture has exactly one init name, so `numFauxCopiedNames() > 1` is never exercised
  with a shadow. And no fixture leaves a shadowing clause by labelled `break` or `goto`, which take
  the `GOTO` arm of [`op_exec.go:708-717`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/op_exec.go#L708-L717) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/op_exec.go#L708) and restore frames without touching `b.Values`. Four files, all passing at the head, all panicking
  at the merge base, all matching `go run` byte for byte: [`if10.gno`](tests/if10.gno),
  [`switch54.gno`](tests/switch54.gno), [`switch55.gno`](tests/switch55.gno) and
  [`switch57.gno`](tests/switch57.gno), the last covering both slots of one name heap-captured in the
  same clause, which `switch53.gno` splits across two switches.
  </details>

## Suggestions

- **[dead code]** `gnovm/pkg/gnolang/nodes.go:2334-2339` — the `NSTypeSwitch` branch cannot be reached, and the comment above it plus the ADR paragraph behind it name a rejection that never happens.
  <details><summary>details</summary>

  The type switch variable is reserved at
  [`preprocess.go:567`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L567) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/preprocess.go#L567) right after the copy loop has created exactly `Parent.GetNumNames()` slots, so its index equals
  `numFauxCopiedNames()` and the return four lines above fires first. Two independent probes replaced
  the branch body with a panic and ran the `switch`, `typeswitch`, `if`, `type` and `select` filetest
  families: never reached. Deleting the branch leaves `TestFiles` green in full, along with
  `TestStaticBlock`, `TestRunMemPackage`, `TestDebug` and `TestPreprocess`, and takes `Reserve` from 28
  lines to 22. `leave it to Define2 to reject` is also
  the wrong mechanism: `switch t := x.(type) { case int: t := 5 }` is rejected by `go/types`, with
  Go's own `no new variables on left side of :=`, at the head and at the merge base alike. Fix: delete
  the branch, and reword
  [`pr6058_faux_block_shadowing.md:81-88`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/adr/pr6058_faux_block_shadowing.md?plain=1#L81-L88) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/adr/pr6058_faux_block_shadowing.md#L81) to say the slot index is what holds, since that paragraph argues the opposite.
  </details>

- **[hardening]** `gnovm/pkg/gnolang/nodes.go:2365-2370` — `defineFauxCopy`'s only check is compiled out of every shipped build, and no CI job builds the tag.
  <details><summary>details</summary>

  Without `debugAssert` the function is two writes with nothing tying `idx` to `n`, so a boundary that
  ever drifts mis-types a name instead of panicking. `debugAssert` appears in one recipe,
  [`gnovm/Makefile:118`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/Makefile#L118) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/Makefile#L118), and no workflow calls it:
  [`ci-dir-gnovm.yml`](https://github.com/gnolang/gno/blob/391840aa7/.github/workflows/ci-dir-gnovm.yml) delegates to `_ci-go.yml`, whose test step passes no `-tags`. The check is cheap enough to keep
  on: the two call sites are [`preprocess.go:1009`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L1009) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/preprocess.go#L1009) and [`preprocess.go:4022`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L4022) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/preprocess.go#L4022), both
  at preprocess time and neither in the VM loop, so it costs one `Name` comparison per copied name per
  clause and nothing for a statement with no init. `TestFiles` is green in full with it unconditional
  and never trips it.
  `pr6058_faux_block_shadowing.md:142-147` reads "enforced, not just documented", which holds for a
  local run and not for any check or any shipped binary. Fix: drop the `if debugAssert` wrapper here,
  and say in the ADR that the `defineNew` assertion is local-only.
  </details>

- **[docs]** `gnovm/adr/pr6058_faux_block_shadowing.md:58` — two counts in the ADR do not match the tree.
  <details><summary>details</summary>

  "the eleven `Reserve` call sites" is sixteen, all in `preprocess.go`, at lines 409, 439, 450, 456,
  481, 517, 525, 532, 540, 549, 563, 567, 577, 585, 594 and 777. The argument the sentence makes still
  holds, since every one of the sixteen passes a stable triple. Line 143's "`defineNew` is the sole
  append path" has one more exception: the amino decoder appends to `Names` at
  [`pb3_gen.go:12685`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/pb3_gen.go#L12685) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/pb3_gen.go#L12685) and 12707, which the `nameIndex` field comment already carves out and the ADR does not.
  </details>

## Nits

- **[decay]** `gnovm/pkg/gnolang/op_exec.go:736` — `ss.GetNumNames()` open-codes the boundary `numFauxCopiedNames` defines, and the two are equal only because `pushInitBlock` sets the clause block's parent to `ss`.
- **[decay]** `gnovm/pkg/gnolang/nodes.go:1671-1678` — the `nameIndex` contract comment still names `Define2` as the append path and as the thing that maintains the map. After this branch `Define2` never appends; `defineNew` does, and `Reserve` is a second entry to it. Outside the diff, so not posted.

## Verified

- `TestFiles` is green in full at 391840aa7 with 6830e2549, the head of [#6060](https://github.com/gnolang/gno/pull/6060), merged in.
- The bug reproduces at the merge base and the branch fixes it: `switch v := 1; v { case 1: v := 99; println("inner", v); fallthrough; case 2: println("outer", v) }` prints `outer 99` at 754780601 where `go run` prints `outer 1`.
- The shadowing surface was run side by side against `go run`, crossing statement kind, shadow form, position, multiplicity, `fallthrough` direction, `goto` and labelled `break`, closures, `defer`, escaping pointers, realms, and a clause wide enough to drive the map branch of `GetLocalIndex`. Every program matched Go at the head, and the `const` shadow above is the only divergence found. The probe trees were removed at the end of each run, so what survives is the fixtures under `tests/` and the two suggested edits, every one of them re-run here.
- The branch fixes two further defects with no fixture: `p = &x` in a clause read `99` after a `fallthrough` into a clause declaring `y := 99` at the merge base, and a `defer` capturing a clause local read the next clause's value.
- `ExpandWith` assigning `b.Source` above the early return repairs the debugger on current master, not only on this branch. The same fixture answers `unexpected block size shrinkage: 1 vs 0` at 754780601, `runtime error: index out of range [0] with length 0` at 0cf310707, and `name y not declared` at the head. Measurements in [`tests/debugger-source-after-fallthrough.md`](tests/debugger-source-after-fallthrough.md).
- `numFauxCopiedNames` reading `Parent.GetNumNames()` live never drifts. Instrumented twice, independently, to panic on any change of the value returned for a given block, then run over the switch, if, type, closure, heap, for, range, goto, label, recover and defer filetest families plus the preprocess and machine unit tests: no trip. `Init` is transcribed before the clauses and nothing appends to an `IfStmt` or `SwitchStmt` block afterwards.
- `cap(b.Values)` is identical at the head and at the merge base for a switch whose clauses cross the pool capacity, so the truncation moves no storage deposit.
- `TestFiles` is green at 391840aa7 under go1.25.9, the version `go.mod` pins. The ten golden failures the description reports are the local Go 1.26 drift recorded in the workspace, and they do not appear under the pinned toolchain.

## Open questions

- The allocation change belongs to [#6056](https://github.com/gnolang/gno/pull/6056), already merged, so the place to settle it is master rather than this branch. Worth an issue naming the measurement.
- The ADR's `getLocalIsConst` paragraph says fixing it for both block kinds is "left to a follow-up" without naming [#6060](https://github.com/gnolang/gno/pull/6060), which is that follow-up and is open. One sentence in the ADR would save the next reader the search.
- `gnovm/tests/files/switch56.gno` in the round directory is a 33-name clause with a shadow. It passes under both map semantics, so it exercises the branch without asserting it, and is worth adding only beside the unit test.
