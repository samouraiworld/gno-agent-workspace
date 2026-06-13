# Review: PR #5681
Event: REQUEST_CHANGES

## Body
GoStack is captured at every VM raise site except three that still build `&Exception{}` directly: the slice-index-out-of-bounds raise in [`values.go:384-395`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/values.go#L384-L395) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/values.go#L384-L395), and the divide-by-zero raises in `quoAssign` ([`op_binary.go:935`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/op_binary.go#L935) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/op_binary.go#L935)) and `remAssign` ([`op_binary.go:1034`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/op_binary.go#L1034) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/op_binary.go#L1034)), so the new `go stack:` output is blank for these common panics. Route them through a skip-aware `NewException` (the `op_binary` pair return the exception for the caller to panic with, so they need the skip variant to record the real raise site).

The two red checks are merge-behind, not code: `params_valset_rotation_throttle` fails identically on this branch's merge-base with zero exception changes (it carries the old height-counting txtar; master has since rewritten it to `gnoland wait-for-new-block`), and the lint failure is the seal machinery from #5706, which postdates the merge-base. Merging master clears both.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5681-capture-panic-origin/2-bdf44ca8f/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/frame.go:296-298 [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/frame.go#L296)
`NewException` (and `pushPanic`) capture the Go stack on every VM panic including on validators, but the only reader of `GoStack` is the filetest harness; the keeper renders the Descriptor and gno stacktrace only. That is ~6.7µs and 7.7KB/13 allocs of unmetered work per panic (nil-deref, index, slice, map-key, user `panic()`) for data production never shows, and it sidesteps `BoundedPanicRender`, which exists to bound exactly this. Gate the capture behind `BoundedPanicRender` or a debug flag so validators don't pay for it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5681 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_gostack_bench_test.go <<'EOF'
package gnolang

import "testing"

func deepRaise(n int) *Exception {
	if n == 0 {
		return NewException(typedString("boom"))
	}
	return deepRaise(n - 1)
}

func BenchmarkNewException(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = deepRaise(20)
	}
}

func BenchmarkBareException(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = &Exception{Value: typedString("boom")}
	}
}
EOF
go test ./gnovm/pkg/gnolang/ -run x -bench 'BenchmarkNewException|BenchmarkBareException' -benchmem
rm gnovm/pkg/gnolang/zz_gostack_bench_test.go
```

```
BenchmarkNewException-16     183169      6688 ns/op    7702 B/op    13 allocs/op
BenchmarkBareException-16  1000000000       0.22 ns/op     0 B/op     0 allocs/op
```
</details>

*(AI Agent)*

## gnovm/pkg/test/filetest.go:395 [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/test/filetest.go#L395)
`label := "panic"` is dead: both branches just below reassign it before use. Use `var label string`.

*(AI Agent)*
