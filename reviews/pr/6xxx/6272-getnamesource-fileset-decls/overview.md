# `StaticBlock.GetNameSourceForPath` tolerates names with no declaring file
Written by claude-opus-5-5.
PR: [gnolang/gno#6272](https://github.com/gnolang/gno/pull/6272)

## TLDR
`GetNameSourceForPath` resolves a `ValuePath` to its block node, its `NameSource` and, for a package block, the `*FileNode` declaring the name. It used the panicking `FileSet.GetDeclFor`, so a name the REPL defines straight into the package block of a package with an empty `FileSet` panicked with `name X not defined in fileset with files []`. It now calls `GetDeclForSafe` and returns a nil `*FileNode` on a miss. Nothing on master calls the helper; the assignment-target checks of [gnolang/gno#6089](https://github.com/gnolang/gno/pull/6089) are the first caller, and their REPL tests hit this panic.

## Before and after
| Where | Before | After |
| --- | --- | --- |
| a package-block name with a decl in the `FileSet`, through `GetNameSourceForPath` | the declaring file | the same file |
| a name `Reserve`d over an empty `FileSet`, the REPL's shape, through `GetNameSourceForPath` | panic `name X not defined in fileset` | a nil `*FileNode` |

## Read the code in this order
`warm` is changed code worth a close read, `cold` a test; no file here is `hot`.

### 1. `warm` · The miss handling, [`gnovm/pkg/gnolang/nodes.go`](https://github.com/gnolang/gno/pull/6272/files#diff-de40464561985f3fa10d53253bc3b01cdaa01b0e77ce52a87e2ed169f9359065R2073-R2085)
Decides what a package-block lookup returns when no file declares the name. After:
**If wrong:** a path-keyed check run in the REPL panics on any name an earlier statement defined.
```go
	if pn, ok := dbn.(*PackageNode); ok {
		fn, _, _ = pn.GetDeclForSafe(nsrc.NameExpr.Name)
	} else {
```

### 2. `cold` · The regression test, [`gnovm/pkg/gnolang/getnamesource_test.go`](https://github.com/gnolang/gno/pull/6272/files#diff-ebef615a053f6da9cc482abaa74c92902617af40473774eef41e8c29b11dc741R19-R52)
Reserves `outVar` in a package with an empty `FileSet` and asserts a nil file and an `NSValueDecl` name source.
**If wrong:** the panic comes back unnoticed until a caller runs in the REPL.

## Words used here
| Name | What it is |
| --- | --- |
| `GetNameSourceForPath` | `StaticBlock` method returning the block node, the declaring `*FileNode` and the `NameSource` for a `ValuePath`; slow, meant for preprocessing and `gno fix`, never runtime. |
| `NameSource` | per-slot record of where a block name came from: its `NameExpr`, its `Origin` node and its kind (`NSValueDecl` and the rest). |
| `FileSet` | the files of a package; `PackageNode` embeds a `*FileSet`, which the REPL leaves empty. |
| `GetDeclFor` / `GetDeclForSafe` | `FileSet` lookups of the file decl naming a name; the first panics on a miss, the second returns `ok` false. |
