// Repro, from a plain clone of gnolang/gno:
//
//	git fetch origin pull/5593/head && git checkout --detach fa1ce31087d2ff53ae775bb71a1f52362543e45e
//	cp <this file> gno.land/pkg/gnoweb/solo_tab_padding_render_test.go
//	go test ./gno.land/pkg/gnoweb/ -run TestSoloRenderRealmTabDetails -v
//
// Merge base, without writing the tree:
//
//	printf '{"Replace":{"%s/gno.land/pkg/gnoweb/solo_tab_padding_render_test.go":"%s"}}' "$PWD" "<this file>" > /tmp/ov.json
//	go test -overlay /tmp/ov.json ./gno.land/pkg/gnoweb/ -run TestSoloRenderRealmTabDetails -v
//
// Claim: RenderRealm, fed the bytes the way GetMarkdownView builds them
// ([]byte(string), cap rounded to the size class, so a 64-byte page has
// cap == len), panics when the page ends on a `:::details Summary` line
// that a container entered through a tab (a list item's tab continuation,
// `>\t`): the summary segment is padding bytes past the line end.
// Short of the panic, the summary loses its first padding bytes.
package gnoweb

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
)

func TestSoloRenderRealmTabDetails(t *testing.T) {
	u, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatal(err)
	}
	r := NewHTMLRenderer(slog.New(slog.DiscardHandler), NewDefaultRenderConfig(), nil)

	tail := "- item\n\t:::details Summary text"
	for _, name := range []string{"mid_page", "page_end_64"} {
		var s string
		if name == "mid_page" {
			s = tail + "\n\tbody\n\nfooter\n"
		} else {
			s = strings.Repeat("a", 64-len(tail)-2) + "\n\n" + tail
		}
		src := []byte(s)
		t.Run(name, func(t *testing.T) {
			t.Logf("len=%d cap=%d", len(src), cap(src))
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("RenderRealm panicked: %v", p)
				}
			}()
			var buf bytes.Buffer
			if _, err := r.RenderRealm(&buf, u, src, RealmRenderContext{}); err != nil {
				t.Fatal(err)
			}
			t.Logf("output:\n%s", buf.String())
			if strings.Contains(buf.String(), "<summary>") && !strings.Contains(buf.String(), ">Summary text</summary>") {
				t.Errorf("summary text mangled")
			}
		})
	}
}
