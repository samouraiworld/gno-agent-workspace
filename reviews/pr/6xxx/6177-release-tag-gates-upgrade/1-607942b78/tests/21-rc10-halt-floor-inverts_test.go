// Asserts that meetsMinVersion orders two "rc.N" halt floors the way semver
// does, every row cross-checked against golang.org/x/mod/semver.Compare.
// Measured at head 607942b78: the rc.1/rc.2 rows pass and both rc.10 rows fail
// — releaseVersion.compare settles equal-numbered pre-releases with
// `v.pre < o.pre`, so "rc.10" < "rc.2" as bytes. At merge base 1fc4c140 no
// v-tag parses, so every v-tag row falls back to byte equality.
//
// Repro, from a plain clone of github.com/gnolang/gno:
//
//	gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//	cp 21-rc10-halt-floor-inverts_test.go gno.land/pkg/gnoland/
//	go test -count=1 -run 'TestRC10HaltFloor' ./gno.land/pkg/gnoland/
//	rm gno.land/pkg/gnoland/21-rc10-halt-floor-inverts_test.go
package gnoland

import (
	"testing"

	"golang.org/x/mod/semver"
)

func TestRC10HaltFloor(t *testing.T) {
	t.Parallel()

	// binary, floor. want is taken from semver.Compare, not written by hand.
	rows := [][2]string{
		// Baseline: single-digit rc numbers, which the PR's own table covers.
		{"v1.3.0-rc.2", "v1.3.0-rc.1"},
		{"v1.3.0-rc.1", "v1.3.0-rc.2"},
		// The tenth release candidate, which no row covers.
		{"v1.3.0-rc.10", "v1.3.0-rc.2"},
		{"v1.3.0-rc.2", "v1.3.0-rc.10"},
		{"v1.3.0-rc.11", "v1.3.0-rc.9"},
		// A release still outranks every rc of itself.
		{"v1.3.0", "v1.3.0-rc.10"},
		{"v1.3.0-rc.10", "v1.3.0"},
	}

	for _, r := range rows {
		bin, floor := r[0], r[1]
		want := semver.Compare(bin, floor) >= 0 // SHOULD: semver is the reference
		got := meetsMinVersion(bin, floor)
		if got != want {
			t.Errorf("meetsMinVersion(%q, %q) = %v, semver says %v", bin, floor, got, want)
			continue
		}
		t.Logf("ok meetsMinVersion(%q, %q) = %v", bin, floor, got)
	}
}
