# Review: PR [#5732](https://github.com/gnolang/gno/pull/5732)
Posted: https://github.com/gnolang/gno/pull/5732#pullrequestreview-4653547276
Event: APPROVE

## Body
LGTM on 6c5d74b52. Verified Go message parity, cross-tx persistence of the new `.runtimeError` type, and that the `recover().(string)` → ok=false change has zero in-tree blast radius.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5732-typedruntimeerror-runtime-errors/2-b6b3e5d42/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/tests/files/recover26.gno:1 [↗](../../../../../.worktrees/gno-review-5732/gnovm/tests/files/recover26.gno#L1) [posted](https://github.com/gnolang/gno/pull/5732#discussion_r3543430848)
Optional: no test covers a recovered `.runtimeError` stored in realm state and read back in a later block (recover26.gno stays within one VM run). It's now a plain string-based value, so a txtar under `gno.land/pkg/integration/testdata/` would cheaply lock the serialize/reload path.

