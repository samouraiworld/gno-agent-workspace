// Repro, from a plain clone of github.com/gnolang/gno:
//
//	gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//	cp 01-rc-prerelease-ordering_test.go gno.land/pkg/gnoland/
//	go test ./gno.land/pkg/gnoland/ -run 'TestMeetsMinVersionRCOrdering' -count=1
//
// Asserts that a release-candidate halt floor orders by rc number, every row
// cross-checked against golang.org/x/mod/semver. At 607942b78 the two rc.2-vs-
// rc.1 rows pass and the three rc.10 rows fail: releaseVersion.compare settles
// two equal-numbered pre-releases with `v.pre < o.pre`, so "rc.10" sorts below
// "rc.2". At the merge base 1fc4c140 no v-tag parses at all and every row but
// the two refusals fails, so the false accept in "rc.2 refused by rc.10 floor"
// arrives with this diff.
package gnoland

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/semver"
)

// TestMeetsMinVersionRCOrdering pins the rc line of an upgrade gate.
//
// A halt proposal names minVersion, and checkNodeStartupParams measures every
// node's own tag against it. When the floor is a release candidate the only
// thing separating two candidates is the number after "rc.", so that number has
// to order numerically: otherwise the gate answers backwards from the tenth
// candidate of a line on, refusing the binary the upgrade was cut for and
// admitting the stale one it replaces.
func TestMeetsMinVersionRCOrdering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		binary  string
		minimum string
		want    bool // SHOULD: semver ordering, cross-checked in the body.
	}{
		// Baseline: single-digit rc numbers, green at the reviewed head. String
		// order and numeric order agree while the digit counts match, which is
		// why the table the diff ships (rc.1 vs rc.2) cannot see the defect.
		{"rc.2 meets rc.1 floor", "v1.3.0-rc.2", "v1.3.0-rc.1", true},
		{"rc.1 refused by rc.2 floor", "v1.3.0-rc.1", "v1.3.0-rc.2", false},

		// The defect: the same two questions one candidate later.
		{"rc.10 meets rc.2 floor", "v1.3.0-rc.10", "v1.3.0-rc.2", true},       // IS false at head
		{"rc.2 refused by rc.10 floor", "v1.3.0-rc.2", "v1.3.0-rc.10", false}, // IS true at head
		{"rc.10 meets rc.9 floor", "v1.3.0-rc.10", "v1.3.0-rc.9", true},       // IS false at head
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// The expectation is not the reviewer's: semver orders dot-separated
			// numeric identifiers numerically, so a binary meets the floor
			// exactly when semver ranks it at or above that floor.
			assert.Equal(t, tt.want, semver.Compare(tt.binary, tt.minimum) >= 0,
				"semver.Compare(%q, %q) disagrees with the row's want", tt.binary, tt.minimum)

			assert.Equal(t, tt.want, meetsMinVersion(tt.binary, tt.minimum),
				"meetsMinVersion(%q, %q)", tt.binary, tt.minimum)
		})
	}
}
