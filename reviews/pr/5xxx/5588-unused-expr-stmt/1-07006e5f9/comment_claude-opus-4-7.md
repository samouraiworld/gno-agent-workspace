# Review: PR #5588
Posted: https://github.com/gnolang/gno/pull/5588#pullrequestreview-4492097289
Event: REQUEST_CHANGES

## Body
Verified on 07006e5f9.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5588-unused-expr-stmt/1-07006e5f9/claude-opus-4-7_davd-gzl.md [↗](claude-opus-4-7_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/preprocess.go:2519-2521 [↗](../../../../../.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L2519)
To match Go, the `n.X.(*CallExpr)` check is too permissive: type conversions (`int64(x)`, `T(x)`) and the result-only builtins `append`, `cap`, `len`, `make`, `new` all parse as `*CallExpr`, so their unused results are silently accepted while Go rejects each as "... is not used". Fix: reject those too, leaving only genuine calls and the side-effecting builtins (`copy`, `delete`, `panic`, `recover`, `print`, `println`) valid as statements.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5588 -R gnolang/gno
cat > gnovm/tests/files/zz_matchgo.gno <<'EOF'
package main

func main() {
	x := 5
	s := []int{1}
	int64(x)
	append(s, 2)
	println("ok")
}

// Error:
// main/zz_matchgo.gno:6:2: invalid operation: int64(x) is not used
EOF
go test -run 'TestFiles/zz_matchgo.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zz_matchgo.gno
```

```
--- FAIL: TestFiles/zz_matchgo.gno (0.00s)
        files_test.go:111: unexpected output:
            ok
```
The fixture asserts the post-fix rejection; it fails now because the preprocessor accepts both `int64(x)` and `append(s, 2)`, so the program runs to completion (prints `ok`).
</details>

*(AI Agent)*

## gnovm/pkg/gnolang/preprocess.go:2520 [↗](../../../../../.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L2520)
The panic interpolates `n.X.String()`, which dumps internal AST decoration: the committed fixtures show [`(const (1 <untyped> bigint))`](https://github.com/gnolang/gno/blob/07006e5f9/gnovm/tests/files/expr_stmt0.gno#L10) · [↗](../../../../../.worktrees/gno-review-5588/gnovm/tests/files/expr_stmt0.gno#L10) and `x<VPBlock(1,0)> + (const (1 int))` instead of source like `1` or `x + 1`, so the message a user sees is unreadable. Fix: build the message from the source span (`n.X.GetSpan()`) or align with go/types' wording.

*(AI Agent)*
