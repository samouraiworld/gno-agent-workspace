# Gno.land Overview

Measured against [gnolang/gno at a7e4c34b0][gno-tree], 2026-05-25, the commit
[How the GnoVM works](gnovm-architecture.md) is pinned to. Every code link
points at that commit. Network facts, live parameters and external links were
checked on 2026-09-05 and carry that date.

## TLDR

Gno is Go, interpreted deterministically on a blockchain. A realm is a package
whose global variables persist between transactions and which renders its own
web page as markdown. The node is a minimal Tendermint fork, the interpreter
is the GnoVM, and the two meet in a keeper that turns each message into one
interpreter run. Betanet `gnoland1` and the rolling `staging` chain are live,
two fresh testnets named pearl and sapphire launched in August 2026, governance
is a tiered DAO with a two-thirds rule, and GNOT pays for gas and, refundably,
for stored bytes.

## What is Gno?

**Gno** is a deterministic, interpreted dialect of Go for smart contracts on
the **gno.land** blockchain. It is [modeled on Go 1.17][compat-go117]: no
generics, no goroutines, no channels. The language adds two builtin types,
`address` and `realm`, two untyped constant kinds, `bigint` and `bigdec`, and
the `cross` builtin that switches realm identity. The interpreter is the
GnoVM, described end to end in [How the GnoVM works](gnovm-architecture.md).

- **Deterministic**: no goroutines, channels, `unsafe`, `net` or `os`; floats
  run in software; maps iterate in insertion order.
- **Interpreted**: source runs on the GnoVM. Nothing is compiled to machine
  code, and the source is what is stored on chain.
- **Auditable**: every deployed package is readable on the chain's own web
  interface, tests included.
- **Persistent by default**: the global variables of a realm are saved when a
  call returns, without an ORM or a storage API.

File extension `.gno`. Module manifest [`gnomod.toml`][gnomod-example], which
names the module path and the Gno language version, [`0.9` at this
commit][gnover].

![The three package kinds, what each may do, and who may deploy under a
namespace.](figures/package-kinds.svg)

### Toolchain

Install everything from source:

```bash
git clone https://github.com/gnolang/gno.git && cd gno && make install
```

This builds `gno`, `gnodev`, `gnokey`, `gnoland` and `gnoweb`. Three of them
are daily tools, `gno`, `gnokey` and `gnodev`; `gnoland` and `gnoweb` run
underneath.

**`gno`** is the language tool. It tests, lints, formats and documents `.gno`
packages without a node, the way the `go` command does for Go. Its
subcommands at this commit, [from the binary's own help][gno-help]:

| Command | Purpose | Go equivalent |
| --- | --- | --- |
| `gno test` | Run `_test.gno` functions and `_filetest.gno` golden tests | `go test` |
| `gno fmt` | Format `.gno` files | `go fmt` |
| `gno lint` | Type-check and preprocess without running | `go vet` |
| `gno doc` | Show package documentation | `go doc` |
| `gno run` | Run a package's `main` or an expression | `go run` |
| `gno fix` | Rewrite source from older Gno versions | `go fix` |
| `gno mod` | Module maintenance, `gno mod init` among others | `go mod` |
| `gno repl` | Start a GnoVM read-eval-print loop | none |
| `gno tool transpile` | Translate Gno to Go source | none |
| `gno list`, `gno env`, `gno clean`, `gno bug`, `gno version` | Housekeeping | the same |

**`gnodev`** is the local development server: an in-memory `gnoland` node and
a `gnoweb` front end in one process, with hot reload. Write code, `gnodev`
redeploys it, open `http://localhost:8888` to see the realm rendered. The
`examples/` directory of the monorepo [is loaded unless `--minimal` is
passed][gnodev-minimal].

```bash
gnodev ./myrealm    # gnoweb at http://localhost:8888
```

| Feature | What it does |
| --- | --- |
| Hot reload | Watches the given directories and `examples/`, redeploys on change |
| State maintenance | Replays every transaction after a reload, so state survives |
| Premined balances | Every account in the local `gnokey` keybase [starts with 10T ugnot][gnodev-premine] |
| State export | Writes the current state as a genesis file |
| Resolvers | `root`, `local` and `remote`: the last [resolves imports from a running node][gnodev-resolvers], to test local code against on-chain dependencies |

Keys while it runs, [from its README][gnodev-keys]: `H` help, `A` account
balances, `R` reload, `P` cancel the last action, `N` redo it, `Ctrl+S` save
the state, `Ctrl+R` restore it, `E` export a genesis file, `Cmd+R` reset,
`Cmd+C` exit.

**`gnokey`** manages keys and talks to a node over RPC: deploy code, call
functions, send coins, query state. Addresses are bech32 with the `g1` prefix.
`gnodev` imports a default account named [`test1` at
`g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5`][default-account], funded like
every other local key.

```bash
gnokey add MyKey
# Save the mnemonic. Note your g1... address.
```

| Message | Command | Purpose |
| --- | --- | --- |
| `MsgAddPackage` | `gnokey maketx addpkg` | Deploy a package |
| `MsgCall` | `gnokey maketx call` | Call one exported crossing function |
| `MsgSend` | `gnokey maketx send` | Transfer coins |
| `MsgRun` | `gnokey maketx run` | Run a throwaway `main` against on-chain code |
| session messages | `gnokey maketx session create`, `revoke`, `revokeall` | [Session accounts][gnokey-session]: a key with an expiry and an allow-list of message kinds, signed for by a master key through `-master` |

Every `maketx` command takes `-gas-fee`, `-gas-wanted`, `-max-deposit` for
the storage deposit, `-send` for attached coins, `-chainid`, `-remote` and
`-broadcast`. `-simulate only` [estimates without broadcasting][addpkg-flags].

#### Query paths

```bash
gnokey query <path> -data "<data>" -remote "https://rpc.staging.gno.land:443"
```

The `vm` paths are [the ten cases of the keeper's query
handler][query-kinds]; the others belong to the `auth`, `bank` and `params`
modules.

| Path | Data | Returns |
| --- | --- | --- |
| `vm/qrender` | `<pkgpath>:<renderpath>` | The realm's `Render` output, markdown |
| `vm/qeval` | `<pkgpath>.<expression>` | Any expression, as text |
| `vm/qeval_json` | `<pkgpath>.<expression>` | The same, as JSON |
| `vm/qobject_json`, `vm/qobject_binary` | an object id | One persisted object, decoded or raw |
| `vm/qfuncs` | `<pkgpath>` | Exported function signatures, JSON |
| `vm/qfile` | `<pkgpath>` or `<pkgpath>/<file>` | File list, or one file's source |
| `vm/qdoc` | `<pkgpath>` | Package documentation, JSON |
| `vm/qpaths` | `<prefix>` | Package paths under the prefix |
| `vm/qstorage` | `<pkgpath>` | Bytes stored and deposit held by a realm |
| `auth/accounts/<addr>` | | Account: coins, public key, sequence |
| `auth/gasprice` | | The current minimum gas price |
| `bank/balances/<addr>` | | Coin balances |
| `params/<module>:p:<key>` | | One live chain parameter, see *Networks* |

**`gnoland`** is the node: consensus, blocks, persistence. `gnoland start`
runs one. In development `gnodev` runs it for you.

**`gnoweb`** renders realm markdown and browses on-chain source. It is what
gno.land, staging.gno.land and `gnodev` serve.

![From an editor to a rendered page: the loop a realm goes through, and which
tool runs each step.](figures/dev-workflow.svg)

## Core Concepts

### Realms (`r/`)

Stateful contracts with a unique path such as `gno.land/r/demo/counter`. A
realm has an address, can hold coins, and can expose `Render(path string)
string`, whose markdown gnoweb turns into the realm's page. That is how dapps
ship a user interface without a separate front end.

```go
func Render(path string) string {
    return "# Hello World\n\nThis is **markdown** rendered on-chain."
}
```

A public realm is immutable once deployed: [the keeper refuses a second
package at an existing path][keeper-exists]. Upgrade by convention, `v1`,
`v2`, or through a proxy realm such as `r/gov/dao`. A package whose manifest
says `private = true` is the exception: it [may be redeployed by a later
private package][keeper-private], [must be a realm][keeper-private-realm],
and other realms may not persist values of its types.

### Pure packages (`p/`)

Stateless libraries, `gno.land/p/nt/avl/v0` for instance. They hold no
persistent state, cannot declare crossing functions, and [cannot import a
realm][p-imports-r]. Because they are immutable, two realms can trust the same
`p/` code to mediate between them.

### Ephemeral packages (`e/`)

`gnokey maketx run` uploads a throwaway `main` package at
[`gno.land/e/<address>/run`][epath], runs it once and stores nothing. It is
how a user executes several calls in one transaction, or calls a function
with arguments that do not fit on a command line.

### Namespaces

The segment after the kind letter is the namespace. [Two shapes grant deploy
authority][namespace-rule]: the deployer's own address,
`gno.land/r/g1abc.../myrealm`, always; and a registered name,
`gno.land/r/alice/myrealm`, when `r/sys/users` maps that name to the deployer
as their current name. The check is a call the keeper makes into
[`r/sys/names`][names-verifier] on every upload. Names are registered through
[`r/sys/namereg/v1`][namereg], [free by default and priced by
GovDAO][register-price], and the two fresh testnets enforce the check [from
block one][pearl-genesis].

### Gas and storage

Two costs, both in `ugnot`, one millionth of a GNOT.

- **Gas** pays for computation and is spent. Each opcode, allocation and store
  access has a calibrated price, the mechanism is in [the GnoVM
  doc](gnovm-architecture.md#gas-and-memory). The whole `-gas-fee` is taken
  whatever the transaction consumed, [as the fee doc states][gas-charged] and
  [issue 3805][issue-3805] tracks.
- **Storage deposit** pays for bytes and comes back. When a call leaves a
  realm holding more bytes than before, the caller locks
  `delta × storage_price` into the realm's deposit address; freeing bytes
  refunds the same. The code default is [100 ugnot per byte][storage-price],
  which makes one GNOT buy ten kilobytes and one billion GNOT ten terabytes,
  the number the Constitution is built on.

![Four sources feed one gas meter. Only the storage deposit comes
back.](figures/gas-sources.svg)

### Learn by example

The [`r/docs`][r-docs] realm family is an on-chain index of small working
realms, each readable, deployable and modifiable: `hello`, `adder`,
`minisocial`, `avl_pager`, `routing`, `events`, `safeobjects`,
`soliditypatterns`, `txlink`, `charts`, `markdown` and more. It renders at
[staging.gno.land/r/docs](https://staging.gno.land/r/docs).

### Good practices

On-chain code is permanent and public. The rules that save the most:

- **Every write costs gas and bytes.** Batch writes, and free what you no
  longer need to get the deposit back.
- **Never put a secret in code.** Source is public, tests included.
- **Test before deploying.** `gno test` for logic, `gnodev` for the rendered
  result. A deployed path can never be reused.
- **Panics roll back the whole transaction.** Use them for access control and
  invalid input, not to report recoverable errors.
- **Validate every argument.** Anyone can call a public function.
- **Use `ownable`** from `p/nt/ownable/v0` rather than a hand-written admin
  check.
- **Prefer `avl.Tree` to a map** for anything that grows. A map loads whole;
  a tree loads one path per read, iterates in key order and pages through
  [`p/nt/avl/v0/pager`][pager].
- **Keep `Render` cheap.** It runs on every page view, under a query gas
  limit.
- **Read the caller from `cur`.** `cur.Previous()` is the unforgeable caller
  of a crossing function; the stack-walking `unsafe` package is for the rare
  case where transaction-level identity is what you want.

[Effective Gno](https://docs.gno.land/resources/effective-gno) goes deeper:
global variables, `init` as a constructor, package layout, safe objects, coins
versus GRC-20 tokens.

## Networks

Measured on 2026-09-05 by asking each RPC endpoint for its `status`.

| Network | Chain id | Web | RPC | Purpose |
| --- | --- | --- | --- | --- |
| Betanet | `gnoland1` | https://gno.land | `https://rpc.gno.land:443` | The main network. Height 3.66 million, node `v1.0.0-rc.0`. `ugnot` is a restricted denomination there, see below. |
| Staging | `staging` | https://staging.gno.land | `https://rpc.staging.gno.land:443` | Rolling testnet rebuilt from `master`; every package under `examples/` is redeployed each cycle |
| Pearl | `pearl-1` | https://pearl.testnets.gno.land | `https://rpc.pearl.testnets.gno.land:443` | Fresh chain, genesis 2026-08-27, three founding validators, [namespace enforcement on from block one][pearl-genesis] |
| Sapphire | `sapphire-1` | https://sapphire.testnets.gno.land | `https://rpc.sapphire.testnets.gno.land:443` | Fresh chain, two founding validators, [same package set][sapphire-genesis] |
| Local | `dev` | http://localhost:8888 | `http://127.0.0.1:26657` | `gnodev` |

Test11, test12 and test13 no longer answer. A third fresh chain, topaz, has
[its genesis prepared][topaz-genesis] and no reachable endpoint yet. Test
tokens come from [faucet.gno.land](https://faucet.gno.land), which asks for
the network.

Staging keeps its history through rebuilds by archiving every transaction,
pulling `master`, and replaying the archive into the new genesis, [as the
networks doc describes][staging-cycle]. A breaking change on `master` makes
old transactions fail to replay, so data can be lost, and heights and
timestamps cannot be relied on.

![The staging chain's rebuild loop: archive, pull master, replay into a new
genesis.](figures/staging-cycle.svg)

**Live parameters are not the code defaults.** Read one with
`gnokey query params/<module>:p:<key> -remote <rpc>`. Measured 2026-09-05:

| Parameter | Code default | Staging | Betanet |
| --- | --- | --- | --- |
| `vm:p:default_deposit` | [`600000000ugnot`][vm-params-defaults] | `600000000ugnot` | `600000000ugnot` |
| `vm:p:storage_price` | [`100ugnot`][storage-price] | `100ugnot` | `100ugnot` |
| `auth:p:initial_gasprice` | | `1ugnot` per `1000` gas | `1ugnot` per `1000` gas |
| `bank:p:restricted_denoms` | none | none | `ugnot` |
| `vm:p:chain_domain` | `gno.land` | `gno.land` | `gno.land` |

A restricted `ugnot` sends a storage-deposit refund to the storage fee
collector instead of the sender. `-gas-fee` is the whole fee for a
transaction; it is accepted when it covers `-gas-wanted` at the current price,
one ugnot per thousand gas on both networks today.

## Quickstart on Staging

Every command targets staging. Copy, paste, run.

**1. Create a key**

```bash
gnokey add MyKey
```

**2. Get test tokens** at [faucet.gno.land](https://faucet.gno.land), then
check them:

```bash
gnokey query bank/balances/<your-address> -remote "https://rpc.staging.gno.land:443"
```

**3. Write a realm**

```bash
mkdir myapp && cd myapp
gno mod init gno.land/r/<your-address>/myapp
```

Deploying under your own address needs no registration. A name such as
`gno.land/r/myname/...` needs the name registered to you first, see
*Namespaces*.

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

```bash
gno test .
gnodev .    # http://localhost:8888/r/<your-address>/myapp
```

**5. Deploy**

```bash
gnokey maketx addpkg \
  -pkgpath "gno.land/r/<your-address>/myapp" -pkgdir "." \
  -gas-fee 8000ugnot -gas-wanted 8000000 -max-deposit 600000000ugnot \
  -broadcast -chainid staging -remote "https://rpc.staging.gno.land:443" \
  MyKey
```

`-max-deposit` caps the storage deposit the upload may lock. Run with
`-simulate only` first to see the gas and bytes it will use.

**6. Call a function**

```bash
gnokey maketx call \
  -pkgpath "gno.land/r/<your-address>/myapp" -func "Increment" \
  -gas-fee 2000ugnot -gas-wanted 2000000 \
  -broadcast -chainid staging -remote "https://rpc.staging.gno.land:443" \
  MyKey
```

The keeper [writes the call as `pkg.Increment(cross, ...)` and substitutes
the transaction's signer for `cross`][call-origin], so `cur.Previous()`
inside `Increment` is your address.

**7. Query**

```bash
gnokey query vm/qrender \
  -data "gno.land/r/<your-address>/myapp:" -remote "https://rpc.staging.gno.land:443"
```

**8. Send tokens**

```bash
gnokey maketx send \
  -to "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5" -send "1000000ugnot" \
  -gas-fee 2000ugnot -gas-wanted 2000000 \
  -broadcast -chainid staging -remote "https://rpc.staging.gno.land:443" \
  MyKey
```

## Standard library and notable packages

### The chain packages

The old `std` package is gone. Chain access lives in `chain`, `chain/runtime`,
`chain/runtime/unsafe`, `chain/banker` and `chain/params`, all
[under `gnovm/stdlibs`][stdlibs-chain]. The lists below are the exported
names at this commit.

| Package | Exports | Use |
| --- | --- | --- |
| [`chain`][chain-pkg] | `Coin`, `Coins`, `NewCoin`, `NewCoins`, `CoinDenom`, `Emit`, `PackageAddress`, `DeriveStorageDepositAddr`, `PubKeyAddress` | Coins, events, address derivation |
| [`chain/runtime`][runtime-pkg] | `AssertOriginCall`, `ChainID`, `ChainDomain`, `ChainHeight`, `GetSessionInfo`, `NewRealm` | Chain facts, and whether the call came straight from a transaction |
| [`chain/runtime/unsafe`][unsafe-pkg] | `CurrentRealm`, `PreviousRealm`, `OriginCaller`, `OriginSend` | Stack-walking identity. Named `unsafe` because [a non-crossing helper sees the outermost crossing realm, not its caller][unsafe-doc]; prefer `cur.Previous()` |
| [`chain/banker`][banker-pkg] | `NewBanker`, `NewReadonlyBanker`, `IsCanonical`, the `Banker` interface | Moving coins, in [four privilege levels][banker] |
| [`chain/params`][params-pkg] | `SetString`, `SetInt64`, `SetBool`, ... | Writing chain parameters, for the `sys` realms |
| `testing` (tests only) | `SetRealm`, `SetOriginCaller`, `SetOriginSend`, `SkipHeights`, `IssueCoins`, `NewUserRealm`, `NewCodeRealm` | [Rewriting the mock context][testing-overrides] inside `gno test` |

The builtin type `address` carries the bech32 `g1...` string. The builtin
type `realm` is what `cur` is: `cur.Address()`, `cur.PkgPath()`,
`cur.Previous()`, and the `IsUser`, `IsRealm`, `IsCurrent` classifiers.

Of Go's standard library, [these packages exist at this commit][stdlib-list]:

```
bufio bytes chain crypto/bech32 crypto/chacha20 crypto/cipher crypto/ed25519
crypto/sha256 crypto/subtle encoding encoding/base32 encoding/base64
encoding/binary encoding/csv encoding/hex errors hash hash/adler32 html
internal/bytealg io math math/bits math/overflow math/rand net/url path
regexp regexp/syntax sort strconv strings sys/params time unicode
unicode/utf16 unicode/utf8
```

`fmt` [exists only under test][fmt-tests]; on chain use `ufmt`.

### Notable pure packages

Paths are under `gno.land/p/`, in the monorepo's [`examples/`][examples-p].

| Package | Path | What it is |
| --- | --- | --- |
| **avl** | `nt/avl/v0` | The ordered key-value store realms are built on. Lazy-loaded, iterable in key order. |
| **avl pager** | `nt/avl/v0/pager` | Pagination over a tree, for `Render`. |
| **bptree** | `nt/bptree/v0` | A B+ tree, the store behind GovDAO's member tiers. |
| **ufmt** | `nt/ufmt/v0` | The on-chain `fmt`. |
| **ownable** | `nt/ownable/v0` | One owner, transferable, with `AssertOwnedByPrevious`. |
| **pausable** | `nt/pausable/v0` | Pause and resume a realm. |
| **mux** | `nt/mux/v0` | A router for `Render` paths, in the shape of `http.ServeMux`. |
| **seqid** | `nt/seqid/v0` | Sequential ids that sort correctly as tree keys. |
| **uassert**, **urequire** | `nt/uassert/v0`, `nt/urequire/v0` | Test assertions, the second failing fast. |
| **commondao**, **poa**, **treasury** | `nt/commondao`, `nt/poa`, `nt/treasury` | DAO, proof-of-authority and treasury building blocks under the `nt` namespace. |
| **addrset** | `moul/addrset` | A set of addresses. |
| **mgroup** | `n2p5/mgroup` | A managed group: owner, backup owners, members. |
| **daokit** | `samcrew/daokit` | A DAO framework with role-based membership. |

### Token standards

| Standard | Path | Kind |
| --- | --- | --- |
| **GRC-20** | `gno.land/p/demo/tokens/grc20` | Fungible token |
| **GRC-721** | `gno.land/p/demo/tokens/grc721` | Non-fungible token |
| **GRC-1155** | `gno.land/p/demo/tokens/grc1155` | Multi-token |

### Notable realms

| Realm | Path | What it is |
| --- | --- | --- |
| **wugnot** | `gno.land/r/gnoland/wugnot` | [Wrapped GNOT](https://gno.land/r/gnoland/wugnot) as a GRC-20 |
| **boards2** | `gno.land/r/gnoland/boards2/v1` | [Discussion boards](https://gno.land/r/gnoland/boards2/v1) with moderation |
| **grc20reg** | `gno.land/r/demo/defi/grc20reg` | [Registry](https://gno.land/r/demo/defi/grc20reg) of GRC-20 tokens |
| **gov/dao** | `gno.land/r/gov/dao` | [The governance proxy](https://gno.land/r/gov/dao), see *Governance* |
| **sys/validators** | `gno.land/r/sys/validators/v2`, `v3` | Validator set changes through GovDAO proposals; [the fresh chains run v3][pearl-genesis] |
| **sys/params** | `gno.land/r/sys/params` | Chain parameters, set by proposal |
| **sys/users**, **sys/namereg** | `gno.land/r/sys/users`, `gno.land/r/sys/namereg/v1` | Name to address registry, and the realm that registers names for a price |
| **sys/names** | `gno.land/r/sys/names` | The namespace verifier the keeper calls on every upload |
| **sys/txfees**, **sys/rewards** | `gno.land/r/sys/txfees`, `gno.land/r/sys/rewards` | Fee distribution and contribution rewards; [both are stubs at this commit][txfees-todo] |
| **valopers** | `gno.land/r/gnops/valopers` | Validator operator profiles, seeded at genesis on the fresh chains |
| **disperse** | `gno.land/r/demo/disperse` | [Batch sends](https://staging.gno.land/r/demo/disperse) of coins and tokens |
| **chess** | `gno.land/r/morgan/chess` | [A chess server](https://staging.gno.land/r/morgan/chess) with a lobby |
| **hor** | `gno.land/r/leon/hor` | [Hall of realms](https://staging.gno.land/r/leon/hor), an exhibition of community realms |

The last three render on staging; betanet does not carry them.

## Interrealm programming

Gno is a multi-user language: importing another user's realm uses the same
syntax as importing a library. Two questions are kept apart inside the VM,
[per the interrealm specification][interrealm-v2]: who is acting, the
identity, and whose storage is being written, the storage realm. The
mechanics are in [the GnoVM
doc](gnovm-architecture.md#interrealm-identity-and-authority); this is the
builder's view.

### Crossing functions and `cur`

A crossing function has `cur realm` as its first parameter. It is the only
kind of function a transaction can call, and the only kind that can tell who
called it. Call it as `f(cross(cur), args...)` to switch identity to the
callee's realm, or as `f(cur, args...)` inside its own realm to keep the
current one. The [examples use `cross(cur)` about two thousand
times][cross-count]; the bare `cross` in older text is the transaction-level
form the keeper writes for you.

Each crossing call mints a new `cur` whose `Previous()` is the caller's realm.
At the root, the keeper [substitutes the transaction signer][call-origin]. So
`cur.Previous()` is a linked list back to the user, and cannot be forged from
a helper.

![cur is a linked list of identities built by crossing calls, back to the
transaction signer.](figures/cur-chain.svg)

```go
// gno.land/r/alice/bank
package bank

type Token struct{ Balance int }

// Crossing: called as bank.Debit(cross(cur), t, 10) from another realm.
func Debit(cur realm, t *Token, amount int) {
    caller := cur.Previous()     // the realm that crossed into us
    // t.Balance -= amount       // panics: t is a readonly view of bob's object
    _ = caller
}

// Non-crossing: runs with the caller's storage, so it may write t.
func DebitLocal(t *Token, amount int) {
    t.Balance -= amount
}
```

```go
// gno.land/r/bob/app
package app

import "gno.land/r/alice/bank"

var myToken = bank.NewToken(100) // lives in bob's realm

func Spend(cur realm) {
    bank.Debit(cross(cur), myToken, 10) // alice sees bob as the caller
    bank.DebitLocal(myToken, 10)        // same identity; writes bob's object
}
```

`myToken` comes from a constructor because a realm [may only build another
realm's types through that realm's own functions][checkconstruction]: a
`bank.Token{}` literal in `app` would panic at the allocation.

### What the VM decides for each call

![Six calls a realm can make against another realm's code and storage, and
what the VM decides for each.](figures/interrealm-cases.svg)

The storage realm for a call that does not cross follows [three borrow
rules][borrow-rules]: a realm's own code writes its own storage; a `p/`
method on a receiver owned by a realm writes that realm's storage, which is
how `avl.Tree.Set` works; a closure made by a `p/` package writes the storage
that was active when it was created. Three gates then refuse the rest: values
read from another realm are [readonly][readonlypanic], a real object of
another realm [cannot be written][isreadonlyby], and another realm's types
[cannot be constructed][checkconstruction].

### Why explicit crossing

In Solidity every external call silently shifts `msg.sender`, and the callee
may call back into the caller before it finishes: the reentrancy bug behind
the 2016 DAO drain. In Gno the identity change is written in the source, and
a callee cannot write the caller's objects at all. Even if it calls back, it
needs a crossing function the caller exported, and that function sees the
callee, not the user, as its caller. A panic that crosses a realm boundary
[aborts the whole transaction][panic-boundary], so a callee cannot leave a
caller half-written either.

### Rules to keep

- A function meant for users **must** be crossing. `gnokey maketx call`
  [refuses a non-crossing function][keeper-crossing-check].
- Methods are usually non-crossing: they work on the receiver wherever it
  lives.
- A `p/` package cannot declare a crossing function and cannot import a realm.
- Another realm's objects are read-only from outside. To change one, call a
  function that realm exports.
- `unsafe.PreviousRealm` and `unsafe.CurrentRealm` walk the stack and
  [can be fooled from a helper][unsafe-doc]. Use `cur`.

## Governance (GovDAO)

GovDAO selects validators, spends the treasuries and sets chain parameters.
The chain runs as proof of authority: members vote validators in and out. The
[Constitution][constitution] describes a proof-of-contribution philosophy; on
chain, membership is what the tiers below encode, and a fresh chain starts
[with a single T1 member][pearl-genesis] who proposes the rest.

The code is [`r/gov/dao`][gov-dao], a proxy that [delegates to a pluggable
implementation][dao-proxy], today [`r/gov/dao/v3/impl`][dao-impl], with
members in [`r/gov/dao/v3/memberstore`][memberstore].

### Tiers

[Measured from the member store's `init`][tiers]:

| Tier | Base power | Invitation points | Size rule | Power cap |
| --- | --- | --- | --- | --- |
| **T1** | 3 | 3 | at least 70 members | none |
| **T2** | 2 | 2 | between a quarter and twice the T1 count | the tier's total power is capped at two thirds of T1's; the per-member power shrinks to fit |
| **T3** | 1 | 1 | none | capped at one third of T1's total |

T1 and T2 members are added by proposal, [and only to those two
tiers][add-member-tiers]; a T1 or T2 member [adds a T3 member directly by
spending one invitation point][add-member-direct]. A member proposal
[restricts the vote to the target tier][filter-by-tier].

### Proposals

Every proposal carries an executor, a callback that runs when it passes. The
lifecycle, [from the implementation][accept-deny]:

1. A member creates a proposal with a title, a description and an executor.
2. Members vote YES, NO or ABSTAIN.
3. Percentages are computed against the total eligible voting power, so not
   voting weighs like a NO.
4. At [66.66 percent][supermajority] YES the proposal is accepted and the
   executor runs. At 66.66 percent NO it is denied.

The threshold is [the `Law` value][supermajority], itself changeable by
proposal. The request types [at this commit][prop-requests]: change the law,
upgrade the DAO implementation, add, withdraw or promote a member, pay from
the treasury, update the treasury's GRC-20 token list.

![The GovDAO tiers, their power caps, and a proposal's path from creation to
execution.](figures/govdao.svg)

### What GovDAO controls

| Area | Realm |
| --- | --- |
| Validators | [`r/sys/validators/v2`][validators-v2], and `v3` on the fresh chains |
| Chain parameters, gas price floors, transfer locks | [`r/sys/params`][sys-params] |
| Treasury, ugnot and GRC-20 payments | [`r/gov/dao/v3/treasury`][treasury] |
| Name registration: controllers, prices, emergency pause | [`r/sys/users`][sys-users], [`r/sys/namereg/v1`][namereg], [`r/sys/names`][names-verifier] |

Deploying under your own address is never governed; a registered name is.

[Memba](https://memba.samourai.app) is a web interface for proposals, votes
and membership, by [Samourai](https://github.com/samouraiworld/memba).

## Architecture

### Cosmos, Tendermint, CometBFT and Tendermint2

**BFT**, Byzantine fault tolerance, is the property of a distributed system
that reaches agreement while up to one third of its participants are faulty
or malicious. The name is Lamport's [Byzantine Generals
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
| **Round** | One attempt at consensus within a height. Round 0 failing, round 1 tries with a new proposer. |
| **Finality** | A committed block is never reverted. Proof-of-work chains only get there probabilistically. |
| **Halt** | The chain stops producing blocks: fewer than two thirds of validators online, or a determinism bug making validators disagree. Nothing committed is lost. |
| **Supermajority** | More than two thirds of the voting power, needed to prevote and precommit. |
| **Voting power** | A validator's weight. Proof of stake ties it to stake; gno.land assigns it by governance. |
| **Liveness** | Blocks keep coming as long as enough honest validators are online. |
| **Safety** | Validators never commit conflicting blocks at the same height, as long as fewer than a third are Byzantine. |
| **Slashing** | A penalty on a validator's stake for misbehaviour such as signing two blocks at one height. |

**Tendermint** (2014, Jae Kwon) was the first practical BFT engine for
blockchains. It separates consensus from the application through ABCI, the
Application Blockchain Interface: consensus orders transactions, the
application says what they mean.

**Cosmos SDK** is the framework for ABCI applications on Tendermint. The
Cosmos Hub, Osmosis and dYdX are Cosmos SDK chains.

**CometBFT** is the maintained fork of Tendermint most Cosmos chains run
today, with ABCI++, gRPC and Protobuf, and the dependencies that come with
them.

**Tendermint2**, TM2, is Jae Kwon's minimal fork of the original Tendermint,
kept in the monorepo at [`tm2/`][tm2]. It strips gRPC, Protobuf, viper, cobra
and Prometheus: the code is the specification, every dependency is audited
and vendored. [Once gno.land is on mainnet, TM2 is to operate independently
at `tendermint/tendermint2`][tm2-independent]. Its README lists what is
[still proposed][tm2-proposed], among it replacing the evidence reactor by an
evidence mempool lane. The consensus state [still hands a duplicate vote to an
evidence pool][evpool-add], but no penalty follows: there is no slashing.

A standalone consensus library, libtm, lived under `tm2/pkg/libtm` for a
while and [was removed in pull request 5534][libtm-removed] after an analysis
found integration not viable.

In short: Cosmos SDK chains run CometBFT, gno.land runs Tendermint2, both
implement the same algorithm with instant finality, and TM2 trades features,
slashing included, for simplicity.

### How the layers fit together

![From a signed transaction to the database: each layer hands one message to
the next.](figures/vm-stack.svg)

Tendermint2 orders transactions and hands each block to the application over
ABCI: `CheckTx` validates, `DeliverTx` executes through the GnoVM, `Commit`
persists. The application is a tm2 `BaseApp` whose `vm` route reaches the
keeper, and the keeper builds one interpreter per message. After each
transaction, every new or changed object of the realms touched is serialized
and written; the Merkle root of the IAVL store is what validators compare at
commit. The steps are in [the GnoVM
doc](gnovm-architecture.md#how-gnoland-drives-the-vm).

[Reaching Consensus](https://gno.land/r/gnoland/blog:p/reaching-consensus) on
the gno.land blog is a practical walk through TM2 consensus; [the Tendermint
paper](https://arxiv.org/pdf/1807.04938.pdf) is the algorithm.

## Tokenomics

Three tokens in the wider ecosystem, [per the Constitution][constitution]:

| Token | Role |
| --- | --- |
| **ATONE** | Staking and governance on Atom.One, which is to secure gno.land |
| **PHOTON** | Gas on Atom.One, and the token gno.land will pay Atom.One with |
| **GNOT** | Storage deposit token on gno.land: it pays for persistent bytes |

### GNOT

[Genesis allocation][genesis-allocation], 1.333 billion GNOT in total:

| Allocation | GNOT |
| --- | --- |
| Airdrop 1, from a partial Cosmos governance snapshot | 350 M |
| Airdrop 2, from an AtomOne snapshot before launch | 231 M |
| Core treasury | 40 M |
| Ecosystem treasury | 60 M |
| Validator treasury | 20 M |
| Investors | 300 M |
| NT, LLC | 332 M |

- On chain the unit is `ugnot`: one GNOT is one million ugnot.
- [GNOT does not inflate][no-inflation]: the supply never exceeds 1.333
  billion, and [no amendment may raise it][no-amendment].
- [One billion GNOT corresponds to ten terabytes of state][ten-tb], which is
  the code default of 100 ugnot per byte.
- The storage price [never increases and may fall by at most ten percent a
  year][storage-price-rule], and not while used storage exceeds a consumer
  drive.
- Genesis allocations [vest over thirteen months][vesting]: seven percent
  when GNOT becomes transferable, seven percent a month, nine in the last.

### How the tokens relate

Gno.land launches on its own validators, then [migrates to Atom.One
interchain security][ics-migration] when GovDAO decides it is ready. After
that ATONE secures, PHOTON pays Atom.One for compute, GNOT pays for storage.
Until then, and on every testnet, `ugnot` pays for both gas and storage.

![Where a transaction's ugnot goes: the fee to the collector, the deposit
into the realm's deposit address and back.](figures/gnot-flows.svg)

## Go versus Gno

Gno [follows the Go 1.17 specification][compat-go117]; the VM is built with
[Go 1.25][go-mod]. The [compatibility table][compat] is the reference. What
is missing:

| Feature | Status |
| --- | --- |
| Goroutines, `go` | Missing, [after launch][compat-go117] |
| Channels, `select` | Missing, after launch |
| Generics | [Not implemented][issue-5063] |
| `unsafe`, `cgo` | Never |
| `net`, `os`, `syscall` | Non-deterministic, absent |
| `complex64`, `complex128` | Missing |
| `reflect` | [Listed as todo][compat-reflect] |

Behavioural differences that matter:

- `time.Now` returns the block time.
- `fmt` [exists only in tests][fmt-tests]; use `ufmt` on chain.
- `sort.Slice` [is missing][compat-sort]: implement `sort.Interface` and call
  `sort.Sort`.
- `crypto/sha256` has `Sum256` only; `crypto/ed25519` has `Verify` only.
- `init` runs once, when the package is deployed, with the deployer as the
  previous realm. It is a constructor, not a program start.
- `panic` is control flow: it aborts and rolls back the transaction.
- Global variables are the storage. The VM persists them.
- Maps iterate in insertion order, deterministically, but load whole; prefer
  `avl.Tree` for state that grows.
- `[]byte(s)` has capacity equal to its length.
- Import paths are `gno.land/p/...` and `gno.land/r/...`, never
  `github.com/...`. The manifest is `gnomod.toml`.

Gno-only additions: `bigint` and `bigdec`, `cross`, `realm`, `address`, and
the `revive` and `istypednil` builtins.

## Known limitations

Gno.land is release-candidate software: betanet runs `v1.0.0-rc.0`. The gaps
worth knowing, each anchored on the code that says so.

**Consensus.** No slashing, no misbehaviour penalty. A duplicate vote [is
handed to an evidence pool][evpool-add] and goes no further; the evidence
reactor [is to be replaced by a mempool lane][tm2-proposed] that is not
built.

**Fees.** Users [pay the whole `gas-fee` they offered][gas-charged], whatever
the transaction used, [issue 3805][issue-3805]. Fees [go to the fee
collector][deduct-fees] and stay there: the distribution realm [is a
placeholder][txfees-todo]. Gas costs themselves are no longer placeholders:
each opcode carries [a calibrated constant][opcpu], and preprocessing is
metered since [issue 4820][issue-4820] closed.

**Persistence.** The Merkle index of escaped objects [is a
stub][savenewescaped], so a proof over a cross-realm object is not available.
Object graphs with cycles [are not supported][ownership-doc]. `attach`
[panics][uv-attach].

**Language.** No goroutines, channels or generics, no `reflect`. The IBC
meta issue [is closed][issue-4907] and no IBC module exists in the tree.

## Additional tools

Everything under [`contribs/`][contribs] at this commit, plus one external
repository:

| Tool | What it does |
| --- | --- |
| [gnodev][contribs] | The development server, above |
| [gnofaucet][contribs] | A faucet server for test tokens |
| [gnokms][contribs] | Key management for validators: HSM and cloud KMS backends |
| [gnokeykc][contribs] | OS keychain integration for `gnokey` |
| [gnogenesis][contribs] | Builds and edits `genesis.json`: validators, balances, transactions; the fresh chains are built with it |
| [gnobro][contribs] | A terminal browser for realms |
| [gnobr][gnobr] | Rolls a node back to a height and replays blocks locally |
| [gnohealth][contribs] | Health checks for a node |
| [gnomd][contribs] | Renders Gno markdown in the terminal |
| [gnomigrate][contribs] | Migrates old on-chain data formats |
| [tx-archive][contribs] | Backs up and restores transactions; what keeps staging's history |
| [github-bot][contribs] | Pull request automation for the monorepo |
| [tx-indexer](https://github.com/gnolang/tx-indexer) | Indexes blocks, transactions and events for querying |

## Development environment

### Editors

| Editor | Plugin |
| --- | --- |
| VS Code | [Gno for VS Code](https://marketplace.visualstudio.com/items?itemName=Gnoverse.gnolang) |
| NeoVim | [gno.nvim](https://github.com/x1unix/gno.nvim) |
| Any editor | [gnopls](https://github.com/gnoverse/gnopls), a language server |
| Any editor | `gno fmt`, or `gofmt` directly on `.gno` files |

### Clients

| Library | Language | What it is |
| --- | --- | --- |
| [gnoclient][gnoclient] | Go | The client the tooling uses |
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
| Memba | [memba.samourai.app](https://memba.samourai.app) | Multisig wallet and DAO governance: GovDAO proposals and votes, validator dashboard, contributor analytics, by [Samourai](https://github.com/samouraiworld/memba) |

Community realms and projects: [boards2](https://gno.land/r/gnoland/boards2/v1)
on betanet; the [hall of realms](https://staging.gno.land/r/leon/hor),
[chess](https://staging.gno.land/r/morgan/chess) and
[disperse](https://staging.gno.land/r/demo/disperse) on staging;
[Gnoswap](https://github.com/gnoswap-labs/gnoswap), an automated market maker
by Onbloc, whose site did not answer on 2026-09-05.

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
[GitHub](https://github.com/gnolang), [X](https://twitter.com/_gnoland).

*Kept in
[gno-agent-workspace](https://github.com/samouraiworld/gno-agent-workspace).*

[gno-tree]: https://github.com/gnolang/gno/tree/a7e4c34b0
[compat-go117]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/go-gno-compatibility.md#L3
[compat]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/go-gno-compatibility.md
[compat-reflect]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/go-gno-compatibility.md#L236
[compat-sort]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/go-gno-compatibility.md#L288-L289
[gnomod-example]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gnoland/wugnot/gnomod.toml
[gnover]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/gnomod.go#L43
[gno-help]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/cmd/gno/main.go
[gnodev-minimal]: https://github.com/gnolang/gno/blob/a7e4c34b0/contribs/gnodev/README.md#L38-L39
[gnodev-premine]: https://github.com/gnolang/gno/blob/a7e4c34b0/contribs/gnodev/setup_node.go#L25
[gnodev-resolvers]: https://github.com/gnolang/gno/blob/a7e4c34b0/contribs/gnodev/README.md#L59-L72
[gnodev-keys]: https://github.com/gnolang/gno/blob/a7e4c34b0/contribs/gnodev/README.md#L24-L35
[default-account]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/integration/node_testing.go#L25-L27
[gnokey-session]: https://github.com/gnolang/gno/blob/a7e4c34b0/tm2/pkg/crypto/keys/client/session.go
[addpkg-flags]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/keyscli/addpkg.go
[query-kinds]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/sdk/vm/handler.go#L74-L83
[keeper-exists]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/sdk/vm/keeper.go#L623-L625
[keeper-private]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/sdk/vm/keeper.go#L660-L662
[keeper-private-realm]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/sdk/vm/keeper.go#L663-L665
[keeper-crossing-check]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/sdk/vm/keeper.go#L787-L790
[call-origin]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/sdk/vm/keeper.go#L853-L861
[p-imports-r]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/preprocess.go#L5389-L5393
[epath]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/gno-packages.md#L37-L47
[namespace-rule]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/sys/names/verifier.gno#L1-L17
[names-verifier]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/sys/names/verifier.gno#L75-L77
[namereg]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/r/sys/namereg/v1
[register-price]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/sys/namereg/v1/users.gno#L15-L18
[pearl-genesis]: https://github.com/gnolang/gno/blob/chain/pearl/misc/deployments/pearl.gno.land/README.md
[sapphire-genesis]: https://github.com/gnolang/gno/blob/chain/sapphire/misc/deployments/sapphire.gno.land/README.md
[topaz-genesis]: https://github.com/gnolang/gno/blob/chain/topaz/misc/deployments/topaz.gno.land/README.md
[staging-cycle]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/gnoland-networks.md#L21-L67
[gas-charged]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/gas-fees.md#L43
[issue-3805]: https://github.com/gnolang/gno/issues/3805
[issue-4820]: https://github.com/gnolang/gno/issues/4820
[issue-5063]: https://github.com/gnolang/gno/issues/5063
[issue-4907]: https://github.com/gnolang/gno/issues/4907
[storage-price]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/sdk/vm/params.go#L22
[vm-params-defaults]: https://github.com/gnolang/gno/blob/a7e4c34b0/gno.land/pkg/sdk/vm/params.go#L18-L23
[r-docs]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/r/docs
[pager]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/p/nt/avl/v0/pager
[stdlibs-chain]: https://github.com/gnolang/gno/tree/a7e4c34b0/gnovm/stdlibs/chain
[chain-pkg]: https://github.com/gnolang/gno/tree/a7e4c34b0/gnovm/stdlibs/chain
[runtime-pkg]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/stdlibs/chain/runtime/native.gno#L3-L11
[unsafe-pkg]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno
[unsafe-doc]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L1-L11
[banker-pkg]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/stdlibs/chain/banker/banker.gno
[banker]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/stdlibs/chain/banker/banker.gno
[params-pkg]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/stdlibs/chain/params/params.gno
[testing-overrides]: https://github.com/gnolang/gno/tree/a7e4c34b0/gnovm/tests/stdlibs/testing
[stdlib-list]: https://github.com/gnolang/gno/tree/a7e4c34b0/gnovm/stdlibs
[fmt-tests]: https://github.com/gnolang/gno/tree/a7e4c34b0/gnovm/tests/stdlibs/fmt
[examples-p]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/p
[txfees-todo]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/sys/txfees/txfees.gno#L1-L3
[interrealm-v2]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/gno-interrealm-v2.md
[security-guide]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/resources/gno-security-guide.md
[cross-count]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples
[checkconstruction]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/alloc.go#L410-L440
[borrow-rules]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/machine.go#L2323-L2387
[readonlypanic]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/machine.go#L2629-L2637
[isreadonlyby]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/ownership.go#L441-L536
[panic-boundary]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/op_call.go#L532-L547
[constitution]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/CONSTITUTION.md
[laws]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/LAWS.md
[manifesto]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/MANIFESTO.md
[whitepaper]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/gnoland-whitepaper.tex
[gov-dao]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/r/gov/dao
[dao-proxy]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gov/dao/proxy.gno#L146-L170
[dao-impl]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/r/gov/dao/v3/impl
[memberstore]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/r/gov/dao/v3/memberstore
[tiers]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L30-L90
[add-member-tiers]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L53-L61
[add-member-direct]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gov/dao/v3/impl/impl.gno#L26-L46
[filter-by-tier]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L86-L91
[accept-deny]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L124-L138
[supermajority]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gov/dao/v3/impl/impl.gno#L12-L16
[prop-requests]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L17-L183
[validators-v2]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/r/sys/validators/v2
[sys-params]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/r/sys/params
[treasury]: https://github.com/gnolang/gno/tree/a7e4c34b0/examples/gno.land/r/gov/dao/v3/treasury
[sys-users]: https://github.com/gnolang/gno/blob/a7e4c34b0/examples/gno.land/r/sys/users/admin.gno
[tm2]: https://github.com/gnolang/gno/tree/a7e4c34b0/tm2
[tm2-independent]: https://github.com/gnolang/gno/blob/a7e4c34b0/tm2/README.md#L5
[tm2-proposed]: https://github.com/gnolang/gno/blob/a7e4c34b0/tm2/README.md#L30-L49
[evpool-add]: https://github.com/gnolang/gno/blob/a7e4c34b0/tm2/pkg/bft/consensus/state.go#L1541
[libtm-removed]: https://github.com/gnolang/gno/pull/5534
[genesis-allocation]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/CONSTITUTION.md#L114-L124
[no-inflation]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/CONSTITUTION.md#L178-L179
[no-amendment]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/CONSTITUTION.md#L779
[ten-tb]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/CONSTITUTION.md#L177
[storage-price-rule]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/CONSTITUTION.md#L180-L183
[vesting]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/CONSTITUTION.md#L132-L134
[ics-migration]: https://github.com/gnolang/gno/blob/a7e4c34b0/docs/CONSTITUTION.md#L186-L188
[go-mod]: https://github.com/gnolang/gno/blob/a7e4c34b0/go.mod#L3
[deduct-fees]: https://github.com/gnolang/gno/blob/a7e4c34b0/tm2/pkg/sdk/auth/ante.go#L179
[opcpu]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/machine.go#L1371-L1377
[savenewescaped]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/realm.go#L1053-L1056
[ownership-doc]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/ownership.go#L36-L38
[uv-attach]: https://github.com/gnolang/gno/blob/a7e4c34b0/gnovm/pkg/gnolang/uverse.go#L1465
[contribs]: https://github.com/gnolang/gno/tree/a7e4c34b0/contribs
[gnobr]: https://github.com/gnolang/gno/blob/a7e4c34b0/contribs/gnobr/README.md#L1-L3
[gnoclient]: https://github.com/gnolang/gno/tree/a7e4c34b0/gno.land/pkg/gnoclient
