# gno-agent-workspace

Unofficial knowledge base and review workspace for [Gno](https://gno.land), designed to be driven by an AI coding agent.

Made by [samourai](https://github.com/samouraiworld).

> **Note:** This is not an official resource.

## Setup

```bash
git clone --recursive git@github.com:samouraiworld/gno-agent-workspace.git
```

Already cloned? Init the submodule:

```bash
git submodule update --init --recursive
```

## Structure

- `gno/` — gnolang/gno submodule
- `skills/` — AI skill definitions
- `reviews/pr/` — PR review reports (one directory per PR)
- `reports/weekly/` — weekly team status reports
- `reports/weekly-ux/` — weekly UX status reports
- `scripts/` — data-gathering and helper scripts
- `docs/` — architecture references (`overview.md`, `gnovm-architecture.md`)

## Skills

| Skill | File | Description |
|-------|------|-------------|
| PR Review | `skills/review.md` | Review a gnolang/gno pull request |
| Fix Issue | `skills/fix-issue.md` | Fix a gnolang/gno issue and open a PR |
| Weekly Report | `skills/weekly-report.md` | Generate the Samourai team weekly status report |
| Weekly UX Report | `skills/weekly-ux-report.md` | Generate the weekly UX report (a/ux label) |

## Scripts

| Script | Description |
|--------|-------------|
| `scripts/weekly-report.sh` | Gather PR/issue data for the weekly report |
| `scripts/parse-context.sh` | Parse `context.md` for AI-optimized reading |
| `scripts/fetch-pr-history.sh` | Fetch PR history |
| `scripts/fetch-issue-history.sh` | Fetch issue history |
| `scripts/build-reviews-readme.sh` | Rebuild the `reviews/README.md` index |

## Usage

Run from the root of this repository so the agent has access to skills, the gno submodule, and past reviews in a single workspace.
