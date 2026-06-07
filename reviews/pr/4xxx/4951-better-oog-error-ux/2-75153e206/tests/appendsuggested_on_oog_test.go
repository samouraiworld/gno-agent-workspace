// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 4951 -R gnolang/gno && git checkout 75153e206
curl -fsSL -o tm2/pkg/crypto/keys/client/zz_appendsuggested_on_oog_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4951-better-oog-error-ux/2-75153e206/tests/appendsuggested_on_oog_test.go
go test -v -run 'AppendSuggestedGasWanted_OnServerOOG' ./tm2/pkg/crypto/keys/client/
rm tm2/pkg/crypto/keys/client/zz_appendsuggested_on_oog_test.go
*/

// appendSuggestedGasWanted is called at broadcast.go:157 whenever SimulateTx
// returns a non-nil result and the flow is not the DryRun-success branch. That
// includes the case where the server (or the client-synthesized check) marked
// DeliverTx as an out-of-gas error: GasUsed is then the partial gas consumed
// up to the panic, not the gas the tx really needs. The suggestion is computed
// straight off that partial number.
//
// Observed: with GasUsed=47 (partial, error set), Info becomes
// "suggested gas-wanted (gas used + 5%): 50" — a number derived from
// gas-until-panic, exactly the misleading shape this PR set out to remove.
//
// Flip: gate appendSuggestedGasWanted on res.DeliverTx.Error == nil. The
// assertion below then becomes Info == "" (no suggestion on an errored tx).
package client

import (
	"testing"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	ctypes "github.com/gnolang/gno/tm2/pkg/bft/rpc/core/types"
	"github.com/stretchr/testify/require"
)

func TestAppendSuggestedGasWanted_OnServerOOG_PartialGas(t *testing.T) {
	bres := &ctypes.ResultBroadcastTxCommit{
		DeliverTx: abci.ResponseDeliverTx{
			ResponseBase: abci.ResponseBase{Error: abci.StringError("out of gas")},
			GasUsed:      47, // partial gas consumed before the OOG panic
			GasWanted:    50,
		},
	}

	appendSuggestedGasWanted(bres)

	require.Equal(t, "suggested gas-wanted (gas used + 5%): 50", bres.DeliverTx.Info) // IS:     bug — suggestion from partial gas-until-panic
	// require.Equal(t, "", bres.DeliverTx.Info)                                      // SHOULD: no suggestion appended on an errored DeliverTx
}
