#!/usr/bin/env bash
# Asserts that every deployment script's admin address matches the constant it
# names in examples/. Measured at f2bdb07b0: pearl, sapphire and topaz agree;
# test13 disagrees on both r/sys/names and boards2/v1.
#
# Run: from a gno checkout:
# gh pr checkout 6131 -R gnolang/gno && git checkout f2bdb07b0
# curl -fsSL -o /tmp/deployment-admin-consistency.sh \
#   https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6131-govdao-t1-multisig-address/1-f2bdb07b0/tests/deployment-admin-consistency.sh
# bash /tmp/deployment-admin-consistency.sh

set -u
cd "$(git rev-parse --show-toplevel)" || exit 1

names_admin=$(sed -n 's/.*admin[[:space:]]*=[[:space:]]*address("\(g1[0-9a-z]*\)").*/\1/p' \
  examples/gno.land/r/sys/names/verifier.gno)
boards_admin=$(sed -n 's/.*initRealmPermissions("\(g1[0-9a-z]*\)").*/\1/p' \
  examples/gno.land/r/gnoland/boards2/v1/boards.gno)

echo "r/sys/names admin       : $names_admin"
echo "boards2/v1 admin        : $boards_admin"
echo

fail=0
check() { # <label> <expected> <actual>
  if [ "$2" = "$3" ]; then
    printf 'OK    %-52s %s\n' "$1" "$3"
  else
    printf 'MISMATCH %-49s %s\n' "$1" "$3"
    fail=1
  fi
}

for chain in pearl.gno.land sapphire.gno.land topaz.gno.land test13.gno.land; do
  d="misc/deployments/$chain"
  [ -d "$d" ] || continue
  check "$chain NAMES_ADMIN" "$names_admin" \
    "$(sed -n 's/^NAMES_ADMIN=\(g1[0-9a-z]*\).*/\1/p' "$d/gen-genesis.sh")"
  meta="$d/transactions/migration/names-enable/meta.json"
  [ -f "$meta" ] && check "$chain names-enable caller_override" "$names_admin" \
    "$(sed -n 's/.*"caller_override"[[:space:]]*:[[:space:]]*"\(g1[0-9a-z]*\)".*/\1/p' "$meta")"
done

# Only the 7 caller-swap patches, which are MsgCall. The 2 dead-letter noops
# are MsgRun and keep their historical caller by design.
for meta in misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/*/meta.json; do
  [ -f "$meta" ] || continue
  grep -q '"/vm.m_call"' "$meta" || continue
  caller=$(sed -n 's/.*"caller"[[:space:]]*:[[:space:]]*"\(g1[0-9a-z]*\)".*/\1/p' "$meta" | head -1)
  [ -n "$caller" ] || continue
  check "boards2-cascade $(basename "$(dirname "$meta")") caller" "$boards_admin" "$caller"
done

echo
if [ "$fail" -eq 0 ]; then echo "PASS: every deployment admin matches its examples/ constant"; else
  echo "FAIL: a deployment admin no longer matches the constant it names"; fi
exit "$fail"
