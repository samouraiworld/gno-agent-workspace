# gno docs: transaction fees, sample output, and surfaces added since

Target: `docs/` in [gnolang/gno](https://github.com/gnolang/gno), base sha
6df71ae35, the [pearl testnet release candidate](https://github.com/gnolang/gno/commit/6df71ae35).
Every line reference below pins to that sha.

## TLDR

Every fee figure in the transaction docs is wrong in the same direction.
[`-gas-fee` is documented as a price per gas unit](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gas-fees.md#L33);
the chain deducts it whole, once, as
[the transaction fee](https://github.com/gnolang/gno/blob/6df71ae35/tm2/pkg/sdk/auth/ante.go#L200).
So the recommended `1000000ugnot` is one GNOT per transaction where the same
deploy is accepted at 3046ugnot. A reader who runs the quickstart deploy and
two calls as written spends 3.3 GNOT, which is where "addpkg needs 3 GNOT"
comes from. The chain charges 0.32 GNOT for those three transactions, and
0.29 GNOT of that is the storage deposit, which is refundable on delete.

Four measurements back this, each a testscript under
[`tests/`](tests) run against a local node at the base sha.

## Decisions for a human

**A1. One pull request or three.** F1 through F4 and F10 are the same drift
across three files and read as one change. F6 and F7 are new surfaces nobody
has written up, and whoever owns the rollout should say what a user is meant
to do with them. F5, F8, F9 and F11 are short corrections that can ride with
F1.

**A2. What replaces the recommended fee.** `-simulate only` already prints a
suggested gas limit and the fee for it. The docs can drop every hardcoded
`-gas-fee` and route through that one command, or keep worked numbers and
pin them to the initial gas price. The second choice goes stale the next time
governance moves the price.

**A3. Whether the inert flow is user-facing.** No genesis under
`misc/deployments/` sets a code submission policy, so every live network is
still permissionless and `enablepkg` reaches nobody today. F6 is a gap only
if a network is meant to turn it on.

### F1. `-gas-fee` is the whole fee, not a price per gas unit

Severity: high.

[The gas fee section](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gas-fees.md#L33)
reads "how much you're willing to pay per unit of gas" and gives
[`Maximum Fee = Gas Wanted × Gas Fee`](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gas-fees.md#L40).
The ante handler deducts
[the coin as it stands](https://github.com/gnolang/gno/blob/6df71ae35/tm2/pkg/sdk/auth/ante.go#L200),
and the flag's own help calls it
[the gas payment fee](https://github.com/gnolang/gno/blob/6df71ae35/tm2/pkg/crypto/keys/client/maketx.go#L87).
Gas wanted is the divisor, not a multiplier: the fee is accepted when
`gas-fee / gas-wanted` clears the block gas price.

The same page contradicts itself twenty lines down, where
[the worked example](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gas-fees.md#L81)
multiplies the rate by gas wanted to reach a 200000ugnot total. That half is
right.

Two more copies of the wrong reading:
[getting-started](https://github.com/gnolang/gno/blob/6df71ae35/docs/builders/getting-started.md#L232)
says "the price per unit, in `ugnot`", and
[the addpkg flag list](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L170)
says "amount of GNOTs to pay per gas unit".

Fix: `-gas-fee` is the total fee, charged in full whatever the transaction
uses. It must be at least `gas-wanted` divided by 1000 at
[the initial gas price](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/gnoland/genesis.go#L22).

### F2. The recommended fees overcharge by two to four orders of magnitude

Severity: high.

Measured on a local node at the base sha, deploying the counter realm the
[quickstart](https://github.com/gnolang/gno/blob/6df71ae35/docs/builders/quickstart.md#L57)
tells the reader to fetch:

| Transaction | Doc flags | Total cost | Same tx at the accepted minimum |
| --- | --- | --- | --- |
| `addpkg` counter | `-gas-fee 1000000ugnot -gas-wanted 20000000` | 1294600ugnot | 314600ugnot |
| `call Increment` | `-gas-fee 1000000ugnot -gas-wanted 2000000` | 1001000ugnot | 3000ugnot |

Both rows carry the same 294600ugnot and 1000ugnot storage deposit. The
difference is the fee alone: `tests/docs_deploy_cost.txtar` against
`tests/docs_deploy_cost_min.txtar`.

`-simulate only` on that deploy prints
`gas fee: 3046ugnot, current gas price: 1ugnot/1000gas`, so the documented
figure is 328 times what the chain asks. In
[interact-with-gnokey](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L264)
every call carries `-gas-fee 10000000ugnot`, ten GNOT, against a 2000ugnot
minimum for the gas wanted beside it.

[The typical gas values table](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gas-fees.md#L119)
repeats `1000000ugnot` for all four operations, which under F1's correct
reading is a flat one GNOT whether the reader transfers coins or deploys a
complex realm.

Fix: replace the hardcoded fees with what `-simulate only` returns, per A2.

### F3. Every sample transaction output predates storage deposits

Severity: medium.

A state-changing transaction now prints three storage lines and carries a
storage event. Real output for the quickstart deploy:

```text
OK!
GAS WANTED: 20000000
GAS USED:   3499918
HEIGHT:     3
STORAGE DELTA:  2946 bytes
STORAGE FEE:    294600ugnot
TOTAL TX COST:  1294600ugnot
EVENTS:     [{"bytes_delta":2946,"fee_delta":{"denom":"ugnot","amount":294600},"pkg_path":"gno.land/r/test1/counter"}]
INFO:
TX HASH:    ADoadOIkHNSezthTQvh8289rExE0nNZdRApV2JsCSrk=
PKGPATH:    gno.land/r/test1/counter
```

The docs show neither the storage lines nor the event, at
[getting-started deploy](https://github.com/gnolang/gno/blob/6df71ae35/docs/builders/getting-started.md#L346),
[getting-started call](https://github.com/gnolang/gno/blob/6df71ae35/docs/builders/getting-started.md#L389),
[the addpkg receipt](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L215),
[the BalanceOf call](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L320)
and [the gnodev call](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gnodev.md#L160).

The receipt walkthrough goes further and teaches the empty case as the rule:
[`EVENTS: []` ... "in this case, none"](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L225).
An `addpkg` that stores bytes always emits the storage event, so the reader
is told to expect an output they will not see.

Fix: repaste all five blocks from a run, and give `TOTAL TX COST` a line in
the walkthrough, since it is the number a reader is looking for.

### F4. The second addpkg example runs out of gas

Severity: medium.

[That command](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L201)
passes `-gas-wanted 200000` for the hello_world package the page just built.
Deploying that exact package uses 2574509 gas and fails with
`out of gas error` at the documented limit, per
`tests/docs_hello_world_cost.txtar`. The receipt under it claims
[`GAS USED: 117564`](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L213),
twenty-two times under the measurement.

The first example on the same page passes `-gas-wanted 8000000`, which works.
The two differ for no reason the text gives.

Fix: one gas figure on the page, at or above the measured 2574509, and a
repasted receipt.

### F5. The errors package footnote is stale

Severity: medium.

[Footnote 8](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/go-gno-compatibility.md#L327)
says `errors` ships `New` only and calls `Is`, `As`, `Unwrap` and `Join` "not
yet available", tracking them to PR
[#5385](https://github.com/gnolang/gno/pull/5385). That PR merged on
2026-06-12. The stdlib now has
[`Unwrap`](https://github.com/gnolang/gno/blob/6df71ae35/gnovm/stdlibs/errors/wrap.gno#L13),
[`Is`](https://github.com/gnolang/gno/blob/6df71ae35/gnovm/stdlibs/errors/wrap.gno#L41)
and [`Join`](https://github.com/gnolang/gno/blob/6df71ae35/gnovm/stdlibs/errors/join.gno#L15).

Fix: `As` is the one still missing. Keep
[#486](https://github.com/gnolang/gno/issues/486) as the tracker and drop the
merged PR.

### F6. The inert package flow is undocumented

Severity: medium, conditional on A3.

[`gnokey maketx enablepkg`](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/keyscli/enablepkg.go#L31)
and
[`rejectpkg`](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/keyscli/rejectpkg.go#L28)
arrived with
[#6088](https://github.com/gnolang/gno/pull/6088) and appear nowhere under
`docs/`. Neither does the
[code submission policy](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/sdk/vm/params.go#L30),
whose inert setting stores a package without type checking it and leaves it
uncallable until an approver enables it. Under that policy a reader who
deploys and then opens their realm finds nothing, with no page saying why.

The rest of that pull request's surface is undocumented too. `gnomod.toml`
gained [`max_deposit` under `[addpkg]`](https://github.com/gnolang/gno/blob/6df71ae35/gnovm/pkg/gnomod/file.go#L63),
where configuring-gno-projects still lists
[creator](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/configuring-gno-projects.md#L43)
and height alone. Two query routes arrived with it,
[`vm/qpkgmeta_json`](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/sdk/vm/handler.go#L122)
for a path's submit-time status and
[`vm/qinertpaths`](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/sdk/vm/handler.go#L125)
for what is awaiting approval. Neither appears in the query list.

Fix: a section in
[interact-with-gnokey](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L96)
covering the two subcommands and the two queries, the `max_deposit` field in
configuring-gno-projects, and a sentence in
[getting-started's before-you-deploy](https://github.com/gnolang/gno/blob/6df71ae35/docs/builders/getting-started.md#L305),
beside the CLA note it already carries.

### F7. Session accounts and key rotation are undocumented

Severity: low.

[`gnokey session create`](https://github.com/gnolang/gno/blob/6df71ae35/tm2/pkg/crypto/keys/client/session_create.go#L44),
`revoke` and `revokeall` landed in
[#5614](https://github.com/gnolang/gno/pull/5614) on 2026-05-12. The word
session appears zero times across `docs/users`, `docs/builders` and
`docs/resources`.
[`gnokey rotate`](https://github.com/gnolang/gno/blob/6df71ae35/tm2/pkg/crypto/keys/client/rotate.go#L24)
is likewise absent, and it changes a keybase password, which is the kind of
thing a reader looks up rather than guesses.

### F8. The default deposit ceiling is missing from the storage deposit page

Severity: low.

[The page](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/storage-deposit.md#L30)
calls `-max-deposit` optional and stops there. Omitting it does not remove a
ceiling: it takes
[the 100000000ugnot default](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/sdk/vm/params.go#L40),
one megabyte of realm state. A package materializing more is refused before
it runs, which is what
[`addpkg_preprocess_alloc_doubling.txtar`](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/integration/testdata/addpkg_preprocess_alloc_doubling.txtar#L23)
exists to pin.

### F9. The stdlib reference page is three surfaces behind

Severity: low.

[The chain package section](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gno-stdlibs.md#L334)
documents `chain` itself and its banker and runtime subpackages. Three things
added since are missing from it:

| Surface | Landed in | Where it is documented today |
| --- | --- | --- |
| [`chain/params`](https://github.com/gnolang/gno/blob/6df71ae35/gnovm/stdlibs/chain/params/params.gno#L40) | [#6094](https://github.com/gnolang/gno/pull/6094) | one row of the compatibility table |
| [`banker.IsCanonical`](https://github.com/gnolang/gno/blob/6df71ae35/gnovm/stdlibs/chain/banker/banker.gno#L208) | [#5890](https://github.com/gnolang/gno/pull/5890) | the security guide |
| `realm.Sub` | [#5890](https://github.com/gnolang/gno/pull/5890) | [gno-interrealm-v2](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gno-interrealm-v2.md#L427) |

A reader who reaches for the reference page rather than the guide finds the
[banker section](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gno-stdlibs.md#L733)
and the realm section, and neither names them.

### F11. `vm/qobject_binary` is missing from the JSON query page

Severity: low.

[The endpoint table](https://github.com/gnolang/gno/blob/6df71ae35/docs/builders/query-state-api.md#L16)
carries four routes.
[`qobject_binary`](https://github.com/gnolang/gno/blob/6df71ae35/gno.land/pkg/sdk/vm/handler.go#L112)
is the fifth, and has been since
[#5274](https://github.com/gnolang/gno/pull/5274) in April 2026.

### F10. The simulate output is one release behind

Severity: low.

[The sample](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gas-fees.md#L146)
prints `estimated gas usage: 268994, gas fee: 282ugnot`. The command now
prints a margin too:

```text
INFO:       estimated gas usage: 2900585 (suggested, with 5% margin: 3045615), gas fee: 3046ugnot, current gas price: 1ugnot/1000gas
```

[The line under it](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gas-fees.md#L149)
tells the reader to use the raw estimate as `-gas-wanted`, which is the value
the margin exists to replace, and gas usage varies between the simulation and
the broadcast: the two runs in `tests/docs_deploy_cost.txtar` and
`tests/docs_simulate_only.txtar` differ by 599333 gas on the same package.

## What did not hold up

**The network table is current.**
[gnoland-networks](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gnoland-networks.md#L5)
lists pearl with chain id `pearl-1` beside betanet and staging, matching the
release candidate at the base sha.

**The package naming rule is documented.** The lint check from
[#5048](https://github.com/gnolang/gno/pull/5048) is stated as a rule in
[gno-packages](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gno-packages.md#L90),
with the identifier constraint it implies.

**Sub-realm identities are documented.** `cur.Sub` from
[#5890](https://github.com/gnolang/gno/pull/5890) has
[its own section](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gno-interrealm-v2.md#L427).

**Example tests are documented.** The support added by
[#5188](https://github.com/gnolang/gno/pull/5188) is covered in
[gno-testing](https://github.com/gnolang/gno/blob/6df71ae35/docs/resources/gno-testing.md#L58).

**`-max-deposit` is in the addpkg flag list.** It sits
[beside `-send`](https://github.com/gnolang/gno/blob/6df71ae35/docs/users/interact-with-gnokey.md#L167),
and getting-started explains what the cap does.

## Reproducing

The four testscripts under [`tests/`](tests) go in
`gno.land/pkg/integration/testdata/` of a checkout at the base sha:

```bash
go test ./gno.land/pkg/integration -run 'TestTestdata/docs_deploy_cost' -v -timeout 20m
```

`docs_deploy_cost` runs the quickstart deploy and call at the documented
flags, `docs_deploy_cost_min` runs the same two at the accepted minimum,
`docs_hello_world_cost` runs the interact-with-gnokey package at both gas
limits, and `docs_simulate_only` prints what the estimator suggests.
