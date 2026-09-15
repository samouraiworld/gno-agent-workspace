package gnoland

// Asserts that a halt_min_version in the chain/gnoland shape rejects a signed or
// leading-zero component, the guards the v branch added six lines below.
// Measured at 607942b78 and at merge base 1fc4c140, meetsMinVersion(bin, floor)
// with floor "chain/gnoland-1.0" (major parses as -1, so every parsed binary
// clears it); every SHOULD row below fails at the reviewed head.
//
//	binary              head   base   verdict
//	chain/gnoland1.0    true   true   stale betanet binary clears the floor, both trees
//	chain/gnoland1.1    true   true   stale betanet binary clears the floor, both trees
//	v1.2.0              true   false  head only: the v branch made the binary parse
//	gnoland1.1          false  false  base release workflow stripped chain/, never parsed
//	develop             false  false  unparsed binary clears no floor, both trees
//
/* Run: from a gno checkout:
gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
curl -fsSL -o gno.land/pkg/gnoland/zz_legacy_floor_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/46-legacy-floor-coercion-predates-head_test.go
go test -count=1 -run 'TestLegacyFloorCoercion' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/zz_legacy_floor_test.go

For the base column, from the same checkout:
git checkout 1fc4c140ec4064e8d66cb9828ddea280a723cca6 -- gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/node_params_version_test.go
go test -count=1 -run 'TestLegacyFloorCoercion/betanet' ./gno.land/pkg/gnoland/
git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e -- gno.land/pkg/gnoland/
*/

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// typo is what a governance halt proposal would carry for the frozen betanet tag
// "chain/gnoland1.0" with one stray character. strconv.Atoi takes the sign, so
// the floor becomes {major: -1} and orders below every real release.
const typo = "chain/gnoland-1.0"

func TestLegacyFloorCoercion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		binary  string
		want    bool
		alsoAtB bool // the row fails at the merge base too
	}{
		// SHOULD: a floor that does not parse falls back to byte equality, so
		// nothing clears it and the halt holds until governance fixes the string.
		{"betanet 1.0", "chain/gnoland1.0", false, true},
		{"betanet 1.1", "chain/gnoland1.1", false, true},
		{"v-line release", "v1.2.0", false, false},
		// IS at 607942b78: all three are true, so the floor excludes nothing.

		// Baseline, unchanged in both trees: an unparsed binary clears no floor.
		{"ad-hoc build", "develop", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, meetsMinVersion(tt.binary, typo),
				"meetsMinVersion(%q, %q); pre-existing at the merge base: %v", tt.binary, typo, tt.alsoAtB)
		})
	}
}
