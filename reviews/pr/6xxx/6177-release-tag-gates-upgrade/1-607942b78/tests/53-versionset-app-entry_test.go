// Candidate 53 is REFUTED, and this is the run that refuted it. The claim was
// that the "app" versionset entry is covered by no test. It is:
// TestNodeSetAppVersion, tm2/pkg/bft/node/node_test.go:353, boots a node and
// asserts the entry is present and equals kvstore.AppVersion ("v0.0.0").
// What is left, and is pre-existing and identical at the merge base 1fc4c140e:
// gno.land sets its app version to "dev", which semver.MajorMinor reduces to "",
// so two gno.land peers negotiate that entry on an empty major. The mechanism
// subtest passes at 607942b78; the coverage subtest fails there and at the base.
//
// Run: from a gnolang/gno clone:
//   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//   curl -fsSL -o tm2/pkg/bft/version/appentry_test.go \
//     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/53-versionset-app-entry_test.go
//   go test -v -count=1 -run 'TestVersionSetAppEntry' ./tm2/pkg/bft/version/
//   go test -v -count=1 -run 'TestNodeSetAppVersion' ./tm2/pkg/bft/node/   # the existing guard
//   rm tm2/pkg/bft/version/appentry_test.go

package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"

	"github.com/gnolang/gno/tm2/pkg/bft/state"
	"github.com/gnolang/gno/tm2/pkg/bft/version"
	verset "github.com/gnolang/gno/tm2/pkg/versionset"
)

// What gno.land sets, gno.land/pkg/gnoland/app.go:107: baseApp.SetAppVersion("dev").
// The ABCI handshake copies it onto state.AppVersion, tm2/pkg/bft/consensus/replay.go:258.
const gnolandAppVersion = "dev"

// advertised rebuilds the versionset a node puts on the wire: the three lines of
// makeNodeInfo, tm2/pkg/bft/node/node.go:1004-1007, which copy the package set
// and add the runtime "app" entry on top of it.
func advertised(appVersion string) verset.VersionSet {
	vset := version.VersionSet
	vset.Set(verset.VersionInfo{Name: "app", Version: appVersion})
	return vset
}

func TestVersionSetAppEntry(t *testing.T) {
	t.Parallel()

	// Mechanism. Two peers whose "app" versions are not semver both reduce to an
	// empty major, so CompatibleWith pairs them off as agreeing. This is the
	// failure mode TestVersionSetIsComplete's header names, reached through the
	// one entry that test cannot see.
	t.Run("empty major negotiates vacuously", func(t *testing.T) {
		t.Parallel()

		var pre state.State // AppVersion "" until the handshake, state.go:248

		assert.Empty(t, semver.MajorMinor(pre.AppVersion),
			"pre-handshake app version has no major to negotiate on")
		assert.Empty(t, semver.MajorMinor(gnolandAppVersion),
			"gno.land's %q app version has no major to negotiate on", gnolandAppVersion)

		// Two nodes disagreeing on the app entry are still judged compatible.
		res, err := advertised("dev").CompatibleWith(advertised("something-else"))
		require.NoError(t, err, "IS: mismatched non-semver app versions negotiate")
		app, ok := res.Get("app")
		require.True(t, ok)
		assert.Empty(t, app.Version, "IS: the negotiated app version is empty")

		// SHOULD: a mismatch on a required entry is an error, as it is for the
		// four constant entries. Uncomment once the app entry carries semver.
		// require.Error(t, err, "SHOULD: mismatched app versions are incompatible")
	})

	// Coverage. The test the diff should have written: assert the property over
	// the set a node advertises, not over the four compile-time constants this
	// package registers at init. Fails at 607942b78 on the "app" entry.
	t.Run("every advertised entry carries a semver version", func(t *testing.T) {
		t.Parallel()

		wire := advertised(gnolandAppVersion)
		require.Len(t, wire, 5, "the wire set is bft, abci, blockchain, p2p and app")

		for _, name := range []string{"bft", "abci", "blockchain", "p2p", "app"} {
			info, ok := wire.Get(name)
			require.True(t, ok, "advertised VersionSet is missing the %q entry", name)
			assert.NotEmpty(t, semver.MajorMinor(info.Version),
				"advertised entry %q carries %q, which semver cannot read: "+
					"CompatibleWith negotiates it on an empty major", name, info.Version)
		}
	})
}
