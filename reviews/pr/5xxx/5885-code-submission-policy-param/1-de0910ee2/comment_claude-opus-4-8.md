# Review: PR [#5885](https://github.com/gnolang/gno/pull/5885)
Event: REQUEST_CHANGES

## Body
The `docs` check fails on an unrelated remote-link lint for `https://docs.gno.land/` in `docs/MANIFESTO.md`, not on anything in this PR. Repros run at de0910ee2.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5885-code-submission-policy-param/1-de0910ee2/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/sdk/vm/pb3_gen.go:1173 [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/pb3_gen.go#L1173)
`pb3_gen.go` and `vm.proto` are generated from the `Params` struct, but this codec block was hand-written and `vm.proto` was never updated, so `genproto` and `genproto2` are red. The regenerated code is behaviorally identical, so there is no wire-format risk, but both checks block merge. Regenerate `vm.proto` and `pb3_gen.go` from the struct with `make -C misc/genproto && make -C misc/genproto2` instead of hand-editing.

## gno.land/pkg/gnoland/app.go:1134 [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/gnoland/app.go#L1134)
Missing test: `checkCodeSubmissionPolicy` and the new `Params.Validate` branches have no coverage, and the "verify unauthorized MsgAddPackage is rejected" box is unchecked. The gate reads only params and each message's signers, so a bare `std.Tx` drives it without signatures.

<details><summary>test cases</summary>

```go
// Copy into gno.land/pkg/gnoland/code_submission_policy_test.go, then:
//   go test -v -run 'TestCSP' ./gno.land/pkg/gnoland/
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
	require.NoError(t, vmk.SetParams(ctx, vm.DefaultParams()))
	rando := crypto.AddressFromPreimage([]byte("rando"))
	_, abort := checkCodeSubmissionPolicy(ctx, addPkgTx(rando), vmk)
	assert.False(t, abort, "permissionless must allow any add_package")
	_, abort = checkCodeSubmissionPolicy(ctx, runTx(rando), vmk)
	assert.False(t, abort, "permissionless must allow any run")
}

func TestCSP_EmptyPolicyTreatedPermissionless(t *testing.T) {
	ctx, vmk := setupCSPEnv(t)
	p := vm.DefaultParams()
	p.CodeSubmissionPolicy = "" // pre-upgrade chains never wrote the field
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

	_, abort := checkCodeSubmissionPolicy(ctx, addPkgTx(allowed), vmk)
	assert.False(t, abort, "allowlisted add_package must pass")
	_, abort = checkCodeSubmissionPolicy(ctx, runTx(allowed), vmk)
	assert.False(t, abort, "allowlisted run must pass")

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
	rando := crypto.AddressFromPreimage([]byte("rando"))
	tx := std.Tx{Msgs: []std.Msg{vm.MsgCall{Caller: rando, PkgPath: "gno.land/r/x", Func: "F"}}}
	_, abort := checkCodeSubmissionPolicy(ctx, tx, vmk)
	assert.False(t, abort, "permissioned policy must not touch vm/exec")
}

func TestCSP_PermissionedEmptyAllowlistBricksAll(t *testing.T) {
	ctx, vmk := setupCSPEnv(t)
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

	p := base
	p.CodeSubmissionPolicy = "banana"
	assert.Error(t, p.Validate())

	p = base
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{{}}
	assert.Error(t, p.Validate())

	a := crypto.AddressFromPreimage([]byte("dup"))
	p = base
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{a, a}
	assert.Error(t, p.Validate())

	p = base
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{a}
	assert.NoError(t, p.Validate())
}
```
</details>

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5885 -R gnolang/gno
cat > gno.land/pkg/gnoland/code_submission_policy_test.go <<'EOF'
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

func TestCSP_PermissionedRejectsNonAllowlisted(t *testing.T) {
	ctx, vmk := setupCSPEnv(t)
	allowed := crypto.AddressFromPreimage([]byte("allowed"))
	p := vm.DefaultParams()
	p.CodeSubmissionPolicy = vm.CodeSubmissionPolicyPermissioned
	p.CodeSubmitters = []crypto.Address{allowed}
	require.NoError(t, vmk.SetParams(ctx, p))

	allowedTx := std.Tx{Msgs: []std.Msg{vm.MsgAddPackage{Creator: allowed}}}
	_, abort := checkCodeSubmissionPolicy(ctx, allowedTx, vmk)
	assert.False(t, abort, "allowlisted add_package must pass")

	rando := crypto.AddressFromPreimage([]byte("rando"))
	randoTx := std.Tx{Msgs: []std.Msg{vm.MsgRun{Caller: rando}}}
	res, abort := checkCodeSubmissionPolicy(ctx, randoTx, vmk)
	assert.True(t, abort, "non-allowlisted run must be rejected")
	assert.Contains(t, res.Log, "not authorized to submit code")
}
EOF
go test -v -run 'TestCSP' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/code_submission_policy_test.go
```

```
=== RUN   TestCSP_PermissionedRejectsNonAllowlisted
--- PASS: TestCSP_PermissionedRejectsNonAllowlisted (0.00s)
PASS
ok  	github.com/gnolang/gno/gno.land/pkg/gnoland
```
</details>

## gno.land/pkg/gnoland/app.go:1135 [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/gnoland/app.go#L1135)
The gate reads the full `Params` struct on every transaction before looking at the messages, even for a call or send on a permissionless chain. The ante handler already read the same struct for gas config at line 150 and discarded it. Scanning for an `add_package` or `run` first, and returning before the read when there is none, drops the extra metered read in the common case.

## gno.land/pkg/sdk/vm/params.go:186-205 [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/params.go#L186)
`Validate` accepts `permissioned` with an empty `CodeSubmitters`, which blocks every `add_package` and `run` chain-wide. A proposal that sets the policy but omits the submitters, or later clears the list, silently freezes all deployments. Reject that combination here, or is empty-list-as-freeze intended?

## gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:71 [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L71)
The comment block above this constant logs each consensus-hash bump and its cause, but this bump added no entry, so the last note now misattributes the value to the merged Example-test PR. Add a one-line note that the shift comes from the two new default `vm` params in genesis.
