package gnoland

// Asserts that the "chain/gnoland" branch of parseReleaseVersion rejects the
// same malformed components the "v" branch rejects: a signed component, a
// zero-padded component. Measured at 607942b78: "chain/gnoland-1.0" parses as
// major -1 with ok=true, so meetsMinVersion("v1.2.0", "chain/gnoland-1.0") is
// true and the governance floor accepts every parseable binary. Fails at head.
//
/* Run: from a gnolang/gno clone:
gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78
curl -fsSL -o gno.land/pkg/gnoland/legacy_tag_guard_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/22-legacy-tag-unguarded-atoi_test.go
go test -v -run 'TestLegacyTagUnguardedAtoi' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/legacy_tag_guard_test.go
*/

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLegacyTagUnguardedAtoi(t *testing.T) {
	t.Parallel()

	// Baseline: the v branch guards these shapes at node_params.go:302-310, and
	// the two real betanet tags still parse. Green at head, green after a fix.
	t.Run("baseline_v_branch_guarded", func(t *testing.T) {
		t.Parallel()

		for _, in := range []string{"v-1.2.0", "v+1.2.0", "v1.-2.0", "v1.02.0", "v01.2.0"} {
			_, ok := parseReleaseVersion(in)
			assert.False(t, ok, "v branch must reject %q", in)
		}
		for in, want := range map[string]releaseVersion{
			"chain/gnoland1.0": {major: 1},
			"chain/gnoland1.1": {major: 1, minor: 1},
		} {
			got, ok := parseReleaseVersion(in)
			assert.True(t, ok, "the two frozen betanet tags must still parse: %q", in)
			assert.Equal(t, want, got, "parseReleaseVersion(%q)", in)
		}
		// A floor no binary can clear must refuse, not accept.
		assert.False(t, meetsMinVersion("v1.2.0", "v-1.0.0"),
			`meetsMinVersion("v1.2.0", "v-1.0.0")`)
	})

	// The defect: the legacy branch feeds strconv.Atoi with no guard.
	t.Run("legacy_branch_must_guard_the_same_shapes", func(t *testing.T) {
		t.Parallel()

		for _, in := range []string{
			"chain/gnoland-1.0", // a typo for chain/gnoland1.0; Atoi("-1") succeeds
			"chain/gnoland+1.0", // Atoi("+1") succeeds
			"chain/gnoland1.-0",
			"chain/gnoland1.01", // a tag that was never pushed
			"chain/gnoland01.0",
		} {
			_, ok := parseReleaseVersion(in)
			assert.False(t, ok, "legacy branch must reject %q, which was never a tag", in)
		}
	})

	// What the unguarded parse costs: a typo'd governance floor stops gating.
	t.Run("negative_major_floor_must_not_accept_everything", func(t *testing.T) {
		t.Parallel()

		// SHOULD: an unparseable floor falls back to byte equality, so no
		// binary clears it and the halt gate stays closed.
		assert.False(t, meetsMinVersion("v1.2.0", "chain/gnoland-1.0"),
			`meetsMinVersion("v1.2.0", "chain/gnoland-1.0")`)
		// IS at 607942b78: true. Uncomment the line above's inverse to see it.
		// assert.True(t, meetsMinVersion("v1.2.0", "chain/gnoland-1.0"))

		assert.False(t, meetsMinVersion("chain/gnoland1.0", "chain/gnoland-1.0"),
			`meetsMinVersion("chain/gnoland1.0", "chain/gnoland-1.0")`)
	})
}
