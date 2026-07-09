# Review: PR [#5888](https://github.com/gnolang/gno/pull/5888)
Event: REQUEST_CHANGES

## Body
Both blocking findings share a root cause: the inert `AddPackage` branch and [`EnablePackage`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L803) [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L803) are a partial copy of the normal [`AddPackage`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L583) [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L583) path that omits its storage-deposit accounting and its package-policy gates. Reasserting both where the package is activated closes them together.

The red `build` check is not a code problem: [`misc/gendocs`](https://github.com/gnolang/gno/blob/b8818eb8e/misc/gendocs/Makefile) installs [`golang.org/x/pkgsite@latest`](https://pkg.go.dev/golang.org/x/pkgsite), which now resolves to a version needing go 1.26 against the runner's go 1.25.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5888-inert-package-storage-oracle/1-b8818eb8e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/sdk/vm/keeper.go:854 [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L854)
`EnablePackage` runs `RunMemPackage` but never calls `processStorageDeposit`, so activation locks no refundable deposit and the realm's `Storage`/`Deposit` stay at zero while it holds real persisted bytes. Every later deposit computation for that realm then runs off a false-zero baseline, including a refund that divides by `rlm.Storage`. Reproduced on b8818eb8e.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5888 -R gnolang/gno
cat > gno.land/pkg/sdk/vm/zz_enable_deposit_test.go <<'EOF'
package vm

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestEnableDepositProbe(t *testing.T) {
	const src = `package stateful

var G = []string{}

func Grow(cur realm, s string) { G = append(G, s) }

func init() { G = append(G, "seed") }`
	const pkgPath = "gno.land/r/test/stateful"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "stateful.gno", Body: src},
	}
	// normal permissionless addpkg
	{
		env := setupTestEnv()
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
		c := crypto.AddressFromPreimage([]byte("creator"))
		env.acck.SetAccount(ctx, env.acck.NewAccountWithAddress(ctx, c))
		env.bankk.SetCoins(ctx, c, initialBalance)
		if err := env.vmk.AddPackage(ctx, NewMsgAddPackage(c, pkgPath, files)); err != nil {
			t.Fatal(err)
		}
		r := env.vmk.getGnoTransactionStore(ctx).GetPackageRealm(pkgPath)
		t.Logf("NORMAL       Storage=%d Deposit=%d", r.Storage, r.Deposit)
	}
	// inert submit + oracle enable
	{
		env := setupTestEnv()
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
		ap := crypto.AddressFromPreimage([]byte("oracle"))
		sub := crypto.AddressFromPreimage([]byte("submitter"))
		for _, a := range []crypto.Address{ap, sub} {
			env.acck.SetAccount(ctx, env.acck.NewAccountWithAddress(ctx, a))
			env.bankk.SetCoins(ctx, a, initialBalance)
		}
		p := DefaultParams()
		p.CodeSubmissionPolicy = CodeSubmissionPolicyInert
		p.PkgApprovers = []crypto.Address{ap}
		env.vmk.SetParams(ctx, p)
		if err := env.vmk.AddPackage(ctx, NewMsgAddPackage(sub, pkgPath, files)); err != nil {
			t.Fatal(err)
		}
		if err := env.vmk.EnablePackage(ctx, MsgEnablePackage{Approver: ap, PkgPath: pkgPath}); err != nil {
			t.Fatal(err)
		}
		r := env.vmk.getGnoTransactionStore(ctx).GetPackageRealm(pkgPath)
		t.Logf("INERT+ENABLE Storage=%d Deposit=%d", r.Storage, r.Deposit)
	}
}
EOF
go test ./gno.land/pkg/sdk/vm/ -run 'TestEnableDepositProbe' -count=1 -v 2>&1 | grep -E "Storage="
rm gno.land/pkg/sdk/vm/zz_enable_deposit_test.go
```

```
    zz_enable_deposit_test.go:31: NORMAL       Storage=3040 Deposit=304000
    zz_enable_deposit_test.go:55: INERT+ENABLE Storage=0 Deposit=0
```
</details>

## gno.land/pkg/sdk/vm/keeper.go:821 [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L821)
After typecheck the normal path enforces keeper-only gnomod gates: no development packages, no post-genesis draft, no public override of a private package, no deprecated `gno.mod` file. `EnablePackage` skips all of them, and `gnomod.WriteString` preserves `replace`/`draft` into the inert copy, so a package the normal path rejects goes live through inert+enable.

## gno.land/pkg/sdk/vm/keeper.go:831 [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L831)
Activation runs the package's `init` with `OriginCaller` set to the approver, while the normal path uses the submitter. A realm whose `init` records the deploying caller attributes ownership to the oracle, and `gnomod.toml` still records the submitter as creator, so runtime identity and metadata disagree.

## gno.land/pkg/sdk/vm/keeper.go:658 [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L658)
`AddInertPackage` is an unconditional store write, so a second inert submission at the same path overwrites the first, after the first submitter's `msg.Send` was already spent to the package address.

## gnovm/pkg/gnolang/store.go:1075 [↗](../../../../../.worktrees/gno-review-5888/gnovm/pkg/gnolang/store.go#L1075)
Inert bytes are a raw `iavlStore.Set` outside the realm object graph, so `RealmStorageDiffs` never sees them and the storage-deposit machinery cannot bill them even in principle. Under `inert` any address can park large packages in consensus state for one-time gas, with no refundable lock and no eviction if never enabled. Was an inert-submission size cap or deposit considered?

## gno.land/pkg/sdk/vm/keeper_inert_test.go:95 [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper_inert_test.go#L95)
Missing test: the lifecycle test checks resolvability and callability after enable but never asserts that activation accounted storage and locked a deposit, so the accounting gap ships green.

<details><summary>test cases</summary>

Add alongside the resolvability check, asserting the post-fix invariant against the normal-addpkg baseline (fails at b8818eb8e):

```go
rlm := gnostore.GetPackageRealm(pkgPath)
assert.NotZero(t, rlm.Storage, "activated realm must record its persisted bytes")
assert.NotZero(t, rlm.Deposit, "activation must lock a storage deposit")
```
</details>
