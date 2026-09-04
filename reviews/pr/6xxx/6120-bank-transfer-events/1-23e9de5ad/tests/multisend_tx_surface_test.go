// Asserts that a node decodes a std.Tx carrying bank.MsgMultiSend, which is the
// accept-or-reject decision CheckTx makes on a broadcast transaction.
// Measured: passes at 23e9de5ad, fails with master's tm2/pkg/sdk/bank/package.go
// in place with "unrecognized concrete type full name bank.MsgMultiSend".
/* Run: from a gno checkout:
gh pr checkout 6120 -R gnolang/gno && git checkout 23e9de5ad
curl -fsSL -o tm2/pkg/sdk/bank/zz_multisend_surface_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6120-bank-transfer-events/1-23e9de5ad/tests/multisend_tx_surface_test.go
go test -count=1 -v -run 'TestMultiSendIsABroadcastableMsg' ./tm2/pkg/sdk/bank/
git show origin/master:tm2/pkg/sdk/bank/package.go > tm2/pkg/sdk/bank/package.go
go test -count=1 -v -run 'TestMultiSendIsABroadcastableMsg' ./tm2/pkg/sdk/bank/
git checkout tm2/pkg/sdk/bank/package.go && rm tm2/pkg/sdk/bank/zz_multisend_surface_test.go
*/
package bank

import (
	"encoding/hex"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/gnolang/gno/tm2/pkg/std"
)

// One std.Tx holding one MsgMultiSend, amino-encoded at 23e9de5ad. Hardcoded so
// the decode side runs under master's registration too, which cannot encode it.
const multiSendTxWire = "0a7e0a122f62616e6b2e4d73674d756c746953656e6412680a320a2867313974" +
	"6c6763396b767267393736756a7a6a78713239666738767a7a39377671773536" +
	"3836686412063575676e6f7412320a28673173706d6a66357079326a33706a66" +
	"63647375686a647673726d6566797737376132706d39747412063575676e6f74"

func TestMultiSendIsABroadcastableMsg(t *testing.T) {
	bz, err := hex.DecodeString(multiSendTxWire)
	if err != nil {
		t.Fatal(err)
	}

	var tx std.Tx
	if err := amino.Unmarshal(bz, &tx); err != nil {
		t.Fatalf("node refuses the transaction: %v", err)
	}
	if _, ok := tx.Msgs[0].(MsgMultiSend); !ok {
		t.Fatalf("decoded msg is %T, want bank.MsgMultiSend", tx.Msgs[0])
	}
	t.Logf("node accepts the transaction: %T", tx.Msgs[0])
}
