# PR #5648: fix(gnolang): O(N^2) in Go2Gno Span for BinaryExpr chains

**URL:** https://github.com/gnolang/gno/pull/5648
**Author:** @mvertes | **Base:** master | **Files:** 4 | **+166 -8**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR fixes a quadratic-time performance bug in `Go2Gno`'s `Span` computation for chains of binary expressions. When transpiling long `BinaryExpr` chains (e.g. `a + b + c + ... + z`), `go/ast.BinaryExpr` nodes form a left-leaning tree. Computing the span of each node previously called `SpanFromGo(gon)`, which walks the entire subtree to find the minimum and maximum positions — making the total cost O(N²) for an N-term chain.

**The fix:** Before recursing into children of a `*ast.BinaryExpr`, the code now computes the children's spans first (via `toExpr`, which calls `Go2Gno` recursively), then sets the parent's span directly from the already-computed child spans. The existing `setSpan` deferred guard checks `IsZero()` before falling back to `SpanFromGo`, so pre-setting a non-zero span causes the O(N) walk to be skipped entirely. Total cost drops from O(N²) to O(N).

**Additional work:**
- `gnovm/pkg/gnolang/go2gno_test.go`: A new `TestParseFile_BinaryChain_SpanCorrect` test validates span correctness for 3-term, 100-term, and 1000-term chains. A benchmark `BenchmarkGo2Gno_BinaryChain` demonstrates the performance improvement.
- `docs/go2gno-span-binaryexpr-adr.md`: An architecture decision record with CPU profile data, root-cause analysis, and a follow-up audit of other potentially-quadratic AST node types.

## Test Results

- **Existing tests:** PASS — full `go test ./gnovm/pkg/gnolang/... -short` suite (59s). No regressions.
- **New tests:** PASS — `TestParseFile_BinaryChain_SpanCorrect` (3-term, 100-term, 1000-term chains all produce correct spans matching the expected column range).
- **Benchmark:** `BenchmarkGo2Gno_BinaryChain` shows performance scaling well below O(N²) for 4× N increases:
  - 100 terms: baseline
  - 400 terms: ~3.9× time (O(N²) would be 16×)
  - Confirmed sub-quadratic. Remaining overhead is Go parser time (unavoidable).
- **Build:** `go build ./...` passes.

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/go2gno.go` (BinaryExpr fast path, silent fallback) — When the fast path fires, the code reads `lspan` and `rspan` from the already-translated children via `GetSpan()`. If either span is zero (because the child translation set a zero span for some reason), the code falls through to the original `SpanFromGo` call. This fallback is unreachable under current invariants (any successfully-translated `*ast.BinaryExpr` child will have a non-zero span), but it is silent — there is no log or assertion to alert a future developer that the fast path was bypassed. A `panic("BinaryExpr child has zero span after translation")` or `//nolint` comment noting the invariant would make the code easier to audit.

## Nits

- [ ] `docs/go2gno-span-binaryexpr-adr.md` — The ADR is well-written. Minor: the "follow-up audit" section lists `CallExpr`, `IndexExpr`, `SliceExpr` as candidates for the same pattern. A tracking issue number would help, since the ADR will otherwise become stale once those follow-ups land.
- [ ] `go2gno_test.go:TestParseFile_BinaryChain_SpanCorrect` — The test only uses left-associative `+` chains. A brief comment explaining why this covers the right-leaning case (it doesn't, the test only checks left-leaning since that is what `go/ast` produces for `a+b+c+...`) would help readers.

## Missing Tests

- [ ] Right-branching (parenthesized) binary expression: `a + (b + (c + d))`. While `go/ast` always produces left-leaning trees for associative operators, an explicit test with parentheses would confirm the span is still correct after the fix.
- [ ] Mixed-precedence chain: `a + b * c + d`. The AST for this is not fully left-leaning; `b * c` is a subtree. A span correctness test for this case would lock in the expected behavior.
- [ ] Single non-chained `BinaryExpr`: `a + b` with exactly two leaf operands. The fast path should work correctly here (2-term = base case), and a test would document that it isn't broken for the trivial case.

## Suggestions

- The ADR mentions `CallExpr`, `IndexExpr`, and `SliceExpr` as other potentially-quadratic cases. Adding a `TODO(follow-up): apply same span-precompute pattern to CallExpr/IndexExpr/SliceExpr` comment near the `BinaryExpr` fix would ensure the pattern is extended consistently.
- The benchmark is currently in `go2gno_test.go` alongside unit tests. Given its length (it generates source strings), moving it to a dedicated `go2gno_bench_test.go` file would keep the test file readable.

## Questions for Author

1. **Generalization**: The fast path is only applied to `*ast.BinaryExpr`. Are `UnaryExpr`, `ParenExpr`, and `StarExpr` known to be safe to skip, or were they excluded because they don't form long chains in practice?
2. **ADR status**: Is `docs/go2gno-span-binaryexpr-adr.md` intended to stay in the repo permanently, or is it a temporary PR artifact that will be removed after merge? The rest of the repo uses `adr/` directories under packages (e.g. `tm2/adr/`) rather than a top-level `docs/` path.
3. **Profile data**: The ADR references a CPU profile. Was the profile collected against a real gno.land contract, a synthetic benchmark, or the test suite? Knowing this helps calibrate how impactful the fix is in production.

## Verdict

APPROVE — The fix is algorithmically correct, well-benchmarked, and accompanied by a clear ADR; the single warning (silent fallback) and missing edge-case tests are minor and can be addressed in a follow-up.
