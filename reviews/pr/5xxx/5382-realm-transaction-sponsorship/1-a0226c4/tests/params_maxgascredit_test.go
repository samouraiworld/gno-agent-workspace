/* Run: from a gno checkout:
gh pr checkout 5382 -R gnolang/gno && git checkout a0226c4
curl -fsSL -o tm2/pkg/bft/types/zz_pr5382_maxgascredit_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/1-a0226c4/tests/params_maxgascredit_test.go
go test -v -run TestValidateConsensusParamsMaxGasCreditPerTx ./tm2/pkg/bft/types/
rm tm2/pkg/bft/types/zz_pr5382_maxgascredit_test.go

Pins ValidateConsensusParams for the new Block.MaxGasCreditPerTx: rejects < 0 and
any value above Block.MaxGas (unless MaxGas == -1). Passes at a0226c4; the "credit
window bigger than a block" rejection is the consensus-param backstop that keeps
one sponsored tx from being sized larger than any block can hold.
*/
package types

import (
	"testing"

	"github.com/stretchr/testify/assert"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
)

// TestValidateConsensusParamsMaxGasCreditPerTx exercises the PR 5382 validation
// of Block.MaxGasCreditPerTx: it must be >= 0 and must not exceed Block.MaxGas
// (unless MaxGas == -1, meaning no block gas bound). A credit window larger than
// a whole block would let one sponsored tx be sized bigger than any block can
// hold, so it must be rejected at param validation.
func TestValidateConsensusParamsMaxGasCreditPerTx(t *testing.T) {
	t.Parallel()

	mk := func(maxGas, credit int64) abci.ConsensusParams {
		return abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxTxBytes:        1,
				MaxDataBytes:      1024,
				MaxGas:            maxGas,
				TimeIotaMS:        10,
				MaxGasCreditPerTx: credit,
			},
			Validator: &abci.ValidatorParams{PubKeyTypeURLs: []string{"/tm.PubKeyEd25519"}},
		}
	}

	cases := []struct {
		name   string
		maxGas int64
		credit int64
		valid  bool
	}{
		{"disabled (credit 0)", 1_000_000, 0, true},
		{"credit below max gas", 1_000_000, 500_000, true},
		{"credit equal to max gas", 1_000_000, 1_000_000, true},
		{"credit above max gas rejected", 1_000_000, 1_000_001, false},
		{"negative credit rejected", 1_000_000, -1, false},
		{"unbounded block gas allows any credit", -1, 9_000_000_000, true},
		{"unbounded block gas still rejects negative", -1, -5, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateConsensusParams(mk(tc.maxGas, tc.credit))
			if tc.valid {
				assert.NoErrorf(t, err, "expected valid: maxGas=%d credit=%d", tc.maxGas, tc.credit)
			} else {
				assert.Errorf(t, err, "expected error: maxGas=%d credit=%d", tc.maxGas, tc.credit)
			}
		})
	}
}
