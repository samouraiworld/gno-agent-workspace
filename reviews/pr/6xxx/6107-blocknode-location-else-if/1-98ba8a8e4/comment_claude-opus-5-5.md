# PR [#6107](https://github.com/gnolang/gno/pull/6107): fix(gnovm): keep BlockNode.Location unique across else-if

Verdict: APPROVE, no Warning: the change removes all 254 duplicate BlockNode Locations across examples, stdlibs and the filetests, and moves no Location that stored state refers to; three Suggestions ask to remove code that cannot fire.
Event: APPROVE
Model: claude-opus-5-5 at high effort, quick review, solo shape
Commit: 98ba8a8e4
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e)
Round: 1. One finder, no separate reflector, 2 candidates, both Suggestions run from scratch by an agent that was not their finder.

## gnovm/pkg/gnolang/go2gno.go:597-603 [gh](https://github.com/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e/gnovm/pkg/gnolang/go2gno.go#L597-L603) · Suggestion

Suggestion: this `sp.Num = 1` branch can go, since the [counter in `setNodeLocations`](https://github.com/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e/gnovm/pkg/gnolang/preprocess.go#L6458) already gives the else-if wrapper and the nested `IfStmt` distinct Nums.

<details><summary>Repro</summary>

A test walks every `.gno` file under `examples/`, `gnovm/stdlibs` and `gnovm/tests/files`, runs `setNodeLines` and `setNodeLocations`, and counts BlockNodes sharing a Location, at the merge base, at the head, and at the head with `go2gno.go` restored to the merge base:

```
LOCDUMP_ROOT=$PWD/examples:$PWD/gnovm/stdlibs:$PWD/gnovm/tests/files LOCDUMP_OUT=/tmp/head.tsv go test -count=1 -run TestSoloFinderLocDump -v ./gnovm/pkg/gnolang
merge base:                  files=4253 blocknodes=66204 duplicate_locations=254
head:                        files=4253 blocknodes=66204 duplicate_locations=0
head, go2gno.go at base:     files=4253 blocknodes=66204 duplicate_locations=0
```

| Node | merge base Num | head Num | head without the `Go2Gno` branch |
| --- | --- | --- | --- |
| else-if wrapper `IfCaseStmt` (254) | 0 | 1 | 0 |
| nested `IfStmt` (254) | 0 | 2 | 1 |
| empty else of that `IfStmt` (136) | 1 | 3 | 2 |

Without the branch, `TestBlockNodeLocationUniqueElseIf`, `TestBlockNodeLocationUniqueEmptyElse` and `go test ./gnovm/pkg/gnolang -run TestFiles` all pass, and the wrapper keeps its merge-base Location.
</details>

## gnovm/pkg/gnolang/preprocess.go:43 [gh](https://github.com/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e/gnovm/pkg/gnolang/preprocess.go#L43) · Suggestion

Suggestion: this call re-checks Locations stamped on the line above from the same `pn.PkgPath` and `fn.FileName`, so it cannot fail and can go.

## gnovm/pkg/gnolang/preprocess.go:6490-6497 [gh](https://github.com/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e/gnovm/pkg/gnolang/preprocess.go#L6490-L6497) · Suggestion

Suggestion: these two path checks never fire, because [`PredefineFileSet`](https://github.com/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e/gnovm/pkg/gnolang/preprocess.go#L42) and [`preprocess1`](https://github.com/gnolang/gno/blob/98ba8a8e4a1981b9aad23c5487387300d6eda15e/gnovm/pkg/gnolang/preprocess.go#L812-L816) stamp the same `pkgPath` and `fileName` they compare against, so only the uniqueness check needs to stay.

<details><summary>Repro</summary>

A `panic("SENTINEL-PKG")` and a `panic("SENTINEL-FILE")` at the top of these two branches never fire:

```
go test -p 1 -parallel 1 -run TestFiles ./gnovm/pkg/gnolang
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	131.644s
```
</details>
