#!/usr/bin/env bash
# Asserts cut-release.sh classifies v1.3.0 cut after v1.3.0-rc.1 as a coordinated
# MINOR upgrade. Measured at 607942b78: newest_release_tag returns the rc, classify
# compares major.minor only, and prints "PATCH: no validator coordination needed"
# with no --halt-height hint; the feat!: safety net misses too. Fails at the head.
#
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78
#   curl -fsSL -o /tmp/28-cut-release-previous-prerelease.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/28-cut-release-previous-prerelease.sh
#   bash /tmp/28-cut-release-previous-prerelease.sh "$(git rev-parse --show-toplevel)"
set -uo pipefail

CLONE="${1:-$(git rev-parse --show-toplevel)}"
SCRIPT="${CLONE}/misc/release/cut-release.sh"
[[ -f ${SCRIPT} ]] || { echo "not a 6177 checkout: ${SCRIPT} missing"; exit 2; }

# A repo shaped like the release RELEASING.md describes: last release v1.2.0, a
# consensus-breaking commit, then the release candidate for it, then a fix.
REPO="$(mktemp -d)"; trap 'rm -rf "${REPO}"' EXIT
git init -q "${REPO}"; git -C "${REPO}" config user.email r@e.l; git -C "${REPO}" config user.name r
echo 1 > "${REPO}/f"; git -C "${REPO}" add f; git -C "${REPO}" commit -qm "chore: v1.2.0"
git -C "${REPO}" tag -a v1.2.0 -m r
echo 2 >> "${REPO}/f"; git -C "${REPO}" commit -qam "feat!: bump the consensus protocol version"
git -C "${REPO}" tag -a v1.3.0-rc.1 -m r
echo 3 >> "${REPO}/f"; git -C "${REPO}" commit -qam "fix: an rc regression"

# The script's own two functions, lifted verbatim, plus the helpers they print with.
info() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*"; }
eval "$(sed -n '/^newest_release_tag() {/,/^}/p' "${SCRIPT}")"
eval "$(sed -n '/^classify() {/,/^}/p' "${SCRIPT}")"

REPO_ROOT="${REPO}" VERSION="v1.3.0" COMMIT="HEAD"
echo "newest_release_tag: $(newest_release_tag)"

# RELEASING.md:77 says pre-release tags "sort below the release they lead to".
git -C "${REPO}" tag -a v1.3.0 -m r >/dev/null
echo "sort -v:refname with the release present: $(git -C "${REPO}" tag --list 'v*' --sort=-v:refname | tr '\n' ' ')"
git -C "${REPO}" tag -d v1.3.0 >/dev/null

echo "--- default (--previous not passed) ---"
OUT="$(PREVIOUS='' classify 2>&1)"; echo "${OUT}"
echo "--- --previous v1.2.0, the last actual release ---"
PREVIOUS=v1.2.0 classify 2>&1 | grep -E 'MINOR|MAJOR|PATCH|halt-height'

fail=0
grep -q 'PATCH: no validator coordination needed' <<<"${OUT}" \
  && { echo "IS:     classified PATCH, no coordination, no --halt-height"; fail=1; } \
  || echo "SHOULD: not PATCH"
# grep -q 'MINOR bump: coordinated upgrade' <<<"${OUT}" || fail=1   # SHOULD: once --previous defaults to the newest *release*
grep -q 'marked breaking' <<<"${OUT}" \
  || echo "IS:     the feat!: net misses too — it only scans v1.3.0-rc.1..HEAD"

exit "${fail}"
