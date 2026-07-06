# Review: PR [#5741](https://github.com/gnolang/gno/pull/5741)
Posted: https://github.com/gnolang/gno/pull/5741#pullrequestreview-4637298706
Event: REQUEST_CHANGES

## Body
Verified on a6dc98e3b against a side-by-side Go run: constant underflow yields +0, runtime underflow keeps -0, both matching Go.

Related: [#5864](https://github.com/gnolang/gno/pull/5864) applies the same -0 to +0 fold to MsgCall float args in `convertFloat`, the maketx entry point this PR leaves untouched.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5741-float-const-signed-zero/1-a6dc98e3b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values_conversions.go:1020-1025 [↗](../../../../../.worktrees/gno-review-5741/gnovm/pkg/gnolang/values_conversions.go#L1020) [posted](https://github.com/gnolang/gno/pull/5741#discussion_r3530051131)
A negative typed constant that overflows float32, like `float32(-1e39)`, narrows to `-Inf` instead of a compile error. Go and gno's go/types checker both reject it, but the VM does not. The narrowing check only tests the upper bound, so reject values below `-MaxFloat32` too.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5741 -R gnolang/gno
cat > gnovm/tests/files/float13.gno <<'EOF'
package main

const big float64 = -1e39

func main() {
	x := float32(big)
	println(x)
}

// Error:
// main/float13.gno:6:7-19: cannot convert constant of type Float64Kind to Float32Kind

// TypeCheckError:
// main/float13.gno:6:15: cannot convert big (constant -1e+39 of type float64) to type float32
EOF
go test -run 'TestFiles/float13.gno$' ./gnovm/pkg/gnolang/ 2>&1 | head -8
rm gnovm/tests/files/float13.gno
```

```
--- FAIL: TestFiles/float13.gno (0.00s)
    files_test.go:129: unexpected output:
        -Inf

FAIL
```
Real Go rejects the same program: `cannot convert big (constant -1e+39 of type float64) to type float32`. The test passes once the narrowing rejects values below `-MaxFloat32`.
</details>
