# Review: PR [#5935](https://github.com/gnolang/gno/pull/5935)
Event: COMMENT

## Body
The save-time walk was self-correcting: it covered whatever was actually being saved. [`saveFuncLocalTypes`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L917) has to be statically complete at upload time instead, and stay that way as the language grows. Verified on 4369fdca7: commenting out its call makes [`restart_local_type.txtar`](https://github.com/gnolang/gno/blob/4369fdca7/gno.land/pkg/integration/testdata/restart_local_type.txtar) fail after the restart with `unexpected type with id gno.land/r/test/lt2[...].S`, so the eager walk carries the fix rather than the [`ParentLoc` copy](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/realm.go#L1575).

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5935-eager-addpkg-local-types/1-4369fdca7/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/machine.go:917 [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L917)
A package uploaded before this merges never gets a record, and a value of its local types persisted afterwards still writes a dangling ref. Nothing re-runs this walk for a package already in the store, so it stays broken, whereas [#5894](https://github.com/gnolang/gno/pull/5894)'s save-time walk healed those on their next save. Decide here whether the target chain starts fresh, and otherwise ship the migration or keep the save-time walk as a transitional backstop.

<details><summary>the other SetType call sites</summary>

[`machine.go:896`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/machine.go#L896) and [`store.go:392`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/pkg/gnolang/store.go#L392) iterate package block values, which hold package-level types only. This walk is the third and last one.
</details>

## gnovm/pkg/gnolang/store.go:618 [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/store.go#L618)
With the save-time walk gone, this assert is the only thing that catches a missed enumeration route, and it never runs. `-tags debugAssert` appears once in the repo, in the [`test.debugAssert` target](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/Makefile#L116-L119), and no workflow under `.github/workflows/` invokes it. Running the `zrealm_localtype*` filetests under the tag in CI would make it load-bearing.

## gnovm/pkg/gnolang/machine.go:870-874 [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L870-L874)
Missing test: a file-level var initializer that already holds a local-typed value at addpkg-save time, the ordering constraint this comment states. Every committed test assigns inside `main` or inside an explicitly called `Bind`. Moving the call below the `IsRealm` block therefore leaves [`zrealm_localtype0.gno`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/tests/files/zrealm_localtype0.gno) through [`zrealm_localtype2.gno`](https://github.com/gnolang/gno/blob/4369fdca7/gnovm/tests/files/zrealm_localtype2.gno) green under `-tags debugAssert`, and [`restart_local_type.txtar`](https://github.com/gnolang/gno/blob/4369fdca7/gno.land/pkg/integration/testdata/restart_local_type.txtar) through [`restart_local_type3.txtar`](https://github.com/gnolang/gno/blob/4369fdca7/gno.land/pkg/integration/testdata/restart_local_type3.txtar) green.

<details><summary>test case</summary>

```go
// PKGPATH: gno.land/r/test
package test

type T struct{ x int }

func (t T) Get() int { return t.x }

type I interface{ Get() int }

func mk() I {
	type S struct{ T }
	return S{T{3}}
}

// Held by a file-level var initializer, so the value exists before the realm
// finalization inside saveNewPackageValuesAndTypes.
var X = mk()

var Y I

// Held by init, which runs after saveNewPackageValuesAndTypes and is persisted
// by resavePackageValues.
func init() {
	type Q struct{ T }
	Y = Q{T{4}}
}

func main(cur realm) {
	println(X.Get(), Y.Get())
}

// Output:
// 3 4
```

Passes at 4369fdca7 as `gnovm/tests/files/zrealm_localtype3.gno` under `go test -tags debugAssert -run 'TestFiles/zrealm_localtype3.gno$' ./gnovm/pkg/gnolang/`; with the call relocated it panics `dangling function-local type ref gno.land/r/test[...].S in persisted value`.
</details>
