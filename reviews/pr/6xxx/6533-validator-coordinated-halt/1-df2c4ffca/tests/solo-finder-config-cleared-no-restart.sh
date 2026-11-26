#!/usr/bin/env bash
# Section 6 of misc/deployments/{mainnet,onyx}.gno.land/VALIDATOR.md (gnolang/gno#6533)
# tells the operator to "check that `gnoland config get halt_height` prints `0`"
# before a governance halt. That command reads config.toml; the running node read
# halt_height once at startup (gno.land/cmd/gnoland/start.go, SetHaltHeight).
# An operator who finds a future value, clears it with `gnoland config set
# halt_height 0` and re-checks sees `0`, while the running node still halts at
# the old height. This script plays that sequence on a lazy local node, then
# sends SIGTERM to the idle process to time the "wait until it has exited" step.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git checkout df2c4ffcaa229fbc574fdc6bb33de2a52641484f
#   go build -o /tmp/bin/gnoland ./gno.land/cmd/gnoland
#   PATH=/tmp/bin:$PATH GNOROOT=$PWD bash <path to this file> 6
#
# Expected today: "config get after clearing: 0", then the node halts at 6
# anyway ("halt height 6 reached"), the process is still alive after the halt,
# and it exits on SIGTERM.
set -u
HALT=${1:-6}
D=$(mktemp -d)
cd "$D"

gnoland config init >/dev/null
gnoland config set halt_height "$HALT" >/dev/null
echo "config get before start: $(gnoland config get halt_height)"

gnoland start --lazy --skip-genesis-sig-verification >node.log 2>&1 &
PID=$!

# Wait for the node to apply the config value and produce its first block.
for _ in $(seq 1 420); do
  grep -q 'Halt height configured' node.log && grep -qE 'Committed state|Finalizing commit' node.log && break
  kill -0 "$PID" 2>/dev/null || { echo "node died early"; tail -20 node.log; exit 1; }
  sleep 1
done
grep -m1 -o 'Halt height configured.*' node.log

# The operator follows section 6: finds a future value, clears it, re-checks.
gnoland config set halt_height 0 >/dev/null
echo "config get after clearing: $(gnoland config get halt_height)"

# Does the running node still halt at the old value?
for _ in $(seq 1 120); do
  grep -q "halt height $HALT reached" node.log && break
  sleep 1
done
grep -m1 -o "halt height [0-9]* reached[^\"]*" node.log || echo "no halt at $HALT within 120s"

sleep 5
if kill -0 "$PID" 2>/dev/null; then echo "process alive 5s after the halt"; else echo "process exited at the halt"; fi

kill -TERM "$PID"
for i in $(seq 1 60); do
  kill -0 "$PID" 2>/dev/null || break
  sleep 1
done
if kill -0 "$PID" 2>/dev/null; then
  echo "still alive 60s after SIGTERM"
  kill -9 "$PID"
else
  wait "$PID"
  echo "exited ${i}s after SIGTERM, rc=$?"
fi
