package vm

// A node upgraded in place (halt_height, then the next binary on the same
// data directory) keeps the store its previous binary wrote at genesis.
// Stdlibs are loaded only in InitChainer, so that store has no
// crypto/bls12381, while VMKeeper.Initialize, which every boot runs, type
// checks every path in stdlibs.InitOrder(), now including crypto/bls12381.
//
// Simulated here by removing the package's mempackage key from a store loaded
// with every stdlib (IterMemPackage skips a nil mempackage, so the preprocess
// pass sees the same set an older store holds), then booting a fresh keeper on
// it, as the upgraded binary does in gnoland.NewAppWithOptions.
//
// Red means the boot panics. The control boots the same way with the key kept.
//
// Repro from a plain clone of gnolang/gno (gnolang/gno#6375 at 0ee53fefc):
//   git fetch origin pull/6375/head:pr6375 && git checkout 0ee53fefc
//   cp <this file> gno.land/pkg/sdk/vm/topic4_upgrade_new_stdlib_test.go
//   go test ./gno.land/pkg/sdk/vm -run 'TestTopic4UpgradeBoot' -count=1 -v -timeout 2m

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestTopic4UpgradeBootWithoutNewStdlib(t *testing.T) {
	env := setupTestEnv()
	ms := env.ctx.MultiStore()
	ms.GetStore(env.vmk.iavlKey).Delete(nil, []byte("pkg:_/crypto/bls12381"))

	vmk2 := NewVMKeeper(env.vmk.baseKey, env.vmk.iavlKey, env.acck, env.bankk, env.prmk)
	var rec any
	func() {
		defer func() { rec = recover() }()
		vmk2.Initialize(log.NewNoopLogger(), ms.MultiCacheWrap())
	}()
	t.Logf("recovered: %v", rec)
	require.Nil(t, rec, "booting on a store that predates crypto/bls12381 panics")
}

func TestTopic4UpgradeBootControl(t *testing.T) {
	env := setupTestEnv()
	ms := env.ctx.MultiStore()

	vmk2 := NewVMKeeper(env.vmk.baseKey, env.vmk.iavlKey, env.acck, env.bankk, env.prmk)
	require.NotPanics(t, func() { vmk2.Initialize(log.NewNoopLogger(), ms.MultiCacheWrap()) })
}
