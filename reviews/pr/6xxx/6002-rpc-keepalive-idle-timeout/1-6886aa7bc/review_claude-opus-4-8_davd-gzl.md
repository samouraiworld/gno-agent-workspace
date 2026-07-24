# PR [#6002](https://github.com/gnolang/gno/pull/6002): feat(tm2): add config for RPC server keep-alive idle timeout

URL: https://github.com/gnolang/gno/pull/6002
Author: aeddi | Base: master | Files: 7 | +253 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 6886aa7bc (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-6002 6886aa7bc`

**TL;DR:** A gno.land node closes any HTTP connection that has been sitting unused for 10 seconds. Load balancers keep those same connections in a pool for minutes, so when the balancer reuses one the node has already hung up and the user gets a 502. This adds a `[rpc] idle_timeout` setting so an operator can tell the node to hold connections longer, and leaves it off by default.

**Verdict: APPROVE** — plumbing, default, validation and TOML round-trip all check out; the one open concern is that the new config comment frames `max_open_connections` as the bound on the risk without saying what reaching that bound costs (1 Warning, 1 Missing test, 1 Nit, 1 Suggestion).

## Summary

The RPC server sets `ReadTimeout: 10s` and leaves `http.Server#IdleTimeout` unset, and `net/http` falls back to `ReadTimeout` for the keep-alive idle deadline, so every pooled backend connection older than 10s of inactivity is closed server-side. A proxy that pools for 60-600s then reuses a dead connection and reports a 502. The PR adds `IdleTimeout` to the rpcserver [`Config`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/lib/server/http_server.go#L32-L34) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/lib/server/http_server.go#L32-L34), sets it on the plain and TLS servers, and exposes it as `[rpc] idle_timeout`, plumbed through [`startRPC`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/node/node.go#L807) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/node/node.go#L807) alongside `max_open_connections`. The default is [zero](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/config/config.go#L115) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config.go#L115), which keeps the `net/http` fallback and so leaves every existing deployment untouched.

## Glossary

- RPC connection cap: `max_open_connections`, applied by wrapping the listener in `netutil.LimitListener`; the slot is released only when a connection closes, so a full cap blocks `Accept` instead of refusing.

## Fix

Before, the idle deadline was an accident of `ReadTimeout` and unreachable from `config.toml`. After, `RPCConfig.IdleTimeout` reaches `http.Server#IdleTimeout` on both the [plain](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/lib/server/http_server.go#L64) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/lib/server/http_server.go#L64) and [TLS](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/lib/server/http_server.go#L90) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/lib/server/http_server.go#L90) servers, with negatives rejected in [`ValidateBasic`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/config/config.go#L146-L148) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config.go#L146-L148). The load-bearing constraint is that zero must stay meaningful: it is the `net/http` fallback marker, not "no timeout", which is why the default cannot be expressed as an explicit 10s.

## Critical (must fix)

None.

## Warnings (should fix)

- **[operator reads the cap as a safety bound]** `tm2/pkg/bft/rpc/config/config.go:71-74` — the comment names `max_open_connections` as the bound on simultaneously idle connections, but reaching that cap blocks new connections for the full `idle_timeout` rather than the 10 seconds it costs today.
  <details><summary>details</summary>

  [`rpcserver.Listen`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/lib/server/http_server.go#L288-L290) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/lib/server/http_server.go#L288-L290) wraps the listener in `netutil.LimitListener` whenever `max_open_connections > 0`, and that semaphore is released only in [`limitListenerConn.Close`](https://github.com/golang/net/blob/v0.56.0/netutil/listen.go#L83-L86), so an idle keep-alive connection keeps its slot for its whole idle life. Once every slot is held, [`Accept` blocks on the semaphore send](https://github.com/golang/net/blob/v0.56.0/netutil/listen.go#L34-L45) and a new client is not refused, it waits. Measured with two slots, both held idle: a fresh client is served after 3.000s with `IdleTimeout = 3s`, and after 200ms with `IdleTimeout` unset and `ReadTimeout = 200ms`, so the wait tracks the idle timeout exactly ([repro](comment_claude-opus-4-8.md)). At the values this PR targets, 900 slots and 620s, a full pool stalls every new client for up to 10m20s instead of 10s. The comment reads as reassurance about the capped case and warns only about the uncapped one, which is the cheaper of the two. Fix: say what reaching the cap costs.
  </details>

## Nits

- **[non-ASCII in a file operators edit]** `tm2/pkg/bft/rpc/config/config.go:68` — the em-dash makes line 244 of a freshly generated `config.toml` the file's only non-ASCII line.
  <details><summary>details</summary>

  `grep -nP '[^\x00-\x7F]'` over a `gnoland config init` output on this branch returns exactly one line, the [new `idle_timeout` comment](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/config/config.go#L68) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config.go#L68); `git show origin/master:tm2/pkg/bft/rpc/config/config.go | grep -c '—'` returns 0, so the byte arrives with this PR. Purely cosmetic, and no enabled linter covers it: [`.github/golangci.yml`](https://github.com/gnolang/gno/blob/6886aa7bc/.github/golangci.yml#L13) · [↗](../../../../../.worktrees/gno-review-6002/.github/golangci.yml#L13) runs `default: none` with an explicit enable list holding no style linter. Not posted, no change needed.
  </details>

## Missing Tests

- **[config.toml comment can go stale unnoticed]** `tm2/pkg/bft/rpc/config/config_test.go:16-19` — nothing pins the 10s read timeout the new comment names, so changing `rpcserver.DefaultConfig().ReadTimeout` would silently make the `idle_timeout` comment in every generated `config.toml` wrong.
  <details><summary>details</summary>

  The comment at [`config.go:66-67`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/config/config.go#L66-L67) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config.go#L66-L67) states the zero-value fallback is "currently 10s", a fact that lives in another package at [`http_server.go:46`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/lib/server/http_server.go#L46) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/lib/server/http_server.go#L46) and is asserted nowhere: no test in `tm2/pkg/bft/rpc/lib/server` or `tm2/pkg/bft/rpc/config` reads `DefaultConfig().ReadTimeout`. [`TestDefaultRPCConfig`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/config/config_test.go#L11-L20) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config_test.go#L11-L20) pins the zero default but not the value zero resolves to. Confirmed the assertion belongs there: `tm2/pkg/bft/rpc/config` can import `tm2/pkg/bft/rpc/lib/server` without a cycle, and the one-line test compiles and passes in that package. Fix: assert the fallback value the comment quotes.
  </details>

## Suggestions

- **[wrong unit is silent and inverts the intent]** `tm2/pkg/bft/rpc/config/config.go:69-70` — the comment tells operators to set a value larger than their proxy's idle timeout but never shows the written form, and an unquoted `idle_timeout = 620` in `config.toml` loads as 620 nanoseconds.
  <details><summary>details</summary>

  Confirmed behaviorally: hand-editing a generated `config.toml` to `idle_timeout = 620` and running `gnoland config get rpc.idle_timeout` prints `620`, not `620000000000`. Because 620ns is non-zero it displaces the `net/http` fallback, so keep-alive connections are closed on the spot, the opposite of what an operator following this comment wants, and nothing in the load path complains. The same holds for `timeout_broadcast_tx_commit = 10` on the merge base, so this is a property of every duration key rather than something the diff introduces; it matters more here because `idle_timeout` is the one duration whose whole purpose is being hand-set by an operator, and because its wrong-unit failure mode is silent. Fix: put the quoted form in the comment.
  </details>

## Verified

- Zero default is behavior-preserving end to end, not just at the struct: a `config.toml` generated from this branch writes `idle_timeout = "0s"`, and loading it back yields a zero `RPCConfig.IdleTimeout`. An existing deployment's file has no `idle_timeout` key at all, and `loadConfigFile` applies the file over `DefaultConfig()`, so it also lands on zero.
- Negative values are rejected on both operator paths: `gnoland config set rpc.idle_timeout -- -1s` exits 1 with `idle_timeout can't be negative` and leaves the file unchanged, and hand-editing the file to `idle_timeout = "-5s"` makes config loading refuse the file. `loadConfigFile` calls `ValidateBasic` unconditionally, so there is no load path that reaches `http.Server` with a negative deadline.
- Round-trip through the CLI holds: `config set rpc.idle_timeout 620s` writes `idle_timeout = "10m20s"`, and `config get` reads back `620000000000`, the same nanosecond form `timeout_broadcast_tx_commit` prints.
- The connection-slot interaction is measured, not argued: with `max_open_connections = 2` and both slots held by idle keep-alive connections, a new client's first response arrives after 3.000s when `IdleTimeout = 3s` and after 200ms when it is unset with `ReadTimeout = 200ms`. Harness: [`tests/idle_slot_occupancy_test.go`](tests/idle_slot_occupancy_test.go).
- The master merge at the head carries no authored content: `git show 6886aa7bc --cc` prints no hunks, and none of the PR's 7 files differ between 1658c29a0 and the merge.
- Green at 6886aa7bc: the four new tests in `tm2/pkg/bft/rpc/lib/server` under `-race -count=20` with `GOMAXPROCS=2` and 16 busy-loop load generators, plus `tm2/pkg/bft/rpc/config` and `TestConfig_(Get|Set)_RPC` in `gno.land/cmd/gnoland`.

## Open questions

- [`gnoweb`](https://github.com/gnolang/gno/blob/6886aa7bc/gno.land/cmd/gnoweb/main.go#L289-L291) · [↗](../../../../../.worktrees/gno-review-6002/gno.land/cmd/gnoweb/main.go#L289-L291) sets `IdleTimeout` from its own `-timeout` flag, which [defaults to one minute](https://github.com/gnolang/gno/blob/6886aa7bc/gno.land/cmd/gnoweb/main.go#L69) · [↗](../../../../../.worktrees/gno-review-6002/gno.land/cmd/gnoweb/main.go#L69). Behind the same 300s-pooling proxy that motivates this PR, gnoweb has the identical 502 shape unless operators raise it. Not posted: it is a separate binary with a knob that already exists, so nothing about it changes what this PR should do.
- The checked-in deployment configs under `misc/deployments/` carry no `idle_timeout` key, so they stay on the fallback. Not posted: whether those files still drive live networks is an infra question, not a code one.
