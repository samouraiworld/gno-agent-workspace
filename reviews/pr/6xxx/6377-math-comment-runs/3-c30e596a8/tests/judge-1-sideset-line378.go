// Judge probe for candidate 1: the \left} / \bigl} fold inside \sideset converts
// to different MathML at c30e596a8 than at 6c49f5420, both without error.
// Setup as in solo-finder-sideset-curly-fence.go (master's package copied to zzbase).
// Also: reverting tokenize.go:378 alone to master's guard turns TestZZSideset
// green and TestMalformedInputDoesNotCrash red on its \Bigl}} input.
package mathml

import (
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown/mathml/zzbase"
)

func TestZZJudge(t *testing.T) {
	for _, tex := range []string{`\sideset{\left}^a}}{}x`, `\sideset{\bigl}^a}}{}x`, `\sideset{^b\left}^a}}{}x`, `\sideset{\left(^a}{}x`, `\sideset{^a}{}x`, `x\left}^a`, `\left}`} {
		h, he := NewMathMLConverter().ConvertInline(tex)
		b, be := zzbase.NewMathMLConverter().ConvertInline(tex)
		t.Logf("%q same=%v\n  base err=%v %s\n  head err=%v %s", tex, h == b && (he == nil) == (be == nil), be, b, he, h)
	}
}
