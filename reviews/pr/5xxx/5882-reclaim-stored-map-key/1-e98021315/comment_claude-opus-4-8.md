# Review: PR [#5882](https://github.com/gnolang/gno/pull/5882)
Event: APPROVE

## Body
Looks good. Verified on e98021315: pointing the delete builtin's `DidUpdate` at the argument key's object instead of the stored key's removes the `d[...:8](-213)` reclaim from the zrealm_map5 finalize golden, so this change is what deletes the stored key. Delete reclaims only the map's own copy of an array or struct key: an escaped copy of the same key value stays readable after the delete, and a pointer key's shared pointee is updated rather than deleted.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
