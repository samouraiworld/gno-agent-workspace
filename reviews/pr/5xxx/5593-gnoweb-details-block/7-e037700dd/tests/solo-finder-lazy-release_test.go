// Repro: a lazy continuation line starting with `<` releases a gno-details
// block opened in a blockquote or list item while that container stays open,
// so its `</gno-details>` no longer pairs and the block swallows the rest of
// the container; repeated, the release lets details nest past MaxGnoNestDepth.
//
// From a plain clone of gnolang/gno at e037700ddd8261bbc7660382d4953a978b630e55:
//
//	cp solo-finder-lazy-release_test.go gno.land/pkg/gnoweb/markdown/zz_solo_finder_lazy_test.go
//	go test -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestSoloFinderLazy
//
// Expect: the *_control cases print `<p>after</p>` outside `</details>`; the
// *_lazy cases print `<p>after</p>` inside the details plus a stray
// `<!-- unexpected/invalid details tag omitted -->`; nest_lazy prints a nested
// details count above MaxGnoNestDepth (4).
package markdown

import (
	"strings"
	"testing"
)

func TestSoloFinderLazy(t *testing.T) {
	cases := []struct{ name, src string }{
		{"bq_control", "> <gno-details summary=\"S\">\n> body\nplain lazy line\n> </gno-details>\n> after\n"},
		{"bq_lazy", "> <gno-details summary=\"S\">\n> body\n<b>lazy</b> line\n> </gno-details>\n> after\n"},
		{"list_control", "- <gno-details summary=\"S\">\n  body\nplain lazy line\n  </gno-details>\n  after\n"},
		{"list_lazy", "- <gno-details summary=\"S\">\n  body\n<b>lazy</b> line\n  </gno-details>\n  after\n"},
		{"list_lazy_autolink", "- <gno-details summary=\"Changelog\">\n  Fixed the parser, see\n<https://gno.land/r/docs> for details.\n  </gno-details>\n- next item\n"},
		{"nest_control", strings.Repeat("> <gno-details>\n> p\nplain\n", 10)},
		{"nest_lazy", strings.Repeat("> <gno-details>\n> p\n<i>x</i>\n", 10)},
	}
	for _, c := range cases {
		out, depth := parseDetails(t, c.src)
		t.Logf("== %s (depth after parse %d, details opened %d, stray tags %d)\n%s", c.name, depth,
			strings.Count(out, `<details class="gno-details">`),
			strings.Count(out, "unexpected/invalid details tag omitted"), out)
	}
}

// The math and button extensions upstream added: lines the details parser
// now sees, printed for a read.
func TestSoloFinderLazyPeers(t *testing.T) {
	cases := []struct{ name, src string }{
		{"math_holds_details_line", "$$\na\n<gno-details>\nb\n$$\n"},
		{"math_inside_details", "<gno-details>\n$$\nx\n$$\n</gno-details>\nafter\n"},
		{"math_close_after_details_close", "<gno-details>\n$$\nx\n</gno-details>\n$$\n"},
		{"button_in_details", "<gno-details>\n<gno-button href=\"/r/x\" label=\"L\" />\n</gno-details>\n"},
		{"button_in_quote_details", "> <gno-details>\n> <gno-button href=\"/r/x\" label=\"L\" />\n> </gno-details>\n"},
	}
	for _, c := range cases {
		out, depth := parseDetails(t, c.src)
		t.Logf("== %s (depth %d)\n%s", c.name, depth, out)
	}
}
