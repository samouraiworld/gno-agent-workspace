# Claims: gnolang/gno#5258 round 4, b0476e4df, claude-opus-5-5, quick review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`node_test.go:119`](https://github.com/gnolang/gno/blob/b0476e4df/tm2/pkg/bft/node/node_test.go#L119) | 1 | Missing test | CONFIRMED | same | posts |
| [`middleware.go:21`](https://github.com/gnolang/gno/blob/b0476e4df/contribs/gnodev/pkg/emitter/middleware.go#L21) | 4 | Nit | CONFIRMED | same | posts |
| [`config.go:28`](https://github.com/gnolang/gno/blob/b0476e4df/tm2/pkg/bft/rpc/config/config.go#L28) | 5 | Nit | CONFIRMED | same | posts |
| [`hotreload.js:8`](https://github.com/gnolang/gno/blob/b0476e4df/contribs/gnodev/pkg/emitter/static/hotreload.js#L8) | 2 | Suggestion | CONFIRMED | same | posts |
| [`handlers.go:1149`](https://github.com/gnolang/gno/blob/b0476e4df/tm2/pkg/bft/rpc/lib/server/handlers.go#L1149) | 3 | Suggestion | CONFIRMED | same | posts |

### 1 · Missing test · CONFIRMED · `tm2/pkg/bft/node/node_test.go:119`

No node-level test pins cors_allowed_origins = [] as same-origin on /websocket, so startRPC can hand an empty list to NewWebsocketManager as ["*"] with CI green

- Check: go test -overlay with node.go:951 mapping an empty list to []string{"*"}: -run 'TestNodeRPC\|TestWebsocketManagerCheckOrigin' ./tm2/pkg/bft/node ./tm2/pkg/bft/rpc/lib/server; then tests/judge-1-empty-list_test.go at head and under the same mutation
- Observed: mutation: ok tm2/pkg/bft/node, ok tm2/pkg/bft/rpc/lib/server; judge test at head: ok; under mutation: FAIL TestJudgeNodeRPCWebsocketOriginEmptyList, expected: 403, actual: 101, origin "https://evil.test"
- Test: [`tests/judge-1-empty-list_test.go`](tests/judge-1-empty-list_test.go)

### 4 · Nit · CONFIRMED · `contribs/gnodev/pkg/emitter/middleware.go:21`

muRemote is read-locked on every request and write-locked nowhere, and is still named after the remote field this round renamed to path

- Check: grep -rn muRemote contribs/gnodev; git log --oneline -S'muRemote.Lock' -- contribs/gnodev
- Observed: middleware.go:21 muRemote sync.RWMutex; :63 m.muRemote.RLock(); :64 defer m.muRemote.RUnlock(); git log -S'muRemote.Lock': no output

### 5 · Nit · CONFIRMED · `tm2/pkg/bft/rpc/config/config.go:28`

The cors_allowed_origins comment calls [] the default and names only cross-domain HTTP, while the default is ["*"] and the list now also decides /websocket upgrades

- Check: sed -n '28p;108p;170,172p' tm2/pkg/bft/rpc/config/config.go
- Observed: config.go:28 comment text quoted above; config.go:108 CORSAllowedOrigins: []string{"*"}

### 2 · Suggestion · CONFIRMED · `contribs/gnodev/pkg/emitter/static/hotreload.js:8`

The injected script dials 'ws://' + window.location.host, which Chromium refuses with a SecurityError on any https page whose host is not loopback, so gnodev behind a TLS front never hot-reloads

- Check: chromium --headless --ignore-certificate-errors --host-resolver-rules='MAP devhost.test 127.0.0.1' --dump-dom on tests/judge-2-mixed-content/page.html served over TLS by serve.py, at https://devhost.test:18443 and https://localhost:18443
- Observed: https://devhost.test:18443: "ws:// SecurityError: Failed to construct 'WebSocket': An insecure WebSocket connection may not be initiated from a page loaded over HTTPS." / "wss:// constructed"; https://localhost:18443: "ws:// constructed" / "wss:// constructed"
- Test: [`tests/judge-2-mixed-content/page.html`](tests/judge-2-mixed-content/page.html)

### 3 · Suggestion · CONFIRMED · `tm2/pkg/bft/rpc/lib/server/handlers.go:1149`

A client sending Origin http://<the node's own host> upgrades under [] and gets 403 once the list is narrowed, so narrowing the list revokes it

- Check: tests/judge-3-own-host-origin_test.go overlaid into tm2/pkg/bft/rpc/lib/server, go test -v -run TestJudgeOwnHostOrigin at b0476e4df and at dd05df75b
- Observed: b0476e4df: "empty list: Origin http://<own host> -> 101", "narrowed list: Origin http://<own host> -> 403"; dd05df75b: identical
- Test: [`tests/judge-3-own-host-origin_test.go`](tests/judge-3-own-host-origin_test.go)

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`node.go:951`](https://github.com/gnolang/gno/blob/b0476e4df/tm2/pkg/bft/node/node.go#L951) | removed behaviour | dropping the IsCorsEnabled guard could change what reaches NewWebsocketManager | grep config.go: 'return len(cfg.CORSAllowedOrigins) != 0' (IsCorsEnabled); handlers.go NewWebsocketManager builds the validator only under 'if len(allowedOrigins) > 0', so [] and nil both leave gorilla's same-origin check, as the removed guard did |
| [`server.go:30`](https://github.com/gnolang/gno/blob/b0476e4df/contribs/gnodev/pkg/emitter/server.go#L30) | reach | gnobro's dev client could be refused by the emitter's same-origin check | contribs/gnobro/main.go:267 'devcl.Run(ctx, devpoint, nil)' and client_dev.go:90 'dialer.DialContext(ctx, addr, header)' with that nil header: no Origin sent, which gorilla's checkSameOrigin accepts |
| [`hotreload.js:8`](https://github.com/gnolang/gno/blob/b0476e4df/contribs/gnodev/pkg/emitter/static/hotreload.js#L8) | reach | a Content-Security-Policy connect-src on gnoweb pages could block the page-relative socket | grep -rn Security-Policy gno.land contribs (non-test): only gno.land/cmd/gnoweb/main.go:531 sets it; gnodev's setupGnoWebServer path sets no CSP |
| [`server_test.go:349`](https://github.com/gnolang/gno/blob/b0476e4df/contribs/gnodev/pkg/emitter/server_test.go#L349) | catalog: global mutable state & concurrency | the added t.Parallel tests could race on shared state | go test -race -count=1 ./pkg/emitter: ok 1.021s; go test -race -count=1 -run 'TestWebsocketManagerCheckOrigin\|TestNodeRPCWebsocketOrigin\|TestNodeRPCBatchLimit' ./tm2/pkg/bft/rpc/lib/server ./tm2/pkg/bft/node: ok 1.018s, ok 1.055s |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Missing test | tm2/pkg/bft/node/node_test.go:119 | go test -overlay with node.go:951 mapping an empty list to []string{"*"}: -run 'TestNodeRPC\|TestWebsocketManagerCheckOrigin' ./tm2/pkg/bft/node ./tm2/pkg/bft/rpc/lib/server; then tests/judge-1-empty-list_test.go at head and under the same mutation | cold |  |
| 2 | CONFIRMED | Suggestion | contribs/gnodev/pkg/emitter/static/hotreload.js:8 | chromium --headless --ignore-certificate-errors --host-resolver-rules='MAP devhost.test 127.0.0.1' --dump-dom on tests/judge-2-mixed-content/page.html served over TLS by serve.py, at https://devhost.test:18443 and https://localhost:18443 | warm |  |
| 3 | CONFIRMED | Suggestion | tm2/pkg/bft/rpc/lib/server/handlers.go:1149 | tests/judge-3-own-host-origin_test.go overlaid into tm2/pkg/bft/rpc/lib/server, go test -v -run TestJudgeOwnHostOrigin at b0476e4df and at dd05df75b | hot |  |
| 4 | CONFIRMED | Nit | contribs/gnodev/pkg/emitter/middleware.go:21 | grep -rn muRemote contribs/gnodev; git log --oneline -S'muRemote.Lock' -- contribs/gnodev | warm |  |
| 5 | CONFIRMED | Nit | tm2/pkg/bft/rpc/config/config.go:28 | sed -n '28p;108p;170,172p' tm2/pkg/bft/rpc/config/config.go |  |  |

</details>

Hit rate per tier, from the record: hot 1/1 confirmed over 2 files, warm 2/2 confirmed over 3 files, cold 1/1 confirmed over 4 files.

## Completeness

- Siblings by shape: the non-test tree holds two `websocket.Upgrader` literals, `handlers.go:1155` and the emitter's `server.go:29`, both set by this change and both under a test; the only hard-coded `ws://` in a `.js`, `.html` or `.ts` file is `hotreload.js:8`, row 2.
- Catalog: the change is RPC and gnodev tooling, which the gno delta exempts from the catalog walk; of its classes only global mutable state applies, and the one lock in reach, `muRemote`, is row 4. No package-level `var` is added.
- Claims the diff writes: the `handlers.go:1145-1148` comment says gorilla's same-origin check accepts a request with no `Origin`; gorilla v1.5.3 `server.go:91-93` returns `true` on an empty `Origin`, read this round. Its claim about the HTTP CORS handler was not re-read this round.
- Reach not covered by a finder: the decided `[]` wiring at `startRPC` had no node-level test, row 1, now with `tests/judge-1-empty-list_test.go`; the operator-facing `cors_allowed_origins` comment in `config.go:28`, outside the diff, is row 5 from this pass.
- Thin angle: the author's own mutations were not re-run here except the empty-list one of row 1; the fixed-host script mutation reddens `TestMiddlewareUsesHTMLTemplate` by construction, since its `want` string pins `window.location.host`, read and not run.
- Unverified premise: row 3 rests on Python websocket-client sending its target as `Origin` by default, the finder's read of `_handshake.py`; the package is not installed here, and the Go run proves only the server side.

## Applied

| Commit | Rows | What changed |
| --- | --- | --- |
| b5d4588a3 | 1 | `TestNodeRPCWebsocketOrigin` gains the empty-list case |
| 1cd92ef6a | 2, 4 | the script dials `wss://` on an `https` page; the unused `muRemote` lock is gone |
| c2f0fc889 | 5 | the `CORSAllowedOrigins` comment says the list gates `/websocket` |

Row 3 is not applied: it goes to the pull request body's Design alternatives as the node's own origin.
