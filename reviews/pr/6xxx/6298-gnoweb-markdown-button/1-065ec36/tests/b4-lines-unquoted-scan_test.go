// b4-lines-unquoted-scan: an unquoted attribute value in scanGnoTag runs
// through '<', so on a line of `<gno-button a=` repeats every '<' the inline
// parser tries reads up to maxButtonTagLen bytes instead of stopping at the
// next tag, as the "each attempt reads one tag" comment at utils.go:72 says.
//
// Repro from a plain clone of gnolang/gno at 065ec369b (PR #6298):
//
//	git fetch origin pull/6298/head && git checkout --detach 065ec369b321c207815f82b0c1dbf2b0f9f10b01
//	cp <this file> gno.land/pkg/gnoweb/markdown/b4_lines_unquoted_scan_test.go
//	go test ./gno.land/pkg/gnoweb/markdown -run 'TestB4UnquotedScan' -count=1 -v
//	rm gno.land/pkg/gnoweb/markdown/b4_lines_unquoted_scan_test.go
//
// scanGnoTag is in gno.land/pkg/gnoweb/markdown/utils.go, carried by
// gnolang/gno#6297, #6298 and #6299 alike.
package markdown

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func b4Render(t *testing.T, input []byte) (string, time.Duration) {
	t.Helper()
	gnourl, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatal(err)
	}
	ctx := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))
	m := goldmark.New()
	NewGnoExtension().Extend(m)
	start := time.Now()
	node := m.Parser().Parse(text.NewReader(input), ctx)
	var out bytes.Buffer
	if err := m.Renderer().Render(&out, input, node); err != nil {
		t.Fatal(err)
	}
	return out.String(), time.Since(start)
}

// The scan crosses '<' inside an unquoted value: one attempt reads two tags.
func TestB4UnquotedScanCrossesTag(t *testing.T) {
	out, _ := b4Render(t, []byte(`<gno-button href=/r/x label=a<b />`+"\n"))
	t.Logf("output: %s", out)
	n, _ := parseButtonTag([]byte(`<gno-button a=<gno-button href="/r/x" label="y" />`))
	t.Logf("parseButtonTag size on `<gno-button a=<gno-button ...>`: %d (0 means the attempt stopped at the inner tag)", n)
}

// Time per shape, same byte count; prefix-miss is the baseline every '<' pays.
func TestB4UnquotedScanCost(t *testing.T) {
	const size = 256 << 10
	shapes := []struct{ name, unit string }{
		{"prefix-miss  `<gno-buttox a=`", "<gno-buttox a="},
		{"quoted       `<gno-button a=\"x\" `", `<gno-button a="x" `},
		{"unquoted     `<gno-button a=`", "<gno-button a="},
	}
	for _, s := range shapes {
		in := []byte("p " + strings.Repeat(s.unit, size/len(s.unit)) + "\n")
		_, d := b4Render(t, in)
		t.Logf("%-36s %7d bytes  %v", s.name, len(in), d)
	}
}
