# Review: [#6162](https://github.com/gnolang/gno/pull/6162)
Event: REQUEST_CHANGES
Model: claude-opus-5, effort high
Commit: cdf48b1a6 (latest)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/)

## Body

- Freezing `misc/deployments/pearl,sapphire,test13,topaz` keeps the paths those chains deployed but not the directories they name: each resolves its package set from [the working tree](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L475) and asks for [`./gno.land/r/sys/validators/v3`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L79) and [`./gno.land/r/demo/defi/grc20reg`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L80), so the [resolution step](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L711) exits 1 under [`set -eo pipefail`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/pearl.gno.land/gen-genesis.sh#L46). Nobody can rebuild those four published genesis files to check them against the hashes the scripts carry.

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

## misc/govdao-scripts/README.md:13-14 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/govdao-scripts/README.md?plain=1#L13-L14) · [↗](../../../../../.worktrees/gno-review-6162/misc/govdao-scripts/README.md#L13)

`./govdao add-validator-v0` and `./govdao rm-validator-v0` reach no script, since [the wrapper](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/govdao-scripts/govdao-wrapper.sh#L29) resolves a subcommand to `$CMD.sh` and the two files are still [`add-validator-v3.sh`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/govdao-scripts/add-validator-v3.sh#L53) and [`rm-validator-v3.sh`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/govdao-scripts/rm-validator-v3.sh#L42).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6162 -R gnolang/gno
GNOKEY_NAME=x CHAIN_ID=x REMOTE=x ./misc/deployments/gnoland1/govdao add-validator-v0 g1abc 1
GNOKEY_NAME=x CHAIN_ID=x REMOTE=x ./misc/deployments/gnoland1/govdao | grep validator
```

The documented command exits 1, and the listing the wrapper builds from the directory still offers the `v3` spelling:

```
error: unknown command 'add-validator-v0'
run './misc/deployments/gnoland1/govdao' without arguments to list available commands
  add-validator-from-valopers         Add a validator from the r/gnops/valopers registry via govDAO proposal.
  add-validator-v3                    Add an operator to the test-13 active valset via govDAO proposal.
  add-validator                       Add a validator via govDAO proposal.
  rm-validator-v3                     Remove an operator from the test-13 active valset via govDAO proposal.
  rm-validator                        Remove a validator via govDAO proposal.
```

Both scripts also keep `valv3` as the import alias for the `v0` path.
</details>

## contribs/gnofaucet/github/testdata/events.2.json:33041 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/contribs/gnofaucet/github/testdata/events.2.json#L33041) · [↗](../../../../../.worktrees/gno-review-6162/contribs/gnofaucet/github/testdata/events.2.json#L33041)

Test: this captured `diff_hunk` now asserts bytes GitHub never sent, and so do [events.3.json:3995](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/contribs/gnofaucet/github/testdata/events.3.json#L3995) and [issues.1.json:5799](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/contribs/gnofaucet/github/testdata/issues.1.json#L5799).

<details><summary>what the three files hold</summary>

Five occurrences of `gno.land/p/moul/md` became `gno.land/p/moul/md/v0`: two in `events.2.json`, two in `events.3.json`, one in the issue body of `issues.1.json`. The same three files still carry 27 `gno.land/p/demo/avl` references, a path [#6159](https://github.com/gnolang/gno/pull/6159) moved to `p/nt/avl/v0`, so these captures are not kept in step with the tree and the parser under test reads neither field.
</details>

## gnovm/pkg/gnolang/mempackage.go:211-212 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/gnovm/pkg/gnolang/mempackage.go#L211-L212) · [↗](../../../../../.worktrees/gno-review-6162/gnovm/pkg/gnolang/mempackage.go#L211)

Suggestion: `gno.land/p/demo/tests/v0` carries the prefix the clause below tests, so the equality never decides the result.

```suggestion
	return strings.HasPrefix(pkgPath, "gno.land/p/demo/tests/") ||
```

<details><summary>applied and run</summary>

Nine paths answer the same with and without the clause: the five test-namespace ones plus `gno.land/p/demo/tests`, `gno.land/p/demo/testsfoo`, `gno.land/r/tests/vm/subtests` and `gno.land/p/nt/avl/v0`. With the suggestion applied `gofmt -l` is silent, `TestIsTestPkgPath`, `TestValidatePkgNameMatchesPath`, `TestLastPathElement` and `TestIsVersionSuffix` pass, the four `TestFiles` cases that import the moved test packages pass, and `gno test -C examples ./gno.land/p/demo/tests/...` reports `ok`.
</details>

## examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml:1 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml#L1) · [↗](../../../../../.worktrees/gno-review-6162/examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml#L1)

Suggestion: `grc1155` and `grc777` land unversioned beside [`p/nt/pausable/v0`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/pausable/v0/gnomod.toml#L1) and [`p/nt/watchdog/v0`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/watchdog/v0/gnomod.toml#L1), the two quarantined `p/nt` packages that carry one.

## SKIP misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh:6 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh#L6) · [↗](../../../../../.worktrees/gno-review-6162/misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh#L6)

Nit: the filename reads `v3` and all six paths inside route through [`r/sys/validators/v0`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh#L47).

Skipped: a stale filename no linter enforces, and renaming it renames the `scenario-18-govdao-v3-add-remove-validator` job.
