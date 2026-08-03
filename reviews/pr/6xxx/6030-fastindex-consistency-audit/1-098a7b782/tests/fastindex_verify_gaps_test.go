/* Run: from a gno checkout:
gh pr checkout 6030 -R gnolang/gno && git checkout 098a7b782
curl -fsSL -o gno.land/cmd/gnoland/fastindex_verify_gaps_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6030-fastindex-consistency-audit/1-098a7b782/tests/fastindex_verify_gaps_test.go
go test -v -run 'TestFastindexVerify_NothingAudited|TestFastindexVerify_CorruptRecord' -tags goleveldb ./gno.land/cmd/gnoland/
rm gno.land/cmd/gnoland/fastindex_verify_gaps_test.go
*/

// Two paths the command's own tests leave open: a data directory holding no
// store at all, and a stamp-current index carrying an unreadable record.
// At 098a7b782 the first exits 0 after creating gnolang.db, and the second is
// the exit-1 branch the command exists for, with no test behind it.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/config"
	"github.com/gnolang/gno/tm2/pkg/bptree"
	"github.com/gnolang/gno/tm2/pkg/commands"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	_ "github.com/gnolang/gno/tm2/pkg/db/goleveldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runVerifyGaps(t *testing.T, dataDir string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	io := commands.NewTestIO()
	io.SetOut(commands.WriteNopCloser(&out))
	err := execFastindexVerify(&fastindexVerifyCfg{dataDir: dataDir, dbBackend: testBackend}, io)
	return out.String(), err
}

// A db directory with no gnolang store: the audit walked nothing, so it must
// not report success.
func TestFastindexVerify_NothingAudited(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, config.DefaultDBDir)
	require.NoError(t, os.MkdirAll(dbDir, 0o755))

	out, err := runVerifyGaps(t, dir)
	t.Logf("output: %q", out)

	// SHOULD: an empty store is a failed audit, not an OK one.
	assert.Error(t, err)

	// SHOULD: a read-only audit creates nothing in the target directory.
	entries, rerr := os.ReadDir(dbDir)
	require.NoError(t, rerr)
	assert.Empty(t, entries)
}

// A stamp-current index whose record no longer passes its checksum is the
// corruption class the command exists to catch.
func TestFastindexVerify_CorruptRecord(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, config.DefaultDBDir)

	raw, err := dbm.NewDB("gnolang", testBackend, dbDir)
	require.NoError(t, err)
	prefixed := dbm.NewPrefixDB(raw, []byte(mainStorePrefix))
	tree := bptree.NewMutableTreeWithDB(prefixed, 1000, bptree.NewNopLogger(), bptree.FastIndexOption(true))

	_, err = tree.Set([]byte("k"), []byte("authoritative"))
	require.NoError(t, err)
	_, _, err = tree.SaveVersion()
	require.NoError(t, err)

	// Damage the persisted record in place, leaving the stamp current.
	fastKey := append([]byte{bptree.PrefixFast}, 'k')
	rec, err := prefixed.Get(fastKey)
	require.NoError(t, err)
	require.NotEmpty(t, rec)
	rec[len(rec)/2] ^= 0xFF
	require.NoError(t, prefixed.Set(fastKey, rec))
	require.NoError(t, raw.Close())

	out, err := runVerifyGaps(t, dir)
	t.Logf("output: %q", out)

	assert.Error(t, err)
	assert.Contains(t, out, "corrupt")
}
