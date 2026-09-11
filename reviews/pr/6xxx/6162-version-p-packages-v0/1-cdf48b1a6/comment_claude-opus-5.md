# Review: [#6162](https://github.com/gnolang/gno/pull/6162)
Posted: https://github.com/gnolang/gno/pull/6162#pullrequestreview-5177930827
Event: COMMENT
Model: claude-opus-5, effort high
Commit: cdf48b1a6 (latest)
Status: [alexiscolin](https://github.com/gnolang/gno/pull/6162#pullrequestreview-5177498571) approved with four inline findings at 10:11, and this account already replied on one at 10:26, so two sections below ship `SKIP` as duplicates.
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/)

## Body

> AI review, claude-opus-5 at high, [skills](https://github.com/davd-gzl/skills) · Status: COMMENT

- `misc/deployments/pearl,sapphire,test13,topaz` read their package set from [the working tree](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L475) and ask for [`./gno.land/r/sys/validators/v3`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L79) and [`./gno.land/r/demo/defi/grc20reg`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L80), which are gone, so the [resolution step](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L711) exits 1 under [`set -eo pipefail`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L46). A release commit named in each deployment's README would point a reader at the revision those packages still exist on.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6162 -R gnolang/gno
go build -o /tmp/gnoprobe ./gnovm/cmd/gno
(cd examples && /tmp/gnoprobe tool deplist ./gno.land/r/sys/validators/v3 ./gno.land/r/demo/defi/grc20reg)
echo "rc=$?"
rm -f /tmp/gnoprobe
```

The step the four scripts share fails on the first missing directory, which ends the run:

```
./gno.land/r/sys/validators/v3: stat …/examples/gno.land/r/sys/validators/v3: no such file or directory
rc=1
```

The same block on the merge base prints `rc=0`. `misc/deployments/gnoland1` is unaffected: its list is wildcards only.
</details>

- [`"/r/demo/grc20factory:"`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L168) renders an info link to a path no package holds, the realm sitting at `r/demo/defi/grc20factory`, and this realm is in the gnoland1 genesis set so the string freezes with it.

<details><summary>every hardcoded self-link in a deployed realm</summary>

Three `"/r/…"` or `"/p/…"` literals under `examples/gno.land/**` name no package at this head, and the diff edits all three files:

| literal | in | what it should name |
|---|---|---|
| `/r/demo/grc20reg:` | [`grc20reg.gno:158`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno#L158) | `r/nt/grc20reg/v0` |
| `/r/demo/grc20factory:` | [`grc20factory.gno:168`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L168) | `r/demo/defi/grc20factory` |
| `/p/gnops/valopers` | [`proposal.gno:76`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/gnops/valopers/proposal/proposal.gno#L76) | `r/gnops/valopers`, and this one is proposal title text rather than a link |

All three read the same at the merge base. The sweep resolves every such literal against the 325 `gnomod.toml` module paths under `examples/`, counting a prefix match as resolving.
</details>

## contribs/gnofaucet/github/testdata/events.2.json:33041 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/contribs/gnofaucet/github/testdata/events.2.json#L33041) · [↗](../../../../../.worktrees/gno-review-6162/contribs/gnofaucet/github/testdata/events.2.json#L33041) [posted](https://github.com/gnolang/gno/pull/6162#discussion_r3988511685)

Test: this captured `diff_hunk` now asserts bytes GitHub never sent, and so do [events.3.json:3995](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/contribs/gnofaucet/github/testdata/events.3.json#L3995) and [issues.1.json:5799](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/contribs/gnofaucet/github/testdata/issues.1.json#L5799).

<details><summary>what the three files hold</summary>

Five occurrences of `gno.land/p/moul/md` became `gno.land/p/moul/md/v0`: two in `events.2.json`, two in `events.3.json`, one in the issue body of `issues.1.json`. The same three files still carry 27 `gno.land/p/demo/avl` references, a path [#6159](https://github.com/gnolang/gno/pull/6159) moved to `p/nt/avl/v0`, so these captures are not kept in step with the tree. The code reads `Author`, `Login`, `CreatedAt`, `State`, `Title`, `Number` and `CommitsCount`, never either field.
</details>

## gnovm/pkg/gnolang/mempackage.go:211-212 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/gnovm/pkg/gnolang/mempackage.go#L211-L212) · [↗](../../../../../.worktrees/gno-review-6162/gnovm/pkg/gnolang/mempackage.go#L211) [posted](https://github.com/gnolang/gno/pull/6162#discussion_r3988511689)

Nit: `pkgPath == "gno.land/p/demo/tests/v0"` never changes the answer, since the `strings.HasPrefix` beside it is already true for that same string.

```suggestion
	return strings.HasPrefix(pkgPath, "gno.land/p/demo/tests/") ||
```

<details><summary>applied and run</summary>

Nine paths answer the same with and without the clause: the five test-namespace ones plus `gno.land/p/demo/tests`, `gno.land/p/demo/testsfoo`, `gno.land/r/tests/vm/subtests` and `gno.land/p/nt/avl/v0`. With the suggestion applied `gofmt -l` is silent, `TestIsTestPkgPath`, `TestValidatePkgNameMatchesPath`, `TestLastPathElement` and `TestIsVersionSuffix` pass, the four `TestFiles` cases importing the moved test packages pass, and `gno test -C examples ./gno.land/p/demo/tests/...` reports `ok`.
</details>

## examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml:1 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml#L1) · [↗](../../../../../.worktrees/gno-review-6162/examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml#L1) [posted](https://github.com/gnolang/gno/pull/6162#discussion_r3988511699)

Nit: `grc1155` and `grc777` land unversioned beside [`p/nt/pausable/v0`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/pausable/v0/gnomod.toml#L1) and [`p/nt/watchdog/v0`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/watchdog/v0/gnomod.toml#L1), the two quarantined `p/nt` packages that carry one.

## misc/deployments/gnoland1/packages.gen.txt:29 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/gnoland1/packages.gen.txt#L29) · [↗](../../../../../.worktrees/gno-review-6162/misc/deployments/gnoland1/packages.gen.txt#L29)

Suggestion: `boards2/v1` keeps a number whose `v0` never existed here, the absence this PR used to renumber `r/gov/dao/v3`.

<details><summary>what the history shows</summary>

`git log --diff-filter=A` over each `gnomod.toml` path: `r/gnoland/boards2/v0` never existed, [`r/sys/namereg/v1`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/sys/namereg/v1/gnomod.toml#L1) never had a `v0` either, and `r/gov/dao/v1` never existed while `r/gov/dao/v2` was added by [#2581](https://github.com/gnolang/gno/pull/2581) and later deleted. Both surviving `v1` packages reach the gnoland1 genesis set, so both numbers freeze at launch alongside the renumbered ones.
</details>

## SKIP examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno:158 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno#L158) · [↗](../../../../../.worktrees/gno-review-6162/examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno#L158)

Already raised and answered, so skipped; line 158 sits outside the diff and cannot carry an inline comment: https://github.com/gnolang/gno/pull/6162#discussion_r3988191161

`"/r/demo/grc20reg:"` renders an info link to a path no package holds, and [`grc20reg_test.gno:24`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/nt/grc20reg/v0/grc20reg_test.gno#L24) asserts that same string.

## SKIP misc/govdao-scripts/README.md:13-14 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/govdao-scripts/README.md?plain=1#L13-L14) · [↗](../../../../../.worktrees/gno-review-6162/misc/govdao-scripts/README.md#L13)

Already raised and answered, so skipped: https://github.com/gnolang/gno/pull/6162#discussion_r3988139475

`./govdao add-validator-v0` and `./govdao rm-validator-v0` reach no script, since [the wrapper](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/govdao-scripts/govdao-wrapper.sh#L29) resolves a subcommand to `$CMD.sh` and the two files are still [`add-validator-v3.sh`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/govdao-scripts/add-validator-v3.sh#L53) and [`rm-validator-v3.sh`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/govdao-scripts/rm-validator-v3.sh#L42).

## SKIP misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh:6 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh#L6) · [↗](../../../../../.worktrees/gno-review-6162/misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh#L6)

Nit: the filename reads `v3` and all six paths inside route through [`r/sys/validators/v0`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh#L47).

Skipped: one more `v3` that names no package, [raised on the thread](https://github.com/gnolang/gno/pull/6162#discussion_r3988139485) for two siblings.
