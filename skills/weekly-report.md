---
name: weekly-report
description: Generate or update the Samourai team weekly report. Gathers PR data via script, diffs against last week, and produces the updated markdown.
argument-hint: "[date expression]"
---

# Weekly Report

Generate the Samourai team weekly status report.

**Input:** `$ARGUMENTS` — optional date expression for the report end-date (default: today). Examples: `weekly-report last week`, `weekly-report week of 1st march`. Parse to YYYY-MM-DD. Weeks run Mon–Sun. Pass via `--end-date YYYY-MM-DD`.

## Workflow

Steps in order, detailed below. Two user gates: context.md edits (step 4) and recurrent-conflict flags (*Conflict tracking & Discord ping*) — never generate the report before both clear.

1. *Gather data* — `scripts/weekly-report.sh` → `data/weekly-report-data.json`.
2. *Verify mergers list*.
3. *Load last week's context*.
4. *Build context.md* — present to user, **wait for edits**.
5. *Produce report.md* — re-read context.md from disk first.
6. *Save & present*.
7. *Discord conflict ping* — `discord.md`.

Artifacts land in `reports/weekly/YYYY-MM-DD/` (period end-date): `context.md`, `report.md`, `discord.md`.

## Team & Repos

- **Samourai:** `davd-gzl`, `omarsy`, `mvallenet`, `Villaquiranm`, `WaDadidou`, `zxxma`, `louis14448`, `AmozPay` (keep in sync with `scripts/weekly-report.sh`)
- **Mergers** (only their approvals count for ✅): `thehowl`, `moul`, `jeronimoalbi`, `gfanton`, `ltzmaxwell`, `sw360cab`, `alexiscolin`, `aeddi`, `zivkovicmilos`, `jaekwon`, `nemanjantic`, `ajnavarro`, `Kouteki`, `NotJoon`, `tbruyelle`
- **Repos:** `gnolang/gno`, `samouraiworld/gnomonitoring`

A renamed GitHub handle returns zero PRs with no error, silently dropping that member. On a missing-PR complaint, verify handles via `gh api users/<login>` first. `Villaquiranm` appears as "Miguel" in manual cross-repo entries (same person).

## Classification rules

For non-draft, open gnolang/gno PRs — **first match wins** (1→7):

| # | Category | Match |
|---|----------|-------|
| 1 | **Security** | Title has `fix` + security keyword (gas metering, panics, path traversal, bounds, nil, overflow, type assertions, peer protection, consensus safety, ABCI, RPC hardening). Also title/body can contain `NEWTENDG-*` or `GHSA-*`. |
| 2 | **Documentation** | Title starts with `docs` or `docs:` |
| 3 | **Packages** | Title contains `(example`, `(avl)`, `(govdao)`, `(grc20reg)`, `(daokit)`, `(examples)`, or refers to `r/sys/`, `r/docs/` |
| 4 | **GnoVM / TM2** | Title contains `(gnovm)`, `(tm2)`, `(consensus)`, `(autofile)`, `(bank)`, or core VM/TM2 internals |
| 5 | **Gnoweb** | Title contains `(gnoweb)` or `gnoweb` |
| 6 | **Tools** | Title contains: `gnokey`, `gnokms`, `gnofaucet`, `gnogenesis`, `gnohealth`, `gnokeykc`, `gnomd`, `gnomigrate`, `gnobr`, `gnobro`, `github-bot`, `tx-archive` |
| 7 | **Other** | Everything else |

## Emoji indicators (report.md only)

A PR can have multiple prefixes, ordered: `⚠️ 🆕 ✅ 📥 🚫 💥`. `🚫` and `💥` must always be adjacent.

| Emoji | Meaning | Source |
|-------|---------|--------|
| ⚠️ | High priority | `context.md` priority is `high` |
| 🆕 | New this week | PR's `createdAt` ≥ window start date (the report period's Monday) |
| ✅ | Approved by a merger | ≥1 approver from the **Mergers** list in `reviewStats.approvers` AND (`reviewDecision: APPROVED` OR stale CR per the stale rule in step 4) |
| 📥 | Waiting for first review | `review/triage-pending` label |
| 🚫 | Don't merge | `don't merge` label |
| 💥 | Merge conflict (not rendered on `recurrent-conflict` PRs) | `mergeable: "CONFLICTING"` |

## AI review routing (report.md only)

Our own AI review (under `reviews/pr/`) routes a PR, it is not a trailing marker.

| Verdict | Effect |
|---------|--------|
| `REQUEST CHANGES` | Route PR to **🚧 PR In Progress — Not approved by AI** |
| `NEEDS DISCUSSION` | PR stays in its normal category |
| `APPROVE` (incl. `with nits`/`with caveats`) | PR stays in its normal category |
| no review under `reviews/pr/` | PR stays in its normal category |

In Progress has two subsections: **Not approved by AI** (header links to `reviews/README.md`) holds the non-draft `REQUEST CHANGES` PRs; **Draft** holds every `isDraft` PR. `REQUEST CHANGES` routing wins over every other category (Approved, Waiting for review, etc.): if our AI flagged it ❌, it lands in Not-approved-by-AI regardless of core-review state. Draft outranks it: a PR that is both ❌ and draft goes under Draft, carrying a trailing `(AI: changes requested)` note. Under **Not approved by AI** no per-line AI marker is rendered (the subsection header carries the meaning). Keep core-team/status emoji prefixes (✅/📥/🚫/💥 etc.) everywhere.

Derivation per open PR `<n>`: find `reviews/pr/<bucket>/<n>-<slug>/`, take the highest-numbered round dir `<round>-<commit>/`, read the `**Verdict: ...**` line (older reviews omit the `**`) from the `*.md` inside, normalise to `REQUEST CHANGES` / `NEEDS DISCUSSION` / `APPROVE`. Login matching for approvers is case-insensitive (`notJoon` == `NotJoon`).

## Conflict tracking & Discord ping

A 💥 PR is one of two states:

- **Recurrent** — conflicts on a mechanical / auto-regenerated subject (gas snapshots, `go.mod`/`go.sum`, generated files). Drop `💥`; flag via the `recurrent-conflict` token on the `context.md` note. In `report.md`: no `💥`, trailing ` (expected conflict: <subject>)` marker after author/notes, ordered by remaining emoji tier, not the conflict group. No magnet overlap → not recurrent (keep `💥`). Token stale (PR no longer touches magnet) → drop token, restore `💥`. Confirm flags with user before generating.
- **Stale** — conflicting + `updatedAt` older than 7 days before end-date + not recurrent + not draft. Goes in the Discord ping (step 7).

**Subject detection.** Intersect the PR's changed files with the master hot-file set:

```bash
git -C gno log --since="<~4mo ago>" --name-only --pretty=format: origin/master \
  | grep -v '^$' | sort | uniq -c | sort -rn | awk '$1>=6{print $2}' > hot.txt   # hot files
gh pr view <n> --repo gnolang/gno --json files -q '.files[].path' | sort \
  | comm -12 - <(sort hot.txt)                                                    # PR's hot overlap
```

Map overlap to a tag, first match: `gas` (`*gas*.txtar`, `restart_gas`, `gnokey_gasfee`, `gc.txtar`, `stdlib_restart_compare`, gas-table source) → `go.mod` (`go.mod`/`go.sum`) → `apphash` (`gno.land/pkg/sdk/vm/apphash_*_test.go`) → `testdata` (other integration `.txtar`) → `generated` (`*generated*.go`, `*_string.go`). Hand-written code (VM core, `examples/**.gno`, gnoweb) is a normal conflict, not recurrent. A PR overlapping both a magnet and hand-written code is recurrent only if `git merge-tree` puts every conflict in the magnet. Exact conflicting files: `git merge-tree <master> <pr-head>` in a worktree.

Conflict source of truth is the `💥` set in the generated `report.md` (live `gh pr view` `mergeable` is often `UNKNOWN`). Use live fetch only for `updatedAt` and changed files.

## Steps

### 1. Gather data

```bash
./scripts/weekly-report.sh                          # this week
./scripts/weekly-report.sh --end-date 2025-03-30    # specific week
```

The script verifies team handles before fetching.

Writes `data/weekly-report-data.json` — structure: `repos[].open_prs[]`, `repos[].merged_prs[]`, `repos[].issues_opened[]`.

Key open PR fields: `number`, `title`, `url`, `author`, `createdAt`, `updatedAt`, `isDraft`, `labels[]`, `reviewDecision` (APPROVED/CHANGES_REQUESTED/REVIEW_REQUIRED), `reviewRequests[]`, `body`, `mergeable` (MERGEABLE/CONFLICTING/UNKNOWN), `reviewStats`:
```json
{ "approved": N, "commented": N, "changes_requested": N,
  "approvers": ["user", ...], "changes_requesters": ["user", ...] }
```
Uses last review per author as official status. Merged PRs: `number`, `title`, `url`, `author`, `mergedAt`, `labels`. Issues: `number`, `title`, `url`, `author`, `createdAt`, `state`, `labels`.

### 2. Verify mergers list

```bash
gh pr list --repo gnolang/gno --state merged --limit 200 --json mergedBy --jq '[.[].mergedBy.login] | unique | .[]'
```

For each login: if missing from **Mergers** and not a **Samourai** member, add it to the Mergers line in *Team & Repos*. Surface the diff to the user.

### 3. Load last week's context

Sort by directory name. Never `ls -td` (mtime).

```bash
ls -d reports/weekly/*/ | sort -r | grep -v "/$END_DATE/$" | head -1
./scripts/parse-context.sh <path>/context.md
```

Previous `context.md` is for carry-forward priorities/manual notes only — not for 🆕. If the previous directory is more than 7 days before `END_DATE`, flag it to the user before producing the report.

The Highlight section comes from the user, not from this repo: ask for the block if the request doesn't carry it. Read the previous `report.md` **⭐ Highlight** block only as the fallback when none is supplied (see step 5). Do not rebuild the Highlight from `context.md`.

### 4. Build new context.md

List **every open PR**. Line syntax: `` <number> [highlight|high|medium|low]: [note] - `<title>` ``

- Priority optional (default: `medium`). `high` → ⚠️ emoji. (`highlight` may still tag a line for bookkeeping, but the report's Highlight section comes from the previous `report.md`, not from here — see step 5.)
- Note optional, kept short. Appears in parentheses in report.
- Title suffix (`` - `<title>` ``) always appended for readability.

A note may carry the manual `recurrent-conflict` token (see *Conflict tracking & Discord ping*). Carry forward like any manual note; coexists with the status note (e.g. `Approved, recurrent-conflict`).

**Per-PR logic** (in priority order):
1. **Carry forward** from last week — preserve priority and manual note; never overwrite with auto-detected
2. **Auto-detect** if no manual entry (first match):
   - AI verdict `REQUEST CHANGES` → `In progress` note `AI: changes requested`. Wins over all checks below. (`NEEDS DISCUSSION` does not route here; it stays in its normal category with a `AI: needs discussion` note, appended to whatever note the checks below produce — a draft reads `In progress, AI: needs discussion`.)
   - `isDraft` → `In progress`
   - `CHANGES_REQUESTED` and not stale → `Changes requested`. **Stale rule:** every user in `changes_requesters` is also in `reviewRequests`. Stale CRs fall through to the next checks and are treated as no-CR for the `Approved` check.
   - `don't merge` label → `Don't merge`
   - (`APPROVED` OR stale CR) + ≥1 core approver → `Approved`
   - `review/triage-pending` label → `Waiting for first review`
   - title/body `GHSA-*` → `Related to <ID>` (do not auto-add `NEWTENDG-*` notes)
3. **Bare** (`<number>:`) if nothing matches

**Ordering** (blank line between groups): `highlight` → `high` → `Approved` → `Changes requested` → `In progress` → `Don't merge` → other annotated → `Waiting for first review` → bare. Ascending PR number within groups.

**Save** to `reports/weekly/YYYY-MM-DD/context.md`, present to user, and **wait for edits** before generating the report.

### 5. Produce report.md

First re-read `reports/weekly/YYYY-MM-DD/context.md` from disk, even after approval: the user edits it between steps. The on-disk file is the source of truth for priorities and notes.

Use `context.md` + JSON data. The seven category sections (Security through Other) are omitted when empty; all other sections always appear.

```markdown
Verified by:
- [ ]  David
- [ ]  Ghost
- [ ]  Lours
- [ ]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From DD/MM to DD/MM  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by a merger · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

---

**🛡️ PR Waiting for review (Security)**

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

---

**📖 PR Waiting for review (Documentation)**

---

**📦 PR Waiting for review (Packages)**

---

**🌐 PR Waiting for review (Gnoweb)**

---

**🔧 PR Waiting for review (Tools)**

---

**📂 PR Waiting for review (Other)**

---

**🚧 PR In Progress — [Not approved by AI](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/README.md)**

---

**🚧 PR In Progress — Draft**

---

**🐛 Issues Opened:**

---

**🎉 PR Merged**

---

**🖥️ Validators / Infrastructure Tools:**

---

**📝 NOTE:**
```

#### Formatting rules

- Sections separated by `---`. Headers **bold** (not `##`), except `## Gno Core (/gnolang/gno)`.
- PR lines: `- <emoji prefixes> <title> - <url> - <author> <(context note)>`
- Context notes in parentheses after author. Don't duplicate emoji-derived status.
- Recurrent-conflict PRs render per *Conflict tracking & Discord ping*: no `💥`, trailing ` (expected conflict: <subject>)`, ordered by remaining tier.
- AI `REQUEST CHANGES` PRs and drafts route to the In Progress subsections per *AI review routing*; no per-line AI marker.
- **Ordering within sections:** ⚠️ → ✅ → plain → 🚫 → 📥 → 💥. Conflicting PRs always last, grouped together. Within each group: fixes → features → chores; same tier: older first.
- **In Progress subsections** (**Not approved by AI**, **Draft**) order by emoji tier ⚠️ → ✅ → plain → 💥 → 🚫 (each line assigned to its highest tier). Within each tier: fixes → features → chores, older first.
- **Highlight section:** the user supplies the block each week (curated outside this repo); ask for it if not given. Reproduce the entries verbatim, including merged and closed ones. Refresh only the emoji prefixes from current JSON; never add, drop, or reorder entries, and never rewrite a title. Falls back to the previous `report.md`'s **⭐ Highlight** block when the user supplies nothing. `context.md` `highlight:` lines are not a source for this section. Highlight entries may use free-text formatting.
- `Quick Intro Context` and `NOTE` left empty — team fills manually.
- Do NOT fabricate PRs.

### 6. Save & present

Write `reports/weekly/YYYY-MM-DD/report.md` and `context.md` (period end-date). Show the report, highlight 🆕 PRs and any that disappeared from last week.

### 7. Discord conflict ping

Write `reports/weekly/YYYY-MM-DD/discord.md` — copy-paste block for Discord. Lists the **stale** conflicts per *Conflict tracking & Discord ping*.

Format (plain markdown, no per-line emoji):

```markdown
**Conflicting PRs to rebase (no activity for +1 week)**

Please rebase, or move to draft if paused. Recurrent conflicts are not listed.

- #<number> <title> - <author> - <url>
...
```

English, simple. Order oldest `updatedAt` first. Empty set → header plus `None.`. Present block; note count of recurrent/draft PRs excluded.
