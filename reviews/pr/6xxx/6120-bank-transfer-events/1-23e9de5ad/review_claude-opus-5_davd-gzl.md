# PR [#6120](https://github.com/gnolang/gno/pull/6120): feat(bank): emit transfer events for ugnot movements

URL: https://github.com/gnolang/gno/pull/6120
Author: notJoon | Base: master | Files: 15 | +1063 -91
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 23e9de5ad (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6120 23e9de5ad`
Overview: [overview](../overview.md)

## Overview

Every ugnot move that goes through the bank keeper now appends a value to the
transaction's event list, so a service rebuilding account balances from
transaction results sees the moves it could not read off the message itself. A
realm calling `banker.SendCoins` and the `-send` envelope forwarded onward by a
realm were the two invisible ones. One emit inside the unexported `sendCoins`
covers `MsgSend`, the realm banker and every VM send envelope, because all three
funnel through it, and a second emit in `InputOutputCoins` publishes a multisend
as one batch carrying the original input and output lists rather than inventing
a sender for each recipient. The change also deletes the dead Cosmos-era event
comments the port left behind, and registers the two new types plus `Input`,
`Output` and `MsgMultiSend` with the bank amino package.

**Verdict: REQUEST CHANGES** — the emit points and their ordering are right, and
the branch also turns `bank.MsgMultiSend` into a transaction the chain accepts,
which the events do not need (2 warnings, 1 suggestion, 1 nit).

## Verify first

- [`tm2/pkg/sdk/bank/package.go:21`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/package.go#L21) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/package.go#L21) — decide whether `MsgMultiSend` should become a live transaction type in this release. Confirm with `git show origin/master:tm2/pkg/sdk/bank/package.go`, which registers six types and not this one.
- [`tm2/pkg/sdk/bank/keeper.go:187-191`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/keeper.go#L187-L191) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187-L191) — the emit sits after both the debit and the credit returned nil, so a failed transfer leaves nothing. Run `go test -run 'TestBankKeeperSendCoinsZero' ./tm2/pkg/sdk/bank/` and read the `require.Empty` on the event logger.

## Summary

An indexer that reconstructs per-address balances needs every balance change in
a transaction or an event, and two of the four ugnot paths had neither. The fix
is two `EmitEvent` calls in the keeper, at
[`keeper.go:112`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/keeper.go#L112) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L112)
and
[`keeper.go:187`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/keeper.go#L187) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187),
plus two event types and their amino registration. Placing the one-to-one emit
in the private `sendCoins` rather than in the handlers is what makes the realm
banker and the VM send envelope free, and it also picks up the two paths the
body does not list: the `MsgAddPackage` send envelope and the inert submission
charge.

## Balance-mutating call sites

Every reachable caller of a `BankKeeper` mutator at 23e9de5ad, and what the
transaction result says about it.

| Call site | What moves | Event |
| --- | --- | --- |
| [`bank/handler.go:50`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/handler.go#L50) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/handler.go#L50) | `MsgSend` | `TransferEvent` |
| [`bank/handler.go:72`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/handler.go#L72) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/handler.go#L72) | `MsgMultiSend` | `MultiTransferEvent` |
| [`vm/builtins.go:52`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/builtins.go#L52) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/builtins.go#L52) | realm `banker.SendCoins` | `TransferEvent` |
| [`vm/keeper.go:1212`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/keeper.go#L1212) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L1212) | `MsgCall` send envelope | `TransferEvent` |
| [`vm/keeper.go:1425`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/keeper.go#L1425) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L1425) | `MsgRun` send envelope | `TransferEvent` |
| [`vm/keeper.go:1047`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/keeper.go#L1047) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L1047) | `MsgAddPackage` send envelope | `TransferEvent`, unlisted in the body |
| [`vm/keeper.go:952`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/keeper.go#L952) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L952) | inert submission charge | `TransferEvent`, unlisted in the body |
| [`auth/ante.go:481`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/auth/ante.go#L481) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/auth/ante.go#L481) | gas fee to the collector | none; amount on the transaction as `gas_fee`, collector address nowhere |
| [`vm/keeper.go:2381`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/keeper.go#L2381) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L2381) | storage deposit locked | none; `StorageDepositEvent` carries the amount and `pkg_path`, not the addresses |
| [`vm/keeper.go:2396`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/keeper.go#L2396) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L2396) | storage deposit refunded | none; `StorageUnlockEvent`, same shape |
| [`vm/builtins.go:94`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/builtins.go#L94) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/builtins.go#L94) | realm `banker.IssueCoin` | none; realm-qualified denoms only, not ugnot |
| [`vm/builtins.go:103`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/builtins.go#L103) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/builtins.go#L103) | realm `banker.RemoveCoin` | none; same |
| [`gnoland/app.go:212`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/gnoland/app.go#L212) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/gnoland/app.go#L212) | genesis signer funding | none; gated on `BlockHeight() == 0` |
| [`gnoland/app.go:789`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/gnoland/app.go#L789) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/gnoland/app.go#L789) | genesis balances | none; genesis only |

The realm-denom rows cannot carry ugnot: both go through
[`assertIssuable`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/builtins.go#L85-L89) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/builtins.go#L85-L89),
which rejects any denom `std.IsRealmDenom` refuses. `SubtractCoins` and
`AddCoins` have no callers outside the keeper and
[`supply.go:168`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/supply.go#L168) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/supply.go#L168),
which is `MintCoins`.

So at runtime, three ugnot movements stay outside a transfer event: the gas fee
and the two storage deposit legs. All three are declared out of scope, and all
three are the paths where an indexer needs an address the events do not carry.

## Warnings (should fix)

- **[new transaction surface]** [`package.go:21`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/package.go#L21) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/package.go#L21) — registering `MsgMultiSend` turns a message a node refused to decode into one it accepts, and nothing about the events needs it.
  <details><summary>details</summary>

  Master's bank amino package registers six types and not this one, so
  `amino.Unmarshal` of a `std.Tx` carrying `/bank.MsgMultiSend` answers
  `unrecognized concrete type full name bank.MsgMultiSend` and the transaction
  never reaches `handleMsgMultiSend`. At this head the same bytes decode and the
  handler runs, which the branch's own txtar exercises by broadcasting a
  multisend against a live node. Deleting only the `MsgMultiSend{}` line and
  keeping everything else leaves all four bank event tests green, including the
  `MultiTransferEvent` amino round trip, so the registration is separable from
  the feature. Both runs are in
  [`tests/multisend_tx_surface_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6120-bank-transfer-events/1-23e9de5ad/tests/multisend_tx_surface_test.go).
  Fix: drop the line, or make the new transaction type the PR's declared subject
  so it is reviewed as one.
  </details>

- **[wrong wire contract]** [`pr6120_bank_transfer_events.md:37-40`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L37-L40) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L37-L40) — the shape named as indexer-facing is what a CLI result printer produces; an RPC client gets `coins` as one amino string instead.
  <details><summary>details</summary>

  `ResponseBase.EncodeEvents` has three callers and all are CLI result printers:
  [`common.go:41`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/crypto/keys/client/common.go#L41) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/crypto/keys/client/common.go#L41)
  and `:52`, the tm2 client defaults `gnokey` replaces at
  [`root.go:38-40`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/keyscli/root.go#L38-L40) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/keyscli/root.go#L38-L40),
  and
  [`root.go:91`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/keyscli/root.go#L91) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/keyscli/root.go#L91),
  the line a `gnokey` user actually sees.
  Every RPC response goes through
  [`amino.MarshalJSON`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/rpc/lib/types/types.go#L207) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/bft/rpc/lib/types/types.go#L207)
  instead, which calls `std.Coins.MarshalAmino` and renders the field as one
  string. The branch's own [`events_test.go:29`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/events_test.go#L29) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/events_test.go#L29) asserts exactly that string form.
  The PR body carries a third shape again, `"amount":[{...}]`, from an earlier
  field name. Measured pair in
  [`tests/event_wire_shapes_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6120-bank-transfer-events/1-23e9de5ad/tests/event_wire_shapes_test.go).
  Fix: state the amino form as the contract, since that is the one GnoScan
  receives.
  </details>

## Nits

- [`pr6120_bank_transfer_events.md:1`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L1) — the title reads `PRxxxx`; 13 of the 17 other PR-named ADRs under `tm2/adr/` carry their number.

## Suggestions

- **[rollout]** [`keeper.go:187`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/keeper.go#L187) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187) — the event set is inside the block header, so this cannot go out as a rolling restart.
  <details><summary>details</summary>

  `ABCIResult` carries `Events` at
  [`results.go:14-18`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/types/results.go#L14-L18) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/bft/types/results.go#L14-L18),
  the merkle root of that set becomes `LastResultsHash` at
  [`execution.go:456`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/state/execution.go#L456) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/bft/state/execution.go#L456),
  and every node rejects a block whose header disagrees with its own execution at
  [`validation.go:82-86`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/state/validation.go#L82-L86) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/bft/state/validation.go#L82-L86).
  A validator on the old binary and one on the new binary therefore disagree on
  the first block containing any ugnot transfer. The chain has the machinery for
  this in the halt-height and minimum-version params read by
  [`checkNodeStartupParams`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/gnoland/node_params.go#L133-L137) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/gnoland/node_params.go#L133-L137);
  the ADR's Consequences section does not name it. Fix: add the coordinated
  upgrade there.
  </details>

## Verified

- Determinism holds: both emits append in execution order to a slice, no map is
  iterated, and `MultiTransferEvent` keeps the message's own input and output
  order. `go test -run 'TestInputOutputCoinsEmitsTransferEvents' ./tm2/pkg/sdk/bank/`
  asserts that order.
- Events do not survive a failed message. One logger is created per transaction
  at [`baseapp.go:687`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/baseapp.go#L687) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/baseapp.go#L687)
  and copied into the result only under `if err == nil` at
  [`baseapp.go:737-739`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/baseapp.go#L737-L739) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/baseapp.go#L737-L739).
- No context reaching the bank can nil-dereference the new call. `NewContext`
  installs a logger at
  [`context.go:81`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/context.go#L81) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/context.go#L81),
  and the one zero-value `sdk.Context` in the tree, at
  [`node_params.go:137`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/gnoland/node_params.go#L137) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/gnoland/node_params.go#L137),
  reads params and holds no bank keeper.
- The unmetered emit adds no cheap way to inflate a block's results. `chain.emit`
  is already the cheaper path per event byte: it is billed base 362 plus a slope
  over the attribute count at
  [`native_gas.go:150`](https://github.com/gnolang/gno/blob/23e9de5ad/gnovm/stdlibs/native_gas.go#L150) · [↗](../../../../../.worktrees/gno-review-6120/gnovm/stdlibs/native_gas.go#L150)
  while carrying up to 64 pairs of 4096 bytes each, whereas a bank event exists
  only where the metered balance writes that produce it ran. The txtar measured
  932917 gas for one `MsgSend` transaction and 2047951 for the `MsgCall` carrying
  two transfers. This is an argument from the gas table and those two numbers,
  not a benchmark of the cheapest attack.
- Green at 23e9de5ad: `go test -run 'TestTransferEventAminoRoundTrip|TestHandlerEmitsTransferEvents|TestSendCoinsEmitsTransferEvent|TestInputOutputCoinsEmitsTransferEvents|TestBankKeeperSendCoinsZero' ./tm2/pkg/sdk/bank/`,
  `go test -run 'TestDeliverTxIncludesMsgRunSendEvent' ./gno.land/pkg/gnoland/`,
  `go test -run 'TestTestdata/bank_transfer_events' ./gno.land/pkg/integration/`.

## Existing threads

| Reviewer | Gist | State | Overlap |
| --- | --- | --- | --- |
| jinoosss | [`events.go:12`](https://github.com/gnolang/gno/pull/6120#discussion_r3922317503): name the field for what it holds | answered by the rename to `Coins` | none |
| jinoosss | [`keeper.go:128`](https://github.com/gnolang/gno/pull/6120#discussion_r3922334625): the split from-only and to-only multisend events read as burn then mint | answered by `MultiTransferEvent`, and the author asked for confirmation | none |
| junghoon-vans | [the `/docs/.preview/` hunk is unrelated](https://github.com/gnolang/gno/pull/6120#issuecomment-5522036917) | open | takes the `.gitignore` item below |

None of the four findings here appears in a thread.

## Not posted

- [`events.go:9-10`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/events.go#L9-L10) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/events.go#L9-L10) — `TransferEvent.From` and `.To` are `string`
  where `Input.Address` and `Output.Address` in the same package are
  `crypto.Address`, which validates bech32 on decode. The wire shape is the same
  either way, since `crypto.Address` carries both
  [`MarshalAmino`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/crypto/crypto.go#L85-L87) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/crypto/crypto.go#L85-L87)
  and
  [`MarshalJSON`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/crypto/crypto.go#L72-L75) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/crypto/crypto.go#L72-L75)
  returning bech32. Applying it in the worktree failed to build until `pb3_gen.go`
  is regenerated, so the one-click suggestion does not exist and the change buys
  the reader nothing.
- [`.gitignore:60-61`](https://github.com/gnolang/gno/blob/23e9de5ad/.gitignore#L60-L61) · [↗](../../../../../.worktrees/gno-review-6120/.gitignore#L60-L61) — the `/docs/.preview/` entry belongs to a local docs
  workflow and has nothing to do with the bank. Already raised by junghoon-vans
  at [issuecomment-5522036917](https://github.com/gnolang/gno/pull/6120#issuecomment-5522036917),
  and still open at this head.
- [`bank_transfer_events.txtar:28`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/integration/testdata/bank_transfer_events.txtar#L28) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/integration/testdata/bank_transfer_events.txtar#L28) — half the fixture realm's guard is dead: a crossing function's own `cur` is current by construction, per
  [`interrealm_v2.md:336-339`](https://github.com/gnolang/gno/blob/23e9de5ad/gnovm/adr/interrealm_v2.md?plain=1#L336-L339), so `!cur.IsCurrent()` never fires and
  `!cur.Previous().IsUserCall()` carries the check alone. A txtar fixture is not
  deployed realm code, so this is a copying hazard rather than a defect.
- The PR body describes an `Amount std.Coins` field and separate debit-only and
  credit-only events for multisend. The head ships `Coins` and one
  `MultiTransferEvent`. It also says `SendCoinsUnrestricted`'s only caller today
  is gas payment, where the table above finds three. The ADR states both
  correctly, so the body alone is stale.

## Open questions

- The gas fee collector address appears in no event and in no transaction field,
  so an indexer crediting it has to learn it out of band. Not posted: the PR
  declares fees out of scope, and closing it is a second event type rather than
  an edit to this one.
