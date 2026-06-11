#!/usr/bin/env python3
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
"""
reanchor-round.py

Mechanical new review round for a PR whose head advanced WITHOUT changing the
PR's own files (e.g. a master merge). Copies the latest round's .md files into
a new round directory, rewrites every sha reference to the new head, and remaps
line anchors (#L<n>, `file:line` labels, comment.md `## path:line` headers)
against the new head using a per-file diff between the two commits.

Anchors that fall inside changed regions cannot be remapped automatically and
are flagged — fix those by reading the worktree. The round note at the top of
the review file is prose; update it by hand after running.

tests/ subdirectories are NOT copied: their headers pin the reviewed sha and
remain valid in the old round.

Usage:
  ./scripts/reanchor-round.py <pr-number> <new-sha> [--dry-run] [--from-round N]

Requires both shas to be present in the gno/ submodule object store (a prior
`gh pr checkout` in the review worktree fetches them).
"""

import argparse
import difflib
import re
import subprocess
import sys
from pathlib import Path

WORKSPACE = Path(__file__).resolve().parent.parent
REVIEWS = WORKSPACE / "reviews" / "pr"
GNO = WORKSPACE / "gno"
SHORT = 9  # round-dir sha length convention: <n>-<9-char-sha>


def die(msg: str) -> "NoReturn":
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(1)


def git(*args: str) -> str:
    r = subprocess.run(["git", "-C", str(GNO), *args],
                       capture_output=True, text=True, timeout=30)
    if r.returncode != 0:
        raise RuntimeError(r.stderr.strip())
    return r.stdout


def resolve_sha(sha: str) -> str:
    try:
        return git("rev-parse", f"{sha}^{{commit}}").strip()
    except RuntimeError as e:
        die(f"sha {sha} not found in gno/ object store ({e}); "
            f"run `gh pr checkout` in the review worktree first")


def patch_id(base_ref: str, head: str) -> str | None:
    """patch-id of the PR's diff against its merge-base — stable across rebases
    and base merges that only shift line numbers."""
    try:
        base = git("merge-base", base_ref, head).strip()
        diff = git("diff", base, head)
        r = subprocess.run(["git", "-C", str(GNO), "patch-id", "--stable"],
                           input=diff, capture_output=True, text=True, timeout=60)
        if r.returncode == 0 and r.stdout.strip():
            return r.stdout.split()[0]
    except Exception:
        pass
    return None


def find_pr_dir(number: int) -> Path:
    hits = [d for b in REVIEWS.iterdir() if b.is_dir()
            for d in b.iterdir() if d.is_dir() and re.match(rf"^{number}-", d.name)]
    if not hits:
        die(f"no review directory for PR #{number} under reviews/pr/")
    if len(hits) > 1:
        die(f"multiple review directories for PR #{number}: {hits}")
    return hits[0]


def rounds_of(pr_dir: Path) -> list[Path]:
    rs = [d for d in pr_dir.iterdir() if d.is_dir() and re.match(r"^\d+-[a-f0-9]+$", d.name)]
    return sorted(rs, key=lambda d: int(d.name.split("-", 1)[0]))


class LineMapper:
    """Maps old-head line numbers to new-head line numbers, per file."""

    def __init__(self, old_sha: str, new_sha: str):
        self.old_sha = old_sha
        self.new_sha = new_sha
        self.maps: dict[str, dict[int, int] | None] = {}

    def _load(self, path: str) -> dict[int, int] | None:
        if path in self.maps:
            return self.maps[path]
        try:
            old = git("show", f"{self.old_sha}:{path}").splitlines()
            new = git("show", f"{self.new_sha}:{path}").splitlines()
        except RuntimeError:
            self.maps[path] = None
            return None
        mapping: dict[int, int] = {}
        sm = difflib.SequenceMatcher(None, old, new, autojunk=False)
        for tag, i1, i2, j1, j2 in sm.get_opcodes():
            if tag == "equal":
                for k in range(i2 - i1):
                    mapping[i1 + k + 1] = j1 + k + 1
        self.maps[path] = mapping
        return mapping

    def map_line(self, path: str, line: int) -> int | None:
        m = self._load(path)
        return m.get(line) if m is not None else None


# [label](https://github.com/gnolang/gno/blob/<sha>/<path>#L..) or [label]((../)+.worktrees/gno-review-<n>/<path>#L..)
LINK_RE = re.compile(
    r"\[(?P<label>[^\]]+)\]\("
    r"(?P<base>https://github\.com/gnolang/gno/blob/[a-f0-9]{7,40}/"
    r"|(?:\.\./)+\.worktrees/gno-review-\d+/)"
    r"(?P<path>[^#)\s]+)"
    r"(?:#L(?P<a>\d+)(?:-L(?P<b>\d+))?)?"
    r"\)"
)

# comment.md anchor headers: ## <path>:<a>[-<b>] [↗](...)
HEADER_RE = re.compile(r"^## (?:SKIP )?(?P<path>\S+):(?P<a>\d+)(?:-(?P<b>\d+))?", re.M)


def reanchor_text(text: str, mapper: LineMapper, flagged: list[str]) -> str:
    def map_pair(path: str, a: str, b: str | None, ctx: str) -> tuple[int, int | None] | None:
        na = mapper.map_line(path, int(a))
        nb = mapper.map_line(path, int(b)) if b else None
        if na is None or (b and nb is None):
            flagged.append(f"{path}:{a}{'-' + b if b else ''} ({ctx})")
            return None
        return na, nb

    def link_repl(m: re.Match) -> str:
        label, base, path, a, b = (m.group("label"), m.group("base"),
                                   m.group("path"), m.group("a"), m.group("b"))
        if not a:
            return m.group(0)
        pair = map_pair(path, a, b, "link")
        if pair is None:
            return m.group(0)
        na, nb = pair
        anchor = f"#L{na}-L{nb}" if nb else f"#L{na}"
        # Rewrite matching line numbers inside the label (`file.go:121-124` form).
        old_label_lines = f"{a}-{b}" if b else a
        new_label_lines = f"{na}-{nb}" if nb else str(na)
        name = path.rsplit("/", 1)[-1]
        label = label.replace(f"{name}:{old_label_lines}", f"{name}:{new_label_lines}")
        return f"[{label}]({base}{path}{anchor})"

    def header_repl(m: re.Match) -> str:
        path, a, b = m.group("path"), m.group("a"), m.group("b")
        pair = map_pair(path, a, b, "comment.md header")
        if pair is None:
            return m.group(0)
        na, nb = pair
        lines = f"{na}-{nb}" if nb else str(na)
        prefix = "## SKIP " if m.group(0).startswith("## SKIP ") else "## "
        return f"{prefix}{path}:{lines}"

    text = LINK_RE.sub(link_repl, text)
    text = HEADER_RE.sub(header_repl, text)
    return text


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("number", type=int, help="PR number")
    ap.add_argument("new_sha", help="new head sha")
    ap.add_argument("--dry-run", action="store_true", help="print, write nothing")
    ap.add_argument("--from-round", type=int, default=None,
                    help="source round number (default: latest)")
    ap.add_argument("--base", default="origin/master",
                    help="PR base ref for the patch-id check (default: origin/master)")
    args = ap.parse_args()

    pr_dir = find_pr_dir(args.number)
    rounds = rounds_of(pr_dir)
    if not rounds:
        die(f"no round directories in {pr_dir}")
    if args.from_round is not None:
        src = next((r for r in rounds if int(r.name.split("-", 1)[0]) == args.from_round), None)
        if src is None:
            die(f"round {args.from_round} not found in {pr_dir}")
    else:
        src = rounds[-1]
    old_short = src.name.split("-", 1)[1]
    next_n = int(rounds[-1].name.split("-", 1)[0]) + 1

    old_full = resolve_sha(old_short)
    new_full = resolve_sha(args.new_sha)
    new_short = new_full[:SHORT]
    if old_full == new_full:
        die("old and new sha are the same commit")

    dst = pr_dir / f"{next_n}-{new_short}"
    if dst.exists():
        die(f"{dst} already exists")

    # Gate: the PR's diff against its base must be the same patch at both heads,
    # otherwise a full re-review is due, not a mechanical re-anchor.
    pid_old = patch_id(args.base, old_full)
    pid_new = patch_id(args.base, new_full)
    if pid_old and pid_new and pid_old == pid_new:
        print(f"patch-id match vs {args.base}: PR content unchanged between heads, "
              "mechanical re-anchor is sound")
    else:
        print(f"WARNING: PR diff vs {args.base} differs between the two heads "
              "(or patch-id failed) — the PR's own content may have changed; "
              "a full re-review round may be due instead.", file=sys.stderr)

    mapper = LineMapper(old_full, new_full)
    mds = sorted(f for f in src.iterdir() if f.is_file() and f.suffix == ".md")
    if not mds:
        die(f"no .md files in {src}")

    print(f"{src.name} -> {dst.name}  (PR #{args.number}, {len(mds)} files)")
    for md in mds:
        text = md.read_text(encoding="utf-8")
        flagged: list[str] = []
        out = text.replace(old_full, new_full).replace(old_short, new_short)
        out = reanchor_text(out, mapper, flagged)
        n_anchors = len(LINK_RE.findall(out)) + len(HEADER_RE.findall(out))
        status = f"{n_anchors} anchors" + (f", {len(flagged)} FLAGGED" if flagged else "")
        print(f"  {md.name}: {status}")
        for f in flagged:
            print(f"    UNMAPPED {f}")
        if not args.dry_run:
            dst.mkdir(parents=True, exist_ok=True)
            (dst / md.name).write_text(out, encoding="utf-8")

    if (src / "tests").is_dir():
        print(f"\nnote: {src.name}/tests/ not copied (pins the old sha by design)")
    print("\nnext: update the round note in the review file, verify flagged anchors, "
          "run ./scripts/build-indexes.sh")
    if args.dry_run:
        print("(dry run: nothing written)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
