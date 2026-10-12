// Repro: detailsParser.Continue walks the whole last-descendant chain on every
// line, closed nodes included. A details holding a deep (closed) blockquote
// followed by blank lines pays the chain depth once per blank line, so render
// time grows with D*N. Measured at c05644643: D=40000 N=200000 (240 KB) takes
// 23 s against 0.56 s for the same bytes without the details opener; at
// 8ab095f49 both take 0.56 s.
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/5593/head && git checkout c05644643
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_walk_cost_test.go
//   go test ./gno.land/pkg/gnoweb/markdown -run TestZZWalkCost -v   # FAIL at c05644643
//   git checkout 8ab095f49 -- gno.land/pkg/gnoweb/markdown/ext_details.go
//   go test ./gno.land/pkg/gnoweb/markdown -run TestZZWalkCost -v   # PASS before 87aa65612
//   git checkout c05644643 -- gno.land/pkg/gnoweb/markdown/ext_details.go
package markdown

import (
	"strings"
	"testing"
	"time"
)

func TestZZWalkCost(t *testing.T) {
	const d, n = 20000, 100000
	deep := strings.Repeat(">", d) + "\n" + strings.Repeat("\n", n)
	t0 := time.Now()
	parseDetails(t, deep)
	plain := time.Since(t0)
	t0 = time.Now()
	parseDetails(t, ":::details a\n"+deep+":::\n")
	details := time.Since(t0)
	t.Logf("D=%d N=%d plain=%v details=%v", d, n, plain, details)
	if details > 10*plain {
		t.Errorf("details parse %v is over 10x the plain parse %v", details, plain)
	}
}
