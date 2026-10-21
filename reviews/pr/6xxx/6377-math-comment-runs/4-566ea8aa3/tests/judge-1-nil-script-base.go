// Judge check for candidates 1 and 1001: each input converted at the head and
// at the merge base 6c49f5420 (package copied as zzbase per the recipe in
// solo-finder-2-diff-fuzz.go). Prints both; fails where the head puts y inside
// a script the source does not give it.
//
//	cp judge-1-nil-script-base.go gno.land/pkg/gnoweb/markdown/mathml/zz_judge1_test.go
//	go test -count=1 -run TestZZJudge1 -v ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown/mathml/zzbase"
)

func TestZZJudge1(t *testing.T) {
	cut := func(s string) string {
		if i := strings.Index(s, "<semantics>"); i >= 0 {
			s = s[i+11:]
		}
		if i := strings.Index(s, "<annotation"); i >= 0 {
			s = s[:i]
		}
		return s
	}
	for _, tex := range []string{
		`x^\class{a}{} y`, `x_\class{a}{} y`, `x^\class{a}{}`, `\class{a}{}^2 y`,
		`x^\bigl\mathrm{} y`, `x^\middle\class{a}{} y`,
		`x^\mathrm{} y`, `x^\kern y`, `x^{} y`, `x^\class{a}{b} y`,
		`\left\mathrm{} a \right) b`, `\left. a \right) b`, `\left\class{a}{} a \right) b`,
	} {
		h, he := NewMathMLConverter().ConvertInline(tex)
		b, be := zzbase.NewMathMLConverter().ConvertInline(tex)
		t.Logf("%-32q head err=%v %s", tex, he, cut(h))
		t.Logf("%-32s base err=%v %.110s", "", be, cut(b))
		if strings.HasPrefix(tex, "x^") && strings.HasSuffix(tex, " y") && strings.Contains(cut(h), "<mi>y</mi></msup>") {
			t.Errorf("%s: y rendered as the superscript", tex)
		}
	}
}
