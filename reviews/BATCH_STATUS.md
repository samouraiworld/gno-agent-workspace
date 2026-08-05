# Batch status — review all (started 2026-08-05)

Model claude-opus-5, reviewer davd-gzl. Normal (non-deep) mode, one subagent per PR via parallel
dispatch.

Synced head `74201ec` (`origin/main`, `samouraiworld`), `0 0` against `HEAD`, so `reviews/pr/` is
current. gno master at dispatch: `fb02547fd`.

## Scope

148 open non-draft PRs; 18 absent from `reviews/pr/`. After dropping `WIP`-titled, dependabot,
reviewer-authored and already-reviewed-on-GitHub PRs, the final set is four.

## External-contribution safety gate

No PR in the final set is `FIRST_TIME_CONTRIBUTOR`: 6037 is `MEMBER` (Villaquiranm), 6038, 6039 and
6040 are `MEMBER` (moul). No static danger pass needed.

## Dropped

| Reason | PRs |
|---|---|
| already reviewed (present in `reviews/pr/`) | the other 130 open non-draft PRs |
| already reviewed on GitHub (`CHANGES_REQUESTED` by davd-gzl, no review dir) | 6022 |
| dependabot | 6021, 6008, 5992, 5990, 5989 |
| WIP-titled | 5922, 5263, 5223, 4949 |
| authored by the reviewer | 6006, 5993, 5936, 5934 |

## Final set (4)

All four are first rounds.

| PR | Head sha | Author | Size | Worktree | Review dir |
|---|---|---|---|---|---|
| [6037](https://github.com/gnolang/gno/pull/6037) | `390bffe90` | Villaquiranm | +23-4, 2f | `.worktrees/gno-review-6037` | `reviews/pr/6xxx/6037-map-composite-key-in-loop/1-390bffe90/` |
| [6038](https://github.com/gnolang/gno/pull/6038) | `3ced3553a` | moul | +126-6, 5f | `.worktrees/gno-review-6038` | `reviews/pr/6xxx/6038-running-a-node-page/1-3ced3553a/` |
| [6039](https://github.com/gnolang/gno/pull/6039) | `2c817cec4` | moul | +306-27, 10f | `.worktrees/gno-review-6039` | `reviews/pr/6xxx/6039-move-node-operator-docs/1-2c817cec4/` |
| [6040](https://github.com/gnolang/gno/pull/6040) | `4b05e8faf` | moul | +71-0, 11f | `.worktrees/gno-review-6040` | `reviews/pr/6xxx/6040-skip-heavy-workflows-docs-only/1-4b05e8faf/` |

## Coupling, and how it resolved

6038 and 6039 both restructure node-operator documentation, and 6040 gates CI on documentation-only
paths that both of them create. Three conclusions were re-derived from the diffs rather than taken
from any one agent's summary, and written into every affected review file.

1. **6039 closes 6038.** 6039's branch carries `3ced3553a` as its first commit and its body carries
   `Closes #6038` at line 67, so 6038 does not land under its own number. Re-checked at
   `2c817cec4`: two of 6038's three Warnings survive that head untouched, the release-artifacts
   bullet at `gnoland-networks.md:35-37` and the one-line installer. Both are carried into the 6039
   review and draft as Warnings, and the installer one gained a second anchor, because 6039's own
   diff adds a copy-pasteable `--full` command at `gno.land/cmd/gnoland/README.md:26`. Ran it: exits
   1 with `no v* release found`. 6038's remaining Warning is absorbed by 6039's relative-link
   Warning, since `../../misc/deployments` still resolves to `master`'s stale copy on GitHub.
   Both drafts stay postable; whichever pull request the author acts on, the findings reach them.
2. **6040 does not constrain 6039.** 6040's review left this conditional: if a file 6039 moves out
   of `docs/` carries an `embedmd` directive or a link the docs linter was checking, 6039 must keep
   it under `docs/`. Checked at `2c817cec4`: no `embedmd` in either moved file, `TMKMS.md` carries
   no markdown link, and all seven relative links the rewritten README adds resolve on disk with a
   valid `#deployment-files` anchor. The condition does not bind. What survives is a gap for the
   next edit, recorded in both reviews rather than as a finding on either.
3. **Merge order is free, and `examples/**` stays gated.** No workflow 6040 touches is a required
   check, measured from `repos/gnolang/gno/branches/master` and an empty ruleset, so no order leaves
   a pull request pending. `examples/**` stays outside 6040's documentation-only set, so 6039 keeps
   the filetest run that covers its `init.gno` golden.

No agent reached a conclusion another contradicted.

## Environment

No Go toolchain on `PATH` at session start (`bash: go: command not found`). `gno/go.mod` requires
`go 1.25.9`; fetched the release tarball into `/tmp/go` and exported `PATH=/tmp/go/bin:$PATH` for
every run.

## Results

| PR | Verdict | Findings |
|---|---|---|
| 6037 | APPROVE (awaiting human confirmation) | 1 Warning, 2 Missing tests, 2 Nits, 1 Suggestion |
| 6038 | REQUEST CHANGES | 3 Warnings, 6 Nits, 1 Suggestion |
| 6039 | REQUEST CHANGES | 7 Warnings, 2 Nits, 2 Suggestions (2 Warnings carried from 6038) |
| 6040 | APPROVE (awaiting human confirmation) | 2 Warnings, 2 Nits, 1 Suggestion |

## Resume / finalize

1. Reconcile 6038 / 6039 / 6040 merge-order and documentation-path conclusions.
2. One commit covering all four review dirs plus this file, then push to `origin/main`.
3. Hand over each `comment_claude-opus-5.md` draft. Post only on the literal `post`.
