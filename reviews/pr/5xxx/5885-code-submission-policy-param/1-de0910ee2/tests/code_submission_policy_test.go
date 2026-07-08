/* Run: from a gno checkout:
gh pr checkout 5885 -R gnolang/gno && git checkout de0910ee2
curl -fsSL -o gno.land/pkg/gnoland/code_submission_policy_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5885-code-submission-policy-param/1-de0910ee2/tests/code_submission_policy_test.go
go test -v -run 'TestCSP' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/code_submission_policy_test.go
*/
// The PR ships no test for checkCodeSubmissionPolicy or the new Params.Validate
// branches. checkCodeSubmissionPolicy only reads params and each msg's signers,
// so a bare std.Tx (no signatures) drives it directly.
// All cases pass at de0910ee2, proving the enforcement and validation behave.

package gnoland

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/sdk/auth"
	"github.com/gnolang/gno/tm2/pkg/sdk/bank"
	"github.com/gnolang/gno/tm2/pkg/sdk/params"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	"github.com/gnolang/gno/tm2/pkg/store/iavl"
)

func setupCSPEnv(t *testing.T) (sdk.Context, *vm.VMKeeper) {
	t.Helper()
	db := memdb.NewMemDB()
	mainKey := store.NewStoreKey("main")
	baseKey := store.NewStoreKey("base")
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(mainKey, iavl.StoreConstructor, db)
	ms.MountStoreWithDB(baseKey, dbadapter.StoreConstructor, db)
	ms.LoadLatestVersion()

	prmk := params.NewParamsKeeper(mainKey)
	acck := auth.NewAccountKeeper(mainKey, prmk.ForModule(auth.ModuleName), ProtoGnoAccount, ProtoGnoSessionAccount)
	bankk := bank.NewBankKeeper(acck, prmk.ForModule(bank.ModuleName))
	vmk := vm.NewVMKeeper(baseKey, mainKey, acck, bankk, prmk)
	prmk.Register(auth.ModuleName, acck)
	prmk.Register(bank.ModuleName, bankk)
	prmk.Register(vm.ModuleName, vmk)

	ctx := sdk.NewContext(sdk.RunTxModeDeliver, ms, &bft.Header{Height: 1, ChainID: "test-chain-id"}, log.NewNoopLogger())
	ctx = ctx.WithConsensusParams(&abci.ConsensusParams{Block: &abci.BlockParams{MaxGas: 10_000_000}})
	return ctx, vmk
}

func addPkgTx(creator crypto.Address) std.Tx {
	return std.Tx{Msgs: []std.Msg{vm.MsgAddPackage{Creator: creator}}}
}

func runTx(caller crypto.Address) std.Tx {
	return std.Tx{Msgs: []std.Msg{vm.MsgRun{Caller: caller}}}
}

func TestCSP_PermissionlessAllowsEveryone(t *testing.T) {
	ctx, vmk := setupCSPEnv(t)
	p := vm.DefaultParams() // policy = permissionless
	require.NoError(t, vmk.SetParams(ctx, p))

	rando := crypto.AddressFromPreimage([]byte("rando"))
	_, abort := checkCodeSubmissionPolicy(ctx, addPkgTx(rando), vmk)
	assert.False(t, abort, "permissionless must allow any add_package")
	_, abort = checkCodeSubmissionPolicy(ctx, runTx(rando), vmk)
	assert.False(t, abort, "permissionless must allow any run")
}

func TestCSP_EmptyPolicyTreatedPermissionless(t *testing.T) {
	ctx, vmk := setupCSPEnv(t)
	// Simulate an existing chain: never wrote the new fields.
	p := vm.DefaultParams()
	p.CodeSubmissionPolicy = "" // zero value
	require.NoError(t, vmk.SetParams(ctx, p))

	rando := crypto.AddressFromPreimage([]byte("rando"))
	_, abort := checkCodeSubmissionPolicy(ctx, addPkgTx(rando), vmk)
	assert.False(t, abort, "empty policy must behave as permissionless")
}

func TestCSP_PermissionedRejectsNonAllowlisted(t *testing.T) {
	ctx, vmk := setupCSPEnv(t)
	allowed := crypto.AddressFromPreimage([]byte("allowed"))
	p := vm.DefaultParams()
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{allowed}
	require.NoError(t, vmk.SetParams(ctx, p))

	// Allowlisted signer: passes.
	_, abort := checkCodeSubmissionPolicy(ctx, addPkgTx(allowed), vmk)
	assert.False(t, abort, "allowlisted add_package must pass")
	_, abort = checkCodeSubmissionPolicy(ctx, runTx(allowed), vmk)
	assert.False(t, abort, "allowlisted run must pass")

	// Non-allowlisted signer: rejected for both message types.
	rando := crypto.AddressFromPreimage([]byte("rando"))
	res, abort := checkCodeSubmissionPolicy(ctx, addPkgTx(rando), vmk)
	assert.True(t, abort, "non-allowlisted add_package must be rejected")
	assert.Contains(t, res.Log, "not authorized to submit code")
	res, abort = checkCodeSubmissionPolicy(ctx, runTx(rando), vmk)
	assert.True(t, abort, "non-allowlisted run must be rejected")
	assert.Contains(t, res.Log, "not authorized to submit code")
}

func TestCSP_PermissionedIgnoresNonCodeMsgs(t *testing.T) {
	ctx, vmk := setupCSPEnv(t)
	allowed := crypto.AddressFromPreimage([]byte("allowed"))
	p := vm.DefaultParams()
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{allowed}
	require.NoError(t, vmk.SetParams(ctx, p))

	// A non-code vm msg (exec) from a non-allowlisted signer is untouched.
	rando := crypto.AddressFromPreimage([]byte("rando"))
	tx := std.Tx{Msgs: []std.Msg{vm.MsgCall{Caller: rando, PkgPath: "gno.land/r/x", Func: "F"}}}
	_, abort := checkCodeSubmissionPolicy(ctx, tx, vmk)
	assert.False(t, abort, "permissioned policy must not touch vm/exec")
}

func TestCSP_PermissionedEmptyAllowlistBricksAll(t *testing.T) {
	ctx, vmk := setupCSPEnv(t)
	// Validate() accepts permissioned with an empty CodeSubmitters list.
	p := vm.DefaultParams()
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = nil
	require.NoError(t, vmk.SetParams(ctx, p))

	rando := crypto.AddressFromPreimage([]byte("rando"))
	_, abort := checkCodeSubmissionPolicy(ctx, addPkgTx(rando), vmk)
	assert.True(t, abort, "permissioned + empty allowlist blocks every submitter")
}

func TestCSP_ValidateNewFields(t *testing.T) {
	base := vm.DefaultParams()

	// invalid policy string
	p := base
	p.CodeSubmissionPolicy = "banana"
	assert.Error(t, p.Validate())

	// zero address in allowlist
	p = base
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{{}}
	assert.Error(t, p.Validate())

	// duplicate addresses
	a := crypto.AddressFromPreimage([]byte("dup"))
	p = base
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{a, a}
	assert.Error(t, p.Validate())

	// valid permissioned with one submitter
	p = base
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{a}
	assert.NoError(t, p.Validate())
}
