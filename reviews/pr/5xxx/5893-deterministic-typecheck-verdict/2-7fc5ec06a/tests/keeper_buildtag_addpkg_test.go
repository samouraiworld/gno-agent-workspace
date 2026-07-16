/* Run:

	gh pr checkout 5893 -R gnolang/gno && git checkout 7fc5ec06a
	cp <this-file> gno.land/pkg/sdk/vm/zz_buildtag_addpkg_test.go
	go test -count=1 -run 'TestVMKeeperAddPackage_BuildTagCannotRaisePin' -v ./gno.land/pkg/sdk/vm/
	rm gno.land/pkg/sdk/vm/zz_buildtag_addpkg_test.go

Expected at 7fc5ec06a: RED (both subtests). Proves the //go:build pin bypass is
reachable from the real consensus entry point (MsgAddPackage ->
VMKeeper.AddPackage -> gno.TypeCheckMemPackage at keeper.go:702), not just from
the gnovm unit API.

Expected after a fix that clears ast.File.GoVersion on every parsed .gno file
(e.g. in GoParseMemPackage, alongside the existing prepareGoGno0p9 AST pass):
GREEN.
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

// A submitted package's accept/reject verdict must depend only on the package,
// never on the Go toolchain the validator binary happens to be built with. The
// pinned types.Config.GoVersion ("go1.18", gotypecheck.go) establishes that on
// the Config axis. It does not hold on the per-file axis: go/types lets a
// `//go:build go1.N` line upgrade a file's language version above the Config
// pin (go/types.(*Checker).initFiles). File bodies arrive attacker-supplied
// over the wire, so the gate is settable by the submitter.
func TestVMKeeperAddPackage_BuildTagCannotRaisePin(t *testing.T) {
	// NOTE: pkgPath must stay "gno.land/r/test" — setupTestEnv only permits
	// that namespace, and any other path fails earlier with
	// InvalidPkgPathError, which would make every assertion below vacuous.
	addPkg := func(t *testing.T, body string) error {
		t.Helper()
		env := setupTestEnv()
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
		addr := crypto.AddressFromPreimage([]byte("addr1"))
		acc := env.acck.NewAccountWithAddress(ctx, addr)
		env.acck.SetAccount(ctx, acc)
		env.bankk.SetCoins(ctx, addr, initialBalance)

		const pkgPath = "gno.land/r/test"
		files := []*std.MemFile{
			{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
			{Name: "test.gno", Body: body},
		}
		return env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files))
	}

	// Preconditions: the harness admits a valid package, and the pin rejects a
	// go1.22 construct with the TypeCheckError sentinel. If either fails, the
	// assertions below prove nothing.
	require.NoError(t, addPkg(t, "package test\nfunc F() int { return 1 }\n"),
		"precondition: a plain valid package must be accepted")
	err := addPkg(t, "package test\nfunc F() { for range 10 {} }\n")
	require.Error(t, err, "precondition: the pin must reject range-over-int on the AddPackage path")
	require.ErrorIs(t, err, TypeCheckError{},
		"precondition: rejection must come from the type-check gate")

	t.Run("build tag must not raise the pinned version", func(t *testing.T) {
		// Same package as the precondition, plus one comment line. It must be
		// rejected by the *type-check gate*, exactly as the untagged form is.
		//
		// At 7fc5ec06a it is not: the tag raises the file to go1.22, the gate
		// passes it, and the package falls through to the GnoVM preprocessor,
		// which rejects it with an unrelated error ("range iteration requires
		// map, string, array, slice, or pointer to array; got BigintKind").
		// The tx still fails, so this alone is not an escape — but the pin is
		// not doing the job it is documented to do, and the next subtest shows
		// what that costs.
		err := addPkg(t, "//go:build go1.22\n\npackage test\nfunc F() { for range 10 {} }\n")
		require.Error(t, err)
		assert.ErrorIs(t, err, TypeCheckError{},
			"a //go:build line must not let a go1.22 construct past the pinned type-check gate")
	})

	t.Run("verdict must not depend on the building toolchain", func(t *testing.T) {
		// go/types rejects a file version newer than the toolchain that built
		// the binary: "file requires newer Go version goX (application built
		// with goY)", where Y is the *builder's* Go version. The body here is
		// plain, valid go1.18 code; only the tag decides the verdict.
		//
		// Measured on a go1.26-built binary: //go:build go1.26 -> ACCEPTED,
		// //go:build go1.27 -> REJECTED. go.mod declares `go 1.25.9` with no
		// toolchain directive, so a go1.25-built validator is in scope and
		// would reject `//go:build go1.26` while a go1.26-built one accepts it.
		// Two honest validators, opposite verdicts, same bytes — the fork this
		// PR closes on the Config axis, still open on the per-file axis.
		const body = "//go:build go1.99\n\npackage test\nfunc F() int { return 1 }\n"
		assert.NoError(t, addPkg(t, body),
			"verdict must not be a function of the building toolchain (this binary: %s)",
			runtime.Version())
	})
}
