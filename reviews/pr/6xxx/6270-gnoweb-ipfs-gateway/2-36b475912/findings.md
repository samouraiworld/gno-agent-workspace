# Findings in posting order, from round assemble: 2 to post, 0 SKIP, 3 refuted kept out

## gno.land/pkg/gnoweb/render.go:98 [gh](https://github.com/gnolang/gno/blob/36b475912eb31badc961e9b35fa765def60f250f/gno.land/pkg/gnoweb/render.go#L98) · Missing test
State: CONFIRMED, band: Missing test, angle: lines + claims (secret handling)
TL;DR: no test reads NewHTMLRenderer's warning, so putting the gateway value back on it stays green
Check: mutate render.go:98 to echo cfg.IPFSGateway; go test ./gno.land/pkg/gnoweb -run IPFS -count=1: expect red, observe green
Details: TestNewHTMLRendererIPFSGateway (ipfs_test.go:141-158) passes log.NewTestingLogger(t) and asserts only the rendered src; the s3cret loop at ipfs_test.go:117-132 calls NewRouter alone. Reach: NewHTMLRenderer's only in-tree caller is NewRouter (app.go:214), which hands it an already-normalized gateway, so the warning fires for out-of-tree callers of the exported constructor only. A paste-ready test capturing the slog output passes at head and fails under the mutation.
Evidence: mutation render.go:98 -> logger.Warn("ignoring invalid IPFS gateway", "gateway", cfg.IPFSGateway, "error", err); go test ./gno.land/pkg/gnoweb -run IPFS -count=1: ok (green). tests/judge-1-renderer-warn-redaction_test.go: ok at head; under the mutation FAIL, "... gateway=https://user:s3cret@gw.example error=\"must not contain credentials\"" should not contain "s3cret"
Artifact: tests/judge-1-renderer-warn-redaction_test.go
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: the eight secret-bearing shapes run through NewRouter only; the renderer's own warning has no test
Check: mutate render.go:98; go test ./gno.land/pkg/gnoweb -run IPFS -count=1; grep -rn s3cret gno.land contribs --include='*_test.go'
Details: grep -rn s3cret over *_test.go in gno.land and contribs: gnoweb hits only in the NewRouter loop of TestNewRouterIPFSGateway, the rest in indexer/client_test.go and omnisearch/handler_test.go. Same mutation, same green run; the judge's test reddens it.
Evidence: grep -rn s3cret --include='*_test.go' gno.land contribs: ipfs_test.go NewRouter loop, indexer/client_test.go:537-592, omnisearch/handler_test.go:320; mutation run: ok
Artifact: tests/judge-1-renderer-warn-redaction_test.go

## gno.land/cmd/gnoweb/main.go:241 [gh](https://github.com/gnolang/gno/blob/36b475912eb31badc961e9b35fa765def60f250f/gno.land/cmd/gnoweb/main.go#L241) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: -ipfs-gateway help says only ipfs:// and retired-gateway URLs are rewritten, though URLs on ipfs.filebase.io are rewritten too
Check: grep -rn 'retired gateway URLs' gno.land/cmd/gnoweb/main.go contribs/gnodev: expect no hit after the fix
Details: ext_ipfs.go adds "ipfs.filebase.io": false to ipfsRewrittenGateways, so an operator who moves -ipfs-gateway off Filebase has every stored ipfs.filebase.io URL sent to the new gateway. The help at main.go:241, gnodev's -web-ipfs-gateway help (contribs/gnodev/app_config.go:152, README.md:113) and the godocs at ipfs.go:13-15 and markdown/context.go:24 name only retired gateways; docs/users/explore-with-gnoweb.md and the ADR were updated.
Evidence: sed -n 241p main.go: "IPFS gateway origin that ipfs:// URLs and retired gateway URLs (ipfs.io, dweb.link, ...) are rewritten to; empty disables the rewrite"; grep -rn 'retired gateway' hits app_config.go:152, README.md:113, context.go:24; ext_ipfs.go "ipfs.filebase.io": false, // gnoweb.DefaultIPFSGateway
