# Review: PR #5775
Event: APPROVE

## Body
Looks good. Verified on 996467249: the local backend builds and runs the scenarios end-to-end, and the halt/reset assertions abort the scenario when a local node crashes rather than passing it, so the process-based backend introduces no false-green.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5775-local-runtime-backend/1-996467249/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## misc/val-scenarios/lib/scenario.sh:882-893 [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L882)
Under the local backend a node that dies at startup isn't detected: `wait_for_rpc` polls `/status` for the full 120s, then fails with a generic timeout. Have it check the node PID each iteration and fail immediately with the log path.

## misc/val-scenarios/lib/scenario.sh:20 [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L20)
`RUNTIME` isn't validated, so any value other than exactly `local` runs the docker path: `RUNTIME=lcoal` silently gives you docker, not the local run you asked for. Reject unknown values with a one-line guard after the default.
