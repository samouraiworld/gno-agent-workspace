package gnoland

// Asserts that the chain/gnoland branch of parseReleaseVersion rejects a signed
// and a leading-zero component, the two shapes the v branch added guards for.
// Measured: at 607942b78 "chain/gnoland-1.0" parses as {major:-1} and
// "chain/gnoland1.01" as {minor:1}; both rows below fail at the reviewed head.
//
/* Run: from a gno checkout:
gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
curl -fsSL -o gno.land/pkg/gnoland/zz_legacy_tag_guards_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/04-legacy-tag-guards_test.go
go test -count=1 -run 'TestParseReleaseVersionLegacyGuards|TestMeetsMinVersionLegacyTypo' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/zz_legacy_tag_guards_test.go
*/

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The two rows TestParseReleaseVersion is missing: its only legacy negative is
// "chain/gnolandX.Y", the non-numeric case Atoi already rejects.
func TestParseReleaseVersionLegacyGuards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want releaseVersion
		ok   bool
	}{
		// Baseline: the two frozen betanet tags still parse.
		{"legacy betanet .0", "chain/gnoland1.0", releaseVersion{major: 1}, true},
		{"legacy betanet .1", "chain/gnoland1.1", releaseVersion{major: 1, minor: 1}, true},

		// SHOULD: the legacy branch rejects what the v branch rejects.
		{"legacy signed major", "chain/gnoland-1.0", releaseVersion{}, false},
		{"legacy signed minor", "chain/gnoland1.-0", releaseVersion{}, false},
		{"legacy plus minor", "chain/gnoland1.+1", releaseVersion{}, false},
		{"legacy leading zero", "chain/gnoland1.01", releaseVersion{}, false},
		// IS at 607942b78, for the four rows above:
		// {"legacy signed major", "chain/gnoland-1.0", releaseVersion{major: -1}, true},
		// {"legacy signed minor", "chain/gnoland1.-0", releaseVersion{major: 1}, true},
		// {"legacy plus minor",   "chain/gnoland1.+1", releaseVersion{major: 1, minor: 1}, true},
		// {"legacy leading zero", "chain/gnoland1.01", releaseVersion{major: 1, minor: 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseReleaseVersion(tt.in)
			assert.Equal(t, tt.ok, ok, "parse ok for %q (got %+v)", tt.in, got)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// What the parse hole costs at the halt gate: a governance halt_min_version
// carrying a sign parses as a floor of {-1,0,0}, which every release tag clears.
func TestMeetsMinVersionLegacyTypo(t *testing.T) {
	t.Parallel()

	const typo = "chain/gnoland-1.0" // intended "chain/gnoland1.0"

	tests := []struct {
		name    string
		binary  string
		minimum string
		want    bool
	}{
		// SHOULD: an unparseable floor falls back to byte equality, so no
		// released binary clears it and the halt holds.
		{"betanet binary against signed floor", "chain/gnoland1.0", typo, false},
		{"v-line binary against signed floor", "v1.2.0", typo, false},
		{"v0.0.0 against signed floor", "v0.0.0", typo, false},
		// IS at 607942b78: all three are true — the floor is a no-op.

		// Unchanged either way: a binary that does not parse never clears a floor.
		{"develop against signed floor", "develop", typo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, meetsMinVersion(tt.binary, tt.minimum),
				"meetsMinVersion(%q, %q)", tt.binary, tt.minimum)
		})
	}
}
