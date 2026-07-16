/*
Run:

	git clone https://github.com/gnolang/gno && cd gno
	git checkout 03ab3eea2
	cp <this file> gno.land/pkg/sdk/vm/preprocess_gas_test.go
	go test ./gno.land/pkg/sdk/vm/ -run 'PreprocessGas' -v

Blue-team coverage for PR #5892 (meter type-check+preprocess gas at
AddPackage and Run). At 03ab3eea2 the PR's charge is asserted only by
integration txtar gas pins; no Go test asserts (a) the byte-counting rule,
(b) that the charge is wired at AddPackage, (c) that it is wired at Run, or
(d) the Validate upper bound. All four tests below PASS at 03ab3eea2 — they
are regression guards for rules the merged code already honours, not bug
reports.

Notable: TestRunPreprocessGasCharged is the only assertion anywhere in the
tree that keeper.go:1081's `chargePreprocessGas(..., "RunPreprocess")` call
exists. maketx_run.txtar pins `GAS USED: ` with no value, so deleting that
line leaves the whole suite green.
*/
package vm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

// TestChargePreprocessGasByteBase pins chargePreprocessGas' byte base: every
// .gno file counts (prod, _test AND _filetest), every non-.gno file does not.
// That base is consensus gas, and no other test constrains it: the only txtar
// carrying a _test.gno (addpkg_import_testdep_gas.txtar) does not pin the gas
// of the deploy that contains it, so narrowing the base to prod-only would
// leave every existing pin green.
func TestChargePreprocessGasByteBase(t *testing.T) {
	env := setupTestEnv()

	const (
		prodBody     = "package p\n\nfunc Hi() string { return \"hi\" }\n"
		testBody     = "package p\n\nimport \"testing\"\n\nfunc TestHi(t *testing.T) {}\n"
		filetestBody = "package main\n\nfunc main() { println(\"x\") }\n\n// Output:\n// x\n"
		// Non-.gno files: must contribute zero.
		modBody = "module = \"gno.land/p/demo/x\"\ngno = \"0.9\"\n"
		docBody = "# readme with plenty of bytes that must not be charged\n"
	)

	mpkg := &std.MemPackage{
		Name: "p",
		Path: "gno.land/p/demo/x",
		Files: []*std.MemFile{
			{Name: "gnomod.toml", Body: modBody},
			{Name: "README.md", Body: docBody},
			{Name: "lib.gno", Body: prodBody},
			{Name: "lib_test.gno", Body: testBody},
			{Name: "z_filetest.gno", Body: filetestBody},
		},
	}

	params := DefaultParams()
	params.PreprocessGasPerByte = 7 // small prime: any miscount shows up exactly

	gm := types.NewInfiniteGasMeter()
	chargePreprocessGas(env.ctx.WithGasMeter(gm), params, mpkg, "test")

	wantBytes := int64(len(prodBody) + len(testBody) + len(filetestBody))
	assert.Equal(t, wantBytes*7, gm.GasConsumed(),
		"charge must be PreprocessGasPerByte * (prod + _test + _filetest .gno bytes), excluding non-.gno files")

	// Cross-check the exclusion directly: dropping the two non-.gno files
	// must not move the number.
	mpkg2 := &std.MemPackage{Name: "p", Path: "gno.land/p/demo/x", Files: mpkg.Files[2:]}
	gm2 := types.NewInfiniteGasMeter()
	chargePreprocessGas(env.ctx.WithGasMeter(gm2), params, mpkg2, "test")
	assert.Equal(t, gm.GasConsumed(), gm2.GasConsumed(), "non-.gno files must contribute zero gas")
}

// TestHasProdGnoFileEdgeCases pins hasProdGnoFile (keeper.go:606), the
// drive-by replacement for `MPFProd.FilterMemPackage(memPkg).IsEmpty()`.
// TestVMKeeperAddPackage_NoProdFiles covers only the _test.gno-only reject;
// the _filetest.gno-only reject and the non-.gno guard (FilterGno panics on a
// non-.gno name, so dropping the HasSuffix check would panic rather than
// return an error) are unasserted.
func TestHasProdGnoFileEdgeCases(t *testing.T) {
	mp := func(files ...*std.MemFile) *std.MemPackage {
		return &std.MemPackage{Name: "p", Path: "gno.land/p/demo/x", Files: files}
	}
	var (
		prod     = &std.MemFile{Name: "lib.gno", Body: "package p\n"}
		test     = &std.MemFile{Name: "lib_test.gno", Body: "package p\n"}
		filetest = &std.MemFile{Name: "z_filetest.gno", Body: "package main\n"}
		mod      = &std.MemFile{Name: "gnomod.toml", Body: "module = \"gno.land/p/demo/x\"\n"}
		doc      = &std.MemFile{Name: "README.md", Body: "# x\n"}
	)

	for _, tc := range []struct {
		name string
		pkg  *std.MemPackage
		want bool
	}{
		{"prod only", mp(prod), true},
		{"prod + test + filetest", mp(mod, prod, test, filetest), true},
		{"test only", mp(mod, test), false},
		{"filetest only", mp(mod, filetest), false},
		{"test + filetest", mp(mod, test, filetest), false},
		// The HasSuffix guard: FilterGno panics on a non-.gno file, so these
		// must be skipped, not passed through to it.
		{"non-gno only", mp(mod, doc), false},
		{"non-gno + prod", mp(mod, doc, prod), true},
		{"empty", mp(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				assert.Equal(t, tc.want, hasProdGnoFile(tc.pkg))
			}, "hasProdGnoFile must never reach FilterGno's non-.gno panic")
		})
	}
}

// addPkgGasAtRate deploys a fixed package into a fresh env whose
// PreprocessGasPerByte is rate, and returns the gas the deploy consumed.
// Two fresh envs are byte-identical apart from the param, so the gas
// difference between two rates isolates the preprocess charge exactly.
func addPkgGasAtRate(t *testing.T, rate int64, files []*std.MemFile, pkgPath string) int64 {
	t.Helper()
	env := setupTestEnv()

	params := DefaultParams()
	params.PreprocessGasPerByte = rate
	require.NoError(t, env.vmk.SetParams(env.ctx, params))

	addr := crypto.AddressFromPreimage([]byte("addr1"))
	{
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
		acc := env.acck.NewAccountWithAddress(ctx, addr)
		env.acck.SetAccount(ctx, acc)
		env.bankk.SetCoins(ctx, addr, initialBalance)
		env.vmk.CommitGnoTransactionStore(ctx)
	}

	gm := types.NewInfiniteGasMeter()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx.WithGasMeter(gm))
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files)))
	return gm.GasConsumed()
}

// TestAddPackagePreprocessGasCharged asserts the charge is actually wired into
// AddPackage at the PreprocessGasPerByte rate: doubling the param must raise
// the deploy's gas by exactly (rate delta) * .gno source bytes.
func TestAddPackagePreprocessGasCharged(t *testing.T) {
	const pkgPath = "gno.land/r/test"
	const libBody = `package test

func Hello() string { return "hello" }
`
	testBody := "package test\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) { _ = Hello() }\n"

	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "lib.gno", Body: libBody},
		{Name: "lib_test.gno", Body: testBody},
	}
	srcBytes := int64(len(libBody) + len(testBody))

	lo := addPkgGasAtRate(t, 1_250, files, pkgPath)
	hi := addPkgGasAtRate(t, 2_500, files, pkgPath)

	assert.Equal(t, (2_500-1_250)*srcBytes, hi-lo,
		"AddPackage gas must move by exactly (rate delta) * .gno source bytes")
	// The _test.gno bytes participate: the delta above is computed over
	// prod+test bytes, so an equal check with prod-only bytes must differ.
	assert.NotEqual(t, (2_500-1_250)*int64(len(libBody)), hi-lo,
		"_test.gno bytes must be part of the charged base")
}

// TestRunPreprocessGasCharged asserts the same for MsgRun (keeper.go:1081).
// This is the only assertion in the tree that pins the Run charge:
// maketx_run.txtar matches `GAS USED: ` with no value, so removing the Run
// call site leaves every other test green.
func TestRunPreprocessGasCharged(t *testing.T) {
	const scriptBody = `package main

func main() {
	println("hello world!")
}
`
	runGasAtRate := func(rate int64) int64 {
		env := setupTestEnv()
		params := DefaultParams()
		params.PreprocessGasPerByte = rate
		require.NoError(t, env.vmk.SetParams(env.ctx, params))

		addr := crypto.AddressFromPreimage([]byte("addr1"))
		gm := types.NewInfiniteGasMeter()
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx.WithGasMeter(gm))
		acc := env.acck.NewAccountWithAddress(ctx, addr)
		env.acck.SetAccount(ctx, acc)

		files := []*std.MemFile{
			{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest("gno.land/r/test")},
			{Name: "script.gno", Body: scriptBody},
		}
		before := gm.GasConsumed()
		res, err := env.vmk.Run(ctx, NewMsgRun(addr, std.MustParseCoins(""), files))
		require.NoError(t, err)
		require.Equal(t, "hello world!\n", res)
		return gm.GasConsumed() - before
	}

	lo := runGasAtRate(1_250)
	hi := runGasAtRate(2_500)

	assert.Equal(t, (2_500-1_250)*int64(len(scriptBody)), hi-lo,
		"Run gas must move by exactly (rate delta) * .gno source bytes")
}

// TestParamsValidatePreprocessGasPerByteCap covers the upper bound at
// params.go:199. Zero and negative are covered by existing tests
// (TestGetParamsDefaultsPreprocessGasPerByte, TestGenesisToleratesLegacy...);
// the cap was not.
func TestParamsValidatePreprocessGasPerByteCap(t *testing.T) {
	const maxPreprocessGasPerByte = 100_000

	for _, tc := range []struct {
		v       int64
		wantErr bool
	}{
		{1, false},
		{maxPreprocessGasPerByte, false},    // boundary: inclusive
		{maxPreprocessGasPerByte + 1, true}, // boundary: first rejected
		{1 << 40, true},
	} {
		t.Run(fmt.Sprint(tc.v), func(t *testing.T) {
			p := DefaultParams()
			p.PreprocessGasPerByte = tc.v
			err := p.Validate()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "PreprocessGasPerByte must be <= 100000")
			} else {
				assert.NoError(t, err)
			}
		})
	}

	// A governance proposal over the cap must panic in WillSetParam rather
	// than land an unpriceable rate.
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	require.NoError(t, env.vmk.SetParams(ctx, DefaultParams()))
	assert.Panics(t, func() {
		env.vmk.WillSetParam(ctx, "p:preprocess_gas_per_byte", int64(maxPreprocessGasPerByte+1))
	})
}
