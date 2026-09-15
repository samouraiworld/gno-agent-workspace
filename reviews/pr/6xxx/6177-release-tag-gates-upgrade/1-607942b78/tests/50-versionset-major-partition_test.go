package versionset

/* Run: from a gno checkout:
gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78
curl -fsSL -o tm2/pkg/versionset/versionset_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/50-versionset-major-partition_test.go
go test -v -run 'TestCompatibleWith' ./tm2/pkg/versionset/
rm tm2/pkg/versionset/versionset_test.go
*/

// Asserts the claim tm2/pkg/bft/version/version.go:20,
// misc/release/bump-protocol-version.sh:131, misc/release/README.md:67 and
// RELEASING.md:163 make: CompatibleWith refuses a peer whose MAJOR differs and
// accepts one whose MINOR differs. Measured at head 607942b78: both pass, and
// the package ships no test file, so nothing held either half in place.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func set(version string) VersionSet {
	return VersionSet{{Name: "bft", Version: version}}
}

func TestCompatibleWith(t *testing.T) {
	t.Parallel()

	t.Run("major differs: refused", func(t *testing.T) {
		t.Parallel()

		// The four release-tooling claims: bumping the protocol major is a
		// flag day because these two nodes cannot gossip.
		_, err := set("v1.0.0-rc.0").CompatibleWith(set("v2.0.0"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "VersionInfos not compatible")

		// Symmetric: the refusal does not depend on which side asks.
		_, err = set("v2.0.0").CompatibleWith(set("v1.0.0-rc.0"))
		require.Error(t, err)
	})

	t.Run("minor differs: accepted", func(t *testing.T) {
		t.Parallel()

		// The other half of the same claim: a MINOR bump is not a partition,
		// which is why bump-protocol-version.sh warns only on a major.
		res, err := set("v1.0.0").CompatibleWith(set("v1.9.0"))
		require.NoError(t, err)
		require.Len(t, res, 1)

		// Today's negotiated minor is the receiver's, not the lower of the
		// two: versionset.go:92 guards with semver.Compare inside a branch
		// where both majors are already equal, so that comparison is always 0
		// and the first arm never runs.
		assert.Equal(t, "v1.0", res[0].Version) // IS:     receiver's minor wins
		// assert.Equal(t, "v1.0", res[0].Version) // SHOULD: the lower minor, whichever side asks

		// Same pair, asked the other way round: the intersection differs.
		res, err = set("v1.9.0").CompatibleWith(set("v1.0.0"))
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "v1.9", res[0].Version)
	})
}
