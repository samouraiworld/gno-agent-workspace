/* Run: from a gno checkout:
gh pr checkout 5937 -R gnolang/gno && git checkout b79972d22
curl -fsSL -o tm2/pkg/store/rootmulti/zz_immutwrite_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5937-bptree-clean-tree-fast-index/1-b79972d22/tests/immutable_query_open_writes_test.go
go test -v -run TestImmutableQueryOpenWrites ./tm2/pkg/store/rootmulti/
rm tm2/pkg/store/rootmulti/zz_immutwrite_test.go
*/

// rootmulti.constructStore prefers params.db over the ImmutableDB wrapper that
// MultiImmutableCacheWrapWithVersion installs, so a store mounted with an
// explicit db (as gno.land/pkg/gnoland/app.go mounts mainKey) hands the query
// view a writable database and ensureFastIndex can rebuild onto live state.
// At b79972d22 this fails: the query-height open restores the stamp and
// overwrites the doctored entry. It passes once the immutable open is read-only.

package rootmulti

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	storebptree "github.com/gnolang/gno/tm2/pkg/store/bptree"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestImmutableQueryOpenWrites(t *testing.T) {
	db := memdb.NewMemDB()
	key := types.NewStoreKey("main")
	ms := NewMultiStore(db)
	ms.MountStoreWithDB(key, storebptree.FastStoreConstructor, db)
	if err := ms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	st := ms.GetCommitStore(key)
	st.Set(nil, []byte("k"), []byte("v1"))
	cid := ms.Commit()

	// Locate the fast-index stamp and the 'F' entry under the store prefix.
	var stampKey, fKey []byte
	itr, _ := db.Iterator(nil, nil)
	for ; itr.Valid(); itr.Next() {
		k := string(itr.Key())
		if len(k) > 7 && k[len(k)-7:] == "fastidx" {
			stampKey = append([]byte{}, itr.Key()...)
		}
		if len(k) >= 5 && k[:4] == "s/_/" && k[4] == 'F' {
			fKey = append([]byte{}, itr.Key()...)
		}
	}
	itr.Close()
	if stampKey == nil || fKey == nil {
		t.Fatalf("setup: stampKey=%v fKey=%v", stampKey, fKey)
	}

	// The documented operator escape hatch for a bad stamp: delete it by hand.
	if err := db.Delete(stampKey); err != nil {
		t.Fatal(err)
	}
	// Doctor the surviving entry so any rebuild is observable.
	if err := db.Set(fKey, []byte("junk")); err != nil {
		t.Fatal(err)
	}

	if _, err := ms.MultiImmutableCacheWrapWithVersion(cid.Version); err != nil {
		t.Fatal(err)
	}

	stampAfter, _ := db.Get(stampKey)
	fAfter, _ := db.Get(fKey)
	if stampAfter != nil {
		t.Errorf("query-height open re-wrote the fast-index stamp; the immutable open must not write")
	}
	if string(fAfter) != "junk" {
		t.Errorf("query-height open rebuilt the 'F' entry (%q); the immutable open must not write", fAfter)
	}
}
