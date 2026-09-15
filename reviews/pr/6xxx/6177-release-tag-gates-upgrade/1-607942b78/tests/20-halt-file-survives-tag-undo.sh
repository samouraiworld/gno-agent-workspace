#!/usr/bin/env bash
# 20-halt-file-survives-tag-undo.sh — misc/release/README.md:30 says "the tag
# is trivially undone with `git tag -d`", but --halt-height writes a .gno file
# into the worktree *before* the tag is created, and `git tag -d` only removes
# the tag. Asserts the current (undocumented) state: after `git tag -d` the
# halt directory is still untracked, and the next cut-release.sh invocation
# dies at check_worktree with "worktree is dirty". Fails at head 607942b78 in
# the sense that the doc's claim does not hold for a --halt-height dry run.
#
# Run: from a local clone of gnolang/gno, at the clone root:
#   gh pr checkout 6177 -R gnolang/gno
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   curl -fsSL -o /tmp/20-halt-file-survives-tag-undo.sh \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/20-halt-file-survives-tag-undo.sh
#   bash /tmp/20-halt-file-survives-tag-undo.sh
#
# Optional: $1 overrides the clone root (default: a detached worktree at the
# pinned commit, cleaned up on exit).

set -uo pipefail

ROOT="${1:-}"
CLEANUP_WORKTREE=0
if [[ -z "${ROOT}" ]]; then
	ROOT="$(mktemp -d)"
	git worktree add --detach "${ROOT}" 607942b78fa4fdf6f378fecce32bc1d1d984ab8e >/dev/null 2>&1 \
		|| { echo "SKIP: could not create worktree at 607942b78"; exit 0; }
	CLEANUP_WORKTREE=1
fi
cleanup() {
	[[ ${CLEANUP_WORKTREE} -eq 1 ]] && git worktree remove --force "${ROOT}" >/dev/null 2>&1
}
trap cleanup EXIT

VERSION="v1.3.0-readmecheck"
DIR="${ROOT}/misc/deployments/mainnet.gno.land/transactions/migration/halt-${VERSION}"

(cd "${ROOT}" && misc/release/cut-release.sh "${VERSION}" --commit HEAD \
	--halt-height 120000 --allow-dirty >/dev/null 2>&1)

wrote=0
[[ -f "${DIR}/halt_${VERSION//[.-]/_}.gno" ]] && wrote=1

(cd "${ROOT}" && git tag -d "${VERSION}" >/dev/null 2>&1)

dirty_after_undo="$(cd "${ROOT}" && git status --porcelain)"

# The exact command the README's "trivially undone" claim implies is safe to
# repeat: cut-release.sh again for a coordinated upgrade.
rerun_output="$(cd "${ROOT}" && misc/release/cut-release.sh v1.3.0-readmecheck2 \
	--commit HEAD --halt-height 120000 2>&1)"
rerun_died_dirty=0
grep -q "worktree is dirty" <<<"${rerun_output}" && rerun_died_dirty=1

printf '\nhalt file written by first run: %s\n' "${wrote}"
printf 'git status --porcelain after git tag -d:\n%s\n' "${dirty_after_undo}"
printf 'cut-release.sh re-run without --allow-dirty dies dirty: %s\n\n' "${rerun_died_dirty}"

# cleanup regardless of pass/fail so a real clone is left clean
(cd "${ROOT}" && git tag -d v1.3.0-readmecheck2 >/dev/null 2>&1; git checkout -- . 2>/dev/null; git clean -fdq)

rc=0
# IS: git tag -d does not remove the halt proposal file, and the worktree
# stays dirty until the operator finds and deletes it by hand.
[[ ${wrote} -eq 1 && -n "${dirty_after_undo}" && ${rerun_died_dirty} -eq 1 ]] || {
	echo "FAIL: expected the halt file to survive 'git tag -d' and dirty the next run"
	rc=1
}
# SHOULD (doc fix, not code): README.md:30 names `git tag -d` as sufficient to
# undo a `--halt-height` dry run; it should also say to remove the emitted
# misc/deployments/.../transactions/migration/halt-<version>/ directory.

[ "${rc}" = 0 ] && echo "CONFIRMED: ${rc}" && exit 0 || exit 1
