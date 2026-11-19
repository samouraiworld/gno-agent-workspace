#!/bin/bash
# Real-browser repro of solo-finder-1-gnodev-localhost-hotreload_test.go: real gnodev,
# real chromium, page opened at http://localhost:18888 (docs/resources/gnodev.md:39
# says "Open http://localhost:8888"), gnodev on its default host 127.0.0.1.
#
# From a plain clone of gnolang/gno, at each of dd05df75bc312f6bd064ba096225699f5e8fe320
# (head) and d25b308c1a22ec04f429a2826e681249126320cd (merge base):
#   (cd contribs/gnodev && go build -o /tmp/gnodev-<tag> .)
#   bash <this file> /tmp/gnodev-<tag> "$PWD" <tag>
# Head: the localhost page logs "WebSocket Error" in the console and gnodev logs
#   unable to upgrade connection ... error="websocket: request origin not allowed by Upgrader.CheckOrigin"
# Base: 0 upgrade errors for both pages.
# usage: browser-run.sh <gnodev binary> <GNOROOT> <tag>
set -u
BIN=$1; ROOT=$2; TAG=$3
S=$(dirname "$0")
D=$(mktemp -d -p "$S" run-$TAG-XXXX)
cd "$D"
GNOROOT=$ROOT "$BIN" local -interactive=false -web-listener 127.0.0.1:18888 -node-rpc-listener 127.0.0.1:36657 > "$D/gnodev.log" 2>&1 &
PID=$!
for i in $(seq 1 120); do curl -s -o /dev/null http://127.0.0.1:18888/ && break; sleep 0.5; done
echo "gnodev up after $i polls"
for U in http://127.0.0.1:18888/ http://localhost:18888/; do
  echo "== $TAG browser at $U"
  timeout 30 chromium --headless=new --no-sandbox --disable-gpu --user-data-dir="$D/prof" \
    --enable-logging=stderr --v=0 --virtual-time-budget=4000 --dump-dom "$U" 2>"$D/chrome.err" | grep -o 'ws://[^'"'"']*' | head -2
  grep -i 'websocket' "$D/chrome.err" | grep -v 'DevTools' | head -3
  echo "-- gnodev upgrade errors so far: $(grep -c 'unable to upgrade connection' $D/gnodev.log)"
done
grep 'unable to upgrade connection' "$D/gnodev.log" | head -2 | cut -c1-300
kill $PID; wait $PID 2>/dev/null
echo "log: $D/gnodev.log"
