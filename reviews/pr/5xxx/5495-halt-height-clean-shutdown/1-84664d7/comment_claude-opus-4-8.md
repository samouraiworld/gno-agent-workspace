# Review: PR #5495
Event: COMMENT

## Body
The code is sound; the only blocker is the open exit-0-vs-full-zombie direction. One thing that may help unstick it: the cold-restart half (the typed error plus the `doHandshake` recover that turns it into a clean exit-1 error) is correct and self-contained, so it can ship even if the warm-halt `state.go` change is dropped in favor of extending #5334.

On 84664d7, the cold-restart path is verified by inspection: the only ABCI client is the in-process `localClient`, nothing in the `BeginBlock` → `doHandshake` replay path recovers, and all three sites match the halt error by value, so the typed panic reaches the new recover and converts to a returned error.

Two notes that don't need a thread: the warm-halt exit-0 completes through gnoland's `signal.NotifyContext` SIGTERM handler ([root.go:16-21](https://github.com/gnolang/gno/blob/84664d7/gno.land/cmd/gnoland/root.go#L16-L21)), not `osm.TrapSignal` (which gnoland never registers), so the self-SIGTERM funnels into the existing graceful-stop path rather than the mechanism the description names. And the red TM2 check is the flaky `TestWALCrash/empty_block` timeout, unrelated to this diff.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5495-halt-height-clean-shutdown/1-84664d7/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/types/errors.go:43-51 [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/types/errors.go#L43)
The halt error is matched by value at three sites and nothing locks that. If a later edit makes the panic `&HaltHeightReachedError{}`, both recover assertions silently stop matching and the node reverts to the old zombie / exit-2 behavior. A one-line comment that it is recovered by value, not pointer, would prevent that.

## tm2/pkg/bft/node/node.go:256-265 [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/node/node.go#L256)
The cold-restart recover that converts the halt panic into a clean error has no test, though `doHandshake` is callable white-box. A test that injects an `AppConns` whose `BeginBlock` panics with the typed error during replay would lock both the error message and that a non-halt panic still propagates.

## tm2/pkg/bft/node/node.go:263 [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/node/node.go#L263)
Re-raising a non-halt replay panic with `panic(r)` resets the stack to this line, so a genuine replay or DB-corruption crash on restart points here instead of its origin. Capture the stack at recover time (`panic(fmt.Sprintf("%v\n%s", r, debug.Stack()))`) before re-raising.

## tm2/pkg/bft/consensus/state.go:616 [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/consensus/state.go#L616)
`osm.Kill()`'s error is dropped here, unlike the sibling kill at state.go:1389 that logs on failure. If the self-SIGTERM fails to send, the node neither exits nor leaves a trace, silently contradicting the clean-shutdown intent. Log it on failure.
