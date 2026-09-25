# Batch status: recent external contributions (started 2026-09-25)

Model claude-opus-5-5, reviewer davd-gzl. Quick word, solo shape: one workflow per PR, two agents
(a finder, then a judge and writer) on the three code fixes, one agent on the two small ones.

Synced head `e9f672f` (`origin/main`), `0 0` against `HEAD`. Every agent runs the pinned Go 1.25.9
from `.worktrees/toolchain/go1.25.9` at the workspace root.

## Scope

Open PRs created 2026-08-26 or later: 50. After the filters, 32 were unreviewed, 5 of them by an
author outside the gno team (`author_association` other than `MEMBER`). The set is those **5**.

| PR | Author | Head | Base | Agents | Round directory |
|---|---|---|---|---|---|
| [6107](https://github.com/gnolang/gno/pull/6107) | crazywriter1, contributor | `98ba8a8e4` | `26c0a7b32` | 2 | `pr/6xxx/6107-blocknode-location-else-if/1-98ba8a8e4` |
| [6129](https://github.com/gnolang/gno/pull/6129) | Fhatu12, first-time | `51b1e76fb` | `1fc4c140e` | 2 | `pr/6xxx/6129-package-init-panic/1-51b1e76fb` |
| [6152](https://github.com/gnolang/gno/pull/6152) | wwqiu, contributor | `a74212699` | `7916d1dd6` | 2 | `pr/6xxx/6152-amino-deepcopy-slices/1-a74212699` |
| [6126](https://github.com/gnolang/gno/pull/6126) | Fhatu12, first-time | `a27598b28` | `2ed70a202` | 1 | `pr/6xxx/6126-gnoweb-csp-help-remote/1-a27598b28` |
| [6108](https://github.com/gnolang/gno/pull/6108) | crazywriter1, contributor | `b512ada64` | `26c0a7b32` | 1 | `pr/6xxx/6108-dot-import-ban-docs/1-b512ada64` |

## Dropped

| Reason | Count |
|---|---|
| Author is a gno team member | 27 |
| Draft | 6 |
| Bot | 3 |
| Reviewer's own PR | 2 |
| Already in `reviews/pr/` | the rest of the 50 |

## Safety pass, before any checkout

- Code: every diff read whole. None touches CI, `go.mod`, a Makefile, a container file or a
  script; no added line executes, reaches the network, reads the environment or writes the
  filesystem; no `init`, `TestMain`, `go:generate` or build tag; no bidirectional or zero-width
  character. The one non-ASCII added character is an em dash in a 6107 test comment.
- Text the agents read: every PR body, comment, review, review comment and commit message, and the
  linked issues 6065, 6051, 6121 and 6076, read whole. No text addressed to an AI, no HTML
  comment, no hidden character. The non-ASCII is the Gno2D2 bot's status table.

## Resume

Each round's inputs sit in `.worktrees/rounds/gno-<n>/` at the workspace root, `args.json`
included; re-launch `scripts/workflows/review-pipeline.js` with it, or `resumeFromRunId` for a
round that died.

## Runs

| PR | Workflow run | State |
|---|---|---|
| 6107 | `wf_83c80f11-1e5` | done: APPROVE, three Suggestions to remove code that cannot fire |
| 6129 | `wf_76033866-f10` | done: REQUEST CHANGES, one Warning on the REPL, one Nit on the results hash |
| 6152 | `wf_705d4ae0-b51` | done: APPROVE, two Suggestions predating the branch |
| 6126 | `wf_410cefc1-822` | done: REQUEST CHANGES, one Warning on the test |
| 6108 | `wf_3b0dc6e1-b87` | done: APPROVE, one Nit |

The laptop shutdown killed 6107, 6129 and 6152 after their finders; each resumed in the same
session, the finder replayed from cache and the judge rerun from a cleared round directory.
