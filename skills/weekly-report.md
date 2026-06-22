---
name: weekly-report
description: Generate or update the Samourai team weekly report. Gathers PR data via script, diffs against last week, and produces the updated markdown.
argument-hint: "[date expression]"
---

# Weekly Report

Generate the Samourai team weekly status report.

**Input:** `$ARGUMENTS` — optional date expression for the report end-date (default: today). Examples: `weekly-report last week`, `weekly-report week of 1st march`. Parse to YYYY-MM-DD. Weeks run Mon–Sun. Pass via `--end-date YYYY-MM-DD`.

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
| ✅ | Approved by core team | ≥1 core approver in `reviewStats.approvers` AND (`reviewDecision: APPROVED` OR stale CR per stale-rule) |
| 📥 | Waiting for first review | `review/triage-pending` label |
| 🚫 | Don't merge | `don't merge` label |
| 💥 | Merge conflict | `mergeable: "CONFLICTING"` |

## AI review routing (report.md only)

Our own AI review (under `reviews/pr/`) routes a PR, it is not a trailing marker.

| Verdict | Effect |
|---------|--------|
| `REQUEST CHANGES` | Route PR to **🚧 PR In Progress — Not approved by AI** |
| `NEEDS DISCUSSION` | PR stays in its normal category |
| `APPROVE` (incl. `with nits`/`with caveats`) | PR stays in its normal category |
| no review under `reviews/pr/` | PR stays in its normal category |

In Progress has two subsections: **Not approved by AI** (header links to `reviews/README.md`) holds the `REQUEST CHANGES` PRs; **Draft** holds the `isDraft` PRs. `REQUEST CHANGES` routing wins over every other category (Approved, Waiting for review, etc.): if our AI flagged it ❌, it lands in Not-approved-by-AI regardless of core-review state. A PR that is both ❌ and draft goes under Not approved by AI. No per-line AI marker is rendered (the subsection header carries the meaning); keep core-team/status emoji prefixes (✅/📥/🚫/💥 etc.).

Derivation per open PR `<n>`: find `reviews/pr/<bucket>/<n>-<slug>/`, take the highest-numbered round dir `<round>-<commit>/`, read the `**Verdict: ...**` line (older reviews omit the `**`) from the `*.md` inside, normalise to `REQUEST CHANGES` / `NEEDS DISCUSSION` / `APPROVE`. Login matching for approvers is case-insensitive (`notJoon` == `NotJoon`).

## Steps

### 1. Gather data

```bash
./scripts/weekly-report.sh                          # this week
./scripts/weekly-report.sh --end-date 2025-03-30    # specific week
```

The script verifies team handles before fetching.

Writes `data/weekly-report-data.json` — structure: `repos[].open_prs[]`, `repos[].merged_prs[]`, `repos[].issues_opened[]`.

Key open PR fields: `number`, `title`, `url`, `author`, `createdAt`, `isDraft`, `labels[]`, `reviewDecision` (APPROVED/CHANGES_REQUESTED/REVIEW_REQUIRED), `reviewRequests[]`, `body`, `mergeable` (MERGEABLE/CONFLICTING/UNKNOWN), `reviewStats`:
```json
{ "approved": N, "commented": N, "changes_requested": N,
  "approvers": ["user", ...], "changes_requesters": ["user", ...] }
```
Uses last review per author as official status. Merged PRs: `number`, `title`, `url`, `author`, `mergedAt`, `labels`. Issues: `number`, `title`, `url`, `author`, `createdAt`, `state`, `labels`.

### 1a. Verify mergers list

```bash
gh pr list --repo gnolang/gno --state merged --limit 200 --json mergedBy --jq '[.[].mergedBy.login] | unique | .[]'
```

For each login: if missing from **Mergers** above and not a **Samourai** member, add it to line 16. Surface the diff to the user.

### 2. Load last week's context

Sort by directory name. Never `ls -td` (mtime).

```bash
ls -d reports/weekly/*/ | sort -r | grep -v "/$END_DATE/$" | head -1
./scripts/parse-context.sh <path>/context.md
```

Previous `context.md` is for carry-forward priorities/manual notes only — not for 🆕. If the previous directory is more than 7 days before `END_DATE`, flag it to the user before producing the report.

### 3. Build new context.md

List **every open PR**. Line syntax: `` <number> [highlight|high|medium|low]: [note] - `<title>` ``

- Priority optional (default: `medium`). `highlight` → Highlight section, `high` → ⚠️ emoji.
- Note optional, kept short. Appears in parentheses in report.
- Title suffix (`` - `<title>` ``) always appended for readability.

**Per-PR logic** (in priority order):
1. **Carry forward** from last week — preserve priority and manual note; never overwrite with auto-detected
2. **Auto-detect** if no manual entry (first match):
   - AI verdict `REQUEST CHANGES` → `In progress` note `AI: changes requested`. Wins over all checks below. (`NEEDS DISCUSSION` does not route here; it stays in its normal category with a `AI: needs discussion` note.)
   - `isDraft` → `In progress`
   - `CHANGES_REQUESTED` and not stale → `Changes requested`. **Stale rule:** every user in `changes_requesters` is also in `reviewRequests`. Stale CRs fall through to the next checks and are treated as no-CR for the `Approved` check.
   - `don't merge` label → `Don't merge`
   - (`APPROVED` OR stale CR) + ≥1 core approver → `Approved`
   - `review/triage-pending` label → `Waiting for first review`
   - title/body `GHSA-*` → `Related to <ID>` (do not auto-add `NEWTENDG-*` notes)
3. **Bare** (`<number>:`) if nothing matches

**Ordering** (blank line between groups): `highlight` → `high` → `Approved` → `Changes requested` → `In progress` → `Don't merge` → other annotated → `Waiting for first review` → bare. Ascending PR number within groups.

**Save** to `reports/weekly/YYYY-MM-DD/context.md`, present to user, and **wait for edits** before generating the report.

### 4. Produce report.md

First re-read `reports/weekly/YYYY-MM-DD/context.md` from disk, even after approval: the user edits it between steps. The on-disk file is the source of truth for priorities and notes.

Use `context.md` + JSON data. Content categories (2-8) omitted if empty; all other sections always appear.

```markdown
Verified by:
- [ ]  Amoz
- [ ]  David
- [ ]  Ghost
- [ ]  Lours
- [ ]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From DD/MM to DD/MM  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by core team · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

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
- PR lines: `- <emoji prefixes> <title> - <url> - <author> <(context note)> <🤖 AI marker>`
- Context notes in parentheses after author. Don't duplicate emoji-derived status.
- AI ❌ PRs are routed to the **In Progress — Not approved by AI** subsection (see "AI review routing"); drafts go to **In Progress — Draft**. No per-line AI marker is rendered.
- **Ordering within sections:** ⚠️ → ✅ → plain → 🚫 → 📥 → 💥. Conflicting PRs always last, grouped together. Within each group: fixes → features → chores; same tier: older first.
- **In Progress subsections** (**Not approved by AI**, **Draft**) order by emoji tier ⚠️ → ✅ → plain → 💥 → 🚫 (each line assigned to its highest tier). Within each tier: fixes → features → chores, older first.
- Highlight entries may use free-text formatting.
- `Quick Intro Context` and `NOTE` left empty — team fills manually.
- Do NOT fabricate PRs.

### 5. Save & present

Write `reports/weekly/YYYY-MM-DD/report.md` and `context.md` (period end-date). Show the report, highlight 🆕 PRs and any that disappeared from last week.
