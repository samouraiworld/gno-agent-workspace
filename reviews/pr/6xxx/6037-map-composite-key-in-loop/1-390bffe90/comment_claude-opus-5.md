# Review: PR [#6037](https://github.com/gnolang/gno/pull/6037)
Event: APPROVE

## Body
Verified on 390bffe90: reverting the two lines that strip the `.loopvar` suffix, with the exclusion-list change kept, makes [`loopvar_struct_field_2.gno`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/tests/files/loopvar_struct_field_2.gno#L1-L19) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/loopvar_struct_field_2.gno#L1-L19) fail with `struct type struct{i int} has no field i.loopvar`. It is the only committed guard on the trim. Seventeen literal shapes, including closure capture and a labeled `continue` over the loop variable's address, print the same under a `gno` built from this branch as under `go run`.

- The [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L194-L197) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L194-L197) contract comment still lists composite-literal keys among the positions the rename skips. A reader who follows it restores the exclusion and reopens [#5910](https://github.com/gnolang/gno/issues/5910).
- The header of [`loopvar_struct_field_2.gno`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/tests/files/loopvar_struct_field_2.gno#L4) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/loopvar_struct_field_2.gno#L4) says the test checks that composite keys are not renamed. It now checks that the rename is undone.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6037-map-composite-key-in-loop/1-390bffe90/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gnovm/tests/files/maplit15.gno:1-13 [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/maplit15.gno#L1-L13)
Missing test: nothing committed fails when the exclusion drop lands without the suffix trim, or the trim without the drop. A nested map literal keyed on the loop variable covers both directions in one case. A map key read inside a closure covers the one shape where a wrong binding prints a wrong number instead of failing to preprocess.

<details><summary>test cases</summary>

`gnovm/tests/files/loopvar_composite_key_nested.gno`:

```go
package main

type T struct{ i int }

type M map[int]T

func main() {
	for i := 0; i < 2; i++ {
		m := M{i: {i: i + 4}}
		println(m[i].i)
	}
}

// Output:
// 4
// 5
```

`gnovm/tests/files/loopvar_map_key_closure.gno`:

```go
package main

func main() {
	var fns []func()
	for i := 0; i < 3; i++ {
		fns = append(fns, func() {
			println(map[int]int{i: i * 2}[i])
		})
	}
	for _, fn := range fns {
		fn()
	}
}

// Output:
// 0
// 2
// 4
```
</details>

## gnovm/pkg/gnolang/preprocess.go:1279-1286 [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1279-L1286)
Nit: after [line 1280](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1280) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1280), `fname` and `n.Name` hold the same string, so switching the `isUpper` and panic arguments to `fname` changes nothing. The block still spells one value two ways, `n.Name` at 1281 and `fname` at 1283 and 1286, which costs a reader a check.

## gnovm/pkg/gnolang/preprocess.go:1279 [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1279)
Suggestion: `".loopvar"` is written out by hand in five places, and nothing ties the two that apply the rename, [line 298](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L298) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L298) and [line 364](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L364) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L364), to the undo here. Change the suffix and [`debugger.go:773`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/debugger.go#L773) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/debugger.go#L773) silently stops matching, reporting `name X not declared` instead of the cause.
