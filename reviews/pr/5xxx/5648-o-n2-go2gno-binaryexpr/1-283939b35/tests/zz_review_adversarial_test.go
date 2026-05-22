/*
Run:

git -C gno worktree add ../.worktrees/gno-review-5648 origin/master
gh pr checkout 5648 -R gnolang/gno --force --branch pr-5648 --repo-clone-dir .worktrees/gno-review-5648
cp reviews/pr/5xxx/5648-o-n2-go2gno-binaryexpr/1-283939b35/tests/zz_review_adversarial_test.go .worktrees/gno-review-5648/gnovm/pkg/gnolang/
go test -C .worktrees/gno-review-5648 -v -run 'TestPR5648_' ./gnovm/pkg/gnolang/
rm .worktrees/gno-review-5648/gnovm/pkg/gnolang/zz_review_adversarial_test.go
git -C gno worktree remove --force ../.worktrees/gno-review-5648

Pins the 1-column Span.End drift introduced by the fast-path Span
computation when the chain's rightmost operand is a ParenExpr.
*/
package gnolang

import "testing"

// Adversarial test: rightmost operand wrapped in ParenExpr.
//
//	const x = 1 + 2 + (3 + 4)
//
// Outer BinaryExpr: X = `1 + 2` (BinaryExpr), Y = `(3 + 4)` (ParenExpr).
// Gate fires (X is BinaryExpr). Fast path uses bx.Right.GetSpan().End.
// bx.Right is the unwrapped inner BinaryExpr `3 + 4`, whose Span.End
// is the column after `4` — NOT the column after `)`.
//
// Original SpanFromGo would use gon.End() = ParenExpr.End() = col after `)`.
//
// This test pins down which behavior the fix produces, surfacing the
// 1-column difference relative to pre-fix.
func TestPR5648_RightmostParen(t *testing.T) {
	// const x = 1 + 2 + (3 + 4)
	// col:    12345678901234567890123456
	//                            ^     ^
	//                           19    25
	// `(` at col 19, `4` at col 24, `)` at col 25.
	src := "package main\n\nconst x = 1 + 2 + (3 + 4)\n\nfunc main(){}\n"
	m := NewMachine("test", nil)
	t.Cleanup(m.Release)
	fn, err := m.ParseFile("test.gno", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	vd := fn.Decls[0].(*ValueDecl)
	bx := vd.Values[0].(*BinaryExpr)
	span := bx.GetSpan()
	t.Logf("Outer Span: %+v", span)
	if span.Pos.Line != 3 || span.Pos.Column != 11 {
		t.Errorf("Span.Pos = %v; want line 3 col 11", span.Pos)
	}
	// Pre-fix behavior: End = col 26 (after `)`).
	// Post-fix behavior: End = col 25 (after `4`, before `)`).
	t.Logf("Span.End = %v  (pre-fix would be col 26, post-fix gives col 25)", span.End)
}

// Adversarial test: chain whose rightmost operand is a parenthesized
// sub-chain — `1 + 2 + (3 + 4 + 5)`. Same column-drift question.
func TestPR5648_RightmostParenChain(t *testing.T) {
	// const x = 1 + 2 + (3 + 4 + 5)
	// col:    1234567890123456789012345678901
	src := "package main\n\nconst x = 1 + 2 + (3 + 4 + 5)\n\nfunc main(){}\n"
	m := NewMachine("test", nil)
	t.Cleanup(m.Release)
	fn, err := m.ParseFile("test.gno", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	vd := fn.Decls[0].(*ValueDecl)
	bx := vd.Values[0].(*BinaryExpr)
	span := bx.GetSpan()
	t.Logf("Outer Span = %+v", span)
}
