# Review: [#6400](https://github.com/gnolang/gno/pull/6400)
Posted: https://github.com/gnolang/gno/pull/6400#pullrequestreview-5630568030
Event: COMMENT
Verdict: APPROVE. A Suggestion on dependencies that hold only test files, a Missing test for the `gno.mod` branch of `readByVerify` and a README Nit remain open, and none of them blocks.
Model: claude-opus-5-5, standard review, solo shape: two finders, then one judge and writer
Commit: 73bc358eb
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6400 73bc358eb`
Round: 1. Two finders, then one judge and writer answering the completeness question, five candidates, each run or read from scratch by an agent that was not its finder; four confirmed, one refuted, the one Nit read rather than run.

## Body

> AI review, claude-opus-5-5, standard review, reviewed at [73bc358](https://github.com/gnolang/gno/pull/6400/commits/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4), [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6400-gpao-fetch-less/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6400-gpao-fetch-less/1-73bc358eb/claims.md) · Status: APPROVE

- Nit: [`contribs/gpao/README.md:190`](https://github.com/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4/contribs/gpao/README.md#L190) says an unservable import file leaves the importer pending, while `fetch` never asks for a `LICENCE`, so [`TestUnservedLicenceDoesNotHoldUpImporters`](https://github.com/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4/contribs/gpao/parkeddep_test.go#L273) expects approval.

## contribs/gpao/getter.go:220 [gh](https://github.com/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4/contribs/gpao/getter.go#L220) · [↗](../../../../../.worktrees/gno-review-6400/contribs/gpao/getter.go#L220) · Missing test [posted](https://github.com/gnolang/gno/pull/6400#discussion_r4361244595)

Missing test: removing `"gno.mod"` from this case keeps the suite green, since no test serves a dependency carrying that file.

<details><summary>test cases</summary>

```go
// A dependency whose module file is the legacy gno.mod carries it to the
// typecheck, which refuses a dependency with no module file at all.
func TestVerifierResolvesChainOnlyImportWithLegacyGnoMod(t *testing.T) {
	const depPath = "gno.land/p/test/legacy"
	dep := chainPackage(depPath, "package legacy\n\nfunc Add(a, b int) int { return a + b }\n")
	for _, f := range dep.Files {
		if f.Name == "gnomod.toml" {
			f.Name, f.Body = "gno.mod", "module "+depPath+"\n"
		}
	}
	dep.Sort()
	mpkg := chainPackage("gno.land/p/test/user",
		"package user\n\nimport \"gno.land/p/test/legacy\"\n\nfunc Use() int { return legacy.Add(1, 2) }\n")

	v := newRPCVerifier(t, dep)
	require.NoError(t, v.prepare(mpkg))
	require.NoError(t, v.verifyPackage(mpkg),
		"a dependency's gno.mod must reach the typecheck")
}
```

With `case "gnomod.toml":` in place of this line, the same dependency answers:

```text
verifyPackage: verification panicked: gnomod.toml not found for package "gno.land/p/test/legacy"
```

and `go test -run 'TestRPCGetter|TestVerifier|TestPackageName|TestOracle|TestPrepare' .` in `contribs/gpao` reports `ok`.

</details>

## contribs/gpao/getter.go:192 [gh](https://github.com/gnolang/gno/blob/73bc358ebb7fbe473d233dd2fa84fa949aac0ad4/contribs/gpao/getter.go#L192) · [↗](../../../../../.worktrees/gno-review-6400/contribs/gpao/getter.go#L192) · Suggestion [posted](https://github.com/gnolang/gno/pull/6400#discussion_r4361244612)

Suggestion: reject the importer here, since this skip leaves a dependency holding only test files with no package name and the importer pending on `could not prebuild`.

<details><summary>checks</summary>

`AddPackage` has refused this shape since #5892. A dependency holding `gnomod.toml` and `onlytests_test.gno` only, imported by `gno.land/p/test/user` through `newRPCVerifier`, answers at this head:

```text
prepare: could not prebuild "gno.land/p/x/onlytests", imported by "gno.land/p/test/user": invalid mempackage: invalid package name "" (that package is not the candidate's code -- ... check this oracle's tree)
```

The same test at the merge base gets past `prepare` and fails the type check on the import:

```text
prepare: <nil>
verifyPackage: gno.land/p/test/user/user.gno:5:19: undefined: onlytests.X
```

No package under `examples/` has this shape, so only user packages stored before the production-file check carry it.

</details>
