# Review: PR [#5963](https://github.com/gnolang/gno/pull/5963)
Event: APPROVE

## Body
Verified on a94c90986. Re-adding the deleted second truncation reproduces the slice bounds out of range [:-1] crash. Output matches go run byte-for-byte across for, range, and switch-clause frame crossings for 3- to 6-deep nesting.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5963-gotojump-nested-loop-crash/1-a94c9098/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/machine.go:2444-2449 [↗](../../../../../.worktrees/gno-review-5963/gnovm/pkg/gnolang/machine.go#L2444)
Suggestion: worth noting in the comment that the [`GOTO` handler](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/op_exec.go#L715) resets `m.Stmts` right after, which is why truncating to `fr.NumStmts` alone is enough.
