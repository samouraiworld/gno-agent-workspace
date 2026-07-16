/*
Run:

	git clone https://github.com/gnolang/gno && cd gno
	git checkout 03ab3eea2
	cp <this file> contribs/gnogenesis/internal/fork/zz_fork_preprocess_fill_test.go
	cd contribs/gnogenesis && go test ./internal/fork/ -run 'PreprocessGasPerByte' -v

Regression guard for commit 0fad27f8e ("fix(gnogenesis): fill
preprocess_gas_per_byte independently of the #5415 all-zero guard"), a bug
found by reviewer ltzmaxwell and fixed in PR #5892 without a test.

The bug: the preprocess_gas_per_byte fill was nested inside the legacy
depth-param fill, which only fires when the source genesis matches an
"untuned" fingerprint. A source chain exported after #5415 but before #5892
with ANY operator-tuned depth param misses the fingerprint, so the fill was
skipped and the tool emitted preprocess_gas_per_byte: 0 — rejected by
Params.Validate(), booting only via the node's applyLegacyDefaults tolerance.

At 03ab3eea2 both tests PASS (generate.go:484 hoists the fill out of the
fingerprint loop). Re-nesting the fill inside the loop — i.e. reintroducing
the exact bug — makes TestBuildHardforkGenesisFillsPreprocessGasPerByteWhenTuned
FAIL while every upstream test in the package stays GREEN: the two existing
fingerprint tests take the match path, and the one test that takes the
fingerprint-miss path (TestBuildHardforkGenesis_PreservesTunedGasParams)
never calls Validate() nor looks at PreprocessGasPerByte.
*/
package fork

import (
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildHardforkGenesisFillsPreprocessGasPerByteWhenTuned covers the
// fingerprint-MISS path: an operator-tuned source genesis (post-#5415,
// pre-#5892) keeps its tuned depth params but must still get
// preprocess_gas_per_byte filled, so the emitted genesis is self-contained
// and passes Validate() on its own.
func TestBuildHardforkGenesisFillsPreprocessGasPerByteWhenTuned(t *testing.T) {
	t.Parallel()

	// Deviate from the legacy fingerprint by one field: this is "tuned", so
	// the depth reprice is skipped by design.
	tuned := legacyFingerprintParams()
	tuned.FixedWriteDepth100 = 450
	require.Zero(t, tuned.PreprocessGasPerByte, "source predates #5892")

	_, appState, err := buildHardforkGenesis(srcGenesisWithParams(tuned), nil, "test-13", "gnoland1", 813643)
	require.NoError(t, err)

	assert.Equal(t, vm.DefaultParams().PreprocessGasPerByte, appState.VM.Params.PreprocessGasPerByte,
		"preprocess_gas_per_byte must be filled even when the depth fingerprint does not match")
	assert.Equal(t, int64(450), appState.VM.Params.FixedWriteDepth100,
		"operator tuning must still be preserved")
	require.NoError(t, appState.VM.Params.Validate(),
		"emitted genesis must be self-contained: Validate() rejects preprocess_gas_per_byte == 0")
}

// TestBuildHardforkGenesisPreservesTunedPreprocessGasPerByte is the
// complement: an operator who explicitly set preprocess_gas_per_byte must not
// have it overwritten by the fill.
func TestBuildHardforkGenesisPreservesTunedPreprocessGasPerByte(t *testing.T) {
	t.Parallel()

	p := legacyFingerprintParams()
	p.PreprocessGasPerByte = 2_000 // operator override, != default

	_, appState, err := buildHardforkGenesis(srcGenesisWithParams(p), nil, "test-13", "gnoland1", 813643)
	require.NoError(t, err)

	assert.Equal(t, int64(2_000), appState.VM.Params.PreprocessGasPerByte,
		"an explicit preprocess_gas_per_byte must not be overwritten")
}
