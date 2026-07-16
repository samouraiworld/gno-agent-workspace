/*
Run:

	gh pr checkout 5891 -R gnolang/gno && git checkout 82e5cb868
	cp store_split_contract_test.go gnovm/pkg/gnolang/
	go test ./gnovm/pkg/gnolang/ -run 'TestSplitProdAllButProdIsLossless|TestGetMemPackageAllRoundTrip|TestIterMemPackageYieldsProdOnly|TestFindByPrefixDeDupesNestedSplitPackages' -v

Pins the store contracts that PR 5891 introduces and currently asserts only in
code comments:

  - splitProdAllButProd is lossless (prod ∪ allButProd == mpkg.Files, no
    overlap, no drops), INCLUDING the prod-less branch that folds non-.gno
    files (gnomod.toml, README.md, ...) into the sibling.
  - GetMemPackageAll reconstructs the stored file set and stamps MPUserAll /
    MPStdlibAll per path.
  - IterMemPackage yields prod-only mempackages and skips prod-less packages
    (the `continue` added in 3a30d0928) without yielding nil.
  - FindPathsByPrefix de-dup survives nested paths, which pins the "prod key
    and #allbutprod sibling are adjacent in iavl order" invariant that the
    de-dup-against-previous-only strategy rests on.
*/
package gnolang

import (
	"slices"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSplitTestStore(t *testing.T) *defaultStore {
	t.Helper()
	d1, d2 := memdb.NewMemDB(), memdb.NewMemDB()
	d1s := dbadapter.StoreConstructor(d1, storetypes.StoreOptions{})
	d2s := dbadapter.StoreConstructor(d2, storetypes.StoreOptions{})
	return NewStore(nil, d1s, d2s)
}

func splitTestFileNames(mpkg *std.MemPackage) []string {
	if mpkg == nil {
		return nil
	}
	names := make([]string, 0, len(mpkg.Files))
	for _, mf := range mpkg.Files {
		names = append(names, mf.Name)
	}
	slices.Sort(names)
	return names
}

// TestSplitProdAllButProdIsLossless asserts the partition contract documented
// on splitProdAllButProd: prod ∪ allButProd == mpkg.Files exactly, with no
// overlap and no drops, on every branch.
func TestSplitProdAllButProdIsLossless(t *testing.T) {
	mf := func(name, body string) *std.MemFile { return &std.MemFile{Name: name, Body: body} }

	cases := []struct {
		name           string
		files          []*std.MemFile
		wantProd       []string // nil => prod must be nil
		wantAllButProd []string
	}{
		{
			name: "prod and test",
			files: []*std.MemFile{
				mf("gnomod.toml", "module = \"gno.land/r/demo/x\"\ngno = \"0.9\"\n"),
				mf("x.gno", "package x\n"),
				mf("x_test.gno", "package x\n"),
				mf("x_filetest.gno", "package main\n"),
			},
			wantProd:       []string{"gnomod.toml", "x.gno"},
			wantAllButProd: []string{"x_filetest.gno", "x_test.gno"},
		},
		{
			name: "prod with non-gno assets stays with prod",
			files: []*std.MemFile{
				mf("LICENSE", "MIT\n"),
				mf("README.md", "# x\n"),
				mf("gnomod.toml", "module = \"gno.land/r/demo/x\"\ngno = \"0.9\"\n"),
				mf("x.gno", "package x\n"),
				mf("x_test.gno", "package x\n"),
			},
			wantProd:       []string{"LICENSE", "README.md", "gnomod.toml", "x.gno"},
			wantAllButProd: []string{"x_test.gno"},
		},
		{
			name: "prod only",
			files: []*std.MemFile{
				mf("gnomod.toml", "module = \"gno.land/r/demo/x\"\ngno = \"0.9\"\n"),
				mf("x.gno", "package x\n"),
			},
			wantProd:       []string{"gnomod.toml", "x.gno"},
			wantAllButProd: nil,
		},
		{
			// The branch that only a code comment describes today: with no
			// production .gno file there is no prod blob, so the non-.gno
			// files must fold into the sibling or they are dropped from
			// storage entirely.
			name: "prod-less folds non-gno files into the sibling",
			files: []*std.MemFile{
				mf("LICENSE", "MIT\n"),
				mf("README.md", "# x\n"),
				mf("gnomod.toml", "module = \"gno.land/r/demo/x\"\ngno = \"0.9\"\n"),
				mf("x_test.gno", "package x\n"),
			},
			wantProd:       nil,
			wantAllButProd: []string{"LICENSE", "README.md", "gnomod.toml", "x_test.gno"},
		},
		{
			name: "prod-less with only test gno",
			files: []*std.MemFile{
				mf("x_test.gno", "package x\n"),
			},
			wantProd:       nil,
			wantAllButProd: []string{"x_test.gno"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mpkg := &std.MemPackage{
				Type: MPUserAll, Name: "x", Path: "gno.land/r/demo/x",
				Files: tc.files,
			}
			prod, allButProd := splitProdAllButProd(mpkg)

			if tc.wantProd == nil {
				require.Nil(t, prod, "prod must be nil when the package has no production .gno file")
			} else {
				require.NotNil(t, prod)
				assert.Equal(t, tc.wantProd, splitTestFileNames(prod))
				assert.Equal(t, MPUserProd, prod.Type, "prod blob must be typed MP*Prod")
			}
			require.NotNil(t, allButProd)
			assert.Equal(t, tc.wantAllButProd, splitTestEmptyToNil(splitTestFileNames(allButProd)))

			// Lossless: union == original, no overlap, no drops.
			union := append(splitTestFileNames(prod), splitTestFileNames(allButProd)...)
			slices.Sort(union)
			assert.Equal(t, splitTestFileNames(mpkg), union,
				"prod ∪ allButProd must equal mpkg.Files exactly")
			assert.Equal(t, len(union), len(slices.Compact(slices.Clone(union))),
				"prod and allButProd must not overlap")

			// Bodies survive the split.
			for _, orig := range mpkg.Files {
				var got *std.MemFile
				if prod != nil {
					got = prod.GetFile(orig.Name)
				}
				if got == nil {
					got = allButProd.GetFile(orig.Name)
				}
				require.NotNil(t, got, "file %q dropped by the split", orig.Name)
				assert.Equal(t, orig.Body, got.Body, "body of %q changed", orig.Name)
			}
		})
	}
}

func splitTestEmptyToNil(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

// TestGetMemPackageAllRoundTrip asserts GetMemPackageAll merges both blobs back
// into the exact stored file set, and stamps the MPAnyAll.Decide(path) type.
func TestGetMemPackageAllRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		pkgName  string
		mptype   MemPackageType
		files    []*std.MemFile
		wantType MemPackageType
	}{
		{
			name: "user package with prod and test", path: "gno.land/r/demo/foo", pkgName: "foo",
			mptype: MPUserAll,
			files: []*std.MemFile{
				{Name: "README.md", Body: "# foo\n"},
				{Name: "foo.gno", Body: "package foo\n"},
				{Name: "foo_filetest.gno", Body: "package main\n"},
				{Name: "foo_test.gno", Body: "package foo\n"},
				{Name: "gnomod.toml", Body: "module = \"gno.land/r/demo/foo\"\ngno = \"0.9\"\n"},
			},
			wantType: MPUserAll,
		},
		{
			name: "stdlib package", path: "math", pkgName: "math",
			mptype: MPStdlibAll,
			files: []*std.MemFile{
				{Name: "math.gno", Body: "package math\n"},
				{Name: "math_test.gno", Body: "package math\n"},
			},
			wantType: MPStdlibAll,
		},
		{
			// prod-less: everything lives in the sibling; the merge must
			// still return every file, including the non-.gno ones.
			name: "prod-less package", path: "gno.land/r/demo/bar", pkgName: "bar",
			mptype: MPUserAll,
			files: []*std.MemFile{
				{Name: "README.md", Body: "# bar\n"},
				{Name: "bar_test.gno", Body: "package bar\n"},
				{Name: "gnomod.toml", Body: "module = \"gno.land/r/demo/bar\"\ngno = \"0.9\"\n"},
			},
			wantType: MPUserAll,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := newSplitTestStore(t)
			orig := &std.MemPackage{Type: tc.mptype, Name: tc.pkgName, Path: tc.path, Files: tc.files}
			st.AddMemPackage(orig, tc.mptype)

			all := st.GetMemPackageAll(tc.path)
			require.NotNil(t, all)
			assert.Equal(t, splitTestFileNames(orig), splitTestFileNames(all),
				"GetMemPackageAll must return the stored file set exactly")
			for _, orig := range orig.Files {
				got := all.GetFile(orig.Name)
				require.NotNil(t, got, "file %q missing from GetMemPackageAll", orig.Name)
				assert.Equal(t, orig.Body, got.Body)
			}
			assert.Equal(t, tc.wantType, all.Type)
			assert.Equal(t, tc.path, all.Path)
			assert.Equal(t, tc.pkgName, all.Name)
			assert.True(t, slices.IsSortedFunc(all.Files, func(a, b *std.MemFile) int {
				if a.Name < b.Name {
					return -1
				} else if a.Name > b.Name {
					return 1
				}
				return 0
			}), "merged files must be sorted")
		})
	}
}

// TestIterMemPackageYieldsProdOnly pins the Store interface contract on
// IterMemPackage: prod mempackages only, prod-less packages skipped, never a
// nil element.
func TestIterMemPackageYieldsProdOnly(t *testing.T) {
	st := newSplitTestStore(t)
	add := func(name string, files ...*std.MemFile) {
		st.AddMemPackage(&std.MemPackage{
			Type: MPUserAll, Name: name, Path: "gno.land/r/demo/" + name, Files: files,
		}, MPUserAll)
	}
	add("alpha",
		&std.MemFile{Name: "alpha.gno", Body: "package alpha\n"},
		&std.MemFile{Name: "alpha_test.gno", Body: "package alpha\n"},
		&std.MemFile{Name: "gnomod.toml", Body: "module = \"gno.land/r/demo/alpha\"\ngno = \"0.9\"\n"})
	// prod-less: must be skipped, not yielded as nil.
	add("beta",
		&std.MemFile{Name: "beta_test.gno", Body: "package beta\n"},
		&std.MemFile{Name: "gnomod.toml", Body: "module = \"gno.land/r/demo/beta\"\ngno = \"0.9\"\n"})
	add("gamma",
		&std.MemFile{Name: "gamma.gno", Body: "package gamma\n"})

	var got []*std.MemPackage
	for mpkg := range st.IterMemPackage() {
		require.NotNil(t, mpkg, "IterMemPackage must never yield nil")
		got = append(got, mpkg)
	}

	require.Len(t, got, 2, "prod-less package must be skipped")
	assert.Equal(t, "gno.land/r/demo/alpha", got[0].Path)
	assert.Equal(t, "gno.land/r/demo/gamma", got[1].Path)

	// Prod-only: no test file, and typed MP*Prod.
	assert.Equal(t, []string{"alpha.gno", "gnomod.toml"}, splitTestFileNames(got[0]),
		"IterMemPackage must yield the prod blob, not the full package")
	assert.Equal(t, MPUserProd, got[0].Type)
	assert.Equal(t, MPUserProd, got[1].Type)
}

// TestFindByPrefixDeDupesNestedSplitPackages pins the ordering invariant the
// de-dup rests on: a package's prod key and its #allbutprod sibling are
// ADJACENT in iavl order, so comparing against only the previous path
// suffices. A nested path (alpha and alpha/sub) is what separates an adjacent
// suffix from a non-adjacent one: with a suffix byte sorting after '/', the
// sibling of alpha would sort after alpha/sub and "alpha" would be yielded
// twice. The existing TestFindByPrefixDeDupesSplitPackages uses only sibling
// paths (alpha/beta/gamma) and stays green under such a change.
func TestFindByPrefixDeDupesNestedSplitPackages(t *testing.T) {
	st := newSplitTestStore(t)
	add := func(path, name string, files ...*std.MemFile) {
		st.AddMemPackage(&std.MemPackage{
			Type: MPUserAll, Name: name, Path: path, Files: files,
		}, MPUserAll)
	}
	// Every package is split (prod blob + #allbutprod sibling).
	add("gno.land/r/demo/alpha", "alpha",
		&std.MemFile{Name: "alpha.gno", Body: "package alpha\n"},
		&std.MemFile{Name: "alpha_test.gno", Body: "package alpha\n"})
	add("gno.land/r/demo/alpha/sub", "sub",
		&std.MemFile{Name: "sub.gno", Body: "package sub\n"},
		&std.MemFile{Name: "sub_test.gno", Body: "package sub\n"})
	add("gno.land/r/demo/alphax", "alphax",
		&std.MemFile{Name: "alphax.gno", Body: "package alphax\n"},
		&std.MemFile{Name: "alphax_test.gno", Body: "package alphax\n"})

	var got []string
	st.FindPathsByPrefix("gno.land")(func(p string) bool {
		got = append(got, p)
		return true
	})
	require.Equal(t, []string{
		"gno.land/r/demo/alpha",
		"gno.land/r/demo/alpha/sub",
		"gno.land/r/demo/alphax",
	}, got, "each split package must be listed exactly once")
}
