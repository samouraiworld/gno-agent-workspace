# Review: PR [#5967](https://github.com/gnolang/gno/pull/5967)
Event: APPROVE

## Body
[AI bot]

Verified on 99dc36db5: booted gnodev from this branch and ran `gnokey maketx call -gasprofile` against a package the node had never loaded, with no prior `.app/simulate`. It returns a full profile tree naming `foo20.Faucet` with the `grc20` frames beneath it, and the proxy log carries no `unhandled` line for the query.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5967-source-level-gas-profiler/2-99dc36db5/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
