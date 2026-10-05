// Repro: an unclosed inline `$` opener scans the rest of its line (and the next
// line) for a closer, so a line of N openers costs O(N * line) in gnoweb.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/4879/head && git checkout --detach 0193f6de7fc5823e3c01d4a5e580e377cde98659
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_quad_repro_test.go
//   go test -v -count=1 -timeout 600s -run TestInlineDollarQuadratic ./gno.land/pkg/gnoweb/markdown/
//   rm gno.land/pkg/gnoweb/markdown/zz_quad_repro_test.go
// Merge base (no math extension), same file, same command:
//   git checkout --detach 87f0357fe2b373476bb92a26d59833402b9916d3
//
// Observed at 0193f6de7 (go1.25.9, linux/amd64):
//   n=8192   input=24576    time=34.6ms
//   n=16384  input=49152    time=144.3ms
//   n=32768  input=98304    time=580.4ms
//   n=349525 input=1048575  time=59.78s   (1 MiB = gnoweb's maxMarkdownRenderBytes)
//   512 KiB of openers + a 512 KiB next line: 37.2s
//   `\\(a ` units (never closed): 16384 -> 57ms, 32768 -> 207ms (same 4x per doubling)
// Observed at merge base 87f0357fe:
//   n=8192 1.2ms, n=16384 1.9ms, n=32768 4.0ms, n=349525 40.5ms
package markdown

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/yuin/goldmark"
)

func TestInlineDollarQuadratic(t *testing.T) {
	for _, n := range []int{1 << 13, 1 << 14, 1 << 15, (1 << 20) / 3} {
		src := strings.Repeat("$a ", n)
		var buf bytes.Buffer
		gm := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
		start := time.Now()
		_ = gm.Convert([]byte(src), &buf)
		t.Logf("n=%-7d input=%-8d time=%v", n, len(src), time.Since(start))
	}
}
