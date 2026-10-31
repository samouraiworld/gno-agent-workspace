# Gno.land Overview

Measured against [gnolang/gno at 5a51b55f4][gno-tree], 2026-10-30, the commit
[How the GnoVM works](gnovm-architecture.md) is pinned to. Every code link
points at that commit. Network facts and external links were checked on
2026-11-01 and carry that date.

## TLDR

Gno is Go, interpreted deterministically on a blockchain. A realm is a package
whose global variables persist between transactions and which can render its
own web page as markdown. The node runs Tendermint2, a minimal fork of the
Tendermint consensus engine; the interpreter is the GnoVM; a keeper between the
two runs the interpreter for each message. Mainnet `gnoland-1`, launched on
2026-09-12, is live at gno.land, the testnet `onyx-1` rehearses each of its
upgrades, and `staging` is rebuilt from `master`. Governance is a vote of
members in three tiers with a two-thirds rule, and GNOT pays for gas and,
refundably, for stored bytes.

## What is Gno?

**Gno** is a deterministic, interpreted dialect of Go for smart contracts on
the **gno.land** blockchain. It [targets the Go 1.17 language
specification][compat-go117]: no generics, and no goroutines or channels yet.
The language adds two builtin types, [`address`][a-uv-address] and
[`realm`][a-uv-realm], and the builtin [`cross`][a-uv-cross]. [A crossing
function](gnovm-architecture.md#crossing-functions-and-cur) is one whose first
parameter has type `realm`, [the only kind a transaction can
call][a-call-crossing], and `cross(cur)` in a call to one moves to the callee
realm's identity. The interpreter is the GnoVM, described end to end in [How
the GnoVM works](gnovm-architecture.md).

- **Deterministic**: no goroutines or channels, no clock, network or file
  system; floats run in software; maps iterate in insertion order, [among the
  rules that keep validators in
  step](gnovm-architecture.md#why-it-is-deterministic).
- **Interpreted**: source runs on the GnoVM. Nothing is compiled to machine
  code, and [the source is what is stored on
  chain](gnovm-architecture.md#the-keeper-the-store-and-the-machine).
- **Auditable**: every deployed package is readable on the chain's own web
  interface, [tests included][a-store-tests].
- **Persistent by default**: the global variables of a realm are [saved when a
  call returns](gnovm-architecture.md#realms-and-persistence), without a
  database layer or a storage API.

File extension `.gno`. Module manifest [`gnomod.toml`][gnomod-example], which
names the module path and the Gno language version, [`0.9` at this
commit][gnover].

![The three package kinds, what each may do, and who may deploy under a
namespace.](figures/package-kinds.svg)

### Toolchain

Install from source:

```bash
git clone https://github.com/gnolang/gno.git && cd gno && make install
```

[`make install`][a-make-install] builds the three daily tools, `gno`, `gnokey`
and `gnodev`. Two more run underneath, the node `gnoland` and the web server
`gnoweb`, and [`make -C gno.land install`][a-make-node] builds them.

**`gno`** is the language tool. It tests, lints, formats and documents `.gno`
packages without a node, the way the `go` command does for Go. Its subcommands
at this commit, [from the binary's own help][gno-help], with what each runs [in
the GnoVM doc](gnovm-architecture.md#tooling):

| Command | Purpose | Go equivalent |
| --- | --- | --- |
| `gno test` | Run `_test.gno` functions, [`Example` functions whose body ends in an `// Output:` comment][a-gno-test-example], and `_filetest.gno` files, programs whose comments hold their expected results | `go test` |
| `gno fmt` | Format `.gno` files and sort their imports | `go fmt` |
| `gno lint` | Check types and compile without running | `go vet` |
| `gno doc` | Show a package's documentation, [asking a node's `vm/qdoc`][a-gno-doc-remote] when the package is not local | `go doc` |
| `gno run` | Run a package's `main` or an expression | `go run` |
| `gno fix` | Rewrite source written against an older Gno API | `go fix` |
| `gno mod` | Module maintenance, `gno mod init` among others | `go mod` |
| `gno repl` | Start a GnoVM read-eval-print loop | none |
| `gno tool transpile` | Translate Gno to Go source | none |
| [`gno tool deplist`][a-gno-deplist] | List a package's dependencies, each before the packages that import it | none |
| `gno list`, `gno env`, `gno clean`, `gno bug`, `gno version` | Housekeeping | the same |

**`gnodev`** is the local development server: an in-memory `gnoland` node and
a `gnoweb` front end in one process, with hot reload. Write code, `gnodev`
redeploys it, open `http://localhost:8888` to see the realm rendered. At
startup it [loads the current directory's package][a-gnodev-workspace], or the
whole workspace when a `gnowork.toml` sits in it or in a directory above. It
loads a package from the monorepo's `examples/` [the first time a query or
transaction names it][a-gnodev-examples], unless
[`-no-examples`][a-gnodev-noexamples] is passed.

```bash
gnodev ./myrealm    # gnoweb at http://localhost:8888
```

| Feature | What it does |
| --- | --- |
| Hot reload | [Watches every loaded package's directory][a-gnodev-watch], redeploys on change |
| State maintenance | [Replays every transaction after a reload][a-gnodev-replay], so state survives |
| Premined balances | Each local `gnokey` key [starts with ten million GNOT][gnodev-premine]; `-add-account` funds another |
| State export | [Writes the current state as a genesis file][a-gnodev-export], the file a chain starts from |
| On-chain dependencies | [`-remote gno.land=https://rpc.gno.land:443`][a-gnodev-remote] fetches from that node an import [no local directory holds][a-gnodev-local-first], to test local code against deployed packages |

Keys while it runs [in a terminal][a-gnodev-interactive], [from its
README][gnodev-keys]: `H` help, `A` account balances, `R` reload, `P` and `N`
step back and forward through the transactions, `Ctrl+S` save the state,
`Ctrl+R` reset to the saved or initial state, `E` export a genesis file,
`Ctrl+C` exit.

**`gnokey`** manages keys and talks to a node over RPC: deploy code, call
functions, send coins, query state. Addresses are bech32, a checksummed text
encoding, [with the `g1` prefix][a-bech32]. `gnodev` deploys [from a default
address][a-gnodev-deploykey],
[`g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5`][default-account], and [prints its
mnemonic at startup][a-gnodev-mnemonic] when no local key holds it. A mnemonic
is the word list a key is derived from, so `gnokey add -recover` with it gives
a local key that signs as the default address.

```bash
gnokey add MyKey
# Save the mnemonic. Note your g1... address.
```

| Message | Command | Purpose |
| --- | --- | --- |
| [`MsgAddPackage`][a-msg-addpkg] | `gnokey maketx addpkg` | Deploy a package |
| [`MsgCall`][a-msg-call] | `gnokey maketx call` | Call one exported crossing function |
| [`MsgSend`][a-msg-send] | `gnokey maketx send` | Transfer coins |
| [`MsgRun`][a-msg-run] | `gnokey maketx run` | Run a throwaway `main` against on-chain code |
| [`MsgEnablePackage`][a-msg-enable] | `gnokey maketx enablepkg` | Approve a package waiting for review, naming the hash of the source read |
| [`MsgRejectPackage`][a-msg-reject] | `gnokey maketx rejectpkg` | Remove a package waiting for review |
| [`MsgCreateSession`][a-msg-session], [`MsgRevokeSession`][a-msg-session-revoke], [`MsgRevokeAllSessions`][a-msg-session-revokeall] | `gnokey maketx session create`, `revoke`, `revokeall` | [Session accounts][gnokey-session]: a second key that may sign for the master account until `-expires-at`, only the messages `-allow-paths` lists, spending nothing past `-spend-limit`; it signs [with `-master`][a-master] naming that account |

The vm parameter [`code_submission_policy`][a-code-policy] decides whether an
upload runs at once or [waits for an
approver](gnovm-architecture.md#deploying-through-approval); read a network's
value with `gnokey query params/vm:p:code_submission_policy -remote <rpc>`.

Every `maketx` command [takes][a-maketx-flags] `-gas-fee` and `-gas-wanted`,
`-chainid` and `-remote` to pick the network, and `-simulate`; it signs and
broadcasts by default. `-simulate only` [prints the gas and the fee the
transaction needs without broadcasting it][a-simulate-only]. `addpkg`, `call`
and `run` also take [`-send` for attached coins and `-max-deposit` to cap the
storage deposit][addpkg-flags].

#### Query paths

```bash
gnokey query <path> -data "<data>" -remote "https://rpc.gno.land:443"
```

The `vm` paths are [the fourteen cases of the keeper's query
handler][query-kinds]; the others belong to the `auth`, `bank` and `params`
modules. `vm/qpaths` and `vm/qinertpaths` take `?limit=<n>` on the path, [a
thousand by default and ten thousand at most][a-paths-limit].

| Path | Data | Returns |
| --- | --- | --- |
| `vm/qrender` | `<pkgpath>:<renderpath>` | The realm's `Render` output, markdown |
| `vm/qeval` | `<pkgpath>.<expression>` | Any expression, as text |
| `vm/qeval_json` | `<pkgpath>.<expression>` | The same, as JSON |
| `vm/qfuncs` | `<pkgpath>` | Exported function signatures, JSON |
| `vm/qfile` | `<pkgpath>` or `<pkgpath>/<file>` | File list, or one file's source |
| `vm/qdoc` | `<pkgpath>` | Package documentation, JSON |
| `vm/qpaths` | `<prefix>`, or `@<name>` | Deployed package paths under the prefix, [or under a registered name][a-qpaths-user] |
| `vm/qstorage` | `<pkgpath>` | Bytes stored and deposit held by a realm |
| `vm/qpkg_json` | `<pkgpath>` | The package-level variables, names and values, JSON |
| `vm/qobject_json`, `vm/qobject_binary` | an object id | One stored object, as JSON or as raw bytes |
| `vm/qtype_json` | a type id such as `gno.land/p/nt/avl/v0.Tree` | One type's definition, JSON |
| `vm/qpkgmeta_json` | `<pkgpath>` | Whether the path holds a live package, one waiting for review, or nothing, with its creator |
| `vm/qinertpaths` | `<prefix>` | Paths of packages waiting for review |
| `auth/accounts/<addr>` | | Account: coins, public key, sequence |
| `auth/accounts/<addr>/sessions` | | [The account's session keys][a-auth-queries] |
| `auth/gasprice` | | The current minimum gas price |
| `bank/balances/<addr>` | | Coin balances |
| `bank/supply/<denom>` | | [Total supply of one coin][a-bank-supply] |
| `params/<module>:p:<key>` | | One live chain parameter, such as `vm:p:storage_price` |

**`gnoland`** is the node: consensus, blocks, persistence. [`gnoland
start`][a-gnoland-start] runs one. In development `gnodev` runs it for you.

**[`gnoweb`][a-gnoweb]** renders realm markdown and browses on-chain source. It
is what gno.land and [`gnodev`][a-gnodev-web] serve.

![From an editor to a rendered page: the loop a realm goes through, and which
tool runs each step.](figures/dev-workflow.svg)

## Core Concepts

### Realms (`r/`)

Stateful contracts with a unique path such as
[`gno.land/r/demo/counter`][a-counter]. A realm has [an address derived from
its path][a-realm-addr], can hold coins, and can expose
`Render(path string) string`, whose markdown [gnoweb turns into the realm's
page][a-qrender]. That is how an on-chain application ships a user interface
without a separate front end.

```go
func Render(path string) string {
    return "# Hello World\n\nThis is **markdown** rendered on-chain."
}
```

A public realm takes one upload for good: [the keeper refuses a package at a
path that holds a public one][keeper-exists]. Upgrade by deploying the next
version at another path, [`/v1` after `/v0`][a-pr-6162], or behind a realm
that forwards to a replaceable implementation, [as `r/gov/dao`
does][a-govdao-proxy]. The exception is a package whose manifest says
`private = true`: it [must be a realm][keeper-private-realm], [only its creator
may upload it again][keeper-private], the replacement [must stay
private][a-stay-private], and other realms may not store [its
objects][a-private-objects] or [values of its types][a-private-types].

### Pure packages (`p/`)

Stateless libraries, [`gno.land/p/nt/avl/v0`][a-avl] for instance. Their
package-level values [cannot change once deployed][a-p-immutable], they
[cannot declare crossing functions][a-crossing-allowed], and outside their
tests they [cannot import a realm][p-imports-r]. Because their code never
changes, two realms can trust the same `p/` code to mediate between them.

### Ephemeral packages (`e/`)

`gnokey maketx run` uploads a throwaway `main` package at
[`gno.land/e/<address>/run`][epath], runs it once and [never saves
it][a-run-nosave]. A user runs one to make several calls in one transaction,
or to pass an argument [`maketx call` cannot write][a-call-convert], a struct
for one. The vm parameter `run_submitters` [can limit who sends
it][a-run-submitters], and an empty list lets anyone; read a network's value
with `gnokey query params/vm:p:run_submitters -remote <rpc>`.

### Namespaces

The namespace is the segment after the kind letter. [Two shapes grant deploy
authority][namespace-rule]: the deployer's own address,
`gno.land/r/g1abc.../myrealm`, always; and a registered name,
`gno.land/r/nym-alice123/myrealm`, when [`r/sys/users`][a-sys-users] maps that
name to the deployer as their current name. The keeper [asks
`r/sys/names`][a-names-call] on every upload. The check binds once [`Enable`
has run][a-names-enable], [a one-way switch][a-names-oneway], and before that
[every upload passes][a-names-bypass]. Mainnet [runs `Enable` in its
genesis][a-mainnet-names], [as onyx does][a-onyx-names]; on any other chain,
this query reads which state it is in:

```bash
gnokey query vm/qeval -data "gno.land/r/sys/names.IsEnabled()" -remote <rpc>
```

Names are registered through [`r/sys/namereg/v0`][namereg], in [the form
`nym-`, then 5 to 13 lowercase letters, then three digits][a-nym-format]. Any
other name [comes from a GovDAO proposal][a-propose-name], a vote of the
members who govern the chain. Registration is [free in the
code][register-price], GovDAO [can set a price][a-register-price-prop], and
[the realm's page shows the current one][a-namereg-page].

The keeper also [asks `r/sys/cla`][a-cla-check] whether the uploader signed
the contributor license agreement, a licence a deployer signs once in that
realm. [Its page says whether that check is on][a-cla-page].

### Gas and storage

Two costs, both in `ugnot`, [one millionth of a GNOT][a-ugnot].

- **Gas** pays for work and is spent. It is charged for each opcode,
  allocation and store access, and for each source byte a message uploads;
  the mechanism is in [the GnoVM doc](gnovm-architecture.md#gas-and-memory).
  The whole `-gas-fee` is taken whatever the transaction consumed, [as the
  fee doc states][gas-charged] and [issue 3805][issue-3805] tracks.
- **Storage deposit** pays for bytes and comes back. Growing a realm by
  `delta` bytes [locks `delta × storage_price`][a-deposit-lock] from the
  message's caller into the realm's deposit address, [an address computed
  from the realm's path][a-deposit-addr]. Freeing bytes returns to the caller
  [the deposit in proportion to the bytes freed][a-refund-ratio], whatever the
  price is then. The code default is [100 ugnot per byte][storage-price]: one
  GNOT buys ten kilobytes, and the whole supply of 1.333 billion GNOT buys
  [13.33 terabytes, the figure the Constitution states][a-constitution-tb].

![Four kinds of work feed the transaction's gas meter. The storage deposit is
paid in coins beside it and comes back.](figures/gas-sources.svg)

### Learn by example

The [`r/docs`][r-docs] realms are working examples, one topic each, readable,
deployable and modifiable: `hello`, `adder`, `minisocial`, `avl_pager`,
`routing`, `events`, `safeobjects`, `soliditypatterns`, `txlink`, `charts`,
`markdown` and more, indexed by [`r/docs/home`][a-docs-home]. Most of them sit
under `examples/quarantined/`, which [the examples README
describes][a-quarantined] as kept out of the audited set and not updated to
the current rules for calls between realms. Check their crossing calls against
[Interrealm programming](#interrealm-programming) before copying them.

`gnodev` serves them at `http://localhost:8888/r/docs/home`, [loading each from
`examples/` the first time it is opened][a-gnodev-examples]. `vm/qpaths` with
the prefix `gno.land/r/docs` tells whether a network carries them.

### Good practices

On-chain code is permanent and public. The rules that follow from it:

- **Every write costs gas and bytes.** Batch writes, and free what you no
  longer need to get the deposit back.
- **Never put a secret in code.** Source is public, tests included.
- **Test before deploying.** `gno test` for logic, `gnodev` for the rendered
  result. A public package's path takes one upload for good.
- **Panics roll back the whole transaction.** [Nothing the messages wrote is
  kept](gnovm-architecture.md#the-keeper-the-store-and-the-machine). Use
  panics for access control and invalid input, not to report recoverable
  errors.
- **Validate every argument.** Anyone can call an exported crossing function.
- **Use `ownable`** from [`p/nt/ownable/v0`][a-ownable] rather than a
  hand-written admin check.
- **Prefer [`avl.Tree`][a-avl] to a map for anything that grows.** [A map is
  one stored object][a-map-object], loaded and saved whole; a tree loads and
  writes only the path to one key, iterates in key order and pages through
  [`p/nt/avl/pager/v0`][pager]. Effective Gno [puts the crossover near 250
  entries][a-map-crossover].
- **Keep `Render` cheap.** It runs on every page view, [under a query gas
  limit][a-query-limit].
- **Read the caller from `cur`.** `cur.Previous()` is the caller of a crossing
  function and [cannot be forged from a
  helper](gnovm-architecture.md#crossing-functions-and-cur). The stack-walking
  [`unsafe` package][unsafe-doc] is for event emission or fee attribution,
  where transaction-level identity is what you want.

[Effective Gno](https://docs.gno.land/resources/effective-gno) goes deeper:
global variables, `init` as a constructor, package layout, safe objects, coins
versus GRC-20 tokens.

## Networks

Every network has a chain id and an RPC endpoint. The chain id is part of
what each transaction [signs][b-chainid-sign], so a signed transaction cannot
be replayed on another network. The RPC endpoint is the address `gnokey`
reaches through `-remote`. From [the networks doc][b-networks-doc], checked
on 2026-11-01:

| Network | Chain id | Web | RPC | Purpose |
| --- | --- | --- | --- | --- |
| Mainnet | `gnoland-1` | [gno.land](https://gno.land) | `https://rpc.gno.land:443` | The production chain, [launched on 2026-09-12][b-mainnet-launch]. [No free test tokens][b-mainnet-nofaucet]. |
| Onyx | `onyx-1` | [onyx.testnets.gno.land](https://onyx.testnets.gno.land) | `https://rpc.onyx.testnets.gno.land:443` | The testnet that tracks mainnet: it [runs mainnet's code and takes each mainnet release first][b-onyx-line]. [Free test tokens and open transfers][b-onyx-open]. |
| Staging | `staging` | [staging.gno.land](https://staging.gno.land) | `https://rpc.staging.gno.land:443` | A rolling testnet rebuilt from `master` |
| Local | `dev` | `http://localhost:8888` | `http://127.0.0.1:26657` | [`gnodev`'s defaults][b-gnodev-defaults] |

Betanet, chain id `gnoland1`, is retired: mainnet [started from a fresh state,
not from betanet's][b-mainnet-fresh], and [`rpc.gno.land` serves
`gnoland-1`][b-rpc-mainnet]. Pearl, Sapphire, Topaz and the numbered test
networks [are retired too][b-testnets-retired]. Onyx's test tokens come from
[faucet.gno.land](https://faucet.gno.land), a site that hands them out free.

A chain's genesis is the state it starts from. [Mainnet's][b-mainnet-inert]
and [Onyx's][b-onyx-inert] genesis set two rules that change how a builder
deploys:

- **An upload waits for approval.** `vm:p:code_submission_policy` starts at
  `inert`, so [a package is stored without
  running](gnovm-architecture.md#deploying-through-approval) until an address
  listed in `vm:p:pkg_approvers` enables it.
- **`gnokey maketx run` is restricted.** `vm:p:run_submitters` starts with a
  list of addresses, and only those may send a run.

Mainnet's genesis also [locks `ugnot` transfers][b-mainnet-lock] through
`bank:p:restricted_denoms`, as [the Constitution][b-constitution-lock]
requires. Onyx's [leaves them open][b-onyx-open].

Staging keeps its history through rebuilds. When `master` changes, staging
[archives every transaction, starts a new chain from `master` and replays the
archive into it][staging-cycle]. [Every package under `examples/` is deployed
first][b-staging-examples], ahead of the replayed uploads. A transaction that
fails under the new code [is lost in the replay][b-staging-warning], and
heights and timestamps cannot be relied on.

![The staging chain's rebuild loop: archive, pull master, replay into a new
genesis.](figures/staging-cycle.svg)

### Chain parameters

Governance [changes a chain's parameters by proposal][b-sys-params], so a
network's value can differ from the code default. Read one with `gnokey query
params/<module>:p:<key>`:

```bash
gnokey query params/vm:p:storage_price -remote "https://rpc.gno.land:443"
```

| Parameter | What it sets | Code default |
| --- | --- | --- |
| `vm:p:storage_price` | [The deposit locked per byte a realm grows](gnovm-architecture.md#storage-deposit) | [`100ugnot`][storage-price] |
| `vm:p:default_deposit` | [The most one message may lock when it sets no `-max-deposit`][b-deposit-cap] | [`100000000ugnot`][vm-params-defaults] |
| `auth:p:initial_gasprice` | [The floor of the gas price][b-gasprice-floor], which [rises after each block that uses more gas than its target][b-gasprice-rise] | [`1ugnot/1000gas`][b-initial-gasprice] |
| `bank:p:restricted_denoms` | Denominations [only allow-listed accounts may send][b-restricted-check] | [none][b-restricted-default] |
| `vm:p:code_submission_policy` | [Who may upload a package, and whether it waits for approval][b-policy-values] | [`permissionless`][b-policy-default] |
| `vm:p:run_submitters` | [Who may send `gnokey maketx run`][b-run-submitters-field] | [empty, which lets anyone][b-run-submitters] |
| `vm:p:inert_submission_charge` | [A flat charge on each upload that waits for approval][b-inert-charge-field] | [empty, which charges nothing][b-inert-charge] |
| `vm:p:chain_domain` | [The domain every uploaded package path must start with][b-chain-domain-check] | [`gno.land`][b-chain-domain] |

When `bank:p:restricted_denoms` lists `ugnot`, [a storage-deposit refund goes
to the chain's storage fee collector address instead of the
caller][b-refund-receiver]. A transaction [is refused when `-gas-fee` divided
by `-gas-wanted` falls below the current gas price][b-fee-check].
`gnokey query auth/gasprice -remote <rpc>` [reads that price][b-gasprice-query].

## Quickstart on the Onyx testnet

Every command targets Onyx, `onyx-1`. Replace `<your-address>` with the
`g1...` address `gnokey add` prints. Copy, paste, run.

**1. Create a key**

```bash
gnokey add MyKey
```

**2. Get test tokens** at [faucet.gno.land](https://faucet.gno.land), then
check them:

```bash
gnokey query bank/balances/<your-address> \
  -remote "https://rpc.onyx.testnets.gno.land:443"
```

**3. Write a realm**

```bash
mkdir myapp && cd myapp
gno mod init gno.land/r/<your-address>/myapp
```

Deploying under your own address needs no registration. A registered name such as
`gno.land/r/nym-alice123/...` needs [the name registered to you first](#namespaces).

`myapp.gno`:

```go
package myapp

import "gno.land/p/nt/ufmt/v0"

var counter int

func Increment(cur realm) { counter++ }

func Render(path string) string {
    return ufmt.Sprintf("# Counter\n\nValue: **%d**", counter)
}
```

`Increment` takes `cur realm`, which makes it a crossing function, the only
kind a transaction can call. `Render` is read-only and takes none.

**4. Test locally**

`myapp_test.gno`:

```go
package myapp

import (
    "strings"
    "testing"
)

func TestIncrement(cur realm, t *testing.T) {
    Increment(cross(cur))
    if !strings.Contains(Render(""), "**1**") {
        t.Fatal(Render(""))
    }
}
```

A test that calls a crossing function takes `cur realm` itself and crosses
with `cross(cur)`, [as the counter example does][b-counter-test].

```bash
gno test .
gnodev .    # http://localhost:8888/r/<your-address>/myapp
```

**5. Deploy**

```bash
gnokey maketx addpkg \
  -pkgpath "gno.land/r/<your-address>/myapp" -pkgdir "." \
  -gas-fee 8000ugnot -gas-wanted 8000000 -max-deposit 10000000ugnot \
  -broadcast -chainid onyx-1 -remote "https://rpc.onyx.testnets.gno.land:443" \
  MyKey
```

`-max-deposit` [caps the storage deposit the package may lock][b-max-deposit].
Run with `-simulate only` first: it [prints the gas used, a suggested
`-gas-wanted` and the fee at the current gas price][b-simulate-estimate].

On Onyx the package then waits for approval, and [its deposit is taken from
you when it is enabled][b-enable-deposit]. Check where it stands:

```bash
gnokey query vm/qpkgmeta_json \
  -data "gno.land/r/<your-address>/myapp" \
  -remote "https://rpc.onyx.testnets.gno.land:443"
```

`status` [reads `inert` while the package waits for approval and `live` once
it is enabled][b-pkgmeta-status]. Move on once it reads `live`.

**6. Call a function**

```bash
gnokey maketx call \
  -pkgpath "gno.land/r/<your-address>/myapp" -func "Increment" \
  -gas-fee 2000ugnot -gas-wanted 2000000 \
  -broadcast -chainid onyx-1 -remote "https://rpc.onyx.testnets.gno.land:443" \
  MyKey
```

The keeper [writes the call as `pkg.Increment(cross)`][b-call-expr] and [puts
a hidden name for the signer in place of `cross`][call-origin], so
`cur.Previous()` inside `Increment` is your address.

**7. Query**

```bash
gnokey query vm/qrender \
  -data "gno.land/r/<your-address>/myapp:" \
  -remote "https://rpc.onyx.testnets.gno.land:443"
```

**8. Send tokens**

```bash
gnokey maketx send \
  -to "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5" -send "1000000ugnot" \
  -gas-fee 2000ugnot -gas-wanted 2000000 \
  -broadcast -chainid onyx-1 -remote "https://rpc.onyx.testnets.gno.land:443" \
  MyKey
```

## Standard library and notable packages

### The chain packages

Chain access lives in seven standard library packages, all [under
`gnovm/stdlibs`][stdlib-list]; there is no `std` package. The lists below are
the exported names at this commit, and [the GnoVM
doc](gnovm-architecture.md#the-chain-packages) covers each package in depth.

| Package | Exports | Use |
| --- | --- | --- |
| [`chain`][chain-pkg] | `Coin`, `Coins`, `NewCoin`, `NewCoins`, `CoinDenom`, `Emit`, `PackageAddress`, `DerivePkgSubAddr`, `SplitPkgSubPath`, `DeriveStorageDepositAddr`, `PubKeyAddress` | Coins, events, address derivation |
| [`chain/runtime`][runtime-pkg] | `AssertOriginCall`, `ChainID`, `ChainDomain`, `ChainHeight`, `GetSessionInfo`, `Realm`, `NewRealm` | Chain facts, and whether the call came straight from a transaction |
| [`chain/runtime/unsafe`][unsafe-pkg] | `CurrentRealm`, `PreviousRealm`, `OriginCaller`, `OriginSend` | Identity read off the call stack. Named `unsafe` because [a plain helper sees the crossing function below it, not its own caller][c-unsafe-walk]; prefer `cur.Previous()` |
| [`chain/banker`][banker-pkg] | `NewBanker`, `NewReadonlyBanker`, `IsCanonical`, the `Banker` interface, `BankerType` | Moving coins, in [four privilege levels][banker]. `NewBanker` [refuses a realm value that is not current][c-newbanker] |
| [`chain/params`][params-pkg] | `SetString`, `GetString`, and the same pair for `Bool`, `Int64`, `Uint64`, `Bytes` and `Strings` | Parameters of the calling realm alone, [stored under `vm:<realm path>:`][c-params-key] |
| [`sys/params`][c-sysparams-pkg] | `SetSysParamString`, `GetSysParamString`, and the same pairs | Chain-wide parameters. [Only `gno.land/r/sys/params` may call it][c-sysparams-gate] |
| [`chain/markdown`][c-markdown-pkg] | `EscapeInline`, `EscapeTitle`, `StripBidiAndZeroWidth`, and seven more | [Cleaning user text before it reaches markdown output][c-markdown-doc], used through the pure package `sanitize` |
| `testing`, tests only | `SetRealm`, `SetOriginCaller`, `SetOriginSend`, `SetHeight`, `SkipHeights`, `IssueCoins`, `NewUserRealm`, `NewCodeRealm`, `MakeRealm` | [Rewriting the mock context][testing-overrides] inside `gno test` |

The builtin type `address` holds a bech32 `g1...` string. A `MsgCall` passes
an `address` argument through [unchecked, as the string it was
given][c-arg-string], so call [`IsValid()`][c-address-valid] on one before
storing it. The builtin type [`realm`][c-realm-type] is what `cur` is:
`Address()`, `PkgPath()`, `Previous()`, the classifiers `IsUser`, `IsUserCall`,
`IsUserRun`, `IsCode`, `IsEphemeral` and `IsCurrent`, and `Sub` and `Subpath`
for sub-realm identities.

The standard library is every package whose path has no domain. It holds [50
packages at this commit][c-initorder], which [the GnoVM doc sorts into
groups](gnovm-architecture.md#the-packages): Go packages ported whole or in the
part [the compatibility table][c-compat-stdlibs] lists, the chain packages
above, and cryptography down to the curve arithmetic and proof checks that
IBC needs to verify another chain's headers. IBC, Inter-Blockchain
Communication, lets one chain act on another's state.

```
bufio bytes chain chain/banker chain/markdown chain/params chain/runtime
chain/runtime/unsafe crypto/bech32 crypto/bn254 crypto/chacha20
crypto/chacha20/chacha crypto/chacha20/rand crypto/cipher crypto/cometbls
crypto/cometblszk crypto/ed25519 crypto/keccak256 crypto/merkle crypto/modexp
crypto/sha256 crypto/subtle encoding encoding/base32 encoding/base64
encoding/binary encoding/csv encoding/hex errors hash hash/adler32 html
internal/bytealg io math math/bits math/overflow math/rand net/url path
regexp regexp/syntax sort strconv strings sys/params time unicode
unicode/utf16 unicode/utf8
```

`fmt` [exists only under `gno test`][fmt-tests]; on chain use `ufmt`.

### Notable pure packages

Paths are under `gno.land/p/`, in the monorepo's [`examples/`][examples-p].
Each path below ends in its version, and `v0` marks [an initial release not
yet audited][c-v0-banner]. A path in the tree is no evidence that a network
holds it. List what mainnet holds under a prefix with:

```bash
gnokey query vm/qpaths -data "gno.land/p/nt/" -remote "https://rpc.gno.land:443"
```

| Package | Path | What it is |
| --- | --- | --- |
| **avl** | [`nt/avl/v0`][c-p-avl] | The ordered key-value store realms are built on. Lazy-loaded, iterable in key order. |
| **avl pager** | [`nt/avl/pager/v0`][pager] | Pagination over a tree, for `Render`. |
| **bptree** | [`nt/bptree/v0`][c-p-bptree] | A B+ tree, the store behind GovDAO's member tiers. |
| **ufmt** | [`nt/ufmt/v0`][c-p-ufmt] | The on-chain `fmt`. |
| **ownable** | [`nt/ownable/v0`][c-p-ownable] | One owner, transferable. [`AssertOwnedBy(cur.Previous().Address())`][c-ownable-assert] guards a function. |
| **mux** | [`nt/mux/v0`][c-p-mux] | A router for `Render` paths, in the shape of `http.ServeMux`. |
| **sanitize** | [`nt/markdown/sanitize/v0`][c-p-sanitize] | [Escapes user text for each place it can sit in markdown][c-sanitize-doc], so input cannot break a `Render` page. |
| **seqid** | [`nt/seqid/v0`][c-p-seqid] | Sequential ids that sort correctly as tree keys. |
| **uassert**, **urequire** | [`nt/uassert/v0`][c-p-uassert], [`nt/urequire/v0`][c-p-urequire] | Test assertions, the second failing fast. |
| **commondao**, **poa**, **treasury** | [`nt/commondao/v0`][c-p-commondao], [`nt/poa/v0`][c-p-poa], [`nt/treasury/v0`][c-p-treasury] | DAO, proof-of-authority and treasury building blocks under the `nt` namespace. |
| **addrset** | [`moul/addrset/v0`][c-p-addrset] | A set of addresses. |

[`examples/quarantined/`][c-quarantine] holds packages that are neither
audited nor updated to the current interrealm rules, among them
`nt/pausable/v0`, `n2p5/mgroup` and `samcrew/daokit`.

### Token standards

| Standard | Path | Kind |
| --- | --- | --- |
| **GRC-20** | [`gno.land/p/nt/grc20/v0`][c-p-grc20] | Fungible token |
| **GRC-721** | [`gno.land/p/nt/grc721/v0`][c-p-grc721] | Non-fungible token |

GRC-1155, the multi-token standard, sits in quarantine at
[`gno.land/p/nt/grc1155/v0`][c-p-grc1155].

### Notable realms

Every realm below is on mainnet, `gnoland-1`, and on the Onyx testnet,
`onyx-1`, measured on 2026-11-01 with `vm/qpaths`:

```bash
gnokey query vm/qpaths -data "gno.land/r/" -remote "https://rpc.gno.land:443"
```

| Realm | Path | What it is |
| --- | --- | --- |
| **[wugnot](https://gno.land/r/gnoland/wugnot)** | [`gno.land/r/gnoland/wugnot`][c-r-wugnot] | Wrapped GNOT as a GRC-20 |
| **[boards2](https://gno.land/r/gnoland/boards2/v0)** | [`gno.land/r/gnoland/boards2/v0`][c-r-boards2] | Discussion boards with moderation |
| **[grc20reg](https://gno.land/r/nt/grc20reg/v0)** | [`gno.land/r/nt/grc20reg/v0`][c-r-grc20reg] | Registry of GRC-20 tokens |
| **[gov/dao](https://gno.land/r/gov/dao)** | [`gno.land/r/gov/dao`][gov-dao] | [The governance proxy](#governance-govdao) |
| **[sys/validators](https://gno.land/r/sys/validators/v0)** | [`gno.land/r/sys/validators/v0`][c-validators-v0] | Validator set changes through GovDAO proposals, [published through `r/sys/params` for the node to read][c-validators-v0-doc] |
| **[sys/params](https://gno.land/r/sys/params)** | [`gno.land/r/sys/params`][sys-params] | Chain parameters, set by proposal |
| **[sys/users](https://gno.land/r/sys/users)**, **[sys/namereg](https://gno.land/r/sys/namereg/v0)** | [`gno.land/r/sys/users`][c-r-users], [`gno.land/r/sys/namereg/v0`][namereg] | The name to address registry, and the realm anyone calls to register a name, [free unless GovDAO sets a price][c-namereg-free] |
| **[sys/names](https://gno.land/r/sys/names)** | [`gno.land/r/sys/names`][c-r-names] | The namespace verifier [the keeper calls on every upload][c-names-check] |
| **[sys/txfees](https://gno.land/r/sys/txfees)**, **sys/rewards** | [`gno.land/r/sys/txfees`][c-r-txfees], [`gno.land/r/sys/rewards`][c-r-rewards] | Fee distribution and contribution rewards. [Both][txfees-todo] [are stubs][c-rewards-todo] at this commit |
| **[sys/cla](https://gno.land/r/sys/cla)** | [`gno.land/r/sys/cla`][c-cla] | The contributor license agreement. Once GovDAO sets one, [the keeper refuses an upload from an address that has not signed it][c-cla-check] |
| **[valopers](https://gno.land/r/gnops/valopers)** | [`gno.land/r/gnops/valopers`][c-valopers] | Validator operator profiles. [A validator proposal names an operator registered here][c-valopers-proposal] |

## Interrealm programming

Gno is a multi-user language: importing another user's realm uses the same
syntax as importing a library. Two questions are kept apart inside the VM,
[per the interrealm specification][interrealm-v2]: who is acting, the
identity, and whose saved state may be written, the storage realm. The
mechanics are in [the GnoVM
doc](gnovm-architecture.md#interrealm-identity-and-authority); this is the
builder's view.

### Crossing functions and `cur`

A crossing function has a first parameter of type `realm`, [named `cur` or
left blank][c-cur-name]. It is the only kind of function a `MsgCall` can name.
Call it as `f(cross(cur), args...)` to switch identity to the realm that
declared `f`, or as `f(cur, args...)` inside that realm to keep the current
one. A bare `cross` as the first argument, found in older text, fails the type
check: `cross` is a builtin function, and the parameter takes a `realm`.

Each crossing call mints a fresh `cur` whose `Previous()` is the caller's
realm. At the root, the keeper [substitutes the transaction
signer][call-origin]. So `cur` heads a linked list back to the signer, and no
user code can change what `cur.Previous()` returns. A realm value [cannot be
saved][c-refuse-persist]: to remember a caller, store
`cur.Previous().Address()`. A realm can also [mint an identity with
`cur.Sub(subpath)`][c-sub] for something it holds inside, such as one account
of a ledger. The result has an address of its own and [rules of its
own](gnovm-architecture.md#sub-realm-identities).

![cur is a linked list of identities built by crossing calls, back to the
transaction signer.](figures/cur-chain.svg)

```go
// gno.land/r/alice/bank
package bank

type Token struct{ Balance int }

func NewToken(n int) *Token { return &Token{Balance: n} }

// Crossing: called as bank.Debit(cross(cur), t, 10) from another realm.
func Debit(cur realm, t *Token, amount int) {
    caller := cur.Previous() // the realm that crossed into bank
    _ = caller
    t.Balance -= amount      // t is a bank.Token, so bank may write it
}

// Plain: the caller still acts, and the write lands in bank's storage.
func DebitLocal(t *Token, amount int) {
    t.Balance -= amount
}
```

```go
// gno.land/r/bob/app
package app

import "gno.land/r/alice/bank"

var myToken = bank.NewToken(100) // bank owns the Token, app holds it

func Spend(cur realm) {
    bank.Debit(cross(cur), myToken, 10) // bank acts, app is its caller
    bank.DebitLocal(myToken, 10)        // app acts, the write lands in bank
    myToken.Balance = 0                 // panics: the object is bank's
}
```

`myToken` comes from a constructor because a realm [may only build another
realm's types inside that realm's own functions][checkconstruction]: a
`bank.Token{}` literal in `app` panics with `cannot allocate`. The object
belongs to bank because bank declared its type, [even though app's variable
holds it][c-stamp].

### What the VM decides for each call

![Six calls a realm can make against another realm's code and storage, and
what the VM decides for each.](figures/interrealm-cases.svg)

The storage realm of a call that does not cross follows [three borrow
rules][borrow-rules]:

1. A function, method or closure declared in a realm runs with that realm's
   storage. This is how `bank.DebitLocal` writes.
2. A method declared in a pure package runs with the storage of the realm that
   owns its receiver. This is how `avl.Tree.Set` writes the tree of whichever
   realm holds it.
3. Any other closure runs with the realm that was active when it was created.

Each object carries a stamp, the realm it belongs to: [the realm that declared
its type][c-stamp], or else the storage realm at the moment it was built. Four
gates then [refuse what the borrow rules
allow](gnovm-architecture.md#the-write-gates):

- **Ownership.** A write to an object [whose stamp names another
  realm][isreadonlyby] panics. The message reads [`cannot directly modify
  readonly tainted object`][readonlypanic], though the test reads the stamp
  alone.
- **Construction.** Building a value of a type another realm declared [panics
  with `cannot allocate`][checkconstruction].
- **Conversion.** Converting another realm's object to another type
  [panics][c-conv], so it cannot be retyped into a pure package type whose
  methods write.
- **Pure package immutability.** A pure package's saved objects [cannot change
  once its initialization is over][c-immutable].

### Why explicit crossing

In Solidity every external call silently shifts `msg.sender`, and the callee
may call back into the caller before it finishes: [reentrancy][c-reentrancy],
the bug behind [the 2016 DAO drain][c-the-dao]. In Gno the identity change is
written in the source, and a callee writes the caller's objects only through
what the caller hands it: a closure the caller declared, or an object of a
pure package type, whose methods [run with its owner's
storage](gnovm-architecture.md#storage-borrow). If the callee calls back, it
needs a crossing function the caller exported, and that function sees the
callee, not the user, as its caller. A panic that crosses a realm boundary
[aborts the whole transaction][panic-boundary], so a callee cannot leave a
caller half-written either.

### Rules to keep

- A function meant for users **must** be crossing. `gnokey maketx call`
  [refuses a non-crossing function][keeper-crossing-check].
- A `p/` package [cannot declare a crossing function][c-crossing-allowed] and
  [cannot import a realm][p-imports-r].
- Another realm's objects cannot be written from outside. To change one, call
  a function that realm exports.
- Handing out a pointer to an object of a pure package type, an `avl.Tree` for
  one, hands out write access to it.
- A realm value received as an ordinary argument may be left over from an
  earlier call: [check `rlm.IsCurrent()`][c-iscurrent] before trusting it.
- `unsafe.PreviousRealm` and `unsafe.CurrentRealm` [can be fooled from a
  helper][c-unsafe-walk], and `unsafe.OriginCaller` [by any realm the signer
  calls][c-unsafe-origin]. Use `cur`.

## Governance (GovDAO)

GovDAO is the on-chain organization that selects validators, spends the
treasury and sets chain parameters. The chain runs as proof of authority:
validators are [voted in and out by GovDAO members][c-validators-v0-doc], not
chosen by stake. The [Constitution][constitution] sets [who qualifies for each
tier][c-tier-criteria]; on chain, membership is what the tiers below encode.
The genesis of [mainnet][c-mainnet-govdao] and of [Onyx][c-onyx-govdao] each
seeded a single T1 member and [locked which implementations may
act][c-loader-doc]. Read a network's members today with:

```bash
gnokey query vm/qrender -data "gno.land/r/gov/dao/memberstore/v0:members" \
  -remote "https://rpc.gno.land:443"
```

The code is [`r/gov/dao`][gov-dao], a proxy whose implementation [an upgrade
proposal replaces][dao-proxy]. At this commit the implementation is
[`r/gov/dao/impl/v0`][dao-impl], with members in
[`r/gov/dao/memberstore/v0`][memberstore].

### Tiers

[Read from the member store's `init`][tiers]:

| Tier | Base power | Invitation points | Target size | Power cap |
| --- | --- | --- | --- | --- |
| **T1** | 3 | 3 | 70 members | none |
| **T2** | 2 | 2 | between a quarter and twice the T1 count | the tier's total power is capped at two thirds of T1's; each member's power shrinks to fit |
| **T3** | 1 | 1 | none | capped at one third of T1's total |

The member store [declares the target sizes][c-tier-size] and no code checks
them; 70 T1 members is [the Constitution's target][c-t1-target]. T1 and T2
members are added by proposal, [and only those two tiers][add-member-tiers];
a T1 or T2 member [adds a T3 member directly by spending one invitation
point][add-member-direct]. A proposal adding or promoting a member to T1 [is
voted by T1 alone, and one to T2 by T1 and T2][filter-by-tier].

### Proposals

Every proposal carries an executor, a callback that runs when the proposal
passes. The lifecycle, [shown end to end by an integration test][c-prop-flow]:

1. A member builds a request with one of the `New...Request` functions and
   submits it with `gnokey maketx run`, because a request [carries an executor
   that a `MsgCall` cannot build from string arguments][c-run-needed]. The
   genesis of [mainnet][c-mainnet-run] and of [Onyx][c-onyx-run] fills
   `vm:p:run_submitters` with the T1 members, and [only a listed address may
   send a run][c-run-gate].
2. Members vote YES, NO or ABSTAIN with `MustVoteOnProposalSimple`.
3. [Any account][c-any-exec] then calls `ExecuteProposal`. At [66.66
   percent][supermajority] YES the proposal is [accepted and its executor
   runs][accept-deny]; at 66.66 percent NO it is denied. Short of both, the
   call fails and voting goes on.

Percentages are taken over the power of the tiers allowed to vote, [counted
from the members at the moment of the call][c-total-power], so an ABSTAIN or
a missing vote works against both thresholds. The threshold is [the `Law`
value][supermajority], itself changeable by proposal.

The request types in [`impl/v0`][prop-requests] change the law, upgrade the
DAO implementation, add, withdraw or promote a member, pay from the treasury,
and update the treasury's GRC-20 token list. [`memberstore/v0` adds
one][c-change-tiers] that replaces the tiers, and each realm GovDAO controls
exports its own. [`misc/govdao-scripts`][c-govdao-scripts] wraps the common
ones as shell scripts.

![The GovDAO tiers, their power caps, and a proposal's path from creation to
execution.](figures/govdao.svg)

### What GovDAO controls

| Area | Realm |
| --- | --- |
| Validators | [`r/sys/validators/v0`][c-validators-v0] |
| Every chain parameter, among them [a height at which every node halts][c-halt], [the fee collector][c-fee-collector], [transfer locks][c-unlock], [who may deploy code](gnovm-architecture.md#who-may-send-code) and [who may send `MsgRun`][c-run-submitters] | [`r/sys/params`][sys-params] |
| Treasury, ugnot and GRC-20 payments | [`r/gov/dao/treasury/v0`][treasury] |
| Names: [controllers and registrations][sys-users], [names and the registration price][c-namereg-admin], [the namespace verifier's pause, voted by T1 alone][c-names-pause] | [`r/sys/users`][c-r-users], [`r/sys/namereg/v0`][namereg], [`r/sys/names`][c-r-names] |
| The contributor license agreement uploaders must sign | [`r/sys/cla`][c-cla] |

No upload is outside governance, even under your own address: the verifier's
pause, the license agreement and the code submission policy are each set by
proposal, and each applies to every deployer.

[Memba](https://memba.club) is a web app for gno.land by
[Samourai](https://github.com/samouraiworld/memba). It offers a multisig
wallet, one that several keys control together, and DAO governance, GovDAO
proposals and votes among them.

## Architecture

### Cosmos, Tendermint, CometBFT and Tendermint2

**BFT**, Byzantine fault tolerance, is the property of a distributed system
that reaches agreement while fewer than one third of its participants are
faulty or malicious. The name comes from Lamport's [Byzantine Generals
Problem](https://en.wikipedia.org/wiki/Byzantine_fault) of 1982. In a BFT
chain, validators vote on blocks in rounds of propose, prevote and precommit,
and a block is final once more than two thirds agree. There are no forks and
no reorganizations: **instant finality**.

| Term | Meaning |
| --- | --- |
| **Validator** | A node that proposes and votes on blocks. It must stay online. |
| **Proposer** | The validator chosen, deterministically, to propose the block of a given round. |
| **Block** | A batch of transactions, referencing the previous block by hash. |
| **Height** | The block number, from 1. |
| **Round** | One attempt at consensus within a height. Round 0 failing, round 1 tries with another proposer. |
| **Finality** | A committed block is never reverted. Proof-of-work chains only get there probabilistically. |
| **Halt** | The chain stops producing blocks: validators holding a third or more of the voting power are offline, or a determinism bug makes validators disagree. Nothing committed is lost. |
| **Supermajority** | More than two thirds of the voting power, the share a round needs in prevotes and then in precommits. |
| **Voting power** | A validator's weight. Proof of stake ties it to stake; gno.land assigns it through [GovDAO](#governance-govdao). |
| **Liveness** | Blocks keep coming as long as enough honest validators are online. |
| **Safety** | Validators never commit conflicting blocks at the same height, as long as fewer than a third are Byzantine. |
| **Slashing** | A penalty on a validator's stake for misbehaviour such as signing two blocks at one height. |

**Tendermint**, by Jae Kwon in 2014, is a BFT consensus engine for
blockchains. It separates consensus from the application through ABCI, the
Application Blockchain Interface: consensus orders transactions, the
application says what they mean.

**Cosmos SDK** is [the framework][d-cosmos-sdk] for ABCI applications.
**CometBFT** is [the fork and successor of Tendermint Core][d-cometbft] that
the Cosmos SDK recommends. It adds ABCI++, hooks into every step of
consensus, and encodes with Protobuf.

**Tendermint2**, TM2, is Jae Kwon's minimal fork of the original Tendermint,
kept in the monorepo at [`tm2/`][tm2]. Its code imports no gRPC, viper, cobra
or Prometheus package, and it [decodes transactions with Amino][d-amino-tx], a
codec of its own that [writes Protobuf-compatible bytes from plain Go
structures][d-amino]. [Its README sets the principles][d-tm2-principles]: the
code is the specification, and every dependency is audited and becomes part of
the repository. The same README says TM2 is to [operate independently, at
`tendermint/tendermint2`, once gno.land is on mainnet][tm2-independent]. At
5a51b55f4, after [the mainnet launch][d-mainnet-launch], TM2 is still
developed in the monorepo.

The README [still proposes][tm2-proposed] carrying evidence of misbehaviour as
a mempool lane of its own. The consensus state [reports two conflicting votes
from one validator][evpool-add] to an evidence pool, and the node [wires in a
pool][d-evpool-wired] that [does nothing][d-evpool-noop], marked to be replaced
by one that enables slashing. There is no slashing.

In short: the Cosmos SDK pairs with CometBFT and gno.land runs Tendermint2.
Both implement the Tendermint algorithm with instant finality, and TM2 trades
features, slashing included, for a smaller code base.

### How the layers fit together

![From a signed transaction to the database, one layer at a time; a query
reads state without a transaction.](figures/vm-stack.svg)

Tendermint2 orders transactions and calls the application over ABCI, one call
per event. `CheckTx` admits a transaction to the mempool, the queue of
transactions waiting for a block. `DeliverTx` runs it inside a block, and
`Commit` [writes the block's changes and returns the application
hash][d-commit-hash], a fingerprint of the resulting state. The application is
tm2's `BaseApp`, which checks each transaction, takes its fee and routes its
messages. Its `vm` route reaches the keeper, and the keeper builds a fresh
interpreter, a Machine, for each run of Gno code.

The VM writes a realm's objects to [a plain key-value store][d-mount-base]
when a call into the realm returns. It writes every object the call created or
changed that the realm still reaches. [The B+32 tree][d-mount-bptree] holds
package sources and [the hash of each escaped object][d-setobject-hash], one
that more than one object refers to. It is a Merkle tree: every node stores
the hash of its children, so the root changes whenever anything under it does.
[Its nodes hold up to 32 children][d-bptree-b]. The plain store
[commits with no hash][d-dbadapter-commit], so only the tree feeds the
application hash. [The next block's header carries that
hash][d-header-apphash], so a validator that computed a different state
disagrees at once. The steps are in [the GnoVM
doc](gnovm-architecture.md#how-gnoland-drives-the-vm), the stores in [The
store](gnovm-architecture.md#the-store).

[Reaching Consensus](https://gno.land/r/gnoland/blog:p/reaching-consensus) on
the gno.land blog is a practical walk through TM2 consensus; [the Tendermint
paper](https://arxiv.org/pdf/1807.04938.pdf) is the algorithm.

## Tokenomics

Three tokens in the wider ecosystem, [per the Constitution][constitution]:

| Token | Role |
| --- | --- |
| **ATONE** | [Staking and governance on Atom.One][d-const-atone], the chain that is to secure gno.land |
| **PHOTON** | [Gas on Atom.One, and the token gno.land is to pay Atom.One with][d-const-photon] |
| **GNOT** | [The storage deposit token on gno.land, and its fee token until the migration][d-const-gnot] |

### GNOT

[Genesis allocation][genesis-allocation], 1.333 billion GNOT in total:

| Allocation | GNOT | Share |
| --- | --- | --- |
| Airdrop 1, from a partial Cosmos governance snapshot | 350 M | 26.26% |
| Airdrop 2, from an AtomOne snapshot before launch | 231 M | 17.33% |
| Core treasury | 40 M | 3.00% |
| Ecosystem treasury | 60 M | 4.50% |
| Validator treasury | 20 M | 1.50% |
| Investors | 300 M | 22.51% |
| NT, LLC | 332 M | 24.91% |

- On chain the unit is `ugnot`: one GNOT is one million ugnot.
- [GNOT does not inflate][no-inflation]: the supply never exceeds 1.333
  billion, and [no amendment may raise it][no-amendment].
- [1.333 billion GNOT corresponds to 13.33 terabytes of state][d-state-size],
  which is [the code default of 100 ugnot per byte][storage-price].
- The storage price [never increases and may fall by at most ten percent a
  year][storage-price-rule], and not while the stored state exceeds a typical
  consumer drive.
- Genesis allocations [vest over 24 months][vesting]: four percent when GNOT
  becomes transferable, then four percent a month. [150 million of the
  Investors' GNOT unlock at once][d-investor-unlock].
- [The mainnet genesis][d-genesis-readme] takes its balances from an audited
  allocation sheet. Its build [stops unless the sheet holds][d-genesis-check]
  exactly [1,332,999,998.328067 GNOT across 3,262,481
  accounts][d-genesis-total]. It [encodes the vesting][d-genesis-vesting] as
  four percent unlocked at launch and the other 96 percent unlocking
  continuously over the 24 months.
- At genesis [ugnot transfers were restricted][d-genesis-restricted] to
  whitelisted addresses, [as the Constitution requires][d-const-whitelist].
  Whether the restriction still holds is the live parameter
  `bank:p:restricted_denoms`, read with `gnokey query
  params/bank:p:restricted_denoms -remote https://rpc.gno.land:443`.

### How the tokens relate

Interchain security, ICS, is [validating a chain with the validator set of
another chain][d-ics-def]. Gno.land runs on its own validators until GovDAO
decides that Atom.One's ICS is ready, then [migrates to it][ics-migration].
After that ATONE secures, PHOTON pays Atom.One for compute, GNOT pays for
storage. Until then, and on every testnet, `ugnot`
[pays for both gas and storage][d-const-launch-fees].

![Where a transaction's ugnot goes: the fee to the collector, the deposit
into the realm's deposit address and back.](figures/gnot-flows.svg)

## Go versus Gno

Gno [targets the Go 1.17 language specification][compat-go117]; the VM is
built with [Go 1.25][go-mod]. Go 1.18 is [the cut-off][d-compat-cutoff]:
generics and the later builtins `min`, `max` and `clear` are refused by
design, not missing by accident. The [compatibility table][compat]
is the reference. What is missing:

| Feature | Status |
| --- | --- |
| Goroutines, `go` | [Missing][d-compat-go] |
| Channels, `select` | [Missing][d-compat-chan] |
| Generics | [Refused by design][d-compat-cutoff] |
| `unsafe`, `net`, `os`, `syscall` | Absent: [each would need non-deterministic behaviour][d-compat-nondet] |
| `cgo` | Absent: [Go-specific][d-compat-cgo] |
| `complex64`, `complex128` | [Missing][d-compat-complex] |
| `reflect` | [Listed as todo][compat-reflect] |
| Dot imports, `import . "pkg"` | [Refused][d-compat-dot]: a dependency that gains an export could collide with a name in an importer already deployed |

Behavioural differences that matter:

- `time.Now` [returns the block time][d-time-now]. `Sleep`, timers and
  tickers [are not implemented][d-compat-time].
- `fmt` [exists only in tests][fmt-tests]; use `ufmt` on chain.
- `sort.Slice` [is missing][compat-sort]: implement `sort.Interface` and call
  `sort.Sort`.
- `math/rand` [follows Go's `math/rand/v2`][d-compat-rand]: `IntN`, not
  `Intn`.
- `crypto/sha256` [has `Sum256` only][d-compat-sha256]; `crypto/ed25519` [has
  `Verify` only][d-compat-ed25519].
- `init` runs once, [when the package is deployed][d-runinit]. Written
  `init(cur realm)`, [it sees the deployer as `cur.Previous()`][d-init-cur].
  It is a constructor, not a program start.
- `panic` is control flow: it [aborts and rolls back the
  transaction](gnovm-architecture.md#when-a-message-panics).
- Global variables are the storage. [The VM persists
  them](gnovm-architecture.md#realms-and-persistence).
- Maps [iterate in insertion order][d-maplist], deterministically, but a map
  is [one stored object, loaded
  whole](gnovm-architecture.md#values-and-the-memory-model); prefer `avl.Tree`
  for state that grows.
- `[]byte(s)` [has capacity equal to its length][d-compat-cap].
- Builtin type names and `iota` [cannot be redeclared][d-compat-shadow].
- Import paths are `gno.land/p/...` and `gno.land/r/...`, never
  `github.com/...`. The manifest is `gnomod.toml`.

Gno-only additions: [the `address` and `realm` types][d-compat-gnotypes], the
[`cross`][d-uv-cross] and [`istypednil`][d-uv-istypednil] builtins, and
[`revive`][d-uv-revive], which [runs only where the machine enables it, in
tests][d-revive-enabled].

## Known limitations

The gaps below are read at 5a51b55f4, each anchored on the code that says so.
Mainnet runs a tagged release, named in [its upgrade log][d-mainnet-upgrades],
so a gap closed after that release stays open there. The interpreter's own
gaps are in [the GnoVM doc](gnovm-architecture.md#what-is-not-built-yet).

**Consensus.** No slashing, no misbehaviour penalty. Two conflicting votes from
one validator [are reported][evpool-add] to [an evidence pool that does
nothing][d-evpool-noop], and evidence as [a mempool lane of its
own][tm2-proposed] is proposed, not built.

**Fees.** Users [pay the whole `-gas-fee` they offered][gas-charged], whatever
the transaction used. In progress: [#3805][issue-3805]. Fees [go to the fee
collector][deduct-fees] and stay there: the distribution realm [is a
placeholder][txfees-todo]. Gas itself is metered throughout: each opcode
carries [a constant calibrated on reference hardware][opcpu], and
type-checking and preprocessing [pay per source byte][d-preprocess-gas] when a
package is uploaded or a script is run.

**Persistence.** Object graphs with cycles [are not
supported][ownership-doc]. `attach` [panics][uv-attach].

**Language.** No goroutines, channels, generics or `reflect`, and `revive`
[runs only in tests][d-revive-enabled].

**Tooling.** `gno test` runs `Test` and `Example` functions. Benchmarks, fuzz
tests and external `_test` packages [are not supported][d-test-nobench].

**IBC.** The node carries no IBC module. IBC, Inter-Blockchain Communication,
lets one chain act on another's state, and for gno.land it is written as Gno
realms kept outside the monorepo, in [allinbits/gno-realms][d-gno-realms]. The
standard library carries [the curve arithmetic, hashes and proof
checks](gnovm-architecture.md#the-packages) of an IBC light client, the code
that follows another chain by checking its block headers. The changes to
the chain that those realms needed are tracked in [a meta issue][issue-4907],
closed as completed.

## Additional tools

Everything under [`contribs/`][contribs] at 5a51b55f4, plus one external
repository:

| Tool | What it does |
| --- | --- |
| [gnodev][d-gnodev] | The local development server |
| [gnofaucet][d-gnofaucet] | A faucet server for test tokens |
| [gnokms][d-gnokms] | Signs for a validator from a separate process or machine, so the key is not kept in plain text on the node's disk; [its one backend is a local `gnokey` key][d-gnokms-gnokey] |
| [gnokeykc][d-gnokeykc] | OS keychain integration for `gnokey` |
| [gnogenesis][d-gnogenesis] | Builds and edits `genesis.json`: validators, balances, transactions; [the mainnet genesis is built with it][d-genesis-gnogenesis] |
| [gpao][d-gpao] | The off-chain approver for chains where an uploaded package stays unusable until approved: it checks each upload and enables it, as [Deploying through approval](gnovm-architecture.md#deploying-through-approval) describes |
| [gnobro][d-gnobro] | A terminal browser for realms |
| [gnobr][gnobr] | Rolls a node back to a height and replays blocks locally |
| [gnohealth][d-gnohealth] | Health checks for a chain, [such as block timestamp drift][d-gnohealth-ts] |
| [gnomd][d-gnomd] | Renders a markdown file in the terminal |
| [gnomigrate][d-gnomigrate] | [Converts legacy transactions][d-gnomigrate-txs] to the format a genesis file's transactions take |
| [tx-archive][d-tx-archive] | Backs up and restores transactions; [what keeps staging's history][d-staging-archive] |
| [github-bot][d-github-bot] | Pull request automation for the monorepo |
| [tx-indexer](https://github.com/gnolang/tx-indexer) | Indexes blocks, transactions and events for querying |

## Development environment

### Editors

| Editor | Plugin |
| --- | --- |
| VS Code | [Gnolang](https://marketplace.visualstudio.com/items?itemName=Gnoverse.gnolang), by Gnoverse |
| Any editor that speaks LSP, the Language Server Protocol | [gnopls](https://github.com/gnoverse/gnopls), the Gno language server; [its README sets up Vim, Neovim, Emacs and Zed](https://github.com/gnoverse/gnopls#editor-setup) |
| Any editor | `gno fmt`, or `gofmt` directly on `.gno` files |

### Clients

| Library | Language | What it is |
| --- | --- | --- |
| [gnoclient][gnoclient] | Go | [The Go client for the gno.land RPC API][d-gnoclient-readme] |
| [gno-js-client](https://github.com/gnolang/gno-js-client) | JavaScript, TypeScript | gno.land client |
| [tm2-js-client](https://github.com/gnolang/tm2-js-client) | JavaScript, TypeScript | Lower-level Tendermint2 client |
| [Gno Native Kit](https://github.com/gnolang/gnonative) | Mobile, desktop | Native app framework |
| [Adena](https://adena.app) | Browser | Wallet extension, by Onbloc |

## Ecosystem

| Tool | Where | What it is |
| --- | --- | --- |
| gnoweb | [gno.land](https://gno.land) | Browse realms, read source, render pages |
| Gnoscan | [gnoscan.io](https://gnoscan.io) | Block explorer, by Onbloc |
| Playground | [play.gno.land](https://play.gno.land) | Write, test, deploy and share Gno in the browser |
| Gno Studio Connect | [gno.studio/connect](https://gno.studio/connect) | Explore and call realm functions |
| Memba | [memba.club](https://memba.club) | Multisig coordination, DAO governance and community tools, by [Samourai](https://github.com/samouraiworld/memba) |

Community realms and projects: [boards2](https://gno.land/r/gnoland/boards2/v0),
discussion boards on mainnet, and [Gnoswap](https://gnoswap.io), an exchange
that prices each trade from pooled reserves, with [its realms on
GitHub](https://github.com/gnoswap-labs/gnoswap).

## Further reading

- [docs.gno.land](https://docs.gno.land): the official documentation.
- [Effective Gno](https://docs.gno.land/resources/effective-gno): patterns and
  design guidance.
- [Interrealm specification v2][interrealm-v2] and the [security
  guide][security-guide]: the crossing rules and the threat classes they close.
- [Interact with gnokey](https://docs.gno.land/users/interact-with-gnokey),
  [Gas fees](https://docs.gno.land/resources/gas-fees), [Storage
  deposit](https://docs.gno.land/resources/storage-deposit), [Data
  structures](https://docs.gno.land/resources/gno-data-structures),
  [Gno packages](https://docs.gno.land/resources/gno-packages), [Gno
  networks](https://docs.gno.land/resources/gnoland-networks).
- [How the GnoVM works](gnovm-architecture.md): the interpreter, with figures.
- [Constitution][constitution], [Laws][laws], [Manifesto][manifesto], and the
  [whitepaper][whitepaper].
- [Tendermint consensus paper](https://arxiv.org/pdf/1807.04938.pdf).
- [getting-started](https://github.com/gnolang/getting-started), a starter
  template, and [awesome-gno](https://github.com/gnoverse/awesome-gno).

**Community:** [Discord](https://discord.gg/S8nKUqwkPn),
[GitHub](https://github.com/gnolang), [X](https://x.com/_gnoland).

*Kept in
[gno-agent-workspace](https://github.com/samouraiworld/gno-agent-workspace).*

[a-auth-queries]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/handler.go#L159
[a-avl]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/avl/v0
[a-bank-supply]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/bank/handler.go#L86
[a-bech32]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/crypto/globals.go#L26
[a-call-convert]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/convert.go#L225-L226
[a-call-crossing]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L1255-L1257
[a-cla-check]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L550-L552
[a-cla-page]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/cla/render.gno#L20-L30
[a-code-policy]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L24-L34
[a-constitution-tb]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L181
[a-counter]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/demo/counter
[a-crossing-allowed]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/preprocess.go#L4750-L4764
[a-deposit-addr]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/misc.go#L218
[a-deposit-lock]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L2398-L2400
[a-docs-home]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/quarantined/gno.land/r/docs/home
[a-gno-deplist]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/cmd/gno/tool.go#L32
[a-gno-doc-remote]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/doc/doc.go#L197-L198
[a-gno-test-example]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/test/test.go#L753-L763
[a-gnodev-deploykey]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/command_local.go#L37
[a-gnodev-examples]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/README.md?plain=1#L14-L15
[a-gnodev-export]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/app.go#L632-L655
[a-gnodev-interactive]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/command_local.go#L40
[a-gnodev-local-first]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/pkg/packages/loader.go#L95-L107
[a-gnodev-mnemonic]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/setup_address_book.go#L54-L62
[a-gnodev-noexamples]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/README.md?plain=1#L37-L38
[a-gnodev-remote]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/README.md?plain=1#L103
[a-gnodev-replay]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/README.md?plain=1#L20
[a-gnodev-watch]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/README.md?plain=1#L18-L19
[a-gnodev-web]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/setup_web.go#L20
[a-gnodev-workspace]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/packages/load.go#L174-L197
[a-gnoland-start]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/cmd/gnoland/root.go#L38
[a-gnoweb]: https://github.com/gnolang/gno/tree/5a51b55f4/gno.land/pkg/gnoweb
[a-govdao-proxy]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/proxy.gno#L12-L20
[a-mainnet-names]: https://github.com/gnolang/gno/blob/chain/mainnet/misc/deployments/mainnet.gno.land/README.md?plain=1#L15
[a-make-install]: https://github.com/gnolang/gno/blob/5a51b55f4/Makefile#L28
[a-make-node]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/Makefile#L47
[a-maketx-flags]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/crypto/keys/client/maketx.go#L75-L127
[a-map-crossover]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/effective-gno.md?plain=1#L650-L651
[a-map-object]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/effective-gno.md?plain=1#L629-L638
[a-master]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/crypto/keys/client/maketx.go#L128-L133
[a-msg-addpkg]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/msgs.go#L19-L24
[a-msg-call]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/msgs.go#L102-L109
[a-msg-enable]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/msgs.go#L284-L317
[a-msg-reject]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/msgs.go#L360-L363
[a-msg-run]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/msgs.go#L192-L197
[a-msg-send]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/bank/msgs.go#L14
[a-msg-session]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/msgs.go#L16
[a-msg-session-revoke]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/msgs.go#L61
[a-msg-session-revokeall]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/msgs.go#L90
[a-namereg-page]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/namereg/v0/render.gno#L64-L74
[a-names-bypass]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/names/verifier.gno#L226-L229
[a-names-call]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L509-L511
[a-names-enable]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/names/verifier.gno#L113-L121
[a-names-oneway]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/names/verifier.gno#L38-L43
[a-nym-format]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/namereg/v0/users.gno#L24-L27
[a-onyx-names]: https://github.com/gnolang/gno/blob/chain/mainnet/misc/deployments/onyx.gno.land/README.md?plain=1#L16
[a-ownable]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/ownable/v0
[a-p-immutable]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/realm.go#L283-L299
[a-paths-limit]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/handler.go#L200-L201
[a-pr-6162]: https://github.com/gnolang/gno/pull/6162
[a-private-objects]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/realm.go#L1182-L1184
[a-private-types]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/realm.go#L1318-L1320
[a-propose-name]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/namereg/v0/admin.gno#L62
[a-qpaths-user]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/handler.go#L220-L221
[a-qrender]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/handler.go#L170-L172
[a-quarantined]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/README.md?plain=1#L17-L20
[a-query-limit]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L52
[a-realm-addr]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/misc.go#L197-L198
[a-refund-ratio]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L2453-L2455
[a-register-price-prop]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/namereg/v0/admin.gno#L108
[a-run-nosave]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L1623
[a-run-submitters]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L123-L132
[a-simulate-only]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/crypto/keys/client/broadcast.go#L244-L254
[a-stay-private]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L662-L664
[a-store-tests]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/store.go#L1106-L1110
[a-sys-users]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/users
[a-ugnot]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/glossary.md?plain=1#L304-L305
[a-uv-address]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1542
[a-uv-cross]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1831
[a-uv-realm]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1571
[addpkg-flags]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/keyscli/addpkg.go#L59-L71
[compat-go117]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L3-L16
[default-account]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/integration/node_testing.go#L26
[epath]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L1530
[gas-charged]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gas-fees.md?plain=1#L45-L46
[gno-help]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/cmd/gno/main.go#L97-L119
[gno-tree]: https://github.com/gnolang/gno/tree/5a51b55f4
[gnodev-keys]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/README.md?plain=1#L24-L34
[gnodev-premine]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/accounts.go#L55-L60
[gnokey-session]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/crypto/keys/client/session_create.go#L55-L89
[gnomod-example]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gnoland/wugnot/gnomod.toml#L1-L2
[gnover]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/gnomod.go#L43
[issue-3805]: https://github.com/gnolang/gno/issues/3805
[keeper-exists]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L801-L805
[keeper-private]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L723-L728
[keeper-private-realm]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L665-L667
[namereg]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/namereg/v0
[namespace-rule]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/names/verifier.gno#L1-L17
[p-imports-r]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/preprocess.go#L5692-L5696
[pager]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/avl/pager/v0
[query-kinds]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/handler.go#L123-L151
[r-docs]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/quarantined/gno.land/r/docs
[register-price]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/namereg/v0/users.gno#L15-L18
[storage-price]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L41
[unsafe-doc]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L1-L11
[b-call-expr]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L1271-L1276
[b-chain-domain]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L39
[b-chain-domain-check]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L797-L798
[b-chainid-sign]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/std/tx.go#L109-L121
[b-constitution-lock]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L126
[b-counter-test]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/demo/counter/counter_test.gno#L5-L7
[b-deposit-cap]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L2357-L2360
[b-enable-deposit]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper_inert.go#L427-L429
[b-fee-check]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/ante.go#L529-L543
[b-gasprice-floor]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/keeper.go#L488-L498
[b-gasprice-query]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/handler.go#L263-L267
[b-gasprice-rise]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/keeper.go#L451-L463
[b-gnodev-defaults]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnodev/command_local.go#L29-L36
[b-inert-charge]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L219
[b-inert-charge-field]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L150-L151
[b-initial-gasprice]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/gnoland/genesis.go#L22
[b-mainnet-fresh]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L3
[b-mainnet-inert]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L17
[b-mainnet-launch]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L5
[b-mainnet-lock]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L18
[b-mainnet-nofaucet]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L11
[b-max-deposit]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L2400-L2405
[b-networks-doc]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gnoland-networks.md?plain=1#L5-L11
[b-onyx-inert]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/onyx.gno.land/README.md?plain=1#L17
[b-onyx-line]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/onyx.gno.land/README.md?plain=1#L3
[b-onyx-open]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/onyx.gno.land/README.md?plain=1#L11
[b-pkgmeta-status]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper_inert.go#L550-L552
[b-policy-default]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L43
[b-policy-values]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L24-L34
[b-refund-receiver]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L2464-L2470
[b-restricted-check]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/bank/keeper.go#L118-L132
[b-restricted-default]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/bank/params.go#L26-L29
[b-rpc-mainnet]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gnoland-networks.md?plain=1#L17-L18
[b-run-submitters]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L131-L132
[b-run-submitters-field]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L123
[b-simulate-estimate]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/crypto/keys/client/broadcast.go#L244-L254
[b-staging-examples]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gnoland-networks.md?plain=1#L143-L152
[b-staging-warning]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gnoland-networks.md?plain=1#L134-L139
[b-sys-params]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/params/params.gno#L66-L71
[b-testnets-retired]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gnoland-networks.md?plain=1#L212-L239
[call-origin]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L1326-L1334
[staging-cycle]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gnoland-networks.md?plain=1#L118-L125
[vm-params-defaults]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L40
[accept-deny]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/impl/v0/govdao.gno#L139-L149
[add-member-direct]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/impl/v0/impl.gno#L24-L55
[add-member-tiers]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/impl/v0/prop_requests.gno#L91-L93
[banker]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/chain/banker/banker.gno#L63-L78
[banker-pkg]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/stdlibs/chain/banker
[borrow-rules]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/machine.go#L2675-L2756
[c-address-valid]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1554-L1570
[c-any-exec]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/govdao-scripts/README.md?plain=1#L5
[c-arg-string]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/convert.go#L50-L52
[c-change-tiers]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/memberstore/v0/prop_requests.gno#L11-L35
[c-cla]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/cla
[c-cla-check]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L550-L561
[c-compat-stdlibs]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L118
[c-conv]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/op_expressions.go#L826-L842
[c-crossing-allowed]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/preprocess.go#L4750-L4764
[c-cur-name]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/preprocess.go#L2833-L2840
[c-fee-collector]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/params/fee_collector.gno#L7-L12
[c-govdao-scripts]: https://github.com/gnolang/gno/tree/5a51b55f4/misc/govdao-scripts
[c-halt]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/params/halt.gno#L18-L36
[c-immutable]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/realm.go#L344-L349
[c-initorder]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/generated.go#L2621-L2672
[c-iscurrent]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1692-L1699
[c-loader-doc]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/loader/v0/loader.gno#L8-L18
[c-mainnet-govdao]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L12
[c-mainnet-run]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L17
[c-markdown-doc]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/chain/markdown/markdown.gno#L1-L13
[c-markdown-pkg]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/stdlibs/chain/markdown
[c-namereg-admin]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/namereg/v0/admin.gno#L36-L126
[c-namereg-free]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/namereg/v0/users.gno#L11-L18
[c-names-check]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L509-L510
[c-names-pause]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/names/verifier.gno#L168-L187
[c-newbanker]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/chain/banker/banker.gno#L129-L131
[c-onyx-govdao]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/onyx.gno.land/README.md?plain=1#L13
[c-onyx-run]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/onyx.gno.land/README.md?plain=1#L17
[c-ownable-assert]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/p/nt/ownable/v0/ownable.gno#L45-L50
[c-p-addrset]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/moul/addrset/v0
[c-p-avl]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/avl/v0
[c-p-bptree]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/bptree/v0
[c-p-commondao]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/commondao/v0
[c-p-grc1155]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/quarantined/gno.land/p/nt/grc1155/v0
[c-p-grc20]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/grc20/v0
[c-p-grc721]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/grc721/v0
[c-p-mux]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/mux/v0
[c-p-ownable]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/ownable/v0
[c-p-poa]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/poa/v0
[c-p-sanitize]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/markdown/sanitize/v0
[c-p-seqid]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/seqid/v0
[c-p-treasury]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/treasury/v0
[c-p-uassert]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/uassert/v0
[c-p-ufmt]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/ufmt/v0
[c-p-urequire]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p/nt/urequire/v0
[c-params-key]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/chain/params/params.go#L98-L110
[c-prop-flow]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/integration/testdata/govdao_proposal_change_law.txtar#L15-L42
[c-quarantine]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/README.md?plain=1#L17-L20
[c-r-boards2]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/gnoland/boards2/v0
[c-r-grc20reg]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/nt/grc20reg/v0
[c-r-names]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/names
[c-r-rewards]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/rewards
[c-r-txfees]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/txfees
[c-r-users]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/users
[c-r-wugnot]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/gnoland/wugnot
[c-realm-type]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L110-L209
[c-reentrancy]: https://docs.soliditylang.org/en/latest/security-considerations.html#reentrancy
[c-refuse-persist]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/realm.go#L986-L992
[c-rewards-todo]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/rewards/rewards.gno#L1-L4
[c-run-gate]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/params.go#L123-L132
[c-run-needed]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/params/params.gno#L51-L53
[c-run-submitters]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/params/run_submitters.gno#L281-L312
[c-sanitize-doc]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1-L8
[c-stamp]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/alloc.go#L562-L566
[c-sub]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1719-L1775
[c-sysparams-gate]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/sys/params/params.go#L101-L113
[c-sysparams-pkg]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/stdlibs/sys/params
[c-t1-target]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L517
[c-the-dao]: https://en.wikipedia.org/wiki/The_DAO
[c-tier-criteria]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L615-L631
[c-tier-size]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/memberstore/v0/types.gno#L163-L167
[c-total-power]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/impl/v0/types.gno#L89-L119
[c-unlock]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/params/unlock.gno#L14-L20
[c-unsafe-origin]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L43-L50
[c-unsafe-walk]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L19-L25
[c-v0-banner]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/p/nt/mux/v0/README.md?plain=1#L1-L4
[c-validators-v0]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/validators/v0
[c-validators-v0-doc]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/validators/v0/validators.gno#L1-L6
[c-valopers]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/gnops/valopers
[c-valopers-proposal]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/validators/v0/proposal.gno#L37-L51
[chain-pkg]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/stdlibs/chain
[checkconstruction]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/alloc.go#L541-L555
[constitution]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md
[dao-impl]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/gov/dao/impl/v0
[dao-proxy]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/proxy.gno#L140-L146
[examples-p]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/p
[filter-by-tier]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/impl/v0/govdao.gno#L56-L71
[fmt-tests]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/tests/stdlibs/fmt
[gov-dao]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/gov/dao
[interrealm-v2]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gno-interrealm-v2.md
[isreadonlyby]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/ownership.go#L548-L553
[keeper-crossing-check]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L1255-L1257
[memberstore]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/gov/dao/memberstore/v0
[panic-boundary]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/op_call.go#L704-L709
[params-pkg]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/stdlibs/chain/params
[prop-requests]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/impl/v0/prop_requests.gno#L18-L311
[readonlypanic]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/machine.go#L3093-L3101
[runtime-pkg]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/stdlibs/chain/runtime
[stdlib-list]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/stdlibs
[supermajority]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/impl/v0/impl.gno#L12-L16
[sys-params]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/sys/params
[sys-users]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/users/admin.gno#L40-L185
[testing-overrides]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/tests/stdlibs/testing/context_testing.gno#L65-L166
[tiers]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/gov/dao/memberstore/v0/memberstore.gno#L29-L98
[treasury]: https://github.com/gnolang/gno/tree/5a51b55f4/examples/gno.land/r/gov/dao/treasury/v0
[txfees-todo]: https://github.com/gnolang/gno/blob/5a51b55f4/examples/gno.land/r/sys/txfees/txfees.gno#L1-L3
[unsafe-pkg]: https://github.com/gnolang/gno/tree/5a51b55f4/gnovm/stdlibs/chain/runtime/unsafe
[compat]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md
[compat-reflect]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L279
[compat-sort]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L364-L367
[contribs]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs
[d-amino]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/amino/README.md?plain=1#L5-L6
[d-amino-tx]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/baseapp.go#L920
[d-bptree-b]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/bptree/const.go#L6-L8
[d-cometbft]: https://github.com/cometbft/cometbft
[d-commit-hash]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/baseapp.go#L1049-L1063
[d-compat-cap]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L101
[d-compat-cgo]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L284
[d-compat-chan]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L97
[d-compat-complex]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L87
[d-compat-cutoff]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L8-L12
[d-compat-dot]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L51-L56
[d-compat-ed25519]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L326-L327
[d-compat-gnotypes]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L103-L110
[d-compat-go]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L35
[d-compat-nondet]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L122-L125
[d-compat-rand]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L358-L363
[d-compat-sha256]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L332-L333
[d-compat-shadow]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L60-L71
[d-compat-time]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/go-gno-compatibility.md?plain=1#L370-L374
[d-const-atone]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L200-L203
[d-const-gnot]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L172-L175
[d-const-launch-fees]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L172-L175
[d-const-photon]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L164-L168
[d-const-whitelist]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L126-L130
[d-cosmos-sdk]: https://github.com/cosmos/cosmos-sdk
[d-dbadapter-commit]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/store/dbadapter/store.go#L91-L97
[d-evpool-noop]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/bft/consensus/state.go#L91-L96
[d-evpool-wired]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/bft/node/node.go#L371
[d-genesis-check]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/gen-genesis.sh#L1126-L1131
[d-genesis-gnogenesis]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L60-L66
[d-genesis-readme]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L3
[d-genesis-restricted]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/gen-genesis.sh#L268-L271
[d-genesis-total]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/gen-genesis.sh#L240-L241
[d-genesis-vesting]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L16
[d-github-bot]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/github-bot
[d-gno-realms]: https://github.com/allinbits/gno-realms
[d-gnobro]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnobro
[d-gnoclient-readme]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/gnoclient/README.md?plain=1#L3
[d-gnodev]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnodev
[d-gnofaucet]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnofaucet
[d-gnogenesis]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnogenesis
[d-gnohealth]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnohealth
[d-gnohealth-ts]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnohealth/internal/timestamp/timestamp.go#L39-L40
[d-gnokeykc]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnokeykc
[d-gnokms]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnokms
[d-gnokms-gnokey]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnokms/internal/gnokey/gnokey.go#L33
[d-gnomd]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnomd
[d-gnomigrate]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gnomigrate
[d-gnomigrate-txs]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnomigrate/README.md?plain=1#L8
[d-gpao]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/gpao
[d-header-apphash]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/bft/types/block.go#L246
[d-ics-def]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L56-L57
[d-init-cur]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/op_call.go#L468-L476
[d-investor-unlock]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L136-L138
[d-mainnet-launch]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/README.md?plain=1#L5
[d-mainnet-upgrades]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/deployments/mainnet.gno.land/UPGRADES.md?plain=1#L3
[d-maplist]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/values.go#L990-L1008
[d-mount-base]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/gnoland/app.go#L120
[d-mount-bptree]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/gnoland/app.go#L116-L119
[d-preprocess-gas]: https://github.com/gnolang/gno/blob/5a51b55f4/gno.land/pkg/sdk/vm/keeper.go#L586-L596
[d-revive-enabled]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1898-L1914
[d-runinit]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/machine.go#L889-L907
[d-setobject-hash]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/store.go#L751-L760
[d-staging-archive]: https://github.com/gnolang/gno/blob/5a51b55f4/misc/loop/README.md?plain=1#L26
[d-state-size]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L181
[d-test-nobench]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/cmd/gno/test.go#L58-L60
[d-time-now]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/stdlibs/time/time.go#L10-L17
[d-tm2-principles]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/README.md?plain=1#L20-L24
[d-tx-archive]: https://github.com/gnolang/gno/tree/5a51b55f4/contribs/tx-archive
[d-uv-cross]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1831
[d-uv-istypednil]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1865
[d-uv-revive]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1889
[deduct-fees]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/sdk/auth/ante.go#L207
[evpool-add]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/pkg/bft/consensus/state.go#L1560
[genesis-allocation]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L114-L124
[gnobr]: https://github.com/gnolang/gno/blob/5a51b55f4/contribs/gnobr/README.md?plain=1#L1-L3
[gnoclient]: https://github.com/gnolang/gno/tree/5a51b55f4/gno.land/pkg/gnoclient
[go-mod]: https://github.com/gnolang/gno/blob/5a51b55f4/go.mod#L3
[ics-migration]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L190-L192
[issue-4907]: https://github.com/gnolang/gno/issues/4907
[laws]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/LAWS.md
[manifesto]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/MANIFESTO.md
[no-amendment]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L784-L785
[no-inflation]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L182-L183
[opcpu]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/machine.go#L1486-L1495
[ownership-doc]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/ownership.go#L19-L20
[security-guide]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/resources/gno-security-guide.md
[storage-price-rule]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L184-L188
[tm2]: https://github.com/gnolang/gno/tree/5a51b55f4/tm2
[tm2-independent]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/README.md?plain=1#L5
[tm2-proposed]: https://github.com/gnolang/gno/blob/5a51b55f4/tm2/README.md?plain=1#L45-L49
[uv-attach]: https://github.com/gnolang/gno/blob/5a51b55f4/gnovm/pkg/gnolang/uverse.go#L1846-L1853
[vesting]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/CONSTITUTION.md?plain=1#L132-L134
[whitepaper]: https://github.com/gnolang/gno/blob/5a51b55f4/docs/gnoland-whitepaper.tex
