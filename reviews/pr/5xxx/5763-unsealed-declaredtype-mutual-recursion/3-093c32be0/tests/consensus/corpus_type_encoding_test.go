// Corpus differential for PR 5763: for every named type in every package
// under examples/gno.land, dump the TypeID and a hash of the exact amino
// bytes that would be persisted for the type and for its Base. Run on the
// merge-base and on the PR head, then diff. Anything that differs is
// on-chain state that would decode differently on an upgraded node.
//
// Measured at 093c32be0 vs merge-base 0397fc87f: 136 packages, 343 types,
// zero differences. Results in corpus_result.txt.
//
/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/pkg/test/zz_corpus_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/consensus/corpus_type_encoding_test.go
ZZ_OUT=/tmp/head_types.txt go test -v -count=1 -timeout 40m -run TestZZCorpusTypeEncoding ./gnovm/pkg/test/
git checkout $(git merge-base origin/master HEAD) -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/types.go
ZZ_OUT=/tmp/base_types.txt go test -v -count=1 -timeout 40m -run TestZZCorpusTypeEncoding ./gnovm/pkg/test/
git checkout HEAD -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/types.go
diff /tmp/base_types.txt /tmp/head_types.txt && echo "no encoding drift"
rm gnovm/pkg/test/zz_corpus_test.go
*/

package test

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/amino"
)

// TestZZCorpusTypeEncoding dumps, for every package-level named type in
// every examples/gno.land/p package, the TypeID plus a hash of the exact
// amino bytes that would be persisted for the type and for its Base.
// Run on two trees and diff to detect any on-chain encoding drift.
func TestZZCorpusTypeEncoding(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	var pkgPaths []string
	base := filepath.Join(root, "examples", "gno.land")
	err = filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		ents, _ := os.ReadDir(p)
		has := false
		for _, e := range ents {
			n := e.Name()
			if strings.HasSuffix(n, ".gno") && !strings.HasSuffix(n, "_test.gno") &&
				!strings.HasSuffix(n, "_filetest.gno") {
				has = true
			}
		}
		if has {
			rel, _ := filepath.Rel(filepath.Join(root, "examples"), p)
			pkgPaths = append(pkgPaths, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(pkgPaths)
	out, err := os.Create(os.Getenv("ZZ_OUT"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	fmt.Fprintf(out, "# packages: %d\n", len(pkgPaths))

	for _, pkgPath := range pkgPaths {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(out, "%s\tPANIC\t%v\n", pkgPath, r)
				}
			}()
			_, st := ProdStore(root, io.Discard, nil)
			pv := st.GetPackage(pkgPath, false)
			if pv == nil {
				fmt.Fprintf(out, "%s\tNILPKG\n", pkgPath)
				return
			}
			pb := pv.GetBlock(st)
			names := pb.Source.GetBlockNames()
			var lines []string
			for i, tv := range pb.Values {
				if tv.T == nil || tv.T.Kind() != gno.TypeKind {
					continue
				}
				var name gno.Name
				if i < len(names) {
					name = names[i]
				}
				ty := tv.GetType()
				h := func(x gno.Type) string {
					bz := amino.MustMarshalAny(gno.PersistedTypeFormForTypeValue(x))
					s := sha256.Sum256(bz)
					return fmt.Sprintf("%x", s[:8])
				}
				line := fmt.Sprintf("%s\t%s\ttid=%s\tself=%s", pkgPath, name, ty.TypeID(), h(ty))
				if dt, ok := ty.(*gno.DeclaredType); ok && dt.Base != nil {
					line += fmt.Sprintf("\tbaseTid=%s\tbase=%s\tbaseStr=%s",
						dt.Base.TypeID(), h(dt.Base), dt.Base.String())
				}
				lines = append(lines, line)
			}
			sort.Strings(lines)
			for _, l := range lines {
				fmt.Fprintln(out, l)
			}
		}()
	}
}
