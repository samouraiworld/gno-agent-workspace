# Review: PR #5811
Event: APPROVE

## Body
Looks good, and the round-1 races are gone. Verified on 870501716: `go test -race -short ./gnovm/pkg/gnolang/` is clean on both TestStdlibs and TestFiles where it fails on master — restoring the `fallbackAllocator` global or dropping the uverse seal brings the races back. The `fallbackAllocator`→`nil` swap is gas-neutral: that allocator never carried a gas meter, so a `nil` allocator charges identically and consensus gas is unchanged. `-p 1` and `-p N` return identical pass/fail sets.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5811 -R gnolang/gno
# race-clean at this head; the same two commands on master both FAIL
# (CI never runs -race, so this regression is otherwise invisible):
go test -race -short -timeout 30m -run 'TestStdlibs' ./gnovm/pkg/gnolang/
go test -race -short -timeout 30m -run 'TestFiles'   ./gnovm/pkg/gnolang/
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	1228.089s   # TestStdlibs, 0 races (master: 6, all fallbackAllocator)
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	RUNs        # TestFiles, 0 races (master: 12, fallbackAllocator + debug enabled)
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5811-parallelize-test-suites/2-870501716/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
