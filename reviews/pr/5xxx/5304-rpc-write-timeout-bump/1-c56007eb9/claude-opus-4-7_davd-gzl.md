# PR #5304: fix(rpc): bump default write timeout

URL: https://github.com/gnolang/gno/pull/5304
Author: thehowl | Base: master | Files: 1 | +1 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7

Verdict: APPROVE — one-line default bump from 10s to 30s that fixes a real production cutoff on `/genesis` for mainnet-sized files; no callers regress and the existing `node.go` auto-bump path still works.

## Summary

The TM2 RPC server cuts off all HTTP responses at 10s wall-clock (`WriteTimeout` on the shared `http.Server`), which truncates `/genesis` on networks whose genesis JSON takes longer than that to serialize — the author observed "exactly 11 seconds" on gnoland1 because [`node.go:755-757`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/node/node.go#L755-L757) raises `WriteTimeout` to `TimeoutBroadcastTxCommit + 1s` (10s + 1s default). Raising the package default to 30s gives one full block-budget worth of headroom for slow endpoints (full genesis, large `/block_results`, `/tx_search`) without removing the slow-client guard entirely. No public `RPCConfig` knob exposes `WriteTimeout` directly — the only operator-controlled lever is `timeout_broadcast_tx_commit`, so this default is what nearly every node will run.

## Glossary

- `DefaultConfig` — [`tm2/pkg/bft/rpc/lib/server/http_server.go:39`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/rpc/lib/server/http_server.go#L39); only constructor of the rpcserver `Config`. Single caller: `node.go:748`.
- `TimeoutBroadcastTxCommit` — operator-tunable RPC config (default 10s); used by `node.go` to floor `WriteTimeout`.
- `defaultWSWriteWait` — [`handlers.go:434`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/rpc/lib/server/handlers.go#L434); websocket-layer 10s write deadline, applied after `Hijack` takes the conn out of `http.Server`'s timeout regime.

## Fix

Before: [`http_server.go:43`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/rpc/lib/server/http_server.go#L43) set `WriteTimeout: 10 * time.Second`, which combined with the `+1s` floor in [`node.go:755-757`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/node/node.go#L755-L757) gave every node 11s to flush any HTTP response — too short for the gnoland1 `/genesis` payload. After: default is 30s, so the floor in `node.go` only kicks in if an operator pushes `timeout_broadcast_tx_commit` above 30s. Websocket connections are unaffected because `Hijack` removes the conn from `http.Server`'s deadline path and the WS layer enforces its own [`defaultWSWriteWait`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/rpc/lib/server/handlers.go#L434).

## Critical (must fix)

None.

## Warnings (should fix)

None — the change is small, well-targeted, and doesn't break any existing override path.

## Nits

- [`tm2/pkg/bft/rpc/lib/server/http_server.go:43`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/rpc/lib/server/http_server.go#L43) — commit message and PR body don't say *why* 30s (vs 20s or 60s). Worth a one-line code comment like `// 30s: full block-production budget; large /genesis responses can exceed 10s`. Future readers tweaking this number have nothing to anchor against.
- [`tm2/pkg/bft/rpc/config/config.go:63`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/rpc/config/config.go#L63) — the `timeout_broadcast_tx_commit` doc comment still says "Using a value larger than 10s will result in increasing the global HTTP write timeout". With the new default that threshold is 30s. Worth updating in the same PR so the WARNING stays accurate.

## Missing Tests

- [`tm2/pkg/bft/rpc/lib/server/http_server_test.go`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/rpc/lib/server/http_server_test.go) — no regression test asserts the default value. A one-line `assert.Equal(t, 30*time.Second, DefaultConfig().WriteTimeout)` would catch accidental future flips. Low priority; the constant is its own contract.

## Suggestions

- `tm2/pkg/bft/rpc/config/config.go` — consider exposing `WriteTimeout` as a first-class `RPCConfig` field (mirroring `MaxBodyBytes`, `MaxHeaderBytes`) in a follow-up, so operators with slow disks or chunky custom endpoints don't have to abuse `timeout_broadcast_tx_commit` to tune it. Out of scope for this PR.
  <details><summary>details</summary>

  Today the only path to a `WriteTimeout` above the default is the auto-bump in [`node.go:755-757`](../../../../../.worktrees/gno-review-5304/tm2/pkg/bft/node/node.go#L755-L757), which couples two semantically distinct settings: how long `/broadcast_tx_commit` waits for a commit, and how long any other endpoint has to flush bytes. Decoupling them is cleaner than chasing the default upward each time a new slow endpoint appears.
  </details>

## Questions for Author

- Did you measure how long `/genesis` actually takes on gnoland1 (closer to 11s or closer to 30s)? If it's near the new ceiling we should pick a wider margin or revisit the chunked-genesis approach upstream Tendermint took (`/genesis_chunked`).
