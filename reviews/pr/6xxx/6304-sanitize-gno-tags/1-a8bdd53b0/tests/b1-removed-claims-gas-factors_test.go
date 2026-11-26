// Probe of the ADR claim (gno.land/adr/pr6304_sanitize_gno_tags.md, Decision 7
// and Consequences "Gas") that the EscapeBlockHazards slope x2.45 and the
// EscapeBlockHazardsRich slope x2.95 in gnovm/stdlibs/native_gas.go cover the
// slowest input shape, measured against the old native's calibration shape
// `[a]: u\n(x\n` at 100 KB. It also times the "Not addressed here" shape the
// ADR says costs the same before and after.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6304/head && git checkout a8bdd53b08e988227136f8b74f4cd6571956e5f4
//	mkdir -p gnovm/stdlibs/chain/markdown/zzold
//	git show d25b308c1a22ec04f429a2826e681249126320cd:gnovm/stdlibs/chain/markdown/markdown.go \
//	  | sed '0,/^package markdown/s//package zzold/' > gnovm/stdlibs/chain/markdown/zzold/markdown.go
//	cp <this file> gnovm/stdlibs/chain/markdown/zz_gasprobe_test.go
//	go test ./gnovm/stdlibs/chain/markdown -run TestGasProbe -v -count=1
//
// Each row prints new(shape) / old(calibration shape), per input byte, the
// figure the slope factor has to cover: above 2.45 (strict) or 2.95 (Rich) the
// row is charged less gas per nanosecond than the calibration shape was.
package markdown

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/gnovm/stdlibs/chain/markdown/zzold"
)

const probeN = 100_000

func fill(n int, shape string) string {
	var b strings.Builder
	for b.Len() < n {
		room := n - b.Len()
		if room >= len(shape) {
			b.WriteString(shape)
		} else {
			b.WriteString(shape[:room])
		}
	}
	return b.String()
}

func prefixThenFill(prefix string, n int, tail string) string {
	return prefix + fill(n-len(prefix), tail)
}

// best returns the fastest of reps calls, in ns per input byte.
func best(f func(string) string, s string, reps int) float64 {
	min := time.Duration(1 << 62)
	for i := 0; i < reps; i++ {
		runtime.GC()
		t0 := time.Now()
		_ = f(s)
		if d := time.Since(t0); d < min {
			min = d
		}
	}
	return float64(min.Nanoseconds()) / float64(len(s))
}

func TestGasProbe(t *testing.T) {
	cal := fill(probeN, "[a]: u\n(x\n")
	k := 316 // about sqrt(n) candidates, per the ADR's "Not addressed here"
	shapes := []struct{ name, s string }{
		{"calibration [a]: u\\n(x\\n", cal},
		{"author strict: 16x lrd-title then `\\n", prefixThenFill(strings.Repeat("[a]: u\n(x\n", 16), probeN, "`\n")},
		{"author rich: 75% fence-dedent + 64x lrd-title + `\\n", func() string {
			p := fill(probeN*3/4, "  ```\nx\n") + strings.Repeat("[a]: u\n(x\n", 64)
			return prefixThenFill(p, probeN, "`\n")
		}()},
		{"blank lines", fill(probeN, "\n")},
		{"quote markers then [", fill(probeN, "> > > > > > > > > > > > > > > [\n")},
		{"list markers then [", fill(probeN, "- - - - - - - - - - - - - - - [\n")},
		{"fence body of lrd lines", prefixThenFill("```\n", probeN, "[a]: u\n")},
		{"fence body of alert lines", prefixThenFill("```\n", probeN, "> [!x\n")},
		{"fence body of [ lines", prefixThenFill("```\n", probeN, "[a\n")},
		{"html tag then fence pair", fill(probeN, "<a\n```\n```\n")},
		{"rich dedent 3/1", fill(probeN, "   ```\n x\n")},
		{"icon in list", fill(probeN, "- <gno-icon />\n")},
		{"gno tag lines", fill(probeN, "<gno-a\n")},
		{"empty item + fence", fill(probeN, "*\n  ```\n")},
		{"list lrd title", fill(probeN, "- [a]: u\n  (x\n")},
		{"quote lrd title", fill(probeN, "> [a]: u\n> (x\n")},
		{"lrd title then [", fill(probeN, "[a]: u\n(x\n[\n")},
		{"lrd title in fence pairs", fill(probeN, "[a]: u\n(x\n```\n```\n")},
		{"lrd dq title", fill(probeN, "[a]: u\n\"x\n")},
		{"not-addressed: sqrt(n) lrd-title then ---", prefixThenFill(strings.Repeat("[a]: u\n(x\n", k), probeN, "-")},
	}
	modes := []struct {
		name     string
		old, new func(string) string
		factor   float64
	}{
		{"strict", zzold.EscapeBlockHazards, EscapeBlockHazards, 2.45},
		{"rich", zzold.EscapeBlockHazardsRich, EscapeBlockHazardsRich, 2.95},
	}
	for _, m := range modes {
		for _, sh := range shapes {
			// Interleave: old calibration, new shape, old shape.
			var oc, ns, os float64 = 1e18, 1e18, 1e18
			for r := 0; r < 5; r++ {
				if v := best(m.old, cal, 1); v < oc {
					oc = v
				}
				if v := best(m.new, sh.s, 1); v < ns {
					ns = v
				}
				if v := best(m.old, sh.s, 1); v < os {
					os = v
				}
			}
			flag := ""
			if ns/oc > m.factor {
				flag = "  <-- above factor"
			}
			fmt.Printf("%-6s %-48s new/oldcal=%5.2f (factor %.2f) new/old(same shape)=%5.2f%s\n",
				m.name, sh.name, ns/oc, m.factor, ns/os, flag)
		}
	}
}

// TestGasProbeSweep varies the two shapes the ADR calls slowest: the share of
// `  ```\nx\n` lines and the count of `[a]: u\n(x\n` candidates, 15 interleaved
// runs each, against the old native on the calibration shape.
//
//	go test ./gnovm/stdlibs/chain/markdown -run TestGasProbeSweep -v -count=1
func TestGasProbeSweep(t *testing.T) {
	cal := fill(probeN, "[a]: u\n(x\n")
	type row struct {
		mode   string
		name   string
		s      string
		old    func(string) string
		new    func(string) string
		factor float64
	}
	var rows []row
	for _, k := range []int{4, 16, 64, 256} {
		s := prefixThenFill(strings.Repeat("[a]: u\n(x\n", k), probeN, "`\n")
		rows = append(rows, row{"strict", fmt.Sprintf("%dx lrd-title + `\\n tail", k), s, zzold.EscapeBlockHazards, EscapeBlockHazards, 2.45})
	}
	for _, pct := range []int{50, 75, 90} {
		for _, k := range []int{16, 64, 256} {
			p := fill(probeN*pct/100, "  ```\nx\n") + strings.Repeat("[a]: u\n(x\n", k)
			s := prefixThenFill(p, probeN, "`\n")
			rows = append(rows, row{"rich", fmt.Sprintf("%d%% fence-dedent + %dx lrd-title + `\\n", pct, k), s, zzold.EscapeBlockHazardsRich, EscapeBlockHazardsRich, 2.95})
		}
	}
	for _, r := range rows {
		var oc, ns float64 = 1e18, 1e18
		for i := 0; i < 15; i++ {
			if v := best(r.old, cal, 1); v < oc {
				oc = v
			}
			if v := best(r.new, r.s, 1); v < ns {
				ns = v
			}
		}
		flag := ""
		if ns/oc > r.factor {
			flag = "  <-- above factor"
		}
		fmt.Printf("%-6s %-44s new/oldcal=%5.2f (factor %.2f)%s\n", r.mode, r.name, ns/oc, r.factor, flag)
	}
}

// TestGasProbeLongFence lengthens the fence run of the Rich shape: the closer
// closeFencesAtDedent writes repeats the opener's indent and run, so the input
// the walker's 8*len(s) budget is taken from grows toward 2x. It also times
// the walker on 1 and 2 MiB of spaces (ADR Decision 1: quadratic before).
//
//	go test ./gnovm/stdlibs/chain/markdown -run TestGasProbeLongFence -v -count=1
func TestGasProbeLongFence(t *testing.T) {
	cal := fill(probeN, "[a]: u\n(x\n")
	for _, run := range []int{3, 12, 30} {
		for _, k := range []int{64, 256} {
			opener := "  " + strings.Repeat("`", run) + "\nx\n"
			p := fill(probeN*3/4, opener) + strings.Repeat("[a]: u\n(x\n", k)
			s := prefixThenFill(p, probeN, "`\n")
			var oc, ns float64 = 1e18, 1e18
			for i := 0; i < 15; i++ {
				if v := best(zzold.EscapeBlockHazardsRich, cal, 1); v < oc {
					oc = v
				}
				if v := best(EscapeBlockHazardsRich, s, 1); v < ns {
					ns = v
				}
			}
			grown := len(closeFencesAtDedent(s))
			fmt.Printf("rich   75%% fence run %2d + %3dx lrd-title: grown %.2fx  new/oldcal=%5.2f (factor 2.95)\n",
				run, k, float64(grown)/float64(len(s)), ns/oc)
		}
	}
	for _, n := range []int{1 << 20, 2 << 20} {
		sp := strings.Repeat(" ", n)
		t0 := time.Now()
		_ = EscapeBlockHazards(sp)
		fmt.Printf("strict %d MiB of spaces: %v\n", n>>20, time.Since(t0))
	}
}
