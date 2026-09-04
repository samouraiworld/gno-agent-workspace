// Asserts that the 1..16 validator bound the "-- cluster --" parser enforces
// also holds for the -validators command-line override.
// Measured: at ddc5acfb9 a script asking for 17 validators is rejected at parse
// time, while the same count from -validators resolves into the cluster config
// and ClusterConfig.Validate returns nil, which is where the run goes on to set
// up 17 nodes. The baseline half passes at the reviewed head, the flag half
// fails, and both pass once the upper bound moves into ClusterConfig.Validate.
/* Run: from a gno checkout:
gh pr checkout 6135 -R gnolang/gno && git checkout ddc5acfb9
curl -fsSL -o misc/gnoe2e/internal/integration/override_bound_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6135-gnoe2e-txtar-harness/1-ddc5acfb9/tests/override_bound_test.go
go test -C misc/gnoe2e -v -run 'TestValidatorsOverrideKeepsTheSectionsBound' ./internal/integration/
rm misc/gnoe2e/internal/integration/override_bound_test.go
*/

package integration

import (
	"testing"

	"github.com/gnolang/gno/misc/gnoe2e/internal/cluster"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/stretchr/testify/require"
)

const overTheCap = maxValidators + 1

// The script route and the flag route reach the same cfg.NumValidators, so the
// bound belongs to whichever of them a caller used, not to the section alone.
func TestValidatorsOverrideKeepsTheSectionsBound(t *testing.T) {
	script := []byte("-- cluster --\nvalidators: 17\n")
	_, err := ParseClusterSpec(script)
	require.ErrorContains(t, err, "at most 16",
		"baseline: the section refuses a count past the cap")

	n := overTheCap
	spec := ClusterOverrides{Validators: &n}.Apply(ClusterSpec{Validators: 4})

	cfg := cluster.DefaultClusterConfig()
	require.NoError(t, spec.ApplyTo(&cfg, crypto.Address{}, crypto.Address{}))
	require.Equal(t, overTheCap, cfg.NumValidators)

	// IS:     the flag route resolves to 17 and validates clean, so the run
	//         goes on to start 17 gnoland processes.
	// SHOULD: the same refusal the section gives.
	require.Error(t, cfg.Validate(),
		"-validators past the cap must be refused the way the section refuses it")
}
