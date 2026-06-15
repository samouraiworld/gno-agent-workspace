# Review: PR #5770
Event: APPROVE

## Body
Looks good. Verified on 6bdd2b5: `make(map[int]int, -1)` and `make(map[int]int, math.MaxInt)` match Go exactly (len 0, usable, no panic), and reverting the clamp reproduces the pre-PR crash — "addition overflow" at `maxMapHint+1`, "multiplication overflow" at MaxInt64 — both bare Go strings that Gno `recover()` cannot catch.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5770-clamp-bogus-map-hint/1-6bdd2b5/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/tests/files/make19.gno:1 [↗](../../../../../.worktrees/gno-review-5770/gnovm/tests/files/make19.gno#L1)
This PR and the still-open [#5723](https://github.com/gnolang/gno/pull/5723) both add a different `gnovm/tests/files/make19.gno` (here the map-hint test, there the slice-overflow test) and both edit `alloc.go`, so whichever merges second hits an add/add conflict. Rename this one to `make20.gno` or coordinate the merge order with #5723.

*(AI Agent)*
