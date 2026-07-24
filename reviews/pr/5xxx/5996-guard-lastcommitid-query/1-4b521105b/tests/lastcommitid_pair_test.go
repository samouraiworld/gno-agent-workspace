/* Run: from a gno checkout:
gh pr checkout 5996 -R gnolang/gno && git checkout 4b521105b
curl -fsSL -o tm2/pkg/store/rootmulti/lastcommitid_pair_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5996-guard-lastcommitid-query/1-4b521105b/tests/lastcommitid_pair_test.go
go test -count=1 -run 'TestLastCommitIDPairNotTorn' ./tm2/pkg/store/rootmulti/
rm tm2/pkg/store/rootmulti/lastcommitid_pair_test.go
*/

// Readers validate every observed {Version, Hash} against the table of pairs the
// committer actually produced, so a mixed pair fails the assertion directly
// instead of only tripping the race detector.
// Passes at 4b521105b. Without the two mutex operations in LastCommitID and
// setLastCommitID it fails on every run with no -race, naming the version whose
// hash never belonged to it.

package rootmulti

import (
	"bytes"
	"sync"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestLastCommitIDPairNotTorn(t *testing.T) {
	t.Parallel()

	const commits = 400

	// The committer's version to hash table, computed off the hot path.
	want := make(map[int64][]byte, commits)
	{
		ref := newMultiStoreWithMounts(memdb.NewMemDB())
		if err := ref.LoadLatestVersion(); err != nil {
			t.Fatal(err)
		}
		s := ref.getStoreByName("store1")
		for i := range commits {
			s.Set(nil, []byte{byte(i), byte(i >> 8)}, []byte("v"))
			id := ref.Commit()
			want[id.Version] = id.Hash
		}
	}

	ms := newMultiStoreWithMounts(memdb.NewMemDB())
	if err := ms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	// One slot per reader, so no reader has to touch *testing.T off the test
	// goroutine.
	bad := make([]types.CommitID, 8)

	for r := range 8 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				id := ms.LastCommitID()
				if id.Version == 0 {
					continue
				}
				if h, ok := want[id.Version]; !ok || !bytes.Equal(h, id.Hash) {
					bad[r] = id
					return
				}
			}
		})
	}

	store1 := ms.getStoreByName("store1")
	for i := range commits {
		store1.Set(nil, []byte{byte(i), byte(i >> 8)}, []byte("v"))
		ms.Commit()
	}
	close(stop)
	wg.Wait()

	for r, id := range bad {
		if id.Version != 0 {
			t.Fatalf("reader %d observed version %d paired with a hash the committer never produced at that version", r, id.Version)
		}
	}
}
