// Repro: an unclosed $$ (or \\[) line opens a display block whenever the same
// delimiter appears on ANY later line within 8KB, across blank lines, block
// boundaries and container ends, so the lines in between are swallowed.
//
// From a plain clone of gnolang/gno:
//   gh pr checkout 4879 -R gnolang/gno   # head 0193f6de7fc5823e3c01d4a5e580e377cde98659
//   cp <this file> gno.land/pkg/gnoweb/markdown/finder1_swallow_test.go
//   go test -run TestFinder1UnclosedDisplaySwallows -v ./gno.land/pkg/gnoweb/markdown/
//   rm gno.land/pkg/gnoweb/markdown/finder1_swallow_test.go
//
// Expected at head: FAIL on all four cases. The merge base (87f0357fe) has no
// math extension, so every $$ line there is plain paragraph text.
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestFinder1UnclosedDisplaySwallows(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"heading after unclosed $$, inline $$ later", "$$\noops forgot to close\n\n## Section 2\n\nMore text with $$y$$ here.\n", "<h2"},
		{"fence after unclosed $$ eats rest of page", "$$\nunclosed\n\n```go\n$$\n```\n\nlast paragraph\n", "<p>last paragraph</p>"},
		{"blockquote $$ closed by $$ outside quote", "> $$\n> unclosed in quote\n\nafter quote\n\n$$\nx\n$$\n", "unclosed in quote</p>"},
		{"list item $$ closed by $$ in later paragraph", "- $$\n  item text\n- second item\n\nthen $$z$$\n", "<li>$$"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(c.src), &buf); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			if !strings.Contains(out, c.want) {
				t.Errorf("want %q in output, got:\n%s", c.want, strings.Join(strings.Fields(out), " "))
			}
		})
	}
}
