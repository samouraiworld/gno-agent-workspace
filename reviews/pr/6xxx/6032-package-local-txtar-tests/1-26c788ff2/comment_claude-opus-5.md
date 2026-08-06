# Review: PR [#6032](https://github.com/gnolang/gno/pull/6032)
Event: REQUEST_CHANGES

## Body
Moving a script under `examples/` also puts it in the path filters of [`ci-dir-gnovm.yml`](https://github.com/gnolang/gno/blob/26c788ff2/.github/workflows/ci-dir-gnovm.yml#L11) and [`ci-dir-examples.yml`](https://github.com/gnolang/gno/blob/26c788ff2/.github/workflows/ci-dir-examples.yml#L10), not only [`ci-dir-gnoland.yml`](https://github.com/gnolang/gno/blob/26c788ff2/.github/workflows/ci-dir-gnoland.yml#L12), so a one-line gas-number update now pays `main / test` at 13m15s and `gno-checks / test` at 3m55s on top of what it paid from `testdata/`.

[`make -C examples fix`](https://github.com/gnolang/gno/blob/26c788ff2/examples/Makefile#L56-L57) passes a directory and [`gno fix` takes a `.txtar` only as an explicit file target](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/cmd/gno/fix.go#L92-L93), so the next migration sweep rewrites a package's `.gno` files and walks past the script now beside them. At 26c788ff2, `go run ./gnovm/cmd/gno fix -diff -v ./examples/gno.land/r/gnoland/wugnot` prints only `wugnot.gno`.

Repros run at 26c788ff2.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6032-package-local-txtar-tests/1-26c788ff2/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gno.land/pkg/integration/testdata_test.go:24 [gh](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/testdata_test.go#L24) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/testdata_test.go#L24)
The examples root comes from [`gnoenv.RootDir`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/gnoenv/gnoroot.go#L21) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/gnoenv/gnoroot.go#L21), which reads `GNOROOT` first, so a `GNOROOT` pointing at a checkout that predates the move drops all 11 package-local scripts and the package still reports `ok`. The [only guard](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L53) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L53) counts scripts across both roots together, and the 175 under `testdata/` satisfy it on their own. A mismatched `GNOROOT` used to run all 186 and fail loudly on what it got wrong.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6032 -R gnolang/gno
stale="$(mktemp -d)"
git worktree add --detach "$stale" origin/master
echo "--- correct GNOROOT ---"
go test ./gno.land/pkg/integration/ -run 'TestTestdata/wugnot$' -count=1 2>&1 | tail -1
echo "--- stale GNOROOT ---"
GNOROOT="$stale" go test ./gno.land/pkg/integration/ -run 'TestTestdata/wugnot$' -count=1 2>&1 | tail -1
git worktree remove --force "$stale"
```

```
--- correct GNOROOT ---
ok  	github.com/gnolang/gno/gno.land/pkg/integration	3.454s
--- stale GNOROOT ---
ok  	github.com/gnolang/gno/gno.land/pkg/integration	0.031s [no tests to run]
```
</details>

## gnovm/pkg/integration/testscript.go:53-59 [gh](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L53-L59) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L53-L59)
Missing test: nothing pins that `Params.Dir` stays empty, which is the condition testscript needs before it honors `Params.Files` at all. A later `Dir` default would send `RunT` down its directory branch and leave every discovered root unrun, with no error. [`testscript_test.go`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript_test.go#L12) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript_test.go#L12) covers `FindTestScripts` and stops there.

<details><summary>test cases</summary>

```go
func TestNewTestingParamsFromRoots(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(rel string) string {
		fpath := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(fpath), 0o755))
		require.NoError(t, os.WriteFile(fpath, nil, 0o644))
		return fpath
	}
	platform := write("testdata/vm.txtar")
	local := write("r/demo/foo/foo.txtar")

	t.Run("files_list_drives_the_run", func(t *testing.T) {
		t.Parallel()

		p, err := NewTestingParamsFromRoots(t,
			filepath.Join(dir, "testdata"), filepath.Join(dir, "r"))
		require.NoError(t, err)
		assert.Empty(t, p.Dir, "a non-empty Dir makes testscript ignore Files")
		assert.Equal(t, []string{platform, local}, p.Files)
	})

	t.Run("no_script_anywhere", func(t *testing.T) {
		t.Parallel()

		_, err := NewTestingParamsFromRoots(t, t.TempDir())
		assert.ErrorContains(t, err, "no testscript file found")
	})
}
```

Full file, including the empty-root case: [`params_from_roots_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6032-package-local-txtar-tests/1-26c788ff2/tests/params_from_roots_test.go#L31).
</details>

## examples/README.md:26-27 [gh](https://github.com/gnolang/gno/blob/26c788ff2/examples/README.md?plain=1#L26-L27) · [↗](../../../../../.worktrees/gno-review-6032/examples/README.md#L26)
Nit: "unique across the whole tree" reads as the `examples/` tree, and the namespace also covers `gno.land/pkg/integration/testdata/`, which [`docs/resources/gno-testing.md`](https://github.com/gnolang/gno/blob/26c788ff2/docs/resources/gno-testing.md?plain=1#L184-L186) · [↗](../../../../../.worktrees/gno-review-6032/docs/resources/gno-testing.md#L184-L186) states as "both locations". A realm-local `params.txtar` is unique inside `examples/` and collides with [`testdata/params.txtar`](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/testdata/params.txtar#L1) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/testdata/params.txtar#L1), failing discovery for the whole suite.

## gno.land/pkg/integration/update_gas_wanted.sh:57 [gh](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/update_gas_wanted.sh#L57) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/update_gas_wanted.sh#L57)
Nit: the comment above this line asks for the roots to stay in sync with `FindTestScripts`, but `find` descends dot-directories and [the walker skips them](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L85) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L85). A script under one would get its gas numbers rewritten and never run.

## gnovm/pkg/integration/testscript.go:66-67 [gh](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L66-L67) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L66-L67)
Suggestion: testscript's directory mode also globs `*.txt`, and [this matches](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L91) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L91) `.txtar` alone for both roots. The reason given, that a bare `.txt` next to real code reads as a fixture, is an argument about `examples/` and not about `testdata/`. No `.txt` script exists in either root today.
