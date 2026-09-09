// Probes whose persistent counter runtime.NewRealmID() spends. Calling a
// /p/-declared method on a foreign realm's stored object takes borrow rule #2
// (machine.go), which moves m.Realm to the object's owning realm, so the
// native mints an ID naming that realm and advances its saved Realm.Time.
// Fails at 20d2a9f2e if the ID names the calling realm.
//
/* Run: from a gno checkout:
gh pr checkout 6101 -R gnolang/gno && git checkout 20d2a9f2e
curl -fsSL -o gno.land/pkg/sdk/vm/foreign_realm_counter_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6101-realm-scoped-token-ids/2-20d2a9f2e/tests/foreign_realm_counter_test.go
go test -v -run 'TestRealmIDSpendsForeignRealmCounter' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/foreign_realm_counter_test.go
*/
package vm

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func TestRealmIDSpendsForeignRealmCounter(t *testing.T) {
	env := setupTestEnv()
	addr := crypto.AddressFromPreimage([]byte("realm-id-foreign"))
	const (
		boxPath    = "gno.land/p/test/pbox"
		victimPath = "gno.land/r/test/victim"
		attackPath = "gno.land/r/test/attack"
	)

	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	// Ping is declared in a /p/ package, so borrow rule #1 does not fire and
	// rule #2 borrows to the receiver's owning realm. chain/params.SetString
	// reads execctx.CurrentRealm(m) from the same frame, which is the pair
	// this test compares.
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, boxPath, []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(boxPath)},
		{Name: "pbox.gno", Body: `package pbox

import (
	"chain/params"
	"chain/runtime"
	"chain/runtime/unsafe"
)

type Box struct{ N int }

func (b *Box) Ping() string {
	params.SetString("probe", "written")
	return runtime.NewRealmID() + " identity=" + unsafe.CurrentRealm().PkgPath()
}`},
	})))
	// victim exports a stored object and never calls the native itself.
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, victimPath, []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(victimPath)},
		{Name: "victim.gno", Body: `package victim

import "gno.land/p/test/pbox"

var B = &pbox.Box{N: 1}`},
	})))
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, attackPath, []*std.MemFile{
		{Name: "attack.gno", Body: `package attack

import (
	"chain/runtime"
	"gno.land/r/test/victim"
)

func Spend(cur realm) string {
	return "viaForeignReceiver=" + victim.B.Ping() + " own=" + runtime.NewRealmID()
}`},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(attackPath)},
	})))
	env.vmk.CommitGnoTransactionStore(ctx)

	before := env.vmk.getGnoTransactionStore(env.vmk.MakeGnoTransactionStore(env.ctx)).GetPackageRealm(victimPath).Time

	callCtx := env.vmk.MakeGnoTransactionStore(env.ctx)
	res, err := env.vmk.Call(callCtx, NewMsgCall(addr, nil, attackPath, "Spend", nil))
	require.NoError(t, err)
	env.vmk.CommitGnoTransactionStore(callCtx)

	// Read the counter back through a fresh transaction store, so the value
	// is the persisted one rather than the cached *Realm the call mutated.
	after := env.vmk.getGnoTransactionStore(env.vmk.MakeGnoTransactionStore(env.ctx)).GetPackageRealm(victimPath).Time
	t.Log(res)
	t.Logf("victimTimeBefore=%d victimTimeAfter=%d", before, after)

	var underVictim, underAttack string
	okVictim := env.prmk.GetString(env.ctx, "vm:"+victimPath+":probe", &underVictim)
	okAttack := env.prmk.GetString(env.ctx, "vm:"+attackPath+":probe", &underAttack)
	t.Logf("params under victim=%q(%v) under attack=%q(%v)", underVictim, okVictim, underAttack, okAttack)

	// IS:     the ID names victim, while the executing identity is attack.
	require.Contains(t, res, "viaForeignReceiver="+victimPath+":")
	require.Contains(t, res, "identity="+attackPath)
	require.Equal(t, before+1, after)
	// The param written from the same frame lands under attack, not victim.
	require.False(t, okVictim)
	require.Equal(t, "written", underAttack)
	// SHOULD: the ID names the realm chain/params agrees is executing.
	// require.Contains(t, res, "viaForeignReceiver="+attackPath+":")
	// require.Equal(t, before, after)
}
