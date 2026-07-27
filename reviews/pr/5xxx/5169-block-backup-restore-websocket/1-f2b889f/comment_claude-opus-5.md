# Review: PR [#5169](https://github.com/gnolang/gno/pull/5169)
Posted: https://github.com/gnolang/gno/pull/5169#pullrequestreview-4790428270
Event: REQUEST_CHANGES

## Body
[AI bot]

The restore path drops every signal that something went wrong: a short read, a failed batch write, and an iterator error are each discarded, so a run that rebuilt the wrong chain still exits zero. Verified on f2b889f84 that a truncated block passes commit verification, because the hash it is checked against is recomputed from the bytes that arrived.

`main / test` is red on `params_valset_rotation_throttle.txtar`, which this branch does not touch.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5169-block-backup-restore-websocket/1-f2b889f/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## tm2/pkg/bft/backup/reader.go:128-131 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/reader.go#L128-L131) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031658)
Critical: this reads once and discards the count, so the tail of a block stays zeroed whenever [the concurrent zstd reader](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/reader.go#L99) does not hand the whole entry over in one call. There is no size threshold to design around: repeating one round trip 20 times, a 524 KB block survived 15 times and a 900 KB block 4 times, and gno permits [1 MB in a single tx](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/types/params.go#L22). The restore then decodes zero padding into real fields, and the commit check passes because the hash is recomputed from the truncated block.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5169 -R gnolang/gno
cat > tm2/pkg/bft/backup/pr5169_short_read_test.go <<'EOF'
package backup

import (
	"bytes"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/stretchr/testify/require"
)

func roundTrip(t *testing.T, txSize int) bool {
	sent := make([]byte, txSize)
	for i := range sent {
		sent[i] = byte(i%251) + 1 // never zero, so zero-fill is unambiguous
	}
	block := &types.Block{
		Header: types.Header{Height: 1, ChainID: "x"},
		Data:   types.Data{Txs: types.Txs{types.Tx(sent)}},
	}
	var got []byte
	dir := t.TempDir()
	require.NoError(t, WithWriter(dir, 0, 0, nil, func(_ int64, w Writer) error { return w(block) }))
	require.NoError(t, WithReader(dir, 1, 1, func(r Reader) error {
		return r(func(b *types.Block) error { got = []byte(b.Txs[0]); return nil })
	}))
	return bytes.Equal(sent, got)
}

func TestPR5169ShortRead(t *testing.T) {
	const runs = 20
	for _, txSize := range []int{400_000, 524_000, 600_000, 900_000} {
		intact := 0
		for range runs {
			if roundTrip(t, txSize) {
				intact++
			}
		}
		require.Equal(t, runs, intact, "tx size %d survived only %d of %d round trips", txSize, intact, runs)
	}
}
EOF
go test -v -run TestPR5169ShortRead ./tm2/pkg/bft/backup/
rm tm2/pkg/bft/backup/pr5169_short_read_test.go
```

```
    Messages:   	tx size 524000 survived only 12 of 20 round trips
--- FAIL: TestPR5169ShortRead (0.13s)
FAIL	github.com/gnolang/gno/tm2/pkg/bft/backup	0.135s
```
</details>

## tm2/pkg/bft/blockchain/reactor.go:394 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L394) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031663)
Critical: [`WriteSync() error`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/db/types.go#L82) is the only signal that a batch of up to 1000 blocks reached disk, and it is discarded here. A full disk leaves `ApplyBlock` still advancing the state DB and [`bs.height`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/store/store.go#L229) still climbing over blocks that never landed, and the command exits zero. The operator finds out on the next start, when the block store reloads from `BlockStoreStateJSON` and rewinds.

## tm2/pkg/bft/blockchain/reactor.go:447-451 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/blockchain/reactor.go#L447-L451) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031666)
The leftover-batch save reassigns `err`, discarding whatever [the iterator](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/blockchain/reactor.go#L405) returned. A corrupt archive entry or a commit that failed to verify is reported as a clean restore, and with a batch size of 1000 there are almost always leftovers.

## tm2/pkg/bft/rpc/core/routes.go:44 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/routes.go#L44) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031670)
`backup` sits above [the `unsafe` gate](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/routes.go#L47-L53) that covers the profiler and mempool-flush methods, so it is on by default wherever WebSocket RPC is served. [`BackupBlocks`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/rpc/core/backup.go#L41-L60) resolves `end=0` to the full history and loops `LoadBlock` synchronously, with no cap on the range and no limit on concurrent streams.

## tm2/pkg/bft/store/store.go:229 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/store/store.go#L229) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031677)
The height advances here, up to 1000 blocks before the batch carrying those blocks is written, so anything reading `Height()` during a restore gets a value with no on-disk backing. The pre-PR path flushed immediately after this line, so the value was durable when it changed.

## tm2/pkg/bft/store/store.go:273 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/store/store.go#L273) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031684)
`BlockStoreStateJSON.Save` changed from taking a `dbm.DB` to a `dbm.Batch`. Both the type and the method are exported, and neither the PR body nor the [ADR](https://github.com/gnolang/gno/blob/f2b889f/gno.land/adr/pr5169_block_backup_restore.md?plain=1#L1) records the break.

## tm2/pkg/bft/backup/util.go:59-87 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/util.go#L59-L87) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031695)
`readState` checks the version and nothing that identifies the chain, so resuming a partial archive against a node on a different chain appends blocks that parse and sit at plausible heights. [`getStartHeight`](https://github.com/gnolang/gno/blob/f2b889f/tm2/pkg/bft/backup/writer.go#L219-L241) then rewinds to the start of the partial chunk and overwrites it, so the other chain's blocks replace the ones already there. Record the chain ID and check it on resume and on restore.

## tm2/pkg/bft/backup/backup_test.go:1 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/backup/backup_test.go#L1) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031700)
Missing test: no round trip carries a block anywhere near the sizes the chain permits, which is why the read above ships green. The test has to repeat each size, because a corrupting size still passes often enough to look fine on a single run.

<details><summary>test cases</summary>

```go
// roundTrip writes one block carrying a tx of exactly txSize bytes, reads it
// back out of the archive, and reports whether the tx survived unchanged.
func roundTrip(t *testing.T, txSize int) bool {
	t.Helper()

	sent := make([]byte, txSize)
	for i := range sent {
		sent[i] = byte(i%251) + 1 // never zero, so zero-fill is unambiguous
	}
	block := &types.Block{
		Header: types.Header{Height: 1, ChainID: "x"},
		Data:   types.Data{Txs: types.Txs{types.Tx(sent)}},
	}

	var got []byte
	dir := t.TempDir()
	require.NoError(t, WithWriter(dir, 0, 0, nil, func(_ int64, w Writer) error {
		return w(block)
	}))
	require.NoError(t, WithReader(dir, 1, 1, func(r Reader) error {
		return r(func(b *types.Block) error {
			got = []byte(b.Txs[0])
			return nil
		})
	}))
	return bytes.Equal(sent, got)
}

func TestBackupRoundTripPreservesLargeBlocks(t *testing.T) {
	t.Parallel()

	require.True(t, roundTrip(t, 600_000), "block tx corrupted between writer and reader")
}

func TestBackupCorruptionRate(t *testing.T) {
	t.Parallel()

	const runs = 20
	for _, txSize := range []int{400_000, 524_000, 600_000, 900_000} {
		intact := 0
		for range runs {
			if roundTrip(t, txSize) {
				intact++
			}
		}
		require.Equal(t, runs, intact, "tx size %d survived only %d of %d round trips", txSize, intact, runs)
	}
}
```
</details>

## tm2/pkg/bft/rpc/core/backup.go:54 [↗](../../../../../.worktrees/gno-review-5169/tm2/pkg/bft/rpc/core/backup.go#L54) [posted](https://github.com/gnolang/gno/pull/5169#discussion_r3660031706)
Suggestion: `WriteRPCResponses` blocks and this loop runs inline in the connection's read path, so a backup holds the whole WebSocket connection until it finishes and a client that stops draining stalls it. An operator sharing one connection between monitoring and backup loses monitoring for the duration.
