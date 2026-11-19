# Claims: gnolang/gno#5258 round 3, dd05df75b, claude-opus-5-5, deep review

Round shape: solo round on the reviewer's own pull request, 2 finders and one judge, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`server.go:30`](https://github.com/gnolang/gno/blob/dd05df75b/contribs/gnodev/pkg/emitter/server.go#L30) | 1 | Warning | CONFIRMED | differs | posts |
| [`server.go:30`](https://github.com/gnolang/gno/blob/dd05df75b/contribs/gnodev/pkg/emitter/server.go#L30) | 1002 | Warning | CONFIRMED | differs | with row 1 |
| [`server.go:30`](https://github.com/gnolang/gno/blob/dd05df75b/contribs/gnodev/pkg/emitter/server.go#L30) | 1005 | Missing test | CONFIRMED | same | with row 1 |
| [`handlers.go:1144`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/rpc/lib/server/handlers.go#L1144) | 1001 | Warning | CONFIRMED | differs | posts |
| [`handlers.go:1144`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/rpc/lib/server/handlers.go#L1144) | 2 | Warning | CONFIRMED | differs | with row 1001 |
| [`node.go:952`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/node/node.go#L952) | 1004 | Missing test | CONFIRMED | same | posts |
| [`handlers_test.go:994`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L994) | 1003 | Missing test | CONFIRMED | same | posts |
| [`node.go:951`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/node/node.go#L951) | 3 | Suggestion | CONFIRMED | same | posts |
| [`handlers.go:1139`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/rpc/lib/server/handlers.go#L1139) | 3001 | Suggestion | CONFIRMED | differs | posts |

### 1 · Warning · CONFIRMED · `contribs/gnodev/pkg/emitter/server.go:30`

Gnodev hot reload is refused for a page opened at the documented http://localhost:8888, since the injected script dials ws://127.0.0.1:8888/_events and gorilla's same-origin default compares the two host strings

- Check: go test -run 'TestSoloFinder1\|TestSoloFinder2_PageOrigin' ./pkg/emitter at head and at the merge base via -overlay; real gnodev + chromium via tests/solo-finder-1-gnodev-localhost-hotreload.sh at head
- Observed: head dd05df75b: page http://127.0.0.1:<p> -> ws://127.0.0.1:<p>/_events status 101; page http://localhost:<p> -> status 403 'websocket: bad handshake', gnodev logs 'request origin not allowed by Upgrader.CheckOrigin'. Merge base d25b308c1 (go test -overlay, same file): both rows 101, PASS. Real gnodev built from the head tree + chromium, page http://localhost:18888/: console 'WebSocket Error: [object Event]', gnodev 'unable to upgrade connection ... request origin not allowed by Upgrader.CheckOrigin'; page http://127.0.0.1:18888/: 0 upgrade errors.
- Test: [`tests/solo-finder-1-gnodev-localhost-hotreload_test.go`](tests/solo-finder-1-gnodev-localhost-hotreload_test.go)

### 1002 · Warning · CONFIRMED · `contribs/gnodev/pkg/emitter/server.go:30`

The same-origin default now in force refuses gnodev's own hot-reload socket whenever the host the user types differs from the -web-listener string

- Check: go test -run TestSoloFinder2_PageOrigin ./pkg/emitter at head and at the merge base via -overlay
- Observed: head dd05df75b: page http://127.0.0.1:<p> -> ws://127.0.0.1:<p>/_events status 101; page http://localhost:<p> -> status 403 'websocket: bad handshake', gnodev logs 'request origin not allowed by Upgrader.CheckOrigin'. Merge base d25b308c1 (go test -overlay, same file): both rows 101, PASS. Real gnodev built from the head tree + chromium, page http://localhost:18888/: console 'WebSocket Error: [object Event]', gnodev 'unable to upgrade connection ... request origin not allowed by Upgrader.CheckOrigin'; page http://127.0.0.1:18888/: 0 upgrade errors.
- Test: [`tests/solo-finder-2-gnodev-page-origin_test.go`](tests/solo-finder-2-gnodev-page-origin_test.go)

### 1005 · Missing test · CONFIRMED · `contribs/gnodev/pkg/emitter/server.go:30`

The emitter's origin restriction has no test: its only dial sends no Origin, which both the old always-true check and the same-origin default accept

- Check: grep contribs/gnodev/pkg/emitter/*_test.go for Dial and Origin
- Observed: grep: the single Dial in the package tests passes a nil header; no Origin header anywhere in emitter tests

### 1001 · Warning · CONFIRMED · `tm2/pkg/bft/rpc/lib/server/handlers.go:1144`

A node whose cors_allowed_origins is narrowed refuses /websocket to every client sending no Origin, the in-tree Go ws client included, while it still serves the same client's HTTP POST

- Check: go test -run TestSoloFinder2_NoOriginClient ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay with the origins argument dropped
- Observed: head dd05df75b: narrowed list ['https://gno.land','https://*.gno.land'], origin "" -> 403; ws.NewClient against cors_allowed_origins=["https://gno.land"] and ["https://*.gno.land"]: 'unable to dial RPC, websocket: bad handshake' while HTTP POST answers 200; [] and ["*"] connect. Merge base (overlay, constructor without origins): every row connects, origin "" -> 101.
- Test: [`tests/solo-finder-2-no-origin-client_test.go`](tests/solo-finder-2-no-origin-client_test.go)

### 2 · Warning · CONFIRMED · `tm2/pkg/bft/rpc/lib/server/handlers.go:1144`

With a narrowed list, OriginAllowed reads an absent Origin as "" and refuses the upgrade, unlike the HTTP CORS path the PR body says it matches

- Check: go test -run TestSoloFinder1 ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay
- Observed: head dd05df75b: narrowed list ['https://gno.land','https://*.gno.land'], origin "" -> 403; ws.NewClient against cors_allowed_origins=["https://gno.land"] and ["https://*.gno.land"]: 'unable to dial RPC, websocket: bad handshake' while HTTP POST answers 200; [] and ["*"] connect. Merge base (overlay, constructor without origins): every row connects, origin "" -> 101.
- Test: [`tests/solo-finder-1-narrowed-no-origin_test.go`](tests/solo-finder-1-narrowed-no-origin_test.go)

### 1004 · Missing test · CONFIRMED · `tm2/pkg/bft/node/node.go:952`

No test drives startRPC's WebSocket origin wiring, so dropping or widening the list node.go passes stays green

- Check: read node_test.go TestNodeRPCBatchLimit and grep the tree for node-level ws dials with a non-default CORS list
- Observed: grep -rln CORSAllowedOrigins --include=*_test.go: gnoland config tests and gnoe2e override_test only; node_test.go:105 Dial(..., nil)

### 1003 · Missing test · CONFIRMED · `tm2/pkg/bft/rpc/lib/server/handlers_test.go:994`

The table pins a missing Origin only under an empty list, so the narrowed-list no-Origin cell of 1001 has no row

- Check: read handlers_test.go:978-1035, the TestWebsocketManagerCheckOrigin table
- Observed: grep of the table: allowedOrigins/origin pairs at 989-1032; the only origin "" row is at 995-996 with []string{}

### 3 · Suggestion · CONFIRMED · `tm2/pkg/bft/node/node.go:951`

The IsCorsEnabled guard repeats the len check NewWebsocketManager already makes; passing n.config.RPC.CORSAllowedOrigins directly behaves the same

- Check: read config.go:171 and handlers.go:1140; apply the direct pass in a scratch tree; go vet ./tm2/pkg/bft/node and go test -run 'TestWebsocket\|TestNodeStartStop' over the rpc server and node packages
- Observed: rewrite applied: go vet ok; ok tm2/pkg/bft/rpc/lib/server 1.513s; ok tm2/pkg/bft/node 0.113s; tree restored with git checkout -- node.go

### 3001 · Suggestion · CONFIRMED · `tm2/pkg/bft/rpc/lib/server/handlers.go:1139`

With cors_allowed_origins = [] a page served from the node's own public origin through a Host-rewriting reverse proxy is refused on /websocket

- Check: go test -run TestJudgeR1_ProxyHost ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay
- Observed: head: proxy_keeps_host 101, proxy_rewrites_host 403 'websocket: bad handshake'; merge base (overlay, constructor without origins): both 101, PASS
- Test: [`tests/judge-r1-proxy-host_test.go`](tests/judge-r1-proxy-host_test.go)

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`go.mod:34`](https://github.com/gnolang/gno/blob/dd05df75b/contribs/gnohealth/go.mod#L34) | reach | another module importing rpc/lib/server lacks the rs/cors requirement the diff adds to gnohealth and gnokeykc | go list -deps ./... \| grep -c 'tm2/pkg/bft/rpc/lib/server$' in each go.mod without rs/cors (contribs/gnokms, gnobr, gnomd, github-bot, misc/gnopreview, misc/audit-pattern-harness): 0 for every one |
| [`handlers.go:1144`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/rpc/lib/server/handlers.go#L1144) | user topic | a cross-site or opaque-origin page still opens the websocket of a node whose list is narrowed | solo-finder-1-narrowed-no-origin_test.go at head: cross_site_page https://evil.example -> 403, null_origin_page 'null' -> 403; only listed origins upgrade |
| [`node.go:951`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/node/node.go#L951) | user topic | with cors_allowed_origins = [] a DNS-rebinding page passes gorilla's same-origin check and opens the websocket | grep -rn 'r.Host\\|req.Host\\|AllowedHosts' tm2/pkg/bft/rpc/lib/server (non-test): no hits, so the HTTP routes carry the same rebinding exposure and the websocket adds no reach beyond them; a Host allowlist is a separate change |
| [`node.go:951`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/node/node.go#L951) | claims | the PR body says any website could open a cross-site websocket to a user's node, and the default cors_allowed_origins ["*"] keeps that true | sed -n 108p tm2/pkg/bft/rpc/config/config.go: CORSAllowedOrigins []string{"*"}; the HTTP CORS middleware serves the same route map to any origin under that default, so the websocket under "*" matches HTTP and only the description's wording is at stake |
| [`go.mod:206`](https://github.com/gnolang/gno/blob/dd05df75b/contribs/gnokeykc/go.mod#L206) | removed behaviour | a module importing rpcserver without rs/cors in its go.mod/go.sum would stop building | grep -L rs/cors over every go.mod replacing github.com/gnolang/gno lists contribs/gnobr, contribs/github-bot, contribs/gnokms; go list -deps -test ./... \| grep -c 'rpc/lib/server\\|rs/cors' prints 0 in each |
| [`server.go:30`](https://github.com/gnolang/gno/blob/dd05df75b/contribs/gnodev/pkg/emitter/server.go#L30) | user topic | gnobro's dev client to gnodev /_events could be refused by the same-origin default | contribs/gnobro/main.go:267 devcl.Run(ctx, devpoint, nil) dials with a nil header; gorilla v1.5.3 server.go:90-93 checkSameOrigin returns true when the Origin header is absent |
| [`handlers.go:1140`](https://github.com/gnolang/gno/blob/dd05df75b/tm2/pkg/bft/rpc/lib/server/handlers.go#L1140) | catalog: error and panic handling | every refused origin logs at Error level, letting a remote fill node logs | WebsocketHandler's 'Failed to upgrade to websocket connection' Error log is unchanged context in the diff and already fires at the merge base for any malformed upgrade, e.g. a GET /websocket with no Upgrade header; the class predates the diff |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | contribs/gnodev/pkg/emitter/server.go:30 | go test -run 'TestSoloFinder1\|TestSoloFinder2_PageOrigin' ./pkg/emitter at head and at the merge base via -overlay; real gnodev + chromium via tests/solo-finder-1-gnodev-localhost-hotreload.sh at head | hot | same-origin-host-string-mismatch |
| 2 | CONFIRMED | Warning | tm2/pkg/bft/rpc/lib/server/handlers.go:1144 | go test -run TestSoloFinder1 ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay | hot | absent-origin-refused |
| 3 | CONFIRMED | Suggestion | tm2/pkg/bft/node/node.go:951 | read config.go:171 and handlers.go:1140; apply the direct pass in a scratch tree; go vet ./tm2/pkg/bft/node and go test -run 'TestWebsocket\|TestNodeStartStop' over the rpc server and node packages | warm |  |
| 1001 | CONFIRMED | Warning | tm2/pkg/bft/rpc/lib/server/handlers.go:1144 | go test -run TestSoloFinder2_NoOriginClient ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay with the origins argument dropped | hot | absent-origin-refused |
| 1002 | CONFIRMED | Warning | contribs/gnodev/pkg/emitter/server.go:30 | go test -run TestSoloFinder2_PageOrigin ./pkg/emitter at head and at the merge base via -overlay | hot | same-origin-host-string-mismatch |
| 1003 | CONFIRMED | Missing test | tm2/pkg/bft/rpc/lib/server/handlers_test.go:994 | read handlers_test.go:978-1035, the TestWebsocketManagerCheckOrigin table | cold |  |
| 1004 | CONFIRMED | Missing test | tm2/pkg/bft/node/node.go:952 | read node_test.go TestNodeRPCBatchLimit and grep the tree for node-level ws dials with a non-default CORS list | warm |  |
| 1005 | CONFIRMED | Missing test | contribs/gnodev/pkg/emitter/server.go:30 | grep contribs/gnodev/pkg/emitter/*_test.go for Dial and Origin | hot |  |
| 3001 | CONFIRMED | Suggestion | tm2/pkg/bft/rpc/lib/server/handlers.go:1139 | go test -run TestJudgeR1_ProxyHost ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay | hot |  |

</details>

Hit rate per tier, from the record: hot 6/6 confirmed over 2 files, warm 2/2 confirmed over 4 files, cold 1/1 confirmed over 4 files.

## Completeness

- Sibling sweep by shape: every `websocket.Upgrader{` and every `.Upgrade(w` call in the head tree outside tests is one of the two sites the diff edits, `tm2/pkg/bft/rpc/lib/server/handlers.go:1149` and `contribs/gnodev/pkg/emitter/server.go:29`; no websocket server was left on an always-true check. Every `NewWebsocketManager(` caller (node.go, doc.go, abci_test.go, handlers_test.go) passes the new argument.
- Thin angle, reach of the same-origin default on the node: candidate 3001, run. With `cors_allowed_origins = []` a same-origin page behind a Host-rewriting proxy gets 403 at head and 101 at the merge base; banded Suggestion since the default `["*"]` and every `misc/deployments` config are unaffected.
- Invariant catalog: its classes (Realm audit patterns, caller identity predicates, the `current-guard` rule) cover realm and GnoVM code; this diff touches tm2 RPC and gnodev tooling only, so no class applies.
- Review threads: mvallenet's question on the empty default list is answered by `handlers.go:1140`, which builds the rs/cors validator only for a non-empty list, since rs/cors with an empty `AllowedOrigins` allows every origin. omarsy's request to bring the branch onto the current RPC server is met by the merge of d25b308c1; the four master call sites pass `nil`.
- Claims: the PR body says the WebSocket check matches the HTTP CORS behaviour; it does for a request carrying an Origin, and diverges on an absent Origin under a narrowed list, which is candidates 2 and 1001.
- Tests: the diff's new table (`TestWebsocketManagerCheckOrigin`) leaves the narrowed-list no-Origin cell (1003); node wiring (1004) and the gnodev emitter (1005) have no origin test at all.

## Applied

| Commit | Rows | What changed |
| --- | --- | --- |
| c2f0fc889 | 2, 1001, 1003 | an upgrade with no Origin is accepted under a narrowed list; the table gains that row |
| b5d4588a3 | 3, 1004 | `startRPC` passes `cors_allowed_origins` unchanged; `TestNodeRPCWebsocketOrigin` drives it |
| 1cd92ef6a | 1, 1002, 1005 | the hot-reload script dials the page's own host; `TestServer_CheckOrigin` pins the emitter's check |

Row 3001 is not applied: the author kept the same-origin check for an empty list, named under Design alternatives in the pull request body.
