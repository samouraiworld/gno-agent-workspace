package mathml

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown/mathml/zzbase"
)

var zzFrags = []string{
	"x", "y", "1", " ", "  ", "\n", "%c\n", "%\n", "%", "\t",
	"{", "}", "{", "}", "[", "]", "(", ")", "|", ".", "+", "=", "&", "^", "_", "'",
	`\left`, `\right`, `\middle`, `\big`, `\bigl`, `\bigr`, `\Big`, `\Bigl`, `\Biggr`,
	`\color`, `\color{red}`, `\color[RGB]`, `\color[RGB]{1,2,3}`, `\bf`, `\rm`, `\it`, `\displaystyle`, `\scriptstyle`, `\large`, `\Huge`, `\tiny`,
	`\sqrt`, `\frac`, `\mathrm`, `\mathbf`, `\mathbb`, `\hat`, `\underline`, `\class`, `\kern`, `\mkern`, `\newcommand`, `\def`, `\foo`, `\over`, `\choose`, `\atop`,
	`\\`, `\cr`, `\begin{matrix}`, `\end{matrix}`, `\begin{array}{cc}`, `\end{array}`, `\begin{aligned}`, `\end{aligned}`, `\begin{cases}`, `\end{cases}`,
	`\text{a b}`, `\text`, `\operatorname`, `\{`, `\}`, `\lbrace`, `\langle`, `\rangle`, `\vert`, `\|`, `\mathop`, `\overset`, `\underbrace`, `\boxed`, `\phantom`,
	`\raisebox`, `\tag`, `\limits`, `\nolimits`, `\sum`, `\int`, `\lim`, `\sin`, `\,`, `\quad`, `\bmod`, `\pmod`, `\LaTeX`, `\binom`, `\stackrel`, `\not`, `\xrightarrow`,
	`\hbox`, `\mbox`, `\textcolor`, `\colorbox`, `\fbox`, `\cancel`, `\overline`, `\vec`, `\dot`, `\ddot`, `\#`, `\&`, `\%`, `#1`, `\$`,
}

func zzGen(r *rand.Rand) string {
	var sb strings.Builder
	n := 1 + r.Intn(zzL)
	for i := 0; i < n; i++ {
		sb.WriteString(zzFrags[r.Intn(len(zzFrags))])
	}
	return sb.String()
}

var zzL = func() int { if v, _ := strconv.Atoi(os.Getenv("ZZL")); v > 0 { return v }; return 14 }()

const zzCrash = "MathML encountered an unexpected error"

// zzRaw runs the converter without its recover, returning the panic site.
func zzRaw(tex string, display bool) (site string) {
	defer func() {
		if r := recover(); r != nil {
			st := string(debug.Stack())
			lines := strings.Split(st, "\n")
			var keep []string
			for _, l := range lines {
				if strings.Contains(l, "/mathml/") && !strings.Contains(l, "zz_diff_test") {
					keep = append(keep, strings.TrimSpace(l))
				}
				if len(keep) >= 3 {
					break
				}
			}
			site = fmt.Sprint(r) + " @ " + strings.Join(keep, " <- ")
		}
	}()
	c := NewMathMLConverter()
	if display {
		_, _ = c.ConvertDisplay(tex) // warm path unused; keep raw below
	}
	c = NewMathMLConverter()
	c.currentExpr = []rune(tex)
	toks, err := tokenize(c.currentExpr)
	if err != nil {
		return ""
	}
	ctx := ctxRoot
	if hasTopLevelLineBreak(toks) {
		ctx |= ctxTable
	}
	root := c.ParseTex(NewTokenBuffer(toks), ctx)
	if hasTopLevelLineBreak(toks) && root != nil {
		processTable(root, "")
	}
	ast := c.wrapInMathTag(root, tex)
	var sb strings.Builder
	ast.Write(&sb)
	return ""
}

func TestZZDiff(t *testing.T) {
	seed, _ := strconv.ParseInt(os.Getenv("ZZSEED"), 10, 64)
	n, _ := strconv.Atoi(os.Getenv("ZZN"))
	if n == 0 {
		t.Skip()
	}
	r := rand.New(rand.NewSource(seed))
	crashes, diffs, regress, baseCrash := 0, 0, 0, 0
	sites := map[string]int{}
	var slowest time.Duration
	var slowTex string
	for k := 0; k < n; k++ {
		tex := zzGen(r)
		display := k%2 == 1
		var hOut, bOut string
		var hErr, bErr error
		t0 := time.Now()
		if display {
			hOut, hErr = NewMathMLConverter().ConvertDisplay(tex)
		} else {
			hOut, hErr = NewMathMLConverter().ConvertInline(tex)
		}
		if d := time.Since(t0); d > slowest {
			slowest, slowTex = d, tex
		}
		if display {
			bOut, bErr = zzbase.NewMathMLConverter().ConvertDisplay(tex)
		} else {
			bOut, bErr = zzbase.NewMathMLConverter().ConvertInline(tex)
		}
		if bErr != nil && bErr.Error() == zzCrash {
			baseCrash++
		}
		if hErr != nil && hErr.Error() == zzCrash {
			crashes++
			s := zzRaw(tex, display)
			if sites[s] < 2 {
				t.Logf("HEAD CRASH %q: %s", tex, s)
			}
			sites[s]++
			continue
		}
		if bErr == nil && hErr != nil {
			regress++
			if regress <= 10 {
				t.Logf("REGRESS %q: head err %v", tex, hErr)
			}
			continue
		}
		if bErr == nil && hOut != bOut {
			diffs++
			if diffs <= 10 {
				t.Logf("DIFF %q\n base %s\n head %s", tex, bOut, hOut)
			}
		}
	}
	t.Logf("n=%d seed=%d baseCrash=%d headCrash=%d regress=%d diffs=%d sites=%d slowest=%v %q", n, seed, baseCrash, crashes, regress, diffs, len(sites), slowest, slowTex)
}

var zzEnvs = []string{"matrix", "pmatrix", "bmatrix", "vmatrix", "array}{cc", "aligned", "cases", "subarray}{c", "center", "pmatrix*"}
var zzDelims = []string{"(", ")", "[", "]", "|", ".", `\{`, `\}`, `\langle`, `\rangle`, `\vert`, `\|`, `\mathrm`, `\color`, `\bf`, `\class`, `\kern`, `\newcommand`, `\frac`, `\hat`, `\lbrace`, `\sideset`, "&", `\\`, "%c\n", " ", "x"}
var zzCmds = []string{`\multirow`, `\multicolumn`, `\prescript`, `\sideset`, `\textcolor`, `\frac`, `\cfrac`, `\binom`, `\overset`, `\underset`, `\class`, `\raisebox`, `\cancel`, `\mathop`, `\operatorname`, `\pmod`, `\phantom`, `\hspace`, `\tag`, `\label`, `\substack`, `\underbrace`, `\overbrace`, `\not`, `\sqrt`, `\text`, `\color`, `\mathrm`, `\mathbb`, `\hat`, `\newcommand`, `\def`, `\kern`, `\bf`, `\Huge`, `\sideset`, `\sideset`, `\prescript`}

func zzGram(r *rand.Rand, d int) string {
	var sb strings.Builder
	n := 1 + r.Intn(6)
	for i := 0; i < n; i++ {
		k := r.Intn(14)
		if d > 4 {
			k = 0
		}
		switch k {
		case 0, 1, 2, 3:
			f := zzFrags[r.Intn(len(zzFrags))]
			for f == "{" || f == "}" || strings.HasPrefix(f, `\begin`) || strings.HasPrefix(f, `\end`) {
				f = zzFrags[r.Intn(len(zzFrags))]
			}
			sb.WriteString(f)
		case 4, 5:
			sb.WriteString("{" + zzGram(r, d+1) + "}")
		case 6:
			sb.WriteString("[" + zzGram(r, d+1) + "]")
		case 7:
			e := zzEnvs[r.Intn(len(zzEnvs))]
			end := strings.SplitN(e, "}", 2)[0]
			sb.WriteString(`\begin{` + e + `}` + zzGram(r, d+1) + `\end{` + end + `}`)
		case 8:
			sb.WriteString(`\left` + zzDelims[r.Intn(len(zzDelims))] + zzGram(r, d+1) + `\right` + zzDelims[r.Intn(len(zzDelims))])
		case 9:
			sb.WriteString(`\middle` + zzDelims[r.Intn(len(zzDelims))])
		case 10:
			sb.WriteString([]string{`\big`, `\bigl`, `\bigr`, `\Bigg`}[r.Intn(4)] + zzDelims[r.Intn(len(zzDelims))])
		case 11, 12:
			sb.WriteString(zzCmds[r.Intn(len(zzCmds))])
		case 13:
			sb.WriteString(`\newcommand{\foo}{` + zzGram(r, d+1) + `}\foo`)
		}
	}
	return sb.String()
}

func TestZZGram(t *testing.T) {
	seed, _ := strconv.ParseInt(os.Getenv("ZZSEED"), 10, 64)
	n, _ := strconv.Atoi(os.Getenv("ZZN"))
	if n == 0 {
		t.Skip()
	}
	r := rand.New(rand.NewSource(seed))
	crashes, diffs, regress, baseCrash, tokErr := 0, 0, 0, 0, 0
	sites := map[string]int{}
	var slowest time.Duration
	var slowTex string
	for k := 0; k < n; k++ {
		tex := zzGram(r, 0)
		display := k%2 == 1
		var hOut, bOut string
		var hErr, bErr error
		t0 := time.Now()
		if display {
			hOut, hErr = NewMathMLConverter().ConvertDisplay(tex)
		} else {
			hOut, hErr = NewMathMLConverter().ConvertInline(tex)
		}
		if d := time.Since(t0); d > slowest {
			slowest, slowTex = d, tex
		}
		if display {
			bOut, bErr = zzbase.NewMathMLConverter().ConvertDisplay(tex)
		} else {
			bOut, bErr = zzbase.NewMathMLConverter().ConvertInline(tex)
		}
		if bErr != nil && bErr.Error() == zzCrash {
			baseCrash++
		} else if bErr != nil {
			tokErr++
		}
		if hErr != nil && hErr.Error() == zzCrash {
			crashes++
			s := zzRaw(tex, display)
			if sites[s] < 2 {
				t.Logf("HEAD CRASH %q: %s", tex, s)
			}
			sites[s]++
			continue
		}
		if bErr == nil && hErr != nil {
			regress++
			if regress <= 10 {
				t.Logf("REGRESS %q: head err %v", tex, hErr)
			}
			continue
		}
		if bErr == nil && hOut != bOut {
			diffs++
			if diffs <= 10 {
				t.Logf("DIFF %q\n base %s\n head %s", tex, bOut, hOut)
			}
			if f := os.Getenv("ZZOUT"); f != "" && diffs == 1 {
				os.WriteFile(f, []byte(tex), 0o644)
			}
		}
	}
	t.Logf("n=%d seed=%d baseCrash=%d otherErr=%d headCrash=%d regress=%d diffs=%d sites=%d slowest=%v %q", n, seed, baseCrash, tokErr, crashes, regress, diffs, len(sites), slowest, slowTex)
}

func zzIsDiff(tex string) bool {
	for _, display := range []bool{false, true} {
		var h, b string
		var he, be error
		if display {
			h, he = NewMathMLConverter().ConvertDisplay(tex)
			b, be = zzbase.NewMathMLConverter().ConvertDisplay(tex)
		} else {
			h, he = NewMathMLConverter().ConvertInline(tex)
			b, be = zzbase.NewMathMLConverter().ConvertInline(tex)
		}
		if be == nil && (he != nil || h != b) {
			return true
		}
	}
	return false
}

func TestZZMin(t *testing.T) {
	f := os.Getenv("ZZMIN")
	if f == "" {
		t.Skip()
	}
	raw, _ := os.ReadFile(f)
	s := []rune(string(raw))
	if !zzIsDiff(string(s)) {
		t.Fatal("not a diff")
	}
	for chunk := len(s) / 2; chunk >= 1; {
		removed := false
		for i := 0; i+chunk <= len(s); {
			c := append(append([]rune{}, s[:i]...), s[i+chunk:]...)
			if zzIsDiff(string(c)) {
				s = c
				removed = true
			} else {
				i += chunk
			}
		}
		if !removed {
			chunk /= 2
		}
	}
	tex := string(s)
	t.Logf("MIN %q", tex)
	for _, display := range []bool{false, true} {
		var h, b string
		var he, be error
		if display {
			h, he = NewMathMLConverter().ConvertDisplay(tex)
			b, be = zzbase.NewMathMLConverter().ConvertDisplay(tex)
		} else {
			h, he = NewMathMLConverter().ConvertInline(tex)
			b, be = zzbase.NewMathMLConverter().ConvertInline(tex)
		}
		t.Logf("display=%v\n base err=%v %s\n head err=%v %s", display, be, b, he, h)
	}
}
