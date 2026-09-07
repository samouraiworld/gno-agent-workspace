# PR [#6120](https://github.com/gnolang/gno/pull/6120): feat(bank): emit transfer events for ugnot movements

URL: https://github.com/gnolang/gno/pull/6120
Author: notJoon | Base: master | Files: 14 | +347 -49
Reviewed by: davd-gzl | Model: claude-opus-5, effort high | Commit: 639d06bf2 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6120 639d06bf2`
Overview: [overview](../overview.md)

Round 2. The head moved from 23e9de5ad to 639d06bf2 over three commits.
1ad7398d4 merges master, which `git show --cc` prints with no conflict hunks,
and brings the genesis balance work from
[#6134](https://github.com/gnolang/gno/pull/6134). 0128eb5b5 withdraws multisend
from the change. 639d06bf2 stops the emit when the sender and the recipient are
the same address. The branch's own diff fell from 15 files
and +1063 -91 to 14 files and +347 -49. Round 1's first warning is resolved:
[`package.go`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/package.go#L15-L22) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/package.go#L15-L22)
now differs from master by `TransferEvent` alone, and amino refuses a `std.Tx`
carrying a `MsgMultiSend` again. The second warning, the nit and the suggestion
carry unchanged, each re-run against this head.

## Overview

Every ugnot move that goes through the bank keeper now appends a value to the
transaction's event list, so a service rebuilding account balances from
transaction results sees the moves it could not read off the message itself. A
realm calling `banker.SendCoins` and the `-send` envelope forwarded onward by a
realm were the two invisible ones. One emit inside the unexported `sendCoins`
covers `MsgSend`, the realm banker and every VM send envelope, because all three
funnel through it. That emit is now conditional: a transfer whose sender and
recipient are the same address writes nothing, which is what `maketx run`
produces on every send envelope. Multisend is out of the change entirely, and
the ADR records why.

**Verdict: REQUEST CHANGES** — the emit points, their ordering and the
self-transfer guard are right, and the ADR still names the encoding no indexer
reads as the indexer-facing contract (2 warnings, 1 suggestion, 2 nits).

## Verify first

- [`tm2/adr/pr6120_bank_transfer_events.md:40-43`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L40-L43) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L40-L43) — decide which encoding the compatibility contract names. Compare the ADR's shape against the branch's own [`events_test.go:18`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/events_test.go#L18) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/events_test.go#L18) and [`bank.proto:37`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/bank.proto#L37) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/bank.proto#L37), which both say `coins` is a string.
- [`tm2/pkg/sdk/bank/keeper.go:187`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L187) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187) — the guard sits below the session-allowance deduction at [`keeper.go:152`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L152) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L152), so a transaction signed by a session key, sending from its master address to that same master address, is free of an event and not free of allowance. Run `go test -run 'TestTestdata/session_self_transfer_events' ./gno.land/pkg/integration/` from [`tests/session_self_transfer_events.txtar`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6120-bank-transfer-events/2-639d06bf2/tests/session_self_transfer_events.txtar).

## Summary

An indexer that reconstructs per-address balances needs every balance change in a
transaction or an event, and two of the ugnot paths had neither. The fix is one
`EmitEvent` call in the keeper at
[`keeper.go:187-193`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L187-L193) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187-L193),
plus one event type and its amino registration. Placing it in the private
`sendCoins` rather than in the handlers is what makes the realm banker and the VM
send envelope free, and it also picks up the two paths the body does not list:
the `MsgAddPackage` send envelope and the inert submission charge. The
`fromAddr != toAddr` condition answers
[aeddi's request](https://github.com/gnolang/gno/pull/6120#discussion_r3931011661)
and removes the one event that reported a move that did not happen.

## Balance-mutating call sites

Every reachable caller of a `BankKeeper` mutator at 639d06bf2, and what the
transaction result says about it. Four rows moved since round 1: multisend and
the `MsgRun` envelope lost their events, `MsgSend` and the realm banker gained
the self-transfer condition, and genesis balances now enter through two
functions.

| Call site | What moves | Event |
| --- | --- | --- |
| [`bank/handler.go:50`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/handler.go#L50) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/handler.go#L50) | `MsgSend` | `TransferEvent`, unless sender and recipient match |
| [`bank/handler.go:72`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/handler.go#L72) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/handler.go#L72) | `MsgMultiSend` | none; unregistered again, so no transaction reaches this handler |
| [`vm/builtins.go:52`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/builtins.go#L52) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/builtins.go#L52) | realm `banker.SendCoins` | `TransferEvent`, unless sender and recipient match |
| [`vm/keeper.go:1212`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L1212) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L1212) | `MsgCall` send envelope | `TransferEvent` |
| [`vm/keeper.go:1425`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L1425) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L1425) | `MsgRun` send envelope | none; [`pkgAddr := caller`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L1386) makes every one a self-transfer |
| [`vm/keeper.go:1047`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L1047) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L1047) | `MsgAddPackage` send envelope | `TransferEvent`, unlisted in the body |
| [`vm/keeper.go:952`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L952) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L952) | inert submission charge | `TransferEvent`, unlisted in the body |
| [`auth/ante.go:481`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/auth/ante.go#L481) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/auth/ante.go#L481) | gas fee to the collector | none; amount on the transaction as `gas_fee`, collector address nowhere |
| [`vm/keeper.go:2381`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L2381) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L2381) | storage deposit locked | none; `StorageDepositEvent` carries the amount and `pkg_path`, not the addresses |
| [`vm/keeper.go:2396`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L2396) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/keeper.go#L2396) | storage deposit refunded | none; `StorageUnlockEvent`, same shape |
| [`vm/builtins.go:91`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/builtins.go#L91) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/builtins.go#L91) | realm `banker.IssueCoin` | none; realm-qualified denoms only, not ugnot |
| [`vm/builtins.go:100`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/builtins.go#L100) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/builtins.go#L100) | realm `banker.RemoveCoin` | none; same |
| [`gnoland/app.go:212`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/gnoland/app.go#L212) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/gnoland/app.go#L212) | genesis signer funding | none; gated on `BlockHeight() == 0` |
| [`gnoland/app.go:804-808`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/gnoland/app.go#L804-L808) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/gnoland/app.go#L804-L808) | genesis balances, `SetCoins` or `InitCoins` | none; genesis only |

The realm-denom rows cannot carry ugnot: both go through
[`assertIssuable`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/builtins.go#L85-L89) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/sdk/vm/builtins.go#L85-L89),
which rejects any denom `std.IsRealmDenom` refuses. `SubtractCoins` and
`AddCoins` have no callers outside the keeper and
[`supply.go:168`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/supply.go#L168) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/supply.go#L168),
which is `MintCoins`. `InitCoins` is new on master since round 1 and is reached
only from the genesis balance loader beside `SetCoins`.

So at runtime, three ugnot movements stay outside a transfer event: the gas fee
and the two storage deposit legs. All three are declared out of scope, and all
three are the paths where an indexer needs an address the events do not carry.

## Warnings (should fix)

- **[wrong wire contract]** [`pr6120_bank_transfer_events.md:40-43`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L40-L43) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L40-L43) — the shape named as indexer-facing is what a CLI result printer produces; an RPC client gets `coins` as one amino string instead.
  <details><summary>details</summary>

  `ResponseBase.EncodeEvents` has three callers and all are CLI result printers:
  [`common.go:41`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/crypto/keys/client/common.go#L41) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/crypto/keys/client/common.go#L41)
  and `:52`, the tm2 client defaults `gnokey` replaces at
  [`root.go:38-40`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/keyscli/root.go#L38-L40) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/keyscli/root.go#L38-L40),
  and
  [`root.go:91`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/keyscli/root.go#L91) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/keyscli/root.go#L91),
  the line a `gnokey` user actually sees. Every RPC response goes through
  [`amino.MarshalJSON`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/rpc/lib/types/types.go#L207) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/bft/rpc/lib/types/types.go#L207)
  instead, which calls `std.Coins.MarshalAmino` and renders the field as one
  string. Two artifacts inside this diff already say so: the branch's
  [`events_test.go:18`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/events_test.go#L18) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/events_test.go#L18)
  asserts `"coins":"5ugnot"`, and the generated schema at
  [`bank.proto:37`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/bank.proto#L37) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/bank.proto#L37)
  declares `string coins = 3`. The PR body carries a third shape again,
  `"amount":[{...}]`, from an earlier field name. Measured pair re-run at this
  head in
  [`tests/event_wire_shapes_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6120-bank-transfer-events/2-639d06bf2/tests/event_wire_shapes_test.go).
  Fix: state the amino form as the contract, since that is the one GnoScan
  receives.
  </details>

- **[allowance spent on a no-op]** [`keeper.go:187`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L187) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187) — `CheckAndDeductSessionSpend` runs above this guard at `keeper.go:152`, so a session-signed send from a master to that same master spends the session's `SpendLimit` and records nothing.
  <details><summary>details</summary>

  `SendCoins` deducts the session allowance at
  [`keeper.go:152`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L152) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L152)
  before it calls `sendCoins`, so the new guard cannot reach it. On a chain, an
  agent key delegated `bank/send` under a 15_000_000ugnot lifetime cap sends
  10_000_000ugnot from its master to that same master: the tx succeeds with
  `EVENTS:     []`, and the agent's next 10_000_000ugnot send, an amount the
  original cap covers, is refused with `session spend limit would be exceeded:
  attempted=10250001ugnot, used=10250001ugnot, limit=15000000ugnot`. Only
  250_001ugnot of that `used` is the first tx's gas fee, which
  [`ante.go:202`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/auth/ante.go#L202) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/auth/ante.go#L202)
  also charges to the session; gating the deduction on `fromAddr != toAddr` and
  re-running lets the second send through, so the refusal is the self-transfer's
  charge. The deduction predates this
  branch, and the event that used to record it does not: before 639d06bf2 the
  `TransferEvent` was the one artifact showing where the allowance went. The
  fixture is
  [`tests/session_self_transfer_events.txtar`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6120-bank-transfer-events/2-639d06bf2/tests/session_self_transfer_events.txtar),
  whose commented `SHOULD` assertion fails at this head. Fix: gate `CheckAndDeductSessionSpend` on the same `fromAddr != toAddr` condition.
  </details>

## Nits

- **[a guard clause that never fires]** [`bank_transfer_events.txtar:26`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/integration/testdata/bank_transfer_events.txtar#L26) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/integration/testdata/bank_transfer_events.txtar#L26) — `Forward` is a crossing function, so `cur.IsCurrent()` is always true and `!cur.IsCurrent()` is unreachable; [`interrealm_v2.md:336-339`](https://github.com/gnolang/gno/blob/639d06bf2/gnovm/adr/interrealm_v2.md?plain=1#L336-L339) · [↗](../../../../../.worktrees/gno-review-6120/gnovm/adr/interrealm_v2.md#L336-L339) states that the runtime ensures it. `cur.Previous().IsUserCall()` carries the check alone. The whole file is added by this diff, and a fixture is what the next realm gets copied from. Fix: drop the first clause.
- [`pr6120_bank_transfer_events.md:1`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L1) — the title reads `PRxxxx`; 13 of the 17 other PR-named ADRs under `tm2/adr/` carry their number.

## Suggestions

- **[rollout]** [`pr6120_bank_transfer_events.md:71-73`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L71-L73) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L71-L73) — the event set is inside the block header, so this cannot go out as a rolling restart, and Consequences names no coordinated upgrade.
  <details><summary>details</summary>

  `ABCIResult` carries `Events` at
  [`results.go:14-18`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/types/results.go#L14-L18) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/bft/types/results.go#L14-L18),
  the merkle root of that set becomes `LastResultsHash` at
  [`execution.go:456`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/state/execution.go#L456) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/bft/state/execution.go#L456),
  and every node rejects a block whose header disagrees with its own execution at
  [`validation.go:82-86`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/state/validation.go#L82-L86) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/bft/state/validation.go#L82-L86).
  A validator on the old binary and one on the new binary therefore disagree on
  the first block containing any ugnot transfer. The chain has the machinery for
  this in the halt-height and minimum-version params read by
  [`checkNodeStartupParams`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/gnoland/node_params.go#L133-L137) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/gnoland/node_params.go#L133-L137);
  the Consequences paragraph says the change is consensus-visible and stops
  there. Fix: name the coordinated upgrade in that paragraph.
  </details>

## Verified

- Round 1's first warning is closed by 0128eb5b5. That commit takes `Input`,
  `Output`, `MsgMultiSend` and `MultiTransferEvent` back out of the amino
  registration, leaving
  [`package.go:15-22`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/package.go#L15-L22) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/package.go#L15-L22)
  equal to master's list plus `TransferEvent`. Marshalling a `std.Tx` carrying a
  `MsgMultiSend` answers `cannot encode unregistered concrete type
  bank.MsgMultiSend`, as it does on master. Run in
  [`tests/multisend_surface_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6120-bank-transfer-events/2-639d06bf2/tests/multisend_surface_test.go).
- The self-transfer guard has regression cover. Deleting the `fromAddr != toAddr`
  condition and re-running the fixture fails at
  [`bank_transfer_events.txtar:15`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/integration/testdata/bank_transfer_events.txtar#L15) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/integration/testdata/bank_transfer_events.txtar#L15)
  with `no match for EVENTS: \[\] found in stdout`, the printed line being the
  caller-to-caller transfer of 42ugnot. No Go test asserts it, and the txtar
  does.
- Only the cross-address rows emit. Sender identity crossed with zero, partial,
  whole and over-balance amounts gives eight cells: every self row moves nothing
  and emits nothing, and every zero row returns at
  [`keeper.go:138`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L138) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L138)
  before reaching the emit. An over-balance self-transfer still fails with
  `insufficient coins error`, because the debit runs before the credit. Table in
  [`tests/self_transfer_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6120-bank-transfer-events/2-639d06bf2/tests/self_transfer_test.go).
- The merge of master is clean. `git show 1ad7398d4 --cc` prints no hunk, so
  nothing in it is conflict-resolution content.
- Determinism holds: the emit appends in execution order to a slice, no map is
  iterated, and the guard compares two `crypto.Address` values.
- Events do not survive a failed message. One logger is created per transaction
  at [`baseapp.go:687`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/baseapp.go#L687) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/baseapp.go#L687)
  and copied into the result only under `if err == nil` at
  [`baseapp.go:737-739`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/baseapp.go#L737-L739) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/baseapp.go#L737-L739).
- Green at 639d06bf2: `go test -run 'TestTransferEventAminoRoundTrip|TestHandlerEmitsTransferEvents|TestSendCoinsEmitsTransferEvent|TestBankKeeperSendCoinsZero|TestBankKeeper$|TestInitCoinsMatchesSetCoinsOnFreshAddress|TestInitCoinsRejectsInvalidCoinsLikeSetCoins|TestInitCoinsWholeLoadMatchesSetCoins|TestInitCoinsDoesNotDrainStaleKeys' ./tm2/pkg/sdk/bank/`,
  `go test -run 'TestVMKeeperOriginSend1' ./gno.land/pkg/sdk/vm/`,
  `go test -run 'TestTestdata/bank_transfer_events' ./gno.land/pkg/integration/`.

## Existing threads

| Reviewer | Gist | State | Overlap |
| --- | --- | --- | --- |
| julienrbrt | [`bank.proto`](https://github.com/gnolang/gno/pull/6120#discussion_r3929130447): where is `MultiTransferEvent` used | answered by 0128eb5b5, and approved | closes round 1's first warning |
| aeddi | [`keeper.go`](https://github.com/gnolang/gno/pull/6120#discussion_r3931011661): skip the emit when `fromAddr == toAddr` | answered by 639d06bf2, and approved | the second warning here is about where that skip sits |
| jinoosss | [`keeper.go:129`](https://github.com/gnolang/gno/pull/6120#discussion_r3922334625): split from-only and to-only multisend events read as burn then mint | moot; multisend left the change | none |
| junghoon-vans | [the `/docs/.preview/` hunk is unrelated](https://github.com/gnolang/gno/pull/6120#issuecomment-5522036917) | open and unanswered four days on | takes the `.gitignore` item below |

The PR carries four approvals at this head, from moul, aeddi, julienrbrt and
jinoosss, the last of them dated after 639d06bf2. Every check run on the commit
concludes `success`.

## Not posted

- [`.gitignore:59-61`](https://github.com/gnolang/gno/blob/639d06bf2/.gitignore#L59-L61) · [↗](../../../../../.worktrees/gno-review-6120/.gitignore#L59-L61) — the `/docs/.preview/` entry belongs to a local docs
  workflow and has nothing to do with the bank. Already raised by junghoon-vans
  at [issuecomment-5522036917](https://github.com/gnolang/gno/pull/6120#issuecomment-5522036917)
  and unanswered since, with the hunk still in the diff at this head.
- [`events.go:9-10`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/events.go#L9-L10) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/events.go#L9-L10) — `TransferEvent.From` and `.To` are `string`
  where `Input.Address` and `Output.Address` in the same package are
  `crypto.Address`, which validates bech32 on decode. The wire shape is the same
  either way, since `crypto.Address` carries both
  [`MarshalAmino`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/crypto/crypto.go#L85-L87) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/crypto/crypto.go#L85-L87)
  and
  [`MarshalJSON`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/crypto/crypto.go#L72-L75) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/crypto/crypto.go#L72-L75)
  returning bech32. Applying it in the worktree failed to build until `pb3_gen.go`
  is regenerated, so the one-click suggestion does not exist and the change buys
  the reader nothing.
- [`pr6120_bank_transfer_events.md:45-48`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L45-L48) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L45-L48) — the sentence
  names `MsgRun` as the case the guard covers, where the guard is on the shared
  keeper and covers a `MsgSend` to your own address and a realm banker's
  self-send too. Both are balance-neutral and the message names both addresses,
  so nothing is lost; the fix is one word in the ADR and does not change what the
  author does next.
- The PR body still describes an `Amount std.Coins` field and separate debit-only
  and credit-only events for multisend, neither of which exists at this head. It
  also says `SendCoinsUnrestricted`'s only caller today is gas payment, where the
  table above finds three. The ADR states all three as the head has them, so the
  body alone is stale.

## Open questions

- The gas fee collector address appears in no event and in no transaction field,
  so an indexer crediting it has to learn it out of band. Not posted: the PR
  declares fees out of scope, and closing it is a second event type rather than
  an edit to this one.
