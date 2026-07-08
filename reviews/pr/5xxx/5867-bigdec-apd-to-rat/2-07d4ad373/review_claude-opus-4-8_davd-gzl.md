# PR [#5867](https://github.com/gnolang/gno/pull/5867): feat(gnovm): replace cockroachdb/apd -> math/big.Rat

URL: https://github.com/gnolang/gno/pull/5867
Author: Villaquiranm | Base: master | Files: 22 | +287 -282
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 07d4ad373 (stale — +1 commit since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5867 07d4ad373`

Round 2. Head advanced 3c7de91d0 → 07d4ad373 (+5 PR commits + two master merges; GitHub head `ab2e5b3d3` is a base-only master merge on top of 07d4ad373, no PR files changed). Fixed since round 1: the `const63.gno` comment no longer names a missing `_verify` file ([0098308e6](https://github.com/gnolang/gno/commit/0098308e6)); the "not an exact integer" error now renders the decimal form `1.2` instead of `6/5` via a new `bigdecErrString` helper ([e0336945d](https://github.com/gnolang/gno/commit/e0336945d)). New this round: the float32 conversion branch was rewritten to narrow through softfloat ([7014a9431](https://github.com/gnolang/gno/commit/7014a9431)) and still double-rounds, diverging from Go (new Warning below). The round-1 Critical (`ratGuard` numerator gap) is unaddressed and re-verified at 07d4ad373; the `ratGuard` code is byte-identical to round 1.

**TL;DR:** Gno evaluates untyped float constants like `1.0/3.0` with a decimal library that loses exactness, so `(1.0/3.0)*3.0 == 1.0` was `false` where Go says `true`. This PR swaps that library for exact rational arithmetic (`math/big.Rat`), matching Go. The round-trip fix is correct, but the size guard bounds only the denominator (a ~20-line package OOMs the node at deploy) and the float32 conversion double-rounds, so a boundary-value constant differs from Go by one unit in the last place.

**Verdict: REQUEST CHANGES** — the round-1 Critical stands (`ratGuard` bounds only the denominator, leaving integer-valued magnitude unbounded and unmetered at preprocess), and the reworked float32 branch narrows through float64, so `float32` constants on a rounding boundary diverge from Go.

## Summary
`apd.Decimal` stores `1/3` as a truncated decimal `0.333…`, so multiplying back by 3 gives `0.999…`, not `1` ([#5862](https://github.com/gnolang/gno/issues/5862)). `big.Rat` stores it as the exact fraction `{1,3}`, and `{1,3}×3 = 1`, matching Go's `go/constant`. The PR replaces the type throughout `gnovm/pkg/gnolang`, deletes ~107 lines of manual hex-float parsing in favor of `big.Rat.SetString`, and adds `ratGuard` to cap the denominator at 4096 bits.

Two gaps remain. `ratGuard` checks `r.Denom().BitLen()` only, so integer-valued growth (denominator stays 1) slips past it; folding runs on a gasless machine, so the allocation is never charged. And the float32 conversion computes `f64 := r.Float64()` then narrows to float32, double-rounding where Go rounds the constant straight to float32; the two disagree on values that land on a float32 midpoint after the first rounding.

```
ratGuard(r):  Denom().BitLen() > 4096  → panic     ← caught
              Num().BitLen()   unbounded → allocate ← missed

float32(bigdec):  rat --Float64()--> f64 --F64to32--> f32   (two roundings)
Go compiler:      rat -----------round-to-float32----> f32   (one rounding)
```

## Examples
| input | Go compiler | gno master (apd) | this PR (07d4ad373) |
|---|---|---|---|
| `(1.0/3.0)*3.0 == 1.0` | `true` | `false` | `true` (fixed) |
| `float32(0x1.000001000000001p0)` | `1.0000001` | `1.0000001` | `1` (double-round) |
| `int(1.2)` conversion error text | — | `6/5 not an exact integer` | `1.2 not an exact integer` (fixed) |
| 17 lines: `a0=1e10000`, `aN=a(N-1)²` ×14 | reject, 0.02s / 18MB | reject, 0.02s / 32MB | **accept, 72s / 481MB** |

## Glossary
- preprocess: the static pass that resolves names, types, and folds constants before execution.
- double rounding: rounding a value to an intermediate precision (float64) and then to the target (float32); can differ from rounding straight to the target.

## Fix
Two changes. The denominator cap in [`ratGuard`](https://github.com/gnolang/gno/blob/07d4ad373/gnovm/pkg/gnolang/op_binary.go#L806-L808) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L806) needs a companion bound on the numerator (or on total component bits), applied on the paths it already guards; the magnitude must be bounded before the multiply allocates, since the folding machine carries no gas meter. And the float32 branch in [`ConvertUntypedBigdecTo`](https://github.com/gnolang/gno/blob/07d4ad373/gnovm/pkg/gnolang/values_conversions.go#L1437-L1447) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1437) should round the rational directly to float32 (`r.Float32()`, which is pure big.Int arithmetic, so deterministic) rather than narrowing `r.Float64()`, to single-round and match Go.

## Critical (must fix)
- **[20 lines of constants can OOM the node at deploy]** `gnovm/pkg/gnolang/op_binary.go:806` — `ratGuard` bounds only the denominator, so integer-valued constant arithmetic grows the numerator without limit, unmetered, at preprocess.
  <details><summary>details</summary>

  `ratGuard` panics when `r.Denom().BitLen() > 4096` but never inspects `r.Num()`. A `big.Rat` whose value is an integer keeps `Denom() == 1`, so the guard is a no-op for it. Repeated multiplication of an integer-valued constant doubles the numerator's bit-length per step: `const a0 = 1e10000; const aN = a(N-1) * a(N-1)` reaches `a_k = 10^(10000·2^k)`, whose numerator is ≈ `10000·2^k·3.32` bits. The same vector applies to a single literal: `0x1p+10000000` parses to a 1.2 MB numerator, and `1e1000000` to 415 KB, where `apd` stored both compactly as `(coeff, exp)`.

  Constant folding happens in [`evalConst`](https://github.com/gnolang/gno/blob/07d4ad373/gnovm/pkg/gnolang/preprocess.go#L4394) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/preprocess.go#L4394), which builds a machine with no `GasMeter`; [`incrCPU`](https://github.com/gnolang/gno/blob/07d4ad373/gnovm/pkg/gnolang/machine.go#L1392) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/machine.go#L1392) is a no-op without one. So the allocation and CPU are never charged, and `AddPkg` of a malicious package allocates unbounded memory before any gas limit applies. Re-verified on 07d4ad373: the numerator-growth package folds and returns `ok`, where the same shape built from `1.0/3.0` panics on the 4096-bit denominator; the `ratGuard` function is unchanged since round 1 (measured then: 17 source lines = 481 MB / 72 s; 19 lines does not finish in 120 s; Go and gno master reject the same input in ~0.02 s). Fix: bound the numerator (or total component bits) the same way the denominator is bounded, before the operation allocates.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5867 -R gnolang/gno

  # A) denominator growth IS bounded by ratGuard:
  { echo 'package main'; echo 'const a0 = 1.0 / 3.0'
    for i in $(seq 1 12); do echo "const a$i = a$((i-1)) * a$((i-1))"; done
    echo 'func main() { println("ok") }'; } > /tmp/denomA.gno

  # B) numerator growth is NOT bounded — same shape, integer-valued base:
  { echo 'package main'; echo 'const a0 = 1e10000'
    for i in $(seq 1 12); do echo "const a$i = a$((i-1)) * a$((i-1))"; done
    echo 'func main() { println("ok") }'; } > /tmp/numB.gno

  go run ./gnovm/cmd/gno run /tmp/denomA.gno   # a12 denom = 3^4096 ≈ 6492 bits
  go run ./gnovm/cmd/gno run /tmp/numB.gno      # a12 = 1e40960000, numerator ≈ 136 Mbit
  rm /tmp/denomA.gno /tmp/numB.gno
  ```
  ```
  # A:
  panic: constant expression result too large: denominator exceeds 4096 bits

  # B:
  ok
  # numB accepted: a12 = 1e40960000 was folded and materialized (~17 MB) at
  # preprocess with no panic and no gas. Adding lines scales it 4x each
  # (k=14 ≈ 481 MB / 72 s); Go and gno-master both reject this input in ~0.02 s.
  ```
  </details>

## Warnings (should fix)
- **[float32 constant differs from Go by one unit in the last place]** `gnovm/pkg/gnolang/values_conversions.go:1437-1447` — the float32 conversion narrows `r.Float64()` down to float32, double-rounding; Go rounds the constant straight to float32, so boundary values disagree.
  <details><summary>details</summary>

  The Float32Kind branch computes `f64, _ := r.Float64()` then narrows `f64` to float32 via `softfloat.F64to32`. The softfloat step removes hardware dependence but not the double rounding: the value is rounded once to float64 and again to float32. Go's compiler (`go/constant`) rounds the exact constant directly to float32, a single rounding. The two disagree when the first rounding lands the value exactly on a float32 midpoint, which then ties to even. `0x1.000001000000001p0` is just above the midpoint `1+2^-24`; it rounds to that midpoint in float64, then to `1.0` in float32, so gno prints `1`, while Go prints `1.0000001` (bits `3f800001`). The sibling Float64Kind branch at [values_conversions.go:1449-1457](https://github.com/gnolang/gno/blob/07d4ad373/gnovm/pkg/gnolang/values_conversions.go#L1449-L1457) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1449) is correct because float64 is the final target there. Matching Go and staying deterministic requires rounding the rational straight to float32: `big.Rat.Float32()` is pure big.Int arithmetic (no hardware float), single-rounds, and returns `±Inf` on overflow so the existing `IsInf` panic still fires. Fix: round the rational directly to float32 instead of through float64. Red→green filetest: [`tests/bigdec_f32_doubleround.gno`](tests/bigdec_f32_doubleround.gno) asserts `1.0000001` and fails at 07d4ad373 (got `1`), passing once the conversion single-rounds.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5867 -R gnolang/gno

  cat > /tmp/dr.gno <<'EOF'
  package main
  func main() {
  	const c = 0x1.000001000000001p0
  	var f float32 = c
  	println(f)
  }
  EOF
  cat > /tmp/dr.go <<'EOF'
  package main
  func main() {
  	const c = 0x1.000001000000001p0
  	var f float32 = c
  	println(f)
  }
  EOF
  echo -n 'gno: '; go run ./gnovm/cmd/gno run /tmp/dr.gno
  echo -n 'go:  '; go run /tmp/dr.go
  rm /tmp/dr.gno /tmp/dr.go
  ```
  ```
  gno: 1
  go:  1.0000001
  ```
  </details>

## Nits
- `gnovm/pkg/gnolang/values_string.go:69` — `BigdecValue.String()` renders via `FloatString(10)`, rounding to 10 fractional digits, where `apd.String()` was exact. Display-only (conversion errors now use the terminating-decimal-aware `bigdecErrString`, and the exact `RatString()` fallback), but a directly printed untyped bigdec loses precision past 10 places.

## Missing Tests
- **[size bound is untested]** `gnovm/pkg/gnolang/op_binary.go:806` — `ratGuard` has no filetest. A `// Error:` filetest asserting that an oversized constant is rejected would have surfaced the numerator gap and locks the bound once added. The denominator path and the numerator path should each have one.

## Suggestions
None.

## Verified
- float32 divergence from Go, run at 07d4ad373: `float32(0x1.000001000000001p0)` evaluates to `1` in gno and `1.0000001` (bits `3f800001`) under both Go's `gc` compiler and `big.Rat.Float32()`; the gno path narrows `r.Float64()` (which yields the exact float32 midpoint) and ties to even. `tests/bigdec_f32_doubleround.gno` asserts the Go value and fails at this sha.
- `ratGuard` numerator gap, re-run at 07d4ad373: the integer-valued squaring package folds and prints `ok`; the `1.0/3.0` denominator variant panics on the 4096-bit bound. `git diff 3c7de91d0 07d4ad373 -- op_binary.go` shows only an unrelated master change (#5784 blank-field equality), so the guard is byte-identical to round 1.
- Round-1 findings resolved, checked at this sha: the `const63.gno` header no longer references `_verify/const63_go/main.go`; `int(1.2)`-style conversion errors print `1.2 not an exact integer` (the `make6`, `bigdec2`, `bigdec5`, `shift_f4b` filetests were flipped from `6/5` and pass); `bigdecErrString` returns the decimal form for terminating decimals and falls back to `a/b` for non-terminating (`1/3`, `22/7`).
- Tests run green at 07d4ad373: `TestConvertUntypedBigdecTo`, `TestConvertUntypedBigdecToFloat32`, `TestBigdecErrString`, and the `TestFiles` cases `const63`, `make6`, `bigdec2`, `bigdec5`, `shift_f4b`.

## Open questions
- The amino encoding of `BigdecValue` changes from a decimal string to a ratio string (`values.go` `MarshalAmino`), which shifts the app hash for any persisted realm state that contains an untyped float constant — the `apphash_crossrealm38` hash bump in this PR is exactly that shift, so bigdec values do reach committed state (the ADR states the opposite). Unchanged this round. Not posted: whether live state needs a migration vs a testnet reset is the maintainers' call, not a code defect.
