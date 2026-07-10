# PR [#5867](https://github.com/gnolang/gno/pull/5867): feat(gnovm): replace cockroachdb/apd -> math/big.Rat

URL: https://github.com/gnolang/gno/pull/5867
Author: Villaquiranm | Base: master | Files: 22 | +420 -280
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 7817f6e1d (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5867 7817f6e1d`

Round 3. Head advanced 2b5e5a8a5 → 7817f6e1d, a master merge with no new PR commits. It brought in the `float9`/`float10` filetests and hand-resolved a conflict in `ConvertUntypedBigdecTo`. Neither finding below is code the merge wrote: `ratGuard` has rejected `-1e-10000` since `c0adee275` and gained its numerator half at `2b5e5a8a5`, and `var f32 float32 = -1e-50` has stored `-0` at every branch head. The merge is where both became visible, one through master's new tests and one through master's new fix. The round-2 APPROVE is overturned. Round-2 findings all stay resolved; the round-1 display nit still carries.

**TL;DR:** Gno evaluated untyped float constants like `1.0/3.0` with a decimal library that loses exactness, so `(1.0/3.0)*3.0 == 1.0` was `false` where Go says `true`. This PR swaps that library for exact rational arithmetic (`math/big.Rat`), matching Go. The size cap added in round 2 to stop a compile-time memory blowup turns out to reject ordinary Go constants like `1e10000`, and the merge with master mislaid a sign fix, so `float32` constants can now come out as negative zero.

**Verdict: REQUEST CHANGES** — `ratGuard` rejects untyped float constants past roughly 1e±1233 that Go evaluates, which is what turns CI's `main / test` red; separately, the merge placed `posZero` before the float32 narrowing, so the float32 half of [#5741](https://github.com/gnolang/gno/pull/5741) never takes effect on this branch.

## Summary
Round 1 flagged that an unbounded `big.Rat` lets a short package allocate gigabytes at gasless preprocess. Round 2 answered with [`ratGuard`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/op_binary.go#L809-L816) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L809), which panics when either component of the rational exceeds 4096 bits. That bound is the right size but the wrong response. Go's `go/constant` uses the identical 4096-bit test ([`maxExp = 4 << 10`](https://github.com/golang/go/blob/go1.25.0/src/go/constant/value.go#L351), [`smallInt`/`smallFloat`](https://github.com/golang/go/blob/go1.25.0/src/go/constant/value.go#L353-L377)) to decide when a value is too big for an exact rational, then [falls back](https://github.com/golang/go/blob/go1.25.0/src/go/constant/value.go#L328-L345) to a `big.Float` at its parser's [512-bit precision](https://github.com/golang/go/blob/go1.25.0/src/go/constant/value.go#L69) rather than rejecting. So `1e10000 / 1e10000` is `1` in Go and on master, and a panic at preprocess here. The merge also imported [`posZero`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/values_conversions.go#L1412-L1420) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1412) from [#5741](https://github.com/gnolang/gno/pull/5741) and applied it one step too early on the float32 path, leaving a whole band of constants with a negative zero.

Both findings sit on ground [@ltzmaxwell](https://github.com/ltzmaxwell) has already worked. [#5741](https://github.com/gnolang/gno/pull/5741) is the merged signed-zero fix, and its float32 half does not take effect on this branch. The draft [#5740](https://github.com/gnolang/gno/pull/5740) is related; it rewrites the same literal-parse block in [`op_eval.go`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/op_eval.go#L88-L110) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_eval.go#L88) as this PR.

## Examples
Untyped float constants, as a user would write them:

| Written | Go | master | this head |
|---|---|---|---|
| `println(1e10000 / 1e10000)` | `1` | `1` | panic: numerator exceeds 4096 bits |
| `var m = -1e-10000` | `+0` | `+0` | panic: denominator exceeds 4096 bits |
| `var m = 1e-1233` | `+0` | `+0` | `+0` |
| `var m = 1e-1234` | `+0` | `+0` | panic: denominator exceeds 4096 bits |
| `var m = -1e-779137` | `+0` | panic: invalid decimal constant | panic: denominator exceeds 4096 bits |
| `var f float32 = -1e-50` | `+0` | `+0` | `-0` |
| `var f float32 = -1e-400` | `+0` | `+0` | `+0` |

The `-1e-779137` row is related to [#5740](https://github.com/gnolang/gno/pull/5740). Master rejects it through `apd`'s parser; this head rejects it, and everything past `1e±1233`, through `ratGuard`.

## Glossary
- bigdec: the VM's arbitrary-precision value backing an untyped float constant before it is typed; `ConvertUntypedBigdecTo` narrows it to `float32`/`float64` in constant space.
- preprocess: the static pass that resolves names, types, and blocks before execution.
- signed zero: IEEE-754 keeps distinct `+0` and `-0`; Go constants do not, so any constant that converts to zero yields `+0`, while runtime float ops preserve the sign of an underflowed zero.
- softfloat: the VM's deterministic software floating-point implementation; `F64to32` narrows float64 bits to float32, where `-0` is `1<<31`.
- filetest: a `.gno` file executed by the VM and asserted against golden directives (`// Output:`, `// Error:`).

## Fix
`ratGuard` currently treats "too big for an exact rational" and "illegal program" as the same thing, so [`op_eval.go:95`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/op_eval.go#L95) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_eval.go#L95) rejects a literal Go would keep at reduced precision. The bound itself has to stay: without it, repeated squaring at gasless preprocess reaches gigabytes, which is the round-1 blocker. What has to change is what happens at the bound, since Go's own constant package keeps the value in a fixed-precision float once the exact fraction stops being reasonable. On the conversion side the constraint is narrower: [`values_conversions.go:1459`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/values_conversions.go#L1459) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1459) has to clear the sign after `F64to32`, because that is where float32 underflow happens.

## Critical (must fix)
- **[valid Go programs rejected at compile time]** `gnovm/pkg/gnolang/op_binary.go:809-816` — untyped float constants past roughly 1e±1233 panic at preprocess, where Go evaluates them.
  <details><summary>details</summary>

  [`ratGuard`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/op_binary.go#L809-L816) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L809) panics when `Num().BitLen()` or `Denom().BitLen()` exceeds 4096, at the two literal-parse sites in [`op_eval.go`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/op_eval.go#L95) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_eval.go#L95) and the four arithmetic results in `op_binary.go`. Go's `go/constant` applies the same 4096-bit threshold, then switches the value to a 512-bit `big.Float` instead of rejecting it, so `1e10000` is an ordinary untyped constant there and `1e10000 / 1e10000` folds to `1`. Here the literal never survives long enough to be divided. The measured cutoff is `1e-1233` accepted, `1e-1234` rejected, matching 4096 / log2(10). Master's [`float9.gno`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/tests/files/float9.gno#L9) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/float9.gno#L9) and [`float10.gno`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/tests/files/float10.gno#L9) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/float10.gno#L9) both open with `var m = -1e-10000`, and both fail on this head; they are the only two failures in `main / test`. The rejection is not uniformly a regression: master's `apd` parser already refuses `-1e-779137`, the case the related [#5740](https://github.com/gnolang/gno/pull/5740) addresses. What this guard does is widen the refused set down to `1e±1234` and give it a second cause, so that work would land on a path that rejects earlier. Nor is it new code: the denominator check dates to the PR's first commit `c0adee275`, and the numerator check to `2b5e5a8a5`, where it answered round 1's memory blowup. The merge changed nothing here except to import the tests that catch it. The `512` is `go/constant`'s own precision, not a language guarantee: the [Go spec](https://go.dev/ref/spec#Constants) only floors an implementation at a 256-bit mantissa and a 16-bit signed binary exponent, which would not by itself require accepting `1e-10000`. The operative standard is what `gc` does and what master's own filetests already assert. Observed result in [repro](comment_claude-opus-4-8.md). Fix: values past the exact-rational threshold need a bounded fallback representation, not a panic.
  </details>

- **[an imported fix that misses one of its three paths]** `gnovm/pkg/gnolang/values_conversions.go:1458-1466` — `posZero` runs on the float64 before the narrowing, so a constant that underflows only at float32 width stores `-0`.
  <details><summary>details</summary>

  On master, `ConvertUntypedBigdecTo` normalizes the sign after narrowing, so `var f32 float32 = -1e-50` is `+0` there. That shape came from the related [#5741](https://github.com/gnolang/gno/pull/5741), which is what the author's [question on this line](https://github.com/gnolang/gno/pull/5867#discussion_r3556812259) is asking about. The branch stored `-0` here before the merge too, because `posZero` did not exist on it; the merge is where the fix arrived and where two of its three call sites took effect. `ConvertUntypedBigdecTo` normalizes the float64 and interface results correctly, so `var f float64 = -1e-400` and `var i interface{} = -1e-400` flipped from `-0` at `2b5e5a8a5` to `+0` here. Only the float32 case still diverges. The merge resolution applies `posZero` at [`values_conversions.go:1459`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/values_conversions.go#L1459) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1459), where it can only act if `r.Float64()` itself underflowed, meaning `|x| < 5e-324`. For a constant between that and float32's smallest subnormal near `1.4e-45` the float64 is a normal number, [`softfloat.F64to32`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/values_conversions.go#L1461) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1461) underflows it to `-0`, and [line 1466](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/values_conversions.go#L1466) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1466) stores those bits unchanged. `var f32 float32 = -1e-50` reports `math.Signbit` true on this head and false on both Go and master. The `var float32: +0` line of [`float9.gno`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/tests/files/float9.gno#L23) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/float9.gno#L23) covers this, but never executes: line 9 of the same file trips the guard first. The explicit-conversion spelling `float32(-1e-50)` is rescued by the constant-folding normalization at [`preprocess.go:4437-4444`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/preprocess.go#L4437-L4444) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/preprocess.go#L4437), which is why only the variable-declaration spelling shows the defect. Observed result in [repro](comment_claude-opus-4-8.md).
  </details>

## Warnings (should fix)
None.

## Nits
- **[printed constants round without saying so]** [`values_string.go:68-69`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/values_string.go#L68-L69) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_string.go#L68) — `BigdecValue.String()` renders via `FloatString(10)`, so a directly printed untyped bigdec whose exact form needs more than 10 fractional digits (`1.0/2048.0` = `0.00048828125`) is rounded. Display-only: the exact conversion and error paths use `RatString`/`bigdecErrString`, and an untyped float constant reaches this method only through debug and error printing. Carries from round 1; no change needed.

## Missing Tests
None. Master's `float9.gno` covers both findings once the guard stops rejecting its first line.

## Verified
- Built `gnovm/cmd/gno` at each branch head and ran the three cases, to place each defect at the commit that caused it rather than at the merge:

  | | `3c7de91d0` round 1 | `2b5e5a8a5` round 2 | `7817f6e1d` merge | `d2869dceb` master |
  |---|---|---|---|---|
  | `var m = -1e-10000` | panic | panic | panic | `+0` |
  | `println(1e10000 / 1e10000)` | `1` | panic | panic | `1` |
  | `var f float32 = -1e-50` | `-0` | `-0` | `-0` | `+0` |
  | `var f float64 = -1e-400` | `-0` | `-0` | `+0` | `+0` |
  | `var i interface{} = -1e-400` | `-0` | `-0` | `+0` | `+0` |

  The denominator rejection is as old as the PR; the numerator rejection arrived with round 2's guard. The float32 sign is unchanged across the branch, and the merge is where `posZero` fixed the float64 and interface rows and left the float32 one.
- The two failures are independent, which the suite cannot show because `float9.gno` aborts at its first line. Lifting float9's float32 assertions into a standalone file, so no `-1e-10000` literal reaches the guard, `var f32 float32 = -1e-50` has `math.Signbit` true at 7817f6e1d and false at master d2869dceb. Fixing `ratGuard` alone leaves `float9.gno` red. Two of float9's declarations trip the guard, [line 9](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/tests/files/float9.gno#L9) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/float9.gno#L9) and [line 26](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/tests/files/float9.gno#L26) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/float9.gno#L26), so both have to clear before the float32 assertion runs.
- Built [`gnovm/cmd/gno`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/cmd/gno/main.go#L1) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/cmd/gno/main.go#L1) at 7817f6e1d and at master d2869dceb and ran both on `println(1e10000 / 1e10000)`: master prints `1`, this head panics at preprocess with `numerator exceeds 4096 bits`. Go prints `1`.
- The guard's placement after `big.Rat.SetString` is not a denial-of-service surface. `SetString` caps literal exponents at ±1e6 on its own, so the worst-case rational a literal can build is about 3.3 Mbit and `var m = -1e1000000` reaches the guard and rejects in 40 ms.
- `1e-1233` is accepted and `1e-1234` rejected at 7817f6e1d, fixing the cutoff at 4096 bits of denominator.
- Green at 7817f6e1d: `TestParity`, `TestConvert*`, `TestBounded*`, `TestValuesString*`. `TestFiles` fails only on `float9.gno` and `float10.gno`; the other local failures (`redeclaration3/4`, `redeclaration_global1`, `type41`, `switch13`, `types/*_f0`, `types/eql_0b4`) reproduce identically on master d2869dceb and are `go/types` message drift under go1.26.

## Open questions
- The amino encoding of `BigdecValue` changes from a decimal string to a ratio string, shifting the app hash for any persisted realm state holding an untyped float constant. Old decimal-string state still parses under `big.Rat.SetString`, but yields the truncated decimal as an exact rational, not the original fraction. Not posted: whether live state needs a migration or a testnet reset is the maintainers' call, not a code defect, and the hash bump is documented as expected.
- [@thehowl](https://github.com/gnolang/gno/pull/5867#issuecomment-4834298443) asked for a `big.Rat` fork wired to `softfloat`, and the author agreed. Nothing in the current `Rat` code path uses hardware floating point: `Rat.Float64` assembles the result from integer operations and `Rat.SetString` goes through `big.Int`. Not posted: the work is already agreed in-thread, and the determinism argument for it should be settled there, not restated as a review finding.
- If the fallback lands as a fixed-precision `big.Float`, the amino encoding of `BigdecValue` needs to round-trip that shape too, not just `RatString()`. Not posted: it depends on a fix that does not exist yet.
- The related [#5740](https://github.com/gnolang/gno/pull/5740) and this PR both rewrite the literal-parse block in [`op_eval.go`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/pkg/gnolang/op_eval.go#L88-L110) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_eval.go#L88), against `apd` and `big.Rat` respectively. Whichever lands second rewrites the other. Not posted: it is a sequencing call for the maintainers, and the guard finding already tells the author what the parse path has to do.
