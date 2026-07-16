/* Run: from a gno checkout:
gh pr checkout 5971 -R gnolang/gno && git checkout ec9b0de56
curl -fsSL -o gnovm/pkg/gnolang/findpaths_samestore_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5971-test-blobs-out-of-consensus/1-ec9b0de56/tests/findpaths_samestore_test.go
go test -v -run 'TestFindByPrefixSameStoreBackend' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/findpaths_samestore_test.go
*/

// Tooling builds the store as NewStore(_, base, base), so FindPathsByPrefix
// runs its two iterators over one key stream and sees every package key twice.
// Green at ec9b0de56; every store_test.go case that lists paths uses two
// distinct backends, so nothing else covers this shape.
package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
	"github.com/stretchr/testify/require"
)

func TestFindByPrefixSameStoreBackend(t *testing.T) {
	db := memdb.NewMemDB()
	one := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
	store := NewStore(nil, one, one)

	add := func(name string, files ...*std.MemFile) {
		store.AddMemPackage(&std.MemPackage{
			Type:  MPUserAll,
			Name:  name,
			Path:  "gno.land/r/demo/" + name,
			Files: files,
		}, MPUserAll)
	}
	// Production file plus test file: a prod blob and an #allbutprod blob.
	add("alpha", &std.MemFile{Name: "alpha.gno", Body: "package alpha\n"},
		&std.MemFile{Name: "alpha_test.gno", Body: "package alpha\n"})
	// Test file only: an #allbutprod blob and no prod blob.
	add("beta", &std.MemFile{Name: "beta_test.gno", Body: "package beta\n"})
	// Production file only: a prod blob and no #allbutprod blob.
	add("gamma", &std.MemFile{Name: "gamma.gno", Body: "package gamma\n"})

	var got []string
	store.FindPathsByPrefix("gno.land")(func(p string) bool {
		got = append(got, p)
		return true
	})
	require.Equal(t, []string{
		"gno.land/r/demo/alpha",
		"gno.land/r/demo/beta",
		"gno.land/r/demo/gamma",
	}, got, "each package must be listed exactly once")
}
