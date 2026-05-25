// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5169 -R gnolang/gno && git checkout f2b889f8
curl -fsSL -o tm2/pkg/bft/backup/pr5169_reader_short_read_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5169-block-backup-restore-websocket/1-f2b889f/tests/pr5169_reader_short_read.go
go test -v -run TestRoundTripLargeBlock ./tm2/pkg/bft/backup/
rm tm2/pkg/bft/backup/pr5169_reader_short_read_test.go
*/
package backup

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/stretchr/testify/require"
)

// TestRoundTripLargeBlock writes a single block whose serialized form is
// larger than the zstd-decoder/tar internal buffers, then reads it back.
//
// reader.go:128-131 calls `r.Read(blockBz)` exactly once. `archive/tar.Reader.Read`
// is allowed to return a short count with `err == nil`; over a streaming
// zstd reader backed by a real file, short reads happen for any entry whose
// size exceeds the decoder's internal buffer (~128 KB in practice).
// The unread bytes stay zero-filled in blockBz, then amino.Unmarshal either
// returns a truncated Block (silent state corruption) or fails outright
// (restore aborts mid-chain).
//
// Result on f2b889f8: round-trip of a ~600 KB block returns "data mismatch
// at <byte index ~130 KB>" — the first ~130 KB matches, the rest is zero.
//
// Fix: replace r.Read(blockBz) with io.ReadFull(r, blockBz) in reader.go.
func TestRoundTripLargeBlock(t *testing.T) {
	dir := t.TempDir()

	bigTx := make([]byte, 600_000)
	for i := range bigTx {
		bigTx[i] = byte(i % 251)
	}

	require.NoError(t, WithWriter(dir, 0, 0, nil, func(start int64, write Writer) error {
		return write(&types.Block{
			Header: types.Header{Height: 1, ChainID: "x"},
			Data:   types.Data{Txs: types.Txs{types.Tx(bigTx)}},
		})
	}))

	require.NoError(t, WithReader(dir, 1, 1, func(reader Reader) error {
		return reader(func(b *types.Block) error {
			require.Len(t, b.Txs, 1)
			got := []byte(b.Txs[0])
			require.Equal(t, len(bigTx), len(got), "block tx truncated by short read")
			require.Equal(t, bigTx, got, "block tx corrupted by short read")
			return nil
		})
	}))
}
