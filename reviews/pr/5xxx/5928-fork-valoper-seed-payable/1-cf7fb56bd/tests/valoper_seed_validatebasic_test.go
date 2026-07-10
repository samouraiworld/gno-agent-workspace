/* Run: from a gno checkout:
gh pr checkout 5928 -R gnolang/gno
curl -fsSL -o contribs/gnogenesis/internal/fork/valoper_seed_validatebasic_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5928-fork-valoper-seed-payable/1-cf7fb56bd/tests/valoper_seed_validatebasic_test.go
cd contribs/gnogenesis && go test -run TestValoperSeed_EmittedTxPassesValidateBasic -v ./internal/fork/
rm internal/fork/valoper_seed_validatebasic_test.go
*/

// The PR's whole point is that the emitted .jsonl now replays: each line
// survives the amino round-trip and passes std.Tx.ValidateBasic. No
// existing test asserts that end state, so a revert to a zero fee or an
// empty signature slice would pass CI. This locks it in.

package fork

import (
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/stretchr/testify/require"
)

func TestValoperSeed_EmittedTxPassesValidateBasic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	csv := validHeader + "\n" +
		opAddrA + "," + validPubKeyA + ",alice,Alice,cloud\n" +
		opAddrB + "," + validPubKeyB + ",bob,Bob,on-prem\n"

	out, err := runSeed(t, dir, csv)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	for i, line := range lines {
		var at AnnotatedTx
		require.NoError(t, amino.UnmarshalJSON([]byte(line), &at))
		// Non-zero fee keeps the ugnot denom through the round-trip and one
		// placeholder signature per signer satisfies the sig-count rule.
		// A zero fee (denom collapses to "") or an empty sig slice fails here.
		require.NoErrorf(t, at.Tx.ValidateBasic(),
			"emitted valoper-seed tx %d must pass ValidateBasic", i)
	}
}
