// Asserts that nodeParamsKeeper.WillSetParam accepts a halt_min_version string
// parseReleaseVersion cannot read, so the unusable floor is caught only at
// restart, after the chain has halted.
// Measured: at head 607942b78 both subtests pass, i.e. "chain/mainnet" and
// "1.2.0" are accepted and the release the halt was cut for is then refused.
// Subtest "legacy" passes identically at merge base 1fc4c140e, so the gap
// predates the PR; subtest "semver" fails at that base, which is the regression
// the PR fixes. Recovery is `gnoland config set skip_upgrade_height <H>`
// (UPGRADES.md), which bypasses both startup checks.
//
// Run: from a clone of gnolang/gno:
//   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78
//   curl -fsSL -o gno.land/pkg/gnoland/zz_haltfloor_test.go \
//     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/24-halt-min-version-unvalidated_test.go
//   go test -count=1 -run 'TestHaltMinVersionUnparseableAccepted' ./gno.land/pkg/gnoland/
//   rm gno.land/pkg/gnoland/zz_haltfloor_test.go

package gnoland

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/sdk/params"
	"github.com/gnolang/gno/tm2/pkg/store"
	tmver "github.com/gnolang/gno/tm2/pkg/version"
)

// newTestParamsKeeper (app_test.go) writes both halt params through
// params.ParamsKeeper, which runs the registered nodeParamsKeeper.WillSetParam
// hook, the same hook the r/sys/params halt callback goes through.
func TestHaltMinVersionUnparseableAccepted(t *testing.T) {
	// Not parallel: overrides the package-level tmver.Version.
	prev := tmver.Version
	t.Cleanup(func() { tmver.Version = prev })

	// The hook is live on this write path: a halt_height of -1 is refused by
	// WillSetParam, so an accepted halt_min_version is a decision, not a gap in
	// the harness.
	require.Panics(t, func() { newTestParamsKeeper(t, -1, "") },
		"WillSetParam must be on the ParamsKeeper write path")

	t.Run("legacy", func(t *testing.T) {
		tmver.Version = "chain/gnoland1.2" // parses at head and at the merge base

		var (
			prmk params.ParamsKeeper
			ms   store.MultiStore
		)
		// A hand-written proposal naming the chain instead of the release.
		require.NotPanics(t, func() { prmk, ms = newTestParamsKeeper(t, 100, "chain/mainnet") },
			"IS:     unparseable floor accepted, only type-checked")
		// require.Panics(t, func() { newTestParamsKeeper(t, 100, "chain/mainnet") },
		//	"SHOULD: WillSetParam rejects a floor parseReleaseVersion cannot read")

		// The chain halted at 100. The upgraded binary is refused at restart.
		err := checkNodeStartupParams(prmk, ms, 100, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not meet the minimum version")

		// Baseline, same binary: a floor that parses lets it restart.
		prmk2, ms2 := newTestParamsKeeper(t, 100, "chain/gnoland1.1")
		require.NoError(t, checkNodeStartupParams(prmk2, ms2, 100, 0))
	})

	t.Run("semver", func(t *testing.T) {
		tmver.Version = "v1.2.0" // the tag .github/workflows/release-chain-tag.yml builds

		var (
			prmk params.ParamsKeeper
			ms   store.MultiStore
		)
		// The same release tag with the "v" dropped, the shape RELEASING.md and
		// every proposal body write by hand.
		require.NotPanics(t, func() { prmk, ms = newTestParamsKeeper(t, 100, "1.2.0") },
			"IS:     floor without the v accepted, only type-checked")
		// require.Panics(t, func() { newTestParamsKeeper(t, 100, "1.2.0") },
		//	"SHOULD: WillSetParam rejects a floor parseReleaseVersion cannot read")

		err := checkNodeStartupParams(prmk, ms, 100, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not meet the minimum version")

		// Baseline, same binary: the v-shaped floor this PR taught it lets it restart.
		prmk2, ms2 := newTestParamsKeeper(t, 100, "v1.1.0")
		require.NoError(t, checkNodeStartupParams(prmk2, ms2, 100, 0))
	})
}
