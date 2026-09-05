# AGENTS.md

Knowledge base and review workspace for [gnolang/gno](https://github.com/gnolang/gno).

## Repo Layout

- `gno/` — gnolang/gno submodule
- `skills/` — AI skill definitions. `skills/core/` is a submodule of
  [`davd-gzl/skills`](https://github.com/davd-gzl/skills), the canonical shared
  rules; the top-level files carry the gno deltas and point at it. Edit core
  rules there, edit gno deltas here. The submodule tracks `main`, so
  `git submodule update --init --remote skills/core` after cloning or pulling
  takes the tip. Never commit the moved gitlink on its own: it rides along with
  the next commit this repo makes.
- `reviews/pr/` — PR review reports
- `reports/weekly/` — Weekly team reports (Samourai)
- `reports/weekly-ux/` — Weekly UX team reports (a/ux label)
- `scripts/` — Data-gathering and helper scripts
- `docs/` — architecture references (`overview.md`, `gnovm-architecture.md`), the site that
  serves them on GitHub Pages, and `figures/gen.py`, which draws every `figures/*.svg`: edit the
  script, never an SVG

## PR Review

When given a PR number or URL, read and follow `skills/review.md`.

**Every gno review uses this repo's `skills/review.md`, whatever the target repo is.** That file is
the delta; the core it overrides is the submodule beside it at `skills/core/review.md`, and the
delta wins wherever the two disagree. Read both, in that order, and never the core alone: it is
repo-agnostic by design and carries none of the gno rules, so a review written against it comes
out confident and well-formed in the wrong house style, missing the comment anchor format, the
`Fix:` sentence, the banned finding classes and the repro conventions. Nothing in the output
announces the miss, which is what makes it worth stating here. The rule holds for a fork or a
mirror of gno as much as for `gnolang/gno` itself: the target changes, the skill does not.

When asked to **review all** (e.g. "review all", "review all non-reviewed recent PRs"), read and follow `skills/review.md` — see its "Review all" section: review every open, non-draft PR whose number is absent from `reviews/pr/`, excluding `WIP`-titled and dependabot PRs unless explicitly included.

When asked for a **parallel**, **red-team / blue-team**, or **deeper** review of a single PR (or "review and loop until perfect"), read and follow `skills/review.md` — see its "Deep mode" section.

When the user says **post** pointing at a `comment_<model>.md` draft (open file or path in the message), that is one-shot approval for any event: run `./scripts/post-pr-review.py <number> <path>` directly, without reading the draft or the review file. The PR number is the `<number>-<slug>/` segment of the path. If it reports invalid anchors, follow the "GitHub review draft" section of `skills/review.md`. When the draft already carries a `Posted:` line, the script rewrites the posted review in place (body and `[posted]`-linked inline comments); the event doesn't change. After a successful post, commit and push the script-updated draft: `review: PR <number> posted (<event>)`.

## Review History

When asked what was already reviewed (whether a PR was reviewed, what a past review found, which drafts were posted, recurring patterns across reviews), read and follow `skills/review-history.md`. Read-only over `reviews/`; never re-review to answer a history question.

## Fix Issue

When asked to fix a gnolang/gno issue (bug, security fix, etc.), read and follow `skills/fix-issue.md`. Supports two modes: `fix` to implement and open a PR, `cleanup` to remove worktrees for merged PRs.

## Security Advisory

When asked to verify a security finding against deployed/merged gnolang/gno code and write it up for private disclosure, read and follow `skills/security-advisory.md`. Verify by execution (filetest repro), route output to the private disclosure repo, produce a CVSS vector and a paste-ready body for the GitHub Security Advisory form. Findings against an open PR's own diff are reviews, not disclosures: use `skills/review.md`.

## Weekly Report

When asked to generate or update the weekly team report, read and follow `skills/weekly-report.md`. The data-gathering script is `scripts/weekly-report.sh`. Reports are saved in `reports/weekly/`.

The script dies on `ERROR: 'jq' is required but not found in PATH` on a machine with no `jq` and no `sudo`. Fetch the static binary and put it on the path for the run; it needs no privileges:

```bash
mkdir -p ~/bin && curl -fsSL -o ~/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-amd64 && chmod +x ~/bin/jq
export PATH="$HOME/bin:$PATH"
```

The full fetch takes about four minutes: it walks every open PR one GraphQL call at a time.

`:: Done. Open PRs: 0` with no error is a token-scope failure, not an empty week. `gh pr list --json reviewRequests` resolves `login` on the Team variant, which needs `read:org`; without it the whole call fails and every member silently contributes zero PRs. Check with `gh auth status` (`! Missing required token scopes: 'read:org'`). The script now asks for requested reviewers through its own GraphQL query restricted to the User variant, so no scope beyond `repo` is needed. Never accept a zero count from that line — compare against last week's `context.md`.

## Weekly UX Report

When asked to generate or update the weekly UX report (a/ux label), read and follow `skills/weekly-ux-report.md`. Data is fetched directly via `gh` CLI. Reports are saved in `reports/weekly-ux/`.

## Issue

When a fix needs an issue and none covers it upstream, read and follow `skills/issue.md`. It searches first and records what the search covered.

## PR body

When writing the title and body of a PR, read and follow `skills/pr-body.md`. It loops on the draft until a pass changes nothing, and covers when a change ships a screenshot or a video.

## Writing docs & comments

When writing or editing gno docs (`docs/resources/*.md`, READMEs), code comments, or PR review comments, read and follow `skills/writing-style.md`.

## Rules

- **This repository is public.** Everything committed here is world-readable the moment it is pushed, commit messages and branch names included. Never commit: credentials or tokens; private infrastructure names, hostnames, or repo names; absolute local paths; or a security finding that is exploitable against deployed code. Check `gh repo view <repo> --json visibility` before committing anything sensitive to any repo. Force-pushing does not erase: orphaned commits stay reachable by SHA until GitHub Support removes them, so a leak is permanent until reported. Treat anything already pushed as exposed.
- **Security findings follow deployment, not severity.** A finding against an open PR's own diff publishes normally, at any severity. A finding exploitable against already-merged or deployed code is a disclosure: keep it out of this repo, store it privately, and raise it with the user before writing anything. `gno/SECURITY.md` forbids a public issue, and a public fix PR telegraphs the vector just as loudly. Audit output stays out of the tree; `reviews/security/` is gitignored.
- **Sync this checkout before reading any state out of it.** A fetch in a parent directory does not reach inside a submodule, and this repo may be checked out as one. Before the first task: `git remote -v`, fetch every remote listed, and for each compare `git rev-list --left-right --count HEAD...<remote>/<branch>`. `samouraiworld` is canonical; a personal fork on `origin` may sit hundreds of commits behind it, carrying a stale `reviews/` and stale `skills/`. Fast-forward only; on diverged history report and stop. Pass `git -C <path>` on every command — the shell working directory does not persist between commands.
- **Always read `gno/AGENTS.md`** at the start of any task involving the gno repository. It contains project-specific conventions, build instructions, and coding guidelines that must be followed.
- **`bash: go: command not found` means the toolchain is missing, not that the test failed.** A container without Go on `PATH` fails every `go test`, every `gno` build and every filetest with that one line, so a review that reports it as a test result reports nothing. `gno/go.mod` requires `go 1.25.9`; install it without privileges and link it onto `~/bin`:
  ```bash
  curl -fsSL https://go.dev/dl/go1.25.9.linux-amd64.tar.gz | tar -xz -C /tmp && ln -sf /tmp/go/bin/go /tmp/go/bin/gofmt ~/bin/
  ```
  **`~/bin` is not on `PATH`.** Measure it rather than assuming: a session has run with `~/bin/go` present and `go version` still answering `command not found`, because the default `PATH` carries neither `~/bin` nor `/tmp/go/bin`. Every shell that needs the toolchain opens with `export PATH=$HOME/bin:$PATH`, and every dispatched agent's prompt says so, since the export does not survive into another shell. `/tmp` is wiped between sessions, so the symlink can outlive its target: check `go version`, not `ls ~/bin`, before dispatching a batch.
- **Never write into the `gno/` submodule in-place.** Any task that modifies files under `gno/` — code, docs, READMEs, anything — happens inside a worktree at `.worktrees/gno-<slug>/`. See `skills/fix-issue.md` for the worktree-creation procedure. Docs/README work is not an exception: "small" is not a reason to skip a worktree.
- **Never push to gnolang/gno** for review purposes. Pushing to a fork of gnolang/gno is acceptable for specific cases (e.g. cherry-picks).
- **When working on the fork, always pull from `origin` (upstream master) first, then run the command.**
- After writing a review, commit and push to this repo only: `git add reviews/pr/<thousand>xxx/<slug> && git commit -m "review: PR <number>" && git push origin HEAD:refs/heads/main`. Stage the review's own directory, never `reviews/` whole, which sweeps up another session's unfinished round. The explicit refspec is not optional: this repo is usually consumed as a submodule, so `HEAD` is detached and a bare `git push` answers `You must fully qualify the ref`. No bare `#<number>` in the subject: it autolinks to this repo, not gnolang/gno.
- **Every `scripts/*.sh` carries the NOT AUDITED line as line 2**, right after the shebang: `# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.` then a `#` separator. Never on adversarial test files under `reviews/.../tests/`.

## Authoring skills, prompts, and these instruction files

- Strip rationale, justifications, and war-stories; keep only directives, in imperative form. State a non-obvious *condition* tersely, never the historical reason. When a template defines the full output, removing a section is the rule: don't add a "No X section" note explaining the absence. Applies to skills, agent prompts, and `AGENTS.md`; not to commit messages, PR descriptions, code comments, or chat.
- A prompt that delegates to a skill points at it, never restates its steps. A routine or dispatched-agent prompt carries only: the skill pointer, automation-specific deltas, output requirements, error boundaries.
- When one of a skill's rules proves unclear, missing, or wrong during use, update the skill in the same turn — don't wait to be asked. Cross-PR conventions belong in the skill; one-off PR specifics don't.
