# PR #5221: fix(gnovm): correct parsing of float values from args

URL: https://github.com/gnolang/gno/pull/5221
Author: ltzmaxwell | Base: master | Files: 3 | +129 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 (1M context) (deep) | Commit: `28383f0` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5221 28383f0`

**TL;DR:** When you call a realm function with `gnokey maketx call ... -args '1.5'`, the node turns that text into a typed float before running the function. This PR makes that text-to-float step reject `NaN` and `Inf` and turn `-0.0` into `0.0`.

**Verdict: NEEDS DISCUSSION** — the code is correct and well-tested, but its stated reason (determinism / anti-malleability) does not hold up: NaN, Inf, and every other input already parse to deterministic canonical bits here, and the signature commits to the arg string, not the float. Rejecting NaN/Inf is a policy choice that contradicts how the VM treats those values everywhere else; [@thehowl's pushback](https://github.com/gnolang/gno/pull/5221#issuecomment-3990173584) is well-founded and still unanswered. Resolve the policy before merge.

## Summary

`convertArgToGno` turns MsgCall string args into typed Gno values; for floats it delegates to [`convertFloat`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L202-L227) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L202-L227), which runs `apd.NewFromString` then `strconv.ParseFloat`. The PR adds three post-parse steps: panic on `NaN`, panic on `±Inf`, and rewrite any signed zero to `+0`. The change is reached only from MsgCall ([`keeper.go:678`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/keeper.go#L678) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/keeper.go#L678)); MsgRun and qeval parse Gno source directly and never hit it. The PR description motivates the change as "determinism / transaction malleability", but verification (below) shows there is no malleability or non-determinism at this boundary to fix: the change is a value-policy restriction, and a partial one, since the VM produces and accepts NaN/Inf freely everywhere else.

## Glossary

- `convertArgToGno` / `convertFloat` — string-to-TypedValue marshaller for MsgCall args, and its float helper
- MsgCall vs MsgRun — MsgCall passes string args to a named function (the only path this PR touches); MsgRun executes user-supplied Gno source
- softfloat — the deterministic software IEEE-754 implementation the VM uses for all float arithmetic
- malleability — a third party rewriting a signed transaction into a different-but-still-valid one, or one logical operation having multiple valid signed encodings

## Go / Gno parity

What each input does at Go's parser, as a Go source literal, inside the GnoVM, and at this MsgCall boundary after the PR. Bits are float64.

| Input | Go `strconv.ParseFloat` | Go source literal | GnoVM can hold it? | MsgCall arg (this PR) |
|---|---|---|---|---|
| `NaN` | accepted, `0x7FF8000000000001`, nil err | not expressible (no literal) | yes — `math.NaN()` = `0x7FF8000000000000`; `0.0/0.0` via softfloat | **rejected (panic)** |
| `Inf` / `-Inf` | accepted, `0x7FF0…` / `0xFFF0…`, nil err | not expressible | yes — `math.Inf(±1)`; `1.0/0.0` via softfloat | **rejected (panic)** |
| `-0.0` | `-0`, `0x8000000000000000` | folds to `+0`, `0x0` | computed `-0` exists; map keys normalize to `+0` | canonicalized to `+0` |
| `1e309` (overflow) | `+Inf` + `ErrRange` | n/a | n/a | rejected — pre-existing parse-error path, not the new `IsInf` check |

Two takeaways. (1) Go's own parser accepts NaN/Inf with fixed canonical bits; the PR is stricter than Go here. (2) `-0.0` is the one place the PR matches Go: a Go *source literal* `-0.0` folds to `+0`, so canonicalizing the arg mirrors literal semantics, not malleability.

## Critical (must fix)

None.

## Warnings (should fix)

- **[the stated reason for the change isn't real]** [@thehowl](https://github.com/gnolang/gno/pull/5221#issuecomment-3990173584) [`convert.go:214-218`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L214-L218) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L214-L218) — "determinism / prevents malleability" does not hold at this boundary; rejecting NaN/Inf is a policy choice and should be justified as one or dropped.
  <details><summary>details</summary>

  The signature and tx hash commit to the arg *string* (`MsgCall.Args []string`, [`msgs.go:98`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/msgs.go#L98) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/msgs.go#L98)), and conversion runs only at execution time, long after signature verification. No third party can rewrite `"0.0"` to `"-0.0"` or `"1.0"` to `"NaN"` without invalidating the signature, so there is no malleability to close. Determinism is equally intact without the change: `strconv.ParseFloat` yields one canonical NaN (`0x7FF8000000000001`) and one canonical `±Inf` for every spelling (`NaN`, `nan`, `Inf`, `Infinity`), and apd strips any NaN payload, so exactly one NaN bit pattern is even reachable from a string. Verified by exhausting the input forms (see repro). Fix: replace the "malleability" rationale with the real one ("we choose to forbid NaN/Inf in the MsgCall ABI because …"), or drop the NaN/Inf rejection. This is the load-bearing question for the verdict; thehowl already raised it and it is unanswered.
  </details>

- **[rejecting NaN/Inf contradicts how the VM treats them everywhere else]** [`convert.go:214-218`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L214-L218) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L214-L218) — a realm can compute, store, and receive NaN/Inf by every other route; only this one boundary forbids them, so the same function accepts a value from one caller and rejects it from another.
  <details><summary>details</summary>

  `math.NaN()` and `math.Inf()` are in the stdlib ([`bits.gno:20`, `:31`](https://github.com/gnolang/gno/blob/28383f0/gnovm/stdlibs/math/bits.gno#L20-L31) · [↗](../../../../../.worktrees/gno-review-5221/gnovm/stdlibs/math/bits.gno#L20-L31)), and float division has no zero guard — unlike the integer cases, the float cases of `quoAssign` call `softfloat.Fdiv` directly ([`op_binary.go:890-895`](https://github.com/gnolang/gno/blob/28383f0/gnovm/pkg/gnolang/op_binary.go#L890-L895) · [↗](../../../../../.worktrees/gno-review-5221/gnovm/pkg/gnolang/op_binary.go#L890-L895)), so `1.0/0.0` yields `+Inf` and `0.0/0.0` yields `NaN`, deterministically (confirmed). These values are stored as raw bits and handled at every consensus-sensitive point: map keys normalize `-0`→`0` and intercept NaN keys ([`values.go:1117-1131`](https://github.com/gnolang/gno/blob/28383f0/gnovm/pkg/gnolang/values.go#L1117-L1131) · [↗](../../../../../.worktrees/gno-review-5221/gnovm/pkg/gnolang/values.go#L1117-L1131)). So `func F(cur realm, x float64)` can receive NaN/Inf when called from another realm or via `maketx run`, can compute them internally, and can persist them — but cannot be handed them by a direct `maketx call`. Fix: decide deliberately whether the MsgCall ABI should differ from the rest of the VM here; if yes, say so in a comment and accept the asymmetry; if no, drop the rejection.
  </details>

## Nits

- [`convert.go:220-224`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L220-L224) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L220-L224) — the `-0`→`+0` step is fine, but its comment ("prevent malleability") is the wrong reason and it is redundant at the one consensus-relevant place: `MapKeyBytes` already maps `-0`→`0` ([`values.go:1117`, `:1126`](https://github.com/gnolang/gno/blob/28383f0/gnovm/pkg/gnolang/values.go#L1117-L1126) · [↗](../../../../../.worktrees/gno-review-5221/gnovm/pkg/gnolang/values.go#L1117-L1126)). The honest justification is Go parity: a Go source literal `-0.0` folds to `+0`, so the arg boundary should too. Reword the comment to say that.
- [`convert.go:215`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L215) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L215) — `"float%d does not accept NaN"` reads like a parser-internal spec; `"float%d argument cannot be NaN"` is clearer to the realm caller who sees it as a tx error. Carried from round 1.

## Missing Tests

- **[float32 signed-zero underflow path untested]** [`convert_test.go:117`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert_test.go#L117) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert_test.go#L117) — the canonicalization test covers float64 `-0.0`/`-0` but not a value that underflows to float32 `-0`. Verified correct behaviorally: `convertArgToGno("-1e-50", Float32Type)` stores `0x00000000` because `convertFloat` passes precision 32 to `ParseFloat`, which rounds to float32 `-0` (signbit set, value zero) and trips the canonicalization before the float32 cast. A regression test on `"-1e-50"` for `Float32Type` would lock this in; without it, a refactor that canonicalized only literal `-0` could regress the float32 underflow case invisibly.

## Suggestions

- [`maketx_call_float_args.txtar:38-39`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L38-L39) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L38-L39) — the float64 `-0.0` case only asserts the *formatted* output is `"0"`, which `strconv.FormatFloat` also prints for true `+0`, so a float64 canonicalization regression would slip through. Only the float32 path has a `CheckSignBitFloat32` sign-bit assertion ([`:45`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L45) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L45)). Add a `CheckSignBitFloat64` for parity.

## Open questions

- If the policy decision is to keep the rejection, NaN/Inf produced by another realm or by `maketx run` will still flow into the same functions, so realm authors who want to be safe must guard internally regardless. That weakens the "protect the contract" argument for boundary rejection; worth stating in the discussion, not a code change here.
