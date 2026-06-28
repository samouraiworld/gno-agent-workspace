# Review: PR #5763
Event: APPROVE

## Body
Verified on 093c32be0: reverting the [`fillTypeInPlace`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1553) fill reproduces the empty-base panic on `type T2 T1`. Mutual recursion matches a side-by-side Go run across both declaration orders, conversions, and multi-level chains. An illegal finite-size value-cycle still rejects like Go.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/2-093c32be0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/tests/files/decltype_mutual.gno:10-13 [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L10)
The test pins only the T1-first order and a pointer-field read. The swapped declaration order and a direct `var b T2` field read are not covered, though both work today. Pinning them would lock in that both ends of the cycle and both orders agree.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
cat > gnovm/tests/files/decltype_mutual_order_swap.gno <<'EOF'
package main

type T2 T1
type T1 struct {
	Next *T2
	Val  int
}

func main() {
	var a T1
	a.Next = &T2{Val: 9}
	println(a.Next.Val)
	var b T2
	b.Val = 7
	println(b.Val)
	println("ok")
}

// Output:
// 9
// 7
// ok
EOF
go test -v -run 'TestFiles/decltype_mutual_order_swap.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/decltype_mutual_order_swap.gno
```

```
--- PASS: TestFiles/decltype_mutual_order_swap.gno (0.00s)
PASS
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang
```
</details>
