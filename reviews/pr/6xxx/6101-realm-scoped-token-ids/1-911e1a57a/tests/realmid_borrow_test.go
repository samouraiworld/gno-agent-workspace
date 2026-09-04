// Probes whether the realm NewRealmID() names is the same realm an
// IsCurrent() realm value names. A non-crossing call into a /r/ package
// takes borrow rule #1 (machine.go), which moves m.Realm to the callee's
// realm without opening a crossing frame, so the caller's cur stays
// IsCurrent. Fails at 911e1a57a if the two agree.
//
/* Run: from a gno checkout:
gh pr checkout 6101 -R gnolang/gno && git checkout 911e1a57a
curl -fsSL -o gno.land/pkg/sdk/vm/realmid_borrow_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6101-realm-scoped-token-ids/1-911e1a57a/tests/realmid_borrow_test.go
go test -v -run 'TestRealmIDBorrowSplitsIdentity' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/realmid_borrow_test.go
*/
package vm

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func TestRealmIDBorrowSplitsIdentity(t *testing.T) {
	env := setupTestEnv()
	addr := crypto.AddressFromPreimage([]byte("realm-id-borrow"))
	const (
		helperPath = "gno.land/r/test/bhelper"
		callerPath = "gno.land/r/test/acaller"
	)

	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	// bhelper.Build is NOT a crossing function: it takes a realm value the
	// caller threads in, exactly the shape grc20.NewToken's `rlm realm`
	// parameter exists for.
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, helperPath, []*std.MemFile{
		{Name: "bhelper.gno", Body: `package bhelper

import "chain/runtime"

func Build(tag string, rlm realm) (string, string, bool) {
	return runtime.NewRealmID(), rlm.PkgPath(), rlm.IsCurrent()
}`},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(helperPath)},
	})))
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, callerPath, []*std.MemFile{
		{Name: "acaller.gno", Body: `package acaller

import (
	"chain/runtime"
	"gno.land/r/test/bhelper"
)

// Mint threads acaller's own cur into bhelper.Build, the shape
// grc20.NewToken's trailing "rlm realm" parameter exists for.
func Mint(cur realm) string {
	id, origin, current := bhelper.Build("t", cur)
	out := "id=" + id + " origin=" + origin + " direct=" + runtime.NewRealmID()
	if current {
		return out + " current=true"
	}
	return out + " current=false"
}`},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(callerPath)},
	})))
	env.vmk.CommitGnoTransactionStore(ctx)

	callCtx := env.vmk.MakeGnoTransactionStore(env.ctx)
	res, err := env.vmk.Call(callCtx, NewMsgCall(addr, nil, callerPath, "Mint", nil))
	require.NoError(t, err)
	t.Log(res)

	// IS:     the id names bhelper while the IsCurrent realm names acaller.
	require.Contains(t, res, "id=gno.land/r/test/bhelper:")
	require.Contains(t, res, "origin=gno.land/r/test/acaller")
	require.Contains(t, res, "current=true")
	// SHOULD: both name acaller, the realm whose cur was verified.
	// require.Contains(t, res, "id=gno.land/r/test/acaller:")
}
