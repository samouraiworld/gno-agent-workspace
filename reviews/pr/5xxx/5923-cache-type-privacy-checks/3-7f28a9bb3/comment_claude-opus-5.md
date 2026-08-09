# Review: [#5923](https://github.com/gnolang/gno/pull/5923)
Event: REQUEST_CHANGES

## Body
The commit gate does what it says: `runTx` returns before `endTxHook` for every mode but Deliver, and CheckTx returns before `beginTxHook` runs at all, so no query or simulation reaches `Write`. Running every `Type` kind through `assertTypeIsPublic` at 7f28a9bb3 and at the merge base, 42 cases, the two walks agree on 35; of the seven that differ, two are renamed panic strings and four are built on an unnamed `FieldType`, which the preprocessor cannot produce because it renames every unnamed parameter and result first.

- The description's benchmark table reports `RepeatedCommits`, `AlwaysNewType` and `RepeatedCommits_SelfReferential`, and the diff ships `ColdAcyclic`, `WarmAcyclic`, `ColdSelfReferential` and `WarmSelfReferential`, so the 25% self-referential regression it reports cannot be reproduced from the branch.
- Red `main / test` is `TestNodeBootWithInitialHeight`, which fails the same way on master.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/3-7f28a9bb3/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

Repros run at 7f28a9bb3.

## gnovm/pkg/gnolang/realm.go:1295-1306 [gh](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L1295-L1306) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1295)
Critical: `TypeID` drops `PkgPath` for a struct whose field names are all exported, so a public realm's committed verdict is returned for a structurally identical private type and `assertTypeIsPublic` skips the walk before comparing any package path. Both transactions commit, so the buffer-and-promote gate never applies, and the identical call is accepted on a node that has been up and rejected on one that restarted. Fix: make the key determine the packages the verdict was computed over.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5923 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_collide_test.go <<'EOF'
package gnolang

import "testing"

func TestCollide(t *testing.T) {
	store, pubPath, privPath := newPrivateDepTestStore(t)
	pub := &StructType{PkgPath: pubPath, Fields: []FieldType{{Name: "X", Type: IntType}}}
	priv := &StructType{PkgPath: privPath, Fields: []FieldType{{Name: "X", Type: IntType}}}
	if pub.TypeID() != priv.TypeID() {
		t.Fatalf("TypeIDs differ, nothing to measure: %s vs %s", pub.TypeID(), priv.TypeID())
	}
	if typeHasPrivateDep(store, pub) {
		t.Fatal("typeHasPrivateDep(pub) = true, want false")
	}
	if !typeHasPrivateDep(store, priv) {
		t.Fatal("typeHasPrivateDep(priv) = false, want true")
	}
}
EOF
go test -run TestCollide ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_collide_test.go
```

```
--- FAIL: TestCollide (0.00s)
    zz_collide_test.go:16: typeHasPrivateDep(priv) = false, want true
FAIL
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	0.195s
```

End to end, a public realm and a private realm each persisting `[3]*struct{ N int }` through a third public realm, no simulation and every transaction committing:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5923 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_collide.txtar <<'EOF'
gnoland start

gnokey maketx addpkg -pkgdir $WORK/pub -pkgpath gno.land/r/zz/pub -gas-fee 1000001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1
stdout OK!
gnokey maketx addpkg -pkgdir $WORK/warm -pkgpath gno.land/r/zz/warm -gas-fee 1000001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1
stdout OK!
gnokey maketx addpkg -pkgdir $WORK/priv -pkgpath gno.land/r/zz/priv -gas-fee 1000001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1
stdout OK!

gnokey maketx call -pkgpath gno.land/r/zz/warm -func Warm -gas-fee 5000000ugnot -gas-wanted 50000000 -chainid=tendermint_test test1
stdout OK!

! gnokey maketx call -pkgpath gno.land/r/zz/priv -func Leak -gas-fee 5000000ugnot -gas-wanted 50000000 -chainid=tendermint_test test1
stderr 'cannot persist object of type defined in the private realm gno.land/r/zz/priv'

-- pub/gnomod.toml --
module = "gno.land/r/zz/pub"
gno = "0.9"

-- pub/pub.gno --
package pub

var Slot any

func Save(cur realm, v any) {
	Slot = v
}

-- warm/gnomod.toml --
module = "gno.land/r/zz/warm"
gno = "0.9"

-- warm/warm.gno --
package warm

import "gno.land/r/zz/pub"

func Warm(cur realm) {
	var arr [3]*struct {
		N int
	}
	pub.Save(cross(cur), arr)
}

-- priv/gnomod.toml --
module = "gno.land/r/zz/priv"
gno = "0.9"
private = true

-- priv/priv.gno --
package priv

import "gno.land/r/zz/pub"

func Leak(cur realm) {
	var arr [3]*struct {
		N int
	}
	pub.Save(cross(cur), arr)
}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/zz_collide' -timeout 900s
rm gno.land/pkg/integration/testdata/zz_collide.txtar
```

```
    FAIL: testdata/zz_collide.txtar:13: unexpected "gnokey" command success
    FAIL: testdata/zz_collide.txtar:14: no match for `cannot persist object of type defined in the private realm gno.land/r/zz/priv` found in stderr
--- FAIL: TestTestdata/zz_collide (8.83s)
```

Adding `gnoland restart` before the last call turns that same call green, which is the divergence between a node that has been up and one that has not. Both files pass at the merge base ddb752cac.
</details>

## gnovm/pkg/gnolang/realm_privatedep_test.go:24-44 [gh](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm_privatedep_test.go#L24-L44) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_privatedep_test.go#L24)
Missing test: `TestTypeHasPrivateDep_PublicStruct` and `_OwnPackagePrivate` already build a pair of structs that share one `TypeID`, and pass only because `newPrivateDepTestStore` gives each of them its own store. Nothing in the suite drives two packages through one store, so a fix that moves the commit gate around would leave it green.

<details><summary>test cases</summary>

```go
func TestTypeHasPrivateDep_TypeIDCollidesOnStruct(t *testing.T) {
	store, pubPath, privPath := newPrivateDepTestStore(t)

	pub := &StructType{PkgPath: pubPath, Fields: []FieldType{{Name: "X", Type: IntType}}}
	priv := &StructType{PkgPath: privPath, Fields: []FieldType{{Name: "X", Type: IntType}}}
	if pub.TypeID() != priv.TypeID() {
		t.Fatalf("TypeIDs differ, nothing to measure: %s vs %s", pub.TypeID(), priv.TypeID())
	}

	if typeHasPrivateDep(store, pub) {
		t.Fatal("typeHasPrivateDep(pub) = true, want false")
	}
	if !typeHasPrivateDep(store, priv) {
		t.Fatal("typeHasPrivateDep(priv) = false, want true")
	}
}

func TestTypeHasPrivateDep_TypeIDCollidesOnInterface(t *testing.T) {
	store, pubPath, privPath := newPrivateDepTestStore(t)

	method := func() []FieldType {
		return []FieldType{{Name: "Get", Type: &FuncType{Results: []FieldType{{Type: IntType}}}}}
	}
	pub := &InterfaceType{PkgPath: pubPath, Methods: method()}
	priv := &InterfaceType{PkgPath: privPath, Methods: method()}
	if pub.TypeID() != priv.TypeID() {
		t.Fatalf("TypeIDs differ, nothing to measure: %s vs %s", pub.TypeID(), priv.TypeID())
	}

	if typeHasPrivateDep(store, pub) {
		t.Fatal("typeHasPrivateDep(pub) = true, want false")
	}
	if !typeHasPrivateDep(store, priv) {
		t.Fatal("typeHasPrivateDep(priv) = false, want true")
	}
}

// Passes at 7f28a9bb3: a declared type keys on PkgPath and does not collide.
func TestTypeHasPrivateDep_TypeIDCollidesNotOnDeclared(t *testing.T) {
	store, pubPath, privPath := newPrivateDepTestStore(t)

	base := func(p string) *StructType {
		return &StructType{PkgPath: p, Fields: []FieldType{{Name: "X", Type: IntType}}}
	}
	pub := &DeclaredType{PkgPath: pubPath, Name: "Data", Base: base(pubPath)}
	priv := &DeclaredType{PkgPath: privPath, Name: "Data", Base: base(privPath)}

	if typeHasPrivateDep(store, pub) {
		t.Fatal("typeHasPrivateDep(pub) = true, want false")
	}
	if !typeHasPrivateDep(store, priv) {
		t.Fatal("typeHasPrivateDep(priv) = false, want true")
	}
}
```
</details>

## gnovm/pkg/gnolang/realm.go:1337 [gh](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L1337) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1337)
Suggestion: `isPkgPrivateFromPkgPath` now runs for the realm's own package too, where master reached it only after `pkgPath != rlm.Path`, and it panics with `cannot find package value from store for path` when the store has no `PackageValue`. A sweep of every `Type` kind at both shas turns this one case from `ok` into that panic, and no path from gno source reaches it, since the realm's package is loaded before its objects are saved.

## gnovm/pkg/gnolang/realm_assertpublic_bench_test.go:76-82 [gh](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L76-L82) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L76)
Suggestion: `clearTypePrivacyMemo` replaces `typePrivacyCache.m` without taking `mu`, so the only in-tree example of touching that map shows it touched the way the type forbids. A `reset` method beside `get` and `set` would keep the lock where the type puts it.
