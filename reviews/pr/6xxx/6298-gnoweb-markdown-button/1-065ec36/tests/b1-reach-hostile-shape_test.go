// b1-reach-hostile-shape: TestButtonHostileBudget and BenchmarkButtonHostileLine
// pin two hostile shapes, `<gno-button href="/r/x" ` and `<gno-button href="/>" `,
// where each parse attempt stops at the next '<' after a few bytes. This probe
// re-runs the sibling shape where the '<' sits inside an attribute name or an
// unquoted value, so scanGnoTag never reaches its `case c == '<'` and every
// attempt reads the whole maxButtonTagLen window.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 065ec369b321c207815f82b0c1dbf2b0f9f10b01
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_b1_reach_probe_test.go
//   go test ./gno.land/pkg/gnoweb/markdown -run 'TestB1Reach' -count=1 -v
//   rm gno.land/pkg/gnoweb/markdown/zz_b1_reach_probe_test.go
//
// Result at 065ec369b (Go 1.25.9, 1 MiB on one line, full gnoweb render):
//   pinned: unterminated           attempt reads   22 bytes, 162ms
//   pinned: slash in value         attempt reads   20 bytes, 155ms
//   sibling: '<' in attr name      attempt reads 2048 bytes, 380ms
//   sibling: '<' in unquoted value attempt reads 2048 bytes, 360ms
//   control: not the prefix                                 140ms
//   control: markdown link                                  243ms
// The sibling shape reads the full maxButtonTagLen window per attempt, yet the
// render stays linear (2.3x the pinned shapes, 1.6x plain links): the cap holds,
// and TestParseButtonTagBound asserts n == 0 one byte past the cap, so the cap
// itself is pinned (read, not mutated). Refuted.
package markdown

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

func b1Render(t *testing.T, src []byte) time.Duration {
	t.Helper()
	gnourl, _ := weburl.Parse("https://gno.land/r/test")
	m := goldmark.New(
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithExtensions(extension.Strikethrough, extension.Table, extension.Footnote, extension.TaskList,
			NewGnoExtension(WithImageValidator(AllowSvgDataImage))))
	var buf bytes.Buffer
	ctx := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))
	start := time.Now()
	if err := m.Convert(src, &buf, ctx); err != nil {
		t.Fatal(err)
	}
	return time.Since(start)
}

// b1AttemptRead returns how far into src one scanGnoTag attempt read: the end
// offset of the last attribute it handed to the callback.
func b1AttemptRead(item string) int {
	src := []byte(strings.Repeat(item, 2000))
	last := 0
	scanGnoTag(src, buttonTagPrefix, maxButtonTagLen, func(k, v []byte) {
		last = cap(src) - cap(k) + len(k)
		if len(v) > 0 {
			last = cap(src) - cap(v) + len(v)
		}
	})
	return last
}

func TestB1ReachHostileShapes(t *testing.T) {
	const size = 1 << 20 // 1 MiB on one line
	shapes := []struct{ name, item string }{
		{"pinned: unterminated", `<gno-button href="/r/x" `},
		{"pinned: slash in value", `<gno-button href="/>" `},
		{"sibling: '<' in attr name", `<gno-button x`},
		{"sibling: '<' in unquoted value", `<gno-button x=`},
		{"control: not the prefix", `<gno-buttonx x`},
		{"control: markdown link", `[a](/r/x) `},
	}
	for _, s := range shapes {
		src := []byte("a " + strings.Repeat(s.item, size/len(s.item)))
		d := b1Render(t, src)
		t.Logf("%-32s attempt reads %4d bytes; %d bytes rendered in %v", s.name, b1AttemptRead(s.item), len(src), d)
	}
}
