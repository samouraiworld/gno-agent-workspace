/* Run: from a gno checkout, with the fix applied to contribs/gnodev/app.go
   (the /reset handler and Ctrl+R sharing a resetState helper that clears
   pathManager):
gh pr checkout 5889 -R gnolang/gno
# append the helper + test below into contribs/gnodev/app_test.go, then:
GNOROOT="$PWD" go test -run 'TestGnodev_Reset_RebrowseRedeploys' -v ./contribs/gnodev/
*/

// Drop resetState's pathManager.Reset() to reproduce the /reset regression:
// the re-browse then hits the pathManager dedup and the assertion at the end
// fails (package never redeploys). With the fix present the test passes.
// Belongs in package main (contribs/gnodev), same package as app_test.go.

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// browseLazyPath mirrors what the proxy path callback does on a fresh browse:
// resolve the path, dedup it against pathManager, re-seed the node paths from
// pathManager, and reload. It returns whether pathManager treated the path as
// new (exist == false means it was re-registered and the node was reloaded).
func browseLazyPath(t *testing.T, ctx context.Context, app *App, path string) (registered bool) {
	t.Helper()
	_, err := app.loader.Resolve(path)
	require.NoError(t, err, "path must resolve for a lazy browse")

	exist := app.pathManager.Save(path)
	if exist {
		// pathManager still holds a stale entry: the proxy short-circuits and
		// never redeploys the package. This is the /reset regression.
		return false
	}

	app.devNode.SetPackagePaths(app.paths...)
	app.devNode.AddPackagePaths(app.pathManager.List()...)
	require.NoError(t, app.devNode.Reload(ctx))
	return true
}

// TestGnodev_Reset_RebrowseRedeploys guards the /reset path: resetState must
// clear pathManager the same way Ctrl+R does. Without the pathManager reset, a
// re-browse of a package loaded before the reset hits the dedup and never
// redeploys, so the package stays 404 for the rest of the session.
func TestGnodev_Reset_RebrowseRedeploys(t *testing.T) {
	if os.Getenv("GNOROOT") == "" {
		t.Skip("needs GNOROOT for the examples root")
	}

	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "gnowork.toml"), []byte(""), 0o644))
	writeWorkspacePkg(t, filepath.Join(workspace, "w"), "gno.land/p/ws/only", "package only\n")
	t.Chdir(workspace)

	// An examples package: browsable lazily through the proxy, and absent from
	// the initial genesis set because examples stay lazy under Reload (only the
	// workspace and extra roots load eagerly). This is the shape r/sys/users has
	// in the live repro.
	const browsed = "gno.land/p/demo/nestedpkg"

	cfg := defaultLocalAppConfig
	cfg.root = os.Getenv("GNOROOT")
	cfg.home = filepath.Join(t.TempDir(), "nokeybase")
	cfg.nodeRPCListenerAddr = "127.0.0.1:0"
	cfg.noWatch = true

	ctx := context.Background()
	app := NewApp(discardLogger(), &cfg, commands.NewTestIO())
	require.NoError(t, app.Setup(ctx))
	t.Cleanup(app.Close)

	// Not in the initial set.
	require.NotContains(t, importPaths(app.devNode.ListPkgs()), browsed,
		"examples package must not be in the initial genesis set")

	// Browse it: registers with pathManager and deploys.
	require.True(t, browseLazyPath(t, ctx, app, browsed))
	require.True(t, app.devNode.HasPackageLoaded(browsed),
		"browsed package must deploy on first browse")

	// Reset via the shared reset sequence (what /reset and Ctrl+R both call).
	require.NoError(t, app.resetState(ctx))

	// The reset returned the node to its initial set.
	assert.False(t, app.devNode.HasPackageLoaded(browsed),
		"reset must drop the browsed package from the node")

	// The fix: pathManager was cleared, so a re-browse re-registers and
	// redeploys instead of being deduped away.
	require.True(t, browseLazyPath(t, ctx, app, browsed),
		"re-browse after reset must re-register the path (pathManager was left dirty)")
	assert.True(t, app.devNode.HasPackageLoaded(browsed),
		"re-browse after reset must redeploy the package")
}
