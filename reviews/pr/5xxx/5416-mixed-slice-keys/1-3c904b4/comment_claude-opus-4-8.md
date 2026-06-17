# Review: PR #5416
Event: REQUEST_CHANGES

## Body
Verified on 3c904b4: restoring the base to master's dense 342 reproduces the pre-PR charge, confirming the unified handler kept the deleted sparse opcode's base (966) rather than a refit cost. Also confirmed a non-const slice key fails the Go typechecker before reaching the new interpreter guard, so the tightening blocks no deployable realm.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5416-mixed-slice-keys/1-3c904b4/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/op_expressions.go:476-477 [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L476)
Two consecutive blank lines between `doOpArrayLit` and `doOpSliceLit` fail `gofmt`, and `main / lint` is red because of it. Delete one blank line.

## gnovm/pkg/gnolang/machine.go:1330 [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/machine.go#L1330)
The unified opcode kept the deleted sparse base (966) instead of the dense 342, so `[]int{1, 2, 3}` now costs 1059 gas where master charged 426, a 2.5x jump on every plain slice literal. The slope (31) matches the benchmark, so only the base is wrong: refit it for the unified handler, or keep 342.

## gnovm/tests/files/slice6.gno:9 [↗](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice6.gno#L9)
Missing trailing newline.

## gnovm/tests/files/slice7.gno:20 [↗](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice7.gno#L20)
Missing trailing newline.

## gnovm/tests/files/slice8.gno:30 [↗](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice8.gno#L30)
No filetest covers an explicit key colliding with a leading unkeyed element, e.g. `[]int{1, 0: 2}`, which should panic with `duplicate index 0 in array or slice literal`. Add one so the slice path's duplicate-index detection can't regress silently.

## gnovm/tests/files/composite16.gno:19-20 [↗](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/composite16.gno#L19)
Add a line to the PR description noting the interpreter now rejects non-const slice keys to match the Go typechecker. This changes only the typecheck-skipping filetest path, so no deployable realm is affected.
</content>
