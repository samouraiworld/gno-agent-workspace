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
- **Mergers** (only their approvals count for ✅): `thehowl`, `moul`, `jeronimoalbi`, `gfanton`, `ltzmaxwell`, `sw360cab`, `alexiscolin`, `aeddi`, `zivkovicmilos`, `jaekwon`, `nemanjantic`, `ajnavarro`, `Kouteki`
- **Repos:** `gnolang/gno`, `samouraiworld/gnomonitoring`

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
| ✅ | Approved by core team | `reviewDecision: APPROVED` AND `reviewStats.approvers` has ≥1 **Core team member** |
| 📥 | Waiting for first review | `review/triage-pending` label |
| 🚫 | Don't merge | `don't merge` label |
| 💥 | Merge conflict | `mergeable: "CONFLICTING"` |

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

### 2. Load last week's context

```bash
ls -td reports/weekly/*/ | head -1                  # find latest
./scripts/parse-context.sh <path>/context.md         # read (strips titles + auto-detected statuses)
```

Do NOT read the previous `report.md`. The previous `context.md` is only used for carry-forward priorities and manual notes (Step 3) — **never** for deciding 🆕. 🆕 is computed from `createdAt` against the report window (see Emoji indicators table). This avoids spurious 🆕 flags when the prior report is older than 7 days (skipped weeks).

### 3. Build new context.md

List **every open PR**. Line syntax: `` <number> [highlight|high|medium|low]: [note] - `<title>` ``

- Priority optional (default: `medium`). `highlight` → Highlight section, `high` → ⚠️ emoji.
- Note optional, kept short. Appears in parentheses in report.
- Title suffix (`` - `<title>` ``) always appended for readability.

**Per-PR logic** (in priority order):
1. **Carry forward** from last week — preserve priority and manual note; never overwrite with auto-detected
2. **Auto-detect** if no manual entry (first match):
   - `isDraft` → `In progress`
   - `CHANGES_REQUESTED` → `Changes requested` — **stale rule:** if every user in `changes_requesters` also appears in `reviewRequests`, skip (they were re-requested)
   - `don't merge` label → `Don't merge`
   - `APPROVED` + core approver → `Approved`
   - `review/triage-pending` label → `Waiting for first review`
   - title/body `NEWTENDG-*`/`GHSA-*` → `Related to <ID>`
3. **Bare** (`<number>:`) if nothing matches

**Ordering** (blank line between groups): `highlight` → `high` → `Approved` → `Changes requested` → `In progress` → `Don't merge` → other annotated → `Waiting for first review` → bare. Ascending PR number within groups.

**Save** to `reports/weekly/YYYY-MM-DD/context.md`, present to user, and **wait for edits** before generating the report.

### 4. Produce report.md

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

**🚧 PR In Progress:**

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
- **Ordering within sections:** ⚠️ → ✅ → plain → 🚫 → 📥 → 💥. Conflicting PRs always last, grouped together. Within each group: fixes → features → chores; same tier: older first.
- Highlight entries may use free-text formatting.
- `Quick Intro Context` and `NOTE` left empty — team fills manually.
- Do NOT fabricate PRs.

### 5. Save & present

Write `reports/weekly/YYYY-MM-DD/report.md` and `context.md` (period end-date). Show the report, highlight 🆕 PRs and any that disappeared from last week.
