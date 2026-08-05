# PR [#6040](https://github.com/gnolang/gno/pull/6040): chore(ci): don't run heavy workflows on documentation-only pull requests

URL: https://github.com/gnolang/gno/pull/6040
Author: moul | Base: master | Files: 11 | +71 -0
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 4b05e8faf (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6040 4b05e8faf`

**TL;DR:** Editing a README today can start a fifteen-way validator matrix, two Docker image builds and a live chain. This adds path filters to eleven workflows so a pull request that only touches prose runs none of them, while keeping Markdown that lives inside gno packages on the full suite.

**Verdict: APPROVE** — the design holds and the author's before/after table reproduces exactly; two path lists are one entry short, `.dockerignore` in `ci / e2e` and a third mempackage tree in the `gnovm` re-includes (2 Warnings, 2 Nits, 1 Suggestion).

## Verify first

- [`.github/workflows/ci-e2e.yml:12`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-e2e.yml#L12) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-e2e.yml#L12) — `Dockerfile` is listed, [`.dockerignore:1-12`](https://github.com/gnolang/gno/blob/4b05e8faf/.dockerignore#L1-L12) · [↗](../../../../../.worktrees/gno-review-6040/.dockerignore#L1-L12) is not, and it selects the build context the job builds from. Push a branch that changes only `.dockerignore` and confirm `ci / e2e` is absent from `gh pr checks`.
- Nothing on `master` is a required status check: `gh api repos/gnolang/gno/branches/master --jq '.protection.required_status_checks'` returns `{"checks":[],"contexts":[],"enforcement_level":"non_admins"}` and `gh api 'repos/gnolang/gno/rulesets?includes_parents=true'` returns `[]`. That is the single fact that makes a zero-workflow pull request mergeable rather than permanently pending. Re-read it before merging.
- [`.github/workflows/ci-dir-gnovm.yml:20-21`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-dir-gnovm.yml#L20-L21) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-dir-gnovm.yml#L20-L21) — the re-include names two gno source trees. Run `comm -12 <(git ls-files '*.gno' | xargs -n1 dirname | sort -u) <(git ls-files '*.md' | xargs -n1 dirname | sort -u)` and confirm the trees it prints are the ones listed.

## Summary

Two workflows, [`ci / e2e`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-e2e.yml#L1) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-e2e.yml#L1) and [`deploy / pages`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/deploy-pages.yml#L3) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/deploy-pages.yml#L3), had no `pull_request` path filter at all and so ran on every pull request. The nine dir-scoped workflows filtered on whole trees, and `gno.land/`, `gnovm/` and `tm2/` hold 145 Markdown files outside `gnovm/stdlibs/`, every one of them prose, so a README edit in any of them reached the fifteen-way scenario matrix in [`ci / val-scenarios`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-val-scenarios.yml#L45-L63) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-val-scenarios.yml#L45-L63). The fix gives the two unfiltered workflows a list, and adds `'!**/*.md'` to the rest paired with a re-include of the trees where Markdown is package content rather than prose.

The mechanism is sound. `.md` is a mempackage file extension at [`gnovm/pkg/gnolang/mempackage.go:291-296`](https://github.com/gnolang/gno/blob/4b05e8faf/gnovm/pkg/gnolang/mempackage.go#L291-L296) · [↗](../../../../../.worktrees/gno-review-6040/gnovm/pkg/gnolang/mempackage.go#L291-L296), so a README under `examples/` or `gnovm/stdlibs/` is on-chain content and must keep triggering CI; the ordered re-include restores exactly those. The remaining gaps are additions, not corrections: a third tree with the same property is missing from the re-include, and `.dockerignore` is missing from the `ci / e2e` list.

## Diagram

How one path resolves through [`ci-dir-gnoland.yml:9-20`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-dir-gnoland.yml#L9-L20) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-dir-gnoland.yml#L9-L20). State starts excluded; the last matching pattern wins.

```
pattern list (in order)        gno.land/cmd/gnoland/README.md   examples/nt/avl/README.md
-----------------------------  ------------------------------   -------------------------
gno.land/**                    include                          -
gnovm/**  tm2/**               -                                -
examples/**                    -                                include
go.mod  <self>.yml             -                                -
!**/*.md                       EXCLUDE                          exclude
examples/**/*.md               -                                INCLUDE
gnovm/stdlibs/**/*.md          -                                -
-----------------------------  ------------------------------   -------------------------
final                          does not trigger                 triggers
```

## Examples

Replay of GitHub's documented filter semantics over the tracked tree, before and after the diff. Each row is a whole changeset.

| Changeset | Before | After |
|---|---|---|
| [#6038](https://github.com/gnolang/gno/pull/6038) real file list | codegen-verify, e2e, misc, pages | codegen-verify, misc |
| [#6039](https://github.com/gnolang/gno/pull/6039) real file list | codegen-verify, contribs, e2e, examples, gnoland, gnovm, misc, val-scenarios, pages | codegen-verify, e2e, examples, gnoland, gnovm, misc, val-scenarios |
| `gno.land/cmd/gnoland/TMKMS.md` alone | contribs, e2e, gnoland, val-scenarios, pages | none |
| `tm2/pkg/p2p/README.md` | contribs, e2e, gnoland, gnovm, multiarch, tm2, val-scenarios, pages | none |
| `examples/gno.land/p/nt/avl/v0/README.md` | e2e, examples, gnoland, gnovm, val-scenarios, pages | e2e, examples, gnoland, gnovm, val-scenarios |
| `gnovm/stdlibs/errors/README.md` | contribs, e2e, examples, gnoland, gnovm, multiarch, val-scenarios, pages | unchanged |
| `gnovm/tests/files/extern/redeclaration1/README.md` | contribs, e2e, examples, gnoland, gnovm, multiarch, val-scenarios, pages | none |
| one `.go` under `tm2/` | ten workflows | unchanged |
| `.dockerignore` | e2e, pages | none |
| `.github/workflows/_ci-go.yml` | e2e, pages, actions-lint, dependabot-tidy | actions-lint, dependabot-tidy |

## Warnings (should fix)

- **[the file that decides what Docker sees is no longer watched]** [`.github/workflows/ci-e2e.yml:9-24`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-e2e.yml#L9-L24) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-e2e.yml#L9-L24) — [`Dockerfile`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-e2e.yml#L12) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-e2e.yml#L12) is in the new list and `.dockerignore` is not, so after this change no pull-request workflow runs when `.dockerignore` changes.
  <details><summary>details</summary>

  [`misc/e2e/docker-compose.yml:3-6`](https://github.com/gnolang/gno/blob/4b05e8faf/misc/e2e/docker-compose.yml#L3-L6) · [↗](../../../../../.worktrees/gno-review-6040/misc/e2e/docker-compose.yml#L3-L6) builds with `context: ../..` and `dockerfile: Dockerfile`, and the root [`Dockerfile:13`](https://github.com/gnolang/gno/blob/4b05e8faf/Dockerfile#L13) · [↗](../../../../../.worktrees/gno-review-6040/Dockerfile#L13) does `COPY . ./`, so [`.dockerignore`](https://github.com/gnolang/gno/blob/4b05e8faf/.dockerignore#L1-L12) · [↗](../../../../../.worktrees/gno-review-6040/.dockerignore#L1-L12) alone decides which files reach the image. Adding a directory to it can produce a `gnoland` image with no `examples/`, which the e2e chain would fail on. `ci / e2e` was the only pull-request workflow covering that file, because it had no filter; [`ci / val-scenarios`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-val-scenarios.yml#L79-L110) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-val-scenarios.yml#L79-L110) builds its three images from the same repository-root context and already listed [`Dockerfile`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-val-scenarios.yml#L24) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-val-scenarios.yml#L24) without it. A filter replay over every workflow with a `pull_request` trigger returns no workflow for a `.dockerignore`-only changeset, against `ci / e2e` and `deploy / pages` at the merge base. Fix: list `.dockerignore` beside `Dockerfile` in both workflows.
  </details>

- **[a third tree holds package Markdown and is not re-included]** [`.github/workflows/ci-dir-gnovm.yml:16-21`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-dir-gnovm.yml#L16-L21) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-dir-gnovm.yml#L16-L21) — the comment says `*.md` is prose except where it is mempackage content, but `gnovm/tests/files/extern/*` is a third such tree and only `examples/` and `gnovm/stdlibs/` are re-included.
  <details><summary>details</summary>

  [`gnovm/pkg/test/imports.go:157-160`](https://github.com/gnolang/gno/blob/4b05e8faf/gnovm/pkg/test/imports.go#L157-L160) · [↗](../../../../../.worktrees/gno-review-6040/gnovm/pkg/test/imports.go#L157-L160) reads `gnovm/tests/files/extern/<pkg>` through `MustReadMemPackage`, the same call that reads a stdlib. Running it on `redeclaration1` returns `[README.md redeclaration.gno redeclaration2.gno]`, byte for byte the shape `gnovm/stdlibs/errors` returns; see the [repro](comment_claude-opus-5.md). Nothing observable breaks today, because unknown extensions are dropped at read time in [`mempackage.go:812-818`](https://github.com/gnolang/gno/blob/4b05e8faf/gnovm/pkg/gnolang/mempackage.go#L812-L818) · [↗](../../../../../.worktrees/gno-review-6040/gnovm/pkg/gnolang/mempackage.go#L812-L818) and the three READMEs are never parsed, so the risk is that the rule the comment states already has a counterexample rather than a current failure. The same three trees also apply to `ci-dir-contribs.yml`, `ci-dir-multiarch-determinism.yml` and `ci-e2e.yml`, each of which watches `gnovm/**`. Fix: add `gnovm/tests/**/*.md` to every workflow that already re-includes `gnovm/stdlibs/**/*.md`.
  </details>

## Nits

- **[one of the eleven edits changes nothing today]** [`.github/workflows/ci-tmkms-integration.yml:19-20`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-tmkms-integration.yml#L19-L20) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-tmkms-integration.yml#L19-L20) — the replay moves zero files for this workflow. Only one of its five positive patterns can ever match a `.md` path, [`tm2/pkg/bft/privval/upstream/**`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-tmkms-integration.yml#L14) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-tmkms-integration.yml#L14), and that directory holds no Markdown. The line goes live the first time someone adds a README there and saves a five-minute cargo build, which is a fair reason to keep it. No change needed.

- **[the re-include patterns skip a tree-root file]** [`.github/workflows/ci-dir-examples.yml:19`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-dir-examples.yml#L19) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-dir-examples.yml#L19) — GitHub documents `**` as ["zero or more of any character"](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#filter-pattern-cheat-sheet), so `examples/**/*.md` needs a literal `/` after `examples/` and does not re-include [`examples/README.md`](https://github.com/gnolang/gno/blob/4b05e8faf/examples/README.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6040/examples/README.md#L1); the same holds for `gnovm/stdlibs/README.md`. Both are prose, and a mempackage always sits in a subdirectory, so the result is the same under either reading of `**`. No change needed.

## Suggestions

- **[the shared build definition now runs nothing]** [`.github/workflows/ci-dir-gnovm.yml:15`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-dir-gnovm.yml#L15) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-dir-gnovm.yml#L15) — [`_ci-go.yml`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/_ci-go.yml#L1) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/_ci-go.yml#L1) and [`_ci-gno.yml`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/_ci-gno.yml#L1) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/_ci-gno.yml#L1) define lint, build and test for every module, and no dir workflow lists them.
  <details><summary>details</summary>

  Each dir workflow lists only its own file, for example [`ci-dir-gnovm.yml:15`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-dir-gnovm.yml#L15) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-dir-gnovm.yml#L15). That predates the diff, but until now a pull request touching `_ci-go.yml` still started `ci / e2e` and `deploy / pages`; after the change it starts `actionlint` and nothing else. The root [`Makefile:104`](https://github.com/gnolang/gno/blob/4b05e8faf/Makefile#L104) · [↗](../../../../../.worktrees/gno-review-6040/Makefile#L104) is in the same position: it defines the `tidy` target that [`ci / codegen-verify`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-codegen-verify.yml#L39) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-codegen-verify.yml#L39) runs, and no workflow watches it. A sweep of every path filter is the moment to add them. Fix: list `.github/workflows/_ci-go.yml` in each workflow that calls it, `_ci-gno.yml` in `ci-dir-examples.yml` and `ci-dir-gnovm.yml`, and `Makefile` in `ci-codegen-verify.yml`.
  </details>

## Cross-PR constraint ([#6038](https://github.com/gnolang/gno/pull/6038), [#6039](https://github.com/gnolang/gno/pull/6039))

Reconciliation input for the parent. Measured, not inferred.

- **Merge order is free.** Neither coupled pull request adds a path this one handles wrongly, and no workflow is a required check, so a merge in either order leaves nothing pending. If 6040 lands first, 6039's own run loses `ci / contribs` and `deploy / pages`; they are absent, not pending.
- **6039's destination falls inside the prose globs, correctly.** It creates [`gno.land/cmd/gnoland/TMKMS.md`](https://github.com/gnolang/gno/blob/2c817cec4/gno.land/cmd/gnoland/TMKMS.md?plain=1#L1) and edits [`gno.land/cmd/gnoland/README.md`](https://github.com/gnolang/gno/blob/4b05e8faf/gno.land/cmd/gnoland/README.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6040/gno.land/cmd/gnoland/README.md#L1). Under this diff a changeset holding only those triggers zero workflows, down from `ci / contribs`, `ci / e2e`, `ci / gnoland`, `ci / val-scenarios` and `deploy / pages`. That is the intended outcome: nothing reads those files, and no `//go:embed` in the repository names a `.md`.
- **The one constraint 6039 must satisfy.** Markdown is linted in exactly one place. [`docs/Makefile:1-2`](https://github.com/gnolang/gno/blob/4b05e8faf/docs/Makefile#L1-L2) · [↗](../../../../../.worktrees/gno-review-6040/docs/Makefile#L1-L2) runs the link linter with `-path "$(PWD)"` from `docs/`, and [`docs/Makefile:4-6`](https://github.com/gnolang/gno/blob/4b05e8faf/docs/Makefile#L4-L6) · [↗](../../../../../.worktrees/gno-review-6040/docs/Makefile#L4-L6) runs `embedmd` over `find . -name "*.md"` in the same directory. [`ci / codegen-verify`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/ci-codegen-verify.yml#L69-L87) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/ci-codegen-verify.yml#L69-L87) is the job that runs both, and this diff deliberately leaves its `docs/**` entry alone. So the content 6039 moves out of `docs/` loses link checking and `embedmd` expansion, and after 6040 no other workflow picks it up. If either moved file carries an `embedmd` directive or an external link the linter was checking, 6039 needs to keep it under `docs/` or extend the linter's path. Resolved against 6039's head `2c817cec4`: it does not. No `embedmd` directive appears in either moved file, [`TMKMS.md`](https://github.com/gnolang/gno/blob/2c817cec4/gno.land/cmd/gnoland/TMKMS.md?plain=1#L1) carries no markdown link at all, and the seven relative links the rewritten [`README.md`](https://github.com/gnolang/gno/blob/2c817cec4/gno.land/cmd/gnoland/README.md?plain=1#L1) adds all resolve on disk, anchor included. So this diff costs 6039 nothing today. What it leaves is a gap for the next edit: those seven links land in a tree no job checks, which is the same class as 6039's own dead cross-reference in [`tm2/adr/adr-003-tmkms-compat.md:262`](https://github.com/gnolang/gno/blob/2c817cec4/tm2/adr/adr-003-tmkms-compat.md?plain=1#L262).
- **6038 is unaffected.** Its changeset still reaches `ci / codegen-verify` through `docs/**` and `ci / misc` through `misc/docs/sidebar.json`, which is the check that matters for it.

## Verified

- No workflow this diff touches is a required status check. `gh api repos/gnolang/gno/branches/master --jq '.protection.required_status_checks'` reads `{"checks":[],"contexts":[],"enforcement_level":"non_admins"}` and `gh api 'repos/gnolang/gno/rulesets?includes_parents=true'` reads `[]`, both from a token with `pull` and `triage` only. The `Merge Requirements` status is the github-bot, and its [four automatic checks](https://github.com/gnolang/gno/blob/4b05e8faf/contribs/github-bot/internal/config/config.go#L33-L104) · [↗](../../../../../.worktrees/gno-review-6040/contribs/github-bot/internal/config/config.go#L33-L104) are maintainer-edit, gnoweb codeowner review, a `don't merge` label and initial approval; none of them waits on a workflow. A pull request that starts zero workflows therefore merges rather than hanging.
- Replayed GitHub's documented ordered-pattern semantics over all 8,500 tracked files for every workflow with a `pull_request` trigger, at the merge base and at 4b05e8faf. The before and after sets reproduce the author's table row for row, including `ci / val-scenarios` surviving on 6039 and `ci / contribs` dropping from it. Every file that changes trigger state in the nine dir-scoped workflows is a `.md` file; the count of non-`.md` files that lose a trigger is zero for all nine.
- `MustReadMemPackage` on `gnovm/tests/files/extern/redeclaration1` returns `[README.md redeclaration.gno redeclaration2.gno]`, against `[README.md errors.gno gnomod.toml join.gno wrap.gno]` for `gnovm/stdlibs/errors`. Dropping a stray `NOTES.MD` into that directory and running `TestFiles/redeclaration6.gno` leaves it green, which is what puts the finding at Warning rather than higher.
- No `//go:embed` directive in the repository names a `.md` file, and `git ls-files '*testdata*.md' '*golden*.md'` returns nothing, so the only Markdown a test consumes is mempackage content.

## Not verified

- The before and after trigger sets come from a re-implementation of GitHub's matcher, not from a run. No commit touching only documentation has been pushed to this branch to see which workflows GitHub actually starts, and the branch cannot produce that observation itself: every commit on it edits `.github/workflows/**`, which matches [`meta-actions-lint`](https://github.com/gnolang/gno/blob/4b05e8faf/.github/workflows/meta-actions-lint.yml#L10-L11) · [↗](../../../../../.worktrees/gno-review-6040/.github/workflows/meta-actions-lint.yml#L10-L11) and several others. A one-commit branch off this head touching a single `.md` file under `docs/` would settle it, and is the check a maintainer can run that this review cannot.
- The negation ordering the diff relies on is taken from GitHub's documentation rather than measured. Every `!` pattern here is a trailing override on a `paths` list, which is the documented shape, but a mis-ordered list fails silently: the job simply stops running, and no signal says so.

## Open questions

- The author offers to drop the `push.paths` blocks on `ci / val-scenarios` and `ci / multiarch-determinism` in a follow-up, so every merge to `master` runs the full suite. That is a coverage policy decision for a maintainer, not a defect in this diff. Not posted.
- GitHub stops evaluating a path filter past 3,000 files in the generated diff, per its [own documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#onpushpull_requestpull_request_targetpathspaths-ignore). A negation-based list is more exposed to that cap than a bare positive list, since a re-include can sit past the cut. No gno pull request comes close today. Not posted.
