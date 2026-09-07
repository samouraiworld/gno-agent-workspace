// Re-checks the transaction surface after 0128eb5b5 unregistered MsgMultiSend.
// A std.Tx carrying one must be refused by amino again, as it is on master.
// Measured at 639d06bf2, where the test passes.
/* Run: from a gno checkout:
gh pr checkout 6120 -R gnolang/gno && git checkout 639d06bf2
curl -fsSL -o tm2/pkg/sdk/bank/zz_multisend_surface_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6120-bank-transfer-events/2-639d06bf2/tests/multisend_surface_test.go
go test -count=1 -v -run 'TestMultiSendStaysOffTheWire' ./tm2/pkg/sdk/bank/
rm tm2/pkg/sdk/bank/zz_multisend_surface_test.go
*/
package bank

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestMultiSendStaysOffTheWire(t *testing.T) {
	t.Parallel()

	a := crypto.AddressFromPreimage([]byte("a"))
	amt := std.NewCoins(std.NewCoin("ugnot", 5))

	_, err := amino.Marshal(std.Tx{Msgs: []std.Msg{NewMsgMultiSend(
		[]Input{NewInput(a, amt)}, []Output{NewOutput(a, amt)})}})
	require.Error(t, err, "MsgMultiSend must stay unregistered")
	t.Logf("encode refused: %v", err)
	require.True(t, strings.Contains(err.Error(), "unregistered concrete type bank.MsgMultiSend"))

	// TransferEvent is the one type the branch adds to the bank package.
	ev := TransferEvent{From: "g1from", To: "g1to", Coins: amt}
	bz, err := amino.MarshalAny(ev)
	require.NoError(t, err)
	t.Logf("TransferEvent still crosses the boundary: %d bytes", len(bz))
}
