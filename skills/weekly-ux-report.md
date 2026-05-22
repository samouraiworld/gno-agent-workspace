---
name: weekly-ux-report
description: Generate or update the a/ux (UX team) weekly report. Fetches all PRs and issues with the `a/ux` label from gnolang/gno and produces a categorized markdown report.
argument-hint: "[date expression]"
---

# Weekly UX Report

Generate the a/ux team weekly status report for the `gnolang/gno` repository.

**Input:** `$ARGUMENTS` — optional date expression for the report end-date (default: today). Examples: `weekly-ux-report`, `weekly-ux-report last week`. Parse to YYYY-MM-DD. The report covers a rolling 7-day window ending on that date.

## Scope

- **Repository:** `gnolang/gno`
- **Filter:** all PRs and issues carrying the `a/ux` label
- **Authors:** all (not team-filtered)

## Report sections (in order)

Content categories (1-7) are **omitted if empty**. All other sections always appear.

1. Highlight
2. Gnoweb
3. Documentation
4. Packages
5. GnoVM / TM2
6. Tools
7. Other
8. In Progress
9. Issues Opened (this week)
10. Open Issues (a/ux)
11. Merged
12. NOTE

## Classification rules

For non-draft, open PRs, assign to the **first matching** content category:

1. **Documentation** — title starts with `docs` or `docs:`.
2. **Packages** — title contains `(example`, `(avl)`, `(govdao)`, `(grc20reg)`, `(daokit)`, `(examples)`, or refers to `r/sys/`, `r/docs/` realms.
3. **GnoVM / TM2** — title contains `(gnovm)`, `(tm2)`, `(consensus)`, `(autofile)`, `(bank)`, or touches core VM/TM2 internals.
4. **Gnoweb** — title contains `(gnoweb)` or `gnoweb`.
5. **Tools** — title contains a known tool/contrib name: `gnokey`, `gnokms`, `gnofaucet`, `gnogenesis`, `gnohealth`, `gnokeykc`, `gnomd`, `gnomigrate`, `gnobr`, `gnobro`, `github-bot`, `tx-archive`.
6. **Other** — everything else.

Rules are checked in order 1→6.

## Emoji indicators

Emoji prefixes appear **only in `report.md`**, never in `context.md`. A PR can have multiple prefixes, ordered: `⚠️ 🆕 ✅ 📥 🚫`.

| Emoji | Meaning | Source |
|-------|---------|--------|
| `⚠️` | High priority | `context.md` priority is `high` |
| `🆕` | New this week | PR not in last week's `context.md` |
| `✅` | Approved by core team | `reviewDecision: APPROVED` |
| `📥` | Waiting for first review | `review/triage-pending` label |
| `🚫` | Don't merge | `don't merge` label |

## Steps

### 1. Gather data

Fetch directly via `gh` CLI — no script needed.

```bash
# Open PRs with a/ux label
gh pr list --repo gnolang/gno --label "a/ux" --state open \
  --json number,title,url,author,createdAt,isDraft,labels,reviewDecision,mergeable \
  --limit 200

# Merged PRs with a/ux label (returns all; filter by date in step 4)
gh pr list --repo gnolang/gno --label "a/ux" --state merged \
  --json number,title,url,author,mergedAt,labels \
  --limit 100

# Open issues with a/ux label
gh issue list --repo gnolang/gno --label "a/ux" --state open \
  --json number,title,url,author,createdAt,labels \
  --limit 200
```

Also check for issues opened this week:

```bash
gh issue list --repo gnolang/gno --label "a/ux" --state all \
  --json number,title,url,author,createdAt,labels,state \
  --limit 200
```

Filter the result to only issues with `createdAt >= <start-date>`.

### 2. Load last week's context

Find the most recent report directory:

```bash
ls -td reports/weekly-ux/*/ | head -1
```

Read `context.md` from it. The previous `report.md` is NOT read.

**`🆕` detection:** a PR is "new this week" if its number does not appear in last week's `context.md`.

### 3. Carry forward context

Build the new `context.md` listing **every open PR** from the data.

**Line syntax:** `<number> [highlight|high|medium|low]: [note] - \`<title>\``

- Priority keyword is optional (default: `medium`). `highlight` PRs go to the Highlight section.
- Note after colon is optional. Keep notes short. They appear in parentheses in the report.
- Title suffix is appended for human readability.

**For each open PR**, write a line using:
1. Carried-forward entry from last week's `context.md` (preserving priority and note) — never overwrite manual entries with auto-detected ones
2. Auto-detected note if no manual entry: `isDraft` → `In progress`, `CHANGES_REQUESTED` → `Changes requested`, `don't merge` label → `Don't merge`, `APPROVED` → `Approved`, `review/triage-pending` label → `Waiting for first review`
3. Bare entry (`<number>:`) if nothing to carry or detect

**Ordering:** group with blank lines: `highlight` → `high` → `Approved` → `Changes requested` → `In progress` → `Don't merge` → other annotated → `Waiting for first review` → bare. Within each group, sort by PR number ascending.

**Write the file** to `reports/weekly-ux/YYYY-MM-DD/context.md` (creating the directory if needed) **before** presenting it. Then show the user the file contents and ask if they want to add or edit any annotations or priorities before generating the report. Wait for their response.

### 4. Produce the new report

Using `context.md` and fresh data only. Filter merged PRs to those with `mergedAt` within the 7-day window. Follow this template (content categories omitted if empty):

```markdown
From DD/MM to DD/MM  **: a/ux (UX team)**

> 🆕 New this week · ✅ Approved by core team · 📥 Waiting for first review · 🚫 Don't merge

## Gno Core (/gnolang/gno) — `a/ux` label

**⭐ Highlight**

---

**🌐 PR Waiting for review (Gnoweb)**

---

**📖 PR Waiting for review (Documentation)**

---

**📦 PR Waiting for review (Packages)**

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

---

**🔧 PR Waiting for review (Tools)**

---

**📂 PR Waiting for review (Other)**

---

**🚧 PR In Progress:**

---

**🐛 Issues Opened (this week):**

---

**🐛 Open Issues (a/ux):**

---

**🎉 PR Merged (this week)**

---

**📝 NOTE:**
```

#### Formatting rules

- Sections separated by `---`. Headers are **bold** (not `##`, except `## Gno Core ...`).
- PR lines: `- <emoji prefixes> <title> - <url> - <author> <(context note)>`.
- Context notes in parentheses after author. Don't duplicate emoji-derived status.
- **Ordering within sections:** `⚠️` → `✅` → plain → `🚫` → `📥`. Within each group: fixes before features, features before chores; same tier: older first.
- Highlight entries may use free-text formatting.
- `NOTE` is always left empty — team fills manually.
- Do NOT fabricate PRs. Only include PRs present in the data.

### 5. Save

Write to `reports/weekly-ux/YYYY-MM-DD/report.md` and `context.md` (using the report end-date).

### 6. Present to user

Show the report. Highlight `🆕` PRs and any that disappeared from last week's context.
