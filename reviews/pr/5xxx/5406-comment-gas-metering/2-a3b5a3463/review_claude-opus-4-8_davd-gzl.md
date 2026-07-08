# PR [#5406](https://github.com/gnolang/gno/pull/5406): fix(gnovm): exclude comment tokens from parsing gas metering

URL: https://github.com/gnolang/gno/pull/5406
Author: notJoon | Base: master | Files: 6 | +527 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: a3b5a3463 (stale — 242 commits behind master, CONFLICTING)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5406 a3b5a3463`

Round 2. Head moved bf988dd → a3b5a3463 (patch-ids differ, real content). Since round 1: the `ConsumeGas(0, …)` no-op was replaced by a clean early return (round-1 blocker resolved), a large characterization test `gas_test.go` was added by ltzmaxwell, and three gas filetest goldens were re-cut.

**TL;DR:** Comments in a `.gno` file made the first on-chain caller of a realm pay a little extra gas. This change stops the parser from charging gas for comment tokens, so documentation no longer adds parsing cost.

**Verdict: APPROVE** — parser comment-exclusion is correct and endorsed by [ltzmaxwell](https://github.com/gnolang/gno/pull/5406#issuecomment-4206154290) and [thehowl](https://github.com/gnolang/gno/pull/5406#issuecomment-4261609216); merge is gated only on routine mechanics: rebase onto master (currently CONFLICTING, gas model recalibrated so all three goldens regenerate), a `prealloc` lint, and the known-flaky `slice_alloc` gas test. The runtime store-load cost is out of scope by maintainer consensus.

## Summary
Issue [#4919](https://github.com/gnolang/gno/issues/4919) reports that comments raise runtime gas on the first realm load in a transaction. This change skips `token.COMMENT` in the parser callback so comment tokens no longer consume parsing gas. The effect is small: for a documented function the parser charged about 8 gas that it now does not. The larger part of #4919, measured by the PR's own new test at +64 gas per package here and cited at +1458–2264 in the integration case, comes from a separate mechanism the change does not touch: comments shift line numbers, those line numbers live in each saved `FuncValue`'s `RefNode`, and amino serializes them as varints that are read back at `GasGetObject` cost on every store load. ltzmaxwell confirmed this split and accepted the store-load cost as expected for now, since the full source is persisted on-chain.

## Glossary
- RefNode: saved `FuncValue.Source` stand-in carrying only the node `Location` (path, file, `Span` line/column); line numbers are amino-serialized into every persisted realm object and read back, at gas cost, on each store load.
- amino: gno's deterministic serialization codec; encodes line numbers as zig-zag varints, so a larger line number can cost an extra byte.
- persist-copy: `copyValueWithRefs`, the ref-collapsed copy that amino-marshals to the store; exercised by the new test to measure per-object bytes.
- filetest: a `gnovm/tests/files/*.gno` file run by the VM and asserted against golden directives, here `// Gas:`.
- gas: metered CPU/memory cost; consensus-relevant, so any change to it is a behavior change.

## Fix
Before, the parser callback charged `tokenCostFactor + nestLev*nestingCostFactor` for every token including comments. After, `token.COMMENT` returns early with no charge at [`go2gno.go:192-198`](https://github.com/gnolang/gno/blob/a3b5a3463/gnovm/pkg/gnolang/go2gno.go#L192-L198) · [↗](../../../../../.worktrees/gno-review-5406/gnovm/pkg/gnolang/go2gno.go#L192). The `commentCostFactor` constant from round 1 is gone; this is the cleaner shape round 1 asked for. Verified on a3b5a3463: removing the early return makes the small example's parsing gas diverge (31 with comments vs 31 without becomes 39 vs 31), and `TestCommentsDoNotAffectParsingGas` fails, so the early return is what closes the parsing-gas gap.

## Critical (must fix)
None.

## Warnings (should fix)
None. The two substantive concerns a reviewer would raise here were both raised and settled by maintainers:

- **[comments still cost runtime gas on load]** [@ltzmaxwell](https://github.com/gnolang/gno/pull/5406#issuecomment-4206154290) — the fix removes parsing gas but not the `RefNode` line-number serialization that dominates #4919. ltzmaxwell added [`gas_test.go`](https://github.com/gnolang/gno/blob/a3b5a3463/gnovm/pkg/gnolang/gas_test.go#L48) · [↗](../../../../../.worktrees/gno-review-5406/gnovm/pkg/gnolang/gas_test.go#L48) to document it and accepted it as expected while full source is persisted on-chain; thehowl points to a future addpkg storage deposit as the real remedy. Not blocking.
- **[exempting only parsing gas is an arbitrary line]** [@ltzmaxwell](https://github.com/gnolang/gno/pull/5406#issuecomment-4203688787) — comments are still charged for TxSize and storage, so exempting parsing alone is inconsistent. ltzmaxwell then [withdrew this](https://github.com/gnolang/gno/pull/5406#issuecomment-4206154290) ("I don't think we need to change anything there") and thehowl agreed per-token gas is reasonable as-is. Settled.

## Nits
- [`gnovm/pkg/gnolang/gas_test.go:309`](https://github.com/gnolang/gno/blob/a3b5a3463/gnovm/pkg/gnolang/gas_test.go#L309) · [↗](../../../../../.worktrees/gno-review-5406/gnovm/pkg/gnolang/gas_test.go#L309) — `funcNames` is grown in a loop with no preallocated capacity; `golangci-lint` (`prealloc`) fails CI on it. This is the `main / lint` red check. Give it `make([]string, 0, len(results["commentno"].funcs))`.

## Missing Tests
None. `TestCommentsDoNotAffectParsingGas` guards the parser change (fails on revert), and `gas_test.go` characterizes the store-load overhead.

## Suggestions
- [`gnovm/pkg/gnolang/gas_test.go:48`](https://github.com/gnolang/gno/blob/a3b5a3463/gnovm/pkg/gnolang/gas_test.go#L48) · [↗](../../../../../.worktrees/gno-review-5406/gnovm/pkg/gnolang/gas_test.go#L48) — `TestCommentGasRuntimeOverhead` asserts the store-load overhead still exists (`commenthi` > `commentno`). It is a characterization of unfixed behavior, so a later change that strips line numbers or uses relative spans will flip these assertions and require an update. Worth a one-line note in the test that it documents current, intentionally-unfixed behavior.

## Open questions
- CI `main / test` is red on [`gnovm/tests/files/gas/slice_alloc.gno`](https://github.com/gnolang/gno/blob/a3b5a3463/gnovm/tests/files/gas/slice_alloc.gno#L14) · [↗](../../../../../.worktrees/gno-review-5406/gnovm/tests/files/gas/slice_alloc.gno#L14): the golden `500003097` holds in an isolated run but the full parallel `TestFiles` suite computes `500003081` (both on the same toolchain, so ordering/parallelism, not Go version). This is the flaky store-load gas test [@omarsy](https://github.com/gnolang/gno/pull/5406#issuecomment-4169338309) flagged, tracked in [#5436](https://github.com/gnolang/gno/pull/5436); `const` and `nested_alloc` are stable in-suite. Not a defect this PR introduces. Left out of comment.md as a known-flaky rerun item.
- The branch is 242 commits (~3 months) behind master and CONFLICTING. Master recalibrated the gas model, so the three goldens are now entirely different there (`const` 2343, `slice_alloc` 70970781, `nested_alloc` 8559690088 vs this branch's 2964 / 500003097 / 13273859). A rebase regenerates all three. Routine; noted here, not posted.
