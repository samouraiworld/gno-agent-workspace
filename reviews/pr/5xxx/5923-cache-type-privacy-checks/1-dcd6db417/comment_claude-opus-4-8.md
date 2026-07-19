# Review: PR [#5923](https://github.com/gnolang/gno/pull/5923)
Event: REQUEST_CHANGES

## Body
The three findings below share one root cause. The memo is keyed to the lifetime of a Go `Type` object, and that lifetime is both shorter and narrower than the design assumes: it ends at every transaction boundary, and it never begins for a type carrying a method. Nothing detects that, because the benchmark and the gas guard both drive shapes where the difference cannot show.

Correctness of the memo itself holds on dcd6db417: over 5000 random type graphs with a random private subset, each queried in four shuffled rounds so earlier answers feed later ones, the memoized answer never disagreed with a cache-free reachability walk.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/1-dcd6db417/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/realm.go:1273-1276 [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1273-L1276)
The memo does not survive a transaction boundary. Every transaction gets [a fresh `cacheTypes` map](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/store.go#L254) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L254), so `GetType` hands back a different object with `privateDep` at zero. The update path, the one the description says re-pays the walk every block, never hits.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5923 -R gnolang/gno
cat > gnovm/pkg/gnolang/privdep_reach_test.go <<'EOF'
package gnolang

import (
	"io"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestPrivateDepCacheSurvivesTransaction(t *testing.T) {
	db := memdb.NewMemDB()
	tm2Store := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
	st := NewStore(nil, tm2Store, tm2Store)

	w1 := tm2Store.CacheWrap()
	tx1 := st.BeginTransaction(w1, w1, nil, nil)
	m := NewMachineWithOptions(MachineOptions{PkgPath: "gno.vm/t/hello", Store: tx1, Output: io.Discard})
	m.RunMemPackage(&std.MemPackage{
		Type: MPUserProd, Name: "hello", Path: "gno.vm/t/hello",
		Files: []*std.MemFile{{Name: "hello.gno", Body: "package hello\n\ntype Coin struct{ Denom string; Amount int }\n"}},
	}, true)
	tx1.Write()
	w1.Write()

	first := st.GetType("gno.vm/t/hello.Coin").(*DeclaredType)
	typeHasPrivateDep(st, first)
	if first.privateDep == 0 {
		t.Fatalf("privateDep = 0 after the first walk, want it memoized")
	}

	w2 := tm2Store.CacheWrap()
	tx2 := st.BeginTransaction(w2, w2, nil, nil)
	second := tx2.GetType("gno.vm/t/hello.Coin").(*DeclaredType)

	t.Logf("tx1 %p privateDep=%d / tx2 %p privateDep=%d", first, first.privateDep, second, second.privateDep)
	if second.privateDep == 0 {
		t.Fatalf("privateDep = 0 in the next transaction: the memo does not survive a transaction boundary")
	}
}
EOF
go test -v -run 'TestPrivateDepCacheSurvivesTransaction' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/privdep_reach_test.go
```

```
=== RUN   TestPrivateDepCacheSurvivesTransaction
    privdep_reach_test.go:38: tx1 0x10e673d83860 privateDep=1 / tx2 0x10e6741aa000 privateDep=0
    privdep_reach_test.go:40: privateDep = 0 in the next transaction: the memo does not survive a transaction boundary
--- FAIL: TestPrivateDepCacheSurvivesTransaction (0.00s)
FAIL
```
</details>

## gnovm/pkg/gnolang/realm.go:1371-1379 [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1371-L1379)
Any type with a method is never memoized. A bound method carries its receiver, so the walk reaches the type again and [trips `sawCycle`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm.go#L1314) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1314), discarding the whole `pending` list. `type Coin struct{ Amount int }` memoizes; adding `func (c Coin) Get() int` stops it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5923 -R gnolang/gno
cat > gnovm/pkg/gnolang/privdep_method_test.go <<'EOF'
package gnolang

import (
	"io"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestPrivateDepCacheWithMethods(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"no method", "package hello\n\ntype Coin struct{ Amount int }\n"},
		{"one method", "package hello\n\ntype Coin struct{ Amount int }\n\nfunc (c Coin) Get() int { return c.Amount }\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := memdb.NewMemDB()
			tm2Store := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
			st := NewStore(nil, tm2Store, tm2Store)
			w := tm2Store.CacheWrap()
			tx := st.BeginTransaction(w, w, nil, nil)
			m := NewMachineWithOptions(MachineOptions{PkgPath: "gno.vm/t/hello", Store: tx, Output: io.Discard})
			m.RunMemPackage(&std.MemPackage{
				Type: MPUserProd, Name: "hello", Path: "gno.vm/t/hello",
				Files: []*std.MemFile{{Name: "hello.gno", Body: tc.body}},
			}, true)

			dt := tx.GetType("gno.vm/t/hello.Coin").(*DeclaredType)
			typeHasPrivateDep(tx, dt)
			t.Logf("%s: methods=%d privateDep=%d", tc.name, len(dt.Methods), dt.privateDep)
			if dt.privateDep == 0 {
				t.Fatalf("privateDep = 0: a type with %d method(s) is never memoized", len(dt.Methods))
			}
		})
	}
}
EOF
go test -v -run 'TestPrivateDepCacheWithMethods' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/privdep_method_test.go
```

```
=== RUN   TestPrivateDepCacheWithMethods/no_method
    privdep_method_test.go:32: no method: methods=0 privateDep=1
=== RUN   TestPrivateDepCacheWithMethods/one_method
    privdep_method_test.go:32: one method: methods=1 privateDep=0
    privdep_method_test.go:34: privateDep = 0: a type with 1 method(s) is never memoized
--- FAIL: TestPrivateDepCacheWithMethods (0.00s)
    --- PASS: TestPrivateDepCacheWithMethods/no_method (0.00s)
    --- FAIL: TestPrivateDepCacheWithMethods/one_method (0.00s)
FAIL
```
</details>

## gno.land/pkg/integration/testdata/typecache_restart_gas.txtar:44-58 [↗](../../../../../.worktrees/gno-review-5923/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar#L44-L58)
This test stays green at the same `EXACT_GAS` with `getPrivateDepCache` forced to always miss. `SaveItem` only updates an existing object, so the warm and cold calls it compares are both cold and the gas equality holds for a reason unrelated to the memo. It cannot catch the metering trap it names.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5923 -R gnolang/gno
python3 - <<'PY'
p='gnovm/pkg/gnolang/realm.go'
s=open(p).read()
old='func getPrivateDepCache(t Type) (result, ok bool) {\n\tp := privateDepPtr(t)'
new='func getPrivateDepCache(t Type) (result, ok bool) {\n\tif true {\n\t\treturn false, false\n\t}\n\tp := privateDepPtr(t)'
assert old in s
open(p,'w').write(s.replace(old,new,1))
PY
go test ./gno.land/pkg/integration/ -run 'TestTestdata/typecache_restart_gas'
git checkout HEAD -- gnovm/pkg/gnolang/realm.go
```

```
ok  	github.com/gnolang/gno/gno.land/pkg/integration	3.149s
```
</details>

## gnovm/pkg/gnolang/realm_assertpublic_bench_test.go:53-60 [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L53-L60)
Missing test: the two repeated-commit benchmarks hand `assertTypeIsPublic` the same object on every iteration. That is the memo's best case and not a shape the realm save path ever presents, so the numbers say nothing about the real hit rate.

## gno.land/pkg/integration/testdata/typecache_restart_gas.txtar:2 [↗](../../../../../.worktrees/gno-review-5923/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar#L2)
Nit: the ADR path here, `gnovm/adr/prxxxx_type_privacy_dependency_cache.md`, does not exist. The file added is [`pr5923_type_privacy_dependency_cache.md`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/adr/pr5923_type_privacy_dependency_cache.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/adr/pr5923_type_privacy_dependency_cache.md#L1). Same stale name at [`realm_assertpublic_bench_test.go:50`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L50) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L50) and [`:86`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L86) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L86).
