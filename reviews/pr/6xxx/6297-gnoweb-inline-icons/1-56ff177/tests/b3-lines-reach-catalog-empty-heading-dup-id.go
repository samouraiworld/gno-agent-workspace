// gnolang/gno#6297 at 56ff1770722eee73f51f207f97554ea1e2ea2b00: an empty ATX
// heading keeps the id "heading" from goldmark, but iconHeadingIDTransformer
// skips it (h.Lines().Len() == 0) and never registers that id in the fresh
// linearIDs it regenerates every other heading from, so the next heading that
// slugs to "heading" (icon-only, "Heading", CJK-only) gets the same id.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6297/head && git checkout --detach 56ff1770722eee73f51f207f97554ea1e2ea2b00
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_b3_dupid_test.go
//	cd gno.land/pkg/gnoweb/markdown && go test -count=1 -run TestB3EmptyHeadingDuplicateID -v .
//
// Expect: the control case logs [heading heading-1] and passes; the two
// icon cases FAIL with duplicate heading id "heading".
package markdown

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/yuin/goldmark/parser"
)

var b3DupHeadingID = regexp.MustCompile(`<h[1-6] id="([^"]*)"`)

func TestB3EmptyHeadingDuplicateID(t *testing.T) {
	m := newProductionLikeMarkdown()
	for _, src := range []string{
		"##\n\n## Heading\n", // control: no icon anywhere
		"##\n\n## Heading\n\n## <gno-icon name=\"star\" /> A\n",
		"##\n\n## <gno-icon name=\"star\" label=\"Top\" />\n",
	} {
		var buf bytes.Buffer
		if err := m.Convert([]byte(src), &buf, parser.WithContext(NewGnoParserContext(GnoContext{}))); err != nil {
			t.Fatal(err)
		}
		var ids []string
		seen := map[string]int{}
		for _, mm := range b3DupHeadingID.FindAllStringSubmatch(buf.String(), -1) {
			ids = append(ids, mm[1])
			seen[mm[1]]++
		}
		t.Logf("src=%q ids=%v", src, ids)
		for id, n := range seen {
			if n > 1 {
				t.Errorf("duplicate heading id %q (%d times) for %q", id, n, src)
			}
		}
	}
}
