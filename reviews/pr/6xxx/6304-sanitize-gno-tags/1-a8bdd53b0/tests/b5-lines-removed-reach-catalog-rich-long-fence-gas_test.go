// Gas-shape probe for chain/markdown EscapeBlockHazards{,Rich} at PR #6304 head.
//
// native_gas.go prices the two natives at the old native's fitted slope times
// the ratio of the slowest shape found to that old fit: x2.45 strict (16x
// `[a]: u\n(x\n` then a "`\n" tail) and x2.95 Rich (~75% "  ```\nx\n", 64 LRDs,
// "`\n" tail). Both factors carry no margin. The bracket walker's budget is
// 8*len(s) of the string AFTER closeFencesAtDedent, so in Rich mode a fence
// region that expands more than the calibrated 1.75x (a longer fence run gets
// a longer inserted closer, approaching 2x) and covers more of the input buys
// more title-walk steps per input byte than the calibrated shape.
//
// This test times, in one process and interleaved, the old native (merge base
// d25b308c1) on its fitted shape and the new native on the PR's slow shapes
// and on long-fence variants, and prints each shape's factor over the old fit
// shape, which is the number native_gas.go multiplies by.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6304/head && git checkout a8bdd53b08e988227136f8b74f4cd6571956e5f4
//	mkdir -p gnovm/stdlibs/chain/markdown/zzold
//	git show d25b308c1a22ec04f429a2826e681249126320cd:gnovm/stdlibs/chain/markdown/markdown.go \
//	  | sed 's/^package markdown$/package zzold/' > gnovm/stdlibs/chain/markdown/zzold/markdown.go
//	cp <this file> gnovm/stdlibs/chain/markdown/zz_gas_shapes_test.go
//	go test ./gnovm/stdlibs/chain/markdown -run TestGasShapeFactors -v -count=1
//
// A shape whose printed factor exceeds the charged one (2.45 strict, 2.95 Rich)
// runs longer than the gas it pays for.
package markdown

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	old "github.com/gnolang/gno/gnovm/stdlibs/chain/markdown/zzold"
)

const probeN = 100_000

func repeatTo(unit string, n int) string {
	return strings.Repeat(unit, n/len(unit)+1)[:n]
}

// fillStrictSlow and fillRichSlow are copied from
// gnovm/cmd/calibrate/markdown_bench_test.go at the PR head.
func fillStrictSlow(n int) string {
	prefix := strings.Repeat("[a]: u\n(x\n", 16)
	tail := strings.Repeat("`\n", (n-len(prefix)+1)/2)
	return (prefix + tail)[:n]
}

func fillRichSlow(n int) string {
	n75 := (n * 3) / 4
	part1 := strings.Repeat("  ```\nx\n", (n75+7)/8)
	if len(part1) > n75 {
		part1 = part1[:n75]
	}
	s := part1 + strings.Repeat("[a]: u\n(x\n", 64)
	tail := strings.Repeat("`\n", (n-len(s)+1)/2)
	return (s + tail)[:n]
}

// richLongFence: a fraction frac of the input is fences of run length runLen
// at indent 2, each followed by a dedented "x" line (closeFencesAtDedent
// inserts a closer as long as the opener), then enough LRDs whose paren title
// walks the "`\n" tail to spend the walker's whole 8*len(expanded) budget.
func richLongFence(n int, frac float64, runLen int) string {
	unit := "  " + strings.Repeat("`", runLen) + "\nx\n"
	part1 := strings.Repeat(unit, int(float64(n)*frac)/len(unit))
	expanded := len(closeFencesAtDedent(part1 + "\n"))
	rest := n - len(part1)
	budget := 8 * (expanded + rest)
	k := 1
	for ; 10*k < rest/2; k++ {
		if k*rest-5*k*k >= budget+budget/4 {
			break
		}
	}
	s := part1 + strings.Repeat("[a]: u\n(x\n", k)
	tail := strings.Repeat("`\n", (n-len(s)+1)/2)
	return (s + tail)[:n]
}

type probeShape struct {
	name   string
	fn     func(string) string
	in     string
	charge float64 // factor native_gas.go charges over the old fit
}

func timeOnce(fn func(string) string, in string) time.Duration {
	reps := 3
	start := time.Now()
	for i := 0; i < reps; i++ {
		_ = fn(in)
	}
	return time.Since(start) / time.Duration(reps)
}

func TestGasShapeFactors(t *testing.T) {
	if testing.Short() {
		t.Skip("timing probe")
	}
	shapes := []probeShape{
		{"old native, old fit shape", old.EscapeBlockHazards, repeatTo("[a]: u\n(x\n", probeN), 1},
		{"strict, PR slow shape", EscapeBlockHazards, fillStrictSlow(probeN), 2.45},
		{"rich, PR slow shape", EscapeBlockHazardsRich, fillRichSlow(probeN), 2.95},
		{"rich, frac=0.85 run=3", EscapeBlockHazardsRich, richLongFence(probeN, 0.85, 3), 2.95},
		{"rich, frac=0.90 run=3", EscapeBlockHazardsRich, richLongFence(probeN, 0.90, 3), 2.95},
		{"rich, frac=0.93 run=3", EscapeBlockHazardsRich, richLongFence(probeN, 0.93, 3), 2.95},
		{"rich, frac=0.90 run=30", EscapeBlockHazardsRich, richLongFence(probeN, 0.90, 30), 2.95},
	}
	// Each round times every shape once, old native first, and records each
	// shape's factor over the old native measured in that same round; the
	// median over the rounds is printed, which drifts less than a min when
	// the machine is shared.
	const rounds = 31
	factors := make([][]float64, len(shapes))
	for round := 0; round < rounds; round++ {
		base := float64(timeOnce(shapes[0].fn, shapes[0].in).Nanoseconds()) / float64(len(shapes[0].in))
		for i, sh := range shapes[1:] {
			nsb := float64(timeOnce(sh.fn, sh.in).Nanoseconds()) / float64(len(sh.in))
			factors[i+1] = append(factors[i+1], nsb/base)
		}
	}
	for i, sh := range shapes[1:] {
		f := factors[i+1]
		sort.Float64s(f)
		med := f[len(f)/2]
		verdict := "covered"
		if med > sh.charge {
			verdict = fmt.Sprintf("UNDERCHARGED by %.0f%%", (med/sh.charge-1)*100)
		}
		fmt.Printf("%-24s out=%d  factor median %.2f [p25 %.2f, p75 %.2f] vs charged %.2f  %s\n",
			sh.name, len(sh.fn(sh.in)), med, f[len(f)/4], f[3*len(f)/4], sh.charge, verdict)
	}
}
