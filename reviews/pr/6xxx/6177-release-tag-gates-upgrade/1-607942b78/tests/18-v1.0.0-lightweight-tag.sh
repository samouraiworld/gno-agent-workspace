#!/usr/bin/env bash
# 18-v1.0.0-lightweight-tag.sh — RELEASING.md:76 states "Tags are annotated
# (git tag -a), so git describe prefers them". Asserts v1.0.0 is not annotated,
# contradicting the rule's own premise, and that `git describe` at v1.0.0's
# commit does not resolve to v1.0.0. Fails (v1.0.0 is a plain commit object,
# and describe misses it) at head 607942b78.
#
# Run: from a local clone of gnolang/gno, at the clone root:
#   gh pr checkout 6177 -R gnolang/gno
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   bash 18-v1.0.0-lightweight-tag.sh

set -euo pipefail

echo "== object type of v1.0.0 (should be 'tag' if the doc's rule held) =="
v100_type=$(git cat-file -t v1.0.0)
echo "v1.0.0: ${v100_type}"

echo "== object type of v1.1.0 (annotated, for contrast) =="
git cat-file -t v1.1.0

commit=$(git rev-parse v1.0.0^{commit})
echo
echo "== git describe at v1.0.0's own commit (${commit:0:9}) =="
echo "describe:        $(git describe "${commit}" 2>&1)"
echo "describe --tags: $(git describe --tags "${commit}" 2>&1)"

if [[ ${v100_type} == "tag" ]]; then
	echo "PASS: v1.0.0 is annotated"
else
	echo "FAIL: v1.0.0 is a lightweight tag (object type '${v100_type}'), so RELEASING.md:76's 'Tags are annotated, so git describe prefers them' is false for the release line this PR documents. Confirmed intentional: PR comment 2026-09-14T17:07:44Z from the author states the restore preserved v1.0.0's original lightweight object on purpose."
fi
