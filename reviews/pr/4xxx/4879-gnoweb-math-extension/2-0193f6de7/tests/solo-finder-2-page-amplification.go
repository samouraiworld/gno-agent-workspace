// Repro: the 8 KiB per-expression cap bounds one expression, not a page. An
// aligned environment of bare `&` renders at ~107x, past FuzzMathRender's own
// 64x+4096 invariant, and a 1 MiB page (gnoweb's maxMarkdownRenderBytes) of
// such expressions renders to ~112 MB of HTML.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/4879/head && git checkout --detach 0193f6de7fc5823e3c01d4a5e580e377cde98659
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_amp_repro_test.go
//   go test -v -count=1 -run TestPageAmplification ./gno.land/pkg/gnoweb/markdown/
//   rm gno.land/pkg/gnoweb/markdown/zz_amp_repro_test.go
// The PR's own fuzz target fails on the same input as a corpus entry:
//   mkdir -p gno.land/pkg/gnoweb/markdown/testdata/fuzz/FuzzMathRender
//   python3 -c 'open("gno.land/pkg/gnoweb/markdown/testdata/fuzz/FuzzMathRender/aligned_amp","w").write("go test fuzz v1\nstring(\"\\\\begin{aligned}" + "&"*8150 + "\\\\end{aligned}\")\n")'
//   go test -count=1 -run FuzzMathRender ./gno.land/pkg/gnoweb/markdown/
//   -> ext_math_test.go:137: output too large: 872458 bytes for 8180 bytes of input
//
// Observed at 0193f6de7 (go1.25.9):
//   aligned-amp      exprs=128   in=1048576  out=111811584  ratio=107  alloc=2645MB time=1.41s
//   pmatrix-amp      exprs=128   in=1048576  out=56463360   ratio=54   alloc=1826MB time=0.94s
//   letters          exprs=128   in=1044992  out=18834048   ratio=18   alloc=797MB  time=0.35s
//   blockquote-base  (no math)   in=1044990  out=28200960   ratio=27   alloc=252MB  time=1.50s
package markdown

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yuin/goldmark"
)

func ampPage(t *testing.T, name, para string) {
	src := strings.Repeat(para, (1<<20)/len(para))
	runtime.GC()
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	var buf bytes.Buffer
	gm := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
	start := time.Now()
	_ = gm.Convert([]byte(src), &buf)
	d := time.Since(start)
	runtime.ReadMemStats(&m1)
	t.Logf("%-16s in=%-8d out=%-10d ratio=%-4.0f alloc=%dMB time=%v", name, len(src), buf.Len(),
		float64(buf.Len())/float64(len(src)), (m1.TotalAlloc-m0.TotalAlloc)>>20, d)
}

func TestPageAmplification(t *testing.T) {
	amp := MaxMathInputLen - 32
	ampPage(t, "aligned-amp", "$\\begin{aligned}"+strings.Repeat("&", amp)+"\\end{aligned}$\n\n")
	ampPage(t, "pmatrix-amp", "$\\begin{pmatrix}"+strings.Repeat("&", amp)+"\\end{pmatrix}$\n\n")
	ampPage(t, "letters", "$"+strings.Repeat("a", amp)+"$\n\n")
	ampPage(t, "blockquote-base", strings.Repeat(">", 4096)+"\n\n")
}
