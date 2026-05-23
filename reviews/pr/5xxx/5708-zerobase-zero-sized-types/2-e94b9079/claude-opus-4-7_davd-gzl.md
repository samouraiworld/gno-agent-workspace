# PR #5708 (round 2): fix(gnovm): implement zerobase semantics for zero-sized types

URL: https://github.com/gnolang/gno/pull/5708
Author: ltzmaxwell | Base: master | Files: 8 | +210 -13 (since 350d630e3: +64 -48)
Reviewed by: davd-gzl | Model: claude-opus-4-7
Prior review: [1-350d630e3/claude-opus-4-7_davd-gzl.md](../1-350d630e3/claude-opus-4-7_davd-gzl.md)

**Verdict: NEEDS DISCUSSION** — round-1 Critical (parity gap on arrays of zero-sized elements) is fully addressed: `isZeroSizeType` now recurses (`ct.Len == 0 || isZeroSizeType(ct.Elt)`) and ptr12b/ptr12c filetests pin the new behavior. Both round-1 Warnings remain unaddressed in code — cross-tx pointer-identity break and cross-realm HIV ownership leak — and the cross-tx asymmetry question to the author is still open. Neither is a bug in the new commits, but both shape the contract worth pinning before merge.

## What changed since 350d630e3

Six commits, two substantive, four cosmetic:

| commit | substance |
|---|---|
| `a69061f7` chore: dedup ptr12a test logic | moved `[0]int` cases out of `ptr12a.gno` (kept in `ptr12.gno`) |
| `f4cb2cdb` chore: move zerobaseAllocs to end of Machine struct | addressed round-1 Nit 3 |
| `2496186b` fmt | — |
| `3d1a8c7d` fix isZeroSizeType for arrays of zero-sized elements | addressed round-1 Critical |
| `e94b9079` note Go discrepancy in ptr12b/c tests | clarifies Go's escape-analysis non-determinism |

Plus a `// see GetZerobase` short-form comment in [`op_expressions.go:216`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/op_expressions.go#L216) addressing round-1 Nit 4 (doc duplication).

## Round-1 follow-up

| round-1 finding | severity | status |
|---|---|---|
| parity gap: `[N]T` with N>0 and zero-sized T | Critical | **fixed** ([`types.go:1557`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/types.go#L1557)); ptr12b/c added |
| cross-tx pointer-identity break once persisted | Warning | **unaddressed**; `GetZerobase` doc hedges "within a machine execution" but no test pins the boundary |
| cross-realm HIV ownership leak in one tx | Warning | **unaddressed**; `GetZerobase` still calls `NewHeapItem` which stamps `PkgID = currentRealmID` |
| `&CompositeLit{}` double-allocate in `doOpRef` | Nit | **unaddressed**; still allocates a fresh HIV in `PopAsPointer2` before discarding for the shared zerobase |
| `uverse.go new` two-branch `PushValue` duplication | Nit | **unaddressed** |
| `zerobaseAllocs` field alignment | Nit | **fixed** ([`machine.go:55-60`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L55-L60)) |
| spec-citation duplication | Nit | **fixed** ([`op_expressions.go:216`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/op_expressions.go#L216)) |
| cross-tx asymmetry question | Question | **no response** |

## Verification

All four `ptr12*` filetests pass on the new HEAD:

```bash
# from a gno checkout:
gh pr checkout 5708 -R gnolang/gno
go test -v -run 'TestFiles/ptr12' -test.short -timeout 60s ./gnovm/pkg/gnolang/
# === RUN   TestFiles/ptr12.gno   --- PASS
# === RUN   TestFiles/ptr12a.gno  --- PASS
# === RUN   TestFiles/ptr12b.gno  --- PASS
# === RUN   TestFiles/ptr12c.gno  --- PASS
```

CI is red but **unrelated**: `gno-checks / lint` and `params_valset_rotation_throttle` both fail on master HEAD too (run `26301907221` and `26290866204` on `master`). Not a PR-introduced regression.

## Critical (must fix)

None. Round-1 Critical resolved.

## Warnings (should fix)

Both carry over from round 1 unchanged; no further analysis needed beyond what's in [the round-1 review](../1-350d630e3/claude-opus-4-7_davd-gzl.md). Briefly:

- **[cross-tx pointer-identity break once persisted]** [`machine.go:300-312`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L300-L312) — `zerobaseAllocs` lives on the live `Machine`. Once a `new(struct{})` pointer reaches realm storage, the persisted HIV gets a stable `ObjectID`; the next tx starts with `m.zerobaseAllocs == nil` and mints a new HIV with a different `ObjectID`. `globalP == new(struct{})` flips from true to false on reload. `GetZerobase`'s new doc string hedges "within a machine execution" — accurate but doesn't pin the boundary with a filetest.

- **[cross-realm HIV ownership leak in one tx]** [`machine.go:300-312`](../../../../../.worktrees/gno-review-5708/gnovm/pkg/gnolang/machine.go#L300-L312) — `NewHeapItem` stamps `PkgID = currentRealmID`. The first realm to call `new(struct{})` owns the cached HIV; if a second realm later allocates the same type and persists the result, `r/B`'s storage rent attributes to `r/A`. Fix is the same as round 1: skip `stampPkgID` inside `GetZerobase` so `assignNewObjectID` adopts via the "PkgID.IsZero" branch.

## Nits

- [`gnovm/tests/files/ptr12.gno:16-23`](../../../../../.worktrees/gno-review-5708/gnovm/tests/files/ptr12.gno#L16-L23) — the `if &x != &y { ... }` block has only commented-out body. The intent is "Gno chose `may equal`, so don't assert the inequality Go's gc compiler would observe," but as written it's a dead branch — the predicate is evaluated and discarded. A plain comment block above the `if p != q` assertion (or just deleting the empty `if`) reads more clearly. Pre-existing from 350d630e3; only flagging now because it's adjacent to the test edits.
- Round-1 nits 1 (`doOpRef` double-allocate) and 2 (`uverse.go new` two-branch duplication) remain. Both are small and optional but cheap to land.

## Missing Tests

- **[cross-tx persistence]** still no filetest rounds a zero-sized pointer through realm save/load and asserts equality against a fresh `new(T)` in a follow-up tx. The new doc on `GetZerobase` makes the within-machine scoping explicit in prose — a test would lock it in mechanically and document the failure mode for future readers.
- **[cross-realm reuse]** still no test pinning storage-rent attribution when two realms allocate the same zero-sized type in one tx. The PR notes ran `Gas`, `integration`, and `TestFiles` per the test plan; none of those exercise cross-realm sharing of the cached HIV.

## Suggestions

- The new `ptr12b.gno`/`ptr12c.gno` comments ("In Go, escape analysis may or may not route these through `runtime.zerobase`") are more accurate than what I claimed in round 1 — Go's `runtime.newobject` does return `&zerobase` for size-0 allocations, but escape-to-stack can bypass that and give distinct addresses. The author's framing ("Gno consistently returns zerobase") correctly characterizes Gno's choice as picking determinism over Go's compiler-dependent outcome. No change requested, just acknowledging the round-1 simplification was lossy.

## Questions for Author

- The cross-tx asymmetry question from round 1 is still open: is the within-machine scoping intentional (in which case a filetest locking it in would help future readers), or an artifact of the per-`Machine` cache choice (in which case Warning 1 needs a structural fix)? The new `GetZerobase` doc says "within a machine execution" but stops short of confirming intent.
- Was the cross-realm storage-rent attribution path considered? The fix is small (skip `stampPkgID` in `GetZerobase`, let `assignNewObjectID`'s zero-PkgID branch take over), so deferring it to a follow-up is reasonable — but the decision is worth making explicit.
