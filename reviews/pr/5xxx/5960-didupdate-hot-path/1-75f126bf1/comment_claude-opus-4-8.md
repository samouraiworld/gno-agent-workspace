# Review: PR [#5960](https://github.com/gnolang/gno/pull/5960)
Event: REQUEST_CHANGES

## Body
The devirtualization is a real win, but `PkgID.eq` gives it back on amd64: taking `PkgID` by value makes every call site copy 20 bytes four times, costing about what the `memequal` call it removes cost. On 75f126bf1 I ran an interleaved core-pinned A/B on Zen4, benchstat over n=8, same bench file both sides: `DidUpdate_RealPrimitive` does not move, p=0.855, against a claimed −43.3%. `RealAttach` and `RealSwap` do improve, −14.09% and −11.34%. Giving `eq` a pointer receiver takes the same benchmarks to −46.58% geomean and `RealPrimitive` to −74.20%; tests green. Your table may well hold on M1, since the copies plausibly forward for free there, but amd64 is what validators run.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5960-didupdate-hot-path/1-75f126bf1/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/realm.go:127 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L127)
The value receiver costs more than the devirtualization saves: `eq` copies both 20-byte operands at every call site, four copies at [`realm.go:341`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L341) before a single PkgID comparison. Slicing `pid.Hashlet[0:8]` forces the operands addressable, so the inlined body cannot compare in registers, and the overlapping 16-byte stores it emits straddle a store boundary on read-back and stall forwarding on amd64. Taking the operands by pointer removes the copies and moves `RealPrimitive` from p=0.855 to −74.20%.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5960 -R gnolang/gno
cd gnovm/pkg/gnolang
go test -c -o /tmp/head.test .
echo "--- eq symbols surviving linking ---"
go tool objdump -s 'PkgID' /tmp/head.test 2>/dev/null | grep '^TEXT' | grep -i '\.eq'
echo "--- MOVUPS at the real inlined call site (realm.go:341) ---"
go tool objdump -s '\(\*Realm\)\.DidUpdate$' /tmp/head.test | grep -c 'realm.go:341.*MOVUPS'
rm -f /tmp/head.test
```

```
--- eq symbols surviving linking ---
--- MOVUPS at the real inlined call site (realm.go:341) ---
16
```
No `eq` symbol survives linking: it is inlined everywhere, so the clean three-MOV/CMP body that `-gcflags=-S` prints for the standalone `PkgID.eq` (`size=43, locals=0x0`) never executes. That body is copy-free only because the ABI already passes the two 20-byte structs on the stack. At the inlined call site the compiler has to materialize them: the 16 `MOVUPS` are four 20-byte copies into `0x58(SP)`, `0x1d4(SP)`, `0x184(SP)`, `0x10c(SP)`, two loads and two stores each. With a pointer receiver the same grep returns 4, and 0 once the intermediate `poPkgID` local goes too.
</details>

## gnovm/pkg/gnolang/realm_didupdate_bench_test.go:39-47 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm_didupdate_bench_test.go#L39)
This benchmark never reaches the code the PR changes, so it cannot show −12.8%: `benchMachine()` leaves `Stage` at `""`, and `DidUpdate` returns on the stage check at [`realm.go:295`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L295), well before the first changed line at 331. The empty `Stage` also skips the `/p/`-immutability gate, so the nil-realm shape a real transaction takes goes unexercised.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5960 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_stage_probe_test.go <<'EOF'
package gnolang

import "testing"

func TestZZStage(t *testing.T) {
	m := benchMachine()
	defer m.Release()
	t.Logf("benchMachine Stage=%q StageRun=%q equal=%v", m.Stage, StageRun, m.Stage == StageRun)
}
EOF
go test -run TestZZStage -v ./gnovm/pkg/gnolang/ 2>&1 | grep 'Stage='
echo "--- first changed line inside DidUpdate (new-file numbering) ---"
git diff -U0 $(git merge-base origin/master HEAD)..HEAD -- gnovm/pkg/gnolang/realm.go | grep '^@@' | sed -n '3p'
rm gnovm/pkg/gnolang/zz_stage_probe_test.go
```

```
    zz_stage_probe_test.go:8: benchMachine Stage="" StageRun="StageRun" equal=false
--- first changed line inside DidUpdate (new-file numbering) ---
@@ -318 +331,8 @@ func (rlm *Realm) DidUpdate(m *Machine, po, xo, co Object) {
```
The nil-realm block is lines 288-315, so the benchmark returns before line 331.
</details>

## gnovm/pkg/gnolang/hash_image.go:54 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/hash_image.go#L54)
Nit: `ObjectID.IsZero` reaches `Hashlet.IsZero` through the same value-receiver chain, so the copies apply here too and the rewrite shows no gain: 9.768n before versus 10.170n after. A pointer receiver fixes it here as well, but changes the method set on an exported type.

## gnovm/pkg/gnolang/realm.go:337 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L337)
Missing test: nothing catches an `Object` implementation overriding an `ObjectInfo` accessor, which is the assumption the devirtualization rests on. It holds today, but only a grep enforces it, and `DidUpdate` would silently bypass such an override while every other caller honored it.

<details><summary>test cases</summary>

Green at 75f126bf1; fails if any implementation shadows an accessor. The surface is wider than the `var _ Object = ...` block suggests: a `go/types` sweep finds 20 concrete implementations, since the `BlockNode` family reaches `ObjectInfo` via `StaticBlock` -> `Block`. Full file, with the refcount and owner checks elided here: [`tests/realm_devirt_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5960-didupdate-hot-path/1-75f126bf1/tests/realm_devirt_test.go)

```go
func allObjectImpls() []Object {
	return []Object{
		// carry ObjectInfo directly (the `_ Object = &X{}` block in ownership.go)
		&ArrayValue{}, &StructValue{}, &FuncValue{}, &BoundMethodValue{},
		&MapValue{}, &PackageValue{}, &Block{}, &HeapItemValue{},
		// reach ObjectInfo via StaticBlock -> Block
		&BlockStmt{}, &FileNode{}, &ForStmt{}, &FuncDecl{},
		&FuncLitExpr{}, &IfCaseStmt{}, &IfStmt{}, &PackageNode{},
		&RangeStmt{}, &SelectCaseStmt{}, &SwitchClauseStmt{}, &SwitchStmt{},
	}
}

func TestObjectInfoAccessorsAreNotOverridden(t *testing.T) {
	t.Parallel()
	for _, oo := range allObjectImpls() {
		oi := oo.GetObjectInfo()
		if oo.GetObjectInfo() != oi {
			t.Errorf("%T: GetObjectInfo() not identity-stable", oo)
		}
		pid := PkgIDFromPkgPath("gno.land/r/devirt")
		oi.SetObjectID(ObjectID{PkgID: pid, NewTime: 7})
		if oo.GetObjectID() != oi.ID {
			t.Errorf("%T: GetObjectID() != oi.ID", oo)
		}
		flags := []struct {
			name       string
			set        func(bool)
			viaO, viaI func() bool
		}{
			{"IsReal", func(b bool) {
				if b {
					oi.SetNewTime(7)
				} else {
					oi.SetNewTime(0)
				}
			}, oo.GetIsReal, oi.GetIsReal},
			{"IsDirty", func(b bool) { oi.SetIsDirty(b, 42) }, oo.GetIsDirty, oi.GetIsDirty},
			{"IsEscaped", oi.SetIsEscaped, oo.GetIsEscaped, oi.GetIsEscaped},
			{"IsNewReal", oi.SetIsNewReal, oo.GetIsNewReal, oi.GetIsNewReal},
			{"IsNewEscaped", oi.SetIsNewEscaped, oo.GetIsNewEscaped, oi.GetIsNewEscaped},
			{"IsNewDeleted", oi.SetIsNewDeleted, oo.GetIsNewDeleted, oi.GetIsNewDeleted},
			{"IsDeleted", oi.SetIsDeleted, oo.GetIsDeleted, oi.GetIsDeleted},
		}
		for _, f := range flags {
			for _, want := range []bool{true, false} {
				f.set(want)
				if f.viaO() != f.viaI() {
					t.Errorf("%T: Get%s() via Object = %v, via *ObjectInfo = %v",
						oo, f.name, f.viaO(), f.viaI())
				}
			}
		}
	}
}
```
</details>

## gnovm/pkg/gnolang/realm.go:485 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L485)
Nit: `MarkNewDeleted` has no callers left; its only one now calls the unexported [`markNewDeleted`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L410) instead. The other three wrappers keep callers, and no linter flags an exported function.

## gnovm/pkg/gnolang/realm.go:61-63 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L61-L63)
Nit: `eq` reads exactly 20 bytes, so it agrees with `==` only while `PkgID` is nothing but its `Hashlet`; the tests catch `HashSize` drift, but nothing catches a new field. Adding one fails to build only at the unkeyed literal on [`realm.go:101`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L101), and once that is keyed the build is clean, both tests pass, and `alloc.go`'s sizeof guard stays silent since the field hides in `ObjectID`'s padding, while `a == b` is false and `a.eq(b)` is true. [`IsReadonlyBy`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/ownership.go#L461) and [`isExternalRealm`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/machine.go#L2737) gate realm identity through `eq`, so that direction fails open; four lines in the idiom [`alloc.go:146`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/alloc.go#L146) already uses pin both drifts:

```go
// PkgID.eq and Hashlet.IsZero hard-code an 8+8+4 layout covering all of PkgID.
var (
	_ [unsafe.Sizeof(PkgID{}) - 20]struct{}
	_ [20 - unsafe.Sizeof(PkgID{})]struct{}
)
```

## gnovm/pkg/gnolang/realm.go:430 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L430)
Suggestion: the four `markX(oo, oi)` bodies require `oi == oo.GetObjectInfo()` and nothing checks it. `markDirty` sets the flag on `oi` but appends `oo` to `rlm.updated`, so a mismatched pair flags one object and enqueues another. Latent while the helpers stay unexported, and `debugAssert` is a build-tag const so the guard is free in production.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5960 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_pair_test.go <<'EOF'
package gnolang

import "testing"

func TestZZMarkPair(t *testing.T) {
	rlm := NewRealm("gno.land/r/x")
	rlm.Time = 1
	mk := func(nt uint64) *StructValue {
		sv := &StructValue{}
		sv.SetPkgID(rlm.ID)
		sv.SetNewTime(nt)
		return sv
	}
	a, b := mk(2), mk(3)
	rlm.markDirty(a, b.GetObjectInfo()) // mismatched pair
	t.Logf("a.GetIsDirty()=%v  b.GetIsDirty()=%v", a.GetIsDirty(), b.GetIsDirty())
	t.Logf("rlm.updated contains a: %v", len(rlm.updated) == 1 && rlm.updated[0] == Object(a))
}
EOF
go test -run TestZZMarkPair -v -tags debugAssert ./gnovm/pkg/gnolang/ 2>&1 | grep -E 'GetIsDirty|contains|PASS'
rm gnovm/pkg/gnolang/zz_pair_test.go
```

```
    zz_pair_test.go:16: a.GetIsDirty()=false  b.GetIsDirty()=true
    zz_pair_test.go:17: rlm.updated contains a: true
--- PASS: TestZZMarkPair (0.00s)
```
The flag lands on `b` while `a` is what gets enqueued for saving, and `-tags debugAssert` stays silent.
</details>

## gnovm/pkg/gnolang/machine.go:2408 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/machine.go#L2408)
Suggestion: borrow rule #3 keeps `!=` for the same `PkgID` compare that rule #2 converts thirty lines up at [`machine.go:2377`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/machine.go#L2377), in the same function. Two more expressions of the same shape stay on `!=` at [`realm.go:710`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L710) and [`realm.go:807`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L807), differing from the converted 382 and 404 only in the variable name. Those are finalize-time so leaving them is defensible, but two idioms for one comparison now coexist with nothing saying which to reach for.

## gnovm/pkg/gnolang/realm.go:434 [↗](../../../../../.worktrees/gno-review-5960/gnovm/pkg/gnolang/realm.go#L434)
Nit: this branch reads `pv.GetOwner()`/`pv.GetRefCount()` while the `else` at [`realm.go:442`](https://github.com/gnolang/gno/blob/75f126bf1/gnovm/pkg/gnolang/realm.go#L442) reads `oi.GetOwner()`. Same values, but the split invites a reader to think they differ.
