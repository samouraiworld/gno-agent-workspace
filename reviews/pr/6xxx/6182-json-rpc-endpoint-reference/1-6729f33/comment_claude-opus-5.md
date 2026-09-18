# PR [#6182](https://github.com/gnolang/gno/pull/6182): docs: add a JSON-RPC endpoint reference, drop the parts that were false
Verdict: APPROVE. Two low-band findings, no Warning: the page's claims about parameters, defaults, caps and absences hold against the route map and the handlers.
Event: APPROVE
Model: claude-opus-5, standard review, solo shape from the triage
Commit: 6729f335f
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6182 6729f335f`
Round: 1. One agent as finder, judge and writer over 457 changed lines, 4 candidates, 2 confirmed by a run from the head worktree and 2 shipped SKIP.

## Body
Comment-only in Go: `doc.go`, `mempool.go` and `pipe.go` change no statement.

## tm2/pkg/bft/rpc/core/pipe.go:25 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/core/pipe.go#L25) · [↗](../../../../../.worktrees/gno-review-6182/tm2/pkg/bft/rpc/core/pipe.go#L25) · Suggestion

Suggestion: nothing outside [`pipe_test.go`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/core/pipe_test.go#L45) calls [`validatePage`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/core/pipe.go#L123), which implements `?page` and `?per_page` pagination no handler reads, so it can be deleted.

<details>

<summary>Sweep</summary>

```bash
git checkout 6729f335f6214da9c8f96058fa82dbeb6bef130b
grep -rn validatePage tm2 | grep -v _test
```

One hit, the definition:

```text
tm2/pkg/bft/rpc/core/pipe.go:123:func validatePage(page, perPage, totalCount int) (int, error) {
```

Removing it touches `pipe.go:123` and the `validatePage` cases in `pipe_test.go`.

</details>

## docs/resources/rpc-endpoints.md:245 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L245) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L245) · Nit

Related nit: [`config.go`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/config/config.go#L38) still describes `grpc_laddr` as the address of a gRPC server and [`unsafe`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/config/config.go#L48) as activating `/dial_seeds`, so every generated `config.toml` carries both claims.

<details>

<summary>Sweep</summary>

```bash
git checkout 6729f335f6214da9c8f96058fa82dbeb6bef130b
grep -rn GRPCListenAddress tm2 gno.land --include='*.go'
```

Config plumbing, a WAL generator and the config get and set tests. Nothing starts a listener:

```text
tm2/pkg/bft/rpc/config/config.go:38,101,123
tm2/pkg/bft/consensus/wal_generator.go:38
gno.land/cmd/gnoland/config_set_test.go:634
gno.land/cmd/gnoland/config_get_test.go:734,742
```

Both strings are `comment:` struct tags, so both are written into every generated `config.toml`.

</details>

## SKIP tm2/pkg/bft/rpc/lib/server/handlers.go:242 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/lib/server/handlers.go#L242) · [↗](../../../../../.worktrees/gno-review-6182/tm2/pkg/bft/rpc/lib/server/handlers.go#L242) · Nit

Nit: the argument conversion runs only when `len(req.Params) > 0`, so a client omitting `params` on a method that declares one is answered from the panic recovery. Not posted: no run confirmed it, and it predates the branch.

## SKIP docs/resources/rpc-endpoints.md:52 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L52) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L52) · Nit

Nit: both write deadlines are in the source, 30 seconds at [`http_server.go:47`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/lib/server/http_server.go#L47) and 10 at [`handlers.go:492`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/lib/server/handlers.go#L492). Not posted: a mainnet request confirms the page, 122,681,708 bytes and a connection reset at 31 seconds.
