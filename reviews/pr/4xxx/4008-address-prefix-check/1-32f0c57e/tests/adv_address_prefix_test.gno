/* Run:
   cp reviews/pr/4xxx/4008-address-prefix-check/1-32f0c57e/tests/adv_address_prefix_test.gno \
      .worktrees/gno-review-4008/gnovm/stdlibs/chain/adv_prefix_test.gno
   (cd .worktrees/gno-review-4008/gnovm/stdlibs/chain && /tmp/gno-master test -v -run TestPrefixCheck .)
   rm -f .worktrees/gno-review-4008/gnovm/stdlibs/chain/adv_prefix_test.gno

   Note: this test file lives at the post-std-split location (`gnovm/stdlibs/chain/`),
   not the PR's pre-split path. Build /tmp/gno-master from origin/master beforehand:
   (cd gno && go build -o /tmp/gno-master ./gnovm/cmd/gno)
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
