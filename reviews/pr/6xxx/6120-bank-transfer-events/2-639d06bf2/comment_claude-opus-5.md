# Review: [#6120](https://github.com/gnolang/gno/pull/6120)
Event: REQUEST_CHANGES

## Body
Two further paths funnel through [`sendCoins`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L173-L196) and so emit too: the `MsgAddPackage` send envelope at [`keeper.go:1047`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L1047) and the inert submission charge at [`keeper.go:952`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L952).

## tm2/pkg/sdk/bank/keeper.go:187 [gh](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L187) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187)
This guard sits below the session key's `SpendLimit` deduction at [`keeper.go:152`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L152), so a transaction signed by a session key, sending from its master address to that same master address, spends the allowance with nothing on chain recording where the coins went. Wrapping the `CheckAndDeductSessionSpend` call in that same condition skips the charge and nothing else: the restricted-denom check above it and `SubtractCoins`'s balance check below it both still run on a self-transfer.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6120 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/session_self_transfer_events.txtar <<'EOF'
gnoland start

# Fixed mnemonic, so the agent key's address and the bech32 pubkey below are
# deterministic. It cannot come from adduserfrom: that writes a genesis balance,
# and handleMsgCreateSession refuses a session key that already has an account.
input wage renew timber answer someone model torch cake ostrich sort appear walk kiss expose magnet crisp keen skin enter opinion desk dice lyrics reflect
input test123
input test123
gnokey add agent --recover --insecure-password-stdin
stdout 'g1k4a0flmuxppxhj4k80d0s3lj8zw3tqcxpju054'

# test1 grants the agent key MsgSend on its behalf, with a 15_000_000ugnot cap
# for the session's whole lifetime.
gnokey maketx session create -pubkey gpub1pgfj7ard9eg82cjtv4u4xetrwqer2dntxyfzxz3pqtr8vl7pwruxukl3kd3h9zs378r3alnykuxrm7avfeh8mcym30vu2d27anu -expires-at none -allow-paths bank/send -spend-limit 15000000ugnot -gas-fee 1000000ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1
stdout 'OK!'

# The agent moves 10_000_000ugnot from test1 to test1. Sender and recipient are
# both the master address, which is the case the guard covers.
input test123
gnokey maketx send -send 10000000ugnot -to $test1_user_addr -master test1 -gas-fee 250001ugnot -gas-wanted 2_500_000 -chainid=tendermint_test -insecure-password-stdin agent
stdout 'EVENTS:     \[\]'   # IS:     the 10_000_000ugnot the session paid for is unrecorded
# stdout 'EVENTS:     \[{"from":"'$test1_user_addr'","to":"'$test1_user_addr'","coins":\[{"denom":"ugnot","amount":10000000}\]}\]'  # SHOULD: the movement the allowance was spent on is on the event stream

# The budget was charged all the same. The agent's next MsgSend, for an amount
# the original 15_000_000ugnot cap covers, is refused: SpendUsed already stands
# at 10_250_001ugnot, and only 250_001ugnot of that is the previous tx's gas.
input test123
! gnokey maketx send -send 10000000ugnot -to g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5 -master test1 -gas-fee 250001ugnot -gas-wanted 2_500_000 -chainid=tendermint_test -insecure-password-stdin agent
stderr 'session spend limit would be exceeded: attempted=10250001ugnot, used=10250001ugnot, limit=15000000ugnot'
EOF
go test -count=1 -v -run 'TestTestdata/session_self_transfer_events' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/session_self_transfer_events.txtar
```

The chain refuses the agent's second send because the first one, which the event list does not record, took 10_000_000ugnot of the 15_000_000ugnot cap; uncommenting the `# SHOULD:` line is what fails at this head:

```
> gnokey maketx send -send 10000000ugnot -to $test1_user_addr -master test1 ... agent
OK!
EVENTS:     []
# …
> ! gnokey maketx send -send 10000000ugnot -to g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5 -master test1 ... agent
"gnokey" error: session spend limit would be exceeded: attempted=10250001ugnot, used=10250001ugnot, limit=15000000ugnot
--- PASS: TestTestdata/session_self_transfer_events (6.14s)
```

Gating the deduction on `fromAddr != toAddr` and re-running lets that second send through, so the refusal is the self-transfer's charge and not the gas.
</details>

## tm2/adr/pr6120_bank_transfer_events.md:40-43 [gh](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L40-L43) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L40-L43)
[`EncodeEvents`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/keyscli/root.go#L91) runs only in a CLI result printer, so an indexer gets `coins` as the one amino string that [`events_test.go:18`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/events_test.go#L18) and [`bank.proto:37`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/bank.proto#L37) both already declare, not the array named here.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6120 -R gnolang/gno
cat > tm2/pkg/sdk/bank/zz_shapes_test.go <<'EOF'
package bank

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestTransferEventWireShapes(t *testing.T) {
	res := abci.ResponseDeliverTx{ResponseBase: abci.ResponseBase{
		Events: []abci.Event{
			TransferEvent{From: "g1from", To: "g1to", Coins: std.NewCoins(std.NewCoin("ugnot", 7))},
		},
	}}
	t.Logf("amino JSON:    %s", amino.MustMarshalJSON(res.Events))
	t.Logf("EncodeEvents:  %s", res.EncodeEvents())
}
EOF
go test -count=1 -v -run TestTransferEventWireShapes ./tm2/pkg/sdk/bank/
rm tm2/pkg/sdk/bank/zz_shapes_test.go
```

The first line is what every RPC response carries, since [`rpc/lib/types/types.go:207`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/rpc/lib/types/types.go#L207) serialises through `amino.MarshalJSON`, which calls [`std.Coins.MarshalAmino`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/std/coin.go#L216-L218):

```
amino JSON:    [{"@type":"/bank.TransferEvent","from":"g1from","to":"g1to","coins":"7ugnot"}]
EncodeEvents:  [{"from":"g1from","to":"g1to","coins":[{"denom":"ugnot","amount":7}]}]
```

[`ResponseBase.EncodeEvents`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/abci/types/types.go#L124-L132) has three callers, all CLI result printers: [`common.go:41`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/crypto/keys/client/common.go#L41) and [`:52`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/crypto/keys/client/common.go#L52), which are the tm2 client defaults `gnokey` replaces at [`root.go:38-40`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/keyscli/root.go#L38-L40), and [`root.go:91`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/keyscli/root.go#L91), which is the line a `gnokey` user sees.
</details>

## gno.land/pkg/integration/testdata/bank_transfer_events.txtar:26 [gh](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/integration/testdata/bank_transfer_events.txtar#L26) · [↗](../../../../../.worktrees/gno-review-6120/gno.land/pkg/integration/testdata/bank_transfer_events.txtar#L26)
Nit: `Forward` is a crossing function, so `cur.IsCurrent()` is always true here and this half of the guard never fires, per [`interrealm_v2.md:336-339`](https://github.com/gnolang/gno/blob/639d06bf2/gnovm/adr/interrealm_v2.md?plain=1#L336-L339). `cur.Previous().IsUserCall()` carries the check alone, and a fixture is what the next realm gets copied from.

```suggestion
	if !cur.Previous().IsUserCall() {
```

## tm2/adr/pr6120_bank_transfer_events.md:1 [gh](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L1)
Nit: the title reads `PRxxxx`; 13 of the 17 other PR-named ADRs under `tm2/adr/` carry their number.

```suggestion
# PR6120: Structured bank transfer events
```

## tm2/adr/pr6120_bank_transfer_events.md:71-73 [gh](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L71-L73) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L71-L73)
Suggestion: the event set feeds the header's [`LastResultsHash`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/state/execution.go#L456) through [`ABCIResult`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/types/results.go#L14-L18), so a validator on the old binary [rejects](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/state/validation.go#L82-L86) the first block holding a ugnot transfer, which this paragraph should name as a coordinated upgrade.
