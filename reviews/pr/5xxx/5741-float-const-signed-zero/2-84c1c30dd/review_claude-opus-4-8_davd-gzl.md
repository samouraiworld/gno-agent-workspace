# PR [#5741](https://github.com/gnolang/gno/pull/5741): fix(gnovm): convert underflowing negative float constant to +0

URL: https://github.com/gnolang/gno/pull/5741
Author: ltzmaxwell | Base: master | Files: 6 | +209 -6
Reviewed by: davd-gzl | Model: claude-opus-4-8 (max) | Commit: 84c1c30dd (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5741 84c1c30dd`

Round 2. Head advanced a6dc98e3b → 84c1c30dd with real PR content: the round-1 blocker (negative-overflow leak) is fixed by a magnitude-based narrowing check ([686f0b057](https://github.com/gnolang/gno/commit/686f0b057)), constant folding gained a `-0`→`+0` normalization in `evalConst` for the negation case ([944254865](https://github.com/gnolang/gno/commit/944254865)), and the test corpus grew `float13` (negative overflow) and `float14` (verbatim Go corpus). Verdict flips REQUEST CHANGES → APPROVE.

**TL;DR:** In Go, a numeric constant has no negative zero, so a tiny negative constant like `-1e-50` that underflows to zero becomes `+0`, not `-0`, and a constant that overflows the target type is a compile error, not `±Inf`. Gno diverged on both. This PR routes constant float conversions through constant space and normalizes folded `-0` results, so both now match Go, while runtime float ops still keep the sign of an underflowed zero per IEEE.

**Verdict: APPROVE** — round-1's one-sided overflow leak is fixed and pinned by `float13`; constant `-0`→`+0` and Go-matching overflow rejection verified against a side-by-side Go run. Two open maintainer threads from [@thehowl](https://github.com/gnolang/gno/pull/5741) remain (use the softfloat `inf32` constant at the check; an adjacent pre-existing `bigint`→float divergence), both non-blocking and left to the author.

## Summary
Constants in Go carry no signed zero: any constant that evaluates to zero is `+0`, and an overflowing constant conversion is a compile error, not `±Inf` ([golang/go#12621](https://github.com/golang/go/issues/12621)). The fix has three arms. `posZero` clears the sign bit whenever a bigdec-to-float conversion lands on zero, covering the untyped-constant path ([values_conversions.go:1414](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/pkg/gnolang/values_conversions.go#L1414) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/values_conversions.go#L1414)). The `float64→float32` narrowing now rejects overflow on magnitude, either sign, and clears a `-0` result for typed constants ([values_conversions.go:1013-1029](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/pkg/gnolang/values_conversions.go#L1013-L1029) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/values_conversions.go#L1013)). A new branch in `evalConst` normalizes a folded `-0` result (e.g. from negation of a typed zero) to `+0` after machine evaluation ([preprocess.go:4431-4445](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/pkg/gnolang/preprocess.go#L4431-L4445) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/preprocess.go#L4431)), while the existing branch routes explicit `float32(c)`/`float64(c)` of float-valued constants through `convertConst` (constant space) ([preprocess.go:1595-1606](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/pkg/gnolang/preprocess.go#L1595-L1606) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/preprocess.go#L1595)). Runtime float ops still keep the sign of an underflowed zero, matching IEEE and Go. Sibling PR [#5864](https://github.com/gnolang/gno/pull/5864) applies the same `-0`→`+0` fold to MsgCall float args in `convertFloat` (`gno.land/pkg/sdk/vm/convert.go`), the maketx entry point this PR does not touch.

## Examples
Constant vs runtime, verified against real Go on the same inputs at 84c1c30dd:

| Written form | Go | gno at 84c1c30dd |
|---|---|---|
| `var m = -1e-10000` (constant) | `+0` | `+0` |
| `const zt float64 = 0.0; -zt` (const negation) | `+0` | `+0` |
| `float32(-1e-50)` (constant conv) | `+0` | `+0` |
| `const sub float32 = -1e-40` (subnormal) | kept negative | kept negative |
| `float32(3.4028235e38)` (round to max) | `MaxFloat32` | `MaxFloat32` |
| `d := -math.SmallestNonzeroFloat64; d/3` (runtime) | `-0` | `-0` |
| `float32(1e39)` (const overflow, positive) | compile error | compile error |
| `float32(-3.5e38)` (const overflow, negative) | compile error | compile error |

Every row now matches Go; the last row was the round-1 finding (gno printed `-Inf`) and is fixed. The full `float9` matrix and `float14` (verbatim `go/test/fixedbugs/bug434.go`) print the same lines as the gno goldens, checked against a side-by-side Go run.

## Glossary
- signed zero: IEEE keeps `+0`/`-0`; Go constants do not, so a constant zero is always `+0`.
- bigdec: the VM's arbitrary-precision decimal backing an untyped float constant before it is typed.
- softfloat: the VM's deterministic software floating point over raw IEEE bit patterns; `-0` float32 is `1<<31`, `-0` float64 is `1<<63`.
- filetest: a `*_filetest.gno` asserted against golden `// Output:` / `// Error:` / `// TypeCheckError:` directives.
- preprocess: the static pass that resolves names, types, and constants before execution; `evalConst` folds constant expressions here.
- type-check: go/types-based validation of gno source, distinct from preprocessing; drives the `// TypeCheckError:` golden.

## Fix
Round 1 flagged that the `float64→float32` narrowing validator only tested the upper bound (`Fle64(x, MaxFloat32)`), so a negative typed constant past `-MaxFloat32` narrowed silently to `-Inf`. The new validator rounds first, then rejects if the magnitude hits `±Inf`: `r := F64to32(x); return r&^(1<<31) != Float32bits(+Inf)` at [values_conversions.go:1014-1020](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/pkg/gnolang/values_conversions.go#L1014-L1020). Masking the sign bit before comparing to `+Inf` bits rejects both `+Inf` and `-Inf` while still accepting a value that rounds down to `MaxFloat32`, matching Go's "representable after rounding" rule. Separately, `evalConst` normalizes a folded `-0` result to `+0` for `Float32Kind`/`Float64Kind`, covering constant negation that the machine (runtime IEEE) would otherwise leave as `-0`.

## Critical (must fix)
None.

## Warnings (should fix)
None. Round-1's negative-overflow Warning is fixed at 686f0b057 and pinned by `float13`.

## Nits
None.

## Missing Tests
None. Round-1's missing negative-overflow golden is added as [float13.gno](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/tests/files/float13.gno) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/tests/files/float13.gno) (asserts both the VM `// Error:` and go/types `// TypeCheckError:`), and the const-negation and round-to-max cases are pinned in [float9.gno](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/tests/files/float9.gno) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/tests/files/float9.gno).

## Suggestions
- `gnovm/pkg/gnolang/values_conversions.go:1019` — [@thehowl](https://github.com/gnolang/gno/pull/5741#discussion_r3531840670) already asked to use the softfloat package's own `inf32` constant instead of `math.Float32bits(float32(math.Inf(1)))`. Style/consistency only: `math.Float32bits(float32(math.Inf(1)))` deterministically yields `0x7F800000` on every host, so there is no correctness or determinism gap, just a round-trip through `math` where `softfloat.inf32` already holds the value ([runtime_softfloat64.go:29](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/pkg/gnolang/internal/softfloat/runtime_softfloat64.go#L29) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/internal/softfloat/runtime_softfloat64.go#L29)). Already raised by the maintainer, left to the author; not posted.

## Open questions
- `bigint`→float divergence, adjacent to the touched switch: `float64(1 << 100)` is legal Go (yields `1.2676506002282294e+30`) but gno rejects it with `bigint overflows target kind`, because a bigint constant keeps the default-type (int) path and overflows int64 before the float target is considered. Confirmed: gno errors `main/...:4:7-24: bigint overflows target kind` where Go prints `1.2676506002282294e+30`. [@thehowl](https://github.com/gnolang/gno/pull/5741#discussion_r3531846521) flagged it at [preprocess.go:1603](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/pkg/gnolang/preprocess.go#L1603) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/preprocess.go#L1603) and notes `ConvertUntypedBigintTo` already handles float targets ([values_conversions.go:1311-1338](https://github.com/gnolang/gno/blob/84c1c30dd/gnovm/pkg/gnolang/values_conversions.go#L1311-L1338) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/values_conversions.go#L1311)), so adding `BigintKind` to the switch at preprocess.go:1602 would fix it. Pre-existing, not introduced by this PR, and its own signed-zero-free case; deferred-scope. Not posted (maintainer already raised it and framed it as optional for this PR).
- Two local `TestFiles` failures (`type41.gno`, `redeclaration3.gno`) are go/types message-text diffs from a newer local Go (1.26.4: `nil (untyped nil) is not a type`), on files this PR does not touch; CI's `main / test` is green. Not a PR regression; not posted.
