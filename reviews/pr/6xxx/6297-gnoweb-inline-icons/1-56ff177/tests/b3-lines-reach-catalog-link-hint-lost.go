// gnolang/gno#6297 at 56ff1770722eee73f51f207f97554ea1e2ea2b00:
// aloneInNamedParent walks the whole heading for the first unlabeled icon in
// it and marks every nested icon hintDone with hint=false, so an icon alone in
// a link inside that heading never gets its "add label" hint, while the same
// link in a heading with no earlier icon does.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6297/head && git checkout --detach 56ff1770722eee73f51f207f97554ea1e2ea2b00
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_b3_hint_test.go
//	cd gno.land/pkg/gnoweb/markdown && go test -count=1 -run TestB3LinkHintLostInsideHeading -v .
//
// Expect: the control renders the hint; both probes FAIL with
// "an icon-only link inside a heading got no hint".
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark/parser"
)

func TestB3LinkHintLostInsideHeading(t *testing.T) {
	const hint = "gno-icon: alone in a link or heading"
	m := newProductionLikeMarkdown()
	render := func(src string) string {
		var buf bytes.Buffer
		if err := m.Convert([]byte(src), &buf, parser.WithContext(NewGnoParserContext(GnoContext{}))); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	if control := render("## Title [<gno-icon name=\"star\" />](/r/x)\n"); !strings.Contains(control, hint) {
		t.Fatalf("control lost the hint too: %s", control)
	}
	for _, src := range []string{
		"## <gno-icon name=\"star\" /> Title [<gno-icon name=\"star\" />](/r/x)\n",
		"## <gno-icon name=\"star\" /> Title [<gno-icon name=\"book\" />](/r/x) [<gno-icon name=\"book\" />](/r/y)\n",
	} {
		if out := render(src); !strings.Contains(out, hint) {
			t.Errorf("an icon-only link inside a heading got no hint: %q", src)
		}
	}
}
