// judge-reflector, PR 6377 round 5 at 2d8efc2df: random \color, empty-command, script and cell-end
// sequences, head against the merge base 6c49f5420 copied to zzbase (see solo-finder-switch-trailing-blanks.go).
// Run twice, with and without \class{a}{}, \mathrm{} and \kern. Counts inputs only head rejects (a regression), only base rejects, and both-ok outputs that differ.
//   cp judge-reflector-fuzz-vs-base.go gno.land/pkg/gnoweb/markdown/mathml/zz_fuzz_test.go
//   go test -count=1 -run TestZZFuzzVsBase -v ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"math/rand"
	"strings"
	"testing"

	base "github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown/zzbase"
)

func TestZZFuzzVsBase(t *testing.T) {
	fuzzVsBase(t, true)
	fuzzVsBase(t, false)
}

// withEmpty keeps the commands that draw nothing; without them every both-ok output should match.
func fuzzVsBase(t *testing.T, withEmpty bool) {
	pieces := []string{`\color`, `[RGB]`, `[`, `]`, `{red}`, `{`, `}`, `&`, `\\`, `\cr`, "%c\n", " ",
		`x`, `^`, `_`, `\class{a}{}`, `\mathrm{}`, `\kern`, `\left(`, `\right)`, `\middle|`, `\bigl`, `\bigr`,
		`\begin{matrix}`, `\end{matrix}`, `\rm`, `\bf`, `\over`, `{}`, `\sqrt`, `\hat`}
	if !withEmpty {
		var kept []string
		for _, p := range pieces {
			if p != `\class{a}{}` && p != `\mathrm{}` && p != `\kern` {
				kept = append(kept, p)
			}
		}
		pieces = kept
	}
	strip := func(s string) string {
		if i := strings.Index(s, "<annotation"); i >= 0 {
			return s[:i]
		}
		return s
	}
	r := rand.New(rand.NewSource(5))
	var headOnly, baseOnly, differ, bothOK int
	var samplesH, samplesD []string
	for n := 0; n < 200000; n++ {
		var sb strings.Builder
		for k := 1 + r.Intn(10); k > 0; k-- {
			sb.WriteString(pieces[r.Intn(len(pieces))])
		}
		tex := sb.String()
		ho, he := NewMathMLConverter().ConvertInline(tex)
		bo, be := base.NewMathMLConverter().ConvertInline(tex)
		switch {
		case he != nil && be == nil:
			headOnly++
			if len(samplesH) < 5 {
				samplesH = append(samplesH, tex+" => "+he.Error())
			}
		case he == nil && be != nil:
			baseOnly++
		case he == nil && be == nil:
			bothOK++
			if strip(ho) != strip(bo) {
				differ++
				if len(samplesD) < 5 {
					samplesD = append(samplesD, tex)
				}
			}
		}
	}
	t.Logf("withEmpty=%v bothOK=%d differ=%d headOnlyErr=%d baseOnlyErr=%d", withEmpty, bothOK, differ, headOnly, baseOnly)
	for _, s := range samplesH {
		t.Logf("head only: %q", s)
	}
	for _, s := range samplesD {
		t.Logf("differ: %q", s)
	}
}
