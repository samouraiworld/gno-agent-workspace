package main

/* Run: from a gno checkout:
gh pr checkout 6078 -R gnolang/gno && git checkout b0913fc73
curl -fsSL -o gnovm/cmd/gno/lint_filetests_drift_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6078-reject-tooling-directives/1-b0913fc73/tests/lint_filetests_drift_test.go
go test -v -run 'TestLintGnoFilesSkipsNonFiletests' ./gnovm/cmd/gno/
rm gnovm/cmd/gno/lint_filetests_drift_test.go
*/

// ReadMemPackage takes only *_filetest.gno out of filetests/, so any other
// .gno there never reaches a mempackage. lintGnoFiles takes every .gno in the
// directory, so lint scans a file AddPackage never sees.
// At b0913fc73 this fails with one extra name on the lint side.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/stretchr/testify/require"
)

func TestLintGnoFilesSkipsNonFiletests(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "filetests"), 0o755))
	write := func(rel, body string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644))
	}
	write("gnomod.toml", "module = \"gno.land/p/demo/zz\"\ngno = \"0.9\"\n")
	write("zz.gno", "package zz\n\nfunc F() {}\n")
	write(filepath.Join("filetests", "a_filetest.gno"), "package main\n\nfunc main() {}\n")
	// Not a _filetest.gno: ReadMemPackage drops it, so the chain never
	// validates it and lint must not either.
	write(filepath.Join("filetests", "helper.gno"), "//go:build ignore\n\npackage helper\n\nfunc H() {}\n")

	mpkg, err := gno.ReadMemPackage(dir, "gno.land/p/demo/zz", gno.MPAnyAll)
	require.NoError(t, err)
	// The package the chain sees carries no directive, so AddPackage accepts it.
	require.NoError(t, gno.ValidateMemPackageAny(mpkg))

	var fromMemPkg []string
	for _, f := range mpkg.Files {
		if strings.HasSuffix(f.Name, ".gno") {
			fromMemPkg = append(fromMemPkg, filepath.Base(f.Name))
		}
	}
	var fromLint []string
	for _, p := range lintGnoFiles(dir) {
		fromLint = append(fromLint, filepath.Base(p))
	}
	sort.Strings(fromMemPkg)
	sort.Strings(fromLint)
	require.Equal(t, fromMemPkg, fromLint,
		"lintGnoFiles must cover exactly the .gno files ReadMemPackage puts in the package")
}
