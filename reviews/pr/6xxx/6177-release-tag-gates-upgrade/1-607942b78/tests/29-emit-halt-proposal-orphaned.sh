#!/usr/bin/env bash
# Candidate #29: misc/release/cut-release.sh:280 emit_halt_proposal writes a
# bare .gno straight into transactions/migration/, a tree whose every other
# entry is a meta.json a genesis or gen-genesis.sh step consumes by name.
# Confirms: the emitted file has no meta.json, gen-genesis.sh reads only the
# migration dirs it names explicitly (names-enable), and every other
# hand-written post-genesis proposal lives under transactions/patched/<name>/h<height>/.
#
# from a local clone of gnolang/gno:
#   git fetch origin pull/6177/head:pr6177 && git checkout pr6177
set -euo pipefail

echo "== every file under transactions/migration/*/ (all chains) =="
find misc/deployments -path '*/transactions/migration/*' -type f

echo
echo "== transactions/patched/ holds the hand-written post-genesis proposals =="
find misc/deployments/test13.gno.land/transactions/patched -maxdepth 2 -type d | head -10

echo
echo "== gen-genesis.sh names its migration dir explicitly and reads meta.json from it =="
grep -n 'migration/names-enable\|meta.json' misc/deployments/mainnet.gno.land/gen-genesis.sh | head -5

echo
echo "== run cut-release.sh --halt-height and inspect what it wrote =="
misc/release/cut-release.sh v1.3.0 --chain test13 --commit HEAD --halt-height 999999 \
  >/tmp/cut-release-halt.log 2>&1 || true
find misc/deployments/test13.gno.land/transactions/migration/halt-v1.3.0 -type f

# cleanup
git tag -d v1.3.0 >/dev/null 2>&1 || true
git checkout -- . && git clean -fdq
