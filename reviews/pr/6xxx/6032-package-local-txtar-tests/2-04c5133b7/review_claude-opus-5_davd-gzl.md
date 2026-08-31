# PR [#6032](https://github.com/gnolang/gno/pull/6032): test(gno.land): let package-local .txtar integration tests live next to their realm

URL: https://github.com/gnolang/gno/pull/6032
Author: moul | Base: master | Files: 27 | +653 -41
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 04c5133b7 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6032 04c5133b7`
Overview: [visual overview](../overview.html)

Round 2. The head moved from 26c788ff2 to 04c5133b7 over two master merges and one commit answering round 1. The first merge resolves one conflict, in `AGENTS.md`, keeping master's new verification rule beside the branch's paragraph; the second carries no conflict hunk and follows the six `commondao_*` scripts out of `examples/quarantined/`, byte-identical, where master unquarantined their realm. Every round-1 finding is answered: the `GNOROOT` Warning now fails the suite loudly, the `Dir`/`Files` coupling is pinned, a directory target covers the `.txtar` beside it, the two walkers agree on hidden directories, and the `examples/README.md` wording names both roots. What is left is that the new `gno fix` behaviour ships with a test that passes without it.

## Overview

Integration scripts for a realm used to sit in one flat folder far from the realm they test. This moves eleven of them next to the code they exercise, teaches the runner to walk both places, and makes every root prove it contributed at least one script so a wrong `GNOROOT` cannot drop a whole root in silence. Around that, four consumers follow the scripts to their new home: the gas-refresh sweep, the failing-test grep in `gno.land/Makefile`, two CI path filters, and `gno fix`, which now rewrites the `.gno` inside a `.txtar` when the directory is the target.

**Verdict: APPROVE** — the round-1 hole is closed and verified, and the one gap left is that `fix_dir_txtar.txtar` passes with the behaviour it pins switched off (1 Missing test, 1 Suggestion, 1 Nit).

## Verify first

- [`gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar:8`](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar#L8) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar#L8) — the nested archive is flattened by the parser, so this `cmp` compares two identical 51-byte files holding no `.gno`. Confirm with `txtar.ParseFile` that the outer archive holds eight entries rather than five.
- [`gno.land/pkg/integration/update_gas_wanted.sh:61`](https://github.com/gnolang/gno/blob/04c5133b7/gno.land/pkg/integration/update_gas_wanted.sh#L61) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/update_gas_wanted.sh#L61) — the sweep rewrites the repository's `examples/`, and the run in step 2 reads `$GNOROOT/examples`. Confirm the two are the same tree on the machine the sweep runs on.

## Summary

Discovery walks the two roots through [`findTestScriptsIn`](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/pkg/integration/testscript.go#L113) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L113) and hands testscript an explicit file list, with a shared `seen` map so base names stay unique across both. [`NewTestingParamsFromRoots`](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/pkg/integration/testscript.go#L53) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L53) now errors when any single root contributes nothing, which is what turns a stale `GNOROOT` back into a red run. 208 scripts are discovered today, and the `find` in the gas sweep returns the same 208.

## Missing Tests

- **[the pin passes with the behaviour removed]** `gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar:8` — txtar has no nesting, so the inner `-- gnomod.toml --` line ends `local.txtar` and the fixture never puts a `.gno` inside a script.
  <details><summary>details</summary>

  `txtar.ParseFile` on [the fixture](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar#L1) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar#L1) returns eight entries, not five: `main.gno`, `main.gno.golden`, a 51-byte `local.txtar`, `gnomod.toml`, `local.gno`, a 51-byte `local.txtar.golden`, `gnomod.toml`, `local.gno`. Both `local.txtar` and `local.txtar.golden` stop at the first nested marker and hold `# package-local integration script` plus `gnoland start`, with no `.gno` inside, so [`cmp local.txtar local.txtar.golden`](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar#L8) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/cmd/gno/testdata/fix/fix_dir_txtar.txtar#L8) compares two equal files whatever `gno fix` does. Making `txtarFilesInDir` return nothing keeps `Test_Scripts/fix/fix_dir_txtar` green; the only signal is `go vet`'s `unreachable code`.

  [`tests/fix_dir_txtar.txtar`](tests/fix_dir_txtar.txtar) is the same fixture with the two nested archives quoted and an `unquote` line ahead of the run. It passes at 04c5133b7 and fails with the directory-covers-txtar block disabled: `local.txtar and local.txtar.golden differ`.

  Fix: quote the nested archives and unquote them in the script.
  </details>

## Suggestions

- **[the sweep and the run can read different trees]** `gno.land/pkg/integration/update_gas_wanted.sh:61` — the file list is built from the repository's `examples/`, and the test run that produces the gas numbers reads `$GNOROOT/examples`.
  <details><summary>details</summary>

  The script [cds to `REPO_ROOT`](https://github.com/gnolang/gno/blob/04c5133b7/gno.land/pkg/integration/update_gas_wanted.sh#L47) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/update_gas_wanted.sh#L47) and never sets `GNOROOT`, while [`TestTestdata`](https://github.com/gnolang/gno/blob/04c5133b7/gno.land/pkg/integration/testdata_test.go#L24) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/testdata_test.go#L24) resolves its second root through [`gnoenv.GuessRootDir`](https://github.com/gnolang/gno/blob/04c5133b7/gnovm/pkg/gnoenv/gnoroot.go#L41) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/gnoenv/gnoroot.go#L41), which reads the environment variable first. Before this change the script's set and the test's set were both `testdata/`, so no environment variable could separate them. A second checkout on `GNOROOT` now sends step 2 through that tree's scripts while step 4 writes the captured numbers back here, matched by base name. The new per-root guard does not catch it: that checkout contributes scripts, so the count is satisfied.

  Fix: `export GNOROOT="$REPO_ROOT"` beside the `cd`.
  </details>

## Nits

- **[the count does not match the list]** `docs/resources/gno-testing.md:182` — [Two things to know](https://github.com/gnolang/gno/blob/04c5133b7/docs/resources/gno-testing.md?plain=1#L182) · [↗](../../../../../.worktrees/gno-review-6032/docs/resources/gno-testing.md#L182) introduces three bullets.

## Verified

- A stale `GNOROOT` is now a red run, which is the round-1 Warning: `GNOROOT=/tmp/stale-root go test ./gno.land/pkg/integration/ -run 'TestTestdata/wugnot$'` fails at `testdata_test.go:25` with `no testscript file found below /tmp/stale-root/examples`. The same command at 26c788ff2 exited 0 with `[no tests to run]`.
- The two walkers now return the same list: the script's `find` and `FindTestScripts` over both roots each yield 208 paths, `diff` clean.
- All eleven moved scripts pass at their new locations at 04c5133b7, including the six `commondao_*` that the last merge moved out of `examples/quarantined/`.
- A directory target covers the script beside it: `gno fix -diff -v ./gno.land/r/gnoland/wugnot` from `examples/` prints `wugnot.txtar` then `wugnot.gno`, and `gno fix -diff -v ./...` there reaches all eleven.
- No `.txtar` outside the six moved-script directories sits beside a `.gno`, so no unrelated archive enters a `gno fix ./...` sweep.
- `TestFindTestScripts` and `TestNewTestingParamsFromRoots` pass at 04c5133b7.

## Existing threads

- gfanton, [`FindTestScripts` has no caller](https://github.com/gnolang/gno/pull/6032#discussion_r3894897091), unresolved. Correct, and it overlaps nothing here.
- gfanton, [`make -C examples fix` is not affected](https://github.com/gnolang/gno/pull/6032#discussion_r3894897100), unresolved. Measured the same thing independently: `gno fix -v .` from `examples/` exits 0 having printed nothing, against eleven `.txtar` for `./...`. Not posted, already raised.
- gfanton, [the `find` pipeline hides a renamed root](https://github.com/gnolang/gno/pull/6032#discussion_r3894897105), unresolved. Same line as the Suggestion above and a different failure.
- gfanton, [the docs section addresses a reader who cannot run it](https://github.com/gnolang/gno/pull/6032#discussion_r3894897065), unresolved. The Nit above survives wherever that section lands.
- gfanton, [a stale path in the pr6012 ADR](https://github.com/gnolang/gno/pull/6032#discussion_r3894897068), unresolved.

## Open questions

- `FindTestScripts` and `NewTestingParamsFromRoots` disagree on the empty case: the exported walker returns an empty list, the params builder errors. Only the second has a caller, so nothing is wrong today, and gfanton's thread proposes dropping the first outright. Not posted.
- `gnofaucet / test` is red at this head and at master; unrelated to the diff, so not posted.
