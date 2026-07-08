# PR [#5864](https://github.com/gnolang/gno/pull/5864): fix(gnovm): fold -0 to +0 for float call args

URL: https://github.com/gnolang/gno/pull/5864
Author: davd-gzl | Base: master | Files: 2 | +57 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 2c6396d90 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5864 2c6396d90`

Round 2. Head advanced 662cbc5ba → 2c6396d90 (squash/force-push, PR content changed). The round-1 Nit landed: the fold now reads `f64 = math.Copysign(0, 1)` instead of the no-op `f64 = 0`. The round-1 Warning (NaN/Inf admitted on the same path) is resolved by decision, not by rejection: the author keeps NaN/Inf accepted and now documents why in the code comment and commit message, and a new go-parity test `TestConvertFloatMatchesGo` (replacing the removed `TestConvertFloatNegativeZeroFolded`) locks both the -0 fold and the NaN/Inf acceptance. The round-1 Missing Test (NaN/Inf behavior unpinned) is resolved. One delta: the new test dropped the float32-underflow `-1e-50` case the old test carried.

**TL;DR:** When a realm function is called on-chain with a float argument like `-0.0`, the value used to arrive with its sign bit set (negative zero), which no Gno source literal can produce. This clears the sign bit so a `-0.0` argument reaches the realm as plain `0`, matching how Go compiles a `-0.0` literal, and leaves NaN and Inf arguments accepted since the VM produces those at runtime anyway.

**Verdict: APPROVE** — the -0 fold is correct and verified end-to-end; the round-1 scope question is closed by an explicit, documented, tested decision to accept NaN/Inf, which introduces no determinism or malleability surface. One non-blocking missing-test note below.

## Summary
`convertFloat` turns a MsgCall string argument into a float64 via `apd.NewFromString` then `strconv.ParseFloat`, both of which keep the sign bit, so `"-0.0"` and `"-0"` arrived as negative zero. A Go source literal `-0.0` folds to `+0` at compile time, so the on-chain argument path diverged from what the same literal would produce in code. The fix clears the sign bit on any zero result, so `-0` folds to `+0` on both the float64 and float32 paths, including the float32-underflow case where a nonzero literal like `-1e-50` rounds to `-0`. Round 1 left open whether NaN/Inf should be rejected on the same path (as the superseded [#5221](https://github.com/gnolang/gno/pull/5221) did); this round settles it: they stay accepted because the VM produces them at runtime and every textual spelling collapses to a single canonical bit pattern, so admitting them adds nothing a realm float could not already hold.

## Examples
| MsgCall arg | float64 result before | after this PR |
|-------------|----------------------|---------------|
| `-0.0` | `-0` (sign bit set) | `+0` |
| `-0` | `-0` (sign bit set) | `+0` |
| `-1e-50` (as float32) | `-0` (underflow, sign bit set) | `+0` |
| `NaN` / `nan` / `NaN123` | canonical NaN | canonical NaN (accepted) |
| `Inf` / `Infinity` / `-Inf` | ±Inf | ±Inf (accepted) |
| `+Inf` / `sNaN` / `1e400` | rejected | rejected |

## Glossary
- malleability: two distinct byte encodings of one logical value, letting a tx be re-signed or replayed in a variant form. Here `-0` and `+0` are the same number with different bits; NaN and Inf, by contrast, canonicalize to one bit pattern each through the parse.

## Fix
`convertFloat` gains a zero-fold at [convert.go:216-223](https://github.com/gnolang/gno/blob/2c6396d90/gno.land/pkg/sdk/vm/convert.go#L216-L223) · [↗](../../../../../.worktrees/gno-review-5864-h/gno.land/pkg/sdk/vm/convert.go#L216-L223): after parsing, `if f64 == 0 { f64 = math.Copysign(0, 1) }` overwrites the sign bit with a `+0` whose bits are `0x0`. The single call site is [keeper.go:894](https://github.com/gnolang/gno/blob/2c6396d90/gno.land/pkg/sdk/vm/keeper.go#L894) · [↗](../../../../../.worktrees/gno-review-5864-h/gno.land/pkg/sdk/vm/keeper.go#L894), shared by both the MsgCall execution path and the `vm/qeval` query path, so both are covered. `math.Copysign(0, 1)` is byte-identical to the `0` literal round 1 verified live, so the runtime behavior is unchanged from that verification.

## Critical (must fix)
None.

## Warnings (should fix)
None. The round-1 Warning is resolved: NaN/Inf acceptance is now a documented, tested decision, and the parse leaves no non-canonical NaN or Inf bit pattern reachable (see Verified).

## Nits
None. The round-1 Nit (`f64 = 0` no-op) is fixed.

## Missing Tests
- **[dropped a Go-parity case that the old test covered]** `gno.land/pkg/sdk/vm/convert_goparity_test.go:25-36` — The new test asserts the fold only for strings that are literally zero (`"0"`, `"-0"`, `"-0.0"`); it dropped the float32-underflow `"-1e-50"` case the removed `TestConvertFloatNegativeZeroFolded` carried, which exercises a nonzero magnitude rounding to `-0` at float32 precision.
  <details><summary>details</summary>

  The zeros table loops both precisions over strings whose numeric value is already zero, so it never reaches the fold via underflow rounding. `"-1e-50"` is a normal negative float64 that rounds to `-0` only at float32 precision, so the fold catches it through a different route than a literal `-0`. Go folds the same source constant identically: `float32(-1e-50)` has bits `0x0` (verified), so this is a Go-parity case that belongs in this file. The production code still folds it correctly (verified), so nothing is broken; the coverage guarding it is simply gone. Fix: add a float32-only assertion after the zeros loop.

  Ready to add (paste after the zeros loop, before the NaN/Inf block):

  ```go
  // "-1e-50" is a normal negative float64 that underflows to -0 at float32
  // precision; Go folds float32(-1e-50) to +0, and so must the arg path.
  if got := convertFloat("-1e-50", 32); math.Signbit(got) {
      t.Errorf("convertFloat(%q, 32) has sign bit set, want +0", "-1e-50")
  }
  ```
  </details>

## Suggestions
None.

## Verified
- Fold, all zero routes: `convertFloat` returns `+0` (bits `0x0`, sign clear) for `"0"`, `"-0"`, `"-0.0"` at float32 and float64, and for the float32-underflow `"-1e-50"`. Reverting the fold restores negative zero (bits `0x8000000000000000`) for `"-0"`, `"-0.0"`, and the `"-1e-50"` underflow, so the fold is load-bearing on every route. [worktree harness on `convertFloat`]
- No NaN/Inf malleability: every NaN spelling (`"NaN"`, `"nan"`, `"NAN"`, payload `"NaN123"`) parses to Go's canonical NaN bits `0x7ff8000000000001`; `"Inf"`/`"Infinity"` give `+Inf`, `"-Inf"` gives `-Inf`; `"+Inf"`, signaling `"sNaN"`, and overflow `"1e400"`/`"-1e400"` are all rejected. No non-canonical NaN or Inf bit pattern reaches a realm, so accepting NaN/Inf adds no malleable encoding. [worktree harness]
- Go parity: `convertFloat("NaN", 64)` bits equal Go's `math.NaN()` (`0x7ff8000000000001`); `float32(-1e-50)` in Go is `0x0`; Go's `-0.0` literal is `0x0`. All match the arg path. [Go program]
- VM produces NaN/Inf at runtime: a Gno program computing `1.0/z`, `-1.0/z` (`z` a zero `float64`), `math.NaN()`, and `math.Inf(1)` prints `+Inf`, `-Inf`, and a true NaN inequality, so a realm float parameter can already hold these independent of this path. [`gno run`]
- `math.Copysign(0, 1)` bits equal the `0` literal (`0x0`), so the fold value is byte-identical to round 1's live-verified `f64 = 0`; end-to-end maketx behavior is unchanged from that round's live check.
- Green at 2c6396d90: `go test ./gno.land/pkg/sdk/vm/ -run 'Convert|Float'`, `TestConvertFloatMatchesGo`, `go vet ./gno.land/pkg/sdk/vm/`.

## Open questions
- The code comment justifies keeping NaN/Inf with "the VM produces them," but the VM also produces `-0` at runtime (`math.Copysign(0, -1)`), so that clause alone does not distinguish the fold from the acceptance. The load-bearing distinction is malleability: zero has several textual spellings mapping to different bits, while NaN/Inf spellings all canonicalize to one pattern each. Not posted: the decision and its outcome are correct, and the comment is defensible as written.
