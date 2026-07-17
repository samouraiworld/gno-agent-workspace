# Review: PR [#5967](https://github.com/gnolang/gno/pull/5967)
Event: APPROVE

## Body
Verified on aeb88c9eb. Ran the full `gno test -gasprofile` CLI on a real example package and opened the result in `go tool pprof`. `-top`, `-traces`, and per-line `-list` all resolve against the emitted profile.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5967-source-level-gas-profiler/1-aeb88c9eb/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/crypto/keys/client/maketx.go:385 [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/crypto/keys/client/maketx.go#L385)
Suggestion: `-profile` together with `-broadcast` writes the profile and returns before broadcasting, exiting 0, so against a dev node the intended transaction is dropped. `Validate` rejects no such combination. Reject both flags together, or note on stderr that `-broadcast` is ignored under `-profile`.
