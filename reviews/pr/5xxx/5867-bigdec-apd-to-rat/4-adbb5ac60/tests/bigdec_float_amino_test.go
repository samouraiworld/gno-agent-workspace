/* Run: from a gno checkout:
gh pr checkout 5867 -R gnolang/gno && git checkout adbb5ac60
curl -fsSL -o gnovm/pkg/gnolang/bigdec_float_amino_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/4-adbb5ac60/tests/bigdec_float_amino_test.go
go test -v -run 'TestBigdecFloatFormAmino' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/bigdec_float_amino_test.go
*/
// parity_test.go round-trips only the rat form. The float form ("f:" prefix,
// big.Float.Text('p',0) / big.ParseFloat) has no coverage, yet it reaches
// persisted realm state: a package-level const past the 4096-bit rat ceiling
// (e.g. `const Huge = 1e5000`) persists in float form. A decoded big.Float is
// not reflect.DeepEqual to the original (non-canonical mant slice length), so
// aminotest.AssertCodecParity rejects it; the round-trip must be asserted by
// numeric equality plus re-marshal stability instead. Green at adbb5ac60.

package gnolang

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBigdecFloatFormAmino(t *testing.T) {
	orig := parseBigdecLiteral("1e5000", "decimal")
	require.NotNil(t, orig.F, "1e5000 overflows the rat ceiling and must be float form")

	s, err := orig.MarshalAmino()
	require.NoError(t, err)

	var back BigdecValue
	require.NoError(t, back.UnmarshalAmino(s))
	require.NotNil(t, back.F, "unmarshal must keep float form")
	require.Zero(t, back.F.Cmp(orig.F), "float-form value must survive the round-trip")

	// Consensus needs a byte-stable re-encoding: a value decoded from the store
	// must marshal back to the identical string despite big.Float normalizing
	// its internal mantissa slice on decode.
	s2, err := back.MarshalAmino()
	require.NoError(t, err)
	require.Equal(t, s, s2, "re-marshal must be byte-identical")

	// The prefix keeps float-form and rat-form encodings distinguishable.
	var ratForm BigdecValue
	require.NoError(t, ratForm.UnmarshalAmino("1/3"))
	require.Nil(t, ratForm.F)
	require.NotNil(t, ratForm.V)
	require.Zero(t, ratForm.V.Cmp(big.NewRat(1, 3)))
}
