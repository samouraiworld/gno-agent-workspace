// A command that draws nothing, used as a script, lifts the next atom into
// the script: x^\class{a}{} y renders as x^y. At the merge base the same
// input fails with "MathML encountered an unexpected error" (nil deref at
// n.Tok = tok in ProcessCommand); the PR's nil guards turn it into a
// silent wrong render. Every sibling with an empty argument (\phantom{},
// \mathop{}, \text{}, \cancel{}, \textcolor{red}{}, \substack{}) and {}
// itself keep an empty superscript with y after it.
//
// Repro from a plain clone (PR head c30e596a8 carries the same guard at
// commands.go:373-376; the round's snapshot 566ea8aa3 is local only):
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6377/head && git checkout --detach FETCH_HEAD
//	cp <this file> gno.land/pkg/gnoweb/markdown/mathml/zz_solo1_class_empty_script_test.go
//	go test -count=1 -run TestZZSolo1ClassEmptyScript -v ./gno.land/pkg/gnoweb/markdown/mathml/
//
// Expected at head: FAIL for \class{a}{}, \bigl\mathrm{}, \middle\class{a}{}
// (y inside <msup>). At the merge base 6c49f5420: the first input errors
// instead. \mathrm{} as a script base lifts y at the base too.
//
// Observed at 566ea8aa3:
//
//	x^\class{a}{} y          err=<nil> <mrow><msup><mi>x</mi><mi>y</mi></msup></mrow>
//	x^{} y                   err=<nil> <mrow><msup><mi>x</mi><mrow></mrow></msup><mi>y</mi></mrow>
//	x^\phantom{} y           err=<nil> <mrow><msup><mi>x</mi><mphantom><mrow></mrow></mphantom></msup><mi>y</mi></mrow>
//	x^\bigl\mathrm{} y       err=<nil> <mrow><msup><mi>x</mi><mi>y</mi></msup></mrow>
//	x^\middle\class{a}{} y   err=<nil> <mrow><msup><mi>x</mi><mi>y</mi></msup></mrow>
//
// Observed at 6c49f5420 (package copied beside it):
//
//	x^\class{a}{} y          err=MathML encountered an unexpected error <math ...><merror><mtext>invalid math input</mtext></merror></math>
//	x^\mathrm{} y            err=<nil> <mrow><msup><mi>x</mi><mi>y</mi></msup></mrow>
package mathml

import (
	"strings"
	"testing"
)

func TestZZSolo1ClassEmptyScript(t *testing.T) {
	body := func(tex string) string {
		out, err := NewMathMLConverter().ConvertInline(tex)
		if err != nil {
			t.Fatalf("%s: %v", tex, err)
		}
		return out[strings.Index(out, "<semantics>")+11 : strings.Index(out, "<annotation")]
	}
	for _, tex := range []string{`x^\class{a}{} y`, `x^\bigl\mathrm{} y`, `x^\middle\class{a}{} y`} {
		out := body(tex)
		if !strings.HasSuffix(out, "</msup><mi>y</mi></mrow>") {
			t.Errorf("%s: y is not after the script: %s", tex, out)
		}
	}
}
