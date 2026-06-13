# Review: PR #5588
Event: REQUEST_CHANGES

## Body
Verified on 07006e5f9.

One scope question: bare `append(s, x)` and other value-returning builtins as statements are still accepted, but modern Go rejects an unused `append` result, so decide whether to match Go or document the loophole.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5588-unused-expr-stmt/1-07006e5f9/claude-opus-4-7_davd-gzl.md [↗](claude-opus-4-7_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/preprocess.go:2519-2521 [↗](../../../../../.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L2519)
The `n.X.(*CallExpr)` check also matches type conversions: `int64(x)` and `T(x)` parse as `*CallExpr` and stay that way, so a standalone conversion is accepted even though its result is unused (go/types rejects it). That is the same unused-result bug this PR fixes, left half-open. Fix: when `n.X` is a `*CallExpr`, also reject it when it resolves to a type conversion (e.g. check `evalStaticTypeOf` for a type value).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5588 -R gnolang/gno
cat > gnovm/tests/files/zz_typeconv.gno <<'EOF'
package main

func main() {
	x := 5
	int64(x)
	println("ok")
}

// Error:
// main/zz_typeconv.gno:5:2: invalid operation: int64(x) is not used
EOF
go test -run 'TestFiles/zz_typeconv.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zz_typeconv.gno
```

```
--- FAIL: TestFiles/zz_typeconv.gno (0.00s)
        files_test.go:111: unexpected output:
            ok
```
The fixture asserts the post-fix rejection; it fails now because the preprocessor accepts `int64(x)` and the program runs to completion (prints `ok`).
</details>

*(AI Agent)*

## gnovm/pkg/gnolang/preprocess.go:2520 [↗](../../../../../.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L2520)
The panic interpolates `n.X.String()`, which dumps internal AST decoration: the committed fixtures show [`(const (1 <untyped> bigint))`](https://github.com/gnolang/gno/blob/07006e5f9/gnovm/tests/files/expr_stmt0.gno#L10) · [↗](../../../../../.worktrees/gno-review-5588/gnovm/tests/files/expr_stmt0.gno#L10) and `x<VPBlock(1,0)> + (const (1 int))` instead of source like `1` or `x + 1`, so the message a user sees is unreadable. Fix: build the message from the source span (`n.X.GetSpan()`) or align with go/types' wording.

*(AI Agent)*
