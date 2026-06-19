# Review: PR #5811
Event: APPROVE

## Body
Looks good, round-1 races resolved. Verified on 870501716: `go test -race -short ./gnovm/pkg/gnolang/` reports 0 data races on TestStdlibs and TestFiles. Pre-fix d668a22c2 failed both with 6 and 12 races, from `fallbackAllocator` and the debug `enabled` flag. The `fallbackAllocator`→`nil` swap is gas-neutral: that allocator never had a gas meter, so a `nil` allocator charges identically and consensus gas is unchanged. `-p 1` and `-p N` give identical pass/fail sets.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5811 -R gnolang/gno
# 0 data races at this head; CI never runs -race, so the round-1 regression was otherwise invisible:
go test -race -short -timeout 30m -run 'TestStdlibs' ./gnovm/pkg/gnolang/ 2>&1 | grep -c 'DATA RACE'
go test -race -short -timeout 30m -run 'TestFiles'   ./gnovm/pkg/gnolang/ 2>&1 | grep -c 'DATA RACE'
```

```
0   # TestStdlibs: 0 races (d668a22c2: 6)
0   # TestFiles:   0 races (d668a22c2: 12)
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5811-parallelize-test-suites/2-870501716/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
