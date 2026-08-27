# PR [#4885](https://github.com/gnolang/gno/pull/4885): fix(gnovm): correctly reuse/count string in alloc and gc

URL: https://github.com/gnolang/gno/pull/4885
Author: ltzmaxwell | Base: master | Files: 23 | +1230 -34
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: `6c2839ca0` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-4885 6c2839ca0`
Overview: [visual overview](../overview.html)

Round 3. The head moved from `ff05ec11f` to `6c2839ca0` over a master merge and four branch commits. Every round-2 finding is answered except one fixture-header nit: the six stale `alloc_*.gno` goldens were re-pinned and now pass, the sorted-slice range set became a treap, and an audit of `StringValue` producers added two mint sites plus an ADR. The round-2 Warning on undefended direct `StringValue` construction is resolved as a documented invariant rather than an enforced one. One new item, on the query path, is below.

## Overview

The GnoVM caps how much memory one transaction may hold, and rebuilds that total from scratch at each garbage collection by walking live values. Strings broke the rebuild in both directions: a string loaded back from storage was never counted at all, and two strings sharing one Go byte array were each billed for the whole array. This branch records the address extent of every string's byte array in the allocator and bills each extent once per collection cycle, so aliases and slices resolve to the extent that contains them. Since round 2 it also replaced the sorted slice holding those extents with a balanced tree, made the extent set independent of whether Go decided to share a byte array, and routed two paths that handed Gno strings out untracked through the allocator.

**Verdict: COMMENT** — nothing measured blocks the merge; the one open item is where the load-path tracking lands on the query path, and `main / lint` is red on `go fix` formatting in the new test file (1 suggestion, 4 nits).

## Verify first

- [`gno.land/pkg/sdk/vm/keeper.go:1778`](https://github.com/gnolang/gno/blob/6c2839ca0/gno.land/pkg/sdk/vm/keeper.go#L1778) · [↗](../../../../../.worktrees/gno-review-4885/gno.land/pkg/sdk/vm/keeper.go#L1778) — the query machine gets a fresh allocator while the store keeps its own, so `fillTypesOfValue` and `GCVisitorFn` use different allocators here and nowhere else. Build a machine this way, load a string through `fillTypesOfValue`, visit it through `GCVisitorFn`, and confirm the tally you get is the one you want.
- [`gnovm/pkg/gnolang/alloc.go:431-439`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/alloc.go#L431-L439) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L431-L439) — `trackString` clones a string whose extent overlaps a tracked one, which is what makes the extent set independent of Go's backing-sharing choices. Delete the clone branch and run `go test -run 'TestTrackString_OverlapClones' ./gnovm/pkg/gnolang/` to see it is the only thing holding that property.

## Summary

`StringValue.GetShallowSize()` returns the header alone; the byte count comes from `Allocator.CountStringBytes`, which resolves a pointer to the tracked extent containing it and returns that extent's full length once per cycle. Since round 2 the extents live in a treap keyed by start ([`string_ranges.go:6-8`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/string_ranges.go#L6-L8) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/string_ranges.go#L6-L8)) instead of a sorted slice, which removes the O(N) insert round 2 measured, and `trackString` clones on overlap ([`alloc.go:431-432`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/alloc.go#L431-L432) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L431-L432)) so every mint owns exactly one extent whatever the Go toolchain did. Two producers that handed out untracked strings now mint through the allocator: `MsgCall` string arguments at [`convert.go:52`](https://github.com/gnolang/gno/blob/6c2839ca0/gno.land/pkg/sdk/vm/convert.go#L52) · [↗](../../../../../.worktrees/gno-review-4885/gno.land/pkg/sdk/vm/convert.go#L52), which were attacker-controlled and charged nothing, and the realm-handle fields at [`uverse.go:272-273`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/uverse.go#L272-L273) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/uverse.go#L272-L273).

## Benchmarks / Numbers

`alloc_13a.gno` post-GC total, reverting one line at a time from the head:

| tree | after GC | what it shows |
|---|---|---|
| head `6c2839ca0` | 8972 | one 1052-byte backing billed once |
| `GetShallowSize` back to header+bytes | 11076 | the same backing billed for `s`, `s1` and the package slot |

`NewString` cost against the number of live tracked strings, `-benchtime=200x` on this machine:

| live tracked strings | ns/op |
|---|---|
| 0 | 538 |
| 1 000 | 447 |
| 10 000 | 2 127 |
| 100 000 | 491 |

The 10 000 row is a single noisy sample at 200 iterations; the shape is flat, which is what the treap was for.

Post-GC recount of one live 4096-byte string loaded through `fillTypesOfValue`, when the machine allocator and the store allocator are different objects:

| tree | recount |
|---|---|
| merge-base `c04f8793d` | 4144 |
| head `6c2839ca0` | 48 |

## Suggestions

- **[load-path tracking and the GC recount use different allocators on the query path]** [`realm.go:1895`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/realm.go#L1895) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/realm.go#L1895) — `fillTypesOfValue` mints into `store.GetAllocator()`, `GCVisitorFn` asks `m.Alloc`, and [`withQueryEvalMachine`](https://github.com/gnolang/gno/blob/6c2839ca0/gno.land/pkg/sdk/vm/keeper.go#L1740) · [↗](../../../../../.worktrees/gno-review-4885/gno.land/pkg/sdk/vm/keeper.go#L1740) is the one place those are not the same object.
  <details><summary>details</summary>

  `NewMachineWithOptions` installs the passed allocator on the store only when the store has none ([`machine.go:189-190`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/machine.go#L189-L190) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/machine.go#L189-L190)), and the query's throwaway transaction store already carries one forked at [`store.go:263`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/store.go#L263) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/store.go#L263). The four transaction paths pass `gnostore.GetAllocator()` and are unaffected; `withQueryEvalMachine` passes a fresh `NewAllocator(maxAllocQuery)`. So on `vm/qeval` and everything built on it, every string loaded from storage is recounted as its 48-byte header whatever its length, measured at 48 against 4144 on the merge-base for a 4096-byte string. The bytes are not lost: they are charged to the store's allocator at load time, which this branch also introduced, and that allocator has no collect function so it hard-caps at `maxAllocTx`. So this is an accounting split rather than a hole, but the ADR's audit lists `fillTypesOfValue` as one of the three entry points that make every Gno-visible string tracked, and on this path it tracks into an allocator the collector never asks. Fix: mint into the allocator that will recount, or say in the ADR that the query machine's tally excludes stored strings by design.
  </details>

## Nits

- [`string_ranges_test.go:47`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/string_ranges_test.go#L47) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/string_ranges_test.go#L47) (also [`:112`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/string_ranges_test.go#L112) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/string_ranges_test.go#L112), [`:132`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/string_ranges_test.go#L132) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/string_ranges_test.go#L132)) — three counted loops the Go 1.26 modernizer rewrites to `for i := range n`, which is the whole of the red `main / lint` job. Reproduced with `GOTOOLCHAIN=go1.26.1 CGO_ENABLED=1 go fix -omitzero=false -diff ./pkg/gnolang/` from `gnovm/`. Not posted: `make fix` from the repo root closes it and the job already names it.
- [`alloc_13a.gno:14-17`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/tests/files/alloc_13a.gno#L14-L17) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13a.gno#L14-L17) — the worked example still cites the round-2 numbers, `12004` for the before value and `Observed: after = 8444`, against the file's own assertions of `11980` and `8972` two lines below. Round 2 raised the same drift and the re-pin refreshed the assertions without the header. Not posted: a fixture comment changes no behaviour.
- [`string_ranges.go:124`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/string_ranges.go#L124) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/string_ranges.go#L124) — `ranges()` has one caller, [`string_ranges_test.go:35`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/string_ranges_test.go#L35) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/string_ranges_test.go#L35), so it can sit in the test file beside it. Not posted: no enabled linter in [`.github/golangci.yml:13`](https://github.com/gnolang/gno/blob/6c2839ca0/.github/golangci.yml#L13) · [↗](../../../../../.worktrees/gno-review-4885/.github/golangci.yml#L13) flags a method a test uses.
- [`realm.go:1895-1896`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/realm.go#L1895-L1896) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/realm.go#L1895-L1896) — `cv2` is assigned and returned on the next line; `return store.GetAllocator().NewString(string(cv))` says the same in one line, matching the arms around it. Not posted: cosmetic, no enabled linter.

## Verified

- Reverting `StringValue.GetShallowSize()` to header plus bytes moves `alloc_13a.gno`'s post-GC total from 8972 to 11076, so the fixture pins the dedup rather than a number that happens to match.
- Reverting [`convert.go:52`](https://github.com/gnolang/gno/blob/6c2839ca0/gno.land/pkg/sdk/vm/convert.go#L52) · [↗](../../../../../.worktrees/gno-review-4885/gno.land/pkg/sdk/vm/convert.go#L52) to `tv.SetString(gno.StringValue(arg))` makes `TestConvertArgToGno_StringArgIsCharged` report `"0" is not greater than "1000"`, so a 1000-byte call argument was charged nothing before this commit.
- A 100 000-byte package-level string literal costs 3 042 165 gas on a `gnokey maketx call`, against 1 025 033 for an 8-byte one in the same shape, so a literal's bytes are paid for on the call path even though the collector's recount treats them as header-only. This is the ADR's "charged once at preprocess" claim, measured.
- `go test -run 'TestFiles/^alloc_' ./gnovm/pkg/gnolang/` passes at this head, closing the round-2 Warning that all six fixtures failed.
- The `overview.html` simulator's mirror of the accounting, including the new clone-on-overlap rule, reproduces the PR's own unit-test scenarios; run with `node` before this file was written.

## Existing threads

- thehowl's two threads on the `map[uintptr]int64` draft, `6638a4168`, are closed by the containment design and were reacted to in round 1. Not re-posting.

## Open questions

- The extent set is per-`Allocator`, and [`ClearObjectCache`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/store.go#L1426-L1427) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/store.go#L1426-L1427) resets the byte total between the messages of one transaction without dropping the extents, while each message's machine restarts `GCCycle` at zero. An extent stamped with the previous message's cycle number therefore survives one `retain` it should not, and a string minted outside `NewString` that landed on a recycled address inside it would be billed that extent's length. Not posted: reaching it needs Go to recycle a specific address inside a one-cycle window and I could not construct it.
- The ADR labels the approach experimental because backing identity is a host-runtime concept. The clone-on-overlap rule removes the toolchain's say for strings minted through `NewString`; strings from the deliberately untracked producers, panic text and const-folded literals, still resolve by containment into whatever extent holds their address. Not posted: the ADR already states the residual risk.
