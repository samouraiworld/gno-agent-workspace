/* Run: from a gno checkout:
gh pr checkout 5169 -R gnolang/gno && git checkout f2b889f84
curl -fsSL -o tm2/pkg/bft/backup/pr5169_short_read_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5169-block-backup-restore-websocket/1-f2b889f/tests/pr5169_short_read_test.go
go test -v -run 'TestBackupRoundTripPreservesLargeBlocks|TestBackupCorruptionRate' ./tm2/pkg/bft/backup/
rm tm2/pkg/bft/backup/pr5169_short_read_test.go
*/

// backupReader.readChunk issues one r.Read into a full-size buffer and ignores
// the byte count, so a tar entry the zstd decoder does not hand over in one
// call comes back with its tail still zeroed.
// At f2b889f84 there is no size that is merely "too big": zstd.NewReader
// decodes concurrently by default, so the same block round-trips intact on one
// run and corrupts on the next. Twenty sequential runs per size gave 400 KB
// 20/20 intact, 524 KB 15/20, 600 KB 9/20, 900 KB 4/20; with another decode
// running alongside, 400 KB fails too. Treat no size as safe.
// Filling the read completely makes both tests pass at every size.

package backup

import (
	"bytes"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/stretchr/testify/require"
)

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

// A block larger than the decoder hands over in one call must still survive the
// archive. gno permits 2 MB of block data and 1 MB in a single tx, so this size
// is inside consensus limits rather than an extreme.
func TestBackupRoundTripPreservesLargeBlocks(t *testing.T) {
	t.Parallel()

	require.True(t, roundTrip(t, 600_000), "block tx corrupted between writer and reader")
}

// Repeats each size, because a single run of a corrupting size passes often
// enough to look green by luck.
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
