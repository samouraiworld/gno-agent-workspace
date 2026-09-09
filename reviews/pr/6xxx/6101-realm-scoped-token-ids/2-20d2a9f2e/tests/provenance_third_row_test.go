// The third cell of TestVMKeeperNewRealmIDProvenance's case space, the shape
// grc20.NewToken's trailing "rlm realm" parameter exists for: a caller threads
// its own cur into a plain function declared in another /r/ package. Borrow
// rule #1 (machine.go) moves m.Realm onto that package while the threaded cur
// stays IsCurrent. Fails at 20d2a9f2e if the ID names the realm the callee
// authenticated.
//
//	NewFromHelper   a /p/ helper, plain call      keeper_test.go covers it
//	NewFromIssuer   an /r/ issuer, cross() call   keeper_test.go covers it
//	NewFromWrapper  an /r/ helper, plain call     nothing covers it
//
/* Run: from a gno checkout:
gh pr checkout 6101 -R gnolang/gno && git checkout 20d2a9f2e
curl -fsSL -o gno.land/pkg/sdk/vm/provenance_third_row_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6101-realm-scoped-token-ids/2-20d2a9f2e/tests/provenance_third_row_test.go
go test -v -run 'TestVMKeeperNewRealmIDProvenanceThirdRow' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/provenance_third_row_test.go
*/
package vm

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func TestVMKeeperNewRealmIDProvenanceThirdRow(t *testing.T) {
	env := setupTestEnv()
	addr := crypto.AddressFromPreimage([]byte("realm-id-third-row"))
	const (
		wrapperPath = "gno.land/r/test/cwrapper"
		callerPath  = "gno.land/r/test/dcaller"
	)

	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	require.NoError(t, env.bankk.SetCoins(ctx, addr, initialBalance))

	// cwrapper.New is NOT a crossing function. It takes the realm value the
	// caller threads in, the same signature shape grc20.NewToken has.
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, wrapperPath, []*std.MemFile{
		{Name: "cwrapper.gno", Body: `package cwrapper

import "chain/runtime"

func New(tag string, rlm realm) (string, string, bool) {
	return runtime.NewRealmID(), rlm.PkgPath(), rlm.IsCurrent()
}`},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(wrapperPath)},
	})))
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, callerPath, []*std.MemFile{
		{Name: "dcaller.gno", Body: `package dcaller

import "gno.land/r/test/cwrapper"

func NewFromWrapper(cur realm) string {
	id, verified, current := cwrapper.New("t", cur)
	if !current {
		return "id=" + id + " verified=" + verified + " current=false"
	}
	return "id=" + id + " verified=" + verified + " current=true"
}`},
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(callerPath)},
	})))
	env.vmk.CommitGnoTransactionStore(ctx)

	callCtx := env.vmk.MakeGnoTransactionStore(env.ctx)
	res, err := env.vmk.Call(callCtx, NewMsgCall(addr, nil, callerPath, "NewFromWrapper", nil))
	require.NoError(t, err)
	t.Log(res)

	// The realm value the callee authenticated is the caller's, and it is current.
	require.Contains(t, res, "verified="+callerPath)
	require.Contains(t, res, "current=true")

	// IS: the ID names the wrapper, not the realm whose cur passed IsCurrent.
	require.Contains(t, res, "id="+wrapperPath+":")
	// SHOULD: the ID names the realm the callee authenticated.
	// require.Contains(t, res, "id="+callerPath+":")
}
