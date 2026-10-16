package gnoweb_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A same-size rewrite keeping the mtime of the first read is served on the
// next request, since that read was within staticFileSettle. Passes at
// 4ee1b62ff; fails with the zeroing block at alias_static.go:190-192 removed.
func TestNewStaticFileAliasRereadsUnsettledFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home.md")
	require.NoError(t, os.WriteFile(path, []byte("# One"), 0o644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	target, err := gnoweb.NewStaticFileAlias(path)
	require.NoError(t, err)

	// Rewritten within the same timestamp tick: same size, same mtime.
	require.NoError(t, os.WriteFile(path, []byte("# Two"), 0o644))
	require.NoError(t, os.Chtimes(path, info.ModTime(), info.ModTime()))
	assert.Equal(t, "# Two", gnoweb.ResolveAlias(target).Value)
}
