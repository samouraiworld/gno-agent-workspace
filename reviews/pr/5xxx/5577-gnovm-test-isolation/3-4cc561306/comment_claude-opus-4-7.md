# Review: PR #5577
Event: REQUEST_CHANGES

## Body
Verified on 4cc561306: the head moved only by a `master` merge since the previous push, leaving the PR's own files unchanged.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5577-gnovm-test-isolation/3-4cc561306/claude-opus-4-7_davd-gzl.md · [↗](./claude-opus-4-7_davd-gzl.md)

## gnovm/pkg/gnolang/machine.go:785-794 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L785)
This doc comment describes [`instantiatePackageFiles`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L661), but it sits directly above the new exported [`NewPackageInstance`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L795), so `go doc` and IDE hover show the wrong description for a public API. Rewrite it to describe `NewPackageInstance`.

## gnovm/pkg/test/test.go:429 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L429)
This per-test machine, and the pre-loop one at [line 406](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/test/test.go#L406), is pulled from the pool but never returned with [`m.Release()`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L174), so each package run drops N+1 machines for the GC instead of reusing them. Release each machine before it is reassigned.

## gnovm/pkg/gnolang/machine.go:831 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L831)
[`IsTestFile`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/mempackage.go#L154) matches both `_test.gno` and `_filetest.gno`, so the new exported [`RunMemPackageSkipTestFileInits`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L309) also drops filetest `init()`s. The only current caller is filetest-free, so this is decay risk for a future caller, not a current bug; narrow this check to `_test.gno`, or rename the API to say it skips filetest inits too.

## gnovm/pkg/test/test.go:396-399 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L396)
The `if IsAll` filter is correct for today's two call sites, but a future call site passing an already-filtered Test or Prod type falls through silently with `tmpkg = mpkg` and keeps the wrong file set. Make the non-`IsAll` branch explicit, handling Integration and panicking on anything else, or widen the comment to cover that case.

## gnovm/pkg/gnolang/machine.go:795-808 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L795)
`NewPackageInstance` relies on top-level func closures resolving names through the package block, not the per-file blocks it rebuilds, so the copied `FuncValue`s keep working without re-parenting. That holds only because file blocks are immutable after preprocessing; a one-line comment recording the invariant would keep a future change that lets file blocks hold mutable state from silently breaking per-test isolation.

## gnovm/pkg/test/test.go:302 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L302)
All three new txtars use a same-package (`package counter`) layout; none exercises this `xxx_test` integration path, where the fresh `cacheObjects` plus `save=false` chain is what keeps seed-init mutations from leaking. Add a txtar with a `foo_test` package whose `init()` mutates `foo`, plus a follow-up test asserting clean state, to lock that path in.

## gnovm/pkg/gnolang/machine.go:683-697 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L683)
On the fresh-instantiation path `fdeclared` starts empty, so the topological sort re-resolves every cross-file var dependency from scratch on each test, and nothing exercises that. A two-file txtar, `a.gno: var X = Y + 1` and `b.gno: var Y = 2`, would verify init order survives re-instantiation.
