# PR [#5741](https://github.com/gnolang/gno/pull/5741): fix(gnovm): convert underflowing negative float constant to +0

URL: https://github.com/gnolang/gno/pull/5741
Author: ltzmaxwell | Base: master | Files: 6 | +125 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: a6dc98e3b (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5741 a6dc98e3b`

**TL;DR:** In Go, a numeric constant has no negative zero, so a tiny negative constant like `-1e-50` that underflows to zero becomes `+0`, not `-0`. Gno kept the `-0`. This PR routes constant float conversions through constant space so they match Go, and as a side effect rejects constants that overflow the target type instead of folding them to `±Inf`.

**Verdict: REQUEST CHANGES** — the signed-zero fix is correct and Go-matching, but the overflow rejection is one-sided: a negative typed constant that overflows `float32` still folds silently to `-Inf` where Go and gno's own type checker reject it.

## Summary
Constants in Go carry no signed zero: any constant that evaluates to zero is `+0`, and an overflowing constant conversion is a compile error, not `±Inf` ([golang/go#12621](https://github.com/golang/go/issues/12621)). Gno diverged on both. The fix has two arms. `posZero` clears the sign bit whenever a bigdec-to-float conversion lands on zero, covering the untyped-constant path ([values_conversions.go:1412](https://github.com/gnolang/gno/blob/a6dc98e3b/gnovm/pkg/gnolang/values_conversions.go#L1412) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/values_conversions.go#L1412)). A guard in the `float64→float32` narrowing clears `-0` for the typed-constant path ([values_conversions.go:1020-1025](https://github.com/gnolang/gno/blob/a6dc98e3b/gnovm/pkg/gnolang/values_conversions.go#L1020-L1025) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/values_conversions.go#L1020)). A new branch in `preprocess.go` routes explicit `float32(c)`/`float64(c)` of float-valued constants through `convertConst` (constant space) instead of default-type plus runtime narrowing, which is what carried the `-0` and the `±Inf` fold ([preprocess.go:1595-1607](https://github.com/gnolang/gno/blob/a6dc98e3b/gnovm/pkg/gnolang/preprocess.go#L1595-L1607) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/preprocess.go#L1595)). Runtime float ops still keep the sign of an underflowed zero, matching IEEE and Go.

## Examples
Constant vs runtime, verified against real Go on the same inputs:

| Written form | Go | gno at a6dc98e3b |
|---|---|---|
| `var m = -1e-10000` (constant) | `+0` | `+0` |
| `float32(-1e-50)` (constant conv) | `+0` | `+0` |
| `d := -math.SmallestNonzeroFloat64; d/3` (runtime) | `-0` | `-0` |
| `float32(1e39)` (const overflow) | compile error | compile error |
| `float32(-1e39)` (const overflow) | compile error | **`-Inf`** |

Last row is the finding; everything else matches Go. The full float9 matrix run under real Go prints the same eight lines as the gno golden ([float9_parity.go](tests/float9_parity.go)), and all four overflow cases are compile errors in Go while gno rejects three and folds only the typed-negative one to `-Inf`.

## Glossary
- signed zero: IEEE keeps `+0`/`-0`; Go constants do not, so a constant zero is always `+0`.
- bigdec: the VM's arbitrary-precision decimal backing an untyped float constant before it is typed.
- softfloat: the VM's deterministic software floating point over raw IEEE bit patterns; `-0` float32 is `1<<31`.
- filetest: a `*_filetest.gno` asserted against golden `// Output:` / `// Error:` / `// TypeCheckError:` directives.
- preprocess: the static pass that resolves names, types, and constants before execution.
- type-check: go/types-based validation of gno source, distinct from preprocessing; drives the `// TypeCheckError:` golden.

## Fix
Before, `float32(c)` and `float64(c)` for a float-valued constant fell through to `convertConst(store, last, n, arg0, nil)`, converting to the default type and letting the machine narrow at runtime, which preserved `-0` and folded overflow to `±Inf`. After, the new float branch routes those conversions to `convertConst(..., ct)` so they resolve in constant space, where `posZero` and the narrowing guard normalize `-0` to `+0` and the existing `IsInf` / bounds checks reject overflow. Integer sources stay on the default-type path since they carry no `-0`.

## Critical (must fix)
None.

## Warnings (should fix)
- **[overflow rejection only covers the positive side]** `gnovm/pkg/gnolang/values_conversions.go:1020-1025` — a negative typed constant that overflows `float32` narrows to `-Inf` instead of a compile error, unlike the positive case the PR now rejects and pins in `float12`.
  <details><summary>details</summary>

  The `float64→float32` narrowing validator only checks the upper bound: `Fle64(tv.GetFloat64(), MaxFloat32)`, flagged by the in-code TODO "doesn't account for negative values" at [values_conversions.go:1016](https://github.com/gnolang/gno/blob/a6dc98e3b/gnovm/pkg/gnolang/values_conversions.go#L1016) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/values_conversions.go#L1016). For `-1e39` the check passes because `-1e39 <= MaxFloat32`, so no overflow error fires and the value narrows to `-Inf`. The PR newly makes this path user-reachable for typed constants and newly rejects the positive counterpart ([float12.gno](https://github.com/gnolang/gno/blob/a6dc98e3b/gnovm/tests/files/float12.gno) · [↗](../../../../../.worktrees/gno-review-5741/gnovm/tests/files/float12.gno)), so the two sides now disagree. Real Go rejects `float32(-1e39)` with `cannot convert big (constant -1e+39 of type float64) to type float32`, and gno's own go/types parity checker emits the same message while the VM prints `-Inf` — a VM-vs-type-checker split that a future negative-overflow filetest would catch as a parity failure. Untyped negative overflow (`float32(-1e39)` with no `const`) already errors via the `IsInf` guard, so only the typed-constant narrowing leaks. Ready mirror of `float12` at [float13.gno](tests/float13.gno); fails at a6dc98e3b, passes once the narrowing also rejects values below `-MaxFloat32`. Fix: reject on magnitude, not just the upper bound, so a negative constant past `-MaxFloat32` is a compile error too.
  </details>

## Nits
None.

## Missing Tests
- **[negative overflow has no golden]** `gnovm/tests/files/float12.gno:1` — the corpus pins positive const overflow (`float11`, `float12`) but no case covers the negative side, so the `-Inf` leak ships untested.
  <details><summary>details</summary>

  Adding the negative counterpart both closes the gap above and guards against regression. The ready filetest asserts the post-fix reject (VM `// Error:` plus `// TypeCheckError:`), red at a6dc98e3b and green once the narrowing rejects values below `-MaxFloat32`: [float13.gno](tests/float13.gno).
  </details>

## Suggestions
None.

## Open questions
None.
