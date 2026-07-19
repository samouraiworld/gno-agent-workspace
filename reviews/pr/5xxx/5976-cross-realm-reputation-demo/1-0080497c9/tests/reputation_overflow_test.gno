/* Run: from a gno checkout:
gh pr checkout 5976 -R gnolang/gno && git checkout 0080497c9
curl -fsSL -o examples/gno.land/r/demo/reputation/reputation_overflow_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5976-cross-realm-reputation-demo/1-0080497c9/tests/reputation_overflow_test.gno
cd examples && go run ../gnovm/cmd/gno test -v -run 'TestAddPointsRejectsOverflow' ./gno.land/r/demo/reputation
rm gno.land/r/demo/reputation/reputation_overflow_test.gno
*/

package reputation

import (
	"testing"

	"gno.land/p/nt/uassert/v0"
)

// AddPoints checks each individual amount is positive but adds it to the
// stored total without an overflow check. At 0080497c9 two MaxInt64 awards
// wrap the score to -2; a fixed AddPoints aborts on the second award.
func TestAddPointsRejectsOverflow(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/ovfcaller"))
	target := address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")

	const maxInt64 = int64(9223372036854775807)
	AddPoints(cross(cur), target, "ovf", maxInt64)

	uassert.AbortsWithMessage(t, cur,
		"reputation: points overflow",
		func() {
			AddPoints(cross(cur), target, "ovf", maxInt64)
		})

	// Baseline invariant: a rejected award leaves both ledgers untouched.
	uassert.Equal(t, maxInt64, GetScore(target, "ovf", "gno.land/r/demo/ovfcaller"))
	uassert.Equal(t, maxInt64, GetTotalScore(target))
}
