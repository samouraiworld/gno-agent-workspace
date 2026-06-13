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

A draft that already carries a "Posted:" line (written back by a previous
run) is updated in place instead of re-posted: the review body and every
[posted]-linked inline comment are rewritten on GitHub. Anchors without a
[posted] link abort — comments can't be added to an existing review.

If the author already has a pending (unsubmitted) review on the PR, the
draft is folded into it: its comments are added and the review is
submitted in place, since GitHub forbids a second review while one is
pending.
"""

import argparse
import json
import re
import subprocess
import sys

# "<path>:<line>" or "<path>:<start>-<end>" (the text after "## ").
# Anything after a space (e.g. a local [↗](...) IDE link) is ignored.
ANCHOR_RE = re.compile(r"^(\S+):(\d+)(?:-(\d+))?(?:\s.*)?$")
# Local [↗](...) IDE links, with their optional " · " separator.
LOCAL_LINK_RE = re.compile(r"\s*(?:·\s*)?\[↗\]\([^)]*\)")
# "[posted](<url>)" links written back onto anchor headers after a post.
POSTED_RE = re.compile(r"\[posted\]\(([^)]*)\)")
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


def strip_local_links(text):
    """Remove local [↗](...) IDE links from posted content — comment.md keeps
    them for one-click navigation while pruning, but their relative targets
    don't resolve on GitHub. Fence-aware: code blocks are left untouched."""
    out, in_fence = [], False
    for line in text.splitlines():
        if line.lstrip().startswith("```"):
            in_fence = not in_fence
        elif not in_fence:
            line = LOCAL_LINK_RE.sub("", line)
        out.append(line)
    return "\n".join(out)


def parse_comment_md(text):
    """Return (event, body, comments) in GitHub reviews-API shape."""
    m = re.search(r"^Event:\s*(\S+)\s*$", text, re.MULTILINE)
    event = m and m.group(1)
    if event not in EVENTS:
        sys.exit(f"error: missing or invalid 'Event:' line (got {event!r}, "
                 f"want one of {sorted(EVENTS)})")

    body, comments = None, []
    for header, content in sections(text):
        content = strip_local_links(content)
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
        m = POSTED_RE.search(header)
        if m:
            c["_posted"] = m.group(1)
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


def gh_json(path):
    return json.loads(subprocess.run(
        ["gh", "api", path], capture_output=True, text=True, check=True,
    ).stdout)


def gh_graphql(query, **vars):
    cmd = ["gh", "api", "graphql", "-f", f"query={query}"]
    for k, v in vars.items():
        cmd += ["-f", f"{k}={v}"]
    res = subprocess.run(cmd, capture_output=True, text=True, check=True)
    # GraphQL reports failures in an "errors" field with HTTP 200, so
    # check=True alone won't catch them.
    data = json.loads(res.stdout)
    if data.get("errors"):
        sys.exit(f"graphql error: {json.dumps(data['errors'])}")
    return data.get("data")


def find_pending_review(repo, pr):
    """The current user's unsubmitted (PENDING) review on the PR, if any.
    GitHub allows only one per user, so a fresh review POST 422s when one
    exists. Returns {"id", "node_id"} or None."""
    me = gh_json("user")["login"]
    for r in gh_json(f"repos/{repo}/pulls/{pr}/reviews"):
        if r.get("state") == "PENDING" and r.get("user", {}).get("login") == me:
            return {"id": r["id"], "node_id": r["node_id"]}
    return None


def fold_into_pending(pending, event, body, comments, path_md):
    """Add the draft's comments to an existing pending review and submit it
    as one review. REST can't append to a pending review, so the threads go
    in by GraphQL node id; the submit sets the body and the event. Each
    thread's comment URL is captured from the mutation response — submitted
    fold comments come back from REST with null line fields (only the legacy
    diff `position`), which write_back_links can't match on."""
    node = pending["node_id"]
    url_by_anchor = {}
    for c in comments:
        fields = ["pullRequestReviewId: $reviewId", "path: $path",
                  "body: $body", f"line: {c['line']}", f"side: {c['side']}"]
        if "start_line" in c:
            fields += [f"startLine: {c['start_line']}",
                       f"startSide: {c['start_side']}"]
        data = gh_graphql(
            "mutation($reviewId: ID!, $path: String!, $body: String!) { "
            "addPullRequestReviewThread(input: {" + ", ".join(fields) + "}) "
            "{ thread { comments(first: 1) { nodes { url } } } } }",
            reviewId=node, path=c["path"], body=c["body"])
        url = data["addPullRequestReviewThread"]["thread"]["comments"]["nodes"][0]["url"]
        loc = f"{c['start_line']}-{c['line']}" if "start_line" in c else str(c["line"])
        url_by_anchor[f"{c['path']}:{loc}"] = url
    review = gh_graphql(
        "mutation($reviewId: ID!, $body: String!) { submitPullRequestReview("
        f"input: {{pullRequestReviewId: $reviewId, event: {event}, body: $body}}) "
        "{ pullRequestReview { url state } } }",
        reviewId=node, body=body)["submitPullRequestReview"]["pullRequestReview"]
    rewrite_draft_links(path_md, review["url"], url_by_anchor)
    print(f"folded into pending review and submitted: {review['url']}")
    print(f"{event}, added {len(comments)} inline comment(s) over the existing "
          f"pending review; links written back to {path_md}")


def update_posted(repo, pr, review_url, body, comments):
    """Rewrite an already-posted review in place. The REST review-update
    endpoint (PUT .../reviews/{id}) returns 404 even for the review's own
    author, so updates go through GraphQL by node id instead."""
    m = re.search(r"pullrequestreview-(\d+)", review_url)
    if not m:
        sys.exit(f"error: can't parse a review id from 'Posted: {review_url}'")

    missing = [c for c in comments if "_posted" not in c]
    if missing:
        for c in missing:
            start = c.get("start_line")
            loc = f"{start}-{c['line']}" if start else c["line"]
            print(f"  {c['path']}:{loc}", file=sys.stderr)
        sys.exit("aborting: anchors above have no [posted] link — comments "
                 "can't be added to an already-posted review. Remove them "
                 "or post them separately.")

    node = gh_json(f"repos/{repo}/pulls/{pr}/reviews/{m.group(1)}")["node_id"]
    gh_graphql(
        "mutation($id: ID!, $body: String!) { updatePullRequestReview("
        "input: {pullRequestReviewId: $id, body: $body}) "
        "{ clientMutationId } }",
        id=node, body=body)

    for c in comments:
        m = re.search(r"discussion_r(\d+)", c["_posted"])
        if not m:
            sys.exit(f"error: can't parse a comment id from {c['_posted']}")
        cnode = gh_json(f"repos/{repo}/pulls/comments/{m.group(1)}")["node_id"]
        gh_graphql(
            "mutation($id: ID!, $body: String!) { "
            "updatePullRequestReviewComment(input: "
            "{pullRequestReviewCommentId: $id, body: $body}) "
            "{ clientMutationId } }",
            id=cnode, body=c["body"])

    print(f"review updated in place: {review_url}")
    print(f"body + {len(comments)} inline comment(s) rewritten")


def write_back_links(path_md, review_url, posted):
    """Build the anchor -> URL map from REST review comments and write the
    links back. Used by the fresh-POST path, where comments carry line
    numbers; the fold path builds its own map and calls rewrite_draft_links."""
    url_by_anchor = {}
    for c in posted:
        line = c.get("line") or c.get("original_line")
        start = c.get("start_line") or c.get("original_start_line")
        loc = f"{start}-{line}" if start else str(line)
        url_by_anchor[f"{c['path']}:{loc}"] = c["html_url"]
    rewrite_draft_links(path_md, review_url, url_by_anchor)


def rewrite_draft_links(path_md, review_url, url_by_anchor):
    """Record the posted URLs in comment.md: a "Posted:" line under the
    title and a [posted](url) link on each anchor header whose key
    ("<path>:<line>" or "<path>:<start>-<end>") is in url_by_anchor.
    Idempotent — reposting replaces the previous links."""
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
    ap.add_argument("--approve", action="store_true",
                    help="required to post an APPROVE review (human-confirmed)")
    args = ap.parse_args()

    with open(args.comment_md) as f:
        text = f.read()
    event, body, comments = parse_comment_md(text)

    # A "Posted:" line means this draft is live on GitHub: rewrite the
    # posted review instead of creating a duplicate. No --approve gate —
    # the event doesn't change, only the text.
    posted_m = re.search(r"^Posted:\s*(\S+)\s*$", text, re.MULTILINE)
    if posted_m:
        if args.dry_run:
            print(json.dumps({"update": posted_m.group(1), "body": body,
                              "comments": comments}, indent=2))
            return
        update_posted(args.repo, args.pr, posted_m.group(1), body, comments)
        return

    for c in comments:
        c.pop("_posted", None)

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

    # GitHub allows only one pending (unsubmitted) review per user per PR.
    # If the caller already has one (e.g. started in the web UI), a fresh
    # POST 422s — fold the draft into that review and submit it in place.
    pending = find_pending_review(args.repo, args.pr)

    if args.dry_run:
        preview = {"event": event, "body": body, "comments": comments}
        if pending:
            preview = {"fold_into_pending_review": pending["id"], **preview}
        print(json.dumps(preview, indent=2))
        return

    # Approving a PR is a human decision, not an AI one.
    if event == "APPROVE" and not args.approve:
        sys.exit("Event APPROVE requires explicit human confirmation: "
                 "re-run with --approve once the user has approved.")

    if pending:
        fold_into_pending(pending, event, body, comments, args.comment_md)
        return

    payload = {"event": event, "body": body, "comments": comments}
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
