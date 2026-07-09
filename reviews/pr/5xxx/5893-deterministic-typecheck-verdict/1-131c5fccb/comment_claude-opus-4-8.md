# Review: PR [#5893](https://github.com/gnolang/gno/pull/5893)
Event: APPROVE

## Body
Both determinism axes in the type-check path are closed: the go1.18 pin fixes the accept/reject verdict across toolchains, and the empty `TypeCheckError` sentinel keeps go/types and go/parser text out of the results hash while the full diagnostics stay on the unhashed log. Verified on 131c5fccb: removing the `GoVersion: "go1.18"` line makes this go1.26 build accept `for range 10`, which the pinned checker rejects, reproducing the build-dependent verdict this PR removes.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5893-deterministic-typecheck-verdict/1-131c5fccb/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
