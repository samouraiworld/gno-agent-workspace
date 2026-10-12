// Repro: render shapes for the :::details parser (fence/HTML inside, equal and
// inverted nesting, <gno-foreign>, unclosed in <gno-columns>, CRLF, alert, depth).
// From a plain clone at fbd2bc968:
//	cp <this file> gno.land/pkg/gnoweb/markdown/solo_shapes_test.go
//	go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloProbe -v -count=1
//	SOLO_N=2500,5000,10000 go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloProbeQuadratic -v -count=1
package markdown

import (
	"bytes"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// soloRender renders src with the production gnoweb extension chain,
// the same setup ext_test.go uses for the golden files.
func soloRender(t *testing.T, src string) string {
	t.Helper()
	u, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatal(err)
	}
	ctx := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: u}))
	m := goldmark.New()
	NewGnoExtension().Extend(m)
	in := []byte(src)
	node := m.Parser().Parse(text.NewReader(in), ctx)
	var b bytes.Buffer
	if err := m.Renderer().Render(&b, in, node); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestSoloProbeShapes(t *testing.T) {
	shapes := []struct{ name, src string }{
		{"fence_inside", ":::details A\n```\n:::\n```\nafter\n:::\n\n## Footer\n"},
		{"html_inside", ":::details A\n<div>\n:::\n</div>\nafter\n:::\n\n## Footer\n"},
		{"equal_nest", ":::details A\n:::details B\nx\n:::\ny\n:::\n"},
		{"longer_inner", ":::details O\n::::details I\nx\n::::\ny\n:::\n"},
		{"foreign_details", "before\n\n<gno-foreign>\n:::details X\nbody\n:::\n</gno-foreign>\n\nafter\n"},
		{"columns_details_unclosed", "<gno-columns>\n:::details A\nleft\n</gno-columns>\n\n## Footer\n"},
		{"html_line_then_close", ":::details A\n<div>x</div>\n:::\n\n## Footer\n"},
		{"crlf", ":::details A\r\nbody\r\n:::\r\nafter\r\n"},
		{"alert_details", "> [!NOTE]\n> :::details A\n> body\n> :::\n"},
		{"depth5_then_alert", "::::::::details 1\n:::::::details 2\n::::::details 3\n:::::details 4\n::::details 5\n> [!NOTE]\n> deep alert\n"},
	}
	for _, s := range shapes {
		t.Logf("=== %s\n--- input\n%s--- output\n%s", s.name, s.src, soloRender(t, s.src))
	}
}

// SOLO_N is a comma-separated list of line counts, default 2500,5000,10000.
func TestSoloProbeQuadratic(t *testing.T) {
	ns := os.Getenv("SOLO_N")
	if ns == "" {
		ns = "2500,5000,10000"
	}
	for _, f := range strings.Split(ns, ",") {
		n, _ := strconv.Atoi(f)
		src := strings.Repeat(":::details a\n", n)
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		st := time.Now()
		out := soloRender(t, src)
		el := time.Since(st)
		runtime.ReadMemStats(&after)
		t.Logf("n=%d bytes=%d details_open=%d elapsed=%v alloc_MiB=%d heapsys_MiB=%d",
			n, len(src), strings.Count(out, "<details"), el,
			(after.TotalAlloc-before.TotalAlloc)>>20, after.HeapSys>>20)
	}
}
