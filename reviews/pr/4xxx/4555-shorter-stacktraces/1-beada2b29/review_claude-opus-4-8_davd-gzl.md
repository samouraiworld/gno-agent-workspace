# PR #4555: chore: shorter and less metadata leaking stacktraces

URL: https://github.com/gnolang/gno/pull/4555
Author: moul | Base: master | Files: 2 (intended) | +200 -2 (intended)
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: beada2b29 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4555 beada2b29`

**TL;DR:** Stack traces and message traces in tm2 errors print absolute file paths from the machine that built the binary, like `/home/moul/p/.ws/gh/gnolang/gno/...`. This PR rewrites those paths to short forms (`gno/...`, `mod/...`, `toolchain/...`) so traces are shorter and leak less about the build host. The same change is already on master.

**Verdict: CLOSE** — the PR's entire substantive change is already merged to master (identical patch, landed via #4553), the branch is thousands of commits stale, and the only thing left unique to it is three unrelated files that would be committed if merged.

## Summary
The intended change adds `stripBuildDir` to `tm2/pkg/errors/errors.go` and calls it from the two trace-rendering paths, plus a unit test, two files totalling +200 -2 (matching the codecov report). That exact patch is already on `gnolang/gno` master: `tm2/pkg/errors/errors.go` and `errors_test.go` on master are byte-identical to this branch, and the `errors.go` hunk has the same `git patch-id` (`04700572a2...`) as the twin commit `038bb1173`, which moul folded into PR #4553 ("fix(gnoland): genesis and e2e", merged 2025-08-29). So merging #4555 adds nothing of the feature. What it would add is contamination: `misc/jaekwon/tictac.md`, `misc/jaekwon/gnoland-whitepaper.md`, and a 984 KB binary, all swept onto the branch through Jae Kwon commits and none on master. The branch HEAD (`beada2b29`) also predates a very large amount of master history, so it is far past a clean rebase. The PR is an RFC whose tradeoff the author and a maintainer left unresolved in the thread; the code question is moot now that the change shipped elsewhere.

## Examples
The shortening already lives on master. Its current behavior, for reference (these are observations against the merged code, not findings against this PR):

| `$GOMOD` build root | input path | output |
|---|---|---|
| `/home/u/gno` | `/home/u/gno/tm2/pkg/std/errors.go` | `gno/tm2/pkg/std/errors.go` |
| `/home/u/gno` | `/home/u/gno-simple-e2e/tm2/pkg/std/errors.go` | `gno//home/u/gno-simple-e2e/tm2/pkg/std/errors.go` (leak) |
| (unset, deployed binary) | `/home/u/gno/tm2/pkg/std/errors.go` | `/home/u/gno/tm2/pkg/std/errors.go` (unchanged) |
| any | `/home/u/go/pkg/mod/golang.org/x/sync@v0.13.0/errgroup/errgroup.go` | `mod/golang.org/x/sync@v0.13.0/errgroup/errgroup.go` |
| any | `/home/u/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.1.linux-amd64/src/runtime/asm_amd64.s` | `toolchain/runtime/asm_amd64.s` |

## Critical (must fix)
None. The PR should not be merged at all (the change already shipped), so there is nothing to fix in it. The blocker against merging is recorded under Why close, the residual code concerns under Open questions.

## Warnings (should fix)
None against this PR. The behavior concerns now apply to merged master code and are listed in Open questions.

## Nits
None.

## Missing Tests
None against this PR.

## Suggestions
None against this PR.

## Why close
- **[change already on master]** [`errors.go:44`](https://github.com/gnolang/gno/blob/beada2b29/tm2/pkg/errors/errors.go#L44) · [↗](../../../../../.worktrees/gno-review-4555/tm2/pkg/errors/errors.go#L44) — `stripBuildDir` is live on `gnolang/gno` master, same patch as this PR.
  <details><summary>details</summary>

  The live file `tm2/pkg/errors/errors.go` on master contains `stripBuildDir` (confirmed via the GitHub contents API), introduced by commit `6ae295f3b` in merged PR #4553. The twin commit `038bb1173` (also moul, identical message "chore: shorter and less metadata leaking stacktraces") carries the same `errors.go` patch-id as this PR's `7e7a8fd85`, and both `errors.go` and `errors_test.go` on master are byte-identical to this branch. So the PR's substantive content is already merged and #4555 is superseded.
  </details>

- **[merging would add unrelated files]** `misc/jaekwon/tictac.md` — branch adds two off-topic markdown docs and a 984 KB binary that are not on master.
  <details><summary>details</summary>

  The branch carries `misc/jaekwon/tictac.md` (163 lines), `misc/jaekwon/gnoland-whitepaper.md` (502 lines, first line "DO NOT REMOVE THIS FILE, EVER, FROM THE REPO HISTORY", commit message "DO NOT SHARE, deadman switch"), and `misc/jaekwon/Gyroscopic effect reduces lift force -- more efficient rockets! Also, TicTac UFO! _ r_UFOscience.pdf` (984 KB binary). None exist on `origin/master`; all three were introduced by Jae Kwon commits that ended up on the branch (`dd4de568d`, `9463ea16c`, `a43270e9b`, `6e70cc279`). With the feature already merged, these files are the only net content the PR would add, which is reason to close rather than fix.
  </details>

- **[branch far past clean rebase]** branch HEAD `beada2b29` trails master by 405 commits (the full tm2 store / IAVL line and much more), so there is no light-touch path to make this branch's net diff empty; closing is cleaner than rebasing a superseded RFC.

## Open questions
The shortening shipped via #4553 carries three behaviors worth a separate look against master (not posted to this PR, which is closing):
- Project-prefix match uses `strings.HasPrefix(path, buildDir)` with no separator boundary, so a sibling directory whose name starts with the build-dir name (e.g. `gno` vs `gno-simple-e2e`) yields `gno//<full-abs-path>` and leaks the path it meant to hide. Confirmed behaviorally on beada2b29 (`stripBuildDir("/home/u/gno-simple-e2e/...")` → `"gno//home/u/gno-simple-e2e/..."`).
- The `gno/` project-prefix rewrite depends on `$GOMOD`, which the `go` command sets but a plain compiled binary does not have, and on the disk-walk fallback finding the build's source tree on the runtime host. A normally deployed `gnoland` sees `GOMOD=""` and no source tree, so project-file paths print unchanged there; the metadata-leak reduction is effectively dev-only (the `mod/`/`toolchain/` rewrites still work).
- The merged unit test resets `buildDir`/`buildDirOnce` by direct assignment; it is race-free only because it omits `t.Parallel()` while the package's other tests run parallel and read those globals through `getBuildDir`. `go test -race` reports a `DATA RACE` once the reset runs in parallel with stacktrace rendering.

These are all properties of code already on master and would be a follow-up PR against master, so they stay out of comment.md.
