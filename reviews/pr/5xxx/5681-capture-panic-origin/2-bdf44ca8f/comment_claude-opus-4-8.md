# Review: PR [#5681](https://github.com/gnolang/gno/pull/5681)
Posted: https://github.com/gnolang/gno/pull/5681#pullrequestreview-5042802158
Event: COMMENT
Status: posting as an AI. Original verdict REQUEST_CHANGES, forced to COMMENT.

## Body
[AI review, opus 4.8]
Status: REQUEST_CHANGES

GoStack is captured at every VM raise site except three that still build `&Exception{}` directly, so the new `go stack:` output is blank for the panics those three raise: the slice-index-out-of-bounds raise in [`values.go:384-395`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/values.go#L384-L395) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/values.go#L384), and the divide-by-zero raises in [`quoAssign`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/op_binary.go#L935) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/op_binary.go#L935) and [`remAssign`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/op_binary.go#L1034) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/op_binary.go#L1034). Route all three through a skip-aware `NewException`. The `op_binary` pair return the exception for their caller to panic with rather than panicking themselves, so they need the skip variant to record the real raise site.

## gnovm/pkg/gnolang/frame.go:296-298 [gh](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/frame.go#L296-L298) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/frame.go#L296) [posted](https://github.com/gnolang/gno/pull/5681#discussion_r3873392596)
`NewException` and [`pushPanic`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/machine.go#L2812) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/machine.go#L2812) capture the Go stack on every VM panic, validators included, and the only reader of `GoStack` is the filetest harness: the keeper renders the Descriptor and the gno stacktrace alone. A 20-frame raise benchmarks at 6688 ns/op and 7702 B/op over 13 allocations, unmetered, on every nil-deref, index, slice, map-key and user `panic()`. The capture also sidesteps [`BoundedPanicRender`](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/gnolang/machine.go#L104) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/gnolang/machine.go#L104), which exists to bound this cost. Gate it behind that flag or a debug flag so validators stop paying for output nobody reads.

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

## gnovm/pkg/test/filetest.go:395 [gh](https://github.com/gnolang/gno/blob/bdf44ca8f/gnovm/pkg/test/filetest.go#L395) · [↗](../../../../../.worktrees/gno-review-5681/gnovm/pkg/test/filetest.go#L395) [posted](https://github.com/gnolang/gno/pull/5681#discussion_r3873392604)
`label := "panic"` is dead: both branches just below reassign it before use. Use `var label string`.

