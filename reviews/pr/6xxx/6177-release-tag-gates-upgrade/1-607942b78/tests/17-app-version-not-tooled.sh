#!/usr/bin/env bash
# 17-app-version-not-tooled.sh — RELEASING.md:20 states "The tooling in
# misc/release/ handles all three [version surfaces]". Asserts misc/release/
# never mentions AppVersion while gno.land/pkg/gnoland/app.go hardcodes it to
# "dev". Fails (no misc/release hit, app.go still hardcoded) at head 607942b78.
#
# Run: from a local clone of gnolang/gno, at the clone root:
#   gh pr checkout 6177 -R gnolang/gno
#   git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   bash 17-app-version-not-tooled.sh

set -euo pipefail

echo "== grep AppVersion in misc/release/ =="
hits=$(grep -rn "AppVersion" misc/release/ || true)
echo "hits: '${hits}'"

echo
echo "== grep SetAppVersion in gno.land/pkg/gnoland/app.go =="
grep -n "SetAppVersion" gno.land/pkg/gnoland/app.go

if [[ -n ${hits} ]]; then
	echo "PASS: misc/release/ references AppVersion"
else
	echo "FAIL: misc/release/ has no AppVersion reference, and app.go:107 still calls SetAppVersion(\"dev\") unconditionally. RELEASING.md's three-surface table claims the tooling handles all three; the app-version surface is not touched by any script."
fi
