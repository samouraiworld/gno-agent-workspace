# Batch status — review all (started 2026-08-01)

Model claude-opus-5, reviewer davd-gzl. Normal (non-deep) mode.

Synced head `30bdd39` (`samouraiworld/main`) before building the set. The working tree first read the
set from the parent repo's recorded gitlink, `db4e141`, one commit behind that head, which hid the
already-reviewed 6025; the set below was rebuilt after checking out `main`.

gno master at dispatch: `d1a33f574`.

## External-contribution safety gate

Not applicable. All four PRs come from `MEMBER` accounts (jinoosss, notJoon, Villaquiranm); no
`FIRST_TIME_CONTRIBUTOR` in the set. 6022 was the one such PR and the user excluded it.

## Dropped

| Reason | PRs |
|---|---|
| already reviewed (present in `reviews/pr/`) | 6025 and the rest of the open non-draft set |
| excluded by the user | 6022 |
| dependabot | 6021, 6008, 5992, 5990, 5989 |
| WIP-titled | 5922, 5263, 5223, 4949 |
| authored by reviewer (davd-gzl) | 6006, 5993, 5950, 5936, 5934 |

None of the four in scope carries a prior review or review comment from `davd-gzl` on GitHub.

## Final set (4)

All four are first rounds. No head-unchanged, already-APPROVED, or patch-id gate applied.

| PR | Head sha | Author | Size | Worktree | Review dir | Mode |
|---|---|---|---|---|---|---|
| [6029](https://github.com/gnolang/gno/pull/6029) | `3b5b4a701` | jinoosss | +4282-1700, 46f | `.worktrees/gno-review-6029` | `reviews/pr/6xxx/6029-grc721-token-ledger-teller/1-3b5b4a701/` | normal |
| [6028](https://github.com/gnolang/gno/pull/6028) | `37182a315` | jinoosss | +342-145, 26f | `.worktrees/gno-review-6028` | `reviews/pr/6xxx/6028-registry-owned-id-generator/1-37182a315/` | normal |
| [6027](https://github.com/gnolang/gno/pull/6027) | `854b03529` | notJoon | +221-104, 12f | `.worktrees/gno-review-6027` | `reviews/pr/6xxx/6027-slug-alias-registrations/1-854b03529/` | normal |
| [6020](https://github.com/gnolang/gno/pull/6020) | `764ac4d84` | Villaquiranm | +1447-48, 21f | `.worktrees/gno-review-6020` | `reviews/pr/6xxx/6020-compute-map-keys-once/1-764ac4d84/` | normal |

6029, 6028 and 6027 all touch the token standards. 6028 and 6027 both change how a registration is
keyed in `grc20reg`, so they are likely to collide; read the pair together when synthesizing.

## Dispatch

One `general-purpose` agent per PR, all in one message. The parent created every worktree and
checked out every PR head; subagents never run `worktree add`, `gh pr checkout`, or any branch
switch. Subagents write `review_claude-opus-5_davd-gzl.md` and `comment_claude-opus-5.md`, and do
not commit, push, regenerate indexes, or post.

Environment: no Go toolchain on `PATH`. go1.25.9 lives at `/tmp/go/bin/go`; agents export
`PATH=/tmp/go/bin:$PATH` before running any suite.

## Progress

Dispatched; awaiting agent returns.

| PR | Verdict | Findings |
|---|---|---|
| 6029 | | |
| 6028 | | |
| 6027 | | |
| 6020 | | |

## Finalize

1. Parent commits once: `review: batch of 4 open PRs (6029, 6028, 6027, 6020)`.
2. Push to `review/pr-5999-r2`, the branch this turn started on; it already carries PR 7. No second
   PR on this repo.
3. Nothing reaches GitHub without the literal `post`.

## Carried from the 2026-07-29 batch

- [6002](https://github.com/gnolang/gno/pull/6002) draft verdict is APPROVE and still needs human
  confirmation before posting with `--approve`.
- [5991](https://github.com/gnolang/gno/pull/5991) draft verdict is APPROVE and still needs human
  confirmation before posting with `--approve`.
