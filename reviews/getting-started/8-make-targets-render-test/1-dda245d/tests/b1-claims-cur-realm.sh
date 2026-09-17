#!/bin/sh
# b1-claims-cur-realm.sh — AGENTS.md:43-44 ("they take a first parameter
# `cur realm` ... See `Set` in `hello.gno`") and README.md:54 ("note the
# `cur realm` parameter") both send the reader to `Set`, which declares
# `_ realm`. The rename that makes the example match the lesson is free: this
# script applies it, runs the repo's own bar (`make test lint`), and reverts.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/getting-started && cd getting-started
#   git fetch origin pull/8/head
#   git checkout dda245ddc4556d0b83e471b67830d17c0df7b666
#   sh path/to/b1-claims-cur-realm.sh
#
# Needs `gno` on PATH, built from gnolang/gno master. Measured at master
# bbd9b2ffe; the PR body reports verification on master.184+393b6f92a, a
# different master sha, so a verdict that turns on VM behaviour should say so.
set -e
echo "--- before ---"
grep -n '^func Set' hello.gno
sed -i 's/^func Set(_ realm, newMsg string) {/func Set(cur realm, newMsg string) {/' hello.gno
echo "--- after ---"
grep -n '^func Set' hello.gno
make test
make lint
echo "--- revert ---"
git checkout -- hello.gno
grep -n '^func Set' hello.gno
