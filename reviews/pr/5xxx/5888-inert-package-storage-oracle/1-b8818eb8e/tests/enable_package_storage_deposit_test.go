/* Run: from a gno checkout:
gh pr checkout 5888 -R gnolang/gno
curl -fsSL -o gno.land/pkg/sdk/vm/zz_enable_deposit_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5888-inert-package-storage-oracle/1-b8818eb8e/tests/enable_package_storage_deposit_test.go
go test ./gno.land/pkg/sdk/vm/ -run 'TestEnablePackageChargesStorageDeposit' -count=1 -v
rm gno.land/pkg/sdk/vm/zz_enable_deposit_test.go
*/

// EnablePackage persists a package's realm objects via RunMemPackage but never
// calls processStorageDeposit, so the activation locks no refundable deposit and
// leaves rlm.Storage/rlm.Deposit at zero while the realm holds real persisted
// bytes. Normal permissionless addpkg on the same source locks a deposit.
// The test asserts the post-fix invariant (deposit charged on enable) and fails
// at b8818eb8e.

package vm

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestEnablePackageChargesStorageDeposit(t *testing.T) {
	const src = `package stateful

var G = []string{}

func Grow(cur realm, s string) { G = append(G, s) }

func init() { G = append(G, "seed") }`

	const pkgPath = "gno.land/r/test/stateful"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "stateful.gno", Body: src},
	}

	// Baseline: normal permissionless addpkg locks a deposit.
	{
		env := setupTestEnv()
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
		creator := crypto.AddressFromPreimage([]byte("creator"))
		env.acck.SetAccount(ctx, env.acck.NewAccountWithAddress(ctx, creator))
		env.bankk.SetCoins(ctx, creator, initialBalance)

		if err := env.vmk.AddPackage(ctx, NewMsgAddPackage(creator, pkgPath, files)); err != nil {
			t.Fatalf("normal addpkg: %v", err)
		}
		rlm := env.vmk.getGnoTransactionStore(ctx).GetPackageRealm(pkgPath)
		if rlm.Storage == 0 || rlm.Deposit == 0 {
			t.Fatalf("baseline broken: normal addpkg should account storage+deposit, got Storage=%d Deposit=%d", rlm.Storage, rlm.Deposit)
		}
	}

	// Inert submit + oracle enable: same source, activated on-chain.
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	approver := crypto.AddressFromPreimage([]byte("oracle"))
	submitter := crypto.AddressFromPreimage([]byte("submitter"))
	for _, addr := range []crypto.Address{approver, submitter} {
		env.acck.SetAccount(ctx, env.acck.NewAccountWithAddress(ctx, addr))
		env.bankk.SetCoins(ctx, addr, initialBalance)
	}
	params := DefaultParams()
	params.CodeSubmissionPolicy = CodeSubmissionPolicyInert
	params.PkgApprovers = []crypto.Address{approver}
	env.vmk.SetParams(ctx, params)

	if err := env.vmk.AddPackage(ctx, NewMsgAddPackage(submitter, pkgPath, files)); err != nil {
		t.Fatalf("inert addpkg: %v", err)
	}
	if err := env.vmk.EnablePackage(ctx, MsgEnablePackage{Approver: approver, PkgPath: pkgPath}); err != nil {
		t.Fatalf("enable: %v", err)
	}

	rlm := env.vmk.getGnoTransactionStore(ctx).GetPackageRealm(pkgPath)
	t.Logf("after enable: Storage=%d Deposit=%d", rlm.Storage, rlm.Deposit)
	if rlm.Storage == 0 {
		t.Errorf("activated realm holds persisted objects but records Storage=0: accounting baseline corrupted")
	}
	if rlm.Deposit == 0 {
		t.Errorf("activation charged no storage deposit (Deposit=0): free permanent storage for the enabled package")
	}
}
