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
import re
import sys
from pathlib import Path

WORKSPACE = Path(__file__).resolve().parent.parent
REVIEWS = WORKSPACE / "reviews" / "pr"

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

    # Inject "Local worktree" line right under the "Reviewed by:" line, if absent.
    header_added = 0
    if "Local worktree:" not in new_text:
        # Worktree command, runnable from workspace root.
        wt_cmd = f"`git -C gno worktree add .worktrees/gno-review-{pr_num} {sha} && gh -R gnolang/gno pr checkout {pr_num} && (cd .worktrees/gno-review-{pr_num} && gh pr checkout {pr_num} -R gnolang/gno)`"
        # Simpler one-liner: just the worktree creation; user runs `gh pr checkout` inside.
        wt_cmd = (
            f"`git -C gno worktree add .worktrees/gno-review-{pr_num} {sha}` "
            f"(then `gh -R gnolang/gno pr checkout {pr_num}` inside it)"
        )
        # Insert after the "Reviewed by:" line if present, else after "Author:" line.
        for anchor_pattern in (r"^(Reviewed by:.*?)$", r"^(Author:.*?)$"):
            m = re.search(anchor_pattern, new_text, flags=re.MULTILINE)
            if m:
                insertion = f"{m.group(1)}\nLocal worktree: {wt_cmd}"
                new_text = new_text[: m.start()] + insertion + new_text[m.end() :]
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
    args = parser.parse_args()

    if not REVIEWS.is_dir():
        print(f"reviews/pr not found at {REVIEWS}", file=sys.stderr)
        return 1

    md_files = sorted(REVIEWS.glob(args.only))
    if not md_files:
        print(f"no markdown files matched {args.only}")
        return 0

    total_links = 0
    total_headers = 0
    for md in md_files:
        # Only top-level review files (not tests/ or subdirs we want to leave alone).
        # Pattern: reviews/pr/<bucket>/<num>-*/<round>-<sha>/<file>.md
        if len(md.relative_to(REVIEWS).parts) != 4:
            continue
        links, headers = convert_file(md, args.dry_run)
        total_links += links
        total_headers += headers

    verb = "would convert" if args.dry_run else "converted"
    print(f"\n{verb} {total_links} link(s) across {len(md_files)} file(s); "
          f"injected 'Local worktree:' into {total_headers} file(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
