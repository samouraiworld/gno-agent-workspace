# Review: PR [#5867](https://github.com/gnolang/gno/pull/5867)
Posted: https://github.com/gnolang/gno/pull/5867#pullrequestreview-4672487955
Event: REQUEST_CHANGES

## Body
Reproduced on 7817f6e1d. Ran `println(1e10000 / 1e10000)` on a build of this head and one of master d2869dceb: master prints 1, this head panics at preprocess.

[`float9.gno`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/tests/files/float9.gno) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/float9.gno) dies before its float32 assertion runs, so the suite cannot show what that assertion would catch. Copied into a file of its own, `var f32 float32 = -1e-50` gives -0 here and +0 on master.

The red `build` check is not a code problem.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/3-7817f6e1d/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/op_binary.go:809-816 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L809) [posted](https://github.com/gnolang/gno/pull/5867#discussion_r3559803620)
Critical: untyped float constants past roughly 1e±1233 panic at preprocess, where Go evaluates them. Past its own [4096-bit threshold](https://github.com/golang/go/blob/go1.25.0/src/go/constant/value.go#L351), `go/constant` [switches the constant](https://github.com/golang/go/blob/go1.25.0/src/go/constant/value.go#L328-L345) to a [512-bit](https://github.com/golang/go/blob/go1.25.0/src/go/constant/value.go#L69) [`big.Float`](https://pkg.go.dev/math/big#Float) rather than rejecting it. Related: [#5740](https://github.com/gnolang/gno/pull/5740).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5867 -R gnolang/gno   # ./gnovm/cmd/gno below is built from this tree

cat > /tmp/ratio.gno <<'EOF'
package main

func main() { println(1e10000 / 1e10000) }
EOF
cp /tmp/ratio.gno /tmp/ratio.go      # same source, for the Go compiler

go run ./gnovm/cmd/gno run /tmp/ratio.gno
go run /tmp/ratio.go

rm -f /tmp/ratio.gno /tmp/ratio.go
```

```
panic: constant expression result too large: numerator exceeds 4096 bits [recovered, repanicked]
	panic: main//tmp/ratio.gno:3:23-30: constant expression result too large: numerator exceeds 4096 bits:
	--- preprocess stack ---
	stack 2: func main//tmp/ratio.gno:3:1-43
# … gno panics; the Go build of the same source prints:
1
```
</details>

## gnovm/pkg/gnolang/values_conversions.go:1458-1466 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1458) [posted](https://github.com/gnolang/gno/pull/5867#discussion_r3559803633)
Critical: `posZero` runs on the float64 before the narrowing, so a constant that underflows only at float32 width stores -0. `var f32 float32 = -1e-50` is -0 here and +0 on master. The `var float32: +0` line of [`float9.gno`](https://github.com/gnolang/gno/blob/7817f6e1d/gnovm/tests/files/float9.gno#L23) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/float9.gno#L23) asserts this but never runs, because the size guard rejects line 9 first.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5867 -R gnolang/gno   # ./gnovm/cmd/gno below is built from this tree

cat > /tmp/f32sign.gno <<'EOF'
package main

import "math"

func main() {
	var f32 float32 = -1e-50
	println("signbit:", math.Signbit(float64(f32)))
}
EOF
cp /tmp/f32sign.gno /tmp/f32sign.go  # same source, for the Go compiler

go run ./gnovm/cmd/gno run /tmp/f32sign.gno
go run /tmp/f32sign.go

rm -f /tmp/f32sign.gno /tmp/f32sign.go
```

```
signbit: true
signbit: false
```
</details>
