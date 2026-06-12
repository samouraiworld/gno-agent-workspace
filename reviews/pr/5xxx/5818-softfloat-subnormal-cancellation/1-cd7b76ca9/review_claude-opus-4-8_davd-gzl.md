# PR #5818: fix(gnovm): correct softfloat add/sub for normal operands cancelling to subnormal

URL: https://github.com/gnolang/gno/pull/5818
Author: omarsy | Base: master | Files: 3 | +90 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: cd7b76ca9 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5818 cd7b76ca9`

**TL;DR:** Gno does float math in software (softfloat) so every node gets the same bit-for-bit result. That software code had a bug: when two close, opposite-sign numbers subtract down to a very tiny "subnormal" value, it returned a wrongly-scaled answer that disagreed with real hardware and with `go run`. This PR adds the one missing normalization step that fixes it.

**Verdict: APPROVE** — correct, minimal, well-scoped fix; verified bit-for-bit against hardware over 30M+ pairs and live via `gno run`; no concerns.

## Summary
gno's softfloat is a verbatim copy of Go's `runtime/softfloat64.go`, used on every node so float arithmetic is deterministic instead of hardware-dependent. `fpack64`/`fpack32`, the routines that re-assemble a float from `(sign, mantissa, exponent)`, mishandle the subnormal path: when `fadd64`/`fsub64` cancel two near-equal opposite-sign normals into a subnormal result (`|x+y| < 2.2e-308`), they reset to the un-normalized mantissa `mant0` and only right-shift to align the exponent, never left-normalizing it first. The result comes out off by roughly six orders of magnitude (the issue's case returns `202154086` instead of `847895691526144`). The fix inserts the missing left-shift normalization loop, identical to the one `fpack` already runs on its normal path, before the subnormal alignment. It is a no-op for already-normalized callers (`fmul64`/`fdiv64`/conversions), so only the add/sub cancellation case changes.

## Fix
The denormal branch of `fpack64`/`fpack32` resets `mant, exp = mant0, exp0` (the un-rounded original) and then enters `for exp < bias { mant >>= 1; exp++ }`, which only right-shifts and so leaves a heavily-cancelled `mant0` (well below `1<<mantbits`) un-normalized while `exp0` is still a normal-range exponent. The PR adds `for mant < 1<<mantbits { mant <<= 1; exp-- }` right after the reset, restoring the normalization invariant the alignment loop assumes. Because the file is generated (`DO NOT EDIT`), the patch is applied through the generator with a `strings.Count == 1` anchor guard so it survives `go generate` and fails loudly if a future Go toolchain reformats or fixes the upstream code. See [`runtime_softfloat64.go:130-148`](https://github.com/gnolang/gno/blob/cd7b76ca9/gnovm/pkg/gnolang/internal/softfloat/runtime_softfloat64.go#L130-L148) · [↗](../../../../../.worktrees/gno-review-5818/gnovm/pkg/gnolang/internal/softfloat/runtime_softfloat64.go#L130) and the generator at [`gen/main.go:81-122`](https://github.com/gnolang/gno/blob/cd7b76ca9/gnovm/pkg/gnolang/internal/softfloat/gen/main.go#L81-L122) · [↗](../../../../../.worktrees/gno-review-5818/gnovm/pkg/gnolang/internal/softfloat/gen/main.go#L81).

## Verification

Verified on the reviewed head `cd7b76ca9`:

- The included [`TestFloat64`](https://github.com/gnolang/gno/blob/cd7b76ca9/gnovm/pkg/gnolang/internal/softfloat/runtime_softfloat64_test.go#L35) · [↗](../../../../../.worktrees/gno-review-5818/gnovm/pkg/gnolang/internal/softfloat/runtime_softfloat64_test.go#L35) passes; the two added base operands cross with every other value over `+ - * /` against hardware.
- Re-ran the generator (`cd gen && go run .`) under Go 1.26.4 (a different toolchain than the author's): the anchor guard found exactly one match per patch and reproduced both committed files byte-for-byte. The committed code is in sync with the generator.
- Reviewer adversarial sweep, 0 mismatches vs hardware: 5M cancellation pairs (~110k landing subnormal), 5M mul/div pairs (confirms the no-op claim), 5M `f64->f32` narrowings including float32-subnormal results (the reachable `fpack32` path), and a 20M-pair stress over right-shift / additive-underflow / boundary cases (~10.4M subnormal hits).
- Reverting just the fix made the issue MRE return the wrong `202154086` and produced 205,016 mismatches in the 5M cancellation sweep, confirming the change is load-bearing.
- Live: built `gno` from the worktree and ran the issue's exact MRE through `gno run` — both add and sub return `847895691526144`, matching `go run`.

## Glossary
- **softfloat** — software IEEE-754 implementation; Gno uses it always so float results are deterministic across hardware.
- **fpack64 / fpack32** — assemble a float from sign + mantissa + exponent, handling rounding and the normal/subnormal split.
- **subnormal (denormal)** — float smaller than the smallest normal (`< 2.2e-308` for f64), with no implicit leading 1 bit.
- **cancellation** — subtracting two near-equal values, leaving a tiny result with few significant bits.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`gen/main.go:154-158`](https://github.com/gnolang/gno/blob/cd7b76ca9/gnovm/pkg/gnolang/internal/softfloat/gen/main.go#L154-L158) · [↗](../../../../../.worktrees/gno-review-5818/gnovm/pkg/gnolang/internal/softfloat/gen/main.go#L154) — the regression operands are injected as bare `math.Float64frombits(...)` literals with no decimal value in the comment, unlike the surrounding entries (`// first normal`, `// all 1s mantissa`). A reader can't tell what magnitude they represent without decoding the bits. Optional: append the decimal (`-2.662e-301` / `2.662e-301`) so the test data is self-documenting.

## Missing Tests
None. The two added operands plus the all-pairs cross product cover the add/sub cancellation path; the reviewer sweeps above confirm the broader space (right-shift, additive-underflow, f32 narrowing) is also correct, and those are beyond what a unit test should enumerate.

## Suggestions
None.

## Open questions
- The PR notes the bug was reported upstream to Go. When upstream fixes it, the generator's anchor guard trips on the next toolchain bump and the workaround is removed by hand. That is the intended design and needs no action now; flagging only so the eventual guard failure is read as "expected, remove the patch," not a regression. Not posted: no decision for the author in this PR.
