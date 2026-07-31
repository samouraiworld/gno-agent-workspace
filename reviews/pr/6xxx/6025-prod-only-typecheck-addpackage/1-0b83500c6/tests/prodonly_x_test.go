/* Run: from a gno checkout:
gh pr checkout 6025 -R gnolang/gno && git checkout 0b83500c6
curl -fsSL -o gnovm/pkg/gnolang/prodonly_x_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6025-prod-only-typecheck-addpackage/1-0b83500c6/tests/prodonly_x_test.go
go test -v -run 'TestTypeCheckMemPackage_ProdOnly' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/prodonly_x_test.go
*/

// ProdOnly stops after the production type-check pass. These cases pin both
// directions at the gnovm layer: a broken test file is ignored, and production
// code still cannot reach a symbol that only a test file declares.
// All four pass at 0b83500c6; gnovm's own suite covers none of them today.

package gnolang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestTypeCheckMemPackage_ProdOnly(t *testing.T) {
	t.Parallel()

	// A test file that parses but does not type-check.
	brokenTest := func() *std.MemPackage {
		mpkg := &std.MemPackage{
			Type: MPUserAll,
			Name: "hello",
			Path: "gno.land/p/demo/hello",
		}
		mpkg.SetFile("hello.gno", "package hello\n\nfunc Hi() string { return \"hi\" }\n")
		mpkg.SetFile("hello_test.gno", "package hello\n\nfunc Broken() int { return undefinedSymbol(42) }\n")
		return mpkg
	}

	// Production code referencing a symbol only the test file declares.
	prodBorrows := func() *std.MemPackage {
		mpkg := &std.MemPackage{
			Type: MPUserAll,
			Name: "hello",
			Path: "gno.land/p/demo/hello",
		}
		mpkg.SetFile("hello.gno", "package hello\n\nfunc Hi() string { return helperOnlyInTest() }\n")
		mpkg.SetFile("hello_test.gno", "package hello\n\nfunc helperOnlyInTest() string { return \"hi\" }\n")
		return mpkg
	}

	t.Run("BrokenTestFileRejectedWithoutProdOnly", func(t *testing.T) {
		t.Parallel()
		_, err := TypeCheckMemPackage(brokenTest(), TypeCheckOptions{Mode: TCLatestRelaxed})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "undefinedSymbol")
	})

	t.Run("BrokenTestFileAcceptedWithProdOnly", func(t *testing.T) {
		t.Parallel()
		_, err := TypeCheckMemPackage(brokenTest(), TypeCheckOptions{Mode: TCLatestRelaxed, ProdOnly: true})
		assert.NoError(t, err)
	})

	// The production pass covers exactly the file set the VM runs, so this
	// stays rejected with ProdOnly set.
	t.Run("ProdCannotBorrowFromTestFile", func(t *testing.T) {
		t.Parallel()
		_, err := TypeCheckMemPackage(prodBorrows(), TypeCheckOptions{Mode: TCLatestRelaxed, ProdOnly: true})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "undefined: helperOnlyInTest")
	})

	// An unparseable test file fails before any type-check pass, so ProdOnly
	// does not let it through.
	t.Run("UnparseableTestFileStillRejected", func(t *testing.T) {
		t.Parallel()
		mpkg := &std.MemPackage{
			Type: MPUserAll,
			Name: "hello",
			Path: "gno.land/p/demo/hello",
		}
		mpkg.SetFile("hello.gno", "package hello\n\nfunc Hi() string { return \"hi\" }\n")
		mpkg.SetFile("hello_test.gno", "package hello\n\nfunc Broken( {{{ not gno at all ]]]\n")
		_, err := TypeCheckMemPackage(mpkg, TypeCheckOptions{Mode: TCLatestRelaxed, ProdOnly: true})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hello_test.gno")
	})
}
