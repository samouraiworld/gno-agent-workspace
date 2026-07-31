# Review: PR [#6025](https://github.com/gnolang/gno/pull/6025)
Event: APPROVE

## Body
Reproduced on 0b83500c6. Copying [`addpkg_testfile_restart_gas.txtar`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/integration/testdata/addpkg_testfile_restart_gas.txtar) onto the merge base d1a33f574 makes it fail: the same deploy costs 15,671,980 or 20,595,400 gas there, decided by whether the node restarted earlier in the process. Both arrangements cost 2,862,220 on this branch.

`go test -race ./gno.land/pkg/sdk/vm/` is clean after the removal of the mutex-guarded test-stdlib cache. CI runs no race build.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6025 -R gnolang/gno
git checkout d1a33f574
git show 0b83500c6:gno.land/pkg/integration/testdata/addpkg_testfile_restart_gas.txtar \
  > gno.land/pkg/integration/testdata/gasprobe.txtar
sed 's/^gnoland restart$/# no restart/' gno.land/pkg/integration/testdata/gasprobe.txtar \
  > gno.land/pkg/integration/testdata/gasprobe_norestart.txtar
for t in gasprobe gasprobe_norestart; do
  echo "== $t"
  go test ./gno.land/pkg/integration/ -run "TestTestdata/$t\$" -v 2>&1 | grep -E 'GAS USED: +[0-9]'
done
rm -f gno.land/pkg/integration/testdata/gasprobe*.txtar
```

```
== gasprobe
        GAS USED:   20593054
        GAS USED:   20595400
== gasprobe_norestart
        GAS USED:   20593054
        GAS USED:   15671980
```

`baz` is the second number in each pair. The restart moves it by 4,921,074 gas.
</details>

- The [`TestGetter`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L147-L149) doc says what the field does but not that setting it on a consensus path reintroduces the filesystem read this PR removes. Someone wiring a new keeper call site reads the field doc, not the panic string they only reach after making the mistake.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6025-prod-only-typecheck-addpackage/1-0b83500c6/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gnovm/pkg/gnolang/gotypecheck.go:118-120 [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L118-L120)
Missing test: the nil-getter panic has no test, so nothing catches a later refactor that makes it unreachable or turns it back into a nil dereference. It is dead by construction today, since [`AddPackage`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/sdk/vm/keeper.go#L636-L643) and [`Run`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/sdk/vm/keeper.go#L1034-L1041) both pass a non-nil `Getter`, and the nil-getter `tgetter` sits behind `gimp.testing`.

<details><summary>test cases</summary>

```go
func TestTypeCheckMemPackage_NoGetterPanics(t *testing.T) {
	mpkg := &std.MemPackage{Type: MPUserAll, Name: "hello", Path: "gno.land/p/demo/hello"}
	mpkg.SetFile("hello.gno", "package hello\n\nimport \"strconv\"\n\nfunc Hi() string { return strconv.Itoa(1) }\n")
	assert.PanicsWithValue(t,
		`gotypecheck: no getter configured to import "strconv" `+
			`(set Getter/TestGetter, or set ProdOnly to skip the test passes)`,
		func() { TypeCheckMemPackage(mpkg, TypeCheckOptions{Mode: TCLatestRelaxed}) })
}
```
</details>

## gnovm/pkg/gnolang/gotypecheck.go:163-181 [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L163-L181)
Missing test: `ProdOnly` is defined here but every test that exercises it lives in `gno.land/pkg/integration`. [`gotypecheck_test.go:400-403`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck_test.go#L400-L403) builds one shared `TypeCheckOptions` for the whole table and never sets the field, so a contributor reordering the passes in [`typeCheckMemPackage`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L441) gets no signal from the package's own suite.

<details><summary>test cases</summary>

Four cases, passing at 0b83500c6. Full file: [`tests/prodonly_x_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6025-prod-only-typecheck-addpackage/1-0b83500c6/tests/prodonly_x_test.go) [↗](tests/prodonly_x_test.go)

```go
func TestTypeCheckMemPackage_ProdOnly(t *testing.T) {
	t.Parallel()

	// A test file that parses but does not type-check.
	brokenTest := func() *std.MemPackage {
		mpkg := &std.MemPackage{Type: MPUserAll, Name: "hello", Path: "gno.land/p/demo/hello"}
		mpkg.SetFile("hello.gno", "package hello\n\nfunc Hi() string { return \"hi\" }\n")
		mpkg.SetFile("hello_test.gno", "package hello\n\nfunc Broken() int { return undefinedSymbol(42) }\n")
		return mpkg
	}

	// Production code referencing a symbol only the test file declares.
	prodBorrows := func() *std.MemPackage {
		mpkg := &std.MemPackage{Type: MPUserAll, Name: "hello", Path: "gno.land/p/demo/hello"}
		mpkg.SetFile("hello.gno", "package hello\n\nfunc Hi() string { return helperOnlyInTest() }\n")
		mpkg.SetFile("hello_test.gno", "package hello\n\nfunc helperOnlyInTest() string { return \"hi\" }\n")
		return mpkg
	}

	t.Run("BrokenTestFileRejectedWithoutProdOnly", func(t *testing.T) {
		t.Parallel()
		_, err := TypeCheckMemPackage(brokenTest(), TypeCheckOptions{Mode: TCLatestRelaxed})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "undefinedSymbol")
	})

	t.Run("BrokenTestFileAcceptedWithProdOnly", func(t *testing.T) {
		t.Parallel()
		_, err := TypeCheckMemPackage(brokenTest(), TypeCheckOptions{Mode: TCLatestRelaxed, ProdOnly: true})
		assert.NoError(t, err)
	})

	// The production pass covers exactly the file set the VM runs, so this
	// stays rejected with ProdOnly set.
	t.Run("ProdCannotBorrowFromTestFile", func(t *testing.T) {
		t.Parallel()
		_, err := TypeCheckMemPackage(prodBorrows(), TypeCheckOptions{Mode: TCLatestRelaxed, ProdOnly: true})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "undefined: helperOnlyInTest")
	})

	// An unparseable test file fails before any type-check pass, so ProdOnly
	// does not let it through.
	t.Run("UnparseableTestFileStillRejected", func(t *testing.T) {
		t.Parallel()
		mpkg := &std.MemPackage{Type: MPUserAll, Name: "hello", Path: "gno.land/p/demo/hello"}
		mpkg.SetFile("hello.gno", "package hello\n\nfunc Hi() string { return \"hi\" }\n")
		mpkg.SetFile("hello_test.gno", "package hello\n\nfunc Broken( {{{ not gno at all ]]]\n")
		_, err := TypeCheckMemPackage(mpkg, TypeCheckOptions{Mode: TCLatestRelaxed, ProdOnly: true})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hello_test.gno")
	})
}
```
</details>

## gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar:31 [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L31)
Nit: `type check failed` matches any type error anywhere in the package, so the `baz` case stays green even if the rejection stopped coming from the production pass. The [`aaa`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L39) and [`bbb`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L42) cases name the exact symbol; assert `undefinedInProd` here too.
