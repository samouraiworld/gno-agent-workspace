# PR #5646: fix(gnovm): meter BigInt and BigDec comparison operators

**URL:** https://github.com/gnolang/gno/pull/5646
**Author:** davd-gzl | **Base:** master | **Files:** 3 | **+149 -9**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Moves per-N CPU gas charging for BigInt/BigDec comparisons out of the
`doOpEql`/`doOpLss` handlers and into the `is*` helpers (`isEql`, `isLss`,
`isLeq`, `isGtr`, `isGeq`). The result is that all six comparison operators
(`==`, `!=`, `<`, `<=`, `>`, `>=`) now charge per-N gas for both `BigintKind`
and `BigdecKind` operands. Prior to this PR, only BigInt `==` and `<` were
metered upfront in their handlers; the other four operators and all BigDec
compares were unmetered.

The charge lives inside the `case BigintKind:` / `case BigdecKind:` branches
of each `is*` helper, just before the existing `lv.V.(BigintValue).V` /
`lv.V.(BigdecValue).V` unpack. `doOpNeq` inherits the charge for free because
it delegates to `isEql`.

Two new placeholder constants are added in `machine.go`:
- `OpCPUSlopeBigDecEql = 20` — used by `==`/`!=`.
- `OpCPUSlopeBigDecCmp = 20` — used by `<`/`<=`/`>`/`>=`.

Both reuse the BigDec `Sub` fit (`= 20`) since no dedicated `BenchmarkOpCmp`
exists for BigDec; the split mirrors the existing BigInt `Eql`/`Lss` split so
the two sides can be calibrated independently later. BigInt reuses the
existing `OpCPUSlopeBigIntLss` constant for all four lex ops.

A 132-line ADR (`gnovm/adr/pr5646_bigint_bigdec_compare_gas.md`) documents
the rationale, alternatives considered, and the load-bearing point that
**these runtime paths are unreachable from `.gno` source today**:
`UntypedBigintType`/`UntypedBigdecType` are internal preprocess types,
constant-folded at preprocess (`preprocess.go:1397-1425`), and have no
user-facing keyword. The metering is defensive, matching the existing
arithmetic bigint/bigdec metering (also unreachable today).

## Test Results
- **Existing tests:**
  - `go test ./gno.land/pkg/sdk/vm/ -run Gas` — PASS
  - `go test ./gnovm/pkg/gnolang/ -run 'TestPreprocess|TestMachine|TestEval'` — PASS
  - `go test ./gnovm/pkg/gnolang/ -run TestFiles -test.short` — 2 unrelated failures
    (`TestFiles/types/eql_0f0`, `TestFiles/types/or_f0`); verified pre-existing on
    `origin/master` with the same diff.
  - `gofmt -d` on both touched files: clean.
  - CI status: all green.
- **Edge-case tests:** skipped (no reachable runtime path to exercise — ADR
  acknowledges this).

## Critical (must fix)
- None.

## Warnings (should fix)
- [ ] `gnovm/pkg/gnolang/op_binary.go:653`, `:702`, `:751` — `isLeq`,
  `isGtr`, and `isGeq` all call `m.incrCPUBigInt(lv, rv, OpCPUSlopeBigIntLss)`.
  The ADR justifies the reuse ("`Cmp()` work is identical regardless of
  comparator"), but the constant name `…Lss` is misleading when seen at the
  `<=`/`>`/`>=` call sites — a future reader has to cross-reference the ADR
  or master's `OpCPUSlopeBigIntLss = 9` definition to know it's intentional.
  The BigDec side avoids this by using a generic `OpCPUSlopeBigDecCmp`. Pick
  one:
  - **Preferred:** add `OpCPUSlopeBigIntCmp = OpCPUSlopeBigIntLss` (or
    promote `Lss` → `Cmp` and keep `Lss` as a deprecated alias) and use the
    new name in `isLeq/Gtr/Geq`. Keeps `Eql` separate, matches BigDec
    naming exactly.
  - Or rename BigDec to `OpCPUSlopeBigDecLss` reused (worse: contradicts ADR
    rationale that `Eql`/`Cmp` may calibrate differently).

## Nits
- [ ] `gnovm/pkg/gnolang/machine.go:1503-1504` — inserting the TODO
  comment between `OpCPUSlopeBigDecSub` and `OpCPUSlopeBigDecEql` splits
  the BigDec const block into two gofmt alignment groups: `Add`/`Sub` now
  have a single space before `=`, while `Eql`/`Cmp`/`Inc`/`Dec` (same
  19-char name length) have a double space to line up with the 20-char
  `Uneg`/`MulQ`/`QuoQ`. Visually inconsistent. Move the TODO comment above
  `BigDecAdd` (or to a doc comment on `BigDecEql` only) so the whole BigDec
  block stays one alignment group. `gofmt` accepts both; pick the one that
  reads better.
- [ ] `gnovm/adr/pr5646_bigint_bigdec_compare_gas.md` — filename embeds the
  PR number, which couples the ADR to its origin PR rather than to the
  decision. Other ADRs in `gnovm/adr/` use topic-based names. If the
  convention is "name by topic", consider renaming
  (`bigint_bigdec_compare_gas.md`). Minor — depends on local convention.

## Missing Tests
- [ ] No automated regression test for the new charges. The ADR is correct
  that no `.gno` filetest can exercise the path. But a Go-level unit test in
  `gnovm/pkg/gnolang/` that constructs `TypedValue{T: UntypedBigintType, V:
  BigintValue{V: big.NewInt(N)}}` directly, calls
  `isEql/Lss/Leq/Gtr/Geq(m, lv, rv)` against a `*Machine` with a
  `GasMeter`, and asserts `m.GasMeter.GasConsumed() > 0` would lock in the
  intent. It would also catch a future "is the helper still calling
  `incrCPUBigInt`?" regression — without it, somebody can refactor the
  `case BigintKind:` branch and silently drop the meter call again (which
  is exactly the bug this PR fixes). Same critique applies to the existing
  arithmetic metering, but introducing the pattern here is cheap.

## Suggestions
- The four `case BigintKind: m.incrCPUBigInt(lv, rv, OpCPUSlopeBigIntLss);
  lb := lv.V.(BigintValue).V; rb := rv.V.(BigintValue).V; return lb.Cmp(rb)
  <CMP> 0` blocks across `isLss/Leq/Gtr/Geq` are identical except for the
  comparator. Same for BigDec. A tiny helper
  `bigintCmp(m, lv, rv) int { ...; return lv.V.(BigintValue).V.Cmp(rv.V.(BigintValue).V) }`
  collapses the duplication and makes the gas-charge site singular — easier
  to audit, easier to test. Not blocking; the current duplication is at
  least uniform.
- The "open question" in the ADR (`is dead defensive metering worth keeping
  across the codebase?`) is the right question and worth filing as a
  tracking issue rather than leaving as floating text in a merged ADR.
  Otherwise it will get lost.

## Questions for Author
- Why introduce two distinct BigDec constants (`Eql` and `Cmp`) when both
  are `= 20` placeholders, neither path is reachable today, and the BigInt
  side achieves "calibrate separately later" with the existing `Eql`/`Lss`
  split using only one of them (`Lss`) for the four lex ops? The ADR
  rationale ("symmetry with BigInt") feels weakened by the fact that the
  BigInt side itself reuses `Lss` across four ops — i.e., the symmetric
  thing to do would be one BigDec constant reused four times, not two.
- The ADR notes that `switch x { case y: ... }` on a runtime BigInt/BigDec
  value is "covered" via `doOpSwitchClauseCase → isEql`. Confirmed by
  reading `op_switch.go`? Worth a one-line pointer in the ADR if so.

## Verdict
APPROVE — defensive metering wired into the natural place (the `is*`
helpers), matches the existing arithmetic pattern, no behavior change on
any reachable code path. The naming/alignment cleanups (`OpCPUSlopeBigIntLss`
reuse for `Leq/Gtr/Geq`, BigDec const alignment) and the missing Go-level
regression test are the only items worth folding in before merge; the
question about `Eql`/`Cmp` split for BigDec is a design choice that the
author should affirm rather than fix.
