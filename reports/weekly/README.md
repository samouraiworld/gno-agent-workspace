# Weekly Reports

Samourai team weekly status reports for gnolang/gno and samouraiworld repos.

## Structure

```
reports/weekly/
  YYYY-MM-DD/
    report.md    # the weekly report
    context.md   # per-PR priority + context annotations
```

## How it works

1. **Run the skill** — `weekly-report` (or `weekly-report last week`, etc.)
2. **Script gathers data** — `scripts/weekly-report.sh` fetches open and merged PRs from GitHub via `gh` API, outputs `data/weekly-report-data.json`
3. **AI generates the report** — reads last week's `context.md` + fresh JSON data, diffs against previous week, produces the new week's draft
4. **Team reviews** — edits the report, fills Highlight and HackenProof sections, edits `context.md`
5. **Check the box** — tick `- [x] Verified` at the top when done

## Context file

`context.md` lists every open PR with optional priority and notes. Format:

```
<PR number> [highlight|high|medium|low]: [note]
```

Priority is an optional word between the PR number and the colon. No priority keyword means default (`medium`). `highlight` PRs go to the Highlight section. Everything after the colon is the context note.

Examples:

```
5169 high: Waiting on core team decision. See also #4950
5314:
5331: Approved
4886 low:
5127: Related to GHSA-m7rp-96x5-hvpx
```

- Every open PR gets a line
- Entries carry forward automatically while the PR stays open, dropped on merge/close
- Some annotations are auto-detected (labels like `don't merge`, approval status, security references) — manual entries take precedence
- PRs are sorted by priority (high → medium → low) within report sections

## Emoji indicators

PRs in the report are prefixed with emoji for at-a-glance status:

| Emoji | Meaning | Source |
|-------|---------|--------|
| `⚠️` | High priority | `context.md` priority is `high` |
| `🆕` | New this week | PR not in last week's `context.md` |
| `✅` | Approved by core team | `reviewDecision: APPROVED` with at least one core team approver |
| `📥` | Waiting for first review | `review/triage-pending` label |
| `🚫` | Don't merge | `don't merge` label |
| `💥` | Merge conflict | `mergeable: "CONFLICTING"` |

A PR can have multiple prefixes, ordered: `⚠️ 🆕 ✅ 📥 🚫 💥`.

## Configuration

Team members and repos are listed at the top of:
- `skills/weekly-report.md` — the AI skill
- `scripts/weekly-report.sh` — the data-gathering script (env vars `TEAM_MEMBERS`, `REPOS`)
