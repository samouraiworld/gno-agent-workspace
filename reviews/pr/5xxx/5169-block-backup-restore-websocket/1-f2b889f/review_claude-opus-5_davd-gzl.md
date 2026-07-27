# PR [#5169](https://github.com/gnolang/gno/pull/5169): feat: Blocks backup restore WebSocket

URL: https://github.com/gnolang/gno/pull/5169
Author: Villaquiranm | Base: master | Files: 28 | +1941 -18
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: f2b889f84 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5169 f2b889f84`

**TL;DR:** Adds a way to copy a chain's blocks off a running node into compressed files, and a way to replay those files into a fresh node so it catches up without syncing from peers.

**Verdict: REQUEST CHANGES** — the archive reader can hand back a block with its tail zeroed, and it does so at random rather than above a fixed size, so a restore can silently produce a different chain on one run than on the next (2 Critical, 5 Warnings, 1 Suggestion).

## Verify first

- [`tm2/pkg/bft/backup/reader.go:128-131`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/reader.go#L128-L131) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/reader.go#L128-L131) — everything the restore path produces rests on this read returning a whole block. Drop [`tests/pr5169_short_read_test.go`](tests/pr5169_short_read_test.go) into `tm2/pkg/bft/backup/` and run it: the round trip loses data at sizes the chain permits, and not on every run.
- [`tm2/pkg/bft/blockchain/reactor.go:394`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L394) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L394) — confirm `WriteSync`'s error reaches the operator, by making the block DB fail a write mid-restore and checking that `gnoland restore` exits non-zero.

## Summary

Three pieces. A WebSocket RPC method `backup` streams full blocks from a running node. A new `contribs/tm2backup` binary pulls that stream into chunked tar archives compressed with zstd, 100 blocks per chunk, resumable. A new `gnoland restore` subcommand replays those chunks through `ApplyBlock`, verifying each block's commit against the next one's `LastCommit`, which is why it always stops one block short of the archive's end. `BlockStore.SaveBlock` becomes a wrapper over a new batched `SaveBlockWithBatch`, and restore commits in batches of 1000 rather than flushing per block.

The archive reader is where it breaks. `readChunk` sizes a buffer from the tar header and then issues a single `Read`, discarding the count, so whatever the zstd decoder did not hand over in that one call stays zero. Because `zstd.NewReader` decodes concurrently by default, how much arrives in one call is not a function of block size: the same 524 KB block survived 15 of 20 round trips and a 900 KB block survived 4 of 20. gno permits [1 MB in a single tx](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/types/params.go#L22) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/types/params.go#L22) and [2 MB of block data](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/types/params.go#L25) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/types/params.go#L25), so this is inside the rules rather than at an extreme.

Reading order: `tm2/pkg/bft/backup/` (writer, reader, util) first, since the archive format decides everything downstream; then `tm2/pkg/bft/rpc/core/backup.go` and `routes.go` for the serving side; then `tm2/pkg/bft/store/store.go` for the batching change; then `tm2/pkg/bft/blockchain/reactor.go` for the restore loop; then `gno.land/cmd/gnoland/restore.go` and `contribs/tm2backup/` as the two front ends.

## Diagram

```
node                    tm2backup                  archive              gnoland restore
────                    ─────────                  ───────              ───────────────
BackupBlocks ──WS──►  writes chunk  ──►  <n>.tar.zst   ──►  readChunk ──► ApplyBlock
LoadBlock(h)          100 blocks/chunk    tar entry           │            verify commit
                                          = 1 block           │            of block N using
                                                              │            N+1's LastCommit
                                                    single Read, count
                                                    ignored: tail of the
                                                    entry stays zeroed
```

## Fix

`readChunk` allocates `blockBz` from the tar header's `Size` and calls `r.Read(blockBz)` once, treating any error other than `io.EOF` as fatal but never checking how many bytes landed. `io.Reader` explicitly permits a short read, and `tar.Reader` over a concurrent `zstd.Reader` takes that liberty. Filling the buffer completely, and treating a partial fill as an error, is the whole fix; the surrounding amino decode then either succeeds on a whole block or fails loudly instead of decoding zero padding into real fields.

## Benchmarks / Numbers

Round trips surviving intact, 20 sequential runs per size, one block per archive:

| tx size | intact |
|---|---|
| 100 KB | 20/20 |
| 400 KB | 20/20 |
| 524 KB | 15/20 |
| 600 KB | 9/20 |
| 900 KB | 4/20 |

With a second decode running concurrently, 400 KB also fails. The counts themselves move between runs: a later repeat of the 524 KB row gave 12/20.

## Critical (must fix)

- **[a restore can produce a different chain each time it runs]** [`tm2/pkg/bft/backup/reader.go:128-131`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/reader.go#L128-L131) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/reader.go#L128-L131) — one `Read` with the count discarded, so a block comes back with its tail zeroed, at random rather than above a fixed size.
  <details><summary>details</summary>

  `io.Reader` allows `0 < n < len(p)` with a nil error, and [`tar.Reader` over the concurrent `zstd.Reader` created at `reader.go:99`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/reader.go#L99) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/reader.go#L99) does exactly that. The unread suffix stays zero, and [`amino.Unmarshal`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/reader.go#L133) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/reader.go#L133) then either decodes a block missing its tail or errors mid-archive. There is no threshold to design around: the numbers in the table above come from repeating one size 20 times, and the same size both passes and fails. Commit verification does not catch it either, because the hash being checked is recomputed from the truncated block. Fix: fill the buffer completely and treat a partial read as an error. [`tests/pr5169_short_read_test.go`](tests/pr5169_short_read_test.go) fails at f2b889f84 and passes once the read is filled.
  </details>

- **[a failed write is reported as a successful restore]** [`tm2/pkg/bft/blockchain/reactor.go:394`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L394) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L394) — `blockBatch.WriteSync()` returns an error that is discarded, and restore continues as if 1000 blocks had reached disk.
  <details><summary>details</summary>

  [`WriteSync() error`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/db/types.go#L82) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/db/types.go#L82) is the only signal that a batch of up to 1000 blocks was persisted. With it dropped, a full disk or an I/O error leaves `ApplyBlock` still advancing the state DB and [`bs.height`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/store/store.go#L229) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/store/store.go#L229) still climbing over blocks that never landed, and the command exits zero. The operator learns about it on the next start, when the block store reloads from `BlockStoreStateJSON` and rewinds. Fix: abort the restore on a failed batch write.
  </details>

## Warnings (should fix)

- **[an unauthenticated request can pin a node for hours]** [`tm2/pkg/bft/rpc/core/routes.go:44`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/routes.go#L44) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/routes.go#L44) — `backup` is registered outside the `unsafe` gate, with no range cap and no limit on concurrent streams.
  <details><summary>details</summary>

  The [`if unsafe` block immediately below](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/routes.go#L47-L53) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/routes.go#L47-L53) is where the profiler and mempool-flush methods live; `backup` sits above it and is therefore on by default on any node serving WebSocket RPC. [`BackupBlocks`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/backup.go#L41-L60) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/backup.go#L41-L60) accepts `start=1, end=0`, which it resolves to the full history, and loops `LoadBlock` synchronously with no bound. Did not measure the cost against a large public node, so the size of the effect is unquantified here. Fix: put it behind the same gate as the other operator-only methods, or a dedicated flag defaulting to off.
  </details>

- **[a mid-archive failure is reported as success]** [`tm2/pkg/bft/blockchain/reactor.go:447-451`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L447-L451) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L447-L451) — the leftover-batch save overwrites the iterator's error with its own nil.
  <details><summary>details</summary>

  `err = blocksIterator(...)` at [`reactor.go:405`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L405) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L405) captures a real failure, such as a corrupt archive entry or a commit that did not verify. The very next statement reassigns `err` whenever any blocks are still buffered, which is the common case since the batch size is 1000. Fix: join the two errors rather than replacing one with the other.
  </details>

- **[the reported height outruns what is on disk]** [`tm2/pkg/bft/store/store.go:229`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/store/store.go#L229) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/store/store.go#L229) — `bs.height` advances inside `SaveBlockWithBatch`, up to 1000 blocks before the batch is written.
  <details><summary>details</summary>

  The pre-PR `SaveBlock` flushed immediately after updating the height, so an in-process reader of `Height()` saw a durable value. With batching, anything querying `Height()` during a restore reads a number with no on-disk backing yet. `NewBlockStore` recovers from `BlockStoreStateJSON` on restart, so the window closes by itself, but during that window the value is not crash-safe. Fix: move the height update to where the batch is known to be durable, or say in the doc comment that it is not.
  </details>

- **[an exported method changed shape with no note]** [`tm2/pkg/bft/store/store.go:273`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/store/store.go#L273) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/store/store.go#L273) — `BlockStoreStateJSON.Save` went from taking a `dbm.DB` to a `dbm.Batch`, so out-of-tree callers stop compiling.
  <details><summary>details</summary>

  Both the type and the method are exported from `tm2/pkg/bft/store`, and neither the PR body nor the [ADR](https://github.com/gnolang/gno/blob/f2b889f/gno.land/adr/pr5169_block_backup_restore.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-5169/gno.land/adr/pr5169_block_backup_restore.md#L1) mentions a break. Fix: keep the old signature alongside a batch-taking sibling, or call the break out.
  </details>

- **[nothing ties an archive to the chain it came from]** [`tm2/pkg/bft/backup/util.go:59-87`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/util.go#L59-L87) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/util.go#L59-L87) — the state file records only a version and a height range, so resuming against a different node mixes two chains into one archive.
  <details><summary>details</summary>

  `readState` validates `Version` and nothing else. Resuming a partial archive against a node on another chain appends blocks that parse, decode, and sit at plausible heights, and neither `tm2backup` nor the restore path has anything to compare them against. This compounds with [`getStartHeight`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/writer.go#L219-L241) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/writer.go#L219-L241), which rewinds to the start of a partial chunk and overwrites it, so the first blocks of that chunk are replaced by the other chain's. Fix: record the chain ID in the state file and check it on resume and on restore.
  </details>

## Nits

None.

## Missing Tests

- **[the archive tests only use blocks small enough to pass]** [`tm2/pkg/bft/backup/backup_test.go:1`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/backup_test.go#L1) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/backup_test.go#L1) — no round trip carries a block near the sizes the chain permits, which is why the read bug ships green.
  <details><summary>details</summary>

  The existing round-trip coverage uses minimal blocks, all far below the point where the decoder stops filling a single read. A size-varying round trip is what turns the Critical from invisible into a failing test, and it needs to repeat each size, since a corrupting size still passes often enough to look green on one run. [`tests/pr5169_short_read_test.go`](tests/pr5169_short_read_test.go) is that test, ready to drop into the package.
  </details>

## Suggestions

- **[one slow client stalls everything else on the connection]** [`tm2/pkg/bft/rpc/core/backup.go:54`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/backup.go#L54) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/backup.go#L54) — `WriteRPCResponses` blocks, and the loop runs inline in the connection's read path, so a backup occupies the whole WebSocket connection until it finishes.
  <details><summary>details</summary>

  A client that stops draining stalls the loop, and no other method on that connection is served meanwhile. Not a defect on its own, since a dedicated connection is a reasonable thing to ask of a backup tool, but it is worth stating: an operator sharing one connection between monitoring and backup will see monitoring stop. Fix: say so in the ADR, or run the stream off the read path.
  </details>

## Existing threads

- ajnavarro proposed cacheable HTTP metablocks over a WebSocket stream, [thread](https://github.com/gnolang/gno/pull/5169#discussion_r1556034712). The ADR's transport section argues for WebSocket without engaging the alternative. Overlaps the unauthenticated-endpoint Warning, since an immutable HTTP object is the shape that puts a CDN in front of exactly this load.

## Verified

- The read bug is not size-gated but probabilistic. Twenty sequential round trips per size gave 400 KB intact 20/20, 524 KB 15/20, 600 KB 9/20, 900 KB 4/20; a later repeat put 524 KB at 12/20, and with a second decode running alongside 400 KB failed too. An earlier measurement that reported a clean 512 KiB boundary was wrong: it compared only the returned prefix, so it scored a truncated block as intact.
- Corruption is silent rather than an error. The recovered tx is byte-identical up to the cut and zero after it, and commit verification passes because the hash is recomputed from the truncated block.
- Green at f2b889f84: `./tm2/pkg/bft/backup/...`, `./tm2/pkg/bft/rpc/core/...`, `./tm2/pkg/bft/store/...`.
- The one red CI check is `params_valset_rotation_throttle.txtar` under `main / test`, which touches none of this diff's packages; the branch is far enough behind master that the run predates several releases.

## Open questions

- The restore path stops one block short of the archive end by construction, which the PR body flags as unresolved. It follows from commit N living in block N+1, so an archive would have to carry the trailing commit separately to close it. Not posted, since it is a known limitation rather than a defect.
- `zstd.NewReader` is created with default options, so decoder concurrency scales with GOMAXPROCS. Did not check whether pinning it to one goroutine would make the corruption deterministic, and it should not matter once the read is filled correctly.
