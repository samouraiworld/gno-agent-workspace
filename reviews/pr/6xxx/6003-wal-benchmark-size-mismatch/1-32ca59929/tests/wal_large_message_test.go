/* Run: from a gno checkout:
gh pr checkout 6003 -R gnolang/gno && git checkout 32ca59929
curl -fsSL -o tm2/pkg/bft/wal/wal_large_message_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6003-wal-benchmark-size-mismatch/1-32ca59929/tests/wal_large_message_test.go
go test -v -run 'TestWALWriterReaderLargeMessage' ./tm2/pkg/bft/wal/
rm tm2/pkg/bft/wal/wal_large_message_test.go
*/

// Round-trips a message above the 64KB maxTestMsgSize through WALWriter and
// WALReader. Only the benchmarks cover sizes above 64KB, and `go test ./...`
// never runs benchmarks, so CI has no coverage of this path today.

package wal

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmtime "github.com/gnolang/gno/tm2/pkg/bft/types/time"
	"github.com/gnolang/gno/tm2/pkg/random"
)

func TestWALWriterReaderLargeMessage(t *testing.T) {
	t.Parallel()

	const n = 100 * 1024 // above maxTestMsgSize
	maxSize := int64(n) + 64

	msg := TimedWALMessage{
		Time: tmtime.Now().Round(time.Second).UTC(),
		Msg:  TestMessage{Height: 1, Round: 1, Data: random.RandBytes(n)},
	}

	buf := new(bytes.Buffer)
	require.NoError(t, NewWALWriter(buf, maxSize).Write(msg))

	decoded, meta, err := NewWALReader(buf, maxSize).ReadMessage()
	require.NoError(t, err)
	require.Nil(t, meta)
	assert.Equal(t, msg.Msg, decoded.Msg)
	assert.Equal(t, msg.Time, decoded.Time)
}
