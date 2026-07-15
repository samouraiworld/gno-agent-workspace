# PR [#5867](https://github.com/gnolang/gno/pull/5867): feat(gnovm): replace cockroachdb/apd -> math/big.Rat

URL: https://github.com/gnolang/gno/pull/5867
Author: Villaquiranm | Base: master | Files: 21 | +710 -311
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: 79ebc4058 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5867 79ebc4058`

Round 5 (deep). Head advanced adbb5ac60 → 79ebc4058: one authored commit (`fix agent review`) touching only `op_binary.go` and `values_conversions.go`. Both round-4 Warnings are resolved and verified. The float32 conversion now rounds once from the exact value (`big.Rat.Float32` / `big.Float.Float32`), so subnormal float32 constants match Go; a 800-case gno-vs-Go scan has zero divergences. `parseBigdecLiteral` now probes `big.ParseFloat` and an exponent guard before ever building the exact `big.Rat`, so 500 `1e999999` consts fold in 0.03 s (was 7.9 s), and the representation of every in-band literal is unchanged, so no app hash shifts. Remaining items are one stale doc-comment reference carried from round 4, one stale test comment introduced by this commit, and regression tests for the just-fixed float32 path.

**TL;DR:** Gno evaluated untyped float constants like `1.0/3.0` with a decimal library that loses exactness, so `(1.0/3.0)*3.0 == 1.0` was `false` where Go says `true`. This PR swaps that library for exact rational arithmetic and falls back to a bounded 512-bit float for extreme magnitudes, matching Go's own constant package. This round's commit fixes the two remaining correctness/cost gaps; only comment nits and missing regression tests are left.

**Verdict: APPROVE** — both round-4 Warnings are fixed and verified against Go; the leftovers are a stale `ratGuard` doc reference, a stale "24-bit precision" test comment, and regression tests for the subnormal float32 path.

## Summary
Round 4 rejected two things and both are fixed. The float32 conversion no longer rounds through a 24-bit intermediate; [`bigdecToFloat32`](https://github.com/gnolang/gno/blob/79ebc4058/gnovm/pkg/gnolang/values_conversions.go#L1500-L1509) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1500) rounds once from the exact `big.Rat` (or the 512-bit `big.Float`), the way the float64 path already did, so subnormal float32 constants stop double-rounding and match Go. [`parseBigdecLiteral`](https://github.com/gnolang/gno/blob/79ebc4058/gnovm/pkg/gnolang/op_binary.go#L858-L875) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L858) now parses the literal with `big.ParseFloat` first and returns the float form when the binary exponent is past ±4096, so an extreme-exponent literal never materializes the multi-megabit numerator the overflow check would discard.

The exponent guard is a safe approximation of the old `ratOverflows` check: whenever `|MantExp| > 4096` the exact rational would also have overflowed 4096 bits and promoted, so no in-band literal changes representation. Verified across the boundary (`1e1233` stays rat, `1e1234` promotes) and confirmed by the amino form of a persisted const being identical to round 4.

## Examples
Untyped float constants, as a user would write them, at this head vs Go and the round-4 commit (float32 rows as `math.Float32bits`):

| Written | Go | round 4 (adbb5ac60) | this head |
|---|---|---|---|
| `const s float32 = 0x1.4p-148 + 0x1p-200` | `3` | `2` | `3` |
| `const s3 float32 = 0x1.2p-147 + 0x1p-200` | `5` | `4` | `5` |
| `const x float32 = 1.0000001` | `1065353217` | `1065353217` | `1065353217` |
| `const c float32 = 1.0 + 0x1p-24 + 0x1p-80` | `1065353217` | `1065353217` | `1065353217` |
| 500 × `const = 1e999999` fold time | ~0.02 s | 7.9 s | 0.03 s |
| `(1.0/3.0)*3.0 == 1.0` | `true` | `true` | `true` |

The two subnormal rows round 4 got wrong now match Go; the normal-range rows and the headline exactness are preserved.

## Glossary
- bigdec: the VM's arbitrary-precision value backing an untyped float constant before it is typed; a `big.Rat` or a 512-bit `big.Float`.
- preprocess: the static pass that constant-folds expressions before execution; not gas-metered per operation.
- promotion: the automatic switch from exact `big.Rat` to bounded `big.Float` once a value's numerator or denominator exceeds 4096 bits, mirroring `go/constant`.
- subnormal: an IEEE-754 value below the smallest normal magnitude (float32: `< 2^-126 ≈ 1.2e-38`), represented with reduced significand precision.
- filetest: a `.gno` file executed by the VM and asserted against golden directives (`// Output:`).

## Fix
The float32 case of [`ConvertUntypedBigdecTo`](https://github.com/gnolang/gno/blob/79ebc4058/gnovm/pkg/gnolang/values_conversions.go#L1454-L1465) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1454) dropped the `SetPrec(24)` intermediate and calls the new `bigdecToFloat32`, which mirrors `bigdecToFloat64`: exact rat or 512-bit float straight to the IEEE-754 float32 grid, one rounding. `parseBigdecLiteral` reordered to parse `big.ParseFloat` first, early-return the float form when `MantExp` is past ±`ratOverflowBits`, and only then build and range-check the exact `big.Rat`. In-band literals keep the exact rational; nothing changes for the headline `1.0/3.0` case.

## Critical (must fix)
None.

## Warnings (should fix)
None. Both round-4 Warnings are resolved (see Verified).

## Nits
- **[comment names a symbol that never existed]** [`values.go:128`](https://github.com/gnolang/gno/blob/79ebc4058/gnovm/pkg/gnolang/values.go#L128) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values.go#L128) — the `BigdecValue` doc comment says the rational form holds values "within ratGuard's bit limits", but the guard is named `ratOverflows` / `ratOverflowBits`. Carried from round 4; still the only stale reference in the code.
- **[test comment describes a mechanism the fix removed]** [`bigdec_float32_no_double_round.gno:1-2`](https://github.com/gnolang/gno/blob/79ebc4058/gnovm/tests/files/types/bigdec_float32_no_double_round.gno#L1-L2) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/types/bigdec_float32_no_double_round.gno#L1) — the header says the conversion "must round once at 24-bit precision", but this commit removed the 24-bit `big.Float` intermediate; the value now rounds directly from the exact `big.Rat` to float32. The test still asserts the right bits; only the comment is stale.
- **[constant comparison one bit off Go at the promotion boundary]** [`op_binary.go:801-804`](https://github.com/gnolang/gno/blob/79ebc4058/gnovm/pkg/gnolang/op_binary.go#L801-L804) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L801) — a comparison of two integer-valued float constants whose magnitude sits at exactly 4095 bits (~1233 decimal digits) disagrees with Go: `ratOverflows` keeps the exact `big.Rat` where `go/constant` has already rounded both operands to its bounded 512-bit form. `2^4095-1.0 == 2^4095-2.0` is `false` in gno, `true` in Go. Pre-existing (adbb5ac60 diverges identically) and unobservable through any float conversion, since such values are ±Inf in both float32 and float64. This commit narrows the divergence rather than widening it: `2^4096-1.0 == 2^4096-2.0` was `false` at adbb5ac60 and is now `true`, matching Go, because the new [`MantExp` guard](https://github.com/gnolang/gno/blob/79ebc4058/gnovm/pkg/gnolang/op_binary.go#L867) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L867) promotes the operand that `big.ParseFloat` rounds up to `2^4096`. No change needed.
- **[printed constants round without saying so]** [`values_string.go:75`](https://github.com/gnolang/gno/blob/79ebc4058/gnovm/pkg/gnolang/values_string.go#L75) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_string.go#L75) — rat-form `String()` renders via `FloatString(10)`, so a directly printed untyped bigdec whose exact form needs more than 10 fractional digits (`1.0/2048.0` = `0.00048828125`) is rounded. Display-only. Carries from round 1; no change needed.

## Missing Tests
- **[the just-fixed subnormal path has no regression test]** `gnovm/pkg/gnolang/values_conversions.go:1500-1509` — `bigdec_float32_no_double_round.gno` covers the normal-range double round, but the subnormal case this commit fixed has no filetest, so a future refactor of `bigdecToFloat32` could silently reintroduce the round-4 bug.
  <details><summary>details</summary>

  The subnormal float32 double round was the round-4 Warning. The fix is verified against Go, but nothing in the suite locks it: reverting `bigdecToFloat32` to the `SetPrec(24)` path leaves every committed filetest green. The ready-to-add [`bigdec_float32_subnormal.gno`](tests/bigdec_float32_subnormal.gno) asserts `math.Float32bits(0x1.4p-148 + 0x1p-200) == 3`; it passes at 79ebc4058 and fails (`bits: 2`) at adbb5ac60, so it is a real regression guard. Fix: add it under `gnovm/tests/files/types/`.
  </details>
- **[the new float form is never persisted in a test]** `gnovm/pkg/gnolang/values.go:198-227` — the amino round-trip and app-hash tests cover the rat form but not the `f:`-prefixed float form; a package-level `const X = 1e5000` persists a float-form bigdec whose encoding is untested. Ready-to-add [`bigdec_float_amino_test.go`](../4-adbb5ac60/tests/bigdec_float_amino_test.go) from round 4 still applies. Carries.
- **[dual-form arithmetic and comparison have no filetest]** `gnovm/pkg/gnolang/op_binary.go:821-856` — float-form comparison and mixed rat/float arithmetic are unexercised by a committed filetest. Ready-to-add [`bigdec_dual_form_arith.gno`](../4-adbb5ac60/tests/bigdec_dual_form_arith.gno) from round 4 still applies. Carries.

## Verified
- Both round-4 Warnings are fixed. Built `gnovm/cmd/gno` at 79ebc4058 and ran against Go: the subnormal float32 constants `0x1.4p-148 + 0x1p-200` and `0x1.2p-147 + 0x1p-200` give `math.Float32bits` `3` and `5`, matching Go (were `2` / `4` at adbb5ac60); the normal double-round case `1.0 + 0x1p-24 + 0x1p-80` stays `1065353217`; notJoon's [`1.0000001`](https://github.com/gnolang/gno/pull/5867#discussion_r3567858061) is `1065353217` (was `1065353216` before the wider fix). A 400-case random float32 scan and a 400-case float64 scan (both including the subnormal band) are bit-identical to Go, zero divergences.
- The literal-parse cost is bounded. 500 `1e999999` consts (~6.5 KB source) fold in 0.03 s at this head (7.9 s at adbb5ac60). The residual high-precision path (a literal with thousands of fractional digits at moderate magnitude still builds the exact rat) is linear in source length, the same class as master's `apd`, not an amplification vector: 400 copies of an 8000-digit literal (3.2 MB source) fold in 0.32 s.
- Representation is unchanged versus round 4, so no app hash shift beyond the already-documented apd→rat change. The `MantExp` early-return only fires when the exact rational would also overflow 4096 bits and promote, proven at the boundary: `1e1233` stays rat, `1e1234` promotes, `1e5000`/`1e-5000` promote, `1.0`/`0.5`/`0.333…` and in-band hex floats (`0x1p4000`) stay exact rat. `big.Rat.SetString` accepts hex-float notation, so no in-band hex literal loses exactness through the `!ok` fallback. `TestAppHashCrossrealm38` passes with the documented hash.
- Headline behavior holds vs Go: `(1.0/3.0)*3.0 == 1.0` is `true`, `1e10000 / 1e10000` is `1`, `var f32 float32 = -1e-50` has `math.Signbit` false (+0).
- Promotion-boundary map `(2^p-1).0 == (2^p-2).0`, adbb5ac60 / 79ebc4058 / Go: p=4094 `false`/`false`/`false`; p=4095 `false`/`false`/`true`; p=4096 `false`/`true`/`true`; p=4097 `true`/`true`/`true`. The commit fixes p=4096 (the rounded operand now promotes and matches Go) and leaves the pre-existing p=4095 divergence, which no float conversion can observe (Nit above). Both lenses reached this independently.
- Determinism: `big.Rat.Float32` and `big.Float.Float32` construct the IEEE-754 bits by integer arithmetic (`math.Ldexp` / direct bit manipulation), not hardware floating point; the 800-case scan is reproducible and Go-identical, so the output path stays deterministic across nodes.
- Green at 79ebc4058: `TestParity`, `TestConvert*`, `TestBounded*`, `TestValuesString*`, `TestAppHashCrossrealm38`, and the `const63` / `float9` / `float10` / `bigdec_extreme_magnitude` / `bigdec_float32_no_double_round` filetests, plus the ready-to-add `bigdec_float32_subnormal` regression. Local `TestFiles` failures unrelated to this PR reproduce on master f99caf537 (go/types message drift under the local Go toolchain); CI is green.

## Open questions
- Old realm state persisted under `apd` stored a decimal string; the new `UnmarshalAmino` parses it via `big.Rat.SetString` and yields that truncated decimal as an exact rational, not the original `1/3`. The app hash already shifts (documented in the crossrealm38 test), so any such state is incompatible regardless. Not posted: whether live state needs migration or a testnet reset is the maintainers' call. Carried from round 4.
- [@thehowl](https://github.com/gnolang/gno/pull/5867#issuecomment-4830532288) asked for a `big.Rat` fork wired to `softfloat`, and the author agreed but has not implemented it. Neither `big.Rat` nor `big.Float` uses hardware floating point, so the current code is already deterministic across nodes. Not posted: the determinism argument for the fork should be settled in-thread. Carried from round 4.
