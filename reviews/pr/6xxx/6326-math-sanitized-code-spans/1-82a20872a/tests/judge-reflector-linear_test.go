// Reflector probe: render time of 1 MiB lines of unclosed openers the
// linear-scan test does not list, at three sizes, to read the growth.
// Run: cp into gno.land/pkg/gnoweb/markdown/, then from gno.land:
//   go test ./pkg/gnoweb/markdown -run TestZZReflectorLinear -count=1 -v
package markdown

import (
	"strings"
	"testing"
	"time"
)

func TestZZReflectorLinear(t *testing.T) {
	for _, unit := range []string{"$$a ", "[$$a ](", `[\\[a ](`, `\\([a](x `, `$$[a](x `} {
		for _, n := range []int{1 << 16, 1 << 18, 1 << 20} {
			src := strings.Repeat(unit, n/len(unit))
			start := time.Now()
			out := renderMathMarkdown(t, src)
			t.Logf("%-10q %8d bytes %8.3fs math=%v", unit, len(src), time.Since(start).Seconds(), strings.Contains(out, "<math"))
		}
	}
}
