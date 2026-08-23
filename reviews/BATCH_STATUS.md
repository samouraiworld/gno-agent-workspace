# Batch status — review all (started 2026-08-16)

Model claude-opus-5, reviewer davd-gzl. Normal mode for six PRs, deep mode for
[6068](https://github.com/gnolang/gno/pull/6068), one subagent per PR via parallel dispatch.

Synced head `38fffe9` (`origin/main`, `samouraiworld`), `0 0` against `HEAD`, so `reviews/pr/` is
current. gno master at dispatch: `1b0c2c0bf`.

Go 1.25.9 unpacked at `/tmp/go1259/go`; every dispatched agent exports
`PATH=/tmp/go1259/go/bin:$PATH` before running a test. `/tmp` is wiped between sessions and the box
carries no other Go, so an agent that skips the export reports a failed suite it never ran.

## Scope

141 open non-draft PRs; 22 absent from `reviews/pr/`. After the drops below the final set is **7**.

## Dropped

| Reason | PRs |
|---|---|
| already reviewed (present in `reviews/pr/`) | the other 119 open non-draft PRs |
| already reviewed on GitHub (`CHANGES_REQUESTED` by davd-gzl, no review dir) | 6022 |
| dependabot | 6050, 6047, 5989, 6008, 5992 |
| WIP-titled or draft | 6066, 5922, 5223, 4949, 5263 |
| authored by the reviewer | 6006, 5934, 5936, 5993 |

## External-contribution safety gate

6048 is the only non-`MEMBER` author, `CONTRIBUTOR` now where the 2026-08-12 run measured
`FIRST_TIME_CONTRIBUTOR`. Its head moved since that run's static danger pass, so the pass was
re-run on `57597956b`, nothing executed. The diff touches `gnovm/pkg/gnolang/preprocess.go` and nine
`gnovm/tests/files/*.gno` filetests, and no other surface: no `.github/workflows`, Makefile,
`go.mod`, `go.sum`, `package.json`, Dockerfile or shell script. No added line calls `os/exec`,
`net/http`, `net.Dial`, `syscall`, `go:generate`, `go:embed`, `unsafe`, a base64 or hex decode, an
environment read or a filesystem write. No non-ASCII character in any added line, so no
bidirectional override, zero-width character or homoglyph. Clear; no risk carried into the review.

## Final set (7)

All seven are first rounds. Every worktree exists with its PR head checked out, verified against
`gh api repos/gnolang/gno/pulls/<n> --jq '.head.sha'`.

| PR | Head sha | Author | Size | Mode | Worktree | Review dir |
|---|---|---|---|---|---|---|
| [6068](https://github.com/gnolang/gno/pull/6068) | `304f09a7a` | jaekwon | +1911-48, 26f | deep | `.worktrees/gno-review-6068` | `reviews/pr/6xxx/6068-gov-dao-allowlist-lockdown/1-304f09a7a/` |
| [6067](https://github.com/gnolang/gno/pull/6067) | `019439021` | tbruyelle | +45-0, 1f | normal | `.worktrees/gno-review-6067` | `reviews/pr/6xxx/6067-waitfordialtime-timer-branch/1-019439021/` |
| [6060](https://github.com/gnolang/gno/pull/6060) | `6830e2549` | thehowl | +266-8, 8f | normal | `.worktrees/gno-review-6060` | `reviews/pr/6xxx/6060-type-decls-nested-blocks/1-6830e2549/` |
| [6058](https://github.com/gnolang/gno/pull/6058) | `391840aa7` | thehowl | +615-76, 17f | normal | `.worktrees/gno-review-6058` | `reviews/pr/6xxx/6058-switch-case-shadowing/1-391840aa7/` |
| [6056](https://github.com/gnolang/gno/pull/6056) | `411cbd37c` | thehowl | +131-0, 6f | normal | `.worktrees/gno-review-6056` | `reviews/pr/6xxx/6056-fallthrough-clause-names/1-411cbd37c/` |
| [6048](https://github.com/gnolang/gno/pull/6048) | `57597956b` | crazywriter1 | +441-22, 10f | normal | `.worktrees/gno-review-6048` | `reviews/pr/6xxx/6048-cyclic-type-alias/1-57597956b/` |
| [6035](https://github.com/gnolang/gno/pull/6035) | `407bf9166` | jefft0 | +445-72, 18f | normal | `.worktrees/gno-review-6035` | `reviews/pr/6xxx/6035-playground-dry-run/1-407bf9166/` |

## Why 6068 is deep

6068 changes who may execute a gov/dao proposal and how a proposal page renders attacker-controlled
text, so a missed case is an unauthorised execution or an injection into a page every voter reads,
not a wrong result. It earns the lens dispatch, the critic pass and the claim-verification gate in
the *Deep mode* section of `skills/review.md`.

## Coupling

6060, 6058 and 6056 all change how the preprocessor scopes names inside a block, and 6048 changes
cycle handling in the same file, `gnovm/pkg/gnolang/preprocess.go`. Reconcile their findings against
each other before the batch commit: a rule one PR states and another contradicts is one conclusion
re-derived from the source, written into every affected review file.

## Carried from the 2026-08-12 run

Four worktrees still carry one uncommitted file each from that run's dead agents: 5923, 6053, 6057
and 6061. None is in this set. Left in place per the never-clean rule.

## Progress

Wave one is the six normal-mode PRs, one agent each. 6068's *Fetch & understand* and
*Reproduce the failure* ran in the parent — CI is green at `304f09a7a`, all check runs succeed, the
commit status is `success`, and the PR carries no human review and no inline comment, so nothing is
attributable to another reviewer. Its four lenses went out as wave two: red team, blue team,
correctness, and a gas/denial-of-service lens the PR's own numeric claims earn. Ten agents
concurrent, under the sixteen that hit `You've hit your session limit` on 2026-08-12.

All ten were killed at 00:27 on 2026-08-17, before any of them wrote a review file. `reviews/pr/6xxx/`
holds nothing from this run.

**The box cannot carry ten of these at once.** Six cores, 11 GB, no headroom: load average reached
46 and 3 GB went to swap, with one `gnolang.test` alone holding 3.6 GB. Two orphans outlived their
agents and had to be killed by hand — a `gnolang.test` and a `gnodev` left listening on 127.0.0.1:8891
by the 6035 live boot — because killing an agent does not reap the processes its shell started.
Four concurrent is the ceiling worth trying next, and a live-boot target counts double.

| PR | Agent | Review file | comment.md | Final check + QA | Committed |
|---|---|---|---|---|---|
| 6068 | done | `1-304f09a7a/review_claude-opus-5_davd-gzl.md` | `Event: REQUEST_CHANGES` | both passes applied | pending |
| 6067 | killed | — | — | — | — |
| 6060 | killed | — | — | — | — |
| 6058 | done (deep, re-run) | `1-391840aa7/review_claude-opus-5_davd-gzl.md` | `Event: REQUEST_CHANGES` | both passes applied | yes |
| 6056 | killed | — | — | — | — |
| 6048 | killed | — | — | — | — |
| 6035 | killed | — | — | — | — |

## 6058, re-run in deep mode

Re-dispatched on its own after the batch died, three lenses (red, blue, correctness) each in its own
worktree at `.worktrees/gno-6058-lens-{red,blue,corr}`, all three left pristine. Reconciliation
against the coupled PRs, which the Coupling section above asked for:

- [#6056](https://github.com/gnolang/gno/pull/6056) merged as `0cf310707` while this ran, so the
  branch's first commit duplicates merged code; the hunks are byte-identical and the merge is clean.
- The allocation change the red lens measured belongs to that commit, so it reproduces at current
  master and is not caused by this branch. Comparison worktree `.worktrees/gno-master-6058`.
- [#6060](https://github.com/gnolang/gno/pull/6060) fixes the `const` divergence this review's one
  Warning names, and its body says it was found while reviewing 6058. Merged into this head in
  `.worktrees/gno-6058-plus-6060`: `TestFiles` green in full.

## Left in the 6068 worktree

Two lenses left probe packages behind, untracked, in `.worktrees/gno-review-6068`. All were re-run
and settled. The executor-disclosure probe became the review's Warning and its repro. The gas probes
produced the Suggestion's numbers and killed a candidate finding about the unclamped proposal
description. One probe took the gate in `skills/security-advisory.md` and left this tree; it is
recorded where that skill sends it, and nothing about it belongs here. `.worktrees/` is gitignored,
so none of the three was ever publishable from this repo.

## Resume

Every worktree already exists with its PR checked out, so nothing in the setup has to run again.

1. From `checkout/`, `git fetch origin && git status`, confirm `0 0` against `origin/main`.
2. `export PATH=/tmp/go1259/go/bin:$PATH`, or re-unpack go1.25.9 when `/tmp` was wiped, per
   `projects/gno/AGENTS.md`.
3. `git -C gno fetch origin master`, then per PR
   `git -C .worktrees/gno-review-<number> rev-parse --short=9 HEAD` against the Head sha column. A
   moved head means a re-review round, not a first round: run the patch-id gate first.
4. Re-dispatch only the rows above reading `not started`.
