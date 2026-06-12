# AGENTS.md

Knowledge base and review workspace for [gnolang/gno](https://github.com/gnolang/gno).

## Repo Layout

- `gno/` — gnolang/gno submodule
- `skills/` — AI skill definitions
- `reviews/pr/` — PR review reports
- `reports/weekly/` — Weekly team reports (Samourai)
- `reports/weekly-ux/` — Weekly UX team reports (a/ux label)
- `scripts/` — Data-gathering and helper scripts
- `docs/` — architecture references (`overview.md`, `gnovm-architecture.md`)

## PR Review

When given a PR number or URL, read and follow `skills/review.md`.

When asked to **review all** (e.g. "review all", "review all non-reviewed recent PRs"), read and follow `skills/review.md` — see its "Review all" section: review every open, non-draft PR whose number is absent from `reviews/pr/`, excluding `WIP`-titled and dependabot PRs unless explicitly included.

When asked for a **parallel**, **red-team / blue-team**, or **deeper** review of a single PR (or "review and loop until perfect"), read and follow `skills/review.md` — see its "Deep mode" section.

When the user says **post** pointing at a `comment_<model>.md` draft (open file or path in the message), that is one-shot approval: run `./scripts/post-pr-review.py <number> <path>` directly, without reading the draft or the review file. The PR number is the `<number>-<slug>/` segment of the path. If the script refuses an APPROVE event, get the user's explicit confirmation of the approval, then re-run with `--approve`. If it reports invalid anchors, follow the "GitHub review draft" section of `skills/review.md`. When the draft already carries a `Posted:` line, the script rewrites the posted review in place (body and `[posted]`-linked inline comments); no `--approve`, the event doesn't change. After a successful post, commit and push the script-updated draft: `review: PR #<number> posted (<event>)`.

## Fix Issue

When asked to fix a gnolang/gno issue (bug, security fix, etc.), read and follow `skills/fix-issue.md`. Supports two modes: `fix` to implement and open a PR, `cleanup` to remove worktrees for merged PRs.

## Weekly Report

When asked to generate or update the weekly team report, read and follow `skills/weekly-report.md`. The data-gathering script is `scripts/weekly-report.sh`. Reports are saved in `reports/weekly/`.

## Weekly UX Report

When asked to generate or update the weekly UX report (a/ux label), read and follow `skills/weekly-ux-report.md`. Data is fetched directly via `gh` CLI. Reports are saved in `reports/weekly-ux/`.

## Rules

- **Always read `gno/AGENTS.md`** at the start of any task involving the gno repository. It contains project-specific conventions, build instructions, and coding guidelines that must be followed.
- **Never write into the `gno/` submodule in-place.** Any task that modifies files under `gno/` — code, docs, READMEs, anything — happens inside a worktree at `.worktrees/gno-<slug>/`. See `skills/fix-issue.md` for the worktree-creation procedure. Docs/README work is not an exception: "small" is not a reason to skip a worktree.
- **Never push to gnolang/gno** for review purposes. Pushing to a fork of gnolang/gno is acceptable for specific cases (e.g. cherry-picks).
- After writing a review, commit and push to this repo only: `git add reviews/ && git commit -m "review: PR #<number>" && git push`.
- **Every `scripts/*.sh` carries the NOT AUDITED line as line 2**, right after the shebang: `# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.` then a `#` separator. Never on adversarial test files under `reviews/.../tests/` — those use the test-file disclaimer in `skills/review.md`.

## Authoring skills, prompts, and these instruction files

- Strip rationale, justifications, and war-stories; keep only directives, in imperative form. State a non-obvious *condition* tersely, never the historical reason. When a template defines the full output, removing a section is the rule: don't add a "No X section" note explaining the absence. Applies to skills, agent prompts, and `AGENTS.md`; not to commit messages, PR descriptions, code comments, or chat.
- A prompt that delegates to a skill points at it, never restates its steps. A routine or dispatched-agent prompt carries only: the skill pointer, automation-specific deltas, output requirements, error boundaries.
- When one of a skill's rules proves unclear, missing, or wrong during use, update the skill in the same turn — don't wait to be asked. Cross-PR conventions belong in the skill; one-off PR specifics don't.
