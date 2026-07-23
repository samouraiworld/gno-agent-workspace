/* Run: from a gno checkout:
gh pr checkout 5978 -R gnolang/gno && git checkout da74644bf
curl -fsSL -o gnovm/pkg/gnolang/buildtag_import_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5978-pin-per-file-go-version/1-da74644bf/tests/buildtag_import_test.go
go test -v -run 'TestTypeCheckMemPackage_BuildTagOnImport$' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/buildtag_import_test.go
*/

// GoParseMemPackage parses imports too, so a //go:build go1.N line in a
// dependency raises that dependency's language version the same way it raises
// the submitted package's. Passes at da74644bf; without the blanking in
// GoParseMemPackage the import type-checks clean and the whole package is
// accepted.
package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/assert"
)

type buildTagImportGetter map[string]*std.MemPackage

func (g buildTagImportGetter) GetMemPackage(path string) *std.MemPackage {
	return g[path]
}

func TestTypeCheckMemPackage_BuildTagOnImport(t *testing.T) {
	t.Parallel()

	dep := &std.MemPackage{
		Type: MPUserProd,
		Name: "dep",
		Path: "gno.land/p/demo/dep",
		Files: []*std.MemFile{{Name: "dep.gno", Body: "//go:build go1.22\n\n" +
			"package dep\nfunc G() { for range 10 {} }\n"}},
	}
	root := &std.MemPackage{
		Type: MPUserProd,
		Name: "z",
		Path: "gno.land/p/demo/z",
		Files: []*std.MemFile{{Name: "z.gno", Body: "package z\n" +
			"import \"gno.land/p/demo/dep\"\nfunc F() { dep.G() }\n"}},
	}

	getter := buildTagImportGetter{dep.Path: dep}
	_, err := TypeCheckMemPackage(root, TypeCheckOptions{
		Getter:     getter,
		TestGetter: getter,
		Mode:       TCLatestRelaxed,
	})
	assert.ErrorContains(t, err, "go1.22",
		"a //go:build line in an imported package must not raise the pinned "+
			"GoVersion for that import")
}
