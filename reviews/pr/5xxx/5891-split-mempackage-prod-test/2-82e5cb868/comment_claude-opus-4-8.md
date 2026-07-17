# Review: PR [#5891](https://github.com/gnolang/gno/pull/5891)
Event: REQUEST_CHANGES

## Body
`pkg:<path>#allbutprod` is a second key namespace, and it is itself spellable as a package path. `FindPathsByPrefix` treats `#` as input to defend against; the store's other entry points treat it as impossible. That split assumption is the shared root of most of what follows, and the properties holding the two blobs together are carried by comments rather than by code or tests.

Verified on 82e5cb868 against a live `gnoland start`: the node stays up and keeps answering queries afterwards, so the cost is a log line and a caller's 500, not availability. The same script passes on [#5971](https://github.com/gnolang/gno/pull/5971)'s head, where routing the sibling to `baseStore` closes the alias as a side effect.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5891-split-mempackage-prod-test/2-82e5cb868/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/store.go:1168 [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1168)
[`QueryFile`](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/keeper.go#L1415) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper.go#L1415) answers 500 with a goroutine stack in the node's ERROR log for any path spelled `<path>#allbutprod`, from any unauthenticated client, against every package that ships a test file. The sibling key is a valid `GetMemPackage` lookup, so the nil early return never fires and the unfiltered path reaches [`MPAnyAll.Decide`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/mempackage.go#L568) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/mempackage.go#L568), which panics on the `#`. `vm/qeval` and `vm/qfuncs` take the same input and return `invalid package path` as an error.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5891 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/qfile_sibling_alias.txtar <<'EOF'
gnoland start

gnokey maketx addpkg -pkgdir $WORK/hello -pkgpath gno.land/r/hello -gas-fee 1000000ugnot -gas-wanted 30000000 -chainid=tendermint_test test1

gnokey query vm/qfile --data 'gno.land/r/hello'
stdout 'hello_test.gno'

! gnokey query vm/qfile --data 'gno.land/r/hello#allbutprod'
stdout 'is not available'

-- hello/gnomod.toml --
module = "gno.land/r/hello"
gno = "0.9"
-- hello/hello.gno --
package hello

func Hi() string { return "hi" }
-- hello/hello_test.gno --
package hello

import "testing"

func TestHi(t *testing.T) {}
EOF

go test -v -run 'TestTestdata/qfile_sibling_alias$' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/qfile_sibling_alias.txtar
```

```
> ! gnokey query vm/qfile --data 'gno.land/r/hello#allbutprod'
level=ERROR msg="Panic in RPC HTTP handler" module=rpc-server err="invalid package path \"gno.land/r/hello#allbutprod\""
  stack="... gnolang.MemPackageType.Decide(...) gnovm/pkg/gnolang/mempackage.go:568
         gnolang.(*defaultStore).GetMemPackageAll(...) gnovm/pkg/gnolang/store.go:1168
         vm.(*VMKeeper).QueryFile(...) gno.land/pkg/sdk/vm/keeper.go:1415 ..."
[stderr]
"gnokey" error: Data: unable to call RPC method abci_query, invalid status code received, 500

FAIL: testdata/qfile_sibling_alias.txtar:9: no match for `is not available` found in stdout
--- FAIL: TestTestdata/qfile_sibling_alias (1.35s)
```

Same script on [#5971](https://github.com/gnolang/gno/pull/5971)'s head ec9b0de56:

```
> ! gnokey query vm/qfile --data 'gno.land/r/hello#allbutprod'
Data: &vm.InvalidPackageError{abciError:vm.abciError{}}
    0  gno/tm2/pkg/errors/errors.go:103 - package "gno.land/r/hello#allbutprod" is not available
ok  	github.com/gnolang/gno/gno.land/pkg/integration	1.809s
```
</details>

## gnovm/pkg/gnolang/store.go:88-91 [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L88)
The doc holds for `MP*All` only. An `MP*Test` or `MP*Integration` add is [stored whole](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1021-L1023) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1021-L1023), so `IterMemPackage` yields its test files verbatim, and [`gnovm/pkg/test/imports.go:313`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/test/imports.go#L313) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/test/imports.go#L313) writes exactly that shape. Nothing breaks today because [`machine.go:330`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/machine.go#L330) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/machine.go#L330) still filters, but the doc reads as license to drop that line.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5891 -R gnolang/gno

cat > gnovm/pkg/gnolang/zz_iter_doc_test.go <<'EOF'
package gnolang

import (
	"fmt"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestZZIterDoc(t *testing.T) {
	d1, d2 := memdb.NewMemDB(), memdb.NewMemDB()
	st := NewStore(nil,
		dbadapter.StoreConstructor(d1, storetypes.StoreOptions{}),
		dbadapter.StoreConstructor(d2, storetypes.StoreOptions{}))
	st.AddMemPackage(&std.MemPackage{
		Type: MPStdlibTest, Name: "math", Path: "math",
		Files: []*std.MemFile{
			{Name: "math.gno", Body: "package math\n"},
			{Name: "math_test.gno", Body: "package math // TEST BYTES\n"},
		},
	}, MPStdlibTest)
	for mpkg := range st.IterMemPackage() {
		var names []string
		for _, f := range mpkg.Files {
			names = append(names, f.Name)
		}
		fmt.Printf("IterMemPackage yielded: path=%q type=%v files=%v\n", mpkg.Path, mpkg.Type, names)
	}
}
EOF

go test -v -run TestZZIterDoc ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_iter_doc_test.go
```

```
=== RUN   TestZZIterDoc
IterMemPackage yielded: path="math" type=MPStdlibTest files=[math.gno math_test.gno]
--- PASS: TestZZIterDoc (0.00s)
```
</details>

## gnovm/pkg/gnolang/store.go:1093 [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1093)
This doc is byte-identical to the pre-split one, but the method now returns prod-only for `MP*All`, so nothing at the declaration says the result omits test files, while [`GetMemPackageAll`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1148-L1153) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1148-L1153) right below does state its role. Picking the wrong one of the pair is how a test file reaches a consensus read.

## gnovm/pkg/gnolang/store.go:83 [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L83)
Nit: the delete-before-re-add requirement this method exists to serve is documented only on [`defaultStore.AddMemPackage`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L974-L980) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L974-L980). A caller coding against the interface cannot discover it.

## gno.land/pkg/sdk/vm/keeper.go:1433-1434 [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper.go#L1433)
Nit: this reads as a widening, and it is the opposite. [Master's `QueryDoc` called `GetMemPackage`](https://github.com/gnolang/gno/blob/5b989cad5/gno.land/pkg/sdk/vm/keeper.go#L1415) against a single blob that already held the test files, so `GetMemPackageAll` here restores pre-split behavior rather than adding a capability. A reader who takes the comment at face value could narrow the query back to `GetMemPackage` as a cleanup.

## gno.land/pkg/sdk/vm/params.go:44 [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/params.go#L44)
Nit: this un-cast rides in from [`e5235a533`](https://github.com/gnolang/gno/commit/e5235a533), which also un-cast `minWriteDepth100Default`; master rewrote that line with `int64(540)` and the merge kept it, so the block now carries three cast constants and one bare one. Value-inert, and unrelated to mempackage storage.

## gno.land/pkg/sdk/vm/keeper.go:642 [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper.go#L642)
Missing test: nothing asserts the keeper clears the sibling when a private package is redeployed with a file removed. [`TestVMKeeperAddPackage_UpdatePrivatePackage`](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/keeper_test.go#L359) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper_test.go#L359) redeploys the same file set, and [`TestDeleteMemPackageClearsStaleBlobsOnReAdd`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store_test.go#L115-L161) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store_test.go#L115-L161) does the delete-then-add by hand, so dropping this line leaves every test green while `qfile` keeps serving the deleted `_test.gno`.

<details><summary>test cases</summary>

Source: [`keeper_private_redeploy_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5891-split-mempackage-prod-test/2-82e5cb868/tests/keeper_private_redeploy_test.go) · [↗](tests/keeper_private_redeploy_test.go). Green at 82e5cb868; removing `gnostore.DeleteMemPackage(pkgPath)` turns it red while every other `TestVMKeeperAddPackage_*` stays green:

```
--- FAIL: TestVMKeeperAddPackage_PrivateRedeployClearsStaleTestFile
    Expected nil, but got: &std.MemFile{Name:"test_test.gno", Body:"...\"stale\"..."}
--- PASS: TestVMKeeperAddPackage_UpdatePrivatePackage
--- PASS: TestVMKeeperAddPackage_ChangePublicToPrivate
--- PASS: TestVMKeeperAddPackage_ChangePrivateToPublic
```
</details>

## gnovm/pkg/gnolang/store.go:1085 [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1085)
Missing test: no test reaches the clause that keeps a prod-less package's `gnomod.toml`, `LICENSE` and `README.md` in storage. [`TestFindByPrefixDeDupesSplitPackages`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store_test.go#L250-L293) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store_test.go#L250-L293) builds its prod-less package from a lone `_test.gno` with no non-`.gno` file, so deleting the clause drops every such file on the floor with both store tests still green.

<details><summary>test cases</summary>

Source: [`store_split_contract_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5891-split-mempackage-prod-test/2-82e5cb868/tests/store_split_contract_test.go) · [↗](tests/store_split_contract_test.go). Green at 82e5cb868. Four cases, each mutation-checked against the line it pins: `splitProdAllButProd` losslessness including the prod-less fold, the `GetMemPackageAll` merge and its type stamp, the `IterMemPackage` prod-only contract, and de-dup against a nested path (`alpha` plus `alpha/sub`), which is the only shape that separates an adjacent sibling suffix from a non-adjacent one.

The alias guard has its own golden, red at 82e5cb868 and green on [#5971](https://github.com/gnolang/gno/pull/5971)'s head: [`qfile_sibling_alias.txtar`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5891-split-mempackage-prod-test/2-82e5cb868/tests/qfile_sibling_alias.txtar) · [↗](tests/qfile_sibling_alias.txtar).
</details>
