// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 4008 -R gnolang/gno
# (save this file at gnovm/stdlibs/chain/adv_prefix_test.gno)
# Note: this file targets the post-std-split layout (gnovm/stdlibs/chain/),
# not the PR's original pre-split path. If your checkout predates the split,
# adjust the destination to the equivalent pre-split path.
# The PR's runtime change to address.IsValid is what the test exercises;
# build the gno binary from the PR source (or master if you want to test
# the post-split layout in isolation):
go build -o /tmp/gno-bin ./gnovm/cmd/gno
(cd gnovm/stdlibs/chain && /tmp/gno-bin test -v -run TestPrefixCheck .)
rm -f gnovm/stdlibs/chain/adv_prefix_test.gno
*/
package chain_test

import (
	"testing"
)

func TestPrefixCheck(t *testing.T) {
	type test struct {
		input    address
		expected bool
	}

	cases := []test{
		{"g1f4v282mwyhu29afke4vq5r2xzcm6z3ftnugcnv", true},
		{"cosmos1qxy2gn0hu4xnq03tmgfhfd7zvc2hr5dxztpmcd", false},
		{"bc156tlrfxxxelwrmvu0v986psjln9ry60ef34yp2", false},
		{"", false},
	}
	for _, tc := range cases {
		got := tc.input.IsValid()
		if got != tc.expected {
			t.Fatalf("input %q: expected %t got %t", string(tc.input), tc.expected, got)
		}
	}
}
