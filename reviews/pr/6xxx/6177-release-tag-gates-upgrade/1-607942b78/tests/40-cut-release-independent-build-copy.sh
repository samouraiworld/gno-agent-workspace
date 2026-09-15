#!/usr/bin/env bash
# What this proves: misc/release/cut-release.sh's check_build_reports_tag
# (VERSION_PKG at cut-release.sh:40, build+ldflags at :202-204, awk extractor
# at :213) is a second, independent copy of the build the workflow performs
# at .github/workflows/release-chain-tag.yml:71 and :98 — not a call into the
# same code. Breaking the script's own copy of the symbol does not touch the
# workflow at all, so a rename of tm2/pkg/version.Version that is updated only
# in the workflow leaves the script's copy stale and lying.
#
# Run from a plain clone of gnolang/gno at the reviewed head:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   bash 40-cut-release-independent-build-copy.sh
#
# Requires go1.25 on PATH (see the PR's toolchain line) and network access to
# origin for the "tag already exists" check to be skipped by using an unused
# version number.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "== count of independent copies of the ldflags symbol =="
grep -rn 'tm2/pkg/version\.Version' .github/ misc/release/

echo
echo "== workflow's own copy (untouched) =="
grep -n 'tm2/pkg/version.Version' .github/workflows/release-chain-tag.yml

echo
echo "== mutate ONLY cut-release.sh's copy to a bogus symbol =="
cp misc/release/cut-release.sh /tmp/cut-release.sh.orig
sed -i 's|readonly VERSION_PKG="github.com/gnolang/gno/tm2/pkg/version.Version"|readonly VERSION_PKG="github.com/gnolang/gno/tm2/pkg/version.BogusSymbolXYZ"|' misc/release/cut-release.sh
diff -u /tmp/cut-release.sh.orig misc/release/cut-release.sh || true

echo
echo "== run the script; it dies on its own broken copy =="
set +e
bash misc/release/cut-release.sh v9.9.9-cutrelease-probe --commit HEAD --allow-dirty
rc=$?
set -e
echo "exit code: $rc"

echo
echo "== the workflow file is untouched by the mutation =="
git diff --stat -- .github/workflows/release-chain-tag.yml

echo
echo "== restore =="
mv /tmp/cut-release.sh.orig misc/release/cut-release.sh
git tag -d v9.9.9-cutrelease-probe 2>/dev/null || true
