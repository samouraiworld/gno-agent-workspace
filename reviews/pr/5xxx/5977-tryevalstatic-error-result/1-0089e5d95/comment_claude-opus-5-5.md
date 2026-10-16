# Review: [#5977](https://github.com/gnolang/gno/pull/5977)
Posted: https://github.com/gnolang/gno/pull/5977#pullrequestreview-5547557958
Event: APPROVE
Verdict: APPROVE. The two-line fix is correct and both open findings are missing tests, one for the `*ConstExpr` callee at the same-realm check and one for a negative control that only reaches `EvalStatic`'s precondition.
Model: claude-opus-5-5, standard review, solo shape: 2 finders at xhigh, judge at high
Commit: 0089e5d957dd05eb3892122d050c04a0fda6a1e9
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-5977 0089e5d95`
Round: 1. 2 finders, a reflector pass by the judge, 5 candidates, 4 from the finders run from scratch by the judge, which was not their finder, and the reflector's one Nit judged by read.

## gnovm/pkg/gnolang/preprocess.go:2064 [gh](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/preprocess.go#L2064) · [↗](../../../../../.worktrees/gno-review-5977/gnovm/pkg/gnolang/preprocess.go#L2064) · Missing test [posted](https://github.com/gnolang/gno/pull/5977#discussion_r4294486671)
Missing test: no test sends a `*ConstExpr` callee, the one [`tryEvalStatic`](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/preprocess.go#L4323-L4325) returns unevaluated, through `if err != nil`. Skipping that callee here leaves `TestTryEvalStatic` and all 57 `zrealm_crossrealm` filetests green, and `(func(cur realm))(nil)(cur)` inside a crossing function builds one from Gno source.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5977 -R gnolang/gno
python3 - <<'PY'
p = "gnovm/pkg/gnolang/preprocess.go"
s = open(p).read()
old = "ftv, err := tryEvalStatic(store, ctxpn, last, n.Func)\n"
new = old + "\t\t\t\t\t\t\t\tif _, ok := n.Func.(*ConstExpr); ok {\n\t\t\t\t\t\t\t\t\terr = fmt.Errorf(\"skip\")\n\t\t\t\t\t\t\t\t}\n"
assert s.count(old) == 1
open(p, "w").write(s.replace(old, new))
PY
go test -count=1 -run 'TestTryEvalStatic' ./gnovm/pkg/gnolang/
go test -count=1 -v -run 'TestFiles/zrealm_crossrealm' ./gnovm/pkg/gnolang/ | grep -c -- '--- PASS'
go test -count=1 -run 'TestFiles/zrealm_crossrealm' ./gnovm/pkg/gnolang/
git checkout -- gnovm/pkg/gnolang/preprocess.go
```

Every test passes with the `*ConstExpr` callee skipped, which is the gap:

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.011s
57
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	4.256s
```

The ADR's alternative 4 calls this callee not constructible. The conversion `(func(cur realm))(nil)` folds to a `*ConstExpr` holding a nil func, which reaches `GetUnboundFunc`, gets nil back and leaves the call to the runtime's `call of nil function`. The filetest below pins that route and passes at head; rejecting a `*ConstExpr` callee that holds another realm's func needs a Go test building the `CallExpr`, since no Gno source produces one.
</details>

<details><summary>test cases</summary>

```go
// gnovm/tests/files/zrealm_curcall_constnil.gno
// PKGPATH: gno.land/r/test/curnil
package curnil

// A *ConstExpr callee reaches the preprocess cur-call check; its nil
// func value defers the call to the runtime.
func Bar(cur realm) {
	(func(cur realm))(nil)(cur)
}

func main(cur realm) {
	Bar(cur)
}

// Error:
// runtime error: call of nil function
```
</details>

## gnovm/pkg/gnolang/preprocess_tryevalstatic_test.go:68 [gh](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/preprocess_tryevalstatic_test.go#L68) · [↗](../../../../../.worktrees/gno-review-5977/gnovm/pkg/gnolang/preprocess_tryevalstatic_test.go#L68) · Missing test [posted](https://github.com/gnolang/gno/pull/5977#discussion_r4294486684)
Missing test: `Nx("Undeclared")` is never preprocessed, so `tryEvalStatic` fails on the [`expression not yet preprocessed`](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/machine.go#L1060-L1064) guard, and this test passes even with every evaluation panic dropped. The caller's [`if err != nil`](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/preprocess.go#L2064) branch receives evaluation failures instead, such as `cannot resolve an interface path at static time` from an interface method like `fooer.Foo`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5977 -R gnolang/gno
python3 - <<'PY'
p = "gnovm/pkg/gnolang/preprocess.go"
s = open(p).read()
old = "\t\t\tif r == nil {\n\t\t\t\treturn // FIX: no panic -> eval succeeded -> leave err nil\n\t\t\t}\n"
new = old + "\t\t\tif !strings.Contains(fmt.Sprint(r), \"not yet preprocessed\") {\n\t\t\t\treturn\n\t\t\t}\n"
assert s.count(old) == 1
open(p, "w").write(s.replace(old, new))
PY
go test -count=1 -v -run 'TestTryEvalStatic' ./gnovm/pkg/gnolang/ | grep -E -- '--- |^ok|^FAIL'
git checkout -- gnovm/pkg/gnolang/preprocess.go
```

Both tests pass with every evaluation failure reported as success, which is the gap:

```
--- PASS: TestTryEvalStatic_ErrSetOnFailure (0.00s)
--- PASS: TestTryEvalStatic_ErrNilOnSuccess (0.00s)
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.011s
```
</details>

<details><summary>test cases</summary>

```go
// TestTryEvalStatic_ErrSetOnEvalFailure: a preprocessed expression whose
// evaluation fails must report a non-nil err.
func TestTryEvalStatic_ErrSetOnEvalFailure(t *testing.T) {
	t.Parallel()
	db := memdb.NewMemDB()
	tm2 := dbadapter.StoreConstructor(db, stypes.StoreOptions{})
	st := NewStore(nil, tm2, tm2)
	m := NewMachineWithOptions(MachineOptions{PkgPath: "gno.land/r/test/tryevaliface", Store: st})
	defer m.Release()
	pn, _ := m.RunMemPackage(&std.MemPackage{
		Type:  MPUserProd,
		Name:  "tryevaliface",
		Path:  "gno.land/r/test/tryevaliface",
		Files: []*std.MemFile{{Name: "a.gno", Body: "package tryevaliface\n\ntype Fooer interface{ Foo(cur realm) }\n\nvar fooer Fooer\n"}},
	}, true)

	// An interface method has no concrete func until call time.
	x := Preprocess(st, pn, Sel(Nx("fooer"), "Foo")).(Expr)
	_, err := tryEvalStatic(st, pn, pn, x)

	require.Error(t, err, "err must be non-nil when evaluating a preprocessed expression fails")
}
```
</details>

## SKIP gnovm/pkg/gnolang/preprocess.go:4344 [gh](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/preprocess.go#L4344) · [↗](../../../../../.worktrees/gno-review-5977/gnovm/pkg/gnolang/preprocess.go#L4344) · Nit
Nit: this comment names the patch, `FIX:`, rather than what the early return does.

Not posted: a finding about a code comment's own wording, which no enabled linter checks.
