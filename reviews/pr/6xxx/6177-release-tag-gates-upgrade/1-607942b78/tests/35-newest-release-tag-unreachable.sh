#!/usr/bin/env bash
# Candidate #35: misc/release/cut-release.sh:224 newest_release_tag()
#
# Claim: `newest_release_tag` takes the highest `v*` tag in the whole repo
# (`git tag --list 'v*' --sort=-v:refname | awk 'NR==1'`), not the newest tag
# reachable from the commit being tagged. Once a newer network's tag (e.g.
# v2.0.0, unrelated history) exists, cutting an LTS patch (v1.2.1) on an
# older line (chain/mainnet) picks that unrelated tag as PREVIOUS, and
# `classify` reports "MAJOR bump: a new network, incompatible genesis" for
# a backward-compatible patch release.
#
# Run from a plain clone of gnolang/gno at 607942b78fa4fdf6f378fecce32bc1d1d984ab8e:
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   # build gnoland/gnokey/gno once, with PATH set to a toolchain go, then:
#   bash 35-newest-release-tag-unreachable.sh
#
# Observed at the reviewed head (this run):
#   git tag v2.0.0 HEAD~2                     # an unrelated, later-numbered network's tag
#   bash misc/release/cut-release.sh v1.2.1 --commit HEAD --allow-dirty
#     ==> since v2.0.0:
#           607942b78 docs(releasing): v1.0.0 and v1.1.0 are restored, not deleted
#           3b8248442 docs: follow #6175 — two parseable version shapes, not one
#     ==> MAJOR bump: a new network, incompatible genesis. See RELEASING.md.
#
# Passing --previous v1.1.0 explicitly (the tag actually reachable from HEAD
# on this line) instead reports the correct PATCH classification, confirming
# newest_release_tag's repo-wide pick, not the LTS-branch content, is what
# flips the message.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

if ! git rev-parse -q --verify v1.1.0^{commit} >/dev/null; then
	echo "SKIP: this clone has no v1.1.0 tag (restored in 607942b78); cannot reproduce" >&2
	exit 0
fi

trap 'git tag -d v2.0.0 2>/dev/null; git tag -d v1.2.1 2>/dev/null; true' EXIT

git tag v2.0.0 HEAD~2

out="$(bash misc/release/cut-release.sh v1.2.1 --commit HEAD --allow-dirty 2>&1 || true)"
echo "${out}" | grep -E 'since |MAJOR|MINOR|PATCH'

if echo "${out}" | grep -q 'MAJOR bump'; then
	echo
	echo "CONFIRMED: v1.2.1 (an LTS patch on the v1.x line) classified as a" \
		"MAJOR bump because newest_release_tag picked the repo-wide highest" \
		"tag (v2.0.0, unrelated history) instead of the tag reachable from HEAD."
else
	echo "NOT REPRODUCED: no MAJOR bump reported" >&2
	exit 1
fi
