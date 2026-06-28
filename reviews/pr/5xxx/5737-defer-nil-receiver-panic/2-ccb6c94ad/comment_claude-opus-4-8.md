# Review: PR #5737
Event: APPROVE

## Body
Verified on ccb6c94ad. The gno output matches a side-by-side Go run on every dispatch axis, including across a persist/reload: receiver-snapshot timing, nil-panic timing, embedded promotion, field re-read, dynamic re-dispatch. `defer pt.M()` on a concrete nil panics eagerly like Go. A cyclic embedded interface terminates with the fatal panic instead of hanging, even on an ungas-metered run.

One open item, not a code defect. `OpCPULazyBoundResolve` and the re-fit `OpCPUSelectorInterface` are ratio-scaled placeholders. They set consensus gas for every interface method call, so they need measuring on the reference hardware before the fork activates.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/2-ccb6c94ad/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
