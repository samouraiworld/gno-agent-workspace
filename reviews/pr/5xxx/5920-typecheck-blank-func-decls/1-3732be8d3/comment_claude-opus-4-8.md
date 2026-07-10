# Review: PR [#5920](https://github.com/gnolang/gno/pull/5920)
Posted: https://github.com/gnolang/gno/pull/5920#pullrequestreview-4672531764
Event: APPROVE

## Body
Verified on 3732be8d3.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5920-typecheck-blank-func-decls/1-3732be8d3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/gotypecheck.go:544 [↗](../../../../../.worktrees/gno-review-5920/gnovm/pkg/gnolang/gotypecheck.go#L544) [posted](https://github.com/gnolang/gno/pull/5920#discussion_r3559843622)
Nit: the comment says the loop ignores methods and init functions, but it now also skips blank funcs.
