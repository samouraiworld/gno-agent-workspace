// candidate 6: TestVersionSetIsComplete's assert.NotEmpty(info.Version) can
// never fail — the four VersionSet.Set(...) calls in
// tm2/pkg/bft/version/version.go's init() all take package-const string
// literals that the init guard has already proved equal to Version, so an
// empty entry cannot occur. The comment justifying the assertion also states
// the opposite of what CompatibleWith does with an empty version: it treats
// an empty major as "refuses every non-empty peer", not "matches anything".
//
// Run from a local clone of gnolang/gno, at
// 607942b78fa4fdf6f378fecce32bc1d1d984ab8e:
//   cp tests/06-versionset-empty-not-permissive_test.go tm2/pkg/versionset/zzz_empty_test.go
//   go test -v -run TestEmptyVersionDoesNotMatchAnything ./tm2/pkg/versionset/
//   rm tm2/pkg/versionset/zzz_empty_test.go
package versionset

import "testing"

func TestEmptyVersionDoesNotMatchAnything(t *testing.T) {
	empty := VersionSet{{Name: "bft", Version: ""}}
	nonEmpty := VersionSet{{Name: "bft", Version: "v1.0.0-rc.0"}}

	// IS:     an empty entry is refused by a non-empty peer.
	res, err := empty.CompatibleWith(nonEmpty)
	if err == nil {
		t.Fatalf("expected CompatibleWith to reject the empty version, got res=%v, err=nil", res)
	}
	t.Logf("observed: res=%v err=%v", res, err)
	// SHOULD (per the comment this test's sibling justifies its own
	// assertion with): an empty major "matches anything" — it does not;
	// semver.Major("") == "" != semver.Major("v1.0.0-rc.0") == "v1", so the
	// pair lands in the CompatibleWith "not compatible" branch instead.
}
