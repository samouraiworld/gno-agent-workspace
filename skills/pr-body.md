---
name: pr-body
description: Write the title and body of a pull request. Runs the core pr-body skill with the gno deltas below.
---

# PR body

The rules are `skills/core/pr-body.md`. Gno deltas:

- `pr-body.md` sits in the round directory,
  `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/`,
  not in a `projects/<repo>/changes/` tree, and no `post-fix.sh` reads it: the
  user opens the PR from the compare link.
- Visual-evidence files go under that round directory's `media/`.
