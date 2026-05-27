# PR #5169: feat: Blocks backup restore WebSocket

URL: https://github.com/gnolang/gno/pull/5169
Author: Villaquiranm | Base: master | Files: 28 | +1941 -18
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5169 f2b889f` (then `gh -R gnolang/gno pr checkout 5169` inside it)

**Verdict: REQUEST CHANGES** — `reader.read` truncates blocks larger than the zstd decoder buffer (silent state corruption on restore); the WS `backup` endpoint is unauthenticated and unrate-limited (DoS vector); `WriteSync` errors in the restore batch loop are dropped.

## Summary

Adds a block-level backup/restore pipeline: a WebSocket RPC method (`backup`) streams full blocks, a CLI tool (`tm2backup`) writes them to chunked tar+zstd archives (100 blocks/chunk), and a new `gnoland restore` subcommand replays them through `ApplyBlock` with optional commit verification. Sound architecture — the headline finding is a one-line read bug ([`reader.go:128-131`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/reader.go#L128-L131) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/reader.go#L128-L131)) that silently truncates any block bigger than the zstd decode buffer (~128 KB in practice), which on a real chain means most non-trivial blocks. Combined with swallowed `WriteSync` errors in the restore loop and an unguarded long-running RPC method, the patch needs another pass before it can be trusted to restore chain state.

## Glossary

- **`WithWriter` / `WithReader`** — top-level archive entry points in `tm2/pkg/bft/backup`; callback-style, hold the directory `flock`.
- **`writerImpl.write` / `backupReader.readChunk`** — per-chunk plumbing (tar + zstd encoding/decoding).
- **`BlockchainReactor.Restore`** — the verify-and-apply loop in `tm2/pkg/bft/blockchain/reactor.go`.
- **`SaveBlockWithBatch`** — new batched variant of `BlockStore.SaveBlock`; the old `SaveBlock` is now a wrapper.
- **`ResultBackupBlock`** — RPC stream payload: `{height, block, done}`.
- **`backup` route** — `rpc.NewWSRPCFunc(env.BackupBlocks, "start,end")` in `routes.go`, WebSocket-only, mounted unconditionally.

## Fix

Before: nodes had no block-level backup; `tx-archive` covered tx replay but discarded consensus state. After: a node streams blocks over its WebSocket RPC, `tm2backup` writes a chunked compressed archive, and `gnoland restore` replays them with commit verification. The load-bearing invariant is the N/N+1 pairing — block N's commit lives inside N+1's `LastCommit`, so `Restore` always buffers one block ahead and stops one block short of the backup's end ([`reactor.go:412-444`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L412-L444) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L412-L444)). `BlockStore.SaveBlock` becomes a one-shot wrapper around the new `SaveBlockWithBatch`, and `BlockStoreStateJSON.Save` switches from `dbm.DB` to `dbm.Batch` (caller-visible signature change).

## Critical (must fix)

- **[silent truncation of large blocks]** [`tm2/pkg/bft/backup/reader.go:128-131`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/reader.go#L128-L131) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/reader.go#L128-L131) — `r.Read(blockBz)` reads once; tar over streaming zstd returns short and restore decodes a zero-padded block.
  <details><summary>details</summary>

  `archive/tar.Reader.Read` follows the `io.Reader` contract: it may return `0 < n < len(p)` with `err == nil`. Under a `zstd.Reader` reading from a real `*os.File`, that happens whenever the entry exceeds the decoder's internal buffer (~128 KB on this version of `klauspost/compress`). The unread suffix of `blockBz` stays zero-filled, then `amino.Unmarshal(blockBz, block)` either decodes a truncated block (silent corruption) or returns an error mid-chain (restore aborts). I confirmed this end-to-end against `WithWriter`/`WithReader` — a single 600 KB block round-trips with the tail zeroed from offset ~0x20800. Real gno blocks routinely exceed 128 KB once they carry MsgRun txs or non-trivial `MsgAddPackage` payloads; this bug is reached on the first such block, not on an edge case. Fix: replace with `io.ReadFull(r, blockBz)` and treat anything but `nil`/`io.EOF` as an error. Adversarial test in `tests/pr5169_reader_short_read.go` (fails on `f2b889f8`).

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5169 -R gnolang/gno
  cat > tm2/pkg/bft/backup/pr5169_short_read_test.go <<'EOF'
  package backup
  import (
      "testing"
      "github.com/gnolang/gno/tm2/pkg/bft/types"
      "github.com/stretchr/testify/require"
  )
  func TestPR5169ShortRead(t *testing.T) {
      dir := t.TempDir()
      tx := make([]byte, 600_000)
      for i := range tx { tx[i] = byte(i % 251) }
      require.NoError(t, WithWriter(dir, 0, 0, nil, func(_ int64, w Writer) error {
          return w(&types.Block{Header: types.Header{Height: 1, ChainID: "x"}, Data: types.Data{Txs: types.Txs{types.Tx(tx)}}})
      }))
      require.NoError(t, WithReader(dir, 1, 1, func(r Reader) error {
          return r(func(b *types.Block) error {
              require.Equal(t, tx, []byte(b.Txs[0]), "block tx corrupted by short read")
              return nil
          })
      }))
  }
  EOF
  go test -v -run TestPR5169ShortRead ./tm2/pkg/bft/backup/
  rm tm2/pkg/bft/backup/pr5169_short_read_test.go
  ```
  </details>

- **[dropped WriteSync error in restore loop]** [`tm2/pkg/bft/blockchain/reactor.go:393-403`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L393-L403) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L393-L403) — `blockBatch.WriteSync()` return value discarded; disk-full or I/O error becomes a phantom-committed batch.
  <details><summary>details</summary>

  `saveBatch` calls `blockBatch.WriteSync()` without capturing the error, then proceeds to close and re-create the batch. If the underlying DB rejects the write (out of space, disk full, locked, transient I/O failure), `Restore` keeps iterating, `ApplyBlock` keeps mutating the in-memory state DB, and `BlockStore.height` (which `SaveBlockWithBatch` mutates before the batch commits — see [`store.go:228-230`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/store/store.go#L228-L230) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/store/store.go#L228-L230)) advances past blocks that never reached disk. On restart, the block store reloads from `BlockStoreStateJSON` (also written via the same dropped batch) and silently rewinds — but only after the user has watched a "successful" restore. Fix: capture `WriteSync`'s error, abort restore on failure, and surface it to the caller.
  </details>

- **[unauthenticated, unbounded `backup` RPC]** [`tm2/pkg/bft/rpc/core/routes.go:44`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/routes.go#L44) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/routes.go#L44) — any WebSocket client can ask any node to stream its full block history; no auth, no rate limit, not behind `--rpc.unsafe`.
  <details><summary>details</summary>

  `"backup": rpc.NewWSRPCFunc(env.BackupBlocks, "start,end")` is registered unconditionally — there's no gating on `unsafe` like the profiler/mempool-flush endpoints. The handler runs synchronously inside the WS readRoutine: a single client invoking `backup 1, latest` against a public RPC ties up that connection for the duration of the stream and forces the server to Amino-decode every historical block via `LoadBlock` ([`backup.go:48`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/backup.go#L48) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/backup.go#L48)). On a busy node this is hours of CPU + I/O per anonymous request, with no concurrency limit across connections. Two minimum bars before this can be on a public RPC: (a) gate behind `--rpc.unsafe` or a dedicated `--rpc.backup-enabled` config flag (default off), (b) limit max range per request and concurrent backup streams per node. The ADR at [`gno.land/adr/pr5169_block_backup_restore.md`](https://github.com/gnolang/gno/blob/f2b889f/gno.land/adr/pr5169_block_backup_restore.md) · [↗](../../../../../.worktrees/gno-review-5169/gno.land/adr/pr5169_block_backup_restore.md) doesn't mention auth/rate-limiting at all — that's the gap.
  </details>

## Warnings (should fix)

- **[ajnavarro raised this — see PR review comment]** [`gno.land/adr/pr5169_block_backup_restore.md`](https://github.com/gnolang/gno/blob/f2b889f/gno.land/adr/pr5169_block_backup_restore.md) · [↗](../../../../../.worktrees/gno-review-5169/gno.land/adr/pr5169_block_backup_restore.md) — WebSocket transport vs cacheable HTTP `/metablocks/<size>/<idx>` was raised by [@ajnavarro](https://github.com/gnolang/gno/pull/5169#discussion_r1556034712) on 2026-04-14 and not addressed in the ADR.
  <details><summary>details</summary>

  The ADR section "WebSocket Streaming Endpoint" justifies WS over HTTP with three bullets (streaming, no pagination, backpressure). ajnavarro's alternative — two plain HTTP GETs over immutable metablocks that can sit behind a CDN, no client binary required — directly contradicts those bullets (metablocks are pre-built so streaming-vs-pagination is moot; HTTP range requests handle backpressure). The ADR should either incorporate the comparison or explicitly state why it was rejected. This isn't blocking on its own but lands close to the unauth/DoS finding above: the chosen transport is harder to put behind a CDN/edge cache, which is the standard mitigation for "anyone can pull the chain history".
  </details>

- **[iterator error masked by leftover save]** [`tm2/pkg/bft/blockchain/reactor.go:445-451`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L445-L451) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L445-L451) — `err = blocksIterator(...)`; then `if blocksInBatch > 0 { err = saveBatch() }` overwrites it.
  <details><summary>details</summary>

  When the iterator returns a non-nil error AND there are leftover blocks in the batch, the leftover-save's result (nil on success) silently shadows the iterator's error. Restore reports success despite a mid-stream failure (e.g. a corrupt archive entry that surfaced as an iterator error). Fix: don't overwrite `err`, or `return errors.Join(iterErr, saveBatch())`.
  </details>

- **[in-process `Height()` advances ahead of disk]** [`tm2/pkg/bft/store/store.go:228-230`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/store/store.go#L228-L230) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/store/store.go#L228-L230) — `bs.height = height` is set inside `SaveBlockWithBatch` before the batch is `WriteSync`'d.
  <details><summary>details</summary>

  In the pre-PR path, `SaveBlock` did `bs.db.SetSync(nil, nil)` (flush) immediately after the height update, so an external `Height()` reader saw a value that was already durable. Now, batches accumulate up to 1000 blocks before `WriteSync`; the in-process `Height()` reflects a value that has no on-disk backing for up to ~1000 blocks. Any concurrent component that queries `Height()` during restore (e.g. the RPC `status` endpoint, if served) gets a value that isn't crash-safe. Either move the `bs.height` mutation into a finalize step (called when the batch is durably written) or document the window. On restart NewBlockStore self-corrects from `BlockStoreStateJSON`, so the bug is window-only — but worth being explicit.
  </details>

- **[restore wedges WS connection]** [`tm2/pkg/bft/rpc/core/backup.go:54`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/backup.go#L54) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/backup.go#L54) — `WriteRPCResponses` is BLOCKING on the connection's writeChan; entire WS conn is serialized behind the backup stream.
  <details><summary>details</summary>

  Per [`handlers.go:650-658`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/lib/server/handlers.go#L650-L658) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/lib/server/handlers.go#L650-L658), `WriteRPCResponses` blocks until the writer goroutine drains the channel. On a slow/congested client (or any client that's not actively reading), the BackupBlocks loop stalls — and because `readRoutine` processes each request synchronously inline (no per-request goroutine before line 802), the same WS connection can't service any other RPC method (health, status, subscribe) until the entire backup finishes. Operators running a single shared WS connection for monitoring + backup will see their monitoring stall. Either fire backup in its own goroutine, use `TryWriteRPCResponses` with explicit backpressure, or document that backup requires a dedicated connection.
  </details>

- **[breaking signature change `BlockStoreStateJSON.Save(dbm.Batch)`]** [`tm2/pkg/bft/store/store.go:273`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/store/store.go#L273) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/store/store.go#L273) — was `Save(dbm.DB)`; any external caller fails to compile.
  <details><summary>details</summary>

  This is an exported method on an exported type in `tm2/pkg/bft/store`. The PR doesn't note a breaking change, and there's no compatibility shim. If any downstream tooling (or out-of-tree fork) calls `BlockStoreStateJSON.Save(myDB)` they'll break. Either keep `Save(dbm.DB)` and add a sibling `SaveWithBatch(dbm.Batch)`, or call out the break explicitly in the PR body. Lower-priority since this is an internal interface, but no signal was given.
  </details>

- **[no resume-time chain-identity check]** [`tm2/pkg/bft/backup/util.go:59-87`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/util.go#L59-L87) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/util.go#L59-L87) — `info.json` carries only `Version`/`StartHeight`/`EndHeight`; pointing `tm2backup` at a different node mid-resume silently mixes chains.
  <details><summary>details</summary>

  When `tm2backup` resumes, it does not validate that the remote node serves the same chain as the partial archive. If an operator accidentally points it at a different `chain-id` (e.g. `staging` vs `mainnet`), the archive gains blocks from a different chain with no error. The reader has no way to detect this either — block heights still parse, the tar entries still decode. Fix: include `ChainID` + `Genesis hash` in `info.json` and validate on resume + restore start.
  </details>

- **[`getStartHeight` overwrites partial last chunk without warning]** [`tm2/pkg/bft/backup/writer.go:219-241`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/writer.go#L219-L241) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/writer.go#L219-L241) — comment says "we simply overwrite the latest chunk if it is partial because it's not expensive".
  <details><summary>details</summary>

  Resuming with `state.EndHeight = 50` (partial first chunk) returns `height = 1` and the next call to `openChunk` overwrites the first 50 blocks. Functionally correct for resuming the same chain at the same node — but combined with the no-chain-identity-check above, an operator who points `tm2backup` at a different node sees blocks 1..50 silently replaced with the new node's blocks. Either warn loudly on resume that partial chunks will be rewritten, or skip the partial chunk and start the new chunk at `state.EndHeight + 1` (slightly less efficient, but no overwrite surprise).
  </details>

- **[hardcoded batch size 1000]** [`tm2/pkg/bft/blockchain/reactor.go:392`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L392) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L392) — `batchSize := 1000`; no configurability and not an exported constant.
  <details><summary>details</summary>

  For chains with large blocks (e.g. heavy `MsgAddPackage` traffic), 1000 in-memory blocks can dominate RAM. There's no flag, no env var, no constant, no comment justifying the value. Lift to a package constant and consider a `--restore-batch-size` flag on `gnoland restore`.
  </details>

## Nits

- [`tm2/pkg/bft/backup/writer.go:108`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/writer.go#L108) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/writer.go#L108) — `return errors.New("poisoned")` after `b.poisoned` carries no context; the original failure is gone. At minimum store the first error and return it via `fmt.Errorf("poisoned: %w", b.firstErr)`.
- [`tm2/pkg/bft/backup/writer.go:23`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/writer.go#L23) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/writer.go#L23) — typo "concurently" → "concurrently" (same in `reader.go:20`).
- [`tm2/pkg/bft/backup/util.go:64-71`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/util.go#L64-L71) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/util.go#L64-L71) — `readState` returns the sentinel `{StartHeight:-1, EndHeight:-1, Version:"v1"}` when `info.json` is missing; reader then bails out at [`reader.go:41-43`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/reader.go#L41-L43) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/reader.go#L41-L43) with the bare message "invalid backup state". Surface a clearer error for the case of "no `info.json` in this directory" so users don't think the file is malformed.
- [`contribs/tm2backup/.gitignore`](https://github.com/gnolang/gno/blob/f2b889f/contribs/tm2backup/.gitignore) · [↗](../../../../../.worktrees/gno-review-5169/contribs/tm2backup/.gitignore) — missing trailing newline.
- [`gno.land/cmd/gnoland/restore.go:53`](https://github.com/gnolang/gno/blob/f2b889f/gno.land/cmd/gnoland/restore.go#L53) · [↗](../../../../../.worktrees/gno-review-5169/gno.land/cmd/gnoland/restore.go#L53) — default `--backup-dir` is `blocks-backup` (relative) — easy to footgun if invoked from the wrong working directory; the `--data-dir` default is similarly relative but at least documented. Worth noting in `--help`.
- [`tm2/pkg/bft/blockchain/reactor.go:381-388`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L381-L388) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L381-L388) — `var first, second *types.Block` then later `second = block; ...; first = second` — `second` is dead after the first assignment because `first = second` happens unconditionally. Could be `first = block` directly. Minor.
- [`tm2/pkg/bft/rpc/core/backup.go:12`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/backup.go#L12) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/backup.go#L12) — `env.Logger.Info("On BackupBlocks", ...)` looks like leftover debugging; demote to `Debug`.

## Missing Tests

- **[no large-block round-trip]** [`tm2/pkg/bft/backup/backup_test.go:28-39`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/backup_test.go#L28-L39) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/backup_test.go#L28-L39) — `testWriteBlocks` only uses empty `types.Block{Header: ...}` (~hundreds of bytes amino-marshaled). The whole bug class above is invisible to the existing tests.
  <details><summary>details</summary>

  Add a round-trip test with a block ≥ 200 KB serialized (e.g. a fat `Data.Txs`). Without it, the short-read finding could regress on any compress-library bump. The repro in `tests/pr5169_reader_short_read.go` is a starting point.
  </details>

- **[no restore-with-WriteSync-error coverage]** [`tm2/pkg/bft/blockchain/reactor.go:393-403`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L393-L403) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L393-L403) — `TestRestore` uses an in-memory DB that never fails.
  <details><summary>details</summary>

  Inject a `dbm.DB` whose `Batch.WriteSync` returns an error after N writes and assert that `Restore` propagates it. Without this, the dropped-error finding above won't be caught by regression tests.
  </details>

- **[no resume-after-crash test]** [`tm2/pkg/bft/backup/backup_test.go`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/backup_test.go) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/backup_test.go) — `TestInfo` resumes from a clean shutdown, not a mid-chunk crash.
  <details><summary>details</summary>

  Simulate a crash: kill the writer in the middle of a chunk (leaving `next-chunk.tar.zst` half-written), then resume. The atomic-rename design suggests this should work, but it's untested.
  </details>

- **[no concurrent-writer test]** [`tm2/pkg/bft/backup/util.go:19-33`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/util.go#L19-L33) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/util.go#L19-L33) — `flock` lock isn't exercised by a test.
  <details><summary>details</summary>

  Two simultaneous `WithWriter` calls should fail the second with the documented lock error. One-liner test using two goroutines.
  </details>

## Suggestions

- [`tm2/pkg/bft/rpc/core/backup.go:11-65`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/backup.go#L11-L65) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/backup.go#L11-L65) — consider streaming via the `StreamableResult` interface ([`handlers.go:813`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/lib/server/handlers.go#L813) · [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/lib/server/handlers.go#L813)) rather than firing N `WriteRPCResponses` calls inside a loop. Stream lets the WS writer goroutine pull blocks at its own pace instead of the RPC fn blocking on each `WriteRPCResponses`.
  <details><summary>details</summary>

  The codebase already supports streamable RPC results (see the `wsStreamRequest`/`streamChan` plumbing). Returning a streaming result from `BackupBlocks` (a) avoids the synchronous-write-per-block blocking and (b) frees the WS conn to service interleaved requests. Worth investigating in a follow-up if not in this PR.
  </details>

- [`contribs/tm2backup/main.go:99`](https://github.com/gnolang/gno/blob/f2b889f/contribs/tm2backup/main.go#L99) · [↗](../../../../../.worktrees/gno-review-5169/contribs/tm2backup/main.go#L99) — pass `ctx` into `backup.WithWriter` so SIGINT during long runs cleanly finalizes the current chunk and returns. Right now Ctrl-C kills the dialer's read loop ungracefully.
- [`gno.land/adr/pr5169_block_backup_restore.md`](https://github.com/gnolang/gno/blob/f2b889f/gno.land/adr/pr5169_block_backup_restore.md) · [↗](../../../../../.worktrees/gno-review-5169/gno.land/adr/pr5169_block_backup_restore.md) — add a "Security considerations" section covering: auth on `backup` route, rate limits, chain-id binding in `info.json`. The current "Known Limitations" section only covers operational gotchas.

## Questions for Author

- Why is `backup` mounted unconditionally rather than gated on `--rpc.unsafe`? Is the intent that all public RPC nodes should serve backups by default?
- Has restore been tested end-to-end against a real-world chunk archive with blocks larger than ~200 KB? The unit tests only round-trip empty headers, so the silent-truncation bug isn't caught.
- Did the design consider the HTTP+CDN alternative ajnavarro proposed? The ADR doesn't explain why WS was preferred over `GET /metablocks/<size>/<idx>` which is cacheable, untrusted-safe (restore verifies anyway), and needs no client binary.
- Is the breaking signature change to `BlockStoreStateJSON.Save` intentional, or should the existing `Save(dbm.DB)` be preserved as a wrapper?
