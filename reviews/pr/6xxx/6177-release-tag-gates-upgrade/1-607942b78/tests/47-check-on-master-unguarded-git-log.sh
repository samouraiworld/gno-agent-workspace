#!/usr/bin/env bash
# 47-check-on-master-unguarded-git-log.sh — misc/release/cut-release.sh:170
# assigns `missing="$(git log ... origin/master..COMMIT -- ':!misc/deployments')"`
# with no `2>/dev/null` and no `|| true`, unlike the two guarded checks right
# above it in the same function (lines 158, 164). Under `set -euo pipefail`,
# a failed command substitution assigned to a variable still trips errexit,
# so a checkout with no local `origin/master` (a --single-branch clone, or a
# clone whose tracked remote is not named `origin`) aborts the whole preflight
# with a raw `fatal: bad revision` instead of the script's own die()/warn()
# messages. Fails (raw git fatal, exit 128, no later preflight step runs) at
# head 607942b78.
#
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   # Build the ldflags-stamped binaries the script's own build check needs,
#   # exactly as .github/workflows/release-chain-tag.yml does:
#   go build -trimpath -ldflags "-w -s -X github.com/gnolang/gno/tm2/pkg/version.Version=v1.2.0" \
#     -o /tmp/gnoland ./gno.land/cmd/gnoland
#   export PATH="/tmp:$PATH"  # not required for this repro: it fails before the build step
#   # Simulate a --single-branch clone: make an isolated clone of this tree
#   # that tracks the PR head but never fetched origin/master.
#   git tag -f _repro-head 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   rm -rf /tmp/singlebranch && git clone --single-branch --branch _repro-head -q "file://$PWD" /tmp/singlebranch
#   cd /tmp/singlebranch
#   bash misc/release/cut-release.sh v9.9.8 --allow-dirty --commit HEAD; echo "exit: $?"

set -euo pipefail

echo "== line 170, unguarded =="
grep -n 'missing="\$(git' misc/release/cut-release.sh

echo
echo "== lines 158 and 164, guarded (for contrast) =="
sed -n '156,166p' misc/release/cut-release.sh

echo
echo "== does a local origin/master ref exist here? =="
if git rev-parse -q --verify origin/master >/dev/null; then
	echo "yes: this clone will not reproduce the crash (fetch was done)."
	echo "run the isolated single-branch clone from the header to see the failure."
else
	echo "no: reproducing directly in this checkout"
	set +e
	out=$(bash misc/release/cut-release.sh v9.9.8-repro --allow-dirty --commit HEAD 2>&1)
	code=$?
	set -e
	echo "$out" | tail -8
	echo "exit code: $code"
	if [[ ${code} -eq 128 ]] && echo "$out" | grep -q 'fatal: bad revision'; then
		echo "FAIL (confirms the bug): script died on a raw git error, never reaching check_protocol_constants or the build check."
	else
		echo "did not reproduce as expected"
	fi
fi
