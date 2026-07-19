---
name: gno-fix-issue
description: Fix a gnolang/gno issue — plan, implement in a worktree, open a PR on a fork. Worktrees persist until merged; run 'cleanup' to verify and remove. Also verifies CI after a PR is opened and fixes failures related to our changes.
argument-hint: "<fix <issue-number|URL|description>> | ci <pr-or-id> | cleanup"
---

# Gno Fix Issue

**Input:** `$ARGUMENTS` — three modes:

- **`fix <issue>`** — Implement a fix. `<issue>` is a GitHub issue number, URL, free-text description, or a security report ID (e.g. `NEWTENDG-194`). When fixing a security report, read the corresponding report in `reviews/security/<ID>-*/` for analysis and reproduction details.
- **`ci <pr-or-id>`** — Verify CI on the fix's PR and fix failures related to our changes. `<pr-or-id>` is the PR number or the fix worktree id (e.g. `1234` or `banker-nil-panic`).
- **`cleanup [issue]`** — Check fix worktrees for merged PRs and remove them.

**Before starting:** read `gno/AGENTS.md` — it has all project conventions, build/test commands, and ADR requirements. Follow it.

## Setup

Ensure a fork remote exists (`git -C gno remote -v`). If not:
```
gh repo fork gnolang/gno --remote-only --remote-name fork
```
Never push to `origin` (gnolang/gno). Push only to the user's fork `davd-gzl` (alias `fork`); every other remote is another contributor's fork, never a push target. Read `git remote -v` in full, never truncated.

---

## `fix`

1. **Understand** — Fetch the issue (`gh issue view` or `gh search issues`). Read linked code and repro steps.

2. **Plan** — Trivial fix? Skip ahead. Non-trivial? Note root cause, approach, files, test strategy, and whether an ADR is needed.

3. **Worktree** — All work in `.worktrees/` (gitignored), never in `gno/` directly:
   ```
   git -C gno fetch origin master
   git -C gno worktree add ../.worktrees/gno-fix-<id> origin/master
   git -C .worktrees/gno-fix-<id> checkout -b fix/<branch-name>
   ```
   `<id>` = issue number when available (e.g. `gno-fix-1234`), otherwise a short slug (e.g. `gno-fix-banker-nil-panic`).

4. **Implement** — Fix and test inside the worktree. Follow `gno/AGENTS.md` conventions. **Never `git commit`, `git push`, or open a PR without explicit user permission.**

   Before pushing any Go file, run `go fix -diff ./<changed-pkg>/` and apply what it reports. The `main / lint` job fails on any `go fix` diff, separately from golangci-lint. New test loops use `for i := range N`, never `for i := 0; i < N; i++`.

5. **Summarize** — Report what was done. List changed files and what each change does.

6. **Rename the ADR** — Once the PR number exists, `git mv` any ADR from the `prxxxx_` placeholder to `pr<number>_`, per the naming rule in `gno/AGENTS.md`. The rename only reaches the PR when it is committed and pushed, so do it in the same round as any other post-open change.

7. **Keep the worktree** — Don't remove it. It stays for review feedback, rebases, and further work until merged.

8. **Schedule a CI check** — Only after a PR has been opened (with the user's explicit approval). Use the `/schedule` skill to run `/gno-fix-issue ci <pr-number>` ~25 min after the push (typical gno CI completion window). One-time schedule, not recurring. If the user prefers to monitor manually, skip this step.

### PR description

Write for a reader with no context on the bug. Order: damage, then example, then mechanism. Never open with setup the reader must hold before anything matters. Plain prose throughout: short paragraphs, no headers, no tables, no hard wrap, no AI footer.

- First sentence names what breaks, in the reader's terms, plain present tense. Banned openers: "Today", "Currently", "At the moment", "nothing more than".
- When the bug has a severe consequence and a mild one, lead with whichever is unambiguous. A severe example that looks like obvious garbage reads as correct behaviour and hides the bug; lead with the case where something plainly wrong is accepted.
- Then one fenced block: the smallest input a user would actually write, and what it does. A second contrasting block only when the bug is a disagreement (two builds, two nodes, two versions).
- Example values must be plausible: `go1.26` against a go1.25 build shows the defect with nothing granted; `go1.99` invites "that should fail anyway". Quote real strings: paste the actual error message, don't describe it.
- For a data-layout or ordering fix, show the layout; a few lines of keys can replace a paragraph and make the fix self-evident.
- When the bug is about when things happen, draw the sequence as a few plain lines with the step and what it does, then redraw it with the fix in place. Two small diagrams beat a paragraph of "first, then, but".
- Explain why the existing guard failed only after the reader has seen it fail. For a disagreement bug, state that the disagreement is the defect, not either answer: uniform rejection would be fine, differing answers are not.
- After the example: what the fix does, why nothing legitimate breaks, what test covers it. Name any deliberate omission and why; a deferred half of a fix states what makes the obvious version wrong.
- Plain words, no insider shorthand: "sets the version" not "pins it", "answer" not "verdict", "lookup index" not "fast index", "jump to each end" not "seek". Spell a term out on first use or drop it.
- Cut every sentence that survives deletion: restatements of the opening, truisms the audience already holds, openers announcing what follows.
- State the problem, not its history. Why upstream built a mechanism, when it was introduced, which PR left it behind: none of it changes what the reader does.
- Name an unrelated rider commit in one line. Don't offer to split it; the maintainer will ask if they want that.
- No caveats about local-only failures: CI runs a different toolchain, verify redness there first.

---

## `ci`

Verify CI on the fix's PR and patch failures that are caused by our changes. **This mode never pushes or commits without explicit user permission.**

1. **Locate the PR** — Resolve `<pr-or-id>` to a PR. If given a worktree id, find the branch in `.worktrees/gno-fix-<id>/` and look up the PR via `gh pr list --repo gnolang/gno --head <branch> --state all --json number,headRefName,statusCheckRollup`.

2. **Fetch CI status** —
   ```
   gh pr checks <pr> --repo gnolang/gno
   gh pr view <pr> --repo gnolang/gno --json statusCheckRollup,headRefOid
   ```
   - All green → report success, done.
   - Still running → re-schedule another `ci` check in 15 min via `/schedule` and stop.
   - Failing → continue.

3. **Triage failures** — For each failing check, fetch logs:
   ```
   gh run view <run-id> --repo gnolang/gno --log-failed
   ```
   Classify each failure as one of:
   - **Related** — the failure touches code, tests, files, or behavior our PR changed (matches a path in `git -C .worktrees/gno-fix-<id> diff --name-only origin/master`, or the error references a symbol/test we modified).
   - **Unrelated** — pre-existing failure on master, infra/runner issue, or a flaky test with no link to our diff. Verify by checking the same job on a recent master commit (`gh run list --repo gnolang/gno --branch master --workflow <name> -L 5`).
   - **Ambiguous** — surface to the user; don't guess.

4. **Fix related failures** — Work inside the existing worktree `.worktrees/gno-fix-<id>/`. Reproduce locally when feasible (run the failing test/lint target). Apply the minimal fix. **Do not commit or push** — report the diff and wait for the user's explicit "push" before continuing.

5. **Report** — Summarize: which checks failed, which were related vs unrelated vs ambiguous, what was changed locally (if anything), and what still needs the user's call.

---

## `cleanup`

List fix worktrees and check their PR status:
```
git -C gno worktree list | grep gno-fix-
```

For each (or for a specific `<id>` if given):
- Find the branch → find the PR (`gh pr list --repo gnolang/gno --head <branch> --state all --json number,state,mergedAt`)
- **Merged:** remove worktree and branch.
- **Open:** keep, report status.
- **Closed (not merged):** ask user.

Print a summary table when done.
