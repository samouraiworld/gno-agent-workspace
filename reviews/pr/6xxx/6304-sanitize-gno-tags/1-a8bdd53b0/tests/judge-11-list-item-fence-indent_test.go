// Asserts EscapeBlockHazardsRich keeps user text from binding to a reference
// definition the realm's chrome declares after it, when the user's fence sits
// inside a list item 4-6 spaces in (indent 0-3 from the item's content column).
// Fails at PR #6304 head a8bdd53b0: the item's `see [home]` renders as the realm's link.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6304/head && git checkout a8bdd53b08e988227136f8b74f4cd6571956e5f4
//	cp <this file> gnovm/stdlibs/chain/markdown/zz_judge11_test.go
//	go test ./gnovm/stdlibs/chain/markdown -run TestJudge11ListItemFenceIndent -v -count=1
//	rm gnovm/stdlibs/chain/markdown/zz_judge11_test.go
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

const realmHref = `href="https://realm.example/home"`

// renderWithChrome renders out between realm chrome that defines [home] after
// the user text, with CommonMark goldmark.
func renderWithChrome(t *testing.T, out string) string {
	t.Helper()
	page := "REALM-HEADER\n\n\n" + out + "\n\n" + "[home]: https://realm.example/home\n"
	var buf bytes.Buffer
	if err := goldmark.New().Convert([]byte(page), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestJudge11ListItemFenceIndent(t *testing.T) {
	cases := []struct {
		name, in string
		control  bool
	}{
		{"control-no-fence", "- a\n\n  see [home]\n", true},
		{"bullet-fence-indent3-of-item", "- a\n\n     ```\n  ```\n  see [home]\n", false},
		{"ordered-fence-indent4-abs", "1. a\n\n    ```\n   ```\n   see [home]\n", false},
	}
	for _, c := range cases {
		out := EscapeBlockHazardsRich(c.in)
		html := renderWithChrome(t, out)
		bound := strings.Contains(html, realmHref)
		t.Logf("%s: out=%q realm-link-bound=%v", c.name, out, bound)
		// IS (a8bdd53b0): the fenced cases bind [home] to the realm's URL.
		// SHOULD: no user text binds to the realm's definition.
		if bound {
			t.Errorf("%s: user text links to the realm's [home]\nhtml: %s", c.name, html)
		}
	}
}
