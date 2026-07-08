# Review: PR [#5406](https://github.com/gnolang/gno/pull/5406)
Event: APPROVE

## Body
Two non-code blockers remain before merge. `golangci-lint`'s prealloc check fails at [gas_test.go:309](https://github.com/gnolang/gno/blob/a3b5a3463/gnovm/pkg/gnolang/gas_test.go#L309). The [slice_alloc](https://github.com/gnolang/gno/blob/a3b5a3463/gnovm/tests/files/gas/slice_alloc.gno#L14) gas failure is the flaky gas test tracked in [#5436](https://github.com/gnolang/gno/pull/5436), not this change. The branch also conflicts with [master](https://github.com/gnolang/gno/tree/master), which recalibrated the three gas goldens, so a rebase regenerates them.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5406-comment-gas-metering/2-a3b5a3463/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
