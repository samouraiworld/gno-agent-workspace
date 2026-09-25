# PR [#6108](https://github.com/gnolang/gno/pull/6108): docs(gnovm): document and pin the ban on dot imports
Verdict: APPROVE, on one Nit and no Warning: the doc and the new filetest match what the VM prints, the deleted backup file was read by no runner, and the only finding is that the reworded `tryPredefine` case can never run and can go.
Event: APPROVE
Model: claude-opus-5-5 at high effort, quick review, solo shape
Commit: b512ada64
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/b512ada646e132543f32d48631c8e1bf40a4d1d9) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/b512ada646e132543f32d48631c8e1bf40a4d1d9)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6108 b512ada64`
Round: 1. One agent over 38 changed lines as finder, judge and writer, no reflector, four candidates, three settled by a read or a grep and the Nit run with sentinels at the head and at the merge base.

## gnovm/pkg/gnolang/preprocess.go:5605 [gh](https://github.com/gnolang/gno/blob/b512ada646e132543f32d48631c8e1bf40a4d1d9/gnovm/pkg/gnolang/preprocess.go#L5605) · Nit

Nit: this `case "."` can go with its message, since [`initStaticBlocks2`](https://github.com/gnolang/gno/blob/b512ada646e132543f32d48631c8e1bf40a4d1d9/gnovm/pkg/gnolang/preprocess.go#L491) rejects every dot import before any predefine walk and [`import2.gno`](https://github.com/gnolang/gno/blob/b512ada646e132543f32d48631c8e1bf40a4d1d9/gnovm/tests/files/import2.gno) passes whatever the message says.

<details>

<summary>Repro</summary>

A sentinel in this branch leaves the filetest green, a sentinel in `initStaticBlocks2` turns it red, and the merge base's `preprocess.go` passes the new filetest unchanged.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6108 -R gnolang/gno
cd gnovm/pkg/gnolang
t() { go test . -run 'TestFiles/^import2.gno$' -count=1 2>&1 | grep -m2 'SENTINEL\|^ok\|^FAIL'; }
sed -i '5606s/dot imports not allowed in gno/SENTINEL-tryPredefine/' preprocess.go; t; git checkout -- preprocess.go
sed -i '491s/dot imports not allowed in gno/SENTINEL-isb2/' preprocess.go; t; git checkout -- preprocess.go
```

```text
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.291s
            +main/import2.gno:3:8-19: SENTINEL-isb2
--- FAIL: TestFiles (0.03s)
```

Both callers of the predefine walk run `initStaticBlocks` over the file first: [`PredefineFileSet`](https://github.com/gnolang/gno/blob/b512ada646e132543f32d48631c8e1bf40a4d1d9/gnovm/pkg/gnolang/preprocess.go#L43) and [`Preprocess`](https://github.com/gnolang/gno/blob/b512ada646e132543f32d48631c8e1bf40a4d1d9/gnovm/pkg/gnolang/preprocess.go#L750).

</details>
