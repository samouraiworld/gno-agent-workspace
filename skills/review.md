---
name: gno-review
description: Adversarial review of one or more gnolang/gno PRs. Takes space-separated PR numbers or URLs; runs the core review skill with the gno deltas below. Supports "review all" batch runs, multi-PR parallel dispatch, and a deep multi-angle mode for a single PR.
argument-hint: <pr-number> [pr-number...]
---

# Gno PR review

The workflow is `skills/core/review.md`; prose per `skills/core/writing-style.md`, including its closing Pass. Everything below adds to or overrides the core for `gnolang/gno`. A core section not named here applies unchanged.

**Input:** `$ARGUMENTS` — space-separated PR numbers or GitHub URLs. Process each PR independently.

## Layout

This workspace has no `projects/` tree; the core's `projects/<repo>/reviews/<slug>/` reads here as:

- One directory per review round: `reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/` (`<thousand>` = leading digit(s): 4 for 4000–4999, 5 for 5000–5999). Pre-existing rounds may lack the `review_` prefix.
- Fix artifacts (`plan.md`, `issue.md`) live in the round directory too; the fix worktree is `.worktrees/gno-<slug>/`.
- A PR outside `gnolang/gno` goes under `reviews/<repo>/`, not `reviews/pr/`. First review for a repo: create `reviews/<repo>/README.md` with the repo link and one line. Skip the gno-only steps below (worktree, invariant catalog, dual links); cite plain `file:line` from the repo's own checkout and post via `gh`.
- Never run `./scripts/build-indexes.sh` as part of a review; `reviews/README.md` regenerates only on request.
- There is no `post-fix.sh` here: post reviews with `./scripts/post-pr-review.py`, issues standalone with `gh issue create`.
- There is no `prose-check.py` here either: run the `skills/core/writing-style.md` Pass by hand, every step, and say so when handing over.

## Worktree

The core's worktree rule, with the gno paths:

- `git -C gno fetch origin master`, then one worktree per PR:
  ```bash
  git -C gno worktree add ../.worktrees/gno-review-<number> origin/master
  cd <workspace-root>/.worktrees/gno-review-<number> && gh pr checkout <number> -R gnolang/gno
  ```
- After the review, ask the user before opening the worktree in VSCode (`code <workspace-root>/.worktrees/gno-review-<number>`).
- Confirm symbol existence with `gno lint` from the worktree source (`go run ../gnovm/cmd/gno lint ./path`), never IDE diagnostics; sanity-check that lint typechecks by feeding it a bogus symbol.

## Review all

When invoked with "review all" (no explicit PR numbers), build the target set:

```bash
ls reviews/pr/*xxx 2>/dev/null | grep -oE '^[0-9]+' | sort -un > /tmp/reviewed.txt
gh pr list -R gnolang/gno --state open --limit 200 --json number,title,isDraft \
  --jq '.[] | select(.isDraft==false) | "\(.number)\t\(.title)"' > /tmp/open_nondraft.txt
while IFS=$'\t' read -r num title; do
  grep -qx "$num" /tmp/reviewed.txt || printf '%s\t%s\n' "$num" "$title"
done < /tmp/open_nondraft.txt
```

- Sync this repo per the `AGENTS.md` sync rule before the `ls reviews/pr/` above, and state the synced head when confirming the set. Diverged and unsyncable: derive the set read-only from the remote tree (`git ls-tree -r --name-only <remote>/<branch> -- reviews/pr/`), never from the working tree.
- Exactly 200 rows back from `gh pr list` means `--limit` clipped the list; re-run higher.
- Exclude `WIP*` titles and dependabot PRs (`app/dependabot`) unless the user includes them. Confirm the final list with the user before reviewing more than one PR.
- Exclude PRs the reviewer already reviewed on GitHub, whatever the state and whether or not `reviews/pr/` holds a file. Check both surfaces per PR, drop on any hit, and name the dropped PRs:
  ```bash
  gh api repos/gnolang/gno/pulls/<num>/reviews --jq '[.[]|select(.user.login=="<reviewer>")|.state]|join(",")'
  gh api repos/gnolang/gno/pulls/<num>/comments --jq '[.[]|select(.user.login=="<reviewer>")]|length'
  ```
- Exclude every PR whose author is outside the core team, meaning an `author_association` other than `MEMBER`, `OWNER` or `COLLABORATOR`. Name them in the handover as dropped and review one only when the user asks for that number. Read it per PR, since `gh pr list --json` has no such field:
  ```bash
  gh api repos/gnolang/gno/pulls/<num> --jq '.author_association'
  ```
- An outside-contributor PR the user did ask for gets a static danger pass over the raw diff before any review work, nothing executed: build and dependency surface touched (`.github/workflows`, Makefile, `go.mod`, `go.sum`, `package.json`, Dockerfile, `*.sh`); `os/exec`, `net/http`, `net.Dial`, `syscall`, `go:generate`, `go:embed`, Go `unsafe`, base64 or hex decode, environment or credential reads, filesystem writes; Trojan Source (non-ASCII added lines, bidirectional overrides, zero-width characters, homoglyphs). This covers every outside author, not only a first-time one. Record the result per PR in `reviews/BATCH_STATUS.md` and carry any non-malicious risk into that PR's review.
- Write `reviews/BATCH_STATUS.md` before dispatch and update it as agents return: the confirmed scope; dropped PRs grouped by reason; the final set as a table (PR, head sha, last reviewed sha and next round for re-reviews, worktree path, review dir); the resume steps. Commit it with the batch. Parallel-dispatch conflicts and their resolution are recorded here too.

## Parallel dispatch

The subagent prompt names the worktree: "The worktree already exists at `<worktree-path>` with the PR checked out — never `worktree add` or `gh pr checkout`." The parent's single commit reads `review: PRs <a> and <b>`.

## Deep mode

The catalog the core's lens rule names is `skills/invariant-catalog.md`. A large gno PR earns a consensus-impact lens. The commit message may suffix `(deep)`.

## Run tests

- Run the packages the diff reaches, and read CI for the rest. A module-wide suite, `go test ./gnovm/...` or `./gno.land/...`, runs past an hour here and is killed before it reports, so the review ends holding nothing. This overrides the core's *Reproduce the failure* rule to run the project's own test commands: the CI workflow's invocation is the reference for how a package is called, not for how much to run.
- Run named tests, never a sweep. A bare `-run 'TestFiles'` is three minutes and repeating it is the main cost of a round: name the files, `-run 'TestFiles/(switch52|if9).gno$'`, and take a whole-suite claim from the one run that earned it rather than running it again.
- `.gno` packages: `gno test -v ./path/to/package`. When the PR touches the GnoVM or the `gno` tool itself, the core's run-from-source rule reads `go run ./gnovm/cmd/gno test ...` at the worktree root.
- `.go` packages: `go test -v -run 'relevant' ./path/to/package/...`
- `-run` splits its pattern on `/`, one regex per subtest level. A filetest under a subdirectory is `-run 'TestFiles/types/foo.gno$'`; an alternation may never span a `/`, or it silently matches nothing. One `-run` per test when comparing results.
- Example-package tests on a branch that also modifies a stdlib: run `gno test` with `GNOROOT=<worktree-root>`, else new stdlib symbols fail preprocessing (`name X not declared`).
- gno's `Merge Requirements` bot is a commit status, not a check run.
- Live-boot targets here: `contribs/gnodev`, `gnovm/cmd/gno`, `gnovm/pkg/packages`, `gno.land/pkg/gnoweb`. Boot from the worktree and exercise the changed behavior (gnodev plus `curl` for gnoweb; a real external gno workspace, e.g. `github.com/samouraiworld/gnodaokit`, for loader and tooling changes).

## Review the diff

- **Realm security checklist, mandatory for realm code.** A diff touching `examples/gno.land/r/`, `/p/`, or `/e/` walks every quick check in `docs/resources/gno-ai-contract-review.md` before findings are written, plus `docs/resources/gno-interrealm-v2.md` when a finding turns on caller identity.
- **Read both docs from the newest copy, not from the worktree.** While [PR 5871](https://github.com/gnolang/gno/pull/5871) is open its branch carries corrections master does not, and the reviewed branch carries whatever its base pinned.
  ```bash
  git -C gno fetch origin pull/5871/head:pr5871
  git -C gno show pr5871:docs/resources/gno-ai-contract-review.md
  ```
- **Invariant catalog, mandatory.** For a PR touching gno code (the GnoVM, stdlibs, or `.gno` packages and realms), load `skills/invariant-catalog.md`, walk every class against the diff, and confirm coverage before writing the Output. Skip for docs- or tooling-only PRs. For a PR that adds or changes a realm, also walk that file's *Realm audit patterns*; cite the fixture pair when a finding matches a pattern.
- **Gno vs Go comparison.** When the PR contains `.gno` code, write an equivalent Go test to verify behavior parity, run both, note discrepancies, save to the same `tests/` directory.

## Write tests

Core rules apply; the gno test shapes:

- Fill a filetest golden by seeding the `// Realm:` directive with a placeholder line, then `go test -run 'TestFiles/<name>.gno$' -update-golden-tests .` from `gnovm/pkg/gnolang/`; an empty `// Realm:` is stripped, not populated.
- Assert with `// Output:` carrying the correct values, not `// Error:` matching the panic; `// Error:` only when rejection is the correct outcome.
- The `/* Run: */` header (use `/* */`, not `//` per line) must be runnable from a gnolang/gno clone alone:
  ```
  /* Run: from a gno checkout:
  gh pr checkout <N> -R gnolang/gno && git checkout <short-commit-hash>
  curl -fsSL -o gnovm/tests/files/<name>.gno \
    https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/<thousand>xxx/<number>-<short-slug>/<n>-<short-commit-hash>/tests/<name>.gno
  go test -v -run 'TestFiles/<name>.gno$' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/<name>.gno
  */
  ```
  Same shape for `.txtar` tests — `#` comments, destination `gno.land/pkg/integration/testdata/`.
- Pair the bug with its related baseline invariant in one assertion, and ship two `stdout` assertions side by side, active and commented:
  ```
  stdout 'p==q=false q==r=true'   # IS:     bug — cross-tx pointer-identity break
  # stdout 'p==q=true q==r=true'  # SHOULD: parity preserved across persistence
  ```

## Links & citations

- Every `file:line` reference is a dual link: `` [`file:line`](https://github.com/gnolang/gno/blob/<short-sha>/<path>#L<line>) · [↗](../../../../../.worktrees/gno-review-<number>/<path>#L<line>) `` — GitHub blob URL at the reviewed sha plus local worktree `↗`. `.worktrees/` is gitignored, so `[↗]` is dead on GitHub and the blob link is the one that resolves there. Converter for old reviews: `./scripts/convert-review-links.py`; it also recomputes each review's `<status>` on every run.
- comment.md anchor headers append both links, in order: `## <path>:<start>-<end> [gh](<blob-url>) · [↗](../../../../../.worktrees/gno-review-<number>/<path>#L<start>)`. The path stays a bare token, never a link, or the anchor regex rejects the header. The upload script strips everything after the first space, and strips every `[↗](...)` at post time.
- comment.md carries no `Full review:` line, overriding the core's Body format and its Final check. Anything load-bearing goes in the finding or its collapsed block.
- comment.md opens its Body on the finding, and carries no AI marker unless the core's `post as an AI` trigger fired. The marker is the core's, on its own line, and nothing beyond it.
- **Editing a posted review's body needs GraphQL, and strips `[↗]` first.** `PUT /repos/<owner>/<repo>/pulls/<n>/reviews/<id>` answers `404 Not Found` on a review id the list call just returned. Take `node_id` instead and run `updatePullRequestReview`, which leaves every inline comment untouched. Send the same text the upload script would: it drops every `[↗](...)` at post time, so feeding it the raw draft republishes worktree links that 404 for every reader, and the `[gh]` beside each one already resolves.
  ```bash
  gh api repos/<repo>/pulls/<n>/reviews --jq '.[]|select(.user.login=="<login>")|.node_id'
  gh api graphql -f query='mutation($id:ID!,$body:String!){updatePullRequestReview(input:{pullRequestReviewId:$id,body:$body}){pullRequestReview{state url}}}' -f id=<node-id> -f body='<text>'
  ```
- A code comment a `Refactor:` finding proposes runs to three lines at most, whatever the comment it replaces did. The full patch goes under `tests/` and the review file links it.
- Repro blocks open with `# from a local clone of gnolang/gno:` then `gh pr checkout <N> -R gnolang/gno`.

## Output

The metadata block reads `# PR [#<number>](https://github.com/gnolang/gno/pull/<number>): <title>`, keeps the core fields, and adds:

```markdown
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-<number> <short-sha>`
Overview: [visual overview](../overview.html) <— only when the PR directory has an overview.html>
```

## Re-review rounds

For a patch-id-equal base-only move, `./scripts/reanchor-round.py <number> <new-sha>` does the copy: re-runs the gate, copies the latest round's `.md` files into `<n+1>-<new-sha>/`, rewrites sha references, remaps line anchors, flags unmappable ones. Fix flagged anchors from the worktree, add the round note, regenerate nothing else. `overview.html` untouched.

## Calibration

- gno's linter config is `.github/golangci.yml`, `default: none` with an explicit enable list; check it before flagging a style convention.
- The governing document the core's no-critique rule covers is the ADR, and the contribution policy is the `AGENTS.md` ADR requirement plus the commit conventions.

## Rules

- The single push per review goes to this repo (`git@github.com:samouraiworld/gno-agent-workspace.git`) only: `git add reviews/ && git commit -m "review: PR <number>" && git push`. Never push to gnolang/gno.
- Disclosure: a finding exploitable against merged or deployed code (master, consensus paths, a live realm) follows the core disclosure rule; `gno/SECURITY.md` forbids a public issue, and a public fix PR telegraphs the vector just as loudly.

## PR overview (`overview.html`)

Generate `overview.html` for every PR, before the review file, per the core's *Overview*. The format is the delta: HTML here, markdown there. Nothing about the subject exempts a PR — the judgement call this replaces answered "skip" for subjects a reader could not follow.

- Write it at the PR directory root — `reviews/pr/<thousand>xxx/<number>-<slug>/overview.html`, NOT inside the round directory: it explains the PR, not one commit. Single self-contained HTML file, inline CSS/JS, zero external requests, light theme only, generating model in the `<title>` and the visible subtitle.
- Explainer only — zero review state: no verdict, no findings, no reviewed sha, no round references. Exactly one pointer to the review: a `Review files` link to the PR directory tree on GitHub.
- Content — pick what fits: plain-language explanation, request/state/dataflow diagram, decision table, before/after payload or benchmark bars, an interactive simulator mirroring the changed logic, a short Concepts section when the PR hinges on domain concepts. If the page mirrors PR logic in JS, verify the mirror against the PR's own tests before committing and state the result on the page. No emoji.
- Update only when new commits change the PR's own files. Base-only head bumps, new findings, verdict changes, and new rounds never touch it.
- After writing or updating one, open it in the browser (`xdg-open <path>`); skip the open in subagent and batch runs.

## Posting

Post with `./scripts/post-pr-review.py <number> <path-to-comment.md>` instead of raw `gh api`:

- It pre-validates anchors against the PR diff and reports invalid ones — move those into Body, or re-run with `--skip-invalid`. `--dry-run` prints the payload without posting.
- APPROVE needs the `--approve` flag; the script refuses it otherwise.
- After a successful post it writes the URLs back into comment.md itself; commit and push the updated draft.
- The script enforces the core's re-post rule: a draft carrying `Posted:` rewrites the posted review in place, and an anchor with no `[posted]` link aborts it.
- If the author already has a pending (unsubmitted) review on the PR, the script folds the draft's comments into it and submits in place.

## Final check

Two gno items on top of the core list:

- Every `## <path>:<line>` header carries both links, `[gh](...)` then `[↗](...)`, and its path is a bare token.
- No `Full review:` line anywhere in comment.md.
- No code comment proposed by a `Refactor:` finding runs past three lines.
