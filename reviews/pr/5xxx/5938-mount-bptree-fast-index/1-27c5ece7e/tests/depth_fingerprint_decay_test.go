/* Run:

gh pr checkout 5938 -R gnolang/gno && git checkout 27c5ece7e

curl -sSL -o contribs/gnogenesis/internal/fork/depth_fingerprint_decay_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5938-mount-bptree-fast-index/1-27c5ece7e/tests/depth_fingerprint_decay_test.go

(cd contribs/gnogenesis && go test ./internal/fork/ -run TestDepthDefaults -v)

rm contribs/gnogenesis/internal/fork/depth_fingerprint_decay_test.go

Mechanism: pins vm.DefaultParams()'s seven depth/iteration fields as an
append-only history, then requires every superseded entry to be present in
generate.go's untunedDepthFingerprints — the rule params.go:38-39 and
generate.go:678-679 state only in prose.
Observed at 27c5ece7e: GREEN (the rule holds today; nothing enforces it).
When enforced: editing a depth default without appending its predecessor to
untunedDepthFingerprints turns this RED instead of silently shipping a fork
tool that no longer recognizes the previous era.
*/

package fork

import (
	"fmt"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// depthDefaultsHistory is every depth/iteration gas default set vm.DefaultParams()
// has ever returned, OLDEST FIRST. Append-only. The LAST entry must always equal
// the live defaults; every earlier entry must be an untunedDepthFingerprints era.
//
// Adding an entry is the only sanctioned way to change the vm depth defaults:
// the two assertions below make the append to untunedDepthFingerprints mandatory
// rather than a comment someone is trusted to have read.
var depthDefaultsHistory = []vm.Params{
	// Era 0 — pre-#5415 (gas-storage refactor): the fields did not exist and
	// deserialize to zero.
	{},
	// Era 1 — post-#5415, pre-bptree-mount: IAVL-era untuned defaults.
	{
		MinGetReadDepth100:   300,
		MinSetReadDepth100:   200,
		MinWriteDepth100:     440,
		FixedGetReadDepth100: 300,
		FixedSetReadDepth100: 200,
		FixedWriteDepth100:   440,
		IterNextCostFlat:     1_000,
	},
	// Era 2 — bptree mount + fast index (PR 5938). CURRENT.
	{
		MinGetReadDepth100:   100,
		MinSetReadDepth100:   200,
		MinWriteDepth100:     540,
		FixedGetReadDepth100: 100,
		FixedSetReadDepth100: 200,
		FixedWriteDepth100:   540,
		IterNextCostFlat:     1_000,
	},
}

// TestDepthDefaultsHistoryTracksLiveDefaults is step 1 of the ratchet: the last
// history entry must be what vm.DefaultParams() actually returns. Changing a
// depth default in gno.land/pkg/sdk/vm/params.go reddens this until the new set
// is APPENDED here (never edited in place).
func TestDepthDefaultsHistoryTracksLiveDefaults(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, depthDefaultsHistory)
	current := depthDefaultsHistory[len(depthDefaultsHistory)-1]

	assert.True(t, depthParamsMatch(vm.DefaultParams(), current),
		"vm.DefaultParams() depth/iteration fields no longer match the last entry of "+
			"depthDefaultsHistory.\n"+
			"  live:   %s\n  pinned: %s\n"+
			"If the change is intentional: APPEND the new default set as a new era at the "+
			"end of depthDefaultsHistory (do not edit the existing last entry), then satisfy "+
			"TestDepthDefaultsSupersededAreFingerprinted, which will then require the "+
			"now-superseded set in untunedDepthFingerprints (generate.go).",
		fmtDepth(vm.DefaultParams()), fmtDepth(current))
}

// TestDepthDefaultsSupersededAreFingerprinted is step 2 of the ratchet, and the
// assertion that actually enforces params.go:38-39. Every default set that is no
// longer current shipped on some chain; a genesis exported from such a chain
// carries it verbatim, so buildHardforkGenesis must recognize it as untuned and
// reprice it. That recognition is exactly untunedDepthFingerprints membership.
func TestDepthDefaultsSupersededAreFingerprinted(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, depthDefaultsHistory)
	superseded := depthDefaultsHistory[:len(depthDefaultsHistory)-1]

	for i, era := range superseded {
		found := false
		for _, fp := range untunedDepthFingerprints {
			if depthParamsMatch(era, fp) {
				found = true
				break
			}
		}
		assert.Truef(t, found,
			"era %d of depthDefaultsHistory (%s) is a superseded vm default set but is NOT in "+
				"untunedDepthFingerprints (contribs/gnogenesis/internal/fork/generate.go).\n"+
				"A genesis exported from a chain running those defaults would fork under the "+
				"CURRENT store/pricing without being repriced. Append it as a new era fingerprint.",
			i, fmtDepth(era))
	}
}

// TestDepthDefaultsCurrentIsNotFingerprinted keeps the reprice honest: if the
// live defaults were themselves a fingerprint, the rewrite would match a genesis
// that is already correctly priced and the fingerprint list would stop being a
// record of superseded eras.
func TestDepthDefaultsCurrentIsNotFingerprinted(t *testing.T) {
	t.Parallel()

	for i, fp := range untunedDepthFingerprints {
		assert.Falsef(t, depthParamsMatch(vm.DefaultParams(), fp),
			"untunedDepthFingerprints[%d] equals the CURRENT vm defaults (%s); a fingerprint "+
				"records a SUPERSEDED era only.", i, fmtDepth(fp))
	}
}

func fmtDepth(p vm.Params) string {
	return fmt.Sprintf("min{get:%d set:%d write:%d} fixed{get:%d set:%d write:%d} iterNextCostFlat:%d",
		p.MinGetReadDepth100, p.MinSetReadDepth100, p.MinWriteDepth100,
		p.FixedGetReadDepth100, p.FixedSetReadDepth100, p.FixedWriteDepth100,
		p.IterNextCostFlat)
}
