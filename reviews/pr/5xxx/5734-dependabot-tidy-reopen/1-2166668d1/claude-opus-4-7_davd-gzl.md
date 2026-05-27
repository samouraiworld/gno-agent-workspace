# PR #5734: ci: re-trigger CI on dependabot tidy by close+reopening the PR

URL: https://github.com/gnolang/gno/pull/5734
Author: @thehowl | Base: master | Files: 1 | +14 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `2166668d1` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5734 2166668d1`

**Verdict: APPROVE** — minimal, well-targeted fix for a real CI gap (#3312, untouched for 13 months); the close+reopen idiom is the standard workaround when a PAT/App is off the table, the loop-prevention is correct (second run finds nothing to tidy → close+reopen step gated off by `changes_detected: false`), all `pull_request` workflows pick up the new HEAD via the default `reopened` activity type, and CI is green. Minor concerns are non-blocking: brief PR-state flap visible in the timeline, no fallback if `gh pr close` succeeds but `gh pr reopen` fails, and one stylistic Q on `--repo` explicitness.

## Summary

`meta-dependabot-tidy.yml` pushes a `go mod tidy` commit back to dependabot PRs when the recursive updates land go.mod/go.sum out of sync. Pushes signed by the default `GITHUB_TOKEN` deliberately do NOT trigger further workflow runs (GitHub's recursion guard), so CI on dependabot PRs stayed pinned to the pre-tidy SHA and any subsequent test failure introduced by the tidy went unobserved — open since #3312 (Jan 2025), partial fix #3306 didn't address this, persist-credentials fix #5685 only made the push itself work. This PR grants the workflow `pull-requests: write`, captures the `git-auto-commit-action` step output (`changes_detected`), and conditionally runs `gh pr close && gh pr reopen` to fire a fresh `pull_request: reopened` event against the post-tidy HEAD. All downstream PR workflows use default activity types (which include `reopened`), so they all re-run against the correct SHA.

```
dependabot opens PR (SHA=A)         workflow runs:  CI sees SHA=A
        │
        ▼  meta-dependabot-tidy.yml triggers on `opened`
   checkout A
   make tidy → dirty
   git-auto-commit-action pushes SHA=B (commit signed by GITHUB_TOKEN)
        │
        ▼  GITHUB_TOKEN push DOES NOT retrigger CI ← the bug
                                    CI still sees SHA=A   ← pre-fix endgame
        │
        ▼  this PR: if changes_detected == true:
   gh pr close && gh pr reopen
        │
        ▼  pull_request: reopened fires for every `on: pull_request` workflow
                                    CI now sees SHA=B   ← post-fix endgame
        │
        ▼  meta-dependabot-tidy.yml re-runs against SHA=B
   checkout B
   make tidy → clean (no changes_detected)   ← loop terminates here
```

## Glossary

- `changes_detected` — output of [`stefanzweifel/git-auto-commit-action`](https://github.com/stefanzweifel/git-auto-commit-action#outputs), `"true"` iff the step pushed a commit.
- `GITHUB_TOKEN` recursion guard — documented Actions behavior: workflow runs triggered by `GITHUB_TOKEN` pushes do not themselves trigger new workflow runs. Prevents accidental loops, blocks this use case. See [docs.github.com](https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication#using-the-github_token-in-a-workflow) "events triggered by `GITHUB_TOKEN` will not create a new workflow run".
- `pull_request` default activity types — `opened`, `synchronize`, `reopened` (per [docs.github.com](https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/events-that-trigger-workflows#pull_request)). `closed` is NOT default.

## Fix

Before: tidy commit pushed by `GITHUB_TOKEN`, downstream PR workflows on the dependabot PR never observed the new HEAD. After: a final step runs `gh pr close "$PR_URL" && gh pr reopen "$PR_URL"` (gated on `steps.commit.outputs.changes_detected == 'true'`) to fire `pull_request: closed` then `pull_request: reopened`. The `reopened` activity type is in the defaults for all `on: pull_request` workflows in the repo (verified by reading every workflow under [`.github/workflows/`](https://github.com/gnolang/gno/tree/2166668d1/.github/workflows) · [↗](../../../../../.worktrees/gno-review-5734/.github/workflows)), so they re-run against the new HEAD. The load-bearing gate is `changes_detected`: on the second invocation (triggered by the reopen) the workflow checks out the already-tidied HEAD, `make tidy` is a no-op, the action reports `changes_detected: false`, and the close+reopen step is skipped. No infinite loop.

## Critical (must fix)

None.

## Warnings (should fix)

- **[close succeeds, reopen fails → PR left closed]** [`meta-dependabot-tidy.yml:60-62`](https://github.com/gnolang/gno/blob/2166668d1/.github/workflows/meta-dependabot-tidy.yml#L60-L62) · [↗](../../../../../.worktrees/gno-review-5734/.github/workflows/meta-dependabot-tidy.yml#L60-L62) — two sequential `gh` calls, no retry, no compensating action. If `gh pr close` returns 200 and `gh pr reopen` then fails (transient API blip, abuse-rate-limit), the dependabot PR is left in CLOSED state with the tidy commit and no signal to anyone.
  <details><summary>details</summary>

  In practice the GitHub REST endpoints behind `gh pr close/reopen` (PATCH `/repos/{owner}/{repo}/pulls/{pull_number}` with `state=closed|open`) are reliable, and the abuse-rate-limit window is generous compared to one workflow run per dependabot PR per week. The failure mode is rare. But "rare and silent" is the worst kind: a closed dependabot PR with a tidy commit looks indistinguishable from a maintainer-rejected dependency bump, and the next time dependabot opens a new PR for the same dependency family it has to retidy from scratch (assuming dependabot doesn't go quiet because the PR is "closed unmerged" — see Questions). Fix: wrap the reopen in a small retry, or invert the order (`reopen` is a no-op if the PR is already open, so `gh pr reopen "$PR_URL" || true; gh pr close "$PR_URL" && gh pr reopen "$PR_URL"` is closer to idempotent — though the simplest mitigation is just `gh pr reopen "$PR_URL"` with a couple of `|| sleep 5 && gh pr reopen "$PR_URL"` retries).
  </details>

## Nits

- [`meta-dependabot-tidy.yml:61-62`](https://github.com/gnolang/gno/blob/2166668d1/.github/workflows/meta-dependabot-tidy.yml#L61-L62) · [↗](../../../../../.worktrees/gno-review-5734/.github/workflows/meta-dependabot-tidy.yml#L61-L62) — `gh pr close/reopen "$PR_URL"` works but `--repo gnolang/gno` would make the call independent of the runner's current working directory (the runner is in a checkout, so `gh` infers the repo correctly — but explicit beats implicit for ops scripts).
- [`meta-dependabot-tidy.yml:18`](https://github.com/gnolang/gno/blob/2166668d1/.github/workflows/meta-dependabot-tidy.yml#L18) · [↗](../../../../../.worktrees/gno-review-5734/.github/workflows/meta-dependabot-tidy.yml#L18) — comment "GITHUB_TOKEN pushes don't" is a fragment; full clause `(GITHUB_TOKEN pushes don't retrigger workflows)` would survive being read out of context (e.g. in a permissions audit).

## Missing Tests

- N/A — this is a workflow change, validation is end-to-end on the next dependabot PR that produces a tidy commit. The PR body's test plan correctly notes this. Both bullets there (positive — CI fires on new HEAD; negative — no-op runs leave the PR untouched) cover the contract.

## Suggestions

- [`meta-dependabot-tidy.yml:55-62`](https://github.com/gnolang/gno/blob/2166668d1/.github/workflows/meta-dependabot-tidy.yml#L55-L62) · [↗](../../../../../.worktrees/gno-review-5734/.github/workflows/meta-dependabot-tidy.yml#L55-L62) — consider posting a one-line comment on the PR before the reopen so a human scanning the timeline knows why the PR flapped. `gh pr comment "$PR_URL" --body "Re-running CI after \`go mod tidy\` (GITHUB_TOKEN pushes don't fire workflows)."` is one extra line, makes the timeline self-documenting, and survives the next maintainer who hasn't read this PR.
  <details><summary>details</summary>

  Cost: one comment per dependabot tidy event (≤ weekly in practice). Benefit: removes the "did someone close my PR?" confusion. Not worth blocking on, but it's the cheapest UX improvement on this PR.
  </details>

- The PR body acknowledges this is "the simplest workaround". The structurally cleaner long-term answer is a PAT or a small GitHub App with `pull_request:write` + `contents:write` scoped to the gno org — push via the App's token, no workaround needed. Worth a follow-up issue tracking the migration (the App approach also dodges the rare two-call failure mode in the Warning above), even if this PR ships first as the quick win.

## Questions for Author

- Does closing a dependabot PR — even for ~100ms — risk dependabot interpreting the close as a "user rejected this version" signal and not re-opening / not bumping again? Per GitHub's docs, dependabot watches state transitions for some signals (e.g. `@dependabot rebase` in a comment), but I couldn't find authoritative text on what an immediate-reopen-by-bot does to its internal "rejected versions" state. If you've already observed this working in a smoke test on a fork, that resolves it; otherwise worth a one-line confirmation after the first real dependabot PR fires through it.
- Any preference between this PR's close+reopen and the alternative of an `actions/github-script` call to POST an empty commit status / `dispatch` a `repository_dispatch` event? Both avoid the timeline flap but require more YAML. Close+reopen is fine for now; logging the alternative for the future.
