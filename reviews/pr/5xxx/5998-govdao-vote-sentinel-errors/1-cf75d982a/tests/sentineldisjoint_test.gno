/* Run: from a gno checkout:
gh pr checkout 5998 -R gnolang/gno && git checkout cf75d982a
curl -fsSL -o examples/gno.land/r/gov/dao/v3/impl/sentineldisjoint_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5998-govdao-vote-sentinel-errors/1-cf75d982a/tests/sentineldisjoint_test.gno
cd examples && go run ../gnovm/cmd/gno test -run TestSentinelsAreDisjoint ./gno.land/r/gov/dao/v3/impl
rm gno.land/r/gov/dao/v3/impl/sentineldisjoint_test.gno
*/

// Each sentinel must match itself and nothing else, and ProposalClosedError
// must match only ErrProposalClosed. Green at cf75d982a. Replacing the body of
// ProposalClosedError.Is with "return true" keeps TestVoteOnProposalErrors
// green and turns this red.

package impl

import (
	"errors"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

func TestSentinelsAreDisjoint(t *testing.T) {
	sentinels := []error{
		ErrMemberNotFound,
		ErrProposalNotFound,
		ErrNotAllowedToVote,
		ErrAlreadyVoted,
		ErrInvalidVoteOption,
		ErrProposalClosed,
	}

	for i, a := range sentinels {
		for j, b := range sentinels {
			uassert.Equal(t, i == j, errors.Is(a, b))
		}
	}

	closed := error(&ProposalClosedError{Accepted: true})
	for _, s := range sentinels {
		uassert.Equal(t, s == ErrProposalClosed, errors.Is(closed, s))
	}
	uassert.False(t, errors.Is(ErrProposalClosed, closed))
}
