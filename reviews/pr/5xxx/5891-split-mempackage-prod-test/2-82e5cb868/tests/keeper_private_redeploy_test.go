/*
Run:

	gh pr checkout 5891 -R gnolang/gno && git checkout 82e5cb868
	cp keeper_private_redeploy_test.go gno.land/pkg/sdk/vm/
	go test ./gno.land/pkg/sdk/vm/ -run TestVMKeeperAddPackage_PrivateRedeployClearsStaleTestFile -v

Pins the keeper half of the two-call AddMemPackage/DeleteMemPackage store
contract. AddMemPackage writes an MP*All package as two conditional key writes
(pkg:<path> + pkg:<path>#allbutprod), which is NOT a full replace across both
keys, so VMKeeper.AddPackage must DeleteMemPackage first on the private-redeploy
path (keeper.go:635-643). Nothing asserted that today: the existing
TestVMKeeperAddPackage_UpdatePrivatePackage redeploys the same file set, and the
store-level TestDeleteMemPackageClearsStaleBlobsOnReAdd only mirrors the
delete-then-add by hand. Dropping the keeper's DeleteMemPackage call leaves every
existing test green while qfile/GetMemPackageAll keep serving a deleted
_test.gno.
*/
package vm

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVMKeeperAddPackage_PrivateRedeployClearsStaleTestFile(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)

	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	const pkgPath = "gno.land/r/test"
	const gnomod = `module = "gno.land/r/test"
gno = "0.9"
private = true`
	const prod = `package test

func Echo(cur realm) string {
	return "hello world"
}`

	// Deploy a private package carrying a _test.gno: the prod files land in
	// pkg:<path>, the test file in the pkg:<path>#allbutprod sibling.
	err := env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, []*std.MemFile{
		{Name: "gnomod.toml", Body: gnomod},
		{Name: "test.gno", Body: prod},
		{Name: "test_test.gno", Body: `package test

func helperOnlyInFirstDeploy() string { return "stale" }`},
	}))
	require.NoError(t, err)
	require.NotNil(t, env.vmk.getGnoTransactionStore(ctx).GetMemFile(pkgPath, "test_test.gno"),
		"precondition: the test file is stored in the #allbutprod sibling")

	// Redeploy the same private package WITHOUT the test file. The re-add
	// writes only the prod blob (allButProd is empty, so no sibling write),
	// so the keeper must have cleared both keys first.
	err = env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, []*std.MemFile{
		{Name: "gnomod.toml", Body: gnomod},
		{Name: "test.gno", Body: prod},
	}))
	require.NoError(t, err)

	store := env.vmk.getGnoTransactionStore(ctx)
	assert.Nil(t, store.GetMemFile(pkgPath, "test_test.gno"),
		"a _test.gno dropped by a private redeploy must not survive in the #allbutprod sibling")

	all := store.GetMemPackageAll(pkgPath)
	require.NotNil(t, all)
	names := make([]string, 0, len(all.Files))
	for _, mf := range all.Files {
		names = append(names, mf.Name)
	}
	assert.Equal(t, []string{"gnomod.toml", "test.gno"}, names,
		"GetMemPackageAll must reflect the redeployed file set exactly")
}
