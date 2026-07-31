---
name: issue
description: Draft a GitHub issue for a problem found by a review. Use when a fix is being prepared and no upstream issue covers the problem. Produces issue.md in the fix directory, posted only after user approval.
---

# Issue

An issue states a problem. A pull request states a change. Keep them apart: the issue never describes the fix, the pull request never re-argues the problem, and neither restates the other. When both exist the pull request references the issue.

Prose follows `skills/writing-style.md`; the rules below are the issue deltas.

## When to draft one

Only when no upstream issue already covers the problem, and only when the problem outlives the pull request that addresses it. A change that closes its own problem completely needs no issue.

Search before drafting, both ways, since titles rarely contain the words you expect:

```bash
gh issue list -R <repo> --state all --limit 100 --json number,title --jq '.[]|"\(.number)\t\(.title)"'
gh api "search/issues?q=repo:<repo>+is:issue+<term>+OR+<term>" --jq '.total_count'
```

Record what the search covered in the draft's `Status:` line. A later reader has to know the absence was checked rather than assumed.

## File

`issue.md` in the PR review directory. Header block of `Target:` (the opened issue URL, else `https://github.com/<repo>/issues/new`) and `Status:`, then `## Title` and `## Body`. Only Title and Body get pasted. No narration of how the draft was written.

## Title

Measure the repo's own issue titles first; they usually do not follow its commit convention. `gnolang/gno` commits are `<type>(<scope>): <subject>`, but its issues are plain sentences; measure before assuming either. Applying the commit convention to an issue title is the common mistake.

```bash
gh issue list -R <repo> --state all --limit 25 --json title --jq '.[].title'
```

Name the observable problem, not the diagnosis: "SonarCloud quality gate has been failing on `main`", not "projectVersion is unset".

## Body

- Open with what a maintainer sees, and where. Then how long it has been true, if that is knowable.
- Give the shape before the detail: a diagram when several parts fail differently, per the Diagrams section of `skills/pr-body.md`.
- One short paragraph per part of the problem, each saying what that part actually needs: a diff, a settings change, a decision, a permission nobody in the thread holds. This is the section a maintainer acts on.
- Say which parts are already addressed and where, once a pull request exists. One clause, not a summary of it.
- End with the root cause when it is checkable and cheap to state, and name the evidence. A cause with a command behind it turns the issue from a report into a fix.
- Never propose the implementation. Naming what a part needs is not the same as designing it.

Length follows the repo. Measure it before drafting.

## Posting

Never without the literal `post` in the current turn, same gate as everything else. `gh issue create -R <repo> --title "<title>" --body-file <path>`, body only, header block excluded.

Labels: check whether the repo actually uses them before setting any. Some repos label heavily and some not at all.

After posting, write the issue URL into the draft's `Target:` line, add `Fixes #<n>` or `Refs #<n>` to `pr-body.md` where it applies, and commit both.
