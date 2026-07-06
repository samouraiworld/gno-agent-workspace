# Review: PR [#5894](https://github.com/gnolang/gno/pull/5894)
Event: APPROVE

## Body
Verified on d0bdb5049 with two checks the suite does not run. Running this branch's [restart_local_type.txtar](https://github.com/gnolang/gno/blob/d0bdb5049/gno.land/pkg/integration/testdata/restart_local_type.txtar) · [↗](../../../../../.worktrees/gno-review-5894/gno.land/pkg/integration/testdata/restart_local_type.txtar) against master fails the post-restart read with `unexpected type with id ...lt2[...].S`. Neutralizing the `localTypeSaver` call and running `zrealm_localtype0.gno` under `-tags debugAssert` fires the save-time guard `dangling function-local type ref ... in persisted value`, which CI does not exercise.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5894-persist-function-local-types/1-d0bdb5049/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
