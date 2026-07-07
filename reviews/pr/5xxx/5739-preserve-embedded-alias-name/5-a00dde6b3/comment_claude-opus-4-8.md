# Review: PR #5739
Event: COMMENT

## Body
Deep re-review at a00dde6b3 across correctness, consensus, persisted-state, and coverage lenses.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/5-a00dde6b3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/types.go:1062-1077 [↗](../../../../../.worktrees/gno-review-5739/gnovm/pkg/gnolang/types.go#L1062)
An interface that embeds a foreign sealed interface and also declares its own same-named unexported method resolves a selection of that method by source order. `interface{ ifaceext.Sec; sec() int }` rejects `x.sec()` from main, while listing the own `sec()` first compiles; Go binds `x.sec()` to main's own `sec` either way. The trigger is degenerate and the direction is safe, so this is minor: selection could prefer the caller-package entry to match Go.

<details><summary>confirmed behaviorally</summary>

Red at a00dde6b3 (`cannot access interface {sec func() int; sec func() int}.sec from main`); a go/build oracle accepts both orders. Test [`iface_embed_sel_order.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/5-a00dde6b3/tests/iface_embed_sel_order.gno) · [↗](tests/iface_embed_sel_order.gno) fails now and passes under the localized fix without regressing `iface_embed_xpkg_access`.
</details>
