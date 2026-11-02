# `rpc.allow_code_execution`
Written by claude-opus-5-5.
PR: [gnolang/gno#6413](https://github.com/gnolang/gno/pull/6413)

## TLDR

Anyone who can reach a public node can ask it to run code, for free: a few read-only queries evaluate an expression in a package or test a whole transaction on the node that answers. This adds one node setting that turns those queries off while keeping every other query and transaction submission working. It is on by default, so nothing changes until an operator turns it off.

## Queries that run code

Most `abci_query` paths read stored state. Four of them run code: `vm/qeval`, `vm/qeval_json` and `vm/qrender` evaluate an expression in a deployed package, and `.app/simulate` runs a whole transaction against the last committed state and throws the result away. A query pays no fee, so the node that answers pays for that work with its own CPU and memory.

The three VM queries are the only callers of `withQueryEvalMachine`, the one place a query builds a VM machine. The other VM queries, `qfile`, `qdoc`, `qfuncs`, `qpaths`, `qstorage` and the JSON exports, read stored state and build none. `.app/simulate` matters as much as `vm/qeval`: a simulated `MsgRun` carries a script, so refusing `vm/qeval` alone would leave the same code one query away.

## How a query path reaches its handler

An `abci_query` call carries a path, and the application routes it by segments. `BaseApp.Query` splits the path on `/` and drops one empty leading segment, so `/vm/qeval` and `vm/qeval` both reach the VM module. `.app` and `.store` it answers itself; any other first segment picks a module. The VM module splits the path again and needs `vm` first, so `/vm/qeval` fails there, while `vm/qeval`, `vm/qeval/x` and `vm/qeval?x` all reach the same handler.

The new check reads a path the same way: it drops one leading `/`, cuts at `?`, and refuses each listed path and every path below it. A spelling that would reach one of these handlers is therefore refused, and a few that the application would reject anyway, `/vm/qeval` among them, are refused earlier.

## Where the switch sits

Every RPC transport answers `abci_query` through one method, `Environment.ABCIQuery`: a URI GET, a JSON-RPC POST, each call inside a batch, a WebSocket call and the in-process client. The check sits there, before the query reaches the application, so a proxy filtering URLs is not needed and could not see inside a batch or a WebSocket frame anyway.

Nothing inside a block calls `ABCIQuery`, so the setting changes no block, no gas figure and no app hash. Two nodes with different settings stay in consensus.

## Who owns the list

tm2 knows nothing about the VM, so the list is split. tm2 refuses `.app/simulate` on its own, since `BaseApp` answers it. The VM package publishes its three paths as `vm.CodeExecutionQueryPaths`, and `gnoland` hands that list to the node through the new option `node.WithCodeExecutionQueryPaths`, both in `gnoland start` and in the in-memory node. A VM query added later that runs code gets refused once it is added to that list.

## What a refused call looks like

A refused call returns an error naming the path and the setting, for example `query "vm/qeval" runs code, which this node does not allow over RPC (rpc.allow_code_execution = false)`. The URI transport answers HTTP 403; the other transports carry the same text in the JSON-RPC error. In a batch, only the refused call fails and the others are answered.

## Upgrading

The setting lives in the `[rpc]` section of `config.toml` and defaults to `true`. A config file written before the setting existed has no such key and loads `true`, so upgraded nodes keep serving every query. Set it to `false` on a node that only takes transactions: `broadcast_tx_*` still works, since accepting a transaction checks its signature, sequence and fee and runs none of its messages, but `gnokey maketx` simulates first by default and needs `-simulate skip`. Keep `true` on any node behind gnoweb, which renders every page through `vm/qrender`.

## Words used here

| Name | What it is |
| --- | --- |
| `abci_query` | The RPC method that sends a read-only query to the application, carrying a `path` and `data`. |
| `rpc.allow_code_execution` | The new boolean node setting in `[rpc]`; default `true`, and `false` refuses the four paths below. |
| `vm/qeval`, `vm/qeval_json` | Evaluate an expression inside a deployed package and return the value, as text or JSON. |
| `vm/qrender` | Evaluates a package's `Render(path)`; gnoweb uses it for every page. |
| `.app/simulate` | Runs a transaction without committing it; `gnokey maketx` sends it before broadcasting. |
| `vm.CodeExecutionQueryPaths` | The VM's list of query paths that build a machine, handed to the node at startup. |
| `node.WithCodeExecutionQueryPaths` | The node option that passes that list to the RPC environment. |
| `Environment.ABCIQuery` | The one method every RPC transport calls for `abci_query`, where the check runs. |

<details>
<summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a generated file. This change has no `hot` file. Every excerpt below is the after side, at the reviewed head.

### 1. `warm` · The check, [`tm2/pkg/bft/rpc/core/abci.go`](https://github.com/gnolang/gno/pull/6413/files#diff-ef35cb2ff0f536334634c02b4b8b46f842a82bfc995b7964fbd7f7ca5083837aR29-R34)
Refuses a query that runs code before it reaches the application.
**If wrong:** a node set to `false` still runs code for anyone, or refuses queries that run none.
```go
	// A query pays no fee, so code it runs costs only this node.
	// A refused query never reaches the app, so the setting cannot change a block.
	if !env.Config.AllowCodeExecution && env.runsCode(path) {
		return nil, codeExecutionRefusedError(path)
	}
```

### 2. `warm` · The path match, [`tm2/pkg/bft/rpc/core/abci.go`](https://github.com/gnolang/gno/pull/6413/files#diff-ef35cb2ff0f536334634c02b4b8b46f842a82bfc995b7964fbd7f7ca5083837aR48-R56)
Reads the path the way the application routes it, so every spelling of a listed path is refused.
**If wrong:** `vm/qeval?x` or `.app/simulate/x` slips past the check and runs.
```go
func (env *Environment) runsCode(path string) bool {
	path, _, _ = strings.Cut(strings.TrimPrefix(path, "/"), "?")
	under := func(entry string) bool { return path == entry || strings.HasPrefix(path, entry+"/") }
	return under(simulateQueryPath) || slices.ContainsFunc(env.CodeExecutionQueryPaths, under)
}
```

### 3. `warm` · The VM's list, [`gno.land/pkg/sdk/vm/handler.go`](https://github.com/gnolang/gno/pull/6413/files#diff-822bd9f7cad17ee97db5ce10bac9b94126477e2840fcef438c5c437f549bd444R117-R124)
Names the three VM queries that build a machine.
**If wrong:** a VM query that runs code is served on a node set to `false`.
```go
var CodeExecutionQueryPaths = []string{
	RouterKey + "/" + QueryRender,
	RouterKey + "/" + QueryEval,
	RouterKey + "/" + QueryEvalJSON,
}
```

### 4. `warm` · The wiring, [`gno.land/cmd/gnoland/start.go`](https://github.com/gnolang/gno/pull/6413/files#diff-a723d42a46bf700d9a3ff5e6e2aba1fc7216477d25721f318d0975ff530c71c0R297)
Hands the VM's list to the node that `gnoland start` builds; the in-memory node does the same.
**If wrong:** `false` refuses `.app/simulate` alone and the VM queries keep running.
```go
	opts := []node.Option{node.WithCodeExecutionQueryPaths(vm.CodeExecutionQueryPaths...)}
```

### 5. `warm` · The default, [`tm2/pkg/bft/rpc/config/config.go`](https://github.com/gnolang/gno/pull/6413/files#diff-5d44a98654f3b0fc356a4e731f46cd96d61f919dbd1e200d5336c92c297e5615R119)
Keeps every existing node serving every query after upgrade.
**If wrong:** gnoweb stops rendering on every node that upgrades.
```go
		AllowCodeExecution: true,
```

</details>
