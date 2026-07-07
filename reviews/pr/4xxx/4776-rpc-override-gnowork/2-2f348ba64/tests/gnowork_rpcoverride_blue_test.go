/* Run: from a gno checkout:
gh pr checkout 4776 -R gnolang/gno
curl -fsSL -o gnovm/pkg/packages/gnowork_rpcoverride_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4776-rpc-override-gnowork/2-2f348ba64/tests/gnowork_rpcoverride_blue_test.go
go test -v -run 'Review_|GoTomlBehavior' ./gnovm/pkg/packages/
rm gnovm/pkg/packages/gnowork_rpcoverride_test.go
*/
// White-box (package packages) coverage the PR's own tests miss: the real Load
// entry, a gnodev-style non-RPC fetcher through Load, gnowork-vs-flag precedence,
// and go-toml v1.9.5 silent-accept. All pass at 2f348ba64; the typo cases assert
// that a mistyped key yields no override AND no error (the silent-fallback footgun).
package packages

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
)

// plainFetcher implements only PackageFetcher (no OverrideDomainsRPCs), like
// gnodev's disabledFetcher / domainFetcher.
type plainFetcher struct{}

func (plainFetcher) FetchPackage(string) ([]*std.MemFile, error) { return nil, nil }

func writeWorkspace(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gnowork.toml"), []byte(body), 0o600))
	return dir
}

// A workspace rpc override plus a fetcher that can't honor it aborts Load, via
// the real findLoaderContext -> ReadGnowork path. This is gnodev's default fetcher.
func TestReview_Load_OverrideWithPlainFetcher_Errors(t *testing.T) {
	dir := writeWorkspace(t, "[domains.\"gno.land\"]\nrpc = \"http://localhost:26657\"\n")
	t.Chdir(dir)

	_, err := Load(LoadConfig{Fetcher: plainFetcher{}, AllowEmpty: true}, "./...")
	require.Error(t, err)
	require.ErrorContains(t, err, "does not support")
	t.Logf("Load error (as expected): %v", err)
}

// An empty rpc value is skipped through the real Load entry: a plain fetcher must
// NOT trip the "does not support" abort.
func TestReview_Load_EmptyRPCSkipped_NoError(t *testing.T) {
	dir := writeWorkspace(t, "[domains.\"gno.land\"]\nrpc = \"\"\n")
	t.Chdir(dir)

	_, err := Load(LoadConfig{Fetcher: plainFetcher{}, AllowEmpty: true}, "./...")
	if err != nil {
		require.NotContains(t, err.Error(), "does not support")
	}
	t.Logf("Load error (must not be override-abort): %v", err)
}

// The gnowork.toml override merges on top of a fetcher already seeded with
// --remote-overrides for the same domain, so gnowork silently wins. Nothing
// documents which source has priority.
func TestReview_Precedence_GnoworkOverridesFlag(t *testing.T) {
	dir := writeWorkspace(t, "[domains.\"gno.land\"]\nrpc = \"http://from-gnowork:26657\"\n")
	t.Chdir(dir)

	ctx, err := findLoaderContextFor(dir)
	require.NoError(t, err)
	require.NotNil(t, ctx.Gnowork)
	require.Equal(t, map[string]string{"gno.land": "http://from-gnowork:26657"}, ctx.Gnowork.rpcOverrides())
}

// go-toml v1.9.5 does no strict decoding and is only exact-case for the field.
func TestReview_GoTomlBehavior(t *testing.T) {
	t.Run("unknown top-level key accepted", func(t *testing.T) {
		gw, err := ParseGnowork([]byte("foo = \"bar\"\n[domains.\"gno.land\"]\nrpc = \"http://x\"\n"))
		require.NoError(t, err)
		require.Equal(t, map[string]string{"gno.land": "http://x"}, gw.rpcOverrides())
	})

	t.Run("typo'd rpc key inside domain silently dropped", func(t *testing.T) {
		gw, err := ParseGnowork([]byte("[domains.\"gno.land\"]\nrpcc = \"http://x\"\n"))
		require.NoError(t, err)
		require.Empty(t, gw.rpcOverrides()) // NO override, NO error
	})

	t.Run("only exact rpc/RPC populate the field; other casings dropped", func(t *testing.T) {
		want := map[string]bool{"rpc": true, "RPC": true, "Rpc": false, "rPc": false, "rpC": false}
		for key, expectMatch := range want {
			gw, err := ParseGnowork([]byte("[domains.\"gno.land\"]\n" + key + " = \"http://x\"\n"))
			require.NoError(t, err)
			require.Equal(t, expectMatch, len(gw.rpcOverrides()) == 1, "key=%q", key)
		}
	})

	t.Run("empty file yields empty (existing workspaces unaffected)", func(t *testing.T) {
		gw, err := ParseGnowork([]byte(""))
		require.NoError(t, err)
		require.Empty(t, gw.rpcOverrides())
	})

	t.Run("raw go-toml unknown-key sanity", func(t *testing.T) {
		var g Gnowork
		require.NoError(t, toml.Unmarshal([]byte("totally_unknown = 42\n"), &g))
		require.Nil(t, g.Domains)
	})
}
