# Review: PR [#5882](https://github.com/gnolang/gno/pull/5882)
Event: APPROVE

## Body
Looks good. Verified on e98021315: pointing the delete builtin's `DidUpdate` at the argument key's object instead of the stored key's removes the `d[...:8](-213)` reclaim from the zrealm_map5 finalize golden, so this change is what deletes the stored key. A struct-keyed delete reclaims its key the same way, emitting `d[...:8](-252)`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
