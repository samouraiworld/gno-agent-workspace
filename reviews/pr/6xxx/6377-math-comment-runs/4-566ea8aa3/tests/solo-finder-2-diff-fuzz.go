// Differential fuzz of the gnoweb MathML converter: the round-4 snapshot
// 566ea8aa3 against the PR head c30e596a8 (the round-4 rewrite of cellEnd
// into a shared distance table must change no output) and against the merge
// base 6c49f5420 (every input the base renders without error must render the
// same, or the difference is a behaviour change of the PR).
//
// Repro, from a checkout of gnolang/gno at 566ea8aa3 (the round's snapshot
// over pull/6377/head c30e596a8):
//
//	P=gno.land/pkg/gnoweb/markdown/mathml
//	for d in zzprev:c30e596a819207b6da8d197a166f47288f57eb22 zzbase:6c49f5420ec9c8ad0933b6869461656f889a9a37; do
//	  n=${d%%:*}; s=${d#*:}; mkdir -p $P/$n
//	  for f in $(git ls-tree --name-only $s $P/ | grep '\.go$' | grep -v _test.go); do
//	    git show $s:$f | sed "s/^package mathml$/package $n/" > $P/$n/$(basename $f)
//	  done
//	done
//	cp solo-finder-2-diff-fuzz.go $P/zz_diff_fuzz_test.go
//	ZZN=200000 go test -count=1 -run TestZZDiffFuzz -v ./$P/
package mathml

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown/mathml/zzbase"
	"github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown/mathml/zzprev"
)

var zzFlat = []string{
	`\bf`, `\rm`, `\it`, `\large`, `\displaystyle`, `\color`, `\color{red}`, `\color[RGB]`,
	`{red}`, `[RGB]`, `[`, `]`, `{`, `}`, `&`, `\\`, `\cr`, `\\[2pt]`,
	`\begin{matrix}`, `\end{matrix}`, `\left(`, `\right)`, `\left.`, `\right.`,
	`\middle|`, `\big(`, `\bigl(`, `\bigr)`, `\left`, `\right`, `\over`,
	"%c\n", " ", "  %\n", "x", "1", "^", "_", `\frac`, `\hat`, `\mathrm`,
	`\class`, `\sqrt`, `\text{a}`, `\kern1em`, `\{`, `\}`, `\&`, `\sideset`, `{}`,
}

// zzWide is a second alphabet, ZZALPHA=wide: commands, environments and
// switches the first one leaves out.
var zzWide = []string{
	`\Huge`, `\tiny`, `\textstyle`, `\scriptstyle`, `\text{ a }`, `\mbox{b}`, `\operatorname{argmax}`,
	`\begin{array}{cc}`, `\end{array}`, `\begin{cases}`, `\end{cases}`, `\begin{aligned}`, `\end{aligned}`,
	`\newcommand{\f}{x}`, `\f`, `\overset`, `\underset`, `\stackrel`, `\binom`, `\tfrac`, `\boxed`,
	`\raisebox{1pt}`, `\phantom`, `\mathbb`, `\boldsymbol`, `\vec`, `\overline`, `\underbrace`,
	`\color{#ff0000}`, `\color{}`, `\textcolor{red}`, `\limits`, `\nolimits`, `\sum`, `\int`, `\lim`,
	`\mathop`, `\bmod`, `\pmod`, `\mkern3mu`, `\quad`, `\,`, `\!`, `\ `, `\#`, `\tag{1}`, `\hline`,
	`\substack`, `\middle\|`, `\left\{`, `\right\}`, `\left\langle`, `\right\rangle`, `\Bigl(`, `\Biggr]`,
	`'`, `!`, `<`, `\bf`, `\color`, `\color[rgb]{1,0,0}`, `&`, `\\`, `\cr`, "%\n", "  %\n", "x", "2", "^", "_",
	`\class{a}`, `{}`, `\sqrt[3]`, `\left.`, `\right|`, `\over`, `\choose`, `\atop`,
}

// zzBalanced drops the units that unbalance a group, so most inputs
// tokenize and reach the parser.
var zzBalanced = func() []string {
	var out []string
	for _, u := range zzFlat {
		switch u {
		case `{`, `}`, `\begin{matrix}`, `\end{matrix}`, `\left(`, `\right)`, `\left.`, `\right.`, `\middle|`, `\left`, `\right`:
			continue
		}
		out = append(out, u)
	}
	return out
}()

var zzUnits = zzFlat

func zzGen(r *rand.Rand, depth int) string {
	var sb strings.Builder
	n := r.Intn(6)
	for k := 0; k < n; k++ {
		if depth > 0 && r.Intn(3) == 0 {
			inner := zzGen(r, depth-1)
			switch r.Intn(7) {
			case 0:
				sb.WriteString("{" + inner + "}")
			case 1:
				sb.WriteString(`\begin{matrix}` + inner + `\end{matrix}`)
			case 2:
				sb.WriteString(`\left(` + inner + `\right)`)
			case 3:
				sb.WriteString(`\color{red}` + inner)
			case 4:
				sb.WriteString("[" + inner + "]")
			case 5:
				sb.WriteString(`\left.` + inner + `\middle|` + zzGen(r, depth-1) + `\right.`)
			case 6:
				sb.WriteString(`\color[RGB]{1,2,3}` + inner + "&" + zzGen(r, depth-1))
			}
		} else {
			sb.WriteString(zzUnits[r.Intn(len(zzUnits))])
		}
	}
	return sb.String()
}

func zzErr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestZZDiffFuzz(t *testing.T) {
	n := 50000
	if s := os.Getenv("ZZN"); s != "" {
		n, _ = strconv.Atoi(s)
	}
	r := rand.New(rand.NewSource(1))
	examples := map[string][]string{}
	counts := map[string]int{}
	note := func(cat, tex string) {
		counts[cat]++
		ex := examples[cat]
		if len(ex) < 6 {
			examples[cat] = append(ex, tex)
			return
		}
		// keep the shortest examples
		for i, e := range ex {
			if len(tex) < len(e) {
				ex[i] = tex
				return
			}
		}
	}
	if os.Getenv("ZZALPHA") == "wide" {
		zzFlat = zzWide
	}
	for i := 0; i < n; i++ {
		zzUnits = zzFlat
		if i%2 == 1 {
			zzUnits = zzBalanced
		}
		tex := zzGen(r, 4)
		for _, display := range []bool{false, true} {
			var h, p, b string
			var he, pe, be error
			if display {
				h, he = NewMathMLConverter().ConvertDisplay(tex)
				p, pe = zzprev.NewMathMLConverter().ConvertDisplay(tex)
				b, be = zzbase.NewMathMLConverter().ConvertDisplay(tex)
			} else {
				h, he = NewMathMLConverter().ConvertInline(tex)
				p, pe = zzprev.NewMathMLConverter().ConvertInline(tex)
				b, be = zzbase.NewMathMLConverter().ConvertInline(tex)
			}
			if zzErr(he) == "MathML encountered an unexpected error" {
				note("head crash", tex)
			}
			if h != p || zzErr(he) != zzErr(pe) {
				note("head!=prev", tex)
			}
			switch {
			case be == nil && he != nil:
				note("base ok, head error: "+zzErr(he), tex)
			case be != nil && he == nil:
				note("base error ("+zzErr(be)+"), head ok", tex)
			case be == nil && he == nil && b != h:
				note("both ok, output differs", tex)
			case be == nil && he == nil:
				counts["both ok, same output"]++
			default:
				counts["both error"]++
			}
		}
	}
	for cat, c := range counts {
		t.Logf("%-60s %d", cat, c)
		for _, e := range examples[cat] {
			t.Logf("    %q", e)
		}
	}
	if counts["head!=prev"] > 0 {
		t.Errorf("round-4 rewrite changed output on %d inputs", counts["head!=prev"])
	}
}

// The cost of a run of switches before a long comment run: each switch's
// sub-buffer scans the trailing comments once in NewTokenBuffer.
//
//	go test -count=1 -run TestZZSwitchCommentRun -v ./$P/
func TestZZSwitchCommentRun(t *testing.T) {
	tail := strings.Repeat("%\n", 1<<18) + "x"
	for _, k := range []int{1, 64, 255} {
		tex := strings.Repeat(`\bf`, k) + "y" + tail
		start := testingNow()
		_, err := NewMathMLConverter().ConvertInline(tex)
		d := testingNow().Sub(start)
		startB := testingNow()
		_, errB := zzbase.NewMathMLConverter().ConvertInline(tex)
		dB := testingNow().Sub(startB)
		t.Logf("switches=%d head=%v err=%v base=%v err=%v", k, d, err, dB, errB)
	}
}

func testingNow() time.Time { return time.Now() }

// Candidate 1001: a \left whose delimiter draws nothing loses its group,
// where \left. keeps it.
//
//	go test -count=1 -run TestZZLeftNil -v ./$P/
func TestZZLeftNil(t *testing.T) {
	for _, tex := range []string{`\left\mathrm{} a \right) b`, `\left. a \right) b`, `\left\class{a}{} a \right) b`} {
		h, he := NewMathMLConverter().ConvertInline(tex)
		b, be := zzbase.NewMathMLConverter().ConvertInline(tex)
		if i := strings.Index(h, "<annotation"); i >= 0 {
			h = h[:i]
		}
		t.Logf("%q\n  head err=%v %s\n  base err=%v %.120s", tex, he, h, be, b)
	}
}
