# PR [#5864](https://github.com/gnolang/gno/pull/5864): fix(gnovm): fold -0 to +0 for float call args

URL: https://github.com/gnolang/gno/pull/5864
Author: davd-gzl | Base: master | Files: 2 | +35 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 662cbc5ba (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5864 662cbc5ba`

**TL;DR:** When a realm function is called on-chain with a float argument like `-0.0`, the value used to arrive with its sign bit set (negative zero), which no Gno source literal can ever produce. This clears the sign bit so a `-0.0` argument reaches the realm as plain `0`, matching how Go compiles a `-0.0` literal.

**Verdict: NEEDS DISCUSSION** — the negative-zero fold is correct and verified end-to-end, but this alternative to [#5221](https://github.com/gnolang/gno/pull/5221) drops that PR's rejection of `NaN`/`Inf` args, which the same argument path still admits.

## Summary
`convertFloat` turns a MsgCall string argument into a float64 via `apd.NewFromString` then `strconv.ParseFloat`, both of which keep the sign bit, so `"-0.0"` and `"-0"` arrived as negative zero. A Go source literal `-0.0` is a constant that folds to `+0` at compile time, so the on-chain argument path diverged from what the same literal would produce in code. The fix clears the sign bit on any zero result, so `-0` folds to `+0` on both the float64 and float32 paths, including the float32-underflow case where a nonzero literal like `-1e-50` rounds to `-0`. The residual concern is scope: [#5221](https://github.com/gnolang/gno/pull/5221), which this replaces, also rejected `NaN` and `Inf`; this PR does not, and the argument path still accepts them.

## Examples
| MsgCall arg | float64 result before | after this PR |
|-------------|----------------------|---------------|
| `-0.0` | `-0` (sign bit set) | `+0` |
| `-0` | `-0` (sign bit set) | `+0` |
| `-1e-50` (as float32) | `-0` (underflow, sign bit set) | `+0` |
| `NaN` | NaN | NaN (still accepted) |
| `Inf` / `-Inf` | ±Inf | ±Inf (still accepted) |

## Glossary
- malleability: two distinct byte encodings of one logical value, letting a tx be re-signed or replayed in a variant form. Here `-0` and `+0` are the same number with different bits.

## Fix
`convertFloat` gains a zero-fold at [convert.go:216-221](https://github.com/gnolang/gno/blob/662cbc5ba/gno.land/pkg/sdk/vm/convert.go#L216-L221) · [↗](../../../../../.worktrees/gno-review-5864/gno.land/pkg/sdk/vm/convert.go#L216-L221): after parsing, `if f64 == 0 { f64 = 0 }` reassigns the `+0` constant to overwrite the sign bit. The single call site is [keeper.go:894](https://github.com/gnolang/gno/blob/662cbc5ba/gno.land/pkg/sdk/vm/keeper.go#L894) · [↗](../../../../../.worktrees/gno-review-5864/gno.land/pkg/sdk/vm/keeper.go#L894), shared by both the MsgCall execution path and the `vm/qeval` query path, so both are covered. Two changes are proposed below and verified in the worktree: rewrite the zero case as `f64 = math.Copysign(0, 1)` so the sign-clear is on the code, not the comment (Nit), and reject `NaN`/`Inf` on the same path as [#5221](https://github.com/gnolang/gno/pull/5221) did (Warning).

## Critical (must fix)
None.

## Warnings (should fix)
- **[NaN and Inf args slip through the same path]** `gno.land/pkg/sdk/vm/convert.go:204-224` — This replaces [#5221](https://github.com/gnolang/gno/pull/5221), which rejected `NaN`/`Inf`; the arg path still accepts them, so a realm receives a float value no Gno source constant can express.
  <details><summary>details</summary>

  `apd.NewFromString` plus `strconv.ParseFloat` accept `"NaN"`, `"Inf"`, `"-Inf"`, and `"Infinity"`, and the fold only touches zero, so these reach realm code as real NaN/Inf floats. This is the same class the `-0` fix addresses: a Go source literal cannot produce `NaN` or `Inf` (`1.0/0.0` is a compile error, `1e400` overflows the constant), just as it cannot produce `-0`. The parse itself is deterministic (one NaN bit pattern), so this is not a consensus break, but it lets a MsgCall argument carry a value unreachable from source, which [#5221](https://github.com/gnolang/gno/pull/5221) closed for the stated determinism/malleability reason. Verified live: `maketx call ... -args 'NaN'` returns `("NaN" string)` from a realm that classifies its argument. Fix: reject `NaN` and `Inf` in `convertFloat`, or state in the PR why admitting them is acceptable while folding `-0` is not.

  Proposed fix, verified in the worktree (rejection wording matches [#5221](https://github.com/gnolang/gno/pull/5221)):

  ```go
  // in convertFloat, after ParseFloat, before the -0 fold
  if math.IsNaN(f64) {
      panic(fmt.Sprintf("float%d does not accept NaN", precision))
  }
  if math.IsInf(f64, 0) {
      panic(fmt.Sprintf("float%d does not accept Inf", precision))
  }
  ```

  Verified: with the guard applied, `maketx call ... -args 'NaN'|'Inf'|'-Inf'` is rejected with `float64 does not accept NaN`/`Inf` on the real MsgCall path, while `-0.0`/`-0`/`-1e-50` still fold to `+0`. Because admitting or rejecting `NaN`/`Inf` is a visible behavior change, this stays the reviewer's proposal; the verdict holds at NEEDS DISCUSSION until the author decides the scope.
  </details>

## Nits
- `gno.land/pkg/sdk/vm/convert.go:219-221` — `if f64 == 0 { f64 = 0 }` reads as a no-op; the sign-bit-clearing effect lives entirely in the comment. Write the zero case so the code shows it clears the sign. Proposed and verified in the worktree: `f64 = math.Copysign(0, 1)`. Verified: `math.Copysign(0, 1)` yields bits `0x0000000000000000`, the exact value Go's `-0.0` literal folds to, and clears the sign bit on a parsed `-0` (`0x8000...` to `0x0000...`); the intent is now on the code, not the comment.

## Missing Tests
- **[NaN/Inf admission is untested either way]** `gno.land/pkg/sdk/vm/convert_test.go` — No test pins whether `NaN`/`Inf` args are accepted or rejected, so the scope decision behind this PR is invisible to the suite.
  <details><summary>details</summary>

  The new `TestConvertFloatNegativeZeroFolded` covers the `-0` fold well, but nothing asserts the `NaN`/`Inf` behavior, so whichever way the Warning is resolved, the suite is silent on it. A txtar through the real MsgCall path documents both the fold and the NaN/Inf rejection: [`maketx_call_float_naninf.txtar`](../../../../../.worktrees/gno-review-5864/reviews/pr/5xxx/5864-fold-negzero-float-args/1-662cbc5ba/tests/maketx_call_float_naninf.txtar). If the decision is to keep admitting `NaN`/`Inf`, invert those four assertions to lock the accept behavior instead. See the finding's [test cases](comment_claude-opus-4-8.md).

  Verified with the proposed rejection applied: a table-driven Go test asserting the six NaN/Inf reject cases and the txtar both pass. Table added in the worktree as `TestConvertFloatRejectsNaNInf`:

  ```go
  {"float64 NaN", "NaN", gnolang.Float64Type, "float64 does not accept NaN"},
  {"float32 NaN", "NaN", gnolang.Float32Type, "float32 does not accept NaN"},
  {"float64 Inf", "Inf", gnolang.Float64Type, "float64 does not accept Inf"},
  {"float32 Inf", "Inf", gnolang.Float32Type, "float32 does not accept Inf"},
  {"float64 -Inf", "-Inf", gnolang.Float64Type, "float64 does not accept Inf"},
  {"float32 -Inf", "-Inf", gnolang.Float32Type, "float32 does not accept Inf"},
  ```
  </details>

## Suggestions
None.

## Open questions
None.
