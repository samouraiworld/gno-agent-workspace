# Review: PR #5811
Event: REQUEST_CHANGES

## Body
`-p 1` and `-p N` produce identical pass/fail sets, verified on d668a22c2 across 8 example packages including an injected `[setup failed]` case. The red CI `docs` job is unrelated: its URL linter hit a dead external link, and this PR touches no docs.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5811-parallelize-test-suites/1-d668a22c2/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/files_test.go:116 [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/files_test.go#L116)
Parallelizing the short filetests (and the stdlib suite, and `gno test -p`) writes two unsynchronized process globals, the [`fallbackAllocator`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/gnolang/alloc.go#L45) [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/alloc.go#L45) (via `copyValueWithRefs`→[`MapList.Append`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/gnolang/realm.go#L1695) [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/realm.go#L1695)) and the debug [`enabled`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/gnolang/debug.go#L203-L209) [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/debug.go#L203) flag, so `go test -race ./gnovm/pkg/gnolang/` now fails where it passes on master. The same writes ship in the default-parallel `gno test`, since `-p` defaults to GOMAXPROCS; both are value-benign today but are real data races. Give these paths per-worker or synchronized state, the same class as the `gnoBuiltinsCache` race this PR already fixed.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5811 -R gnolang/gno
# parallelized short filetests; fails at this PR's head, passes on master:
go test -race -short -run 'TestFiles/(recurse1\.gno|a1\.gno|a2\.gno|a3\.gno|a4\.gno|a5\.gno|a6\.gno|a7\.gno|a8\.gno|a9\.gno|a10\.gno|a11\.gno|a12\.gno|a13\.gno|a14\.gno|a15\.gno|a16\.gno)' ./gnovm/pkg/gnolang/
```

```
WARNING: DATA RACE
Read at 0x00c0001d0e18 by goroutine 2140:
  ...(*Allocator).Allocate()        alloc.go:304
  ...(*Allocator).AllocateMapItem() alloc.go:377
  ...(*MapList).Append()            values.go:727
  ...copyValueWithRefs()            realm.go:1695
  ...(*Realm).FinalizeRealmTransaction() / loadStdlib() / RunFiletest()
Previous write at 0x00c0001d0e18 by goroutine 2319:
  ...(*Allocator).Allocate()        alloc.go:326
  ... (same path)
--- FAIL: TestFiles/recurse1.gno
    testing.go:1712: race detected during execution of test
FAIL    github.com/gnolang/gno/gnovm/pkg/gnolang
# the full `-run TestFiles -short` and `-run TestStdlibs -short` also fail (12 and 6 races);
# the same command on master: ok, 0 races.
```
</details>
