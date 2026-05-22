# PR #5406: fix(gnovm): exclude comment tokens from parsing gas metering

**URL:** https://github.com/gnolang/gno/pull/5406
**Author:** notJoon | **Base:** master | **Files:** 2 | **+66 -0**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary
This PR addresses issue #4919 where documentation comments in `.gno` files increase runtime gas costs on the first (uncached) call to a package. The parser callback (`newParserCallback` in `go2gno.go`) charges gas for every token including `token.COMMENT`. Since parsed packages are cached after the first call, only the initial caller pays this comment-induced gas overhead, creating a disincentive for developers to document their contracts.

The fix introduces a `commentCostFactor` constant (set to `0`) alongside the existing `tokenCostFactor` and `nestingCostFactor` in `gnovm/pkg/gnolang/go2gno.go`. The parser callback now checks for `token.COMMENT` and charges `commentCostFactor` (zero) gas instead of the standard token cost. The Gno parser (a fork of Go's parser at `gnovm/pkg/parser/`) always runs with `scanner.ScanComments`, so `token.COMMENT` is emitted for every `//` and `/* */` comment and reaches the callback before the parser's own comment-skip logic.

A new test `TestCommentsDoNotAffectParsingGas` in `go2gno_test.go` creates a full `Machine` with a `GasMeter`, parses two equivalent files (one with comments, one without), and asserts equal gas consumption.

## Test Results
- **Existing tests:** PASS
- **Edge-case tests:** skipped

## Critical (must fix)
None

## Warnings (should fix)
- [ ] `gnovm/pkg/gnolang/go2gno.go:194` — `ConsumeGas(types.Gas(commentCostFactor), "parsing")` with `commentCostFactor = 0` is a no-op method call on every comment token. Since `commentCostFactor` is a compile-time constant of `0`, the code should simply `return` without calling `ConsumeGas` at all. The current form adds unnecessary function-call overhead for every comment token while doing nothing. If the intent is to make it easy to re-enable later, a comment explaining this would help, but the `ConsumeGas(0, ...)` call serves no purpose.
- [ ] `gnovm/pkg/gnolang/go2gno.go:183-195` — Charging zero gas for comments creates a gas asymmetry: the parser performs the same CPU work scanning and emitting comment tokens as it does for code tokens, but only code tokens are charged. An attacker could craft a `.gno` file that is mostly comments (~250K comment tokens in a 1MB transaction), causing significant parser work with minimal gas cost. The transaction size limit (`MaxBlockTxBytes` ~1MB at `tm2/pkg/bft/types/params.go:22`) bounds the attack, but the asymmetry is real. Consider either: (a) charging comments at the same rate as other tokens (the CPU cost is identical), or (b) charging gas based on input byte length rather than per-token.

## Nits
- [ ] `gnovm/pkg/gnolang/go2gno_test.go:34` — The test function name `TestCommentsDoNotAffectParsingGas` is accurate but could benefit from an adversarial counterpart that verifies a file with many comments doesn't consume disproportionately less gas per byte than an equivalent-sized file of real code.

## Missing Tests
- [ ] No test for a file with a very large number of comments to verify the gas meter behavior and confirm the asymmetry is bounded.

## Suggestions
- If the policy goal is "don't penalize documentation", consider a middle ground: charge comments at the regular `tokenCostFactor` rate (same CPU work) but provide a gas rebate or discount at the *transaction* level for well-documented packages. This would close the asymmetry while still incentivizing documentation.
- Alternatively, if zero-cost comments are the desired policy, remove the `ConsumeGas(0, ...)` call entirely and just `return` early — the constant documents the intent well enough on its own.

## Questions for Author
- Was the gas asymmetry (free comment tokens vs. charged code tokens) considered? Is the transaction size limit deemed sufficient mitigation?
- Is there a reason to keep the `ConsumeGas(0, "parsing")` call rather than a plain `return`?

## Verdict
APPROVE — Correct fix with good test coverage; the gas asymmetry concern is bounded by transaction size limits and is more of a design discussion than a blocking issue.
