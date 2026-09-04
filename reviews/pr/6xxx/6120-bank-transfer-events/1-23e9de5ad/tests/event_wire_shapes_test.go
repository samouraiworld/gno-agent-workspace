// Prints the two JSON encodings a bank.TransferEvent reaches a consumer through.
// amino.MarshalJSON is what every RPC response uses (rpc/lib/types/types.go);
// ResponseBase.EncodeEvents is used only by the gnokey result printer
// (crypto/keys/client/common.go). They disagree on the "coins" field.
/* Run: from a gno checkout:
gh pr checkout 6120 -R gnolang/gno && git checkout 23e9de5ad
curl -fsSL -o tm2/pkg/sdk/bank/zz_wire_shapes_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6120-bank-transfer-events/1-23e9de5ad/tests/event_wire_shapes_test.go
go test -count=1 -v -run 'TestTransferEventWireShapes' ./tm2/pkg/sdk/bank/
rm tm2/pkg/sdk/bank/zz_wire_shapes_test.go
*/
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

	// What an RPC client (a tx indexer) receives.
	t.Logf("amino JSON:    %s", amino.MustMarshalJSON(res.Events))
	// What `gnokey maketx ...` prints on the EVENTS: line.
	t.Logf("EncodeEvents:  %s", res.EncodeEvents())
}
