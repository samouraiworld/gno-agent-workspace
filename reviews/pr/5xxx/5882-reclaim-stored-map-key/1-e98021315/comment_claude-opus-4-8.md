# Review: PR [#5882](https://github.com/gnolang/gno/pull/5882)
Posted: https://github.com/gnolang/gno/pull/5882#pullrequestreview-4633934290
Event: APPROVE

## Body
Looks good. Verified on e98021315: reverting to mark the argument key instead of the stored key drops the `d[...:8](-213)` reclaim from the finalize golden, so this frees the leaked key.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values.go:828-840 [↗](../../../../../.worktrees/gno-review-5882/gnovm/pkg/gnolang/values.go#L828) [posted](https://github.com/gnolang/gno/pull/5882#discussion_r3527274167)
DeleteForKey has the machine and could mark the removed key itself instead of returning it for the builtin. Is that deliberate, to keep `DidUpdate` out of `MapValue`?
