// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5605 -R gnolang/gno && git checkout 58f637cc3
curl -fsSL -o gnovm/pkg/gnolang/addmempkg_write_order_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5605-gnovm-store-addmempkg/3-58f637cc3/tests/addmempkg_write_order_test.go
go test -v -run 'TestAdv_AddMemPackage_RecordedWriteOrder' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/addmempkg_write_order_test.go
*/

// Wraps the base+iavl substores with a recorder that captures every Set in
// order, then asserts body (iavl pkg:) → index slot (base pkgidx:N) → counter
// (base pkgidx:counter). The shipped TestAddMemPackage_WriteOrderIsBodyFirst
// checks only the final post-state, so a regression back to counter→index→body
// passes it; this test fails on that regression with "REGRESSION: body ... must
// be written before index ...".

package gnolang

import (
	"strings"
	"sync"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type writeOp struct {
	store string // "base" or "iavl"
	key   string
}

type recordedStore struct {
	storetypes.Store
	name string
	rec  *[]writeOp
	mu   *sync.Mutex
}

func (r recordedStore) Set(gctx *storetypes.GasContext, key, value []byte) {
	r.mu.Lock()
	*r.rec = append(*r.rec, writeOp{store: r.name, key: string(key)})
	r.mu.Unlock()
	r.Store.Set(gctx, key, value)
}

func TestAdv_AddMemPackage_RecordedWriteOrder(t *testing.T) {
	d1, d2 := memdb.NewMemDB(), memdb.NewMemDB()
	baseInner := dbadapter.StoreConstructor(d1, storetypes.StoreOptions{})
	iavlInner := dbadapter.StoreConstructor(d2, storetypes.StoreOptions{})

	var ops []writeOp
	var mu sync.Mutex
	baseStore := recordedStore{Store: baseInner, name: "base", rec: &ops, mu: &mu}
	iavlStore := recordedStore{Store: iavlInner, name: "iavl", rec: &ops, mu: &mu}

	store := NewStore(nil, baseStore, iavlStore)

	store.AddMemPackage(&std.MemPackage{
		Type:  MPStdlibAll,
		Name:  "ord",
		Path:  "ord",
		Files: []*std.MemFile{{Name: "ord.gno", Body: "package ord"}},
	}, MPStdlibAll)

	// Find the three load-bearing writes.
	var (
		bodyIdx, indexIdx, counterIdx = -1, -1, -1
	)
	for i, op := range ops {
		switch {
		case op.store == "iavl" && strings.HasPrefix(op.key, "pkg:"):
			bodyIdx = i
		case op.store == "base" && strings.HasPrefix(op.key, "pkgidx:") &&
			op.key != "pkgidx:counter":
			indexIdx = i
		case op.store == "base" && op.key == "pkgidx:counter":
			counterIdx = i
		}
	}
	require.NotEqual(t, -1, bodyIdx, "body write not observed; ops=%v", ops)
	require.NotEqual(t, -1, indexIdx, "index write not observed; ops=%v", ops)
	require.NotEqual(t, -1, counterIdx, "counter write not observed; ops=%v", ops)

	assert.Less(t, bodyIdx, indexIdx,
		"REGRESSION: body (iavl pkg:) must be written before index (base pkgidx:N) — ops=%v", ops)
	assert.Less(t, indexIdx, counterIdx,
		"REGRESSION: index (base pkgidx:N) must be written before counter bump (base pkgidx:counter) — ops=%v", ops)
}
