#!/usr/bin/env python3
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
"""
post-pr-review.py

Post a GitHub PR review from a comment.md draft (see skills/review.md,
"GitHub review draft" section).

Usage:
    ./scripts/post-pr-review.py <pr-number> <path-to-comment.md> [--repo OWNER/NAME]
                                [--dry-run] [--skip-invalid]

comment.md format:
    # Review: PR #<number>
    Event: APPROVE | REQUEST_CHANGES | COMMENT

    ## Body
    <review body, posted as the top-level review comment>

    ## <path>:<line>
    <inline comment body>

    ## <path>:<start>-<end>
    <inline comment body, multi-line range>

    ## SKIP <path>:<line>
    <pruned by the reviewer — never posted>

Anchors are validated against the PR diff before posting: the GitHub API
only accepts inline comments on lines present in the diff (added or
context, RIGHT side; the web UI is more permissive, the API is not).
Invalid anchors abort the upload unless --skip-invalid.
"""

import argparse
import json
import re
import subprocess
import sys

# "<path>:<line>" or "<path>:<start>-<end>" (the text after "## ").
# Anything after a space (e.g. a local [↗](...) IDE link) is ignored.
ANCHOR_RE = re.compile(r"^(\S+):(\d+)(?:-(\d+))?(?:\s.*)?$")
# Unified-diff hunk header "@@ -a,b +c,d @@" — capture c, the first
# line number of the hunk on the NEW (RIGHT) side of the diff.
HUNK_RE = re.compile(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@")
EVENTS = {"APPROVE", "REQUEST_CHANGES", "COMMENT"}


def sections(text):
    """Yield (header, content) per "## " section, header without the "## ".
    Fence-aware: a "## " line inside a ``` code block is content.
    Lines before the first header (title, Event:) are dropped."""
    header, lines, in_fence = None, [], False
    for line in text.splitlines():
        if line.lstrip().startswith("```"):
            in_fence = not in_fence
        if not in_fence and line.startswith("## "):
            if header is not None:
                yield header, "\n".join(lines).strip()
            header, lines = line[3:].strip(), []
        elif header is not None:
            lines.append(line)
    if header is not None:
        yield header, "\n".join(lines).strip()


def parse_comment_md(text):
    """Return (event, body, comments) in GitHub reviews-API shape."""
    m = re.search(r"^Event:\s*(\S+)\s*$", text, re.MULTILINE)
    event = m and m.group(1)
    if event not in EVENTS:
        sys.exit(f"error: missing or invalid 'Event:' line (got {event!r}, "
                 f"want one of {sorted(EVENTS)})")

    body, comments = None, []
    for header, content in sections(text):
        if header == "Body":
            body = content
            continue
        # "## SKIP <path>:<line>" — pruned by the reviewer, don't post.
        if header.startswith("SKIP "):
            print(f"skipping (marked SKIP): {header[5:].split()[0]}",
                  file=sys.stderr)
            continue
        m = ANCHOR_RE.match(header)
        if not m:
            sys.exit(f"error: unrecognized section header: '## {header}'\n"
                     "expected '## Body' or '## <path>:<line>[-<end>]'")
        path, start, end = m.groups()
        # The API wants a single line in "line"; for a range, "line" is
        # the LAST line and "start_line" the first.
        c = {"path": path, "side": "RIGHT", "line": int(end or start),
             "body": content}
        if end:
            c["start_line"], c["start_side"] = int(start), "RIGHT"
        comments.append(c)
    if body is None:
        sys.exit("error: missing '## Body' section")
    return event, body, comments


def diff_right_lines(repo, pr):
    """Map path -> set of RIGHT-side line numbers present in the PR diff
    (added and context lines), by replaying the NEW-side line numbering
    of the unified diff hunk by hunk."""
    diff = subprocess.run(
        ["gh", "pr", "diff", str(pr), "-R", repo],
        capture_output=True, text=True, check=True,
    ).stdout
    lines_by_path = {}
    path = None      # file currently being walked (None = deleted file)
    new_line = None  # NEW-side number of the next content line, None between hunks
    for line in diff.splitlines():
        if line.startswith("+++ "):
            # "+++ b/<path>" names the new file; "+++ /dev/null" is a
            # deletion, which has no RIGHT side to comment on.
            path = None if line == "+++ /dev/null" else line[4:].removeprefix("b/")
            new_line = None
        elif line.startswith("@@"):
            # Hunk header: restart the NEW-side counter at its '+c' value.
            m = HUNK_RE.match(line)
            new_line = int(m.group(1)) if m else None
        elif new_line is not None and path is not None:
            # '+' (added) and ' ' (context) lines exist on the RIGHT side:
            # record and advance. A fully empty line is a context line
            # whose trailing space was stripped. '-' lines belong to the
            # LEFT side: don't advance new_line.
            if line.startswith(("+", " ")) or line == "":
                lines_by_path.setdefault(path, set()).add(new_line)
                new_line += 1
    return lines_by_path


def write_back_links(path_md, review_url, posted):
    """Record the posted URLs in comment.md: a "Posted:" line under the
    title and a [posted](url) link on each anchor header. Idempotent —
    reposting replaces the previous links."""
    url_by_anchor = {}
    for c in posted:
        line = c.get("line") or c.get("original_line")
        start = c.get("start_line") or c.get("original_start_line")
        loc = f"{start}-{line}" if start else str(line)
        url_by_anchor[f"{c['path']}:{loc}"] = c["html_url"]

    out, in_fence = [], False
    for line in open(path_md).read().splitlines():
        if line.lstrip().startswith("```"):
            in_fence = not in_fence
        if not in_fence:
            if line.startswith("Posted: "):
                continue  # replaced under the title below
            if line.startswith("# Review:"):
                out += [line, f"Posted: {review_url}"]
                continue
            if line.startswith("## ") and line.rstrip() != "## Body":
                line = re.sub(r" \[posted\]\(\S+\)", "", line)
                m = ANCHOR_RE.match(line[3:].strip())
                if m:
                    path, start, end = m.groups()
                    key = f"{path}:{start}-{end}" if end else f"{path}:{start}"
                    if key in url_by_anchor:
                        line += f" [posted]({url_by_anchor[key]})"
        out.append(line)
    with open(path_md, "w") as f:
        f.write("\n".join(out) + "\n")


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[1])
    ap.add_argument("pr", type=int)
    ap.add_argument("comment_md")
    ap.add_argument("--repo", default="gnolang/gno")
    ap.add_argument("--dry-run", action="store_true",
                    help="print the JSON payload without posting")
    ap.add_argument("--skip-invalid", action="store_true",
                    help="drop comments with anchors outside the diff instead of aborting")
    args = ap.parse_args()

    with open(args.comment_md) as f:
        event, body, comments = parse_comment_md(f.read())

    # Pre-flight: check every anchor against the diff so a single bad
    # line number doesn't make GitHub reject the whole review upload.
    valid = diff_right_lines(args.repo, args.pr)

    def in_diff(c):
        lines = valid.get(c["path"], set())
        return c["line"] in lines and c.get("start_line", c["line"]) in lines

    invalid = [c for c in comments if not in_diff(c)]
    if invalid:
        print("anchors outside the PR diff (GitHub will reject them):",
              file=sys.stderr)
        for c in invalid:
            start = c.get("start_line")
            loc = f"{start}-{c['line']}" if start else c["line"]
            print(f"  {c['path']}:{loc}", file=sys.stderr)
        if not args.skip_invalid:
            sys.exit("aborting: move these into the Body section, fix the "
                     "line numbers, or re-run with --skip-invalid")
        comments = [c for c in comments if in_diff(c)]
        print(f"--skip-invalid: posting the {len(comments)} valid comment(s)",
              file=sys.stderr)

    payload = {"event": event, "body": body, "comments": comments}
    if args.dry_run:
        print(json.dumps(payload, indent=2))
        return

    res = subprocess.run(
        ["gh", "api", f"repos/{args.repo}/pulls/{args.pr}/reviews",
         "--input", "-"],
        input=json.dumps(payload), capture_output=True, text=True, check=True,
    )
    review = json.loads(res.stdout)
    posted = json.loads(subprocess.run(
        ["gh", "api", "--paginate",
         f"repos/{args.repo}/pulls/{args.pr}/reviews/{review['id']}/comments"],
        capture_output=True, text=True, check=True,
    ).stdout)
    write_back_links(args.comment_md, review["html_url"], posted)
    print(f"review posted: {review['html_url']}")
    print(f"{event}, {len(posted)} inline comment(s); links written back "
          f"to {args.comment_md}")


if __name__ == "__main__":
    main()
