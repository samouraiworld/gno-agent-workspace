// solo-finder, PR 6377 round 5 at 2d8efc2df: a cell's trailing comment run is walked once per enclosing switch.
//
// Repro from a plain clone of gnolang/gno:
//   git checkout 2d8efc2df67aa2f5da7d8d1b30925178691a0e18
//   mkdir -p gno.land/pkg/gnoweb/markdown/zzbase
//   for f in $(git ls-tree --name-only 6c49f5420 gno.land/pkg/gnoweb/markdown/mathml/ | grep -v _test.go | grep '\.go$'); do git show 6c49f5420:$f > gno.land/pkg/gnoweb/markdown/zzbase/$(basename $f); done
//   cp solo-finder-switch-trailing-blanks.go gno.land/pkg/gnoweb/markdown/mathml/zz_solo_test.go
//   go test -count=1 -run 'TestZZSoloSwitch' -v ./gno.land/pkg/gnoweb/markdown/mathml/
//
// Observed (16 cores, go1.27.3):
//   head switches 128 blocks of 8192 bytes: 226ms   head plain 128 blocks of 8191 bytes: 104ms
//   base switches 128 blocks of 8192 bytes: 1.118s  base plain 128 blocks of 8191 bytes: 837ms
//   head k=32768 switches=255 trailing=true: 15.396ms, trailing=false: 6.854ms; switches=0: 10.575ms / 8.146ms
// Mechanism: each nested switch cuts its cell with GetNextN -> sub -> NewTokenBuffer, whose loop at
// tokenize.go:314 walks the cell's trailing blank run again; Empty and cellEnd are O(1).
package mathml

import (

	"strings"
	"testing"
	"time"

	base "github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown/zzbase"
)

func zzHead(tex string) (string, error) { return NewMathMLConverter().ConvertInline(tex) }
func zzBase(tex string) (string, error) { return base.NewMathMLConverter().ConvertInline(tex) }

func zzStrip(s string) string {
	if i := strings.Index(s, "<annotation"); i >= 0 {
		return s[:i]
	}
	return s
}


// Timing: a run of nested switches over a cell that ends in a comment run.
func TestZZSoloSwitchTrailingComments(t *testing.T) {
	for _, k := range []int{1 << 12, 1 << 13, 1 << 14, 1 << 15} {
		for _, pre := range []string{"", strings.Repeat(`\rm`, 255)} {
			for _, post := range []string{"", "y"} {
				tex := pre + "x" + strings.Repeat("%\n", k) + post
				for _, side := range []struct {
					name string
					f    func(string) (string, error)
				}{{"head", zzHead}, {"base", zzBase}} {
					if side.name == "base" && k > 1<<13 {
						continue
					}
					start := time.Now()
					_, err := side.f(tex)
					t.Logf("%s k=%d switches=%d trailing=%v: %v err=%v", side.name, k, strings.Count(pre, `\rm`), post == "", time.Since(start).Round(time.Microsecond), err)
				}
			}
		}
	}
}

// Page-sized: 128 blocks of 8 KiB, each 255 switches then x then comments.
func TestZZSoloSwitchPage(t *testing.T) {
	pre := strings.Repeat(`\rm`, 255) + "x"
	block := pre + strings.Repeat("%\n", (8192-len(pre))/2)
	plain := "x" + strings.Repeat("%\n", (8192-1)/2)
	for _, c := range []struct{ name, tex string }{{"switches", block}, {"plain", plain}} {
		for _, side := range []struct {
			name string
			f    func(string) (string, error)
		}{{"head", zzHead}, {"base", zzBase}} {
			start := time.Now()
			for i := 0; i < 128; i++ {
				side.f(c.tex)
			}
			t.Logf("%s %s 128 blocks of %d bytes: %v", side.name, c.name, len(c.tex), time.Since(start).Round(time.Millisecond))
		}
	}
}
