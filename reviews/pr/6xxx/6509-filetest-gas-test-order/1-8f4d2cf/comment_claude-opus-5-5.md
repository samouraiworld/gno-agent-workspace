# Review: [#6509](https://github.com/gnolang/gno/pull/6509)
Posted: https://github.com/gnolang/gno/pull/6509#pullrequestreview-5979593429

Event: APPROVE
Verdict: APPROVE. Nothing blocks: the three Suggestions are `gno run` and `gno repl` losing the shared allocator, realm filetests still preprocessing uncharged, and a fresh-store carve-out in `TestFiles` that has lost its reason.
Model: claude-opus-5-5, quick review
Commit: 8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c
Overview: [overview](../overview.md)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6509 8f4d2cfa3`
Open the code: [gnovm/pkg/test](https://github.com/gnolang/gno/tree/8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c/gnovm/pkg/test)
Round: 1. One finder, a reflector pass, 3 candidates, each run from scratch by a judge that was not its finder; 0 refuted.

## Body

- Related suggestion: a realm filetest is preprocessed on a [transaction store](https://github.com/gnolang/gno/blob/8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c/gnovm/pkg/test/filetest.go#L471) whose allocator has no gas meter, so a `make` site costs 2800 gas there and 2925 in a main filetest. Share the filetest's allocator there too, or list realm filetests as a follow-up.

  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 6509 -R gnolang/gno
  cat > gnovm/pkg/test/realmgas_test.go <<'EOF'
  package test

  import (
  	"io"
  	"path/filepath"
  	"testing"

  	"github.com/stretchr/testify/require"
  )

  func TestRealmGas(t *testing.T) {
  	rootDir, err := filepath.Abs("../../../")
  	require.NoError(t, err)
  	opts := NewTestOptions(rootDir, io.Discard, io.Discard, nil)
  	run := func(name, src string) int64 {
  		_, gas, _, err := opts.RunFiletest(name+".gno", []byte(src), opts.TestStore)
  		require.NoError(t, err)
  		return gas
  	}
  	run("warm", "package main\n\nimport \"strings\"\n\nfunc main() { _ = strings.Repeat(\"a\", 3) }\n")
  	lit := "\n\nfunc main() {\n\tbuf := []byte(\"0123456789\")\n\t_ = string(buf)\n}\n"
  	mk := "\n\nfunc main() {\n\tbuf := make([]byte, 10)\n\t_ = string(buf)\n}\n"
  	for _, hdr := range []string{"package main", "// PKGPATH: gno.land/r/test/judge\npackage judge"} {
  		a, b := run("lit", hdr+lit), run("mk", hdr+mk)
  		t.Logf("GAS %q literal=%d make=%d delta=%d", hdr, a, b, b-a)
  	}
  }
  EOF
  (cd gnovm/pkg/test && go test -run TestRealmGas -v . | grep GAS)
  rm gnovm/pkg/test/realmgas_test.go
  ```

  The `make` site costs 125 gas less in the realm filetest, the preprocessing charge:

  ```
  GAS "package main" literal=4172 make=7097 delta=2925
  GAS "// PKGPATH: gno.land/r/test/judge\npackage judge" literal=9222 make=12022 delta=2800
  ```

  The merge base prints a realm delta of 2800 too, so realm filetests were uncharged before this branch.
  </details>

- Test: the `alloc_*` goldens pass on pooled stores, so the `alloc` clause of the fresh-store condition in [`TestFiles`](https://github.com/gnolang/gno/blob/8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c/gnovm/pkg/gnolang/files_test.go#L128) and its comment can go.

  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 6509 -R gnolang/gno
  sed -i 's/if isLong || strings.HasPrefix(subTestName, "alloc") {/if isLong {/' gnovm/pkg/gnolang/files_test.go
  (cd gnovm/pkg/gnolang && go test -count=3 -run 'TestFiles/alloc' .)
  git checkout -- gnovm/pkg/gnolang/files_test.go
  ```

  ```
  ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	57.353s
  ```

  The same run at the merge base fails eight to ten `alloc_*` goldens, which ones depending on scheduling, the case the comment describes.
  </details>

## gnovm/pkg/test/imports.go:248 [gh](https://github.com/gnolang/gno/blob/8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c/gnovm/pkg/test/imports.go#L248) · [↗](../../../../../.worktrees/gno-review-6509/gnovm/pkg/test/imports.go#L248) · Suggestion [posted](https://github.com/gnolang/gno/pull/6509#discussion_r4632187479)

Suggestion: with this allocator, `gno run` and `gno repl` machines no longer [share the store's allocator](https://github.com/gnolang/gno/blob/8f4d2cfa37a865f98e3b096cc6e27a2c58f5919c/gnovm/pkg/gnolang/machine.go#L189-L190), so store allocations escape the 500 MB cap of `gno run`. Share one allocator there too, or list both commands as follow-ups.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6509 -R gnolang/gno
perl -0pi -e 's/(\tdefer m\.Release\(\)\n)/$1\tfmt.Fprintln(os.Stderr, "SHARED", m.Store.GetAllocator() == m.Alloc)\n/' gnovm/cmd/gno/run.go
printf 'package main\n\nfunc main() { println("hi") }\n' > /tmp/x.gno
GNOROOT=$PWD go run ./gnovm/cmd/gno run /tmp/x.gno
GNOROOT=$PWD go run ./gnovm/cmd/gno run -pkgpath gno.land/r/demo/x /tmp/x.gno
git checkout -- gnovm/cmd/gno/run.go; rm /tmp/x.gno
```

The machine and its store hold different allocators in both modes:

```
SHARED false
hi
SHARED false
hi
```

The merge base prints `SHARED true` in both modes.
</details>
