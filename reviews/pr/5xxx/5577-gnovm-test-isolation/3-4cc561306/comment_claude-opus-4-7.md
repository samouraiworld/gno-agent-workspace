# Review: PR #5577
Event: REQUEST_CHANGES

## Body
Verified on 4cc561306: the head moved only by a `master` merge since the previous push, leaving the PR's own files unchanged.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5577-gnovm-test-isolation/3-4cc561306/claude-opus-4-7_davd-gzl.md · [↗](./claude-opus-4-7_davd-gzl.md)

## gnovm/pkg/gnolang/machine.go:785-794 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L785)
This doc comment describes [`instantiatePackageFiles`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L661) but sits above the exported [`NewPackageInstance`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L795), so `go doc` and IDE hover show the wrong description for a public API.

## gnovm/pkg/test/test.go:429 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L429)
This per-test machine, and the pre-loop one at [line 406](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/test/test.go#L406), comes from the pool via [`Machine()`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/test/test.go#L75) but is never returned with [`m.Release()`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L174), so each package run drops N+1 machines to the GC instead of reusing them.

## gnovm/pkg/gnolang/machine.go:831 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L831)
[`IsTestFile`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/mempackage.go#L154) matches `_test.gno` and `_filetest.gno`, so this skip also drops filetest `init()`s, but [`RunMemPackageSkipTestFileInits`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L309) documents skipping only `_test.gno`. No current caller passes filetests, so it is latent.

## gnovm/pkg/test/test.go:396-399 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L396)
The non-`IsAll` branch falls through with `tmpkg = mpkg`. Right for today's All and Integration callers, but a future Prod or Test type keeps the wrong file set silently.

## gnovm/pkg/gnolang/machine.go:795-808 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L795)
[`NewPackageInstance`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L795) works only because file blocks are immutable after preprocessing: the copied top-level `FuncValue`s resolve through the package block, not the per-file blocks it rebuilds. Nothing records that invariant, so a change letting file blocks hold mutable state would silently break per-test isolation.

## gnovm/pkg/test/test.go:302 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/test/test.go#L302)
The three new txtars are all same-package `counter`; none exercises the `xxx_test` integration path at [line 302](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/test/test.go#L302), where the fresh `cacheObjects` and `save=false` keep seed-init mutations from leaking. Add a `foo_test` package whose `init()` mutates `foo`, then assert clean state.

## gnovm/pkg/gnolang/machine.go:683-697 [↗](../../../../../.worktrees/gno-review-5577/gnovm/pkg/gnolang/machine.go#L683)
On the fresh path [`fdeclared`](https://github.com/gnolang/gno/blob/4cc561306/gnovm/pkg/gnolang/machine.go#L687) starts empty, so the sort re-resolves every cross-file var dependency per test, and nothing covers it. A two-file txtar, `a.gno: var X = Y + 1` and `b.gno: var Y = 2`, would lock init order across re-instantiation.
