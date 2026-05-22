# Gno.land Overview

## What is Gno?

**Gno** is a deterministic, interpreted variant of Go for smart contracts on the
**gno.land** blockchain. Gno uses a real programming language (Go) instead of a
domain-specific one (Solidity, Move).

- **Deterministic**: no goroutines, channels, `unsafe`, `net`, or `os`
- **Interpreted**: code runs on the GnoVM, not compiled to machine code
- **Fully auditable**: all on-chain code is human-readable source
- **Go-compatible**: if you know Go, you mostly know Gno
- **Auto-persistence**: global variables in realms are saved automatically between transactions

File extension: `.gno` | Module config: `gnomod.toml`

### Toolchain

Install all tools from source:

```bash
git clone https://github.com/gnolang/gno.git && cd gno && make install
```

This builds and installs `gno`, `gnodev`, `gnokey`, `gnoland`, and `gnoweb`.

There are five tools. Three of them (`gno`, `gnokey`, `gnodev`) are what you use
day-to-day. The other two (`gnoland`, `gnoweb`) run under the hood.

**`gno`** is the language tool. It compiles, tests, lints, and formats `.gno`
files. It does not need a running node. Think of it as the `go` CLI for Gno.

| Command       | Description              | Go Equivalent  |
|---------------|--------------------------|----------------|
| `gno test`    | Run tests                | `go test`      |
| `gno fmt`     | Format `.gno` files      | `go fmt`       |
| `gno lint`    | Lint for common issues   | `golint`       |
| `gno doc`     | Show package docs        | `go doc`       |
| `gno mod init`| Init a new module        | `go mod init`  |
| `gno run`     | Run a Gno program        | `go run`       |

**`gnodev`** is the local development server. It starts an in-memory
`gnoland` node and a `gnoweb` frontend in a single process, with hot
reload. You write code, `gnodev` detects changes and redeploys automatically.
Open `http://localhost:8888` to see your realm rendered.

```bash
gnodev    # opens gnoweb at http://localhost:8888
```

| Feature | Description |
|---------|-------------|
| Hot reload | Watches for file changes, redeploys automatically |
| Transaction replay | Preserves state across reloads |
| Premined balances | Detects local `gnokey` keybase, premines 10T ugnot to all addresses |
| Lazy loading | Packages loaded on demand |
| State export | Export current state as a genesis `.jsonl` file |
| Remote resolver | Test local code against on-chain dependencies |

**Interactive keys:** `H` help | `R` reload | `A` accounts | `P`/`N` undo/redo TX | `E` export | `Ctrl+S` save | `Ctrl+R` reset

**`gnokey`** manages keys and sends transactions to a node (deploy code, call
functions, query state). It connects to any node via RPC, whether local or
remote.

Addresses use bech32 with `g1` prefix. When you start `gnodev`, it imports a
default account named **`devtest`** at address `g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5`,
pre-funded with 10T ugnot.

To create your own key:

```bash
gnokey add MyKey
# Save the mnemonic. Note your g1... address.
```

| Message Type | Command                | Purpose                        |
|--------------|------------------------|--------------------------------|
| AddPackage   | `gnokey maketx addpkg` | Deploy code on-chain           |
| Call         | `gnokey maketx call`   | Call a realm function          |
| Send         | `gnokey maketx send`   | Transfer coins                 |
| Run          | `gnokey maketx run`    | Run a script against on-chain code |

#### Query types

`gnokey query <path> -remote "https://rpc.staging.gno.land:443"`

| Query Path | Data Format | Description |
|------------|-------------|-------------|
| `vm/qrender` | `"<pkgpath>:<renderpath>"` | Call `Render()` on a realm (read-only) |
| `vm/qeval` | `"<pkgpath>.<expression>"` | Evaluate any Gno expression (read-only) |
| `vm/qfuncs` | `"<pkgpath>"` | List exported function signatures (JSON) |
| `vm/qfile` | `"<pkgpath>"` or `"<pkgpath>/<file>"` | List files or get source code |
| `vm/qdoc` | `"<pkgpath>"` | Package documentation (JSON) |
| `vm/qpaths` | `"<prefix>"` | List package paths matching prefix |
| `vm/qstorage` | `"<pkgpath>"` | Storage usage and deposit for a realm |
| `auth/accounts/<addr>` | | Account info (coins, pubkey, sequence) |
| `auth/gasprice` | | Current minimum gas price |
| `bank/balances/<addr>` | | Coin balances for an address |

**`gnoland`** is the blockchain node (consensus, blocks, persistence).
`gnoland start` runs a full node. You don't run it directly in development,
`gnodev` runs it for you.

**`gnoweb`** is the web interface that renders realm markdown and lets you browse
on-chain source code. Also bundled inside `gnodev`.

---

## Core Concepts

### Realms (`r/`)

Stateful smart contracts with a unique path (e.g. `gno.land/r/demo/counter`).
They can expose `Render(path string) string` which returns **markdown**,
rendered as HTML by gnoweb. This is how dApps show their UI without a separate
frontend.

```go
func Render(path string) string {
    return "# Hello World\n\nThis is **markdown** rendered on-chain."
}
```

Public realms are immutable once deployed. This is a deliberate design choice:
immutability means users can trust the code they interact with will not change
under them. Upgrade paths exist: versioned paths (`v1`, `v2`), proxy patterns,
and private realms (which can be redeployed by their owner).

### Packages (`p/`)

Stateless, immutable libraries (e.g. `gno.land/p/demo/avl`). Cannot hold state.
Since they are immutable on-chain, two mutable realms can trustlessly cooperate
through shared p-package code.

### Gas & Storage

- **Gas fees**: priced in `ugnot` (1 GNOT = 1,000,000 ugnot)
- **Storage deposits**: GNOT locked proportional to storage used; refundable on cleanup

> Details: [Gas Fees](https://docs.gno.land/resources/gas-fees) |
> [Storage Deposit](https://docs.gno.land/resources/storage-deposit)

### Learn by Example

The **[r/docs](https://staging.gno.land/r/docs/home)** realm is an on-chain index
of interactive tutorials: Hello World, MiniSocial, AVL Pager, Solidity patterns,
and more. Each example is a working realm you can read, deploy, and modify.
More examples are being added in [PR #5016](https://github.com/gnolang/gno/pull/5016).

### Good Practices

On-chain code is **permanent and public**. These rules save you from costly mistakes:

- **Every state change costs gas.** Minimize writes. Batch when possible.
- **All code is visible on-chain.** Never put secrets, private keys, or passwords in your code.
- **Transactions are irreversible.** Test thoroughly with `gno test` and `gnodev` before deploying.
- **Storage costs real money.** Clean up state you no longer need to reclaim your deposit.
- **Panics roll back the entire transaction.** Use them intentionally for access control, not for error reporting.
- **Validate all inputs.** Anyone can call your public functions with any arguments.
- **Use `ownable` for access control.** Do not roll your own admin checks.
- **Prefer `avl.Tree` over maps.** Gno maps are deterministic (insertion-order, unlike Go), but this is an implementation detail. Maps load all data at once; AVL trees lazy-load nodes, iterate in sorted key order, and support pagination via [`pager`](https://github.com/gnolang/gno/tree/master/examples/gno.land/p/nt/pager).
- **Keep `Render()` cheap.** It runs on every page view. Avoid heavy computation in it.
- **Import paths are permanent.** Once deployed, a realm's path cannot change. Choose carefully.

For deeper guidance (global variables, panic as control flow, `init()` as constructor,
package organization, safe objects, coins vs GRC20), read
**[Effective Gno](https://docs.gno.land/resources/effective-gno)**.

---

## Networks

| Network | Chain ID | URL | RPC | Purpose |
|---------|----------|-----|-----|---------|
| Betanet | `gnoland1` | https://gno.land | `https://rpc.gno.land:443` | Main network |
| Staging | `staging` | https://staging.gno.land | `https://rpc.staging.gno.land:443` | Rolling testnet, rebuilds from latest `master` |
| Test12 | `test12` | https://test12.testnets.gno.land | `https://rpc.test12.testnets.gno.land:443` | Stable testnet, persisted state |
| Local | `dev` | http://localhost:8888 | `http://127.0.0.1:26657` | Local dev via `gnodev` |

**Faucet:** [faucet.gno.land](https://faucet.gno.land) (select your network)

> Full network details: [Gno Networks](https://docs.gno.land/resources/gnoland-networks)

---

## Quickstart on Staging

All commands target staging. Copy-paste and run.

**1. Create a key**

```bash
gnokey add MyKey
# Save the mnemonic. Note your g1... address.
```

**2. Get test tokens** at [faucet.gno.land](https://faucet.gno.land) (select staging).

```bash
gnokey query bank/balances/<your-address> -remote "https://rpc.staging.gno.land:443"
```

**3. Write a realm**

```bash
mkdir myapp && cd myapp
gno mod init gno.land/r/myname/myapp
```

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

**4. Test locally**

```bash
gnodev    # visit http://localhost:8888/r/myname/myapp
```

**5. Deploy**

```bash
gnokey maketx addpkg \
  -pkgpath "gno.land/r/myname/myapp" -pkgdir "." \
  -gas-fee 10000000ugnot -gas-wanted 8000000 \
  -broadcast -chainid staging -remote "https://rpc.staging.gno.land:443" \
  MyKey
```

View at: `https://staging.gno.land/r/myname/myapp`

**6. Call a function**

```bash
gnokey maketx call \
  -pkgpath "gno.land/r/myname/myapp" -func "Increment" \
  -gas-fee 1000000ugnot -gas-wanted 2000000 \
  -broadcast -chainid staging -remote "https://rpc.staging.gno.land:443" \
  MyKey
```

**7. Query**

```bash
gnokey query vm/qrender \
  -data "gno.land/r/myname/myapp:" -remote "https://rpc.staging.gno.land:443"
```

**8. Send tokens**

```bash
gnokey maketx send \
  -to "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5" -send "1000000ugnot" \
  -gas-fee 1000000ugnot -gas-wanted 2000000 \
  -broadcast -chainid staging -remote "https://rpc.staging.gno.land:443" \
  MyKey
```

---

## Standard Library & Notable Packages

### Gno standard library (`chain/*`)

The `std` package has been refactored into `chain`, `chain/runtime`, and
`chain/banker`. Import these directly in your code.

| Function/Type | Package | Description |
|---------------|---------|-------------|
| `address` | built-in | Gno address type (bech32 `g1...`) |
| `runtime.CurrentRealm()` | `chain/runtime` | Current realm (address + pkg path) |
| `runtime.PreviousRealm()` | `chain/runtime` | Calling realm (caller auth) |
| `runtime.AssertOriginCall()` | `chain/runtime` | Assert direct user call (not cross-realm) |
| `runtime.ChainID()` | `chain/runtime` | Current chain ID |
| `runtime.ChainHeight()` | `chain/runtime` | Current block height |
| `runtime.OriginCaller()` | `chain/runtime` | Original transaction sender |
| `chain.Emit()` | `chain` | Emit events |
| `chain.Coin`, `chain.Coins` | `chain` | Coin types |
| `chain.CoinDenom()` | `chain` | Coin denomination for a realm |
| `banker.NewBanker()` | `chain/banker` | Create banker for coin operations |
| `banker.OriginSend()` | `chain/banker` | Coins sent with the transaction |

### Notable packages

| Package | Path | Description |
|---------|------|-------------|
| **avl** | `gno.land/p/nt/avl/v0` | AVL tree, the fundamental ordered key-value store. Preferred over maps for on-chain state. |
| **ufmt** | `gno.land/p/nt/ufmt/v0` | Simplified `fmt` for on-chain use (since `fmt` is test-only) |
| **ownable** | `gno.land/p/nt/ownable/v0` | Ownership pattern for access control |
| **pausable** | `gno.land/p/nt/pausable/v0` | Pause/unpause functionality |
| **mux** | `gno.land/p/nt/mux/v0` | Router for `Render()` paths (like `http.ServeMux`) |
| **seqid** | `gno.land/p/nt/seqid/v0` | Sequential IDs, ordered correctly in AVL trees |
| **pager** | `gno.land/p/nt/pager` | Pagination helper for AVL tree iteration |
| **addrset** | `gno.land/p/moul/addrset` | Unique address collection with membership checking |
| **mgroup** | `gno.land/p/n2p5/mgroup` | [Managed group](https://github.com/gnolang/gno/tree/master/examples/gno.land/p/n2p5/mgroup) with owner, backup owners, and member roles |
| **daokit** | `gno.land/p/samcrew/daokit` | [DAO framework](https://github.com/gnolang/gno/tree/master/examples/gno.land/p/samcrew/daokit) with role-based membership (basedao) |
| **uassert** | `gno.land/p/nt/uassert/v0` | Test assertions |
| **urequire** | `gno.land/p/nt/urequire/v0` | Test assertions (fail immediately) |

### Token standards

| Standard | Path | Description |
|----------|------|-------------|
| **GRC-20** | `gno.land/p/demo/tokens/grc20` | Fungible token (like ERC-20) |
| **GRC-721** | `gno.land/p/demo/tokens/grc721` | NFT (like ERC-721) |
| **GRC-1155** | `gno.land/p/demo/tokens/grc1155` | Multi-token (like ERC-1155) |

### Notable realms

| Realm | Path | Description |
|-------|------|-------------|
| **wugnot** | `gno.land/r/gnoland/wugnot` | [Wrapped GNOT](https://gno.land/r/gnoland/wugnot) as GRC-20 |
| **boards2** | `gno.land/r/gnoland/boards2/v1` | [Discussion forum](https://gno.land/r/gnoland/boards2/v1) with moderation |
| **grc20reg** | `gno.land/r/demo/defi/grc20reg` | [Central registry](https://gno.land/r/demo/defi/grc20reg) of GRC-20 tokens |
| **gov/dao** | `gno.land/r/gov/dao` | [Governance DAO](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/gov/dao) proxy with proposals, voting, and executor system (see [Governance section](#governance-govdao)) |
| **gov/dao memberstore** | `gno.land/r/gov/dao/v3/memberstore` | [Tiered membership store](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/gov/dao/v3/memberstore) (T1/T2/T3) for GovDAO |
| **sys/validators** | `gno.land/r/sys/validators/v2` | [Validator set management](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/sys/validators/v2) via GovDAO proposals (Proof of Contribution) |
| **sys/users** | `gno.land/r/sys/users` | [User registration](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/sys/users) and name resolution |
| **disperse** | `gno.land/r/demo/disperse` | [Batch-send](https://gno.land/r/demo/disperse) coins/tokens to multiple addresses |
| **chess** | `gno.land/r/morgan/chess` | [Full on-chain chess server](https://gno.land/r/morgan/chess) |

---

## Interrealm Programming

Gno is a **multi-user programming language**: importing and calling another
user's code uses the same syntax as importing a library. The interrealm spec
defines how these interactions work safely.

In Solidity, calling another contract implicitly shifts `msg.sender`. In Gno,
every realm-context transition is **explicit** via the `cross` keyword. The
compiler enforces it.

### How Crossing Works

A **crossing function** declares `cur realm` as its first parameter. Call it
with `cross` to switch realm-context:

- `fn(cross, ...)`: switches realm-context to the callee's realm
- `fn(cur, ...)`: calls a crossing function within the same realm without switching context

**Cross call: `bank.Debit(cross, myToken, 10)`**

```
  User (g1abc...)                /r/bob/app                    /r/alice/bank
  ──────────────                ────────────                  ──────────────
        │                            │                              │
        │  gnokey maketx call        │                              │
        │  ─────────────────────>    │                              │
        │  (cross into /r/bob/app)   │                              │
        │                            │                              │
        │                  Previous: g1abc...                       │
        │                  Current:  /r/bob/app                     │
        │                            │                              │
        │                            │  bank.Debit(cross, ...)      │
        │                            │  ──────────────────────>     │
        │                            │  (cross into /r/alice/bank)  │
        │                            │                              │
        │                            │                    Previous: /r/bob/app
        │                            │                    Current:  /r/alice/bank
        │                            │                    CAN read myToken
        │                            │                    CANNOT write myToken
        │                            │                              │
        │                            │  <──────────────────────     │
        │                            │  (Debit returns)             │
        │                            │                              │
        │                  Previous: g1abc...                       │
        │                  Current:  /r/bob/app                     │
        │                            │  (identity restored)         │
        │                            │                              │
```

**Non-crossing call: `bank.DebitLocal(myToken, 10)`**

```
  User (g1abc...)                /r/bob/app
  ──────────────                ────────────
        │                            │
        │                  Previous: g1abc...     (unchanged)
        │                  Current:  /r/bob/app   (unchanged)
        │                            │
        │                            │  bank.DebitLocal(myToken, 10)
        │                            │  (no crossing, runs in bob's context)
        │                            │  CAN write myToken (bob owns it)
        │                            │  Alice does NOT know who called
        │                            │
```

Each `cross` call shifts the identity: what was Current becomes Previous.
This is how a realm knows who is calling it.

**Crossing = identity and access control.** The callee gets its own identity
and can check who called, but cannot write objects that live elsewhere.

**Non-crossing = data manipulation.** No identity shift. The function can
write the object because it runs in the caller's context.

### Why not just do it implicitly?

Solidity does: every external call silently shifts `msg.sender`, and the callee
can call back into the caller before it finishes. This is the **reentrancy
bug**. A vault contract sends ETH to an attacker, the attacker's fallback
function calls `withdraw()` again, and drains the vault because the balance
was not yet updated. This caused the 2016 DAO hack ($60M lost).

Gno prevents this by design. Context switches are explicit (`cross`), so you
can see every identity change in the source. More importantly, a crossing
function **cannot write objects that live in another realm**. Even if a callee
calls back, it cannot mutate your state directly: it would need you to expose
a crossing function that does the mutation, and that function would see the
callee as the caller (not the original user). The attack surface disappears.

**Code:**

```go
// /r/alice/bank — defines Token and two functions
package bank

type Token struct{ Balance int }

// Crossing: has `cur realm`, must be called with `cross`.
func Debit(cur realm, t *Token, amount int) {
    caller := runtime.PreviousRealm() // who called me?
    // t.Balance -= amount            // FAILS: t lives in bob's realm
}

// Non-crossing: regular function, no `cur realm`.
func DebitLocal(t *Token, amount int) {
    t.Balance -= amount               // OK: runs in bob's context
}

// /r/bob/app — owns a Token
package app

import "gno.land/r/alice/bank"

var myToken = &bank.Token{Balance: 100} // lives in bob's realm

func Spend(cur realm) {
    bank.Debit(cross, myToken, 10)  // alice knows bob called, can't write myToken
    bank.DebitLocal(myToken, 10)    // no identity shift, can write myToken
}
```

### Key Rules

- **Public functions** for end users **must** be crossing (`func Foo(cur realm, ...)`). End users can only call crossing functions (via `gnokey maketx call`). This is because a cross-call establishes the caller's identity via `runtime.PreviousRealm()`: without it, the realm has no way to know who is calling, making access control impossible.
- **Methods** should generally be non-crossing, they work regardless of where the receiver lives
- **P packages** cannot contain crossing functions and cannot import realms
- You **cannot** directly modify another realm's objects (they are read-only from outside). To change them, you must call a function on that realm that performs the modification.
- A `panic()` crossing a realm boundary **aborts** the transaction

> Full spec: [Interrealm Spec](https://docs.gno.land/resources/gno-interrealm) |
> [Whitepaper](https://github.com/gnolang/gno/blob/master/docs/gnoland-whitepaper.tex)

---

## Governance (GovDAO)

GovDAO is the governance of gno.land. It controls validator selection,
treasury spending, and chain parameters. The chain operates as a **Proof of
Authority** system where GovDAO members select validators. The
[Constitution](https://github.com/gnolang/gno/blob/master/docs/CONSTITUTION.md)
describes a "Proof of Contribution" philosophy, but in practice contribution
evaluation is subjective, the initial membership was hand-picked, and there is
no on-chain mechanism to measure contributions objectively.

On-chain code: [`r/gov/dao`](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/gov/dao)

### Tiers

| Tier | Voting Power | Selection |
|------|-------------|-----------|
| **T1** (core) | 3 | Supermajority vote from T1 |
| **T2** | 2 | Supermajority vote from T1+T2 |
| **T3** | 1 | Invited by any T1/T2 member (costs 1 invitation point) |

T1 members get 3 invitation points, T2 get 2, T3 get 1.

Proposals can restrict which tiers are allowed to vote. For example, adding a
T1 member only allows T1 to vote; adding a T2 member allows T1+T2 to vote.

### Proposals

`r/gov/dao` is a proxy that delegates to a pluggable implementation (currently
[`r/gov/dao/v3/impl`](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/gov/dao/v3/impl)).
Each proposal carries an **executor** -- an arbitrary on-chain callback that
runs when the proposal passes.

1. A member creates a proposal with a title, description, and executor
2. Members vote YES, NO, or ABSTAIN
3. Percentages are computed against **total eligible voting power** (not just
   votes cast), so not voting has the same effect as voting NO
4. At **66.66% YES**, the proposal passes and the executor runs
5. At **66.66% NO**, the proposal is denied

The 66.66% threshold is the only one implemented on-chain. The Constitution's
Simple Majority (>1/2) and Constitutional Majority (>9/10) are not enforced.

**Proposal types** ([`prop_requests.gno`](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno)):
add/withdraw/promote member, change the supermajority threshold, treasury
payment, GRC-20 token list update, upgrade the DAO implementation.

### What GovDAO Controls

| Area | Realm |
|------|-------|
| **Validators** -- add/remove via proposals | [`r/sys/validators/v2`](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/sys/validators/v2) |
| **Chain parameters** -- gas prices, transfer locks | [`r/sys/params`](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/sys/params) |
| **Treasury** -- ugnot and GRC-20 payments | [`r/gov/dao/v3/treasury`](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/gov/dao/v3/treasury) |

Namespace control (`r/sys/names`) is **not** governed by GovDAO -- it enforces
a simple rule where addresses can only deploy to their own namespace.

> [Manifesto](https://github.com/gnolang/gno/blob/master/docs/MANIFESTO.md) |
> [Memba](https://memba.samourai.app/test12/dao/gno.land/r/gov/dao) (GovDAO UI for proposals, voting, and member management)

---

## Architecture

### Cosmos, Tendermint, CometBFT, and Tendermint2

Understanding the lineage helps make sense of the stack:

**BFT (Byzantine Fault Tolerance)** is a property of distributed systems that
can reach agreement even when up to 1/3 of participants are malicious or
unresponsive. The name comes from the
[Byzantine Generals Problem](https://en.wikipedia.org/wiki/Byzantine_fault)
(Lamport, 1982): how do generals coordinate an attack when some of them may be
traitors? In a BFT blockchain, validators vote on blocks in rounds of
propose/prevote/precommit. A block is finalized when 2/3+ of validators agree.
Once finalized, it is permanent: there are no forks, no reorganizations, no
"wait for 6 confirmations". This is called **instant finality**.

Key terms used throughout this section:

| Term | Meaning |
|------|---------|
| **Validator** | A node that participates in consensus by proposing and voting on blocks. Validators run the chain software and must stay online. |
| **Proposer** | The validator selected to propose the next block in a given round. Selection is deterministic (all validators can independently compute who proposes). |
| **Block** | A batch of transactions grouped together. Each block references the previous one by hash, forming the chain. |
| **Height** | The block number. Height 1 is the first block, height 2 is the second, etc. |
| **Round** | Within a single height, consensus may take multiple attempts. Each attempt is a round (starting at 0). If round 0 fails, round 1 tries with a new proposer. |
| **Finality** | The guarantee that a committed block will never be reverted. In BFT chains this is instant: once committed, it is final. Proof-of-Work chains (like Bitcoin) have probabilistic finality: you wait for more blocks to be confident. |
| **Halt** | When the chain stops producing blocks. Common causes: fewer than 2/3 of validators are online, or a determinism bug causes validators to compute different state for the same block (consensus cannot be reached). The chain resumes once the issue is resolved. No committed data is lost. |
| **Supermajority** | 2/3+ of total validator voting power. Required for prevote and precommit steps. |
| **Voting power** | A validator's weight in consensus. In Proof-of-Stake, this is proportional to staked tokens. A validator with more stake has more voting power. |
| **Liveness** | The property that the chain will eventually make progress (produce new blocks) as long as enough honest validators are online. Timers in each consensus state guarantee liveness. |
| **Safety** | The property that validators will never commit conflicting blocks at the same height. BFT guarantees safety as long as fewer than 1/3 of validators are Byzantine. |
| **Slashing** | Penalizing a validator (by taking part of their stake) for misbehavior such as double-signing (voting for two different blocks at the same height). |

**Tendermint** (2014, Jae Kwon) was the first practical BFT consensus engine for
blockchains. It separates consensus from application logic via ABCI (Application
Blockchain Interface): the consensus engine proposes and orders transactions,
the application decides what they mean. This is the core idea behind all Cosmos
chains.

**Cosmos SDK** is a framework for building ABCI applications on top of
Tendermint. Chains like the Cosmos Hub, Osmosis, and dYdX are all Cosmos SDK
apps. They share the same consensus model but run different application logic.

**CometBFT** is the current maintained fork of Tendermint, used by most Cosmos
chains today. It added features (ABCI++, gRPC, Protobuf) but also grew in
complexity and dependencies.

**Tendermint2 (TM2)** is Jae Kwon's minimalist fork of the original Tendermint,
built specifically for gno.land. It strips out gRPC, Protobuf, viper/cobra,
and Prometheus. The philosophy: the code is the spec, minimal dependencies, all
dependencies audited and vendored. TM2 was forked without git history, so the
repository starts with a single large initial commit. TM2 currently lives in
the gno monorepo at [`tm2/`](https://github.com/gnolang/gno/tree/master/tm2)
and will become an
[independent project](https://github.com/gnolang/gno/blob/master/tm2/README.md#tendermint2) after mainnet.

Because TM2 forked early and stripped aggressively, CometBFT has features that
TM2 does not: ABCI++ (vote extensions, `FinalizeBlock`), gRPC queries, Protobuf
encoding, and built-in Prometheus metrics. TM2 trades those for simplicity and
auditability. There is also no validator penalty system: misbehaving validators
(double-signing, extended downtime) are not automatically slashed. Accountability
and slashing are not yet implemented
(`tm2/pkg/bft/consensus/state.go:1411,1501`).

**libtm** is a standalone consensus engine library (by Milos Zivkovic) that
implements Algorithm 1 from the
[Tendermint consensus whitepaper](https://arxiv.org/pdf/1807.04938.pdf). It is
minimal by design: no validator set management, no networking, no signature
logic. These are left to the calling context. libtm lives in the gno monorepo
at [`tm2/pkg/libtm`](https://github.com/gnolang/gno/tree/master/tm2/pkg/libtm)
but is **not integrated** into TM2's consensus: it has its own separate
`go.mod` (`github.com/gnolang/libtm`), no code outside of libtm imports it,
and the existing TM2 consensus engine
(`tm2/pkg/bft/consensus/state.go`) is a completely separate implementation.
libtm is an experimental library and is not planned to be merged into TM2.

In short: Cosmos SDK apps use CometBFT. Gno.land uses Tendermint2. Both
implement the same BFT consensus algorithm with instant finality, but TM2 is
deliberately simpler and lacks some features CometBFT has (including slashing).

### How the Layers Fit Together

```
+-----------------------------------------------------+
|                    gno.land                          |
|  (Blockchain node, ABCI app, SDK, CLI tools)         |
+-----------------------------------------------------+
|                     GnoVM                            |
|  (Interpreter, parser, type-checker, preprocessor)   |
+-----------------------------------------------------+
|                  Tendermint2                          |
|  (Consensus engine, P2P, mempool, RPC)               |
+-----------------------------------------------------+
```

**ABCI** separates consensus from app logic. Tendermint2 handles consensus and
block production; gno.land is the ABCI app that routes transactions to GnoVM
for execution.

```
User tx via RPC --> Tendermint2 --> ABCI CheckTx (validate)
                                --> ABCI DeliverTx (execute via GnoVM)
                                --> ABCI Commit (persist state)
```

**GnoVM** interprets `.gno` source through: Parsing, Type-checking,
Preprocessing, Execution.

After each transaction, all realm state (global variables, objects, AVL tree
nodes) is serialized and hashed into a Merkle tree (IAVL database). Validators
compare the root hash at commit time to verify they all computed the same state.

> Architecture details: [GnoVM Architecture](gnovm-architecture.md) |
> [Tendermint consensus paper](https://arxiv.org/pdf/1807.04938.pdf) |
> [Reaching Consensus](https://gno.land/r/gnoland/blog:p/reaching-consensus) (practical guide by Milos Zivkovic) |
> [libtm](https://github.com/gnolang/gno/tree/master/tm2/pkg/libtm) |
> [TM2 README](https://github.com/gnolang/gno/tree/master/tm2)

---

## Tokenomics

Three tokens in the broader ecosystem
([Constitution](https://github.com/gnolang/gno/blob/master/docs/CONSTITUTION.md)):

| Token | Role |
|-------|------|
| **$ATONE** | Staking/governance on Atom.One (secures the network) |
| **$PHOTON** | Gas fee token on Atom.One (pays for compute) |
| **$GNOT** | Storage deposit token on gno.land (pays for on-chain bytes) |

### $GNOT

- **Genesis supply:** 1 billion | **Max:** ~1.333 billion (hard cap)
- **On-chain denomination:** `ugnot` (1 GNOT = 1,000,000 ugnot)
- **1 billion $GNOT = 10TB** of persistent state space
- Storage deposit: lock $GNOT proportional to storage used, refund on cleanup
- Inflation: 3.333%/year decaying 10% annually ([`33.33 * 0.9^Y` M/year](https://github.com/gnolang/gno/blob/master/docs/CONSTITUTION.md#gnot-deflationary-inflation))

**[Genesis allocation](https://github.com/gnolang/gno/blob/master/docs/CONSTITUTION.md#genesis-allocation):** Airdrop1 35% | Airdrop2 23.1% | NT,LLC 23% | Ecosystem contributors 11.9% | Investors 7%

### How They Relate

Gno.land launches independently, then migrates to **Atom.One ICS** (Inter-Chain
Security). After migration: $ATONE secures, $PHOTON pays for compute, $GNOT
pays for storage.

> On testnets, `ugnot` is used for both gas and storage.

---

## Go vs Gno Compatibility

Gno follows the Go 1.17 language specification (no generics, no goroutines/channels).
The GnoVM itself is built with modern Go.

**Not supported:**

| Feature | Status |
|---------|--------|
| Goroutines (`go`) | Not available (planned post-launch) |
| Channels (`chan`, `select`) | Not available (planned post-launch) |
| Generics | Not implemented |
| `unsafe`, `cgo` | Will never be supported |
| `net/*`, `os/*`, `syscall/*` | Non-deterministic; unavailable |
| `complex64`, `complex128` | Not implemented |
| `reflect` | Not yet (planned) |

**Key behavioral differences:**

- `time.Now()` returns block time, not system time
- `fmt` only works in tests. Use `gno.land/p/nt/ufmt/v0` on-chain
- `sort.Slice` is missing. Use `sort.Interface` + `sort.Sort`
- `crypto/sha256` only has `Sum256`; `crypto/ed25519` only has `Verify`
- `init()` runs once per realm lifetime (constructor), not per program start
- `panic()` is used as control flow: it halts and rolls back the transaction
- Global variables are encouraged in realms (the GnoVM auto-persists them)
- Maps are deterministic (unlike Go's randomized iteration), but the iteration order is an implementation detail; prefer `avl.Tree` for lazy-loaded persistent storage
- String to `[]byte` conversion: `cap == len` always (determinism)
- Import paths use `gno.land/p/...` or `gno.land/r/...`, never `github.com/...`
- Module config uses `gnomod.toml` (not `go.mod`)

**Gno-only additions:** `bigint`, `bigdec`, `cross` keyword, `realm` type, `address` type.

> Full reference: [Go-Gno Compatibility](https://docs.gno.land/resources/go-gno-compatibility) |
> [Effective Gno](https://docs.gno.land/resources/effective-gno)

---

## Known Limitations & Technical Debt

Gno.land is pre-mainnet software. These are the most important gaps to be
aware of as a builder, organized by area. Many have open PRs.

The hardest, longest-term gaps are: the gas model needs a full redesign (costs
are placeholders, fees are not refunded, preprocessing is free); the evidence
pool does not exist (the `EvidenceReactor` was removed and evidence is planned
to be reimplemented as a
[mempool lane](https://github.com/gnolang/gno/blob/master/tm2/README.md#what-is-already-proposed-for-tendermint2),
but this is not built yet, so misbehavior is never recorded or penalized);
escaped object hashing is stubbed out
([`realm.go:845-857`](https://github.com/gnolang/gno/blob/master/gnovm/pkg/gnolang/realm.go#L845),
blocks Merkle proofs for cross-realm objects); and cyclic references for
persistence are not supported (planned for
[phase 2](https://github.com/gnolang/gno/blob/master/gnovm/pkg/gnolang/ownership.go#L35)).

**Consensus (TM2).** No slashing, no misbehavior detection, no evidence pool.
Mempool lost on restart. No max gas price cap. See Architecture section above.

**Gas and fees.** Placeholder CPU costs, preprocessing is free
([#4820](https://github.com/gnolang/gno/issues/4820)), users pay full
`gas-fee` regardless of consumption
([#3805](https://github.com/gnolang/gno/issues/3805)), fees burned not
distributed. `MsgRun` has no gas accounting. Active PRs fixing costs per
operation.

**GnoVM.** `.grealm` methods are stubs. `DeepFill` unimplemented for complex
types. No cyclic GC. Mem packages re-preprocessed on reboot.

**Language.** No goroutines, channels, generics
([#5063](https://github.com/gnolang/gno/issues/5063)), or `reflect`. ~40+
stdlib packages missing. No IBC
([#4907](https://github.com/gnolang/gno/issues/4907)).

**Tooling.** `gno lint` directory-only. `gnodev` drops failed txs on reload.
`gno mod tidy` doesn't walk parent dirs. `gnoclient` has no event querying.

---

## Additional Tools

| Tool | Description |
|------|-------------|
| [gnofaucet](https://github.com/gnolang/gno/tree/master/contribs/gnofaucet) | Local faucet server for distributing test tokens |
| [tx-indexer](https://github.com/gnolang/tx-indexer) | Transaction indexer for querying chain data (blocks, txs, events) |
| [gnokms](https://github.com/gnolang/gno/tree/master/contribs/gnokms) | Key Management System for validator nodes (HSM/cloud KMS) |
| [gnobro](https://github.com/gnolang/gno/tree/master/contribs/gnobro) | Terminal UI for browsing realms (alternative to gnoweb) |
| [gnogenesis](https://github.com/gnolang/gno/tree/master/contribs/gnogenesis) | CLI for managing `genesis.json` (validators, balances, txs) |
| [gnokeykc](https://github.com/gnolang/gno/tree/master/contribs/gnokeykc) | OS keychain integration for password-less signing |
| [gnohealth](https://github.com/gnolang/gno/tree/master/contribs/gnohealth) | Health check tool for Gno nodes |
| [gnomd](https://github.com/gnolang/gno/tree/master/contribs/gnomd) | Render Gno/Markdown files as formatted text in the terminal |
| [gnomigrate](https://github.com/gnolang/gno/tree/master/contribs/gnomigrate) | Migrate legacy Gno data formats to current standards |
| [tx-archive](https://github.com/gnolang/gno/tree/master/contribs/tx-archive) | Backup and restore Tendermint2 transaction data |
| [github-bot](https://github.com/gnolang/gno/tree/master/contribs/github-bot) | GitHub PR automation (reviewers, labels, merge requirements) |

---

## Development Environment

### Editor / IDE support

| Editor | Plugin | Notes |
|--------|--------|-------|
| VS Code | [Gno for VS Code](https://marketplace.visualstudio.com/items?itemName=Gnoverse.gnolang) | IntelliSense, code navigation, snippets, testing |
| NeoVim | [gno.nvim](https://github.com/x1unix/gno.nvim) | LSP integration |
| Any editor | [gnopls](https://github.com/gnoverse/gnopls) | Gno Language Server Protocol |
| Any editor | `gofmt` / `gno fmt` | Works directly on `.gno` files |

### Client libraries / SDKs

| Library | Language | Description |
|---------|----------|-------------|
| [gnoclient](https://github.com/gnolang/gno/tree/master/gno.land/pkg/gnoclient) | Go | Official Go client for interacting with gno.land nodes |
| [gno-js-client](https://github.com/gnolang/gno-js-client) | JS/TS | JavaScript/TypeScript client for gno.land |
| [tm2-js-client](https://github.com/gnolang/tm2-js-client) | JS/TS | Lower-level Tendermint2 JavaScript client |
| [Gno Native Kit](https://github.com/gnolang/gnonative) | Mobile | Framework for native mobile/desktop dApps |
| [Adena](https://adena.app) | Browser | Non-custodial wallet with browser extension (by Onbloc) |

---

## Ecosystem & Community

### Block explorers & tools

| Tool | URL | Description |
|------|-----|-------------|
| gnoweb | [gno.land](https://gno.land) | Browse realms, view source, render markdown |
| Gnoscan | [gnoscan.io](https://gnoscan.io) | Block explorer (addresses, txs, contracts) by Onbloc |
| Playground | [play.gno.land](https://play.gno.land) | Web IDE: write, test, deploy, share Gno code |
| Gno Studio Connect | [gno.studio/connect](https://gno.studio/connect) | Web UI for exploring and calling realm functions |
| Memba | [memba.samourai.app](https://memba.samourai.app) | Multisig wallet & DAO governance app: multisig management, GovDAO proposal/voting UI, validator dashboard, contributor analytics, token launchpad (by [Samourai Coop](https://github.com/samouraiworld/memba)) |

### Community apps & projects

| Project | Description |
|---------|-------------|
| [Gnoswap](https://gnoswap.io) | AMM DEX protocol (by Onbloc) |
| [Flippando](https://gno.flippando.xyz/flip) | On-chain memory game with NFT minting |
| [Hall of Realms](https://gno.land/r/leon/hor) | Exhibition of community-built realms |
| [boards2](https://gno.land/r/gnoland/boards2/v1) | On-chain discussion forum with moderation |
| [chess](https://gno.land/r/morgan/chess) | Full on-chain chess server with game lobby |
| [disperse](https://gno.land/r/demo/disperse) | Batch-send to multiple addresses (like disperse.app) |

---

## Further Reading

- [docs.gno.land](https://docs.gno.land): official docs
- [Effective Gno](https://docs.gno.land/resources/effective-gno): best practices, patterns, and design guidance
- [r/docs](https://staging.gno.land/r/docs/home): on-chain interactive tutorials and examples
- [Interact with gnokey](https://docs.gno.land/users/interact-with-gnokey): gnokey usage guide (improved in [PR #5030](https://github.com/gnolang/gno/pull/5030))
- [Interrealm Spec](https://docs.gno.land/resources/gno-interrealm): full cross-realm spec
- [Gas Fees](https://docs.gno.land/resources/gas-fees): gas pricing
- [Storage Deposit](https://docs.gno.land/resources/storage-deposit): how storage deposits work
- [Data Structures](https://docs.gno.land/resources/gno-data-structures): maps, AVL trees, and determinism
- [Go-Gno Compatibility](https://docs.gno.land/resources/go-gno-compatibility): what Go features work
- [Gno Packages](https://docs.gno.land/resources/gno-packages): package paths, namespaces, and deployment
- [GnoVM Architecture](gnovm-architecture.md): VM internals
- [Constitution](https://github.com/gnolang/gno/blob/master/docs/CONSTITUTION.md): governance, tokenomics, GovDAO tiers
- [Manifesto](https://github.com/gnolang/gno/blob/master/docs/MANIFESTO.md): philosophy and values
- [Whitepaper](https://github.com/gnolang/gno/blob/master/docs/gnoland-whitepaper.tex): technical whitepaper
- [Tendermint consensus paper](https://arxiv.org/pdf/1807.04938.pdf): BFT algorithm specification
- [Getting Started](https://github.com/gnolang/getting-started): starter template
- [awesome-gno](https://github.com/gnoverse/awesome-gno): curated list of Gno resources

**Community:** [Discord](https://discord.gg/S8nKUqwkPn) |
[GitHub](https://github.com/gnolang) |
[X/Twitter](https://twitter.com/_gnoland)

---

*Maintained as part of [gno-skills](https://github.com/samouraiworld/gno-skills).
See also [awesome-gno](https://github.com/gnoverse/awesome-gno).*
