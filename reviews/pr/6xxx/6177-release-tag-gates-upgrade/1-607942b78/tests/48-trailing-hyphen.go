// Run: from a gno checkout:
//   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//   cp 48-trailing-hyphen.go gno.land/pkg/gnoland/zz_trailing_hyphen_test.go
//   go test -v -run 'TestTrailingHyphenProbe' ./gno.land/pkg/gnoland/
//   rm gno.land/pkg/gnoland/zz_trailing_hyphen_test.go
//
// Candidate #48: gno.land/pkg/gnoland/node_params.go:296 (strings.Cut(rest, "-")
// on the v-branch). "v1.2.0-" splits into pre="" the same as "v1.2.0" does, and
// nothing rejects an empty-but-present pre-release separator, so the two tag
// strings compare equal.
package gnoland

import "testing"

func TestTrailingHyphenProbe(t *testing.T) {
	rv, ok := parseReleaseVersion("v1.2.0-")
	t.Logf("parseReleaseVersion(v1.2.0-) = %+v, ok=%v", rv, ok)
	// IS: a trailing hyphen with no pre-release text parses as the final release.
	if ok && rv.pre == "" {
		t.Errorf("BUG: %q parsed as final release {%+v}, indistinguishable from %q", "v1.2.0-", rv, "v1.2.0")
	}
	// SHOULD (commented, uncomment once fixed): reject an empty pre-release identifier.
	// if ok {
	// 	t.Errorf("SHOULD reject: %q must not parse, got {%+v}", "v1.2.0-", rv)
	// }

	if got := meetsMinVersion("v1.2.0-", "v1.2.0"); got {
		t.Errorf("BUG: meetsMinVersion(%q, %q) = true, a copy-pasted trailing hyphen satisfies the real floor", "v1.2.0-", "v1.2.0")
	}
}
