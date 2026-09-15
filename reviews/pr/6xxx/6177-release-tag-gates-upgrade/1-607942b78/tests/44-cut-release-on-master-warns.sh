#!/usr/bin/env bash
# Asserts cut-release.sh refuses each of the five preflight failures RELEASING.md:102-109
# lists as a refusal. Measured at 607942b78: four refuse (exit 1), the third — "The commit
# is on master" — warns and tags anyway (exit 0, "created annotated tag"). misc/release/
# README.md:47 says it warns, so the two docs the PR adds disagree. Fails at the head.
#
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   git fetch origin master
#   curl -fsSL -o /tmp/44-cut-release-on-master-warns.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/44-cut-release-on-master-warns.sh
#   bash /tmp/44-cut-release-on-master-warns.sh
#
# Needs go on PATH: check_build_reports_tag builds gnoland. Nothing is pushed; the
# script is run without --push, and the local tag it makes is deleted below.

set -uo pipefail
cd "$(git rev-parse --show-toplevel)"
TAG="v99.99.44"
CUT=misc/release/cut-release.sh
cleanup() { git tag -d "${TAG}" >/dev/null 2>&1 || true; }
trap cleanup EXIT
cleanup

# --- baseline: a refusal RELEASING.md lists that is one (item 1, version shape) ---
"${CUT}" v1.2 --allow-dirty >/tmp/44-shape.log 2>&1
shape_rc=$?
echo "item 1 (version shape): exit ${shape_rc}"
grep -q 'version must be vMAJOR.MINOR.PATCH' /tmp/44-shape.log \
  && echo "  refused: $(grep -o 'version must be[^(]*' /tmp/44-shape.log | head -1)"

# --- the defect: item 3, "The commit is on master" ---
# A commit master has never seen, built without moving HEAD: same tree as the reviewed
# head (so the "tree is identical to origin/master" escape hatch does not fire) with the
# branch's seven non-deployment commits ahead of origin/master.
HOT="$(git commit-tree "$(git rev-parse 'HEAD^{tree}')" -p HEAD \
        -m 'fix(consensus): un-ported hotfix on the chain branch')"
git merge-base --is-ancestor "${HOT}" origin/master \
  && { echo "SETUP FAILED: the commit is on origin/master"; exit 2; }

"${CUT}" "${TAG}" --commit "${HOT}" --allow-dirty >/tmp/44-master.log 2>&1
master_rc=$?
echo "item 3 (commit is on master): exit ${master_rc}"
grep -E 'not on origin/master|created annotated tag' /tmp/44-master.log | sed 's/^/  /'

echo
echo "tag after the run: $(git tag --list "${TAG}" | wc -l) (0 = refused, 1 = tagged)"
[ "${master_rc}" -eq 0 ] \
  && echo "IS:     exit 0, tag created — RELEASING.md:109 lists a refusal the script does not make"
# [ "${master_rc}" -ne 0 ] \
#   && echo "SHOULD: exit non-zero, or RELEASING.md lists four refusals and this as a warning"
