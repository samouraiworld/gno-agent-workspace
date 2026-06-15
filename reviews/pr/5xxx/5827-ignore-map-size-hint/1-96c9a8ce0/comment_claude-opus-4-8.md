# Review: PR #5827
Event: APPROVE

## Body
Sound simplification: ignoring the hint removes the double-charge, the overflow panic, and an unmetered Go-level preallocation together. Verified on 96c9a8ce0: the filetest's `// Output:` for `make(map, -1)` and `make(map, MaxInt)` matches real Go byte for byte. Checked master: the `MaxInt` case there panics with an unrecoverable `"multiplication overflow"`, which this removes.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5827-ignore-map-size-hint/1-96c9a8ce0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/tests/files/make19.gno:1 [↗](../../../../../.worktrees/gno-review-5827/gnovm/tests/files/make19.gno#L1)
The open [#5723](https://github.com/gnolang/gno/pull/5723) also adds `gnovm/tests/files/make19.gno` with different contents, so whichever lands second hits an add/add conflict. Rename this filetest to the next free slot (`make20.gno`).
