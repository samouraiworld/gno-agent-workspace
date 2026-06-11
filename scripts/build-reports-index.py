#!/usr/bin/env python3
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
"""
build-reports-index.py

Build index.html at the repo root: the central page linking every PR review
and its report.html visual explainer (see skills/review.md, "Visual report"
section). Designed to be served via GitHub Pages from the main branch root.

Usage: ./scripts/build-reports-index.py   (from the workspace root)
"""

import html
import re
import subprocess
import sys
from pathlib import Path

REPO_BLOB = "https://github.com/samouraiworld/gno-agent-workspace/blob/main"
GNO_PR = "https://github.com/gnolang/gno/pull"

ROOT = Path(__file__).resolve().parent.parent
REVIEWS = ROOT / "reviews" / "pr"
OUT = ROOT / "index.html"

VERDICT_RE = re.compile(r"\*\*Verdict:\s*([A-Z][A-Z ]*[A-Z])")
TITLE_RE = re.compile(r"^#\s*PR\s*#\d+:\s*(.+)$", re.M)


def verdict_class(verdict: str) -> str:
    if verdict.startswith("APPROVE"):
        return "approve"
    if verdict.startswith("REQUEST"):
        return "request"
    if verdict.startswith("NEEDS"):
        return "discuss"
    if verdict.startswith("CLOSE"):
        return "close"
    return "none"


def git_date(path: Path) -> str:
    try:
        out = subprocess.run(
            ["git", "log", "-1", "--format=%cs", "--", str(path.relative_to(ROOT))],
            cwd=ROOT, capture_output=True, text=True, timeout=30,
        ).stdout.strip()
        return out or ""
    except Exception:
        return ""


def collect():
    prs = []
    for bucket in sorted(REVIEWS.iterdir()):
        if not bucket.is_dir():
            continue
        for prdir in sorted(bucket.iterdir()):
            if not prdir.is_dir():
                continue
            m = re.match(r"^(\d+)-(.+)$", prdir.name)
            if not m:
                continue
            number = int(m.group(1))
            rounds = []
            for rdir in sorted(
                (d for d in prdir.iterdir() if d.is_dir() and re.match(r"^\d+-", d.name)),
                key=lambda d: int(d.name.split("-", 1)[0]),
            ):
                mds = sorted(
                    f for f in rdir.glob("*.md")
                    if f.name != "comment.md" and "_" in f.name
                )
                if not mds:
                    continue
                review_md = mds[0]
                text = review_md.read_text(encoding="utf-8", errors="replace")
                vm = VERDICT_RE.search(text)
                tm = TITLE_RE.search(text)
                rounds.append({
                    "n": int(rdir.name.split("-", 1)[0]),
                    "sha": rdir.name.split("-", 1)[1],
                    "md": review_md.relative_to(ROOT).as_posix(),
                    "verdict": vm.group(1).strip() if vm else "",
                    "title": tm.group(1).strip() if tm else "",
                    "dir": rdir,
                })
            if not rounds:
                continue
            latest = rounds[-1]
            prs.append({
                "number": number,
                "title": latest["title"] or prdir.name.split("-", 1)[1].replace("-", " "),
                "verdict": latest["verdict"],
                "rounds": rounds,
                # One report per PR, at the PR directory root.
                "report": (prdir / "report.html").relative_to(ROOT).as_posix()
                          if (prdir / "report.html").exists() else None,
                "date": git_date(latest["dir"]),
            })
    prs.sort(key=lambda p: p["number"], reverse=True)
    return prs


def render(prs) -> str:
    n_reports = sum(1 for p in prs if p["report"])
    rows = []
    for p in prs:
        e = html.escape
        round_links = [
            f'<a href="{REPO_BLOB}/{e(r["md"])}" title="review file">r{r["n"]} {e(r["sha"][:9])}</a>'
            for r in p["rounds"]
        ]
        # Title opens the report when there is one (blue), the review file otherwise.
        title_href = e(p["report"]) if p["report"] else f'{REPO_BLOB}/{e(p["rounds"][-1]["md"])}'
        title_class = "has-report" if p["report"] else ""
        vclass = verdict_class(p["verdict"])
        vlabel = p["verdict"] or "—"
        rows.append(f"""
    <tr data-search="{e(str(p['number']))} {e(p['title'].lower())} {e(p['verdict'].lower())}">
      <td class="pr"><a href="{GNO_PR}/{p['number']}">#{p['number']}</a></td>
      <td class="title"><a class="{title_class}" href="{title_href}">{e(p['title'])}</a></td>
      <td><span class="v {vclass}">{e(vlabel)}</span></td>
      <td class="rounds">{' · '.join(round_links)}</td>
      <td class="date">{e(p['date'])}</td>
    </tr>""")

    return f"""<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gno PR review reports</title>
<style>
  :root {{
    --bg: #fafafa; --card: #fff; --ink: #1a1a1a; --muted: #6b7280; --line: #e5e7eb;
    --mono: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  }}
  * {{ box-sizing: border-box; }}
  body {{ margin: 0; padding: 2rem 1rem 4rem; background: var(--bg); color: var(--ink);
         font: 15px/1.5 system-ui, -apple-system, "Segoe UI", Roboto, sans-serif; }}
  main {{ max-width: 1080px; margin: 0 auto; }}
  h1 {{ font-size: 1.35rem; margin: 0 0 .25rem; }}
  .meta {{ color: var(--muted); font-size: .88rem; margin-bottom: 1.2rem; }}
  a {{ color: #2563eb; text-decoration: none; }}
  a:hover {{ text-decoration: underline; }}
  input[type=search] {{ width: 100%; max-width: 420px; padding: .5rem .75rem; margin: 0 0 1rem;
       border: 1px solid var(--line); border-radius: 8px; font-size: .9rem; }}
  table {{ border-collapse: collapse; width: 100%; background: var(--card);
           border: 1px solid var(--line); border-radius: 10px; overflow: hidden; }}
  th, td {{ padding: .5rem .7rem; border-bottom: 1px solid var(--line); text-align: left;
            vertical-align: top; font-size: .88rem; }}
  th {{ background: #f3f4f6; font-size: .8rem; letter-spacing: .03em; }}
  td.title {{ max-width: 420px; }}
  td.title a {{ color: inherit; }}
  td.title a:hover {{ color: #2563eb; }}
  td.title a.has-report {{ color: #2563eb; }}
  td.rounds {{ font-family: var(--mono); font-size: .8rem; white-space: nowrap; }}
  td.pr {{ white-space: nowrap; }}
  td.date {{ color: var(--muted); white-space: nowrap; }}
  .v {{ display: inline-block; padding: .05em .55em; border-radius: 999px;
        font-size: .75rem; font-weight: 600; white-space: nowrap; }}
  .v.approve {{ background: #f0fdf4; color: #16a34a; border: 1px solid #bbf7d0; }}
  .v.request {{ background: #fef2f2; color: #dc2626; border: 1px solid #fecaca; }}
  .v.discuss {{ background: #fffbeb; color: #d97706; border: 1px solid #fde68a; }}
  .v.close   {{ background: #f3f4f6; color: #4b5563; border: 1px solid #e5e7eb; }}
  .v.none    {{ background: #f3f4f6; color: #9ca3af; border: 1px solid #e5e7eb; }}
  footer {{ margin-top: 2rem; color: var(--muted); font-size: .8rem; }}
</style>
</head>
<body>
<main>
<h1>gno PR review reports</h1>
<p class="meta">{len(prs)} PRs reviewed · {n_reports} visual reports ·
  <a href="{REPO_BLOB}/reviews/README.md">full index with team coverage</a> ·
  generated by <a href="{REPO_BLOB}/scripts/build-reports-index.py">build-reports-index.py</a></p>
<input type="search" id="q" placeholder="Filter by number, title, verdict..." autocomplete="off">
<table>
  <thead><tr><th>PR</th><th>Title</th><th>Verdict</th><th>Rounds</th><th>Last</th></tr></thead>
  <tbody>{''.join(rows)}
  </tbody>
</table>
<footer>AI-generated review artifacts; verify against the source. Verdicts are the latest round's.</footer>
</main>
<script>
const q = document.getElementById('q');
q.addEventListener('input', () => {{
  const needle = q.value.toLowerCase();
  for (const tr of document.querySelectorAll('tbody tr')) {{
    tr.style.display = tr.dataset.search.includes(needle) ? '' : 'none';
  }}
}});
</script>
</body>
</html>
"""


def main():
    if not REVIEWS.is_dir():
        sys.exit(f"not found: {REVIEWS} (run from the workspace root)")
    prs = collect()
    OUT.write_text(render(prs), encoding="utf-8")
    print(f"wrote {OUT.relative_to(ROOT)} ({len(prs)} PRs)")


if __name__ == "__main__":
    main()
