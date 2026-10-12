// Repro: a close tag scans the whole details stack, refused openers
// included, for an opener of its own parent. Openers in a blockquote and
// closes outside it never match, so n of each cost n*n parent compares.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout 5ad81a10214a0379b6099159e98431176732d1f7
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_close_scan_test.go
//	cd gno.land/pkg/gnoweb/markdown
//	go test -run 'TestDetailsCloseScan' -count=1 -v .
//
// Expected at 5ad81a102: cross_parent time roughly x4 per doubling of n,
// about 1 s at n=32000 (960 KB, under the 1 MiB render cap) against about
// 25 ms for same_parent; the test FAILs on the ratio.
package markdown

import (
	"strings"
	"testing"
	"time"
)

func TestDetailsCloseScan(t *testing.T) {
	var same, cross time.Duration
	for _, n := range []int{4000, 8000, 16000, 32000} {
		for _, shape := range []string{"same_parent", "cross_parent"} {
			open := "<gno-details>\n"
			if shape == "cross_parent" {
				open = "><gno-details>\n"
			}
			src := strings.Repeat(open, n) + strings.Repeat("</gno-details>\n", n)
			start := time.Now()
			parseDetails(t, src)
			d := time.Since(start)
			t.Logf("n=%d shape=%s bytes=%d elapsed=%v", n, shape, len(src), d)
			if n == 32000 {
				if shape == "same_parent" {
					same = d
				} else {
					cross = d
				}
			}
		}
	}
	if cross > 5*same {
		t.Errorf("cross_parent %v is %.0fx same_parent %v at n=32000", cross, float64(cross)/float64(same), same)
	}
}
