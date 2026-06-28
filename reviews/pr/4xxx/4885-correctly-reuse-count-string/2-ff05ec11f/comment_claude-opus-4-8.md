# Review: PR #4885
Event: REQUEST_CHANGES

## Body
The string-tracking logic is unchanged from the last commit and still holds. Reverting the GC inline charge reproduces the double-count. A slice-only GC visit charges the full source backing once. Verified on ff05ec11f.

The six `alloc_*.gno` filetests fail on this head. CI is green because the gnovm job ran against an older `refs/pull/4885/merge`. Both this head and a fresh merge into current master fail.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4885 -R gnolang/gno
go test -count=1 -run 'TestFiles/^alloc_' ./gnovm/pkg/gnolang/
```

```
--- FAIL: TestFiles/alloc_7.gno
    -MemStats:  Allocator{maxBytes:100000000, bytes:6288}
    +MemStats:  Allocator{maxBytes:100000000, bytes:6896}
--- FAIL: TestFiles/alloc_13a.gno
    -===before GC, MemStats:  Allocator{maxBytes:50000, bytes:4456}
    +===before GC, MemStats:  Allocator{maxBytes:50000, bytes:12004}
# alloc_0 alloc_1 alloc_7a alloc_13 also FAIL
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/4xxx/4885-correctly-reuse-count-string/2-ff05ec11f/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/tests/files/alloc_7.gno:16 [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_7.gno#L16)
Asserts 6288 but the head produces 6896. The master merge changed allocation sizes, so this number is stale. alloc_0, alloc_1, alloc_7a are off by the same +608. Re-pin against the current head.

## gnovm/tests/files/alloc_13a.gno:55-56 [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13a.gno#L55)
The asserted before/after of 4456 / 7392 is identical to alloc_13, yet this test runs a 1052-byte string and alloc_13 runs a 20-byte one. The head produces 12004 / 8444. Re-run and re-pin.

## gnovm/tests/files/alloc_13a.gno:9-18 [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13a.gno#L9)
The header cites "Observed: after = 8396" and "11956". These match neither the asserted output 7392 nor the real output 8444. Refresh the worked example when the output line is re-pinned.

## gnovm/pkg/gnolang/uverse.go:217 [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/uverse.go#L217)
A `StringValue` built directly here never passes through `TrackString`, so `CountStringBytes` returns false and GC charges only its header, never the backing. Same direct construction at `values.go:2816` and `values.go:2278`. Bounded today since user strings route through `NewString`, but a future direct-construction site would silently undercount and no test would catch it, so make the tracking invariant explicit.

## gnovm/pkg/gnolang/alloc_test.go:187 [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc_test.go#L187)
Every byte-counting test builds its string through `alloc.NewString`, so all are tracked by construction and none covers the untracked-`StringValue` path. A test that builds a `StringValue` directly, makes it a GC root, visits it through `GCVisitorFn`, and asserts the count includes the backing would lock the tracking invariant.

## gnovm/pkg/gnolang/alloc.go:394-414 [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L394)
`TrackString`'s `slices.Insert` is O(N) per call, so O(N²) over N distinct live backings in a cycle, and it is unmetered. Not DoS-reachable at gas-bounded N, around 21ms at 200K distinct strings. A note that the structure is insert-heavy would help the next reader.

## gnovm/pkg/gnolang/garbage_collector.go:207 [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L207)
"GetShallowSize returns header-only for strings" reads as a general claim but holds only for `StringValue`. Restate as "for `StringValue`, header only; bytes charged inline below".
