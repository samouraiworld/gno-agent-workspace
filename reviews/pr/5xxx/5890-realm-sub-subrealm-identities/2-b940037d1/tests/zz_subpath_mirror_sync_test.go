/* Run:
gh pr checkout 5890 -R gnolang/gno && git checkout b940037d1
curl -fsSL -o gnovm/pkg/gnolang/zz_subpath_mirror_sync_test.go https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5890-realm-sub-subrealm-identities/2-b940037d1/tests/zz_subpath_mirror_sync_test.go
go test ./gnovm/pkg/gnolang/ -run TestSubpathGrammarMirrorInSync -v
rm gnovm/pkg/gnolang/zz_subpath_mirror_sync_test.go
*/

// The sub-realm subpath grammar and separator exist twice: the Go native
// (uverse.go) and the .gno mirror (stdlibs/chain/address.gno), tied only by
// "keep in sync" comments. At b940037d1 both sides agree, so this passes; it
// fails the moment either side drifts alone (verified: teaching only the .gno
// isValidSubpathSegment to accept non-ASCII leaves the entire TestFiles suite
// green, but turns this test red).

package gnolang

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"testing"
)

const (
	goGrammarFile  = "uverse.go"
	gnoGrammarFile = "../../stdlibs/chain/address.gno"
)

// mirroredFuncs are the grammar functions that must be byte-for-byte
// transliterations of each other across the Go/.gno boundary. Both files
// declare them at package scope with identical signatures.
var mirroredFuncs = []string{"isValidSubpath", "isValidSubpathSegment"}

func parseGrammarFile(t *testing.T, path string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return fset, f
}

// funcBody renders the named function's body, dropping comments and
// positions so only the executable form is compared.
func funcBody(t *testing.T, fset *token.FileSet, f *ast.File, name string) string {
	t.Helper()
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != name || fd.Body == nil {
			continue
		}
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, fd.Body); err != nil {
			t.Fatalf("print %s: %v", name, err)
		}
		return buf.String()
	}
	t.Fatalf("func %s not found in %s", name, f.Name.Name)
	return ""
}

// constValue returns the source form of a package-scope string const.
func constValue(t *testing.T, fset *token.FileSet, f *ast.File, name string) string {
	t.Helper()
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, id := range vs.Names {
				if id.Name != name || i >= len(vs.Values) {
					continue
				}
				var buf bytes.Buffer
				if err := printer.Fprint(&buf, fset, vs.Values[i]); err != nil {
					t.Fatalf("print const %s: %v", name, err)
				}
				return buf.String()
			}
		}
	}
	t.Fatalf("const %s not found in %s", name, f.Name.Name)
	return ""
}

// TestSubpathGrammarMirrorInSync pins the mirror relationship structurally
// rather than by convention. cur.Sub (Go) and chain.DerivePkgSubAddr (.gno)
// must accept exactly the same subpaths: if the .gno side accepted a subpath
// the Go side refuses, callers could derive and fund an address that cur.Sub
// can never mint against, stranding those funds — the failure mode
// assertValidSubpath's own doc comment names.
func TestSubpathGrammarMirrorInSync(t *testing.T) {
	t.Parallel()

	goFset, goFile := parseGrammarFile(t, goGrammarFile)
	gnoFset, gnoFile := parseGrammarFile(t, gnoGrammarFile)

	for _, name := range mirroredFuncs {
		goBody := funcBody(t, goFset, goFile, name)
		gnoBody := funcBody(t, gnoFset, gnoFile, name)
		if goBody != gnoBody {
			t.Errorf("grammar drift: %s differs between %s and %s\n--- Go ---\n%s\n--- .gno ---\n%s",
				name, goGrammarFile, gnoGrammarFile, goBody, gnoBody)
		}
	}

	// The separator is likewise declared on both sides. A divergence here
	// silently repoints address derivation at a different preimage.
	goSep := constValue(t, goFset, goFile, "subRealmSep")
	gnoSep := constValue(t, gnoFset, gnoFile, "subRealmSep")
	if goSep != gnoSep {
		t.Errorf("separator drift: subRealmSep is %s in %s but %s in %s",
			goSep, goGrammarFile, gnoSep, gnoGrammarFile)
	}

	// Guard the guard: the extraction must actually be reading the real
	// grammar, not silently matching two empty strings.
	if body := funcBody(t, goFset, goFile, "isValidSubpathSegment"); len(body) < 100 {
		t.Errorf("extracted body implausibly short (%d bytes) — extraction likely broken", len(body))
	}
	if goSep != `"#"` {
		t.Errorf("subRealmSep = %s, want \"#\" at b940037d1", goSep)
	}
}
