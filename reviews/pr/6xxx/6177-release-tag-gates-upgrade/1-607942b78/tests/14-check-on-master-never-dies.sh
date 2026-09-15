#!/usr/bin/env bash
# 14-check-on-master-never-dies.sh — RELEASING.md:109 lists "The commit is on
# master" among what "the script refuses". Asserts every exit path of
# check_on_master() in misc/release/cut-release.sh is `ok` or `warn`, never
# `die`, contradicting the doc and matching misc/release/README.md's own
# wording ("It *warns*, rather than refusing, when the commit is not on
# master"). Fails (no `die` found) at head 607942b78.
#
# Run: from a local clone of gnolang/gno, at the clone root:
#   gh pr checkout 6177 -R gnolang/gno
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   bash 14-check-on-master-never-dies.sh

set -euo pipefail

echo "== check_on_master() body =="
awk '/^check_on_master\(\)/{p=1} p{print} p&&/^}/{exit}' misc/release/cut-release.sh

echo
echo "== exit-path keywords used inside it =="
body=$(awk '/^check_on_master\(\)/{p=1} p{print} p&&/^}/{exit}' misc/release/cut-release.sh)
echo "$body" | grep -oE '\b(ok|warn|die)\b' | sort -u

echo
echo "== README.md's own description (misc/release/README.md) =="
grep -n -A2 'It \*warns\*' misc/release/README.md

if echo "$body" | grep -q '\bdie\b'; then
	echo "PASS: check_on_master can die"
else
	echo "FAIL: check_on_master never calls die; every path is ok or warn, so a commit off master does not stop cut-release.sh, contradicting RELEASING.md:109's refusal list. misc/release/README.md:47 states the correct (warn-only) behavior in the same PR."
fi
