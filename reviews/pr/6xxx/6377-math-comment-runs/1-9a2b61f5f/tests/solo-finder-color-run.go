// Repro: a run of bare \color commands converts in time quadratic in its
// length, at head 9a2b61f5f and at the merge base 6c49f5420 alike.
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6377/head && git checkout --detach 9a2b61f5fe9fbbdee87ba61fbbc2e3b000e7d2fb
//   cp <this file> gno.land/pkg/gnoweb/markdown/mathml/zz_color_test.go
//   go test -count=1 -run TestZZColorRun -v ./gno.land/pkg/gnoweb/markdown/mathml/
//   # base: git checkout --detach 6c49f5420ec9c8ad0933b6869461656f889a9a37, same two commands
//
// Measured (head):  8192 -> 122ms, 16384 -> 329ms, 32768 -> 1.172s
// Measured (base tokenize.go): 8192 -> 62ms, 16384 -> 219ms, 32768 -> 884ms
// Each \color with no {group} after it scans to the end of its cell
// (commands.go:325), fails, and Ungets (commands.go:348); the next \color
// scans the same tokens again. gnoweb caps one expression at
// MaxMathInputLen (8 KiB), and a 1 MiB page of 8 KiB \color blocks rendered
// 20 <math> elements in 56 ms at head (54 ms at base) before the 2 MiB output
// budget stopped conversion.
package mathml

import (
	"strings"
	"testing"
	"time"
)

func TestZZColorRun(t *testing.T) {
	for _, n := range []int{1 << 13, 1 << 14, 1 << 15} {
		tex := strings.Repeat(`\color`, n) + " x"
		t0 := time.Now()
		_, err := NewMathMLConverter().ConvertInline(tex)
		t.Logf("color x%d (%d bytes): %v err=%v", n, len(tex), time.Since(t0).Round(time.Millisecond), err != nil)
	}
}
