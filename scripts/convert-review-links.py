#!/usr/bin/env python3
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
"""
convert-review-links.py

Bulk-converts local-worktree links in reviews/pr/<bucket>/<num>-*/<round>-<sha>/*.md
to dual links: primary GitHub URL · suffix ↗ to the local worktree path.

Also injects a "Local worktree" header line that lets a reader rebuild the same
worktree from this repo's root with one command.

Idempotent: re-running on already-converted files is a no-op.

Usage:
  ./scripts/convert-review-links.py [--dry-run] [--only <glob>]
"""

import argparse
import json
import re
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

WORKSPACE = Path(__file__).resolve().parent.parent
REVIEWS = WORKSPACE / "reviews" / "pr"

# (pr_num -> headRefOid) populated lazily by fetch_pr_heads().
PR_HEADS: dict[str, str] = {}
# (pr_num -> N commits ahead of reviewed sha) populated lazily.
PR_AHEAD_COUNT: dict[tuple[str, str], int] = {}


def fetch_pr_head(pr_num: str) -> tuple[str, str | None]:
    """Returns (pr_num, headRefOid or None on failure)."""
    try:
        r = subprocess.run(
            ["gh", "pr", "view", pr_num, "-R", "gnolang/gno", "--json", "headRefOid"],
            capture_output=True, text=True, timeout=20,
        )
        if r.returncode == 0:
            data = json.loads(r.stdout)
            return (pr_num, data.get("headRefOid"))
    except Exception:
        pass
    return (pr_num, None)


def fetch_pr_heads(pr_nums: list[str]) -> None:
    """Populate PR_HEADS in parallel."""
    if not pr_nums:
        return
    with ThreadPoolExecutor(max_workers=8) as ex:
        futures = {ex.submit(fetch_pr_head, n): n for n in pr_nums}
        for f in as_completed(futures):
            n, head = f.result()
            if head:
                PR_HEADS[n] = head


def ahead_count(reviewed_sha: str, pr_num: str) -> int | None:
    """Returns count of commits between reviewed_sha and PR head, or None on failure."""
    head = PR_HEADS.get(pr_num)
    if not head or head.startswith(reviewed_sha):
        return 0
    key = (pr_num, reviewed_sha)
    if key in PR_AHEAD_COUNT:
        return PR_AHEAD_COUNT[key]
    try:
        r = subprocess.run(
            ["git", "-C", str(WORKSPACE / "gno"), "rev-list", "--count",
             f"{reviewed_sha}..{head}"],
            capture_output=True, text=True, timeout=15,
        )
        if r.returncode == 0:
            n = int(r.stdout.strip())
            PR_AHEAD_COUNT[key] = n
            return n
    except Exception:
        pass
    return None

# Matches `[label](<dots>/.worktrees/gno-review-<N>/<path>(#L<lines>)?)`
# <dots> is one or more "../" segments; some reviews have varying depths.
LINK_RE = re.compile(
    r"\[(?P<label>[^\]]+)\]"
    r"\((?P<local>(?:\.\./)+\.worktrees/gno-review-(?P<n>\d+)/(?P<path>[^#)\s]+)"
    r"(?:#L(?P<lstart>\d+)(?:-L(?P<lend>\d+))?)?"
    r")\)"
)


def convert_file(md: Path, dry_run: bool) -> tuple[int, int]:
    """Returns (links_converted, header_added)."""
    pr_num_dir = md.parent.parent.name  # e.g. "5478-validators-duplicate-entries"
    round_dir = md.parent.name           # e.g. "1-922d6d3"

    pr_num_match = re.match(r"^(\d+)-", pr_num_dir)
    sha_match = re.match(r"^\d+-([a-f0-9]{7,40})$", round_dir)
    if not pr_num_match or not sha_match:
        return (0, 0)

    pr_num = pr_num_match.group(1)
    sha = sha_match.group(1)

    text = md.read_text()
    new_text = text
    link_count = 0

    def repl(m: re.Match) -> str:
        nonlocal link_count
        label = m.group("label")
        local = m.group("local")
        path = m.group("path")
        lstart = m.group("lstart")
        lend = m.group("lend")
        n = m.group("n")
        # Idempotency: never re-process the ↗ suffix of an already-converted dual link.
        if label == "↗":
            return m.group(0)
        # Skip if PR number in the local link doesn't match this review's PR
        # (extremely rare cross-references — leave them alone).
        if n != pr_num:
            return m.group(0)
        anchor = ""
        if lstart and lend:
            anchor = f"#L{lstart}-L{lend}"
        elif lstart:
            anchor = f"#L{lstart}"
        github = f"https://github.com/gnolang/gno/blob/{sha}/{path}{anchor}"
        link_count += 1
        return f"[{label}]({github}) · [↗]({local})"

    new_text = LINK_RE.sub(repl, new_text)

    # Inject/update "| Commit: <sha> (<status>)" on the "Reviewed by:" line.
    # Strip any existing " | Commit: ..." suffix first so re-runs refresh it.
    head = PR_HEADS.get(pr_num)
    if head:
        if head.startswith(sha):
            status = "latest"
        else:
            n = ahead_count(sha, pr_num)
            status = f"stale — +{n} commits since" if n else "stale"
        commit_suffix = f" | Commit: `{sha}` ({status})"
        # Match "Reviewed by: ..." line, drop any prior Commit: clause, append fresh one.
        new_text = re.sub(
            r"^(Reviewed by:[^\n]*?)(?:\s*\|\s*Commit:[^\n]*)?$",
            lambda m: m.group(1) + commit_suffix,
            new_text,
            count=1,
            flags=re.MULTILINE,
        )

    # Normalize the "Local worktree:" line to the single-command form.
    # Old form: `... <sha>` (then `gh -R gnolang/gno pr checkout <n>` inside it)
    # New form: `git -C gno worktree add .worktrees/gno-review-<n> <sha>`
    wt_new = f"Local worktree: `git -C gno worktree add .worktrees/gno-review-{pr_num} {sha}`"
    old_wt_re = re.compile(
        r"^Local worktree: `git -C gno worktree add \.worktrees/gno-review-\d+ [a-f0-9]+`"
        r"(?: \(then `gh[^`]+` inside it\))?$",
        flags=re.MULTILINE,
    )
    header_added = 0
    if old_wt_re.search(new_text):
        new_text = old_wt_re.sub(wt_new, new_text)
    elif "Local worktree:" not in new_text:
        # Insert after the "Reviewed by:" line if present, else after "Author:".
        for anchor_pattern in (r"^(Reviewed by:.*?)$", r"^(Author:.*?)$"):
            m = re.search(anchor_pattern, new_text, flags=re.MULTILINE)
            if m:
                new_text = new_text[: m.end()] + f"\n{wt_new}" + new_text[m.end() :]
                header_added = 1
                break

    if new_text != text:
        if dry_run:
            print(f"[would update] {md.relative_to(WORKSPACE)} — "
                  f"{link_count} links, header_added={header_added}")
        else:
            md.write_text(new_text)
            print(f"[updated]      {md.relative_to(WORKSPACE)} — "
                  f"{link_count} links, header_added={header_added}")

    return (link_count, header_added)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true",
                        help="Don't write; print what would change.")
    parser.add_argument("--only", default="**/*.md",
                        help="Glob under reviews/pr (default: **/*.md).")
    parser.add_argument("--no-commit-status", action="store_true",
                        help="Skip the (slow) PR-head fetch + 'Commit:' staleness annotation.")
    args = parser.parse_args()

    if not REVIEWS.is_dir():
        print(f"reviews/pr not found at {REVIEWS}", file=sys.stderr)
        return 1

    md_files = sorted(REVIEWS.glob(args.only))
    if not md_files:
        print(f"no markdown files matched {args.only}")
        return 0

    # Only top-level review files (not tests/ or other subdirs).
    md_files = [md for md in md_files
                if len(md.relative_to(REVIEWS).parts) == 4]

    # Bulk-fetch PR heads up front (parallel) — needed for the Commit: status line.
    if not args.no_commit_status:
        pr_nums: set[str] = set()
        for md in md_files:
            pr_num_dir = md.parent.parent.name
            m = re.match(r"^(\d+)-", pr_num_dir)
            if m:
                pr_nums.add(m.group(1))
        print(f"Fetching head SHAs for {len(pr_nums)} PRs (parallel)...", file=sys.stderr)
        fetch_pr_heads(sorted(pr_nums))
        print(f"  got {len(PR_HEADS)} / {len(pr_nums)} heads", file=sys.stderr)

    total_links = 0
    total_headers = 0
    for md in md_files:
        links, headers = convert_file(md, args.dry_run)
        total_links += links
        total_headers += headers

    verb = "would convert" if args.dry_run else "converted"
    print(f"\n{verb} {total_links} link(s) across {len(md_files)} file(s); "
          f"injected 'Local worktree:' into {total_headers} file(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
