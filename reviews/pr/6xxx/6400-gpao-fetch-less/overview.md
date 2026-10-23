# perf(gpao): fetch only the files verification reads
Written by claude-opus-5-5.
PR: [gnolang/gno#6400](https://github.com/gnolang/gno/pull/6400)

## TLDR

The approval oracle downloaded every file of every package a candidate imports, tests, readmes and licences included, though its checks only ever read the source code. On a slow node that alone could push one check past its time limit, and a file the node refused to hand over made the import look missing. The oracle now downloads only the source files and the module file, so checks run with fewer requests and a stray unreadable file no longer holds a package back.

## The file listing `vm/qfile` serves

`rpcGetter.fetch` rebuilds a package the chain holds from the node's `vm/qfile` query: one call returns the package's file names, one per line, then one call per name returns that file's body. The listing names every stored file, production `.gno` sources, `_test.gno` files, flattened `_filetest.gno` files, `README.md`, a licence, and the module file. Each name is a round trip to the node, so the cost of resolving an import grows with every file it ships, read or not.

## What verification reads of an import

`verifier.verifyPackage` runs two stages, the stages the validator re-runs when it enables a package. The type check runs with `ProdOnly: true`, and the preprocess parses each import with `ParseMemPackageAsType(dep, gno.MPUserProd)`. Neither stage looks at an import's tests, docs or licence; both need its production `.gno` files, and the type check reads the module file for the gno version under `TCLatestStrict`, panicking with `gnomod.toml not found` when a dependency carries none.

## `readByVerify`, the filter in `fetch`

`fetch` asks the node for a file's body only when `readByVerify` keeps its name. The after side, from the pinned head:

```go
func readByVerify(name string) bool {
	switch name {
	case "gnomod.toml", "gno.mod":
		return true
	}
	return strings.HasSuffix(name, ".gno") && !gno.IsTestFile(name)
}
```

Both module file names are kept because a stored package may carry either: `gno.mod` is still an accepted file name in `gnovm/pkg/gnolang/mempackage.go`. `gno.IsTestFile` matches `_test.gno` and `_filetest.gno`, so a flattened filetest is dropped with the tests. A file the listing names and the node will not serve, such as a dotless `LICENCE`, is never asked for, so it can no longer make a live import look absent.

## `packageName`, the name of a rebuilt package

The rebuilt `std.MemPackage` needs a name, which `typeCheckMemPackage` writes `.gnobuiltins.gno` under. Before, `packageName` walked a listing that mixed production files and filetests, so it preferred a production file's clause, trimmed a `_test` suffix, and fell back to the filetests' clause or the literal `filetests`. After, `fetch` hands it production files only, so it returns the package clause of the first `.gno` file that parses, and `""` when none does. The comment in `fetch` relies on `AddPackage` refusing a package without a production file, which the keeper has done since gnolang/gno#5892.

## `prepare` and `verifyPackage`

The oracle's child process runs `verifier.prepare` first, under the prepare budget: it walks the candidate's chain imports and builds each into the base store, so the stages that follow find them compiled. Before, `verifyPackage` also called `test.LoadImports` itself, a second preload of the same imports; after, it does not, and the stages resolve anything still missing through the getter `injectChainGetter` installed. `TestPrepareBuildsDependenciesBeforeTheBudget` is the test that pins prepare's builds reaching the stages.

## Before and after

**Before:** resolving an import sent one `vm/qfile` query per stored file through `fetch`, tests, docs and licence included, and an unservable licence left every importer pending as unavailable.

**After:** `fetch` sends one query for the listing and one per production `.gno` file and module file, and an importer of a package with an unservable licence is judged on its code.

## What a user notices

- An operator running `gpao` against a slow node sees verifications that import large chain packages finish inside the prepare budget more often, since fewer queries go out per import.
- A package importing a dependency that ships an unservable `LICENCE` gets a verdict instead of staying pending.

## What it leaves open

- A live dependency holding only test files, a shape stored only before gnolang/gno#5892, is rebuilt with no `.gno` file and an empty name, and `prepare` reports it as a prebuild failure of the oracle's tree. Nothing reaches that case yet.
- The `gno.mod` branch of `readByVerify` has no test serving a dependency that carries it.

## Words used here

| Name | What it is |
| --- | --- |
| `vm/qfile` | The node's ABCI query returning a package's file listing, or one file's body when the path names a file. |
| `rpcGetter.fetch` | The function in `contribs/gpao/getter.go` that rebuilds a chain package from `vm/qfile` answers. |
| `readByVerify` | The filter deciding which listed files `fetch` asks the node for. |
| `packageName` | Derives a rebuilt package's name from its first parseable `.gno` file. |
| `verifier.prepare` | Builds the candidate's chain imports under the prepare budget, before the stages run. |
| `verifier.verifyPackage` | Runs the type check and the preprocess, the validator's two enable-time stages. |
| `TCLatestStrict` | The type-check mode requiring every package's module file to declare the latest gno version. |
| prepare budget | The time the oracle's child gets to fetch and build imports, the `-prepare-budget` flag. |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test, a doc or a generated file.

### 1. `hot` · The fetch filter, [`contribs/gpao/getter.go`](https://github.com/gnolang/gno/pull/6400/files#diff-be87ed3eff70106d6ffe1984775bf53150cabb5ebeeccbb339b898b92b21bbe5R179-R224)
Decides which listed files of an import are fetched, and names the rebuilt package.
**If wrong:** an import is rebuilt without a file a stage reads, and the importer is rejected for code that is not its own.

### 2. `hot` · The package name, [`contribs/gpao/getter.go`](https://github.com/gnolang/gno/pull/6400/files#diff-be87ed3eff70106d6ffe1984775bf53150cabb5ebeeccbb339b898b92b21bbe5R250-R262)
Returns the first parseable clause among the production files `fetch` kept.
**If wrong:** a dependency gets an empty or wrong name and fails to build.

### 3. `warm` · The two stages, [`contribs/gpao/verifier.go`](https://github.com/gnolang/gno/pull/6400/files#diff-be24eee787e5e016b7f0b18194406087436484a9234e8de45b61f924e16573f3R140-R185)
Runs the type check and the preprocess without a second preload of the imports.
**If wrong:** dependencies compile again on the verify budget, or an import resolves differently from the validator.

### 4. `cold` · The operator's description, [`contribs/gpao/README.md`](https://github.com/gnolang/gno/pull/6400/files#diff-2b039c2695b0957e35b702da1981db3e47b58ea1e4b75ea271780b23265e72f8)
Tells operators where imports come from and when a package stays pending.
**If wrong:** an operator expects a pending package where the oracle gives a verdict.

</details>
