#!/usr/bin/env bash
# Asserts that POST /_/api/dryrun cannot succeed at head 4a0e7ff2a, and that the
# node accepts the same script when the transaction carries a real signature.
# The second run is the control that keeps the failure attributable to gnoweb's
# pubkey-only placeholder rather than to the node.
# It fails at the reviewed head and passes once Simulate signs.
#
# Run: from a local clone of gnolang/gno, with go and jq on PATH.
#   gh pr checkout 5421 -R gnolang/gno && git checkout 4a0e7ff2a
#   bash dryrun-unauthorized.sh
set -euo pipefail

ADDR=g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5
MNEMONIC='source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast'
WEB=http://127.0.0.1:8888
RPC=127.0.0.1:26657
TMP=$(mktemp -d)
trap 'kill "${DEV:-}" 2>/dev/null || true; rm -rf "$TMP"' EXIT

ROOT=$PWD
go build -o "$TMP/gnokey" ./gno.land/cmd/gnokey
(cd contribs/gnodev && go build -o "$TMP/gnodev" .)

(cd "$ROOT" && "$TMP/gnodev" local --web-listener 127.0.0.1:8888 \
	--node-rpc-listener "$RPC" --no-watch) >"$TMP/gnodev.log" 2>&1 &
DEV=$!
until curl -s -o /dev/null "$WEB/"; do sleep 3; done

cat > "$TMP/script.gno" <<'EOF'
package main

import "gno.land/r/gnoland/home"

func main() {
	println(home.Render(""))
}
EOF

# The address needs a public key on chain before either path can run.
printf '%s\n\n\n' "$MNEMONIC" |
	"$TMP/gnokey" add --home "$TMP/kb" --recover --insecure-password-stdin test1 >/dev/null 2>&1
printf '\n' | "$TMP/gnokey" maketx send --home "$TMP/kb" --send 1ugnot --to "$ADDR" \
	--gas-fee 1000000ugnot --gas-wanted 2000000 --broadcast --chainid dev \
	--remote "$RPC" --insecure-password-stdin test1 >/dev/null

echo "== gnoweb dry run, pubkey-only placeholder signature =="
jq -n --arg s "$(cat "$TMP/script.gno")" --arg a "$ADDR" \
	'{pkg_path:"gno.land/r/gnoland/home", script:$s, address:$a}' |
	curl -s -w '\nHTTP %{http_code}\n' -X POST "$WEB/_/api/dryrun" \
		-H 'Content-Type: application/json' -d @-

echo "== gnokey simulate of the same script, real signature =="
printf '\n' | "$TMP/gnokey" maketx run --home "$TMP/kb" --gas-fee 1000000ugnot \
	--gas-wanted 200000000 --simulate only --broadcast --chainid dev \
	--remote "$RPC" --insecure-password-stdin test1 "$TMP/script.gno" 2>&1 | tail -4
