---
name: issue
description: Draft a GitHub issue for a problem found by a review. Runs the core issue skill with the gno deltas below.
---

# Issue

The rules are `skills/core/issue.md`. Gno deltas:

- `issue.md` sits in the round directory,
  `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/`,
  beside the review file.
- `gnolang/gno` commits are `<type>(<scope>): <subject>`, but its issues are
  plain sentences; measure before assuming either.
- There is no `post-fix.sh` in this workspace: on `post`, run
  `gh issue create -R <repo> --title "<title>" --body-file <path>` yourself,
  body only, header block excluded, then write the URL into `Target:`.
