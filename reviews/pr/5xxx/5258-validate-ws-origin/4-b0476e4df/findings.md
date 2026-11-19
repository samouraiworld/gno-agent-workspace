# Findings in posting order, from round assemble: 5 to post, 0 SKIP, 0 refuted kept out

## Body

- Nit: tm2/pkg/bft/rpc/config/config.go:28 [gh](https://github.com/gnolang/gno/blob/b0476e4dfc86d56f6d740b85b4a40dffe281d1af/tm2/pkg/bft/rpc/config/config.go#L28), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: claims / reach (completeness)
  TL;DR: the cors_allowed_origins comment calls [] the default and names only cross-domain HTTP, while the default is ["*"] and the list now also decides /websocket upgrades
  Check: sed -n '28p;108p;170,172p' tm2/pkg/bft/rpc/config/config.go
  Details: config.go:28 says "Default value '[]' disables cors support"; config.go:108 sets []string{"*"}. Nothing in the comment says the list gates the websocket Origin, that [] keeps it same-origin, or that a narrowed list refuses the node's own host. The line sits outside the round 4 diff and the wording predates it; a comment-wording finding, mechanical.
  Evidence: config.go:28 comment text quoted above; config.go:108 CORSAllowedOrigins: []string{"*"}

## tm2/pkg/bft/node/node_test.go:119 [gh](https://github.com/gnolang/gno/blob/b0476e4dfc86d56f6d740b85b4a40dffe281d1af/tm2/pkg/bft/node/node_test.go#L119) · Missing test
State: CONFIRMED, band: Missing test, angle: removed behaviour / reach
TL;DR: no node-level test pins cors_allowed_origins = [] as same-origin on /websocket, so startRPC can hand an empty list to NewWebsocketManager as ["*"] with CI green
Check: go test -overlay with node.go:951 mapping an empty list to []string{"*"}: -run 'TestNodeRPC|TestWebsocketManagerCheckOrigin' ./tm2/pkg/bft/node ./tm2/pkg/bft/rpc/lib/server; then tests/judge-1-empty-list_test.go at head and under the same mutation
Details: With node.go:951 rewritten to pass []string{"*"} when the list is empty, every existing node and server origin test still passes; a node started with [] then upgrades a websocket for Origin https://evil.test. The judge's test, a node with [] dialled from https://evil.test (403), its own host (101) and no Origin (101), passes at head and fails under the mutation. Fix is mechanical: add that test beside TestNodeRPCWebsocketOrigin.
Evidence: mutation: ok tm2/pkg/bft/node, ok tm2/pkg/bft/rpc/lib/server; judge test at head: ok; under mutation: FAIL TestJudgeNodeRPCWebsocketOriginEmptyList, expected: 403, actual: 101, origin "https://evil.test"
Artifact: tests/judge-1-empty-list_test.go

## contribs/gnodev/pkg/emitter/middleware.go:21 [gh](https://github.com/gnolang/gno/blob/b0476e4dfc86d56f6d740b85b4a40dffe281d1af/contribs/gnodev/pkg/emitter/middleware.go#L21) · Nit
State: CONFIRMED, band: Nit, angle: removed behaviour (rename sweep)
TL;DR: muRemote is read-locked on every request and write-locked nowhere, and is still named after the remote field this round renamed to path
Check: grep -rn muRemote contribs/gnodev; git log --oneline -S'muRemote.Lock' -- contribs/gnodev
Details: Three hits only: the declaration at middleware.go:21 and the RLock/RUnlock pair at 63-64; history holds no muRemote.Lock. path is set once in NewMiddleware and never written, and m.script is guarded by onceExec, so the lock protects nothing. Removal touches the field at 21 and the two lines at 63-64; mechanical.
Evidence: middleware.go:21 muRemote sync.RWMutex; :63 m.muRemote.RLock(); :64 defer m.muRemote.RUnlock(); git log -S'muRemote.Lock': no output

## contribs/gnodev/pkg/emitter/static/hotreload.js:8 [gh](https://github.com/gnolang/gno/blob/b0476e4dfc86d56f6d740b85b4a40dffe281d1af/contribs/gnodev/pkg/emitter/static/hotreload.js#L8) · Suggestion
State: CONFIRMED, band: Suggestion, angle: line by line / reach
TL;DR: the injected script dials 'ws://' + window.location.host, which Chromium refuses with a SecurityError on any https page whose host is not loopback, so gnodev behind a TLS front never hot-reloads
Check: chromium --headless --ignore-certificate-errors --host-resolver-rules='MAP devhost.test 127.0.0.1' --dump-dom on tests/judge-2-mixed-content/page.html served over TLS by serve.py, at https://devhost.test:18443 and https://localhost:18443
Details: On an https page at devhost.test the constructor throws before any request leaves, while the page-derived scheme (location.protocol === 'https:' ? 'wss://' : 'ws://') constructs; on https://localhost both construct, since loopback is exempt. Behind a Host-preserving TLS proxy the wss:// dial then carries Origin https://<host> and Host <host>, which gorilla's checkSameOrigin (v1.5.3 server.go:89-99, host compare only) accepts. At dd05df75b the script dialled ws://127.0.0.1:8888 and the emitter's nil CheckOrigin compared that Host against the proxy's Origin, so the same setup failed there too, by the emitter's refusal rather than the browser's (read, not run). Run headless, since xvfb-run is absent; the result is a thrown SecurityError, not a null. Fix is mechanical, one line.
Evidence: https://devhost.test:18443: "ws:// SecurityError: Failed to construct 'WebSocket': An insecure WebSocket connection may not be initiated from a page loaded over HTTPS." / "wss:// constructed"; https://localhost:18443: "ws:// constructed" / "wss:// constructed"
Artifact: tests/judge-2-mixed-content/page.html

## tm2/pkg/bft/rpc/lib/server/handlers.go:1149 [gh](https://github.com/gnolang/gno/blob/b0476e4dfc86d56f6d740b85b4a40dffe281d1af/tm2/pkg/bft/rpc/lib/server/handlers.go#L1149) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach / removed behaviour (case space)
TL;DR: a client sending Origin http://<the node's own host> upgrades under [] and gets 403 once the list is narrowed, so narrowing the list revokes it
Check: tests/judge-3-own-host-origin_test.go overlaid into tm2/pkg/bft/rpc/lib/server, go test -v -run TestJudgeOwnHostOrigin at b0476e4df and at dd05df75b
Details: Case space for Origin naming the node's own host: [] accepts it (gorilla same-origin), a narrowed list without that host refuses it (corsValidator.OriginAllowed), absent Origin is accepted by both after this round, a foreign Origin is refused by both. The refusal under a narrowed list is also the DNS-rebinding defence, since a rebound page sends an Origin equal to Host, so accepting same-host Origins there reopens it. That the Python websocket-client sends its target as Origin by default rests on the finder's read of _handshake.py; the package is not installed here. The behaviour predates round 4 (same output at dd05df75b). A decision for the author: keep it and say in the cors_allowed_origins comment that a narrowed list must name what non-browser clients send as Origin, or accept same-host Origins and give up the rebinding refusal.
Evidence: b0476e4df: "empty list: Origin http://<own host> -> 101", "narrowed list: Origin http://<own host> -> 403"; dd05df75b: identical
Artifact: tests/judge-3-own-host-origin_test.go
