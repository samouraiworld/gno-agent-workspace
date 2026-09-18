# Findings in posting order, from round assemble: 2 to post, 2 SKIP, 0 refuted kept out

## SKIP docs/resources/rpc-endpoints.md:52 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L52) · Nit
State: PLAUSIBLE, band: Nit, angle: claims
TL;DR: the deadlines are in the source; the truncation needs a node
Check: curl -s -o /dev/null -w '%{http_code} %{size_download}' https://rpc.gno.land:443/genesis against a mainnet node
Details: WriteTimeout is 30s at tm2/pkg/bft/rpc/lib/server/http_server.go:47 and defaultWSWriteWait is 10s at handlers.go:492, both set once per body, and ResultGenesis implements StreamJSON, so the response is streamed and a batch is refused at handlers.go:164. Whether gno.land's genesis outruns the 30 seconds is not decidable from this tree.
Evidence: http_server.go:47 WriteTimeout: 30 * time.Second; handlers.go:492 defaultWSWriteWait = 10 * time.Second; handlers.go:164 streaming results are not supported in batch JSON-RPC requests

## docs/resources/rpc-endpoints.md:245 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L245) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: two config.toml comments still assert the gRPC server and dial_seeds
Check: grep -rn GRPCListenAddress over tm2 and gno.land: config plumbing, a wal generator and config get/set tests, no server start; Routes at routes.go has no dial_seeds
Details: config.go:38 documents grpc_laddr as the address for the gRPC server to listen on and adds NOTE: This server only supports /broadcast_tx_commit; config.go:48 documents unsafe as activating unsafe RPC commands like /dial_seeds and /unsafe_flush_mempool. Both are comment: struct tags, so both are written into every generated config.toml. No Go file reads GRPCListenAddress to start a server, and Routes registers no dial_seeds, so the page's rows are right and the config file is the sibling the sweep missed.
Evidence: grep -rn GRPCListenAddress tm2 gno.land --include=*.go -> config.go:38,101,123, consensus/wal_generator.go:38, config_set_test.go:634, config_get_test.go:734,742; no listener

## SKIP tm2/pkg/bft/rpc/lib/server/handlers.go:242 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/lib/server/handlers.go#L242) · Nit
State: PLAUSIBLE, band: Nit, angle: reach
TL;DR: the arity mismatch is in the read, the 500 was not run
Check: a request with no params key against the JSON-RPC handler for block: expect 500 with a JSON-RPC error body
Details: handlers.go:242 gates the argument conversion on len(req.Params) > 0 and line 253 calls rpcFunc.f.Call(args) regardless, so an endpoint declaring a parameter is called with the context alone. The response a caller sees was not exercised in this round, and the behaviour is the merge base's as much as the branch's: the branch only documents the workaround.
Evidence: handlers.go:242-253 read; no run

## tm2/pkg/bft/rpc/core/pipe.go:25 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/core/pipe.go#L25) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: validatePage has no caller outside its own test
Check: grep -rn validatePage tm2 | grep -v _test: one hit, the definition at pipe.go:123
Details: pipe.go:123 defines validatePage(page, perPage, totalCount); the only other mention in tm2 is pipe_test.go:45. The live consumer of both constants is validatePerPage, called once, at mempool.go:123 for the unconfirmed_txs limit, which is what the new comment describes. Removing validatePage touches pipe.go:123 and the page cases in pipe_test.go.
Evidence: grep -rn validatePage tm2 | grep -v _test -> tm2/pkg/bft/rpc/core/pipe.go:123:func validatePage(page, perPage, totalCount int) (int, error)
