# PR [#6003](https://github.com/gnolang/gno/pull/6003): fix(tm2/wal): fix BenchmarkWalRead size mismatch between writer/reader

URL: https://github.com/gnolang/gno/pull/6003
Author: ygd58 | Base: master | Files: 1 | +7 -7
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 32ca59929 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-6003 32ca59929`

**TL;DR:** The consensus crash-recovery log has read benchmarks at seven message sizes, and five of them were switched off because they always errored. The reader was built with a fixed 64 KB limit while the writer scaled its limit to the message, so anything bigger than 64 KB was rejected on read. This PR gives the reader the writer's limit and switches the five benchmarks back on.

**Verdict: NEEDS DISCUSSION** — the size fix is correct and minimal, but re-enabling the 1 GB case takes the package's benchmark run from 46 MB to 12.8 GB of resident memory, which is a maintainer call (1 Warning, 1 Missing test, 1 Nit, 1 Suggestion).

## Summary

[`benchmarkWalRead`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal_test.go#L234-L265) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L234-L265) built its writer with `int64(n)+64` and its reader with the package constant [`maxTestMsgSize`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal_test.go#L69) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L69), fixed at 64 KB. [`ReadMessage`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal.go#L619-L621) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal.go#L619-L621) rejects any record longer than its own limit, so every size from 100 KB up failed with the `DataCorruptionError` quoted in [issue #910](https://github.com/gnolang/gno/issues/910), and the five affected benchmarks carried `b.Skip("TODO: benchmark failing")`. The PR hoists the writer's limit into a `maxSize` variable, passes it to both ends, and drops the five skips. Reader and writer now bound the same quantity, `len(amino.MustMarshalSized(twm))`, so a record the [writer accepts](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal.go#L488-L491) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal.go#L488-L491) can never be refused on read.

## Glossary

- write-ahead log (WAL): the consensus append-only log (`tm2/pkg/bft/wal`) replayed after a crash; writer and reader each carry their own max record size.
- Amino: gno's deterministic serialization codec (`tm2/pkg/amino`); `MustMarshalSized` produces the record body the WAL length check bounds.

## Fix

Before, the reader's limit was a constant and the writer's scaled with the payload, so the two disagreed above 64 KB. After, [`maxSize := int64(n) + 64`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal_test.go#L243) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L243) feeds both [`NewWALWriter`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal_test.go#L244) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L244) and [`NewWALReader`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal_test.go#L259) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L259). The load-bearing constraint is that both limits are compared against the same amino-sized byte count, measured at 41 to 49 bytes above `n` across 512 B to 100 MB, so the 64-byte allowance is not tight at any benchmarked size.

## Benchmarks / Numbers

Whole-package run, `-bench BenchmarkWalRead -benchmem`, default benchtime, one process:

| size | ns/op | B/op | allocs/op |
|---|---|---|---|
| 512 B | 3,421 | 8,136 | 16 |
| 10 KB | 24,098 | 88,344 | 21 |
| 100 KB | 212,491 | 845,659 | 54 |
| 1 MB | 3,150,344 | 8,437,365 | 365 |
| 10 MB | 23,007,453 | 84,158,704 | 3,442 |
| 100 MB | 171,073,941 | 842,729,728 | 34,171 |
| 1 GB | 2,297,302,710 | 8,631,233,096 | 349,595 |

| package benchmark run | peak RSS | wall |
|---|---|---|
| merge-base d14a03770 (five sizes skipped) | 46 MB | 3.3 s |
| 32ca59929 (all seven) | 12,847 MB | 25.8 s |
| 32ca59929, `BenchmarkWalRead1GB` alone | 12,313 MB | 18 s |

Consensus itself opens the WAL at [`maxMsgSize = 1048576`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/consensus/reactor.go#L29) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/consensus/reactor.go#L29), so the top three rows sit 10x, 100x and 1024x above the largest record a node's WAL will ever hold.

## Critical (must fix)

None.

## Warnings (should fix)

- **[benchmark run needs more memory than the machine has]** `tm2/pkg/bft/wal/wal_test.go:291-293` — `BenchmarkWalRead1GB` peaks at 12.3 GB resident for a single iteration, so the package's benchmark run goes from 46 MB to 12.8 GB and is OOM-killed under a 12 GiB budget.
  <details><summary>details</summary>

  One iteration allocates 8.63 GB across 349,595 allocations and peaks at 12,313 MB resident; the whole package run reaches 12,847 MB against 46 MB at the merge-base. Under a 12 GiB cgroup budget the process is OOM-killed rather than slowed. The size the consensus WAL actually enforces is [1 MB](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/consensus/reactor.go#L29) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/consensus/reactor.go#L29), set when [`OpenWAL`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/consensus/state.go#L400) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/consensus/state.go#L400) constructs it, so the 1 GB row measures a regime the WAL rejects at the writer. `tm2/pkg/iavl` already separates the two audiences, keeping the memory-hungry cases in a [`fullbench`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/iavl/Makefile#L60-L68) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/iavl/Makefile#L60-L68) target described as needing lots of memory and running locally. Fix: decide whether the 1 GB case should run by default, and gate it if not. [repro](comment_claude-opus-4-8.md)
  </details>

## Nits

- **[comment records the defect instead of the rule]** `tm2/pkg/bft/wal/wal_test.go:239-242` — four lines narrating the old failure where one line stating the rule would do: the reader's limit must equal the writer's. Not posted, no change needed; no enabled linter in [`.github/golangci.yml`](https://github.com/gnolang/gno/blob/32ca59929/.github/golangci.yml#L12-L34) · [↗](../../../../../.worktrees/gno-review-6003/.github/golangci.yml#L12-L34) covers comment wording and the meaning is unchanged either way.

## Missing Tests

- **[the repaired path still never runs in CI]** `tm2/pkg/bft/wal/wal_test.go:275-277` — outside the benchmarks nothing in this package round-trips a record above 64 KB, and the tm2 job runs `go test ./...` with no `-bench`, so CI never exercises what this PR repaired.
  <details><summary>details</summary>

  [`TestWALWriterReader`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal_test.go#L41-L67) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L41-L67) writes two messages with empty `Data` at the fixed 64 KB limit, and [`TestWALWrite`](https://github.com/gnolang/gno/blob/32ca59929/tm2/pkg/bft/wal/wal_test.go#L98-L121) · [↗](../../../../../.worktrees/gno-review-6003/tm2/pkg/bft/wal/wal_test.go#L98-L121) exercises only the writer at that same limit, accepting a payload just under it and rejecting one at it. The tm2 workflow runs [`go test -covermode=set -timeout 30m ... ./...`](https://github.com/gnolang/gno/blob/32ca59929/.github/workflows/_ci-go.yml#L124) · [↗](../../../../../.worktrees/gno-review-6003/.github/workflows/_ci-go.yml#L124), which compiles benchmarks and never runs them, so a future regression in large-record framing stays invisible until someone runs `-bench` by hand. A 100 KB round-trip costs a hundredth of a second and closes the gap: [`wal_large_message_test.go`](tests/wal_large_message_test.go), green at 32ca59929. Fix: add a test that writes and reads back a record above 64 KB.
  </details>

## Suggestions

- **[a writer failure would surface as an unrelated read error]** `tm2/pkg/bft/wal/wal_test.go:251` — the `enc.Write` result is discarded, so an oversized message leaves an empty buffer and the benchmark reports EOF from `ReadMessage` instead of `msg is too big`.
  <details><summary>details</summary>

  This is the pre-existing line, unchanged by the diff, and it is why [issue #910](https://github.com/gnolang/gno/issues/910) read as a reader-side corruption problem in the first place: the two ends disagree about a limit, and only the reader ever says so. Checking the write result costs one line and names the right end when the 64-byte allowance is one day too small. `errcheck` is not in the enabled linter set, so nothing flags it. Not posted: line 251 falls between the diff hunks, so it carries no valid review anchor.
  </details>

## Verified

- Reverting the fix reproduces issue #910 verbatim: changing the reader back to `NewWALReader(buf, maxTestMsgSize)` makes `BenchmarkWalRead100KB` fail with `DataCorruptionError[length 102445 exceeded maximum possible value of 65536 bytes]`, byte-identical to the log in the issue.
- The 64-byte allowance is not tight: the amino-sized record exceeds `n` by 41 bytes at 512 B and 10 KB, 45 at 100 KB and 1 MB, and 49 at 10 MB and 100 MB; the 1 GB benchmark runs green, so the writer's limit does not trip there either.
- Memory measured, not inferred: `BenchmarkWalRead1GB` peaks at 12,313 MB resident through `getrusage(RUSAGE_CHILDREN)`, the whole package benchmark run at 12,847 MB, and the same run at the merge-base d14a03770 at 46 MB. Under `systemd-run -p MemoryMax=12G -p MemorySwapMax=0` the 1 GB benchmark is OOM-killed, the journal recording `kernel OOM killer killed some processes in this unit`.
- `go test ./tm2/pkg/bft/wal/` green at 32ca59929 (four tests, 8.1 s); all seven benchmarks green; `gofmt -l` and `go vet` clean on the package.

## Open questions

- The 10 MB and 100 MB rows are also above the consensus WAL's 1 MB ceiling, at 84 MB and 843 MB per iteration. Folded into the 1 GB Warning rather than posted separately, since the decision is the same one.
- CI test workflows have not run on this PR: the fork PR is held at "Pending initial approval by a review team member", so only the bot checks report. Not posted, nothing the author can do.
