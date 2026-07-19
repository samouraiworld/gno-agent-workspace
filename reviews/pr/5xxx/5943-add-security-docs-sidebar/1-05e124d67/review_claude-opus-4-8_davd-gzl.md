# PR [#5943](https://github.com/gnolang/gno/pull/5943): docs: add gno-interrealm-v2, gno-security, gno-security-guide to sidebar

URL: https://github.com/gnolang/gno/pull/5943
Author: zeycan1 | Base: master | Files: 1 | +3 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 05e124d67 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5943 05e124d67`

**TL;DR:** Three reference pages live under `docs/resources/` but never appear in the documentation site's left-hand navigation. This PR adds them to the navigation file so readers can find them.

**Verdict: REQUEST CHANGES** — the three entries are added to a generated file instead of its source, so the next `make -C docs generate` deletes them and merging the PR does not retrigger the docs site build (1 Warning, 1 Suggestion).

## Summary

[`misc/docs/sidebar.json`](https://github.com/gnolang/gno/blob/05e124d67/misc/docs/sidebar.json) · [↗](../../../../../.worktrees/gno-review-5943/misc/docs/sidebar.json) is a build artifact, not a hand-maintained file: [`docs/Makefile:6`](https://github.com/gnolang/gno/blob/05e124d67/docs/Makefile#L6) · [↗](../../../../../.worktrees/gno-review-5943/docs/Makefile#L6) regenerates it wholesale from `docs/README.md` via the [indexparser](https://github.com/gnolang/gno/blob/05e124d67/misc/docs/tools/indexparser/main.go#L87-L112) · [↗](../../../../../.worktrees/gno-review-5943/misc/docs/tools/indexparser/main.go#L87-L112), which walks the README's `##` sections and emits one sidebar entry per markdown link. The PR edits only the artifact. `docs/README.md` still stops at [`gno-interrealm` then jumps to `gno-memory-model`](https://github.com/gnolang/gno/blob/05e124d67/docs/README.md?plain=1#L52-L53) · [↗](../../../../../.worktrees/gno-review-5943/docs/README.md#L52-L53), so regeneration restores the pre-PR file byte for byte and the published docs index page keeps no link to the three pages either. Every earlier sidebar change in the file's history (`1822034dd`, `4c9de5225`, `7922f54af`, `a28468f28`) edited `docs/README.md` in the same commit.

The intent is right: `gno-interrealm-v2.md`, `gno-security.md`, and `gno-security-guide.md` all exist, are cross-linked to each other, and are the pages `AGENTS.md` and `gno-interrealm.md` point realm authors at. Only one inbound link reaches any of them from a navigable page today, at [`docs/builders/getting-started.md:110`](https://github.com/gnolang/gno/blob/05e124d67/docs/builders/getting-started.md?plain=1#L110) · [↗](../../../../../.worktrees/gno-review-5943/docs/builders/getting-started.md#L110). Adding the three list items to the README's References section fixes discoverability, survives regeneration, gives each page the one-line description the index page shows, and puts the change under `docs/` where the deploy workflow's path filter can see it.

## Fix

Add three list items to the References section of [`docs/README.md`](https://github.com/gnolang/gno/blob/05e124d67/docs/README.md?plain=1#L36) · [↗](../../../../../.worktrees/gno-review-5943/docs/README.md#L36), in the same `- [Label](resources/<file>.md) - <description>` shape as its neighbours, placed after the existing `gno-interrealm` entry. Then run `make -C docs generate` and commit the regenerated `misc/docs/sidebar.json`; the three sidebar lines this PR already contains are exactly what the generator emits, so the artifact diff stays identical.

## Critical (must fix)
None.

## Warnings (should fix)

- **[change disappears on the next docs regeneration]** [`misc/docs/sidebar.json:48-50`](https://github.com/gnolang/gno/blob/05e124d67/misc/docs/sidebar.json#L48-L50) · [↗](../../../../../.worktrees/gno-review-5943/misc/docs/sidebar.json#L48-L50) — the three entries are added to a generated file whose source, `docs/README.md`, is unchanged, so `make -C docs generate` deletes them again.
  <details><summary>details</summary>

  [`docs/Makefile:6`](https://github.com/gnolang/gno/blob/05e124d67/docs/Makefile#L6) · [↗](../../../../../.worktrees/gno-review-5943/docs/Makefile#L6) rebuilds `misc/docs/sidebar.json` from `docs/README.md` alone, and nothing in `.github/workflows/` regenerates or diffs the artifact, so the drift is silent until the next contributor runs the target. Regenerating at this head restores the pre-PR blob exactly, `053739fbb`, the same blob the PR diff shows as its `index` base ([repro](comment_claude-opus-4-8.md)). Two further consequences follow from the same missing README edit: the published docs index page still lists no link to the three pages, and [`deploy-docs.yml:8-9`](https://github.com/gnolang/gno/blob/05e124d67/.github/workflows/deploy-docs.yml#L8-L9) · [↗](../../../../../.worktrees/gno-review-5943/.github/workflows/deploy-docs.yml#L8-L9) fires the Netlify build only for pushes touching `docs/**`, which `misc/docs/sidebar.json` does not match, so merging this PR alone leaves the site on its previous navigation until an unrelated `docs/` change lands. Fix: add the three entries to the References list in `docs/README.md` and commit the regenerated sidebar alongside them.
  </details>

## Nits
None.

## Missing Tests
None. No test asserts that `misc/docs/sidebar.json` matches what the indexparser emits from `docs/README.md`; `make -C docs test` is a no-op ([`docs/Makefile:8-9`](https://github.com/gnolang/gno/blob/05e124d67/docs/Makefile#L8-L9) · [↗](../../../../../.worktrees/gno-review-5943/docs/Makefile#L8-L9)). Adding that guard is out of scope here and is filed under Open questions.

## Suggestions

- **[newly published page points at a filename that does not exist]** [`docs/resources/gno-security.md:5`](https://github.com/gnolang/gno/blob/05e124d67/docs/resources/gno-security.md?plain=1#L5) · [↗](../../../../../.worktrees/gno-review-5943/docs/resources/gno-security.md#L5) — the page names its companion `SECURITY_GUIDE.md` twice, but the repository has no such file; the companion is `gno-security-guide.md`.
  <details><summary>details</summary>

  The stale name appears at [line 5](https://github.com/gnolang/gno/blob/05e124d67/docs/resources/gno-security.md?plain=1#L5) · [↗](../../../../../.worktrees/gno-review-5943/docs/resources/gno-security.md#L5) and [line 45](https://github.com/gnolang/gno/blob/05e124d67/docs/resources/gno-security.md?plain=1#L45) · [↗](../../../../../.worktrees/gno-review-5943/docs/resources/gno-security.md#L45). `grep -rn SECURITY_GUIDE` over the worktree returns only those two lines, so the name resolves to nothing. It predates this PR and sits outside the diff, but this PR is what puts the page in front of readers, and the sibling page it should point at is one of the three being added. Fix: rewrite both mentions as a relative link to `gno-security-guide.md`, matching how `gno-interrealm-v2.md` links it.
  </details>

## Verified

- Regenerating the sidebar at the PR head drops exactly the three added lines and reproduces the pre-PR blob `053739fbb`: ran `go run -C misc/docs/tools/indexparser . -path "$PWD/docs/README.md" > misc/docs/sidebar.json` in the worktree, then `git diff misc/docs/sidebar.json`, which reports `3 deletions(-)` and an `index 8a5b7e4da..053739fbb` header matching the PR diff's own base blob.
- The three sidebar ids resolve to real files and no other sidebar id dangles: comparing every `resources/*` id in [`misc/docs/sidebar.json`](https://github.com/gnolang/gno/blob/05e124d67/misc/docs/sidebar.json) · [↗](../../../../../.worktrees/gno-review-5943/misc/docs/sidebar.json) against `ls docs/resources/*.md` leaves nothing sidebar-only, and leaves `resources/gno-ai-contract-review` and `resources/test-halt-height` as the only files still unlisted.
- `make -C docs lint` at the PR head reports `Lint complete, no issues found.`, so the three pages carry no broken relative links the docs linter checks; the `SECURITY_GUIDE.md` mentions are bare code spans, not markdown links, which is why the linter does not see them.
- The head commit `05e124d67` is a clean merge of master into the branch: `git show 05e124d67 --cc` prints no hunks, so it carries no conflict-resolution content of its own.

## Open questions

- `docs/resources/gno-ai-contract-review.md` and `docs/resources/test-halt-height.md` are also absent from both `docs/README.md` and the sidebar. The first is the AI-agent counterpart to `gno-security-guide.md` and is a plausible fourth entry; the second reads as an internal manual test plan that probably should stay unlisted. Not posted: deciding what belongs in public navigation is a maintainer call, not a defect in this diff.
- Nothing enforces that `misc/docs/sidebar.json` matches the indexparser output for `docs/README.md`. A CI step running the generate target and failing on a dirty tree would have caught this PR's shape at submission time. Not posted: it is a separate change to the docs tooling, outside this PR's one-file scope.
- davd-gzl already submitted an APPROVED review on this exact commit ([pullrequestreview-4683747087](https://github.com/gnolang/gno/pull/5943#pullrequestreview-4683747087), 2026-07-13). Posting this round as REQUEST_CHANGES supersedes that approval on the same head. Confirm before posting.
