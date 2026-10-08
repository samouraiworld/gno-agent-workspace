// Repro from a plain clone of gnolang/gno:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/4879/head && git checkout 0b3987540842f0f7a875e2a344d5995fd47f40d5
//   cp <this file> gno.land/pkg/gnoweb/markdown/mathml/zz_b7_infix_test.go
//   go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestB7InfixOverScope -v
// Expect FAIL at head: postProcessInfix (parse.go:340, :579) runs before postProcessScripts and takes only the adjacent sibling, so the exponent becomes the numerator.
package mathml

import (
	"regexp"
	"strings"
	"testing"
)

var _ = strings.Contains

func renderB7InfixOverScope(t *testing.T, tex string) (string, error) {
	t.Helper()
	out, err := NewMathMLConverter().ConvertInline(tex)
	return regexp.MustCompile(`(?s)<annotation.*</annotation>`).ReplaceAllString(out, ""), err
}

func TestB7InfixOverScope(t *testing.T) {
	cases := []struct{ tex, want string }{
		{`{x^2 \over 2}`, `<mfrac><msup><mi>x</mi><mn>2</mn></msup><mn>2</mn></mfrac>`},
		{`1 \over x^2`, `<mfrac><mn>1</mn><msup><mi>x</mi><mn>2</mn></msup></mfrac>`},
		{`a+b \over c`, `<mfrac><mrow><mi>a</mi><mo>+</mo><mi>b</mi></mrow><mi>c</mi></mfrac>`},
		{`{n+1 \choose k}`, `<mfrac linethickness="0"><mrow><mi>n</mi><mo>+</mo><mn>1</mn></mrow><mi>k</mi></mfrac>`},
	}
	for _, c := range cases {
		out, err := renderB7InfixOverScope(t, c.tex)
		if err != nil || !strings.Contains(out, c.want) {
			t.Errorf("%q: err=%v, want substring %s\n  got %s", c.tex, err, c.want, out)
		}
	}
	if _, err := renderB7InfixOverScope(t, `1 \over \int\limits_0^1 f`); err != nil {
		t.Errorf("%q: valid TeX falls back: %v", `1 \over \int\limits_0^1 f`, err)
	}
}
