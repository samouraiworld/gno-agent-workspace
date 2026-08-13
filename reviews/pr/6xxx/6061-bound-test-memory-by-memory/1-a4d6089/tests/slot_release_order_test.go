/* Run: from a gno checkout:
gh pr checkout 6061 -R gnolang/gno && git checkout a4d6089
curl -fsSL -o gno.land/pkg/integration/slot_release_order_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6061-bound-test-memory-by-memory/1-a4d6089/tests/slot_release_order_test.go
go test -v -run TestNodeSlotReleaseOrder ./gno.land/pkg/integration/
rm gno.land/pkg/integration/slot_release_order_test.go
*/

// Asserts the node budget token outlives the node it accounts for.
// SetupGnolandTestscript registers the node-stop defer during setup and
// acquireSlot registers the release defer when `gnoland start` runs;
// testscript runs defers in reverse registration order, so the release runs
// first. Fails at a4d6089: release is observed before the node stops.

package integration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeSlotReleaseOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string
	note := func(s string) func() {
		return func() {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, s)
		}
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "order.txtar"),
		[]byte("acquire\n-- placeholder --\n"), 0o600))

	// testscript runs each script as a parallel subtest, so the order is only
	// complete once they have all finished.
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"node stop", "budget release"}, order,
			"the token must be given back after the node it accounts for is stopped")
	})

	testscript.Run(t, testscript.Params{
		Dir: dir,
		Setup: func(env *testscript.Env) error {
			// Same position as SetupGnolandTestscript's node-stop defer.
			env.Defer(note("node stop"))
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			// Same position as acquireSlot's ts.Defer(nm.budget.release).
			"acquire": func(ts *testscript.TestScript, neg bool, args []string) {
				ts.Defer(note("budget release"))
			},
		},
	})
}
