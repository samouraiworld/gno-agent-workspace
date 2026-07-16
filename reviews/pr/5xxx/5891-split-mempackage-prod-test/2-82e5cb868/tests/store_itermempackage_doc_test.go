/*
Run:

	gh pr checkout 5891 -R gnolang/gno && git checkout 82e5cb868
	cp store_itermempackage_doc_test.go gnovm/pkg/gnolang/
	go test ./gnovm/pkg/gnolang/ -run TestIterMemPackageNeverYieldsTestFiles -v

RED at 82e5cb868 — this test asserts the desired post-fix state, not current
behavior.

The Store interface declares (gnovm/pkg/gnolang/store.go:88-91):

	// Yields each indexed package's PROD mempackage (test/filetest files
	// live under the #allbutprod sibling and are not included), in index
	// order. A package with no production .gno files has no prod blob and
	// is skipped.
	IterMemPackage() <-chan *std.MemPackage

The implementation only reads pkg:<path> and returns it verbatim. The split
that puts test files in the sibling runs only on the mpkgtype.IsAll() branch
of AddMemPackage (store.go:1005-1023); an MP*Test / MP*Integration package
takes the else branch and is stored WHOLE under pkg:<path>, test files
included. IterMemPackage then yields those test files, contradicting the
interface doc. gnovm/pkg/test/imports.go:313 reaches this:
AddMemPackage(mpkg, gno.MPStdlibTest) when testing && preprocessOnly.

Two acceptable fixes:
  - Filter in IterMemPackage (yield MPFProd.FilterMemPackage(mpkg)) so the
    implementation matches the doc; this test then goes green, and the
    MPFProd.FilterMemPackage call at machine.go:330 becomes genuinely
    redundant and can be dropped.
  - Or narrow the interface doc to "yields whatever is stored under
    pkg:<path>, which is prod-only for MP*All packages", keep the
    machine.go:330 filter as load-bearing, and drop this test.

Either way the doc and the implementation must stop disagreeing.
*/
package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIterMemPackageNeverYieldsTestFiles(t *testing.T) {
	cases := []struct {
		name   string
		mptype MemPackageType
		path   string
		pkg    string
	}{
		// The MP*All path: split at write time, so this one already holds.
		{name: "MPUserAll", mptype: MPUserAll, path: "gno.land/r/demo/foo", pkg: "foo"},
		// The non-All path: stored whole under pkg:<path>, test files and all.
		// gnovm/pkg/test/imports.go:313 stores MPStdlibTest exactly like this.
		{name: "MPStdlibTest", mptype: MPStdlibTest, path: "math", pkg: "math"},
		{name: "MPUserTest", mptype: MPUserTest, path: "gno.land/r/demo/bar", pkg: "bar"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d1, d2 := memdb.NewMemDB(), memdb.NewMemDB()
			d1s := dbadapter.StoreConstructor(d1, storetypes.StoreOptions{})
			d2s := dbadapter.StoreConstructor(d2, storetypes.StoreOptions{})
			st := NewStore(nil, d1s, d2s)

			st.AddMemPackage(&std.MemPackage{
				Type: tc.mptype, Name: tc.pkg, Path: tc.path,
				Files: []*std.MemFile{
					{Name: tc.pkg + ".gno", Body: "package " + tc.pkg + "\n"},
					{Name: tc.pkg + "_test.gno", Body: "package " + tc.pkg + "\n"},
				},
			}, tc.mptype)

			var n int
			for got := range st.IterMemPackage() {
				n++
				require.NotNil(t, got)
				for _, mf := range got.Files {
					assert.False(t, IsTestFile(mf.Name),
						"IterMemPackage yielded test file %q in a %v package; the Store "+
							"interface doc says test/filetest files are not included",
						mf.Name, tc.mptype)
				}
			}
			require.Equal(t, 1, n)
		})
	}
}
