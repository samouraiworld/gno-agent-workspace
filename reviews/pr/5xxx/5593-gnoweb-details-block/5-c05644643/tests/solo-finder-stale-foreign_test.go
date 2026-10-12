// Repro: a <gno-foreign> left unclosed inside a blockquote or list item keeps
// Closed == false after its container closes it, so detailsParser.Continue
// still finds it on the last-descendant chain and never takes the details'
// own `:::` closer: the closer renders as a literal paragraph and everything
// after it (here `footer`) folds into the collapsed block.
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/5593/head && git checkout c05644643
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_stale_foreign_test.go
//   go test ./gno.land/pkg/gnoweb/markdown -run TestZZStaleForeign -v   # FAIL at c05644643
//   git checkout 8ab095f49 -- gno.land/pkg/gnoweb/markdown/ext_details.go
//   go test ./gno.land/pkg/gnoweb/markdown -run TestZZStaleForeign -v   # PASS before 87aa65612
//   git checkout c05644643 -- gno.land/pkg/gnoweb/markdown/ext_details.go
package markdown

import (
	"strings"
	"testing"
)

func TestZZStaleForeign(t *testing.T) {
	cases := map[string]string{
		"quote_blank": ":::details A\n> <gno-foreign>\n> text\n\n:::\n\nfooter\n",
		"list_blank":  ":::details A\n- <gno-foreign>\n  text\n\n:::\n\nfooter\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			_, out, _ := parseDetails(t, src)
			if !strings.HasSuffix(out, "</details>\n<p>footer</p>\n") {
				t.Errorf("footer folded into the details:\n%s", out)
			}
		})
	}
}
