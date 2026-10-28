# docs: point the reference pages off staging
Written by claude-opus-5-5.
PR: [gnolang/gno#6412](https://github.com/gnolang/gno/pull/6412)

## TLDR

The staging chain stopped making blocks in July, but many doc pages still told readers to send their commands there, so those examples failed. This change points them at the public testnet instead, or at a local chain where the testnet cannot run them. It also fixes a few dead links and a small build-script bug.

## The staging chain

Staging was a rolling chain that redeployed `master` and replayed its history, run from `misc/loop`. It has produced no block since 2026-07-10, and `https://rpc.staging.gno.land:443` answers `503`. It is not retired, since issue gnolang/gno#6258 plans to rebuild it on `gnodev`, so [`docs/resources/gnoland-networks.md`](https://github.com/gnolang/gno/pull/6412/files#diff-0aab8618f1b0544f08d36a3d69a3dca9feb5a4ad931f43ad5b248b08754888ffR81-R88) keeps its row, marks it down and adds a warning. Every other page stops sending readers there.

## The `latest` testnet endpoint and `<chain-id>`

`https://rpc.latest.testnets.gno.land:443` always points at the current testnet, today Onyx with chain ID `onyx-1` ([`docs/resources/gnoland-networks.md`](https://github.com/gnolang/gno/pull/6412/files#diff-0aab8618f1b0544f08d36a3d69a3dca9feb5a4ad931f43ad5b248b08754888ffR13-R14)). The examples use that endpoint so they survive the next testnet. The chain ID changes with each testnet, so the commands write `-chainid <chain-id>` and send the reader to the networks page for the value, rather than hard-coding one that goes stale.

## Package activation on the testnet

Onyx runs mainnet's inert code-submission policy: an `addpkg` lands, then waits until the gpao oracle approves and enables it. The deploy walkthrough now says so, and names `https://gpao.latest.testnets.gno.land/status/<pkgpath>` as the place to see why a package was refused ([`docs/users/interact-with-gnokey.md`](https://github.com/gnolang/gno/pull/6412/files#diff-a584d6c1436860045c8238fa0be982e0aeee56e3701d1580aa448bdc54064b9dR249-R253)). It also says the namespace can be the reader's own `g1…` address, which needs no registration.

## Why the `Run` example goes local

On Onyx, `MsgRun` is allowed only for the accounts listed in the `run_submitters` parameter, which is one GovDAO member, and the `r/demo/counter` realm it calls is not deployed there. So that one example targets a local `gnodev` chain, `-chainid dev -remote http://localhost:26657` ([`docs/users/interact-with-gnokey.md`](https://github.com/gnolang/gno/pull/6412/files#diff-a584d6c1436860045c8238fa0be982e0aeee56e3701d1580aa448bdc54064b9dR416-R419)). The multisig walkthrough also runs on a local chain, which is gnokey's default, and lists the two flags that move it to the testnet.

## Which examples call what

The examples that do run on the testnet call `wugnot.Deposit` with `-send 1000ugnot`, which is deployed there, in place of `r/demo/counter` and `boards2.CreateBoard`, which ordinary users cannot call on Onyx. Their fees are `-gas-wanted` divided by 1000, the testnet's floor price of 1ugnot per 1000 gas. The sample receipts are labelled as taken on a local chain, since the height and gas used differ on the testnet.

## `gnokey verify` and the account values

A signature covers the chain ID, account number and sequence. When any of them is left out, `gnokey verify` reads the account's current values from `-remote`. After a broadcast the sequence has moved on, so a verify without them fails on a valid signature. The verify examples now pass all three ([`docs/users/interact-with-gnokey.md`](https://github.com/gnolang/gno/pull/6412/files#diff-a584d6c1436860045c8238fa0be982e0aeee56e3701d1580aa448bdc54064b9dR707-R710)).

## Dead anchors and the Makefile

Five links pointed at headings that no longer exist: `effective-gno.md#choose-storage-types-by-access-pattern` and `gno-security-guide.md#510-…`. They now point at `#prefer-avltree-over-map-for-scalable-storage` and at item 10 of `gno-ai-contract-review.md`. In [`docs/Makefile`](https://github.com/gnolang/gno/pull/6412/files#diff-cdec797dddf8fc387304501d6bd4f318e1e9067c6fae012dfa8b69fd85fd3248R1-R9), `$(PWD)` is the caller's shell directory, so `make -C docs lint` from the repository root linted the wrong tree; `$(CURDIR)` is the directory make runs in.

## Before and after

**Before:** a reader copying the `gnokey` examples sent transactions to `rpc.staging.gno.land` with `-chainid staging`, and every one failed against a dead endpoint.

**After:** the same examples reach the current testnet with the chain ID from the networks page, and the two that the testnet cannot run say so and use a local chain.

## What a user notices

- The `gnokey`, glossary, gnoweb and networks pages name the testnet or mainnet wherever they used to name staging.
- Links to example realms that used to open on staging now open their source on GitHub.

## What it leaves open

- The networks page carries a dated "down since" marker that someone has to remove when staging comes back; issue gnolang/gno#6258 tracks the rebuild.
- The four new GitHub links point at `master`, so a reader of an older release's docs sees master's code.

## Words used here

| Name | What it is |
| --- | --- |
| `rpc.latest.testnets.gno.land` | RPC endpoint that always serves the current testnet; today `onyx-1`. |
| `<chain-id>` | Placeholder for the current testnet's chain ID, listed on the networks page. |
| gpao | The oracle that approves and enables packages under the inert submission policy; its `/status/<pkgpath>` page reports a package's state. |
| `run_submitters` | VM parameter listing who may send `MsgRun`; on Onyx, one GovDAO member. |
| `gnokey verify` | Checks a signed transaction; reads chain ID, account number and sequence from `-remote` when they are not passed. |
| `$(CURDIR)` | make's variable for the directory it is running in, set by `-C`. |

<details>
<summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test, a doc or a generated file.

### 1. `hot` · The `verify` example, [`docs/users/interact-with-gnokey.md`](https://github.com/gnolang/gno/pull/6412/files#diff-a584d6c1436860045c8238fa0be982e0aeee56e3701d1580aa448bdc54064b9dR707-R710)
Passes the values the transaction was signed with, so verify does not read the moved-on sequence.
**If wrong:** verify reports a valid signature as invalid after broadcast.

### 2. `hot` · The local `Run` example, [`docs/users/interact-with-gnokey.md`](https://github.com/gnolang/gno/pull/6412/files#diff-a584d6c1436860045c8238fa0be982e0aeee56e3701d1580aa448bdc54064b9dR416-R419)
Sends the one example the testnet refuses to a local `gnodev`.
**If wrong:** readers hit a permission error on the testnet.

### 3. `cold` · The staging warning, [`docs/resources/gnoland-networks.md`](https://github.com/gnolang/gno/pull/6412/files#diff-0aab8618f1b0544f08d36a3d69a3dca9feb5a4ad931f43ad5b248b08754888ffR81-R88)
Keeps staging listed but marked down, and names the testnet endpoint the docs use.
**If wrong:** readers keep trying the dead chain.

### 4. `cold` · The lint path, [`docs/Makefile`](https://github.com/gnolang/gno/pull/6412/files#diff-cdec797dddf8fc387304501d6bd4f318e1e9067c6fae012dfa8b69fd85fd3248R1-R9)
Lints the docs directory whatever directory make was called from.
**If wrong:** `make -C docs lint` checks the caller's tree.

</details>
