// Repro, from a plain clone of github.com/gnolang/gno:
//
//	gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//	cp 05-startup-gate-release-tag_test.go gno.land/pkg/gnoland/
//	go test ./gno.land/pkg/gnoland/ -run 'TestCheckNodeStartupParamsReleaseTag' -count=1
//
// Asserts that checkNodeStartupParams, the caller of the gate, answers a
// vMAJOR.MINOR.PATCH floor: it readmits an upgraded binary after the halt and
// refuses that same binary before it. All four rows pass at 607942b78; at the
// merge base 1fc4c140, where no v-tag parses, those two rows fail and the two
// that byte equality already settled pass. The package's own caller test,
// TestCheckNodeStartupParams in app_test.go, passes unchanged at both.
package gnoland

import (
	"testing"

	tmver "github.com/gnolang/gno/tm2/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckNodeStartupParamsReleaseTag runs the startup gate with a release tag
// compiled into the binary, the state every published gnoland binary is in.
//
// Under `go test` tmver.Version is "develop", which parses as no release at all,
// so a caller test that leaves it alone reaches only the `binaryVersion ==
// minVersion` fallback. Setting it is what puts the release-tag comparison under
// the caller: check 1 (an old binary must not resume after a halt) and check 2
// (an upgraded binary must not run before one) both turn on it.
func TestCheckNodeStartupParamsReleaseTag(t *testing.T) {
	// tmver.Version is a package-level var, so these rows are sequential:
	// no t.Parallel here or in the subtests.
	orig := tmver.Version
	t.Cleanup(func() { tmver.Version = orig })

	const haltHeight = 100

	tests := []struct {
		name        string
		binary      string // the tag linked into the running binary
		minimum     string // the floor a halt proposal named
		lastBlock   int64
		wantErrPart string // "" means the node is allowed to start
	}{
		{
			// Baseline, green at the merge base too: an exact-equal floor is
			// settled by byte equality without any parser.
			name:      "exact floor after halt",
			binary:    "v1.2.0",
			minimum:   "v1.2.0",
			lastBlock: haltHeight,
		},
		{
			// The upgrade the PR exists for: a newer binary clears an older
			// floor and the chain restarts.
			name:      "upgraded binary readmitted after halt",
			binary:    "v1.2.0",
			minimum:   "v1.1.0",
			lastBlock: haltHeight,
		},
		{
			// Check 2: the same binary must be refused until the halt lands,
			// or half the validators upgrade early and fork.
			name:        "upgraded binary refused before halt",
			binary:      "v1.2.0",
			minimum:     "v1.1.0",
			lastBlock:   haltHeight - 50,
			wantErrPart: "upgrade intended for halt height",
		},
		{
			// Check 1: the stale binary must not resume after the halt.
			name:        "stale binary refused after halt",
			binary:      "v1.0.0",
			minimum:     "v1.2.0",
			lastBlock:   haltHeight,
			wantErrPart: "does not meet the minimum version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmver.Version = tt.binary
			prmk, ms := newTestParamsKeeper(t, haltHeight, tt.minimum)

			err := checkNodeStartupParams(prmk, ms, tt.lastBlock, 0)

			if tt.wantErrPart == "" {
				require.NoError(t, err, "binary %q against floor %q at height %d",
					tt.binary, tt.minimum, tt.lastBlock)
				return
			}
			require.Error(t, err, "binary %q against floor %q at height %d",
				tt.binary, tt.minimum, tt.lastBlock)
			assert.Contains(t, err.Error(), tt.wantErrPart)
		})
	}
}
