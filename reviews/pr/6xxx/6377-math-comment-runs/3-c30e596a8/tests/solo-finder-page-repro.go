// Repro: the three claimed defects through NewGnoExtension. From a plain clone at
// c30e596a8: cp this file to gno.land/pkg/gnoweb/markdown/zz_page_test.go and run
//   go test -count=1 -run TestZZPage -v ./gno.land/pkg/gnoweb/markdown/
// then `git checkout 6c49f5420ec9c8ad0933b6869461656f889a9a37 -- gno.land/pkg/gnoweb/markdown/mathml` and run again.
// Measured (best of 5): crash inputs FALLBACK to raw source at master, CONVERTED at head;
// bare \color k=170/341/682/1365: master 0.30/0.75/2.21/6.54 ms, head 0.25/0.58/0.47/0.98 ms;
// indented % lines k=256/512/1024/2040: master 0.35/1.04/3.45/12.3 ms, head 0.14/0.26/0.50/1.17 ms;
// 1 MiB page of 128 bare-\color expressions: master 176 ms, head 48 ms, 19 converted (page budget).
package markdown

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/yuin/goldmark"
)

func zzRender(src string) (string, time.Duration) {
	var buf bytes.Buffer
	gm := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
	best := time.Hour
	for r := 0; r < 5; r++ {
		buf.Reset()
		t0 := time.Now()
		if err := gm.Convert([]byte(src), &buf); err != nil {
			panic(err)
		}
		best = min(best, time.Since(t0))
	}
	return buf.String(), best
}

func TestZZPageCrash(t *testing.T) {
	for _, tex := range []string{`$\bf\sqrt[a&b]x$`, `$x\bigl\mathrm$`, `$x\bigr\mathrm$`, `$\middle\mathrm$`, `$\class{a}{}$`, "$$\\begin{matrix}\\bf\\sqrt[a&b]x\\end{matrix}$$"} {
		out, _ := zzRender("Text " + tex + " end.\n")
		state := "CONVERTED"
		if !strings.Contains(out, "<math") {
			state = "FALLBACK"
		}
		t.Logf("%-10s %q -> %.200s", state, tex, out)
	}
}

func TestZZPageColor(t *testing.T) {
	for _, k := range []int{170, 341, 682, 1365} {
		src := "$" + strings.Repeat(`\color`, k) + "x$\n"
		out, d := zzRender(src)
		t.Logf("bare color k=%d bytes=%d converted=%v time=%v", k, len(src), strings.Contains(out, "<math"), d)
	}
	for _, k := range []int{256, 512, 1024, 2040} {
		src := "$$x\n" + strings.Repeat("  %\n", k) + "y$$\n"
		out, d := zzRender(src)
		t.Logf("indented comment lines k=%d bytes=%d converted=%v time=%v", k, len(src), strings.Contains(out, "<math"), d)
	}
	page := strings.Repeat("$"+strings.Repeat(`\color`, 1365)+"x$\n\n", 128)
	out, d := zzRender(page)
	t.Logf("page of 128 bare-color expressions bytes=%d converted=%d time=%v", len(page), strings.Count(out, "<math"), d)
}
