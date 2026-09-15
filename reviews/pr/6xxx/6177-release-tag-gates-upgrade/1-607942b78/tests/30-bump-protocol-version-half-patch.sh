#!/usr/bin/env bash
# What: misc/release/bump-protocol-version.sh dies mid write-loop and leaves the
#       six protocol-version constants half patched, with no revert and no
#       message about which files changed.
# Measured: a die between constant 4 and constant 5 out of 6 leaves exactly 4
#       files modified, git status shows nothing else, and the next invocation
#       dies on a different message ("constants already disagree") instead of
#       resuming or explaining the mess it inherited.
# Fails at reviewed head 607942b78fa4fdf6f378fecce32bc1d1d984ab8e.
#
# Run:
#   # from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   chmod -w tm2/pkg/p2p/version   # blocks sed -i's temp-file+rename in that dir;
#                                  # chmod -w on the file alone does NOT block it,
#                                  # since GNU sed -i replaces via rename, which
#                                  # only needs the containing directory writable.
#   bash misc/release/bump-protocol-version.sh v1.1.0; echo "exit=$?"
#   git status --porcelain
#   chmod +w tm2/pkg/p2p/version
#   git checkout -- . && git clean -fdq

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

trap 'chmod +w tm2/pkg/p2p/version 2>/dev/null || true; git checkout -- . ; git clean -fdq' EXIT

chmod -w tm2/pkg/p2p/version

set +e
out="$(bash misc/release/bump-protocol-version.sh v1.1.0 2>&1)"
code=$?
set -e

echo "--- script output ---"
printf '%s\n' "${out}"
echo "--- exit=${code} ---"

modified="$(git status --porcelain | wc -l)"
echo "--- files left modified: ${modified} (expected 4 of 6, no revert) ---"
git status --porcelain
