# PR #5221: fix(gnovm): correct parsing of float values from args

URL: https://github.com/gnolang/gno/pull/5221
Author: ltzmaxwell | Base: master | Files: 3 | +129 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 (1M context) (deep) | Commit: `28383f0` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5221 28383f0`

**TL;DR:** When you call a realm function with `gnokey maketx call ... -args '1.5'`, the node turns that text into a typed float before running the function. This PR makes that text-to-float step reject `NaN` and `Inf` and turn `-0.0` into `0.0`.

**Verdict: NEEDS DISCUSSION** — guiding principle for this boundary should be "behave like Go." Go accepts NaN and Inf as float arguments, and the GnoVM produces and stores both itself, so rejecting them here is the divergence, not a fix; the stated determinism/malleability rationale does not hold. Recommended resolution: drop the NaN/Inf rejection (match Go); keep the `-0`→`+0` fold, which is the one part that matches Go, but relabel its reason. [@thehowl raised the same objection](https://github.com/gnolang/gno/pull/5221#issuecomment-3990173584) and it is unanswered.

## Summary

`convertArgToGno` turns MsgCall string args into typed Gno values; for floats it delegates to [`convertFloat`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L202-L227) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L202-L227), which runs `apd.NewFromString` then `strconv.ParseFloat`. The PR adds three post-parse steps: panic on `NaN`, panic on `±Inf`, and rewrite any signed zero to `+0`. The change is reached only from MsgCall ([`keeper.go:678`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/keeper.go#L678) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/keeper.go#L678)); MsgRun and qeval parse Gno source directly and never hit it. Gno's design contract is that it behaves like Go; measured against that, the NaN/Inf rejection moves away from Go, and the malleability rationale in the PR description does not survive inspection (verification below).

## Glossary

- `convertArgToGno` / `convertFloat` — string-to-TypedValue marshaller for MsgCall args, and its float helper
- MsgCall vs MsgRun — MsgCall passes string args to a named function (the only path this PR touches); MsgRun executes user-supplied Gno source
- softfloat — the deterministic software IEEE-754 implementation the VM uses for all float arithmetic
- malleability — a third party rewriting a signed transaction into a different-but-still-valid one, or one logical operation having multiple valid signed encodings

## Guiding principle: behave like Go

Gno is a deterministic Go; the float-arg boundary should match Go unless there is a concrete reason to diverge. There are two distinct Go behaviors to measure against, and they disagree on this PR:

- **Go runtime / argument passing** — calling `func F(x float64)` never rejects a value; NaN, ±Inf, and `-0` are all valid float64 arguments. A `maketx call` is argument passing, so this is the closer analogy.
- **Go source literals (compile-time constants)** — you cannot *write* `NaN` or `Inf` as a literal, and the constant `-0.0` folds to `+0`.

The PR follows the source-literal reading (reject NaN/Inf, fold `-0`). The argument-passing reading is the better fit for a function call and says: accept NaN/Inf, and the `-0` fold is a harmless extra that happens to match the constant-folding rule.

## Go / Gno parity

What each input does at Go's parser, as a Go source literal, inside the GnoVM, at this MsgCall boundary after the PR, and what "behave like Go" implies. Bits are float64.

| Input | Go `strconv.ParseFloat` | Go source literal | GnoVM can hold it? | MsgCall arg (this PR) | Go-consistent answer |
|---|---|---|---|---|---|
| `NaN` | accepted, `0x7FF8000000000001` | not expressible | yes — `math.NaN()` = `0x7FF8000000000000`; `0.0/0.0` | **rejected (panic)** | accept |
| `Inf` / `-Inf` | accepted, `0x7FF0…` / `0xFFF0…` | not expressible | yes — `math.Inf(±1)`; `1.0/0.0` | **rejected (panic)** | accept |
| `-0.0` | `-0`, `0x8000000000000000` | folds to `+0`, `0x0` | computed `-0` exists; map keys normalize to `+0` | folded to `+0` | either; fold matches the literal rule |
| `1e309` (overflow) | `+Inf` + `ErrRange` | n/a | n/a | rejected via parse error, not the new `IsInf` check | reject (Go errors too) |

Go's own parser accepts NaN/Inf with fixed canonical bits, so the PR is stricter than Go. The only place the PR matches Go is `-0`, and only against the *literal* rule.

## Critical (must fix)

None.

## Warnings (should fix)

- **[rejecting NaN/Inf diverges from Go and from the VM's own behavior]** [@thehowl](https://github.com/gnolang/gno/pull/5221#issuecomment-3990173584) [`convert.go:214-218`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L214-L218) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L214-L218) — Go accepts NaN and ±Inf as float arguments and so does the rest of the VM; reject them here only with a concrete reason, and the stated one (determinism / malleability) does not hold.
  <details><summary>details</summary>

  Three things, each pulling toward "accept." (1) Go-fidelity: passing a float64 in Go never rejects a value, and `strconv.ParseFloat` itself accepts `NaN`/`Inf`/`Infinity`; the PR is stricter than Go for no stated Go-incompatibility. (2) VM consistency: `math.NaN()` and `math.Inf()` are in the stdlib ([`bits.gno:20`, `:31`](https://github.com/gnolang/gno/blob/28383f0/gnovm/stdlibs/math/bits.gno#L20-L31) · [↗](../../../../../.worktrees/gno-review-5221/gnovm/stdlibs/math/bits.gno#L20-L31)), and float division has no zero guard — unlike the integer cases, the float cases of `quoAssign` call `softfloat.Fdiv` directly ([`op_binary.go:890-895`](https://github.com/gnolang/gno/blob/28383f0/gnovm/pkg/gnolang/op_binary.go#L890-L895) · [↗](../../../../../.worktrees/gno-review-5221/gnovm/pkg/gnolang/op_binary.go#L890-L895)), so `1.0/0.0` yields `+Inf` and `0.0/0.0` yields `NaN`, deterministically (confirmed). So a function can receive NaN/Inf from another realm or `maketx run`, compute them internally, and persist them, but not be handed them by a direct `maketx call`. (3) The rationale is empty: the signature commits to the arg *string* (`MsgCall.Args []string`, [`msgs.go:98`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/msgs.go#L98) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/msgs.go#L98)), and every spelling of NaN/Inf parses to one canonical bit pattern (apd strips NaN payloads), so there is nothing malleable or non-deterministic to close. Fix: drop the two panics so NaN/Inf are accepted, matching Go; if there is a real policy reason to forbid them, state it and accept that the boundary then differs from both Go and the VM.
  </details>

## Nits

- [`convert.go:220-224`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L220-L224) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L220-L224) — the `-0`→`+0` fold is the one Go-consistent part (Go folds the literal `-0.0` to `+0`), so it can stay, but its comment ("prevent malleability") is the wrong reason and the step is redundant at the only consensus-relevant place since `MapKeyBytes` already maps `-0`→`0` ([`values.go:1117`, `:1126`](https://github.com/gnolang/gno/blob/28383f0/gnovm/pkg/gnolang/values.go#L1117-L1126) · [↗](../../../../../.worktrees/gno-review-5221/gnovm/pkg/gnolang/values.go#L1117-L1126)). Reword the comment to "match Go's `-0.0` literal folding."
- [`convert.go:215`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert.go#L215) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L215) — if the NaN/Inf rejection survives the discussion, `"float%d does not accept NaN"` reads like a parser-internal spec; `"float%d argument cannot be NaN"` is clearer to the realm caller who sees it as a tx error. Moot if the rejection is dropped.

## Missing Tests

- **[float32 signed-zero underflow path untested]** [`convert_test.go:117`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/sdk/vm/convert_test.go#L117) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert_test.go#L117) — if the `-0` fold stays, the test covers float64 `-0.0`/`-0` but not a value that underflows to float32 `-0`. Verified correct behaviorally: `convertArgToGno("-1e-50", Float32Type)` stores `0x00000000` because `convertFloat` passes precision 32 to `ParseFloat`, which rounds to float32 `-0` and trips the fold before the float32 cast. A regression test on `"-1e-50"` for `Float32Type` would lock this in.

## Suggestions

- [`maketx_call_float_args.txtar:38-39`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L38-L39) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L38-L39) — the float64 `-0.0` case only asserts the *formatted* output is `"0"`, which `strconv.FormatFloat` also prints for true `+0`, so a float64 fold regression would slip through. Only the float32 path has a sign-bit assertion ([`:45`](https://github.com/gnolang/gno/blob/28383f0/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L45) · [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L45)). Add a `CheckSignBitFloat64` for parity.

## Open questions

- If the principle is "behave like Go," the parser grammar diverges too: Go float literals allow hex floats (`0x1p-2`) and digit separators (`1_000.0`), but the `apd.NewFromString` front end rejects both (confirmed). Worth a separate decision, not this PR — apd is there for exact decimal→float rounding, so matching Go's literal grammar has its own tradeoffs.
- If NaN is accepted, note that the boundary's NaN (`strconv` → `0x7FF8000000000001`) is not bit-identical to the VM's own `math.NaN()` (`0x7FF8000000000000`). Both are deterministic and it is irrelevant to `==` (NaN never equals NaN) and to map keys (NaN keys are intercepted), so it is not a correctness concern, just a note for whoever resolves the thread.
