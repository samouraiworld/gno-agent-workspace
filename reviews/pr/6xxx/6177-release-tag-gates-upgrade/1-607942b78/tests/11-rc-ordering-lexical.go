// Package gnoland test: meetsMinVersion orders two "rc.N" pre-releases by plain
// string comparison, so "rc.10" sorts below "rc.2" and the ordering inverts at
// the tenth release candidate. node_params.go:250 claims the opposite ("matches
// semver for the `rc.N` shape we use"); this table checks every row against
// golang.org/x/mod/semver and fails at head 607942b78 on the two rc.10 rows.
//
// Measured: at head both rc.10 rows diverge from semver (rc.10 vs rc.2 = false,
// rc.2 vs rc.10 = true). At merge base 1fc4c140e every v-tag pair is false, the
// v shape not parsing at all, so the false-accept is new in this PR.
//
// Repro, from a plain clone of gnolang/gno:
//
//	gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//	cp <this file> gno.land/pkg/gnoland/node_params_rc_order_test.go
//	go test -run 'TestMeetsMinVersionRCOrdering|TestRCOrderingAgainstSemver' ./gno.land/pkg/gnoland/
//	rm gno.land/pkg/gnoland/node_params_rc_order_test.go
package gnoland

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/semver"
)

// rcRows are the four orderings of a pair of release candidates. want is the
// post-fix expectation: numeric ordering of the rc counter, i.e. what semver
// says. The rc.1/rc.2 pair is the baseline the PR's own table already pins.
var rcRows = []struct {
	binary  string
	minimum string
	want    bool // SHOULD: semver ordering
	isNow   bool // IS: head 607942b78
}{
	{"v1.3.0-rc.2", "v1.3.0-rc.1", true, true},   // baseline, single digits
	{"v1.3.0-rc.1", "v1.3.0-rc.2", false, false}, // baseline, single digits
	{"v1.3.0-rc.10", "v1.3.0-rc.2", true, false}, // bug: newer build refused
	{"v1.3.0-rc.2", "v1.3.0-rc.10", false, true}, // bug: stale build accepted
}

// TestMeetsMinVersionRCOrdering asserts the post-fix state: it fails at head on
// the two rc.10 rows and passes once compare() orders numeric pre-release
// identifiers numerically.
func TestMeetsMinVersionRCOrdering(t *testing.T) {
	t.Parallel()

	for _, tt := range rcRows {
		got := meetsMinVersion(tt.binary, tt.minimum)
		assert.Equal(t, tt.want, got, // SHOULD: semver ordering
			// assert.Equal(t, tt.isNow, got,   // IS: head 607942b78, lexical
			"meetsMinVersion(%q, %q)", tt.binary, tt.minimum)
	}
}

// TestRCOrderingAgainstSemver derives the same expectation from the reference
// implementation rather than from hand-written rows, so the table cannot be
// wrong about what semver says.
func TestRCOrderingAgainstSemver(t *testing.T) {
	t.Parallel()

	for _, tt := range rcRows {
		wantBySemver := semver.Compare(tt.binary, tt.minimum) >= 0
		assert.Equal(t, tt.want, wantBySemver,
			"semver.Compare(%q, %q) = %d", tt.binary, tt.minimum, semver.Compare(tt.binary, tt.minimum))
		t.Logf("%-14s vs %-14s  semver=%v  meetsMinVersion=%v",
			tt.binary, tt.minimum, wantBySemver, meetsMinVersion(tt.binary, tt.minimum))
	}
}
