# PR [#6182](https://github.com/gnolang/gno/pull/6182): docs: add a JSON-RPC endpoint reference, drop the parts that were false
Verdict: APPROVE, on two low-band findings and no Warning: a pagination validator no handler calls, and two config comments describing what this page lists as absent. The page's readability findings are already raised on the target and ship SKIP.
Event: APPROVE
Model: claude-opus-5, standard review
Commit: 6729f335f
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6182 6729f335f`
Round: 1. One pass over 457 changed lines as finder, judge and writer, every finding run from the head worktree. The two findings on Go files ship in the Body: the line each asks to edit sits outside the diff.

## Body

- Related suggestion: nothing outside [`pipe_test.go`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/core/pipe_test.go#L45) calls [`validatePage`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/core/pipe.go#L123), the code behind the `?page` and `?per_page` this page lists as absent, so it can be deleted.
- Related nit: [`config.go`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/config/config.go#L38) describes `grpc_laddr` as the address of a gRPC server and [`unsafe`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/config/config.go#L48) as activating `/dial_seeds`, both in `comment:` tags, so every generated `config.toml` carries two claims this page lists as absent.

<details>

<summary>Sweep</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6182 -R gnolang/gno
grep -rn validatePage tm2 | grep -v _test
```

One hit, the definition:

```text
tm2/pkg/bft/rpc/core/pipe.go:123:func validatePage(page, perPage, totalCount int) (int, error) {
```

Deleting it touches `pipe.go:123` and the `validatePage` cases in `pipe_test.go`. The two constants above it stay: `validatePerPage` reads both, and `mempool.go` calls that one for the `unconfirmed_txs` limit.

</details>

## SKIP docs/resources/rpc-endpoints.md:131 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L131) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L131) · Nit

Already raised: https://github.com/gnolang/gno/pull/6182#discussion_r4057657792

Nit: neither `tx` nor `abci_query` links to *Passing byte arguments*, so a reader landing on either meets neither the `%2B` rule nor the `0x` one.

## SKIP docs/resources/rpc-endpoints.md:38 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L38) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L38) · Suggestion

Already raised: https://github.com/gnolang/gno/pull/6182#pullrequestreview-5261279085

Suggestion: the four endpoint groups open on a method heading, where the page's other sections each open with a sentence saying what they hold.

<details>

<summary>Sweep</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6182 -R gnolang/gno
awk '/^## /{h=$0; getline; while($0=="") getline; print (/^### /?"###  ":"prose") "  " h}' docs/resources/rpc-endpoints.md
```

```text
prose  ## Calling an endpoint
###    ## Node and network
###    ## Blocks
###    ## Transactions
###    ## Application
prose  ## Reading a response
prose  ## Not available
prose  ## See also
```

</details>

## SKIP docs/resources/rpc-endpoints.md:206 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L206) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L206) · Suggestion

Already raised: https://github.com/gnolang/gno/pull/6182#pullrequestreview-5261279085

Suggestion: the page shows no complete payload, so nobody can check a parser against a real `tx` response.

## SKIP docs/resources/rpc-endpoints.md:45 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L45) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L45) · Nit

Nit: `status` names its return in a table, every other endpoint in a sentence.

Not posted: reshaping every section belongs to the generated layer, not to this branch.

## SKIP docs/resources/rpc-endpoints.md:84 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L84) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L84) · Nit

Nit: the behaviour a caller cannot infer sits in each endpoint's second paragraph, unmarked, so a reader scanning first paragraphs misses it.

Not posted: a marker holding for every endpoint is a pass over the page, not an edit to a line.

## SKIP docs/resources/rpc-endpoints.md:240 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L240) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L240) · Nit

Nit: the `Not available` table's header row is `| | |`, where the page's other tables are headed, so it renders an empty band above the rows.

Not posted: cosmetic, and no enabled linter reads markdown tables.

## SKIP tm2/pkg/bft/rpc/lib/server/handlers.go:242 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/lib/server/handlers.go#L242) · [↗](../../../../../.worktrees/gno-review-6182/tm2/pkg/bft/rpc/lib/server/handlers.go#L242) · Nit

Nit: the argument conversion runs only when `len(req.Params) > 0`, so a client omitting `params` on a method that declares one is answered from the panic recovery.

Not posted: no run confirmed it, and it predates the branch.

## SKIP docs/resources/rpc-endpoints.md:52 [gh](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/docs/resources/rpc-endpoints.md#L52) · [↗](../../../../../.worktrees/gno-review-6182/docs/resources/rpc-endpoints.md#L52) · Nit

Nit: both write deadlines are in the source, 30 seconds at [`http_server.go:47`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/lib/server/http_server.go#L47) and 10 at [`handlers.go:492`](https://github.com/gnolang/gno/blob/6729f335f6214da9c8f96058fa82dbeb6bef130b/tm2/pkg/bft/rpc/lib/server/handlers.go#L492).

Not posted: a mainnet request confirms the page, 122,681,708 bytes and a connection reset at 31 seconds.
