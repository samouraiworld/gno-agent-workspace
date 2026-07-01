# PR #5867: feat(gnovm): replace cockroachdb/apd -> math/big.Rat

URL: https://github.com/gnolang/gno/pull/5867
Author: Villaquiranm | Base: master | Files: 22 | +287 -282
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 3c7de91d0 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5867 3c7de91d0`

**TL;DR:** Gno evaluates untyped float constants like `1.0/3.0` with a decimal library that loses exactness, so `(1.0/3.0)*3.0 == 1.0` was `false` where Go says `true`. This PR swaps that library for exact rational arithmetic (`math/big.Rat`), matching Go. The fix is correct for the round-trip bug but removes the old size cap, so a few lines of constant arithmetic can now allocate gigabytes at compile time.

**Verdict: REQUEST CHANGES** — the correctness fix is right, but `ratGuard` bounds only the rational's denominator, leaving numerator/integer magnitude unbounded and unmetered during preprocess; a ~20-line package OOMs the node at deploy.

## Summary
`apd.Decimal` stores `1/3` as a truncated decimal `0.333…`, so multiplying back by 3 gives `0.999…`, not `1` ([#5862](https://github.com/gnolang/gno/issues/5862)). `big.Rat` stores it as the exact fraction `{1,3}`, and `{1,3}×3 = 1`, matching Go's `go/constant`. The PR replaces the type throughout `gnovm/pkg/gnolang`, deletes ~107 lines of manual hex-float parsing in favor of `big.Rat.SetString`, and adds `ratGuard` to cap the denominator at 4096 bits.

The regression: `apd` Mul ran at precision 1024 and rejected out-of-range exponents, so any constant stayed bounded (~425 bytes) no matter how it was built. `big.Rat` is exact, so repeated multiplication of integer-valued constants grows the numerator without limit. `ratGuard` checks `r.Denom().BitLen()` only, so integer-valued growth slips past it. Constant folding runs on a machine with no gas meter, so the work is never charged.

```
ratGuard(r):  Denom().BitLen() > 4096  → panic     ← caught
              Num().BitLen()   unbounded → allocate ← missed
```

## Examples
| input | Go compiler | gno master (apd) | this PR (big.Rat) |
|---|---|---|---|
| `(1.0/3.0)*3.0 == 1.0` | `true` | `false` | `true` (fixed) |
| 17 lines: `a0=1e10000`, `aN=a(N-1)²` ×14 | reject, 0.02s / 18MB | reject, 0.02s / 32MB | **accept, 72s / 481MB** |
| same shape, 19 lines | reject instantly | reject instantly | **does not finish in 120s** |

## Glossary
- preprocess: the static pass that resolves names, types, and folds constants before execution.
- app hash: the per-block Merkle commitment to application state agreed in consensus.

## Fix
The denominator cap in [`ratGuard`](https://github.com/gnolang/gno/blob/3c7de91d0/gnovm/pkg/gnolang/op_binary.go#L803) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L803) needs a companion bound on the numerator (or on total component bits), applied on the same paths it already guards: literal parsing in [op_eval.go:90-111](https://github.com/gnolang/gno/blob/3c7de91d0/gnovm/pkg/gnolang/op_eval.go#L90-L111) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_eval.go#L90) and the four arithmetic ops in `op_binary.go`. A hard cap restores the prior gno behavior (apd rejected oversized exponents); matching Go exactly would instead require a bounded-precision fallback, which gno's all-`big.Rat` representation does not have. Either way the magnitude must be bounded before the multiply allocates, since the folding machine carries no gas meter.

## Critical (must fix)
- **[20 lines of constants can OOM the node at deploy]** `gnovm/pkg/gnolang/op_binary.go:803` — `ratGuard` bounds only the denominator, so integer-valued constant arithmetic grows the numerator without limit, unmetered, at preprocess.
  <details><summary>details</summary>

  `ratGuard` panics when `r.Denom().BitLen() > 4096` but never inspects `r.Num()`. A `big.Rat` whose value is an integer keeps `Denom() == 1`, so the guard is a no-op for it. Repeated multiplication of an integer-valued constant doubles the numerator's bit-length per step: `const a0 = 1e10000; const aN = a(N-1) * a(N-1)` reaches `a_k = 10^(10000·2^k)`, whose numerator is ≈ `10000·2^k·3.32` bits. The same vector applies to a single literal: `0x1p+10000000` parses to a 1.2 MB numerator, and `1e1000000` to 415 KB, where `apd` stored both compactly as `(coeff, exp)`.

  Constant folding happens in [`evalConst`](https://github.com/gnolang/gno/blob/3c7de91d0/gnovm/pkg/gnolang/preprocess.go#L4394) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/preprocess.go#L4394), which builds a machine with no `GasMeter`; [`incrCPU`](https://github.com/gnolang/gno/blob/3c7de91d0/gnovm/pkg/gnolang/machine.go#L1392) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/machine.go#L1392) is a no-op without one. So the allocation and CPU are never charged, and `AddPkg` of a malicious package allocates unbounded memory before any gas limit applies. Measured on 3c7de91d0: 17 source lines = 481 MB / 72 s; 19 lines does not finish in 120 s. Go rejects the same input in 0.02 s / 18 MB (it caps rational components at `maxExp` and falls back to bounded `big.Float`), and gno master rejects it in 0.02 s / 32 MB (`apd` "exponent out of range"). The denominator side is correctly bounded, confirming the guard's intent: `1.0/3.0` squared 12 times panics on the 4096-bit denominator. Fix: bound the numerator (or total component bits) the same way the denominator is bounded, before the operation allocates.

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
None.

## Nits
- `gnovm/tests/files/const63.gno:5` — the comment says "run: go run _verify/const63_go/main.go", but the PR adds no `_verify/const63_go/main.go`; the Go cross-check file is missing, so the pointer is dead. Drop the reference or add the file.
- `gnovm/pkg/gnolang/values_string.go:74` — `BigdecValue.String()` now renders via `FloatString(10)`, rounding to 10 fractional digits, where `apd.String()` was exact. Display-only (conversion errors use the exact `RatString()`), but a directly printed untyped bigdec loses precision past 10 places.

## Missing Tests
- **[size bound is untested]** `gnovm/pkg/gnolang/op_binary.go:803` — `ratGuard` has no filetest. A `// Error:` filetest asserting that an oversized constant is rejected would have surfaced the numerator gap and locks the bound once added. The denominator path and the numerator path should each have one.

## Suggestions
- `gnovm/pkg/gnolang/values_conversions.go:1390` — the conversion error now prints the rational form: `cannot convert untyped bigdec to integer -- 6/5 not an exact integer` for a value the user wrote as `1.2`. `6/5` is harder to connect back to the source than `1.2`. Rendering the decimal form (the trimmed `FloatString` already used by `String()`) would read closer to what was written.
  <details><summary>details</summary>

  Same at line 1410 (the integer-kind branch). The filetests `make6.gno`, `bigdec2.gno`, `bigdec5.gno`, `shift_f4b.gno` were updated to expect `6/5`; they would flip back to `1.2` with a decimal renderer. Behavioral, not a correctness issue.
  </details>

## Open questions
- The amino encoding of `BigdecValue` changes from a decimal string to a ratio string (`values.go` `MarshalAmino`), which shifts the app hash for any persisted realm state that contains an untyped float constant — the `apphash_crossrealm38` hash bump in this PR is exactly that shift, so bigdec values do reach committed state. Not posted: the maintainers documented the hash bump as expected, and whether live state needs a migration vs a testnet reset is their call, not a code defect.
