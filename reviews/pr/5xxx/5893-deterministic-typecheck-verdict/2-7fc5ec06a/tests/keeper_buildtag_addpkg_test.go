/* Run: from a gno checkout, with two Go toolchains available:

gh pr checkout 5893 -R gnolang/gno && git checkout 7fc5ec06a
curl -fsSL -o gno.land/pkg/sdk/vm/zz_buildtag_addpkg_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5893-deterministic-typecheck-verdict/2-7fc5ec06a/tests/keeper_buildtag_addpkg_test.go
GOTOOLCHAIN=go1.26.5 go test -count=1 -v -run 'TestVMKeeperAddPackage_' ./gno.land/pkg/sdk/vm/
GOTOOLCHAIN=go1.25.9 go test -count=1 -v -run 'TestVMKeeperAddPackage_' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/zz_buildtag_addpkg_test.go

go/types resolves a file's `//go:build go1.N` line against the version of the Go
toolchain that compiled the binary, so the accept/reject verdict for a submitted
package becomes a function of the build rather than the package. At 7fc5ec06a
the go1.26.5 build accepts and deploys the tagged package while the go1.25.9
build rejects it with TypeCheckError, and go.mod's `go 1.25.9` admits both.
When the verdict no longer depends on the build, both runs agree and go green.
*/

package vm

import (
	"runtime"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addPkgForVerdict(t *testing.T, pkgPath, body string) error {
	t.Helper()
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "test.gno", Body: body},
	}
	return env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files))
}

// The accept/reject verdict for a submitted package must depend only on the
// package. The body here is valid Gno at every language version and the only
// unusual token is a build constraint, which carries no meaning in Gno. Two
// honest validators on this same commit, built with two Go releases go.mod
// admits, must still agree.
func TestVMKeeperAddPackage_VerdictIsBuildIndependent(t *testing.T) {
	// Precondition: the pin is live, so a failure below cannot be misread as a
	// missing pin.
	err := addPkgForVerdict(t, "gno.land/r/plain",
		"package plain\nfunc F() { for range 10 {} }\n")
	require.Error(t, err, "precondition: the pin must reject range-over-int on the AddPackage path")

	body := "//go:build go1.26\n\npackage tagged\n\nfunc Add(a, b int) int { return a + b }\n"
	err = addPkgForVerdict(t, "gno.land/r/tagged", body)

	assert.NoError(t, err,
		"the verdict must not depend on the building toolchain (this binary: %s); "+
			"a validator built with an older Go rejects this same package, so the two fork",
		runtime.Version())
}

// The pin must gate the package rather than merely be present: a construct the
// GnoVM cannot run must be rejected at the type-check gate, carrying the
// TypeCheckError sentinel, not slip past it to the preprocessor.
func TestVMKeeperAddPackage_BuildTagCannotBypassGate(t *testing.T) {
	err := addPkgForVerdict(t, "gno.land/r/tagrange",
		"//go:build go1.22\n\npackage tagrange\nfunc F() { for range 10 {} }\n")
	require.Error(t, err, "range-over-int must be rejected however the file is tagged")

	// At 7fc5ec06a this is the GnoVM preprocessor's "got BigintKind" error rather
	// than the gate's sentinel: the build tag raised the pinned version, so the
	// construct reached code the pin was supposed to keep it away from.
	assert.ErrorIs(t, err, TypeCheckError{},
		"a //go:build line must not raise the pinned GoVersion: range-over-int "+
			"must still be rejected by the type-check gate, not downstream")
}
