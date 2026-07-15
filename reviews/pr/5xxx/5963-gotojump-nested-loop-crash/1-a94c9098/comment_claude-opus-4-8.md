# Review: PR [#5963](https://github.com/gnolang/gno/pull/5963)
Event: APPROVE

## Body
Verified on a94c90986: re-adding the deleted second truncation reproduces the slice bounds out of range [:-1] crash, and output matches go run byte-for-byte across for, range, and switch-clause frame crossings for 3- to 6-deep nesting.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5963-gotojump-nested-loop-crash/1-a94c9098/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/tests/files/goto10.gno:3 [↗](../../../../../.worktrees/gno-review-5963/gnovm/tests/files/goto10.gno#L3)
Missing test: this covers `ForStmt` frames only, but the fix also repairs the crash across `RangeStmt` and `SwitchClauseStmt` frames, which no golden asserts. [`findGotoLabel` counts all three frame types](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/preprocess.go#L4676), and existing `goto*` filetests cross a range or switch-clause frame only at depth 1, which never underflows. Both cases below panic `[:-1]` on master and pass at head.

<details><summary>test cases</summary>

`gnovm/tests/files/goto_nested_range.gno`:
```go
package main

// Backward goto out of three nested range loops to a func-body label.

func main() {
	i := 0
	s := []int{0, 1}
top:
	println("i =", i)
	for range s {
		for range s {
			for range s {
				i++
				if i < 3 {
					goto top
				}
			}
		}
	}
	println("done", i)
}

// Output:
// i = 0
// i = 1
// i = 2
// done 10
```

`gnovm/tests/files/goto_switch_loops.gno`:
```go
package main

// Backward goto out of two loops nested in a switch clause to a func-body label.

func main() {
	i := 0
top:
	println("i =", i)
	switch {
	case true:
		for a := 0; a < 2; a++ {
			for b := 0; b < 2; b++ {
				i++
				if i < 3 {
					goto top
				}
			}
		}
	}
	println("done", i)
}

// Output:
// i = 0
// i = 1
// i = 2
// done 6
```
</details>

## gnovm/pkg/gnolang/machine.go:2444-2449 [↗](../../../../../.worktrees/gno-review-5963/gnovm/pkg/gnolang/machine.go#L2444)
Suggestion: the comment says why re-subtracting `depthFrames` is wrong, but not why truncating to `fr.NumStmts` alone is enough. The [`GOTO` handler](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/op_exec.go#L715) owns the final `m.Stmts` length, and the deleted line mirrored [`PopFrameAndReset`](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/machine.go#L2464), which legitimately pops one stmt because it pops one frame. Naming both would stop a future maintainer from restoring the symmetry and reintroducing the underflow.
