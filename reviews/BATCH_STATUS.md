# Batch status — review all (started 2026-08-12)

Model claude-opus-5, reviewer davd-gzl. Normal mode for nine PRs, deep mode for
[6062](https://github.com/gnolang/gno/pull/6062) and [5814](https://github.com/gnolang/gno/pull/5814),
one subagent per PR via parallel dispatch.

Synced head `3b9b5c3` (`origin/main`, `samouraiworld`), `0 0` against `HEAD`, so `reviews/pr/` is
current. gno master at dispatch: `754780601`.

## Scope

145 open non-draft PRs; 26 absent from `reviews/pr/`. After dropping dependabot, `WIP`-titled,
reviewer-authored and already-reviewed-on-GitHub PRs, the final set is **11**.

## External-contribution safety gate

Two PRs are `FIRST_TIME_CONTRIBUTOR`: 6057 (crazywriter1) and 6048 (crazywriter1). Static danger
pass run on both raw diffs, nothing executed. Neither touches `.github/workflows`, the Makefile,
`go.mod`, `go.sum`, `package.json`, a Dockerfile or any shell script. No added line calls
`os/exec`, `net/http`, `net.Dial`, `syscall`, `go:generate`, `go:embed`, `unsafe`, a base64 or hex
decode, an environment read or a filesystem write. No non-ASCII character in any added line, so no
bidirectional override, zero-width character or homoglyph. Both clear; no risk carried into either
review. The other nine authors are `MEMBER`.

## Dropped

| Reason | PRs |
|---|---|
| already reviewed (present in `reviews/pr/`) | the other 119 open non-draft PRs |
| already reviewed on GitHub (`CHANGES_REQUESTED` by davd-gzl, no review dir) | 6022 |
| dependabot | 6050, 6047, 6008, 5992, 5989 |
| WIP-titled | 5922, 5263, 5223, 4949 |
| authored by the reviewer | 6006, 5993, 5936, 5934 |

## Final set (11)

All eleven are first rounds.

| PR | Head sha | Author | Size | Mode | Worktree | Review dir |
|---|---|---|---|---|---|---|
| [6062](https://github.com/gnolang/gno/pull/6062) | `f6dd8ad37` | jaekwon | +1318-66, 34f | deep | `.worktrees/gno-review-6062` | `reviews/pr/6xxx/6062-coins-lost-send-envelope/1-f6dd8ad37/` |
| [6061](https://github.com/gnolang/gno/pull/6061) | `a4d60893c` | thehowl | +758-5, 11f | normal | `.worktrees/gno-review-6061` | `reviews/pr/6xxx/6061-bound-test-memory/1-a4d60893c/` |
| [6060](https://github.com/gnolang/gno/pull/6060) | `6830e2549` | thehowl | +266-8, 8f | normal | `.worktrees/gno-review-6060` | `reviews/pr/6xxx/6060-type-decls-nested-blocks/1-6830e2549/` |
| [6058](https://github.com/gnolang/gno/pull/6058) | `391840aa7` | thehowl | +615-76, 17f | normal | `.worktrees/gno-review-6058` | `reviews/pr/6xxx/6058-switch-case-shadowing/1-391840aa7/` |
| [6057](https://github.com/gnolang/gno/pull/6057) | `c5485574b` | crazywriter1 | +34-0, 4f | normal | `.worktrees/gno-review-6057` | `reviews/pr/6xxx/6057-reject-tilde-operator/1-c5485574b/` |
| [6056](https://github.com/gnolang/gno/pull/6056) | `411cbd37c` | thehowl | +131-0, 6f | normal | `.worktrees/gno-review-6056` | `reviews/pr/6xxx/6056-fallthrough-clause-names/1-411cbd37c/` |
| [6054](https://github.com/gnolang/gno/pull/6054) | `96e3f353b` | tbruyelle | +190-36, 2f | normal | `.worktrees/gno-review-6054` | `reviews/pr/6xxx/6054-dial-loop-spinning/1-96e3f353b/` |
| [6053](https://github.com/gnolang/gno/pull/6053) | `73487e2ad` | tbruyelle | +2-2, 1f | normal | `.worktrees/gno-review-6053` | `reviews/pr/6xxx/6053-persistent-peer-cap-claim/1-73487e2ad/` |
| [6048](https://github.com/gnolang/gno/pull/6048) | `fb271b82e` | crazywriter1 | +441-22, 10f | normal | `.worktrees/gno-review-6048` | `reviews/pr/6xxx/6048-cyclic-type-alias/1-fb271b82e/` |
| [6035](https://github.com/gnolang/gno/pull/6035) | `407bf9166` | jefft0 | +445-72, 18f | normal | `.worktrees/gno-review-6035` | `reviews/pr/6xxx/6035-playground-dry-run/1-407bf9166/` |
| [5814](https://github.com/gnolang/gno/pull/5814) | `e5ed12eec` | thehowl | +397-3, 5f | deep | `.worktrees/gno-review-5814` | `reviews/pr/5xxx/5814-share-interface-held-values/1-e5ed12eec/` |

## Why 6062 and 5814 are deep

6062 changes who is authorized to move coins and where a send envelope is charged, so a missed case
is a loss of funds rather than a wrong result. 5814 stops copying interface-held values when an
array is copied, which turns a value copy into a shared reference wherever the sharing assumption
does not hold. Both earn the lens dispatch, the critic pass and the claim-verification gate in the
*Deep mode* section of `skills/review.md`.

## Coupling

6060, 6058 and 6056 all change how the preprocessor scopes names inside a block, and 6048 changes
cycle handling in the same file (`gnovm/pkg/gnolang/preprocess.go`). Reconcile their findings
against each other before the batch commit: a rule one PR states and another contradicts is one
conclusion re-derived from the source, written into every affected review file.

## Progress

| PR | Agent | Review file | comment.md | Final check + QA | Committed |
|---|---|---|---|---|---|
| 6062 | not started | — | — | — | — |
| 6061 | not started | — | — | — | — |
| 6060 | not started | — | — | — | — |
| 6058 | not started | — | — | — | — |
| 6057 | not started | — | — | — | — |
| 6056 | not started | — | — | — | — |
| 6054 | not started | — | — | — | — |
| 6053 | not started | — | — | — | — |
| 6048 | not started | — | — | — | — |
| 6035 | not started | — | — | — | — |
| 5814 | not started | — | — | — | — |

## Resume

Every worktree already exists with its PR checked out, so nothing in the setup has to run again.

1. From `checkout/`, `git fetch origin && git status` and confirm `0 0` against `origin/main`.
2. `git -C gno fetch origin master`, then per PR
   `git -C .worktrees/gno-review-<number> rev-parse --short=9 HEAD` against the Head sha column. A
   moved head means a re-review round, not a first round: run the patch-id gate in
   `skills/core/review.md`.
3. Dispatch only the PRs whose Progress row is `not started`, one `general-purpose` agent each,
   with the prompt in the *Parallel dispatch* section of `skills/review.md`. Every prompt names its
   worktree and forbids `worktree add` and `gh pr checkout`.
4. As each returns, run the *Final check* and both QA agents over its draft, then fill its Progress
   row.
5. Reconcile the coupled set above, then one commit and one push for the whole batch.
