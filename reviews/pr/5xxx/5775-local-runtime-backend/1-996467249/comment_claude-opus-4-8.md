# Review: PR #5775
Posted: https://github.com/gnolang/gno/pull/5775#pullrequestreview-4517219424
Event: APPROVE

## Body
Looks good. Verified on 996467249: the local backend builds and runs the scenarios end-to-end, and the halt/reset assertions abort the scenario when a local node crashes rather than passing it, so the process-based backend introduces no false-green.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5775-local-runtime-backend/1-996467249/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## misc/val-scenarios/lib/scenario.sh:961-975 [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L961) [posted](https://github.com/gnolang/gno/pull/5775#discussion_r3429484456)
Local nodes are launched as `disown`ed background processes with nothing watching them, so a node that dies at startup is noticed only when its RPC poll times out 120s later with a generic message. Have the RPC wait loop (`wait_for_rpc`) also check the node PID so a crashed process fails fast with its log path.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5775 -R gnolang/gno
cd misc/val-scenarios
# Drive the PR's own wait_for_rpc against a node whose process has crashed:
# a dead PID plus a closed RPC port is the exact state right after a startup failure.
bash <<'EOF'
source lib/scenario.sh
RUNTIME=local
NODE_RPC_PORT[deadnode]=59999      # nothing is listening here
NODE_PID[deadnode]=999999          # the crashed gnoland process, already gone
kill -0 "${NODE_PID[deadnode]}" 2>/dev/null && echo alive || echo "node process already dead"
t=$SECONDS
( wait_for_rpc deadnode 6 ) || true   # harness default is 120s; shortened to 6 for the demo
echo "wait_for_rpc burned $((SECONDS-t))s and failed without ever checking the dead PID"
EOF
```

```
node process already dead
error: rpc for deadnode did not come up within 6s
wait_for_rpc burned 6s and failed without ever checking the dead PID
```
</details>

## misc/val-scenarios/lib/scenario.sh:20 [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L20) [posted](https://github.com/gnolang/gno/pull/5775#discussion_r3429484472)
`RUNTIME` isn't validated, so any value other than exactly `local` runs the docker path: `RUNTIME=lcoal` silently gives you docker, not the local run you asked for. Reject unknown values with a one-line guard after the default.
