# Handoff: offline PR review dashboard

Brief for a new repository. Move this file into the new repo and start a Claude session there with it as context.

## Goal

A local GUI dashboard for reviewing gnolang/gno pull requests, fully offline once synced. A background job fetches PR data on an interval when the network is available. The dashboard browses the mirror with zero network and triggers the existing agent review workflow that lives in `~/Projects/gno-agent-workspace`.

## Decisions already made

- Custom build. Off-the-shelf tools were surveyed and rejected:
  - gh-dash: TUI only, no GUI, needs network per view.
  - git-appraise: offline reviews but its own format, no GitHub PR bridge.
  - Forgejo/Gitea mirror: code syncs, PRs and comments do not.
  - git-bug: GitHub bridge covers issues only.
  - VS Code GitHub PR extension: good UI but online-only and no hook to launch agent commands with PR context.
- GUI, not TUI.
- Fully offline browsing. Sync is interval-based, not on-demand-only.
- Interactive cockpit, not read-only viewer (user confirmed). Per-PR actions:
  - launch agent review: run `claude "review PR <N>"` from the workspace directory
  - open the review draft directory in VS Code
  - post a draft via the workspace post script

## Architecture sketch (proposed, not yet fully validated)

Three parts:

1. **Sync job.** systemd user timer, interval TBD. Fetches via authenticated `gh` CLI: open PR list, per-PR metadata, diffs, review threads, comments, CI status. Writes JSON/JSONL plus raw diffs into a local cache directory. Optionally `git fetch origin pull/<N>/head:<ref>` in a repo checkout for full local diff capability. Must degrade silently when offline.
2. **Local server.** Localhost-only, small (Python stdlib or Flask, stack TBD). Serves the UI from the cache and exposes a few action endpoints that shell out: spawn claude review, open VS Code, run the post script. No remote exposure, no auth needed beyond localhost binding.
3. **UI.** PR list with state, CI, age, diff stats, and review status (derived from whether a draft exists in the workspace `reviews/pr/` tree). Detail view: description, diff per file, comment threads, link to the local review draft and `overview.html` if present.

Build order suggestion: sync script first (data before UI), then static viewer, then action endpoints.

## Integration points in gno-agent-workspace

All paths under `/home/davd/Projects/gno-agent-workspace/`:

- `scripts/fetch-pr-history.sh`: existing gh-based JSONL fetcher with `--with-comments --with-files --with-reviews` flags. Reuse or crib from it for the sync job.
- `reviews/pr/<thousand>xxx/<number>-<slug>/`: per-PR review artifacts. `review_*.md` (full report), `comment_<model>.md` (GitHub draft), `overview.html` (self-contained PR explainer). Presence of a directory for a PR number means it was reviewed.
- `scripts/post-pr-review.py <number> <draft-path>`: posts a draft review to GitHub. Drafts carrying a `Posted:` line get rewritten in place on re-post.
- `skills/review.md`: the review skill. The dashboard never reimplements it, it only launches `claude "review PR <N>"` in that workspace.
- `.claude/commands/overview.md`: builds the per-PR overview.html.

## Constraints and environment

- Posting a review to GitHub requires explicit user approval. A human clicking a "post" button in the dashboard counts as that approval; the sync job or any automation must never post on its own.
- The dashboard repo is separate from gno-agent-workspace. It must not write into the workspace except through the defined actions (spawning claude, running the post script).
- Environment: Arch Linux, fish shell, systemd user services available, VS Code, `gh` CLI authenticated, `jq` present. Verify tmux availability before depending on it for spawning terminal sessions.
- Writing style for any docs in the new repo: no em-dashes, no emoji, concise.

## Open questions (brainstorming was interrupted here)

1. Mirror scope: open non-draft PRs only, or also drafts, recently merged, recently updated? Issue mirroring too?
2. Sync interval, and whether the UI gets a manual "sync now" button.
3. Diff rendering: pre-rendered server-side, or client-side from raw patch text?
4. Server stack preference: Python stdlib, Flask, Go, Node?
5. Cache location: inside the new repo (gitignored), or XDG cache dir?
6. How the dashboard finds the workspace: hardcoded path, config file, or env var?
7. How "launch claude review" surfaces the running session: detached headless (`claude -p`), tmux window, or new terminal?

## Suggested first step in the new repo

Resume brainstorming from the open questions above, write the design doc, then implement the sync job first.
