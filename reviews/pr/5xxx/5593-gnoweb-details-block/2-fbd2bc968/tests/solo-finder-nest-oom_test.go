// Repro: :::details skips the shared nesting cap (nestdepth.go Push/Pop), so
// a run of opener lines nests without bound and goldmark's per-line
// bookkeeping (parser.go:1106, one lineStat per open block per line, reset
// only on a blank line) grows with lines x depth: memory and time are
// quadratic in the input size.
//
// From a plain clone of gnolang/gno at fbd2bc9687ddebeef9f09c16032068fcca159696:
//
//	cp <this file> gno.land/pkg/gnoweb/markdown/solo_nest_oom_test.go
//	go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloDetailsNestBounded -v -count=1
//
// Measured with go1.25.9 under `ulimit -d 6427937` (6.1 GiB data):
//
//	head fbd2bc968, n=2500 (32 KB):  details=2500  alloc=405 MiB   0.16 s
//	head fbd2bc968, n=5000 (65 KB):  details=5000  alloc=1546 MiB  0.56 s
//	head fbd2bc968, n=10000 (130 KB): details=10000 alloc=5893 MiB 2.27 s
//	head fbd2bc968, n=14000 (182 KB): fatal error: runtime: out of memory
//	base 02606904e, n=80000 (1.04 MB): details=0 alloc=53 MiB 0.04 s
//	head + Push/Pop in Open/Close, n=80000: details=4 alloc=96 MiB 0.06 s
//
// gnoweb's only input cap is maxMarkdownRenderBytes = 1 MiB (render.go:29),
// so a realm Render output of ~180 KB is enough to exhaust 6 GiB.
//
// At head this test FAILS (details=4000, alloc ~0.9 GiB); with Push(pc) after
// parseOpenFence and Pop(pc) in Close it passes.
package markdown

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func soloNestRender(t *testing.T, src string) string {
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

func TestSoloDetailsNestBounded(t *testing.T) {
	const n = 4000 // 52 KB of input
	src := strings.Repeat(":::details a\n", n)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	out := soloNestRender(t, src)
	runtime.ReadMemStats(&after)
	alloc := (after.TotalAlloc - before.TotalAlloc) >> 20
	depth := strings.Count(out, "<details")
	t.Logf("n=%d bytes=%d details=%d alloc_MiB=%d", n, len(src), depth, alloc)
	if depth > MaxGnoNestDepth {
		t.Errorf("details nested %d deep, want at most MaxGnoNestDepth=%d", depth, MaxGnoNestDepth)
	}
	if alloc > 64 {
		t.Errorf("rendering %d bytes allocated %d MiB, want under 64 MiB", len(src), alloc)
	}
}
