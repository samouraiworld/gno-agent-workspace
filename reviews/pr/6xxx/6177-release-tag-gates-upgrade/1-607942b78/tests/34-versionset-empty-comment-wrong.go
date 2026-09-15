// candidate 34: tm2/pkg/bft/version/version_test.go:41-43's doc comment for
// TestVersionSetIsComplete states an empty VersionInfo.Version makes
// CompatibleWith "compare empty majors, which matches anything — silently
// disabling the negotiation rather than failing it." CompatibleWith
// (tm2/pkg/versionset/versionset.go:84-92) does the opposite: an empty
// Version yields semver.Major(semver.MajorMinor("")) == "", which equals a
// peer's major only when the peer's Version is also empty, so an empty entry
// fails negotiation loudly against every real peer instead of passing it
// silently. The comment's rationale is backwards; the assertion it justifies
// (asserting the version is non-empty) still holds for other reasons.
//
// Run from a local clone of gnolang/gno, at
// 607942b78fa4fdf6f378fecce32bc1d1d984ab8e:
//   cp tests/34-versionset-empty-comment-wrong.go tm2/pkg/versionset/zzz_candidate34_test.go
//   go test -v -run TestCandidate34EmptyVersionFailsAgainstRealPeer ./tm2/pkg/versionset/
//   rm tm2/pkg/versionset/zzz_candidate34_test.go
package versionset

import "testing"

func TestCandidate34EmptyVersionFailsAgainstRealPeer(t *testing.T) {
	empty := VersionSet{{Name: "bft", Version: ""}}
	real := VersionSet{{Name: "bft", Version: "v1.0.0-rc.0"}}

	// IS: an empty version is rejected against a real peer's version.
	_, err := empty.CompatibleWith(real)
	if err == nil {
		t.Fatalf("expected CompatibleWith to reject the empty version against a real peer, got nil error")
	}
	t.Logf("observed: err=%v", err)
	// SHOULD (per the comment this test's sibling, TestVersionSetIsComplete,
	// carries): an empty version "matches anything" and "silently disables
	// the negotiation" — it does not; it errors, same as any other mismatched
	// major would.
}
