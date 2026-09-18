# The node's JSON-RPC interface, and the reference PR 6182 adds for it
claude-opus-5

## What it is for

Every client of a gno.land chain, `gnokey`, `gnoweb`, the JavaScript clients, an
exchange's integration, reaches the chain through one HTTP interface on port
26657. It carries 20 methods: block and commit reads, transaction broadcast,
mempool inspection, and `abci_query`, the door to the application's own query
paths.

## How it works today

One route map, [`Routes`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/core/routes.go#L15), binds each
method name to a handler and to the names of its arguments. Three transports
serve that map: `GET /<method>?<arg>=<value>`, `POST /` with a JSON-RPC object,
and `/websocket`. Results are marshalled with Amino JSON, which base64s every
byte array and quotes every 64-bit integer, so `"height": "51942"` sits beside
`"index": 0` in one response.

What existed as documentation before this branch described a different system.
The package comment listed `/dial_seeds` and `/dial_persistent_peers`, which
[`Routes`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/core/routes.go#L15) does not register, and the
package README documented `?page` and `?per_page` pagination that no handler
reads.

## What the change does

It adds `docs/resources/rpc-endpoints.md`, 260 lines, one entry per method with
what it takes and what it returns, plus two cross-cutting sections: how to read
an Amino JSON response, and what the interface does not provide. It deletes the
package README and rewrites the package comment, corrects the Protobuf and gRPC
claims in two older pages, and links the new page from the docs index and the
sidebar. No behaviour changes: every Go line it touches is a comment.

## Concepts

| Term | What it means here |
|---|---|
| Amino JSON | The encoding of every result. Byte arrays go out base64, 64-bit integers as quoted strings, narrower integers as bare numbers. |
| URI transport | `GET /<method>?<arg>=<value>`. Looser than JSON-RPC about arguments: it accepts a bare value and a `0x`-prefixed hex string. |
| `abci_query` | The one method that reaches the application. Its `path` selects a module, and a failed query reports at `response.ResponseBase.Error` rather than at the top level. |
| `rpc.unsafe` | A node setting, off by default, that registers four extra methods: a mempool flush and three profiler calls. |
