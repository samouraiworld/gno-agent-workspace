// Repro: an expression master converts without error converts to different
// MathML at c30e596a8, against the PR's claim that every such expression
// converts the same.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6377/head && git checkout --detach c30e596a819207b6da8d197a166f47288f57eb22
//	P=gno.land/pkg/gnoweb/markdown/mathml; mkdir $P/zzbase
//	for f in $(git ls-tree --name-only 6c49f5420ec9c8ad0933b6869461656f889a9a37 $P/ | grep '\.go$' | grep -v _test.go); do
//	  git show 6c49f5420ec9c8ad0933b6869461656f889a9a37:$f | sed 's/^package mathml$/package zzbase/' > $P/zzbase/$(basename $f)
//	done
//	cp <this file> $P/zz_sideset_test.go
//	go test -count=1 -run TestZZSideset -v ./$P/
//
// Expect FAIL on the first input: master draws <none></none> in the last
// prescript slot, the branch draws <mi>a</mi>. Cause: GetNextToken
// (tokenize.go:378) answered ErrTokenBufferExpr for every unescaped curly
// opener and now only for a matched "{"; a "}" that \left or \bigl folded
// into an opener is handed back as a token, so cmd_sideset's getScripts no
// longer stops at it and reads the ^a after it.
//
// Measured: 5.4M grammar-generated expressions (seeds 1,2,3,5,6,7,11,12) and
// 4.4M coverage-guided go-fuzz execs found no other differing shape and no
// crash at the branch.
package mathml

import (
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown/mathml/zzbase"
)

func TestZZSideset(t *testing.T) {
	for _, tex := range []string{`\sideset{\left}^a}}{}x`, `\sideset{\bigl}^a}}{}x`, `\sideset{^b\left}^a}}{}x`} {
		h, he := NewMathMLConverter().ConvertInline(tex)
		b, be := zzbase.NewMathMLConverter().ConvertInline(tex)
		if be == nil && (he != nil || h != b) {
			t.Errorf("%q\n master %v %s\n branch %v %s", tex, be, b, he, h)
		}
	}
}
