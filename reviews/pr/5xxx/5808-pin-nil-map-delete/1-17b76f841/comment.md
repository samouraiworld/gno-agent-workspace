# Review: PR #5808
Event: APPROVE

## Body
Tests + ADR only, no runtime change. The guard under test already merged in #5196; this PR pins its full semantics. Verified on the current head (17b76f841): all three filetests pass (including #5196's `map48.gno`), the readonly-ordering claim holds at the source, and the documented gc divergence reproduces exactly: `delete(nilMap, []int{1})` panics `hash of unhashable type` under gc (go1.26.4) but no-ops in gno, consistent with gno's pre-existing nil-map read behavior.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/claude-opus-4-8_davd-gzl.md · [↗](./claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/adr/pr5808_delete_nil_map.md:26 [↗](../../../../../.worktrees/gno-review-5808/gnovm/adr/pr5808_delete_nil_map.md#L26)
The ADR cites a `SetReadonly` method that does not exist in the codebase: the only occurrence of that name in the whole tree is this line. Readonly status is computed by `IsReadonlyBy` (a nil value falls to its `default` case and returns false), never set via a setter. The conclusion is correct, so this is a harmless over-citation. Fix: drop the `SetReadonly` clause and point at `IsReadonlyBy`'s `default → return false` instead.

*(AI Agent)*
