# PR [#5867](https://github.com/gnolang/gno/pull/5867): feat(gnovm): replace cockroachdb/apd -> math/big.Rat

URL: https://github.com/gnolang/gno/pull/5867
Author: Villaquiranm | Base: master | Files: 22 | +420 -281
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 2b5e5a8a5 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5867 2b5e5a8a5`

**Round 2** (head `3c7de91d0` → `2b5e5a8a5`, PR content changed): the round-1 blocker is fixed. All three posted comments and every review-file finding are resolved. New verdict: APPROVE.

**TL;DR:** Gno evaluated untyped float constants like `1.0/3.0` with a decimal library that loses exactness, so `(1.0/3.0)*3.0 == 1.0` was `false` where Go says `true`. This PR swaps that library for exact rational arithmetic (`math/big.Rat`), matching Go. Round 1 flagged that the new size cap bounded only the denominator, letting a ~20-line package allocate gigabytes at compile time; round 2 adds the missing numerator bound and the tests that lock it.

**Verdict: APPROVE** — the numerator OOM is closed (verified: the round-1 481 MB / 72 s vector now rejects at preprocess in ~0.2 s), the error text now renders decimals, `const63` is a real filetest, and both guard directions have filetests. Remaining items are a display-only nit and the amino/app-hash migration, which is the maintainers' call.

## Summary
`apd.Decimal` stored `1/3` as a truncated decimal `0.333…`, so multiplying back by 3 gave `0.999…`, not `1` ([#5862](https://github.com/gnolang/gno/issues/5862)). `big.Rat` stores it as the exact fraction `{1,3}`, and `{1,3}×3 = 1`, matching Go's `go/constant`. Round 1's `ratGuard` capped only `Denom().BitLen()`, so an integer-valued `big.Rat` (`Denom() == 1`) grew its numerator without limit through repeated squaring at gasless preprocess. Round 2 adds a symmetric `Num().BitLen() > 4096` check on the same paths, so both components are now bounded before the multiply allocates.

## Resolved since round 1
- **[critical — numerator OOM]** [`op_binary.go:809-815`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/pkg/gnolang/op_binary.go#L809-L815) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L809) — `ratGuard` now panics when `Num().BitLen() > 4096`, on top of the existing denominator check. Applied at the same call sites as before: literal parse ([`op_eval.go:95`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/pkg/gnolang/op_eval.go#L95) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_eval.go#L95), [`:106`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/pkg/gnolang/op_eval.go#L106) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_eval.go#L106)) and the four arithmetic results in `op_binary.go`. Every literal and intermediate is bounded to ≤4096 bits, so each op's transient stays ~1 KB. Verified on 2b5e5a8a5: the round-1 vector (`a0=1e10000; aN=a(N-1)²`) rejects at preprocess in 234 ms with `numerator exceeds 4096 bits`, where 3c7de91d0 accepted a 17-line variant at 481 MB / 72 s.
- **[missing tests — size bound]** [`bigdec_guard_num.gno`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/tests/files/types/bigdec_guard_num.gno#L6) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/types/bigdec_guard_num.gno#L6) and [`bigdec_guard_denom.gno`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/tests/files/types/bigdec_guard_denom.gno#L18) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/types/bigdec_guard_denom.gno#L18) — one filetest per direction, both pass; the numerator test also carries the Go `TypeCheckError` cross-check.
- **[suggestion — error form]** [`values_conversions.go:1484`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/pkg/gnolang/values_conversions.go#L1484) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1484) — `bigdecErrString` renders terminating decimals in decimal form (`1.2`) and only falls back to `a/b` for non-terminating values (`1/3`). Verified: `const y int = 1.2` reports `1.2 not an exact integer`; `const y int = 1.0/3.0` reports `1/3 not an exact integer`. The terminating-decimal test (strip factors of 2 and 5 from the denominator) is correct, and `Denom().BitLen()` is a valid upper bound on the fractional-digit count.
- **[nit — dead reference]** [`const63.gno`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/tests/files/const63.gno#L1) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/const63.gno#L1) — rewritten as a self-contained filetest with a `// Output:` block; the dead `_verify/const63_go/main.go` pointer is gone.

## Also new in round 2 (reviewed, no blockers)
- [`values_conversions.go:1442`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/pkg/gnolang/values_conversions.go#L1442) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1442) — the bigdec→float32 cast now narrows through `softfloat.F64to32` instead of a hardware `float32(f64)`, so the rounding is platform-independent. A determinism improvement.
- [`values.go` `MarshalAmino`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/pkg/gnolang/values.go#L121) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values.go#L121) — encodes `RatString()` (lowest-terms `a/b` or `a`). Because `big.Rat` always normalizes, the encoding is now canonical, unlike `apd`'s decimal text. The one-time app-hash shift is covered by `apphash_crossrealm38_test.go`.
- No bigdec path bypasses the guard. The `bigint→bigdec` conversion ([`values_conversions.go:1334`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/pkg/gnolang/values_conversions.go#L1334) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1334)) does not compound and is bounded by the source integer; large integer literals are already stopped by the shift-amount cap (`1 << 20000` rejects at max 10000).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`values_string.go:68-69`](https://github.com/gnolang/gno/blob/2b5e5a8a5/gnovm/pkg/gnolang/values_string.go#L68-L69) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_string.go#L68) — `BigdecValue.String()` renders via `FloatString(10)`, so a directly printed untyped bigdec whose exact form needs more than 10 fractional digits (e.g. `1.0/2048.0` = `0.00048828125`) is rounded. Display-only: the exact conversion and error paths use `RatString`/`bigdecErrString`, and an untyped float constant reaches this method only through debug/error printing (normal `println` converts to its float64 default first). Not worth blocking; noting it carries from round 1.

## Missing Tests
None.

## Open questions
- The amino encoding of `BigdecValue` changes from a decimal string to a ratio string, shifting the app hash for any persisted realm state that holds an untyped float constant (`apphash_crossrealm38` is exactly that shift). Old decimal-string state still parses under `big.Rat.SetString`, but yields the truncated decimal as an exact rational, not the original fraction. Not posted: whether live state needs a migration vs a testnet reset is the maintainers' call, not a code defect, and the hash bump is documented as expected.
