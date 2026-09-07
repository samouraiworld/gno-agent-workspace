# Review: [#6120](https://github.com/gnolang/gno/pull/6120)
Event: REQUEST_CHANGES

## Body
The description leaves out two paths that emit through [`sendCoins`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L173-L196): the `MsgAddPackage` send envelope at [`keeper.go:1047`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L1047) and the inert submission charge at [`keeper.go:952`](https://github.com/gnolang/gno/blob/639d06bf2/gno.land/pkg/sdk/vm/keeper.go#L952).

## tm2/pkg/sdk/bank/keeper.go:187 [gh](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L187) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187)
This guard sits below the session key's `SpendLimit` deduction at [`keeper.go:152`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/sdk/bank/keeper.go#L152), so a self-transfer drains the allowance with nothing on chain recording it. Wrapping the `CheckAndDeductSessionSpend` call in the same condition leaves the funds check and the restricted-denom check where they are.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6120 -R gnolang/gno
cat > tm2/pkg/sdk/bank/zz_session_test.go <<'EOF'
package bank

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestSelfTransferSpendsSessionLimit(t *testing.T) {
	env := setupTestEnv()
	ctx, master, da := setupSessionCtx(t, env,
		std.NewCoins(std.NewCoin("ugnot", 1000)),
		std.NewCoins(std.NewCoin("ugnot", 500)))

	before := env.bankk.GetCoins(ctx, master).AmountOf("ugnot")
	require.NoError(t, env.bankk.SendCoins(ctx, master, master,
		std.NewCoins(std.NewCoin("ugnot", 400))))

	require.Equal(t, before, env.bankk.GetCoins(ctx, master).AmountOf("ugnot"))
	require.Empty(t, ctx.EventLogger().Events())
	require.Equal(t, int64(0), da.GetSpendUsed().AmountOf("ugnot"))

	require.NoError(t, env.bankk.SendCoins(ctx, master, master,
		std.NewCoins(std.NewCoin("ugnot", 200))))
}
EOF
go test -count=1 -v -run TestSelfTransferSpendsSessionLimit ./tm2/pkg/sdk/bank/
rm tm2/pkg/sdk/bank/zz_session_test.go
```

The balance and the event list come back as the test expects, and the allowance does not, which is the finding:

```
=== RUN   TestSelfTransferSpendsSessionLimit
    zz_session_test.go:23:
# …
        	Error:      	Not equal:
        	            	expected: 0
        	            	actual  : 400
--- FAIL: TestSelfTransferSpendsSessionLimit (0.00s)
```

The 400ugnot is gone from a 500ugnot limit, so the trailing 200ugnot send is refused. The deduction predates this branch and the guard above it does not reach it.
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

The PR description carries a third shape again, `"amount":[{"denom":"ugnot","amount":7}]`, from the earlier field name.
</details>

## tm2/adr/pr6120_bank_transfer_events.md:1 [gh](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L1)
Nit: the title reads `PRxxxx`; 13 of the 17 other PR-named ADRs under `tm2/adr/` carry their number.

```suggestion
# PR6120: Structured bank transfer events
```

## tm2/adr/pr6120_bank_transfer_events.md:71-73 [gh](https://github.com/gnolang/gno/blob/639d06bf2/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L71-L73) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L71-L73)
Suggestion: the event set feeds the header's [`LastResultsHash`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/state/execution.go#L456) through [`ABCIResult`](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/types/results.go#L14-L18), so a validator on the old binary [rejects](https://github.com/gnolang/gno/blob/639d06bf2/tm2/pkg/bft/state/validation.go#L82-L86) the first block holding a ugnot transfer, which this paragraph should name as a coordinated upgrade.
