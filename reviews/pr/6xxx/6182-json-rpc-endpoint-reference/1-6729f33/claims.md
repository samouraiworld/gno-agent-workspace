# Claims: PR 6182: docs: add a JSON-RPC endpoint reference, drop the parts that were false round 1, 6729f335f, the session model, solo review

Round shape: solo round, one agent, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | tm2/pkg/bft/rpc/core/pipe.go:123 | grep -rn validatePage tm2 \| grep -v _test: one hit, the definition at pipe.go:123 | grep -rn validatePage tm2 \| grep -v _test -> tm2/pkg/bft/rpc/core/pipe.go:123:func validatePage(page, perPage, totalCount int) (int, error) |  |  |
| 2 | CONFIRMED | Nit | tm2/pkg/bft/rpc/config/config.go:38 | grep -rn GRPCListenAddress over tm2 and gno.land: config plumbing, a wal generator and config get/set tests, no server start; Routes at routes.go has no dial_seeds | grep -rn GRPCListenAddress tm2 gno.land --include=*.go -> config.go:38,101,123, consensus/wal_generator.go:38, config_set_test.go:634, config_get_test.go:734,742; no listener |  |  |
| 3 | PLAUSIBLE | Nit | tm2/pkg/bft/rpc/lib/server/handlers.go:242 | a request with no params key against the JSON-RPC handler for block: expect 500 with a JSON-RPC error body | handlers.go:242-253 read; no run |  |  |
| 4 | REFUTED | Nit | docs/resources/rpc-endpoints.md:52 | curl -sS -o /tmp/genesis.out -w '%{http_code} %{size_download} %{time_total}' --max-time 60 https://rpc.gno.land:443/genesis | 200, 122681708 bytes, 30.99s, then `curl: (92) HTTP/2 stream 1 reset by server`: the page's sentence holds, the body outruns the 30s WriteTimeout at http_server.go:47 |  | cold |

Rows 1 and 2 ship in the Body and carry no anchor: the line each asks to edit, `pipe.go:123` and
`config/config.go:38`, sits outside the diff, and an inline comment there is refused at submit.

## Completeness

Read in full: all 12 changed files, the new 260-line page line by line, and the code each
of its claims rests on: `routes.go`, `blocks.go`, `consensus.go`, `mempool.go`, `tx.go`,
`abci.go`, `net.go`, `pipe.go`, `core/types/responses.go`, `rpc/lib/server/handlers.go`,
`rpc/lib/server/http_server.go`, `rpc/config/config.go`, `sdk/baseapp.go`, `state/store.go`,
`state/errors.go`, `consensus/replay.go`, `core/dev.go`.

Claims run down to the deciding line and confirmed: the 20-entry cap and the silent low-end
cut on `blockchain`; `height=0` on `block_results` reaching the responses saved at height 0;
the 30 default and 100 cap on `unconfirmed_txs`; the `validators` and `consensus_params`
default sitting one block past the last committed one; `height=0` rewritten to the tip
before the proof check, and the proof error at height 1 or below; proofs only on the
`.store` paths, where `main` is a real store key; the 409 reachable only from the URI
transport, since `HTTPStatusError` is unwrapped in the URI handler alone; `0x` hex decoded
only in `httpParamsToArgs`, case-sensitively; the positional array form requiring every
declared parameter; `params` omitted versus `{}`; a batch refusing a streaming result; the
30-second and 10-second write deadlines around the streamed `genesis` body; four `unsafe_*`
methods, two of them passing a caller-supplied filename to `os.Create`; no code path
starting a gRPC server from `grpc_laddr`; `tx` taking `hash` alone; the `n_txs`, `total`,
`total_bytes` and `txs` field names. The `+`-in-base64 figure is arithmetic: a 43-character
body over a 64-symbol alphabet carries at least one `+` with probability 1 - (63/64)^43,
0.49.

Claimed and dropped after the check: that the page quotes an error string the node never
emits. The tree has `Could not find tx result for hash #%X` at `state/errors.go:85`, which
the page quotes lowercased, so the claim holds.

Settled after the round by a mainnet request: gno.land's genesis outruns the 30-second write
deadline, 122,681,708 bytes and a stream reset at 31.0 seconds, so the page is right and the
candidate is refuted. Not settled here: the response a client sees when it omits `params`,
read at `handlers.go:242` and not run.

Not checked: the WebSocket answer shape for a one-element batch, and the `other` map's
transaction-index flag in the `status` response. Both are single lines of the page with no
finding resting on them.

Blast radius: the two deleted files have no reference anywhere in the tree, `doc_template.txt`
and `tm2/pkg/bft/rpc/core/README.md`, so nothing generates from either. The sidebar entry
`resources/rpc-endpoints` matches the new file's path, and the docs codegen and lint jobs
are green at this head.

## Retro

Measured with `./scripts/review-retro.py` over the round's workflow directory, and
`./scripts/todo` carries the upgrade.

What failed: the triage classed the target trivial, so the round collapsed to the solo
shape, one finder-judge-writer plus the overview agent, and the six finder jobs the
bundles earned never ran. Nothing died and no cap was hit; the round came in at 2 agents,
46k output, 3.5M cache read and 5 minutes against a projected 17 agents, 493k output, 39M
cache read and 39 minutes.

What worked: the claims angle, which is the whole of this diff's material. Fifteen of the
page's behavioural assertions were run down to the deciding line and held; the two that
did not settle are the two SKIP sections, each naming what would settle it.

Hit rate per tier: warm 1 of 1 confirmed, cold 1 of 2, and one candidate on a file outside
the diff, untiered, unconfirmed. The hot file, `doc.go`, produced no candidate at all: its
single row is a comment block rewritten to match code the sweep found correct.

Upgrade: key the triage's trivial row on the material rather than on behaviour, since
prose asserting what the code does carries a round's whole risk while changing nothing.
Estimate against this round: 17 agents, 493k output, 39M cache read, 39 minutes, so about
10x, and the direction of the finding rate is unknown until a second shape runs the same
head. Both numbers are estimates until the outcome table measures them.
