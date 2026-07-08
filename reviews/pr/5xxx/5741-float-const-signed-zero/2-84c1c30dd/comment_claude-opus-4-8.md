# Review: PR [#5741](https://github.com/gnolang/gno/pull/5741)
Event: APPROVE

## Body
Verified on 84c1c30dd: removing the line that rounds then rejects on `±Inf` magnitude makes `float32(-3.5e38)` narrow to `-Inf` again, so the negative-overflow rejection is load-bearing. A side-by-side Go run agrees that constant folds like `-zt` give `+0` while runtime `-v` keeps `-0`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5741-float-const-signed-zero/2-84c1c30dd/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
