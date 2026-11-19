# Findings in posting order, from round assemble: 6 to post, 0 SKIP, 0 refuted kept out

## contribs/gnodev/pkg/emitter/server.go:30 [gh](https://github.com/gnolang/gno/blob/dd05df75bc312f6bd064ba096225699f5e8fe320/contribs/gnodev/pkg/emitter/server.go#L30) · Warning
State: CONFIRMED, band: Warning, angle: user topic / reach
TL;DR: gnodev hot reload is refused for a page opened at the documented http://localhost:8888, since the injected script dials ws://127.0.0.1:8888/_events and gorilla's same-origin default compares the two host strings
Check: go test -run 'TestSoloFinder1|TestSoloFinder2_PageOrigin' ./pkg/emitter at head and at the merge base via -overlay; real gnodev + chromium via tests/solo-finder-1-gnodev-localhost-hotreload.sh at head
Details: CheckOrigin: nil (server.go:30) falls to gorilla checkSameOrigin, which compares the Origin host with r.Host verbatim. app.go:492 builds the hot-reload target from webListenerAddr, default 127.0.0.1:8888 (command_local.go:34, command_staging.go:25), while docs/resources/gnodev.md:39, docs/builders/quickstart.md:22, docs/builders/getting-started.md:198 and contribs/gnodev/README.md:16 send users to http://localhost:8888. The page never reloads on save; the only trace is a console error and a gnodev Event log line. Same defect as candidate 1002; one edit closes both.
Evidence: head dd05df75b: page http://127.0.0.1:<p> -> ws://127.0.0.1:<p>/_events status 101; page http://localhost:<p> -> status 403 'websocket: bad handshake', gnodev logs 'request origin not allowed by Upgrader.CheckOrigin'. Merge base d25b308c1 (go test -overlay, same file): both rows 101, PASS. Real gnodev built from the head tree + chromium, page http://localhost:18888/: console 'WebSocket Error: [object Event]', gnodev 'unable to upgrade connection ... request origin not allowed by Upgrader.CheckOrigin'; page http://127.0.0.1:18888/: 0 upgrade errors.
Artifact: tests/solo-finder-1-gnodev-localhost-hotreload_test.go
State: CONFIRMED, band: Warning, angle: removed behaviour
TL;DR: the same-origin default now in force refuses gnodev's own hot-reload socket whenever the host the user types differs from the -web-listener string
Check: go test -run TestSoloFinder2_PageOrigin ./pkg/emitter at head and at the merge base via -overlay
Details: Rerun: localhost row 403 at head, 101 at base; 127.0.0.1 row 101 on both. Any listener string differing from the typed host (localhost, 0.0.0.0, LAN IP) fails the same check. Same defect as candidate 1 at the same anchor.
Evidence: head dd05df75b: page http://127.0.0.1:<p> -> ws://127.0.0.1:<p>/_events status 101; page http://localhost:<p> -> status 403 'websocket: bad handshake', gnodev logs 'request origin not allowed by Upgrader.CheckOrigin'. Merge base d25b308c1 (go test -overlay, same file): both rows 101, PASS. Real gnodev built from the head tree + chromium, page http://localhost:18888/: console 'WebSocket Error: [object Event]', gnodev 'unable to upgrade connection ... request origin not allowed by Upgrader.CheckOrigin'; page http://127.0.0.1:18888/: 0 upgrade errors.
Artifact: tests/solo-finder-2-gnodev-page-origin_test.go
State: CONFIRMED, band: Missing test, angle: tests
TL;DR: the emitter's origin restriction has no test: its only dial sends no Origin, which both the old always-true check and the same-origin default accept
Check: grep contribs/gnodev/pkg/emitter/*_test.go for Dial and Origin
Details: server_test.go:31 websocket.DefaultDialer.Dial(u, nil); gorilla checkSameOrigin returns true on an absent Origin, so reverting CheckOrigin to return true keeps the suite green, and neither the hardening nor the hot-reload regression of 1 and 1002 is pinned.
Evidence: grep: the single Dial in the package tests passes a nil header; no Origin header anywhere in emitter tests

## tm2/pkg/bft/rpc/lib/server/handlers.go:1144 [gh](https://github.com/gnolang/gno/blob/dd05df75bc312f6bd064ba096225699f5e8fe320/tm2/pkg/bft/rpc/lib/server/handlers.go#L1144) · Warning
State: CONFIRMED, band: Warning, angle: user topic: no-Origin client under every CORSAllowedOrigins value; removed behaviour; twin input paths
TL;DR: a node whose cors_allowed_origins is narrowed refuses /websocket to every client sending no Origin, the in-tree Go ws client included, while it still serves the same client's HTTP POST
Check: go test -run TestSoloFinder2_NoOriginClient ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay with the origins argument dropped
Details: handlers.go:1144 installs rs/cors OriginAllowed, which reads r.Header.Get("Origin") and admits "" only when allowedOriginsAll (cors.go:454-468); the HTTP path passes a missing Origin through (cors.go:415-417). Callers of the in-tree ws client: contribs/tx-archive (backup), contribs/gnohealth (timestamp), tm2/pkg/bft/rpc/client. Default config ["*"] and every misc/deployments config.toml (["*"]) are unaffected; the cut needs an operator narrowing the list. Same defect as candidate 2.
Evidence: head dd05df75b: narrowed list ['https://gno.land','https://*.gno.land'], origin "" -> 403; ws.NewClient against cors_allowed_origins=["https://gno.land"] and ["https://*.gno.land"]: 'unable to dial RPC, websocket: bad handshake' while HTTP POST answers 200; [] and ["*"] connect. Merge base (overlay, constructor without origins): every row connects, origin "" -> 101.
Artifact: tests/solo-finder-2-no-origin-client_test.go
State: CONFIRMED, band: Warning, angle: line by line / claims
TL;DR: with a narrowed list, OriginAllowed reads an absent Origin as "" and refuses the upgrade, unlike the HTTP CORS path the PR body says it matches
Check: go test -run TestSoloFinder1 ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay
Details: Rerun: no_origin_cli 403 at head, 101 at base; allowed_page 101 at head; cross_site_page and null_origin_page 403 at head, 101 at base (the intended hardening). Same defect as candidate 1001 at the same anchor.
Evidence: head dd05df75b: narrowed list ['https://gno.land','https://*.gno.land'], origin "" -> 403; ws.NewClient against cors_allowed_origins=["https://gno.land"] and ["https://*.gno.land"]: 'unable to dial RPC, websocket: bad handshake' while HTTP POST answers 200; [] and ["*"] connect. Merge base (overlay, constructor without origins): every row connects, origin "" -> 101.
Artifact: tests/solo-finder-1-narrowed-no-origin_test.go

## tm2/pkg/bft/node/node.go:952 [gh](https://github.com/gnolang/gno/blob/dd05df75bc312f6bd064ba096225699f5e8fe320/tm2/pkg/bft/node/node.go#L952) · Missing test
State: CONFIRMED, band: Missing test, angle: tests
TL;DR: no test drives startRPC's WebSocket origin wiring, so dropping or widening the list node.go passes stays green
Check: read node_test.go TestNodeRPCBatchLimit and grep the tree for node-level ws dials with a non-default CORS list
Details: The only node-level ws dial, node_test.go:105 in TestNodeRPCBatchLimit, uses cfg.ResetTestRoot (default ["*"], config.go:108) and a nil header, which upgrades under nil, ["*"] and the configured list alike. Other CORSAllowedOrigins test hits (gno.land/cmd/gnoland config_get/set, misc/gnoe2e override_test) assert config parsing only.
Evidence: grep -rln CORSAllowedOrigins --include=*_test.go: gnoland config tests and gnoe2e override_test only; node_test.go:105 Dial(..., nil)

## tm2/pkg/bft/rpc/lib/server/handlers_test.go:994 [gh](https://github.com/gnolang/gno/blob/dd05df75bc312f6bd064ba096225699f5e8fe320/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L994) · Missing test
State: CONFIRMED, band: Missing test, angle: tests
TL;DR: the table pins a missing Origin only under an empty list, so the narrowed-list no-Origin cell of 1001 has no row
Check: read handlers_test.go:978-1035, the TestWebsocketManagerCheckOrigin table
Details: Row 'no origin header is allowed' (994-996) sets allowedOrigins []string{}, which takes the nil branch (handlers.go:1140); every narrowed-list row (1000-1032) sets an Origin. Neither the current refusal nor a fix accepting absent Origin turns a row red.
Evidence: grep of the table: allowedOrigins/origin pairs at 989-1032; the only origin "" row is at 995-996 with []string{}

## tm2/pkg/bft/node/node.go:951 [gh](https://github.com/gnolang/gno/blob/dd05df75bc312f6bd064ba096225699f5e8fe320/tm2/pkg/bft/node/node.go#L951) · Suggestion
State: CONFIRMED, band: Suggestion, angle: line by line (guard the callee already applies)
TL;DR: the IsCorsEnabled guard repeats the len check NewWebsocketManager already makes; passing n.config.RPC.CORSAllowedOrigins directly behaves the same
Check: read config.go:171 and handlers.go:1140; apply the direct pass in a scratch tree; go vet ./tm2/pkg/bft/node and go test -run 'TestWebsocket|TestNodeStartStop' over the rpc server and node packages
Details: config.go:171 IsCorsEnabled is len(CORSAllowedOrigins) != 0 and handlers.go:1140 builds the validator only when len(allowedOrigins) > 0, so an empty list reaches nil either way. Refactor: 6 lines (950-954 plus the argument line) become 1 (git diff --stat: 1 insertion, 6 deletions).
Evidence: rewrite applied: go vet ok; ok tm2/pkg/bft/rpc/lib/server 1.513s; ok tm2/pkg/bft/node 0.113s; tree restored with git checkout -- node.go

## tm2/pkg/bft/rpc/lib/server/handlers.go:1139 [gh](https://github.com/gnolang/gno/blob/dd05df75bc312f6bd064ba096225699f5e8fe320/tm2/pkg/bft/rpc/lib/server/handlers.go#L1139) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: with cors_allowed_origins = [] a page served from the node's own public origin through a Host-rewriting reverse proxy is refused on /websocket
Check: go test -run TestJudgeR1_ProxyHost ./tm2/pkg/bft/rpc/lib/server at head and at the merge base via -overlay
Details: node.go:951 passes nil when CORS is disabled, so handlers.go:1139 leaves gorilla's same-origin check, which compares the Origin host with the Host the node receives. nginx's proxy_pass default rewrites Host to the upstream, so Origin https://rpc.example.com meets Host 127.0.0.1:26657 and gets 403. Needs the non-default [] list; the default ["*"] is unaffected.
Evidence: head: proxy_keeps_host 101, proxy_rewrites_host 403 'websocket: bad handshake'; merge base (overlay, constructor without origins): both 101, PASS
Artifact: tests/judge-r1-proxy-host_test.go
