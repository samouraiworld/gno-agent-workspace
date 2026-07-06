# Review: PR [#5889](https://github.com/gnolang/gno/pull/5889)
Event: REQUEST_CHANGES

## Body
The loader half of the reset fix works. The `/reset` HTTP endpoint at [`app.go:475`](https://github.com/gnolang/gno/blob/197b335e1/contribs/gnodev/app.go#L475) [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/app.go#L475) is left half-reset: it calls `devNode.Reset` alone, which drops the browsed package from the loader but never clears the app's `pathManager`. A re-browse then resolves in the loader but hits the `pathManager.Save` dedup at [`app.go:408`](https://github.com/gnolang/gno/blob/197b335e1/contribs/gnodev/app.go#L408) [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/app.go#L408) and returns without redeploying, so a package loaded before the reset stays 404 for the rest of the session. Ctrl+R avoids this because it resets `pathManager` before reloading; `/reset` should too.

Verified live on 197b335e1: after `/reset` a re-browse of a package loaded before the reset returns 404 for the session, while a package first browsed after the reset loads normally. On master the same reset keeps both reachable, so the regression is on the `/reset` path added here.

Missing test: `/reset` over HTTP followed by a re-browse of a previously-loaded package. Loader-level tests pass while this end-to-end reset path is uncovered.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5889 -R gnolang/gno
( cd contribs/gnodev && go build -o /tmp/gnodev-5889 . )
GNOROOT="$PWD" /tmp/gnodev-5889 local -root "$PWD" \
  -web-listener 127.0.0.1:18888 -node-rpc-listener 127.0.0.1:18657 \
  -unsafe-api=true -log-format json >/tmp/gnodev-5889.log 2>&1 &
GNODEV_PID=$!
until curl -s -o /dev/null http://127.0.0.1:18888/; do sleep 0.5; done
B=http://127.0.0.1:18888
echo "browse r/sys/users (lazy load):"; curl -s -o /dev/null -w "  %{http_code}\n" "$B/r/sys/users"
echo "hit /reset:";                     curl -s -o /dev/null -w "  %{http_code}\n" "$B/reset"; sleep 2
echo "re-browse r/sys/users:";          curl -s -o /dev/null -w "  %{http_code}\n" "$B/r/sys/users"
echo "browse a fresh r/gnoland/pages:"; curl -s -o /dev/null -w "  %{http_code}\n" "$B/r/gnoland/pages"
kill "$GNODEV_PID"
```
```
browse r/sys/users (lazy load):
  200
hit /reset:
  200
re-browse r/sys/users:
  404
browse a fresh r/gnoland/pages:
  200
```
On master the re-browse returns 200, so the regression is on the `/reset` path added here.
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5889-gnodev-reset-loader-fixes/1-197b335e1/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnodev/pkg/dev/node.go:44-46 [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/pkg/dev/node.go#L44)
This says every reset entry point returns the package set to its initial state. That holds for the loader, but on `/reset` a package loaded before the reset can no longer be browsed back, so it overstates what `/reset` delivers today. See the Body for the repro and fix.
