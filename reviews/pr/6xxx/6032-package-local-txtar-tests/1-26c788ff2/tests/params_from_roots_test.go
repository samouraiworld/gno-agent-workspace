/* Run: from a gno checkout:
gh pr checkout 6032 -R gnolang/gno && git checkout 26c788ff2
curl -fsSL -o gnovm/pkg/integration/params_from_roots_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6032-package-local-txtar-tests/1-26c788ff2/tests/params_from_roots_test.go
go test -v -run 'TestNewTestingParamsFromRoots' ./gnovm/pkg/integration/
rm gnovm/pkg/integration/params_from_roots_test.go
*/

// Two properties of NewTestingParamsFromRoots that nothing in the tree pins.
//
// files_list_drives_the_run passes: it locks the coupling the code comment
// calls out, that testscript honors Params.Files only while Params.Dir is
// empty. A later Dir default would send RunT down its directory branch and
// every discovered root would go unrun.
//
// empty_root_goes_unreported passes too, and that is the gap: it asserts the
// current behavior, where a root contributing zero scripts raises nothing
// because the guard counts the total across roots.

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestingParamsFromRoots(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(rel string) string {
		fpath := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(fpath), 0o755))
		require.NoError(t, os.WriteFile(fpath, nil, 0o644))
		return fpath
	}
	platform := write("testdata/vm.txtar")
	local := write("r/demo/foo/foo.txtar")

	t.Run("files_list_drives_the_run", func(t *testing.T) {
		t.Parallel()

		p, err := NewTestingParamsFromRoots(t,
			filepath.Join(dir, "testdata"), filepath.Join(dir, "r"))
		require.NoError(t, err)
		assert.Empty(t, p.Dir, "a non-empty Dir makes testscript ignore Files")
		assert.Equal(t, []string{platform, local}, p.Files)
	})

	t.Run("empty_root_goes_unreported", func(t *testing.T) {
		t.Parallel()

		// The gap. Second root exists and holds no script, so every test it was
		// meant to supply silently vanishes while the run stays green.
		p, err := NewTestingParamsFromRoots(t,
			filepath.Join(dir, "testdata"), t.TempDir())
		require.NoError(t, err)
		assert.Equal(t, []string{platform}, p.Files)
	})

	t.Run("no_script_anywhere", func(t *testing.T) {
		t.Parallel()

		_, err := NewTestingParamsFromRoots(t, t.TempDir())
		assert.ErrorContains(t, err, "no testscript file found")
	})
}
