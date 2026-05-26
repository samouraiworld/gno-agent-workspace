# PR #5221: fix(gnovm): correct parsing of float values from args

URL: https://github.com/gnolang/gno/pull/5221
Author: ltzmaxwell | Base: master | Files: 3 | +129 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7 (1M context)

Verdict: NEEDS DISCUSSION — fix is narrow, correct, and well-tested for the MsgCall string-arg boundary, but [@thehowl](https://github.com/gnolang/gno/pull/5221#issuecomment-3990173584) raised an unanswered design question about whether NaN/Inf should be rejected at all; resolve that thread before merge.

## Summary

`convertArgToGno` parses MsgCall string args into typed Gno values. For floats, the prior code took the parsed `float64` as-is, so `"-0.0"` → bits `0x8000000000000000` and `"NaN"` → bits `0x7FF8000000000001` — different bit patterns for arguably-equivalent values, observable inside the realm via `math.Signbit` / `math.Float64bits`. The PR adds three post-parse checks in [`convertFloat`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L214-L224): reject NaN, reject ±Inf, and force any zero with sign-bit-set back to `+0`. Scope is correct — the function is reached only from MsgCall (see [`keeper.go:678`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/keeper.go#L678)); MsgRun and qeval parse Gno source directly and are unaffected.

## Glossary

- `convertArgToGno` — string-to-TypedValue marshaller for MsgCall args
- `convertFloat` — internal helper called by the float32/float64 cases of `convertArgToGno`
- `assertNoPlusPrefix` — pre-check that panics on a leading `+`
- MsgCall vs MsgRun — public message types; MsgCall takes a function name plus string args (the only boundary this PR touches); MsgRun executes a user-supplied Gno source file

## Fix

Before: `convertFloat` returned whatever `apd.NewFromString` + `strconv.ParseFloat` produced, including NaN, ±Inf, and `-0`. After: the function panics on NaN or Inf and rewrites any `-0` (signbit set, value `== 0`) to `+0`. Verified empirically that the canonicalization fires for all paths reaching `-0` — literal `"-0"` / `"-0.0"`, scientific notation `"-0e0"` / `"-0.0e+10"`, and underflow `"-1e-324"` all collapse to bits `0x0000000000000000`. The check is on the resulting `float64`, not on the string, so case variants (`"nan"`, `"inf"`, `"infinity"`) and any other surface form that apd recognizes are caught uniformly.

## Critical (must fix)

None.

## Warnings (should fix)

- **[design intent contested by maintainer]** [@thehowl](https://github.com/gnolang/gno/pull/5221#issuecomment-3990173584) [`convert.go:214-219`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L214-L219) — thehowl pushed back on rejecting NaN/Inf, no answer recorded.
  <details><summary>details</summary>

  thehowl: "I don't think they should be disallowed; they are float values that a contract can and should expect (if they don't come from an EOA, they can come from another contract or a `maketx run`). Is there a specific 'vulnerability' you had in mind with this fix?" The PR description cites "transaction malleability" but no specific exploit path. The argument for rejection — NaN payload bits differ across IEEE-754 producers — is true in general but in this code path apd+ParseFloat produce a deterministic NaN (`0x7FF8000000000001`) and a deterministic ±Inf, so determinism is already preserved without rejection. The argument against — contracts can legitimately receive NaN/Inf from in-VM math (e.g. `1.0/0.0`, `math.NaN()`) or cross-contract returns; an EOA-facing MsgCall function that handles NaN-safe code already needs to handle the value, so the parser-level rejection adds an inconsistency: same value, different acceptance depending on producer. Fix: either reply to thehowl with the concrete threat model (e.g. "NaN comparison short-circuits an access-control check in realm X") or narrow the PR to canonicalization only (drop NaN/Inf rejection, keep `-0 → +0`, which has a real bit-pattern divergence concern at the string boundary).
  </details>

- **[asymmetric float canonicalization]** [`convert.go:222-224`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L222-L224) — only `-0` is canonicalized; if the design philosophy is "the string boundary must produce canonical bit patterns", document why `-0` is the only canonicalization needed (or applied).
  <details><summary>details</summary>

  The fix asserts that bit-level equivalence at the string-arg boundary matters enough to rewrite `-0`. But there are no other bit-pattern divergences at this boundary today only because the apd+ParseFloat pipeline happens to be deterministic. If the codebase later swaps the parser, or if someone introduces a new boundary that uses a different parser (e.g. `strconv.ParseFloat` directly), the invariant is unstated and will silently drift. Fix: add a one-line comment above the canonicalization explaining the invariant — "MsgCall string args must produce a canonical bit pattern per value; the apd+ParseFloat path already does this for everything except `-0`, which we normalize here" — so a future reader understands why only `-0` is touched.
  </details>

## Nits

- [`convert.go:222`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L222) — `f64 == 0 && math.Signbit(f64)` works but `math.Float64bits(f64) == 0x8000000000000000` reads more intentful when the goal is bit canonicalization. Style only.
- [`convert.go:215`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L215) — `"float%d does not accept NaN"` reads like a parser-spec error; `"float%d argument cannot be NaN"` is clearer to the realm caller who'll see this in the tx error.
- [`convert_test.go:117-136`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert_test.go#L117-L136) — the canonicalization test only checks `-0.0` and `-0`. Adding `-0e0`, `-1e-400` (underflow), and `-0.0e+10` would lock in that the fix is on the *float value*, not on the *string shape* — useful as documentation if anyone later proposes a string-level allowlist.

## Missing Tests

- **[underflow path uncovered]** [`convert_test.go:117`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert_test.go#L117) — `-1e-324` (or any expression that underflows to negative zero) is not tested. Verified locally that the fix handles it; a regression test would prevent a future "optimization" from short-circuiting the canonicalization on literal-`-0`-only input.
  <details><summary>details</summary>

  Repro shows the value reaches the canonicalization branch:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5221 -R gnolang/gno
  cat > /tmp/underflow.go <<'EOF'
  package main
  import ("fmt"; "math"; "strconv"; "github.com/cockroachdb/apd/v3")
  func main() {
      d, _, _ := apd.NewFromString("-1e-400")
      f, _ := strconv.ParseFloat(d.String(), 64)
      fmt.Printf("bits=%016X signbit=%v zero=%v\n", math.Float64bits(f), math.Signbit(f), f == 0)
  }
  EOF
  go run /tmp/underflow.go
  rm /tmp/underflow.go
  # Output: bits=8000000000000000 signbit=true zero=true
  ```
  </details>

- **[no test for valid extreme values]** [`convert_test.go:88`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert_test.go#L88) — `math.MaxFloat64`, smallest normal, smallest subnormal aren't tested. The existing `TestConvertEmptyNumbers` covers `""`; adding edge floats would round out the boundary table.

## Suggestions

- [`maketx_call_float_args.txtar:38-42`](../../../../../.worktrees/gno-review-5221/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L38-L42) — the txtar checks `EchoFloat64(-0.0)` returns `"0"`, which proves the *formatted* output is `"0"`, but the formatter would also print `"0"` for true `+0`. The follow-up `CheckSignBitFloat32` line at L45-46 does prove the sign bit was cleared; consider adding `CheckSignBitFloat64` for parity so both type paths have a sign-bit assertion (currently float64 only has the formatted-output assertion).
  <details><summary>details</summary>

  As-is, if the float64 canonicalization regressed but `strconv.FormatFloat(-0, 'f', -1, 64)` continued to return `"0"` (which it does, per Go stdlib), the test would still pass. The float32 path catches the regression via `math.Signbit`; the float64 path doesn't.
  </details>

## Questions for Author

- What concrete attack or non-determinism motivated rejecting NaN and Inf (vs. canonicalizing or accepting)? thehowl's question is unanswered and load-bearing for the decision to merge.
- Was the realm-level case (NaN/Inf produced by `1.0/0.0` inside a contract or returned from another realm) considered? If those values are accepted in-VM, what's the principled reason the MsgCall boundary differs?
