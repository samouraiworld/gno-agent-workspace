/* Run: from a gno checkout:
gh pr checkout 5861 -R gnolang/gno
curl -fsSL -o tm2/pkg/p2p/discovery/zz_lastseen_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5861-tm2-address-book/1-0170d49a2/tests/store_lastseen_persist_test.go
go test -v -run 'TestStore_ReAdd_PersistsLastSeen' ./tm2/pkg/p2p/discovery/
rm tm2/pkg/p2p/discovery/zz_lastseen_test.go
*/

// AddPeers only sets s.dirty when a brand-new key appears, so re-adding a peer
// updates LastSeen in memory but never reaches disk; Save is a no-op.
// At the reviewed head this fails: persisted LastSeen stays frozen at first-seen.
// After the fix (mark dirty on LastSeen refresh) both persisted values differ.
package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/p2p/types"
	"github.com/stretchr/testify/require"
)

func TestStore_ReAdd_PersistsLastSeen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "addrbook.json")

	s, err := NewStore(path, types.NetAddress{})
	require.NoError(t, err)

	addr := generateTestAddress(t, "1.2.3.4", 26656)

	s.AddPeers(addr)
	require.NoError(t, s.Save())
	first := persistedLastSeen(t, path, addr.String())

	// Advance past one-second Unix granularity, then re-discover the same peer.
	time.Sleep(1100 * time.Millisecond)
	s.AddPeers(addr)
	require.NoError(t, s.Save())
	second := persistedLastSeen(t, path, addr.String())

	require.Greater(t, second, first,
		"re-discovering a peer must refresh its persisted LastSeen, else eviction ranks it stale after restart")
}

func persistedLastSeen(t *testing.T, path, addr string) int64 {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var raw storeJSON
	require.NoError(t, json.Unmarshal(data, &raw))

	for _, e := range raw.Peers {
		if e.Addr == addr {
			return e.LastSeen
		}
	}

	t.Fatalf("address %s not found in persisted store", addr)
	return 0
}
