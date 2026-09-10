# Review: [#6137](https://github.com/gnolang/gno/pull/6137)
Posted: https://github.com/gnolang/gno/pull/6137#pullrequestreview-5171417532
Event: COMMENT

## Body
> AI review, claude-opus-5 at xhigh, [skills](https://github.com/davd-gzl/skills) · Status: APPROVE

## gnovm/tests/files/defer_multivalue_arg.gno:18 [gh](https://github.com/gnolang/gno/blob/861df2c42/gnovm/tests/files/defer_multivalue_arg.gno#L18) · [↗](../../../../../.worktrees/gno-review-6137/gnovm/tests/files/defer_multivalue_arg.gno#L18) [posted](https://github.com/gnolang/gno/pull/6137#discussion_r3982748512)
Missing test: `defer called a nil function` also hit a variadic callee, a bound method and a builtin at the base, and none is covered.

<details><summary>test cases</summary>

Green at 861df2c42, and at the merge base with `op_call.go` reverted each of the three fails with `unexpected panic: runtime error: defer called a nil function`.

```go
package main

type counter struct{ n int }

func (c counter) add(a, b int) { println("method:", c.n+a+b) }

func two() (int, int) { return 7, 8 }

func variadic(a int, rest ...int) { println("variadic:", a, len(rest), rest[0]) }

func main() {
	defer println(two())           // runs 3rd: builtin callee
	defer counter{n: 1}.add(two()) // runs 2nd: bound method callee
	defer variadic(two())          // runs 1st: variadic callee
	println("body")
}

// Output:
// body
// variadic: 7 1 8
// method: 16
// 7 8
```
</details>

## gnovm/pkg/gnolang/op_call.go:752-753 [gh](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L752-L753) · [↗](../../../../../.worktrees/gno-review-6137/gnovm/pkg/gnolang/op_call.go#L752) [posted](https://github.com/gnolang/gno/pull/6137#discussion_r3982748522)
Suggestion: `default:` returns before the trailing [`m.PopValue()`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L756), so it leaves the func value and its `numArgs` arguments on the value stack.

```suggestion
	default:
		m.PopValues(numArgs + 1)
		m.pushPanic(typedString(fmt.Sprintf("invalid defer function call: %v", cv)))
```

<details><summary>what it changes</summary>

Nothing reachable: `defer int64(x)` is the only shape found to land in `default:`, and the go typechecker rejects it with `defer requires function call, not conversion int64(x) (value of type int64)`. With the suggestion applied a filetest on that source still reports `invalid defer function call: typeval{int64}` and the same typecheck error, and the 35 filetests matching `TestFiles/defer` stay green.
</details>

## SKIP gnovm/tests/files/defer_nil_func_args.gno:3-8 [gh](https://github.com/gnolang/gno/blob/861df2c42/gnovm/tests/files/defer_nil_func_args.gno#L3-L8) · [↗](../../../../../.worktrees/gno-review-6137/gnovm/tests/files/defer_nil_func_args.gno#L3)
Test: the header claims this file asserts a balanced value stack, and the file passes with `op_call.go` reverted to the merge base.

Skipped: the only ask is a wording change to a code comment, and no filetest can pin the balance.
