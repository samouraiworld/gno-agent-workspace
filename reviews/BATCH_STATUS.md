# Batch status — review all, recent (started 2026-09-04)

Model claude-opus-5, reviewer davd-gzl. Normal mode for nine PRs, one subagent per PR via parallel
dispatch.

Synced head `d4800b6` (`origin/main`, `samouraiworld`), `0 0` against `HEAD`, so `reviews/pr/` is
current. gno master at dispatch: `bdeccddf6`.

Go 1.25.9 is on this box already and answers `go version`; every dispatched agent exports
`PATH=$HOME/bin:/tmp/go1259/go/bin:$PATH` before running a test. `/tmp` is wiped between sessions,
so an agent that skips the export reports a failed suite it never ran.

## Scope

151 open non-draft PRs; 34 absent from `reviews/pr/`. The filters below left 15, and the user cut
the set to the nine most recent, opened 2026-08-31 or later. The final set is **9**.

## Dropped

| Reason | PRs |
|---|---|
| Older than the recency cut | [6084](https://github.com/gnolang/gno/pull/6084), [6083](https://github.com/gnolang/gno/pull/6083), [6081](https://github.com/gnolang/gno/pull/6081), [6060](https://github.com/gnolang/gno/pull/6060), [5887](https://github.com/gnolang/gno/pull/5887), [5886](https://github.com/gnolang/gno/pull/5886) |
| Already reviewed on GitHub, absent from `reviews/pr/` | [6109](https://github.com/gnolang/gno/pull/6109) APPROVED, [6099](https://github.com/gnolang/gno/pull/6099), [6022](https://github.com/gnolang/gno/pull/6022) CHANGES_REQUESTED, [6006](https://github.com/gnolang/gno/pull/6006), [5993](https://github.com/gnolang/gno/pull/5993), [5934](https://github.com/gnolang/gno/pull/5934) |
| Author outside the core team | [6129](https://github.com/gnolang/gno/pull/6129), [6126](https://github.com/gnolang/gno/pull/6126) first-time; [6108](https://github.com/gnolang/gno/pull/6108), [6107](https://github.com/gnolang/gno/pull/6107), [6048](https://github.com/gnolang/gno/pull/6048) contributor |
| `WIP` title | [5922](https://github.com/gnolang/gno/pull/5922), [5263](https://github.com/gnolang/gno/pull/5263), [5223](https://github.com/gnolang/gno/pull/5223), [4949](https://github.com/gnolang/gno/pull/4949) |
| Dependabot | [6127](https://github.com/gnolang/gno/pull/6127), [6117](https://github.com/gnolang/gno/pull/6117), [6080](https://github.com/gnolang/gno/pull/6080), [6008](https://github.com/gnolang/gno/pull/6008), [5989](https://github.com/gnolang/gno/pull/5989) |
| Reviewer's own PR | [5936](https://github.com/gnolang/gno/pull/5936) |

No outside-contributor PR entered the set, so no static danger pass was owed.

## The set

| PR | Head sha | Prior round | Worktree | Review dir |
|---|---|---|---|---|
| [6135](https://github.com/gnolang/gno/pull/6135) | `ddc5acfb9` | none | `.worktrees/gno-review-6135` | `reviews/pr/6xxx/6135-gnoe2e-txtar-harness/1-ddc5acfb9/` |
| [6132](https://github.com/gnolang/gno/pull/6132) | `93a0bea09` | none | `.worktrees/gno-review-6132` | `reviews/pr/6xxx/6132-boards2-opendiscussions-board/1-93a0bea09/` |
| [6131](https://github.com/gnolang/gno/pull/6131) | `f2bdb07b0` | none | `.worktrees/gno-review-6131` | `reviews/pr/6xxx/6131-govdao-t1-multisig-address/1-f2bdb07b0/` |
| [6123](https://github.com/gnolang/gno/pull/6123) | `e014175d2` | none | `.worktrees/gno-review-6123` | `reviews/pr/6xxx/6123-grc20-eoa-writes/1-e014175d2/` |
| [6120](https://github.com/gnolang/gno/pull/6120) | `23e9de5ad` | none | `.worktrees/gno-review-6120` | `reviews/pr/6xxx/6120-bank-transfer-events/1-23e9de5ad/` |
| [6115](https://github.com/gnolang/gno/pull/6115) | `c7ac45512` | none | `.worktrees/gno-review-6115` | `reviews/pr/6xxx/6115-retry-startup-queries/1-c7ac45512/` |
| [6112](https://github.com/gnolang/gno/pull/6112) | `6dcb5272a` | none | `.worktrees/gno-review-6112` | `reviews/pr/6xxx/6112-parked-redeploy-not-active/1-6dcb5272a/` |
| [6111](https://github.com/gnolang/gno/pull/6111) | `b51a78a8a` | none | `.worktrees/gno-review-6111` | `reviews/pr/6xxx/6111-debit-max-spend-at-broadcast/1-b51a78a8a/` |
| [6101](https://github.com/gnolang/gno/pull/6101) | `911e1a57a` | none | `.worktrees/gno-review-6101` | `reviews/pr/6xxx/6101-realm-scoped-token-ids/1-911e1a57a/` |

## Resume

An agent that dies leaves its worktree as the only record. To resume one PR: read
`git -C .worktrees/gno-review-<n> status --short` for leftover instrumentation, restore tracked
files, delete `zz_*` after reading them, then re-run `skills/review.md` for that number against the
existing worktree.

## Results

| PR | Verdict | Findings | Round |
|---|---|---|---|
| [6135](https://github.com/gnolang/gno/pull/6135) | APPROVE | 3 nits | `6135-gnoe2e-txtar-harness/1-ddc5acfb9/` |
| [6132](https://github.com/gnolang/gno/pull/6132) | REQUEST CHANGES | 1 warning, 1 suggestion, 1 nit | `6132-boards2-opendiscussions-board/1-93a0bea09/` |
| [6131](https://github.com/gnolang/gno/pull/6131) | REQUEST CHANGES | 1 warning, 1 suggestion | `6131-govdao-t1-multisig-address/1-f2bdb07b0/` |
| [6120](https://github.com/gnolang/gno/pull/6120) | REQUEST CHANGES | 2 warnings, 1 suggestion, 1 nit | `6120-bank-transfer-events/1-23e9de5ad/` |
| [6123](https://github.com/gnolang/gno/pull/6123) | REQUEST CHANGES | 3 warnings, 1 missing test, 1 suggestion | `6123-grc20-eoa-writes/1-e014175d2/` |
| [6101](https://github.com/gnolang/gno/pull/6101) | REQUEST CHANGES | 3 warnings, 1 missing test, 2 suggestions, 2 nits | `6101-realm-scoped-token-ids/1-911e1a57a/` |
| [6115](https://github.com/gnolang/gno/pull/6115) | REQUEST CHANGES | 1 warning, 2 suggestions, 3 nits | `6115-retry-startup-queries/1-c7ac45512/` |
| [6112](https://github.com/gnolang/gno/pull/6112) | APPROVE | 1 nit | `6112-parked-redeploy-not-active/1-6dcb5272a/` |
| [6111](https://github.com/gnolang/gno/pull/6111) | APPROVE | 1 missing test, 2 nits, 1 suggestion | `6111-debit-max-spend-at-broadcast/1-b51a78a8a/` |
