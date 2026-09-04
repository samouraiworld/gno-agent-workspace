# Review: [#6120](https://github.com/gnolang/gno/pull/6120)
Event: REQUEST_CHANGES

## Body
The description leaves out two paths that emit through [`sendCoins`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/keeper.go#L173-L193): the `MsgAddPackage` send envelope at [`keeper.go:1047`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/keeper.go#L1047) and the inert submission charge at [`keeper.go:952`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/sdk/vm/keeper.go#L952).

## tm2/pkg/sdk/bank/package.go:21 [gh](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/package.go#L21) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/package.go#L21)
Registering `MsgMultiSend` makes the chain accept a transaction it refused to decode before, and nothing in the event feature needs that registration.

<details><summary>repro</summary>

Deleting only this line and running the tx alongside the four event tests:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6120 -R gnolang/gno
cat > tm2/pkg/sdk/bank/zz_surface_test.go <<'EOF'
package bank

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestMultiSendTxDecodes(t *testing.T) {
	a := crypto.AddressFromPreimage([]byte("a"))
	amt := std.NewCoins(std.NewCoin("ugnot", 5))
	bz, err := amino.Marshal(std.Tx{Msgs: []std.Msg{NewMsgMultiSend(
		[]Input{NewInput(a, amt)}, []Output{NewOutput(a, amt)})}})
	if err != nil {
		t.Fatalf("node refuses the tx: %v", err)
	}
	var tx std.Tx
	if err := amino.Unmarshal(bz, &tx); err != nil {
		t.Fatalf("node refuses the tx: %v", err)
	}
	t.Logf("node accepts the tx: %T", tx.Msgs[0])
}
EOF
sed -i '/MsgMultiSend{}, "MsgMultiSend",/d' tm2/pkg/sdk/bank/package.go
go test -count=1 -v -run 'TestMultiSendTxDecodes|TestTransferEventAminoRoundTrip|TestHandlerEmitsTransferEvents|TestSendCoinsEmitsTransferEvent|TestInputOutputCoinsEmitsTransferEvents' ./tm2/pkg/sdk/bank/
git checkout tm2/pkg/sdk/bank/package.go && rm tm2/pkg/sdk/bank/zz_surface_test.go
```

The transaction is the only thing that stops working, so the registration is separable from the events it is credited to:

```
=== RUN   TestMultiSendTxDecodes
    zz_surface_test.go:17: node refuses the tx: MarshalAnyBinary2: cannot encode unregistered concrete type bank.MsgMultiSend
--- FAIL: TestMultiSendTxDecodes (0.00s)
--- PASS: TestTransferEventAminoRoundTrip (0.00s)
--- PASS: TestSendCoinsEmitsTransferEvent (0.00s)
--- PASS: TestInputOutputCoinsEmitsTransferEvents (0.00s)
--- PASS: TestHandlerEmitsTransferEvents (0.00s)
```

With the line restored the same test logs `node accepts the tx: bank.MsgMultiSend`, and against master's `package.go` the decode side answers `unrecognized concrete type full name bank.MsgMultiSend`. The `MultiTransferEvent` round trip in `TestTransferEventAminoRoundTrip/multisend` passes in both states, so `Input`, `Output` and the two event types carry the feature on their own.
</details>

## tm2/adr/pr6120_bank_transfer_events.md:37-40 [gh](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L37-L40) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L37-L40)
[`EncodeEvents`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/keyscli/root.go#L91) runs only in a CLI result printer, so an indexer gets `coins` as the one amino string this branch's [`events_test.go:29`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/events_test.go#L29) already asserts, not the array named here.

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

The first line is what every RPC response carries, since [`rpc/lib/types/types.go:207`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/rpc/lib/types/types.go#L207) serialises through `amino.MarshalJSON`, which calls [`std.Coins.MarshalAmino`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/std/coin.go#L216-L218):

```
amino JSON:    [{"@type":"/bank.TransferEvent","from":"g1from","to":"g1to","coins":"7ugnot"}]
EncodeEvents:  [{"from":"g1from","to":"g1to","coins":[{"denom":"ugnot","amount":7}]}]
```

[`ResponseBase.EncodeEvents`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/abci/types/types.go#L124-L132) has three callers, all CLI result printers: [`common.go:41`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/crypto/keys/client/common.go#L41) and [`:52`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/crypto/keys/client/common.go#L52), which are the tm2 client defaults `gnokey` replaces at [`root.go:38-40`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/keyscli/root.go#L38-L40), and [`root.go:91`](https://github.com/gnolang/gno/blob/23e9de5ad/gno.land/pkg/keyscli/root.go#L91), which is the line a `gnokey` user sees.

The PR description carries a third shape again, `"amount":[{"denom":"ugnot","amount":7}]`, from the earlier field name.
</details>

## tm2/adr/pr6120_bank_transfer_events.md:1 [gh](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/adr/pr6120_bank_transfer_events.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6120/tm2/adr/pr6120_bank_transfer_events.md#L1)
Nit: the title reads `PRxxxx`; 13 of the 17 other PR-named ADRs under `tm2/adr/` carry their number.

```suggestion
# PR6120: Structured bank transfer events
```

## tm2/pkg/sdk/bank/keeper.go:187 [gh](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/sdk/bank/keeper.go#L187) · [↗](../../../../../.worktrees/gno-review-6120/tm2/pkg/sdk/bank/keeper.go#L187)
Suggestion: the event set feeds the header's [`LastResultsHash`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/state/execution.go#L456) through [`ABCIResult`](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/types/results.go#L14-L18), so a validator still on the old binary [rejects](https://github.com/gnolang/gno/blob/23e9de5ad/tm2/pkg/bft/state/validation.go#L82-L86) the first block holding a ugnot transfer. Name the coordinated upgrade in the ADR's Consequences section.
