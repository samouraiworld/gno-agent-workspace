// Asserts a $$ or \[ display opener indented by 1-3 spaces opens a math block,
// as an indented closer and an indented ```math fence already do. Fails at
// 0b3987540 for every indented opener; the merge base 87f0357fe has no math.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/4879/head && git checkout --detach 0b3987540842f0f7a875e2a344d5995fd47f40d5
//   cp judge-5-indented-display-opener_test.go gno.land/pkg/gnoweb/zz_judge5_test.go
//   go test -count=1 -v -run 'TestJudge5IndentedOpener$' ./gno.land/pkg/gnoweb/
//   rm gno.land/pkg/gnoweb/zz_judge5_test.go
package gnoweb

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
)

func TestJudge5IndentedOpener(t *testing.T) {
	for _, c := range []struct{ name, src string }{
		{"control: no indent", "$$\nx^2\n$$\n"},
		{"control: unindented \\\\[", "\\\\[\nx^2\n\\\\]\n"},
		{"control: indented closer", "$$\nx^2\n  $$\n"},
		{"1-space $$ opener", " $$\nx^2\n$$\n"},
		{"3-space $$ opener", "   $$\nx^2\n$$\n"},
		{"2-space \\\\[ opener", "  \\\\[\nx^2\n\\\\]\n"},
		{"list item continuation", "- item\n\n   $$\n   x^2\n   $$\n"},
	} {
		r := newTestRenderer()
		var w bytes.Buffer
		if _, err := r.RenderRealm(&w, &weburl.GnoURL{Path: "/r/test"}, []byte(c.src), RealmRenderContext{ChainId: "dev"}); err != nil {
			t.Fatal(err)
		}
		got := strings.Contains(w.String(), `display="block"`)
		t.Logf("%-26s block math=%v", c.name, got)
		if !got { // IS: the raw "$$ x^2 $$" paragraph
			t.Errorf("%s: %q renders no display math:\n%s", c.name, c.src, w.String())
		}
		// SHOULD: block math=true for every case
	}
}
