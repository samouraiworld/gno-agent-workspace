#!/usr/bin/env bash
# 16-chain-betanet-branch-missing.sh — RELEASING.md's Branches and tags table
# (line 59) documents a `chain/betanet` branch. Asserts the branch does not
# exist on origin at the reviewed head, while the branch the table should name,
# `chain/gnoland1`, does. Fails (finds no chain/betanet) at head 607942b78.
#
# Run: from a local clone of gnolang/gno, at the clone root:
#   gh pr checkout 6177 -R gnolang/gno
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   bash 16-chain-betanet-branch-missing.sh

set -euo pipefail

echo "== origin chain/* branches =="
git ls-remote --heads origin 'refs/heads/chain/*' | awk '{print $2}' | sed 's#refs/heads/##'

echo
echo "== RELEASING.md's Branches and tags row =="
grep -n 'chain/betanet' RELEASING.md || true

if git ls-remote --heads origin 'refs/heads/chain/betanet' | grep -q chain/betanet; then
	echo "PASS: chain/betanet exists on origin"
else
	echo "FAIL: chain/betanet does not exist on origin; the table documents a branch that is not there. Only chain/gnoland1 exists, and the PR body's 'Not in this PR' section lists the rename to chain/betanet as deferred."
fi
