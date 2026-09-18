#!/usr/bin/env bash
# Asserts gnopreview validates its command name before reading -changed: an
# unknown command must print the usage error, not block on stdin. Measured at
# ecf7af0f2 with go1.25.9: case A exits 124 (timeout killed it, no output).
# Fails at the reviewed head; passes once the command switch moves above run()'s
# findRoot/readLines/BuildPlan calls.
#
# From a plain clone:
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash <this file>
set -u
cd "$(git rev-parse --show-toplevel)/misc/gnopreview" || exit 2
bin=$(mktemp -d)/gnopreview
go build -o "$bin" . || exit 2

# A: stdin held open (a terminal, or any live pipe) — the head hangs here.
(sleep 30 | timeout 5 "$bin" help) >/dev/null 2>&1
a=$?
# B: stdin at EOF — readLines returns empty and the switch is finally reached.
b_out=$(timeout 5 "$bin" help < /dev/null 2>&1)
b=$?

echo "A (open stdin)  exit=$a   # IS:     124, killed by timeout — blocked in readLines"
#echo "A (open stdin)  exit=1    # SHOULD: 1, 'unknown command \"help\"' before any I/O"
echo "B (closed stdin) exit=$b  out=$b_out"
rm -rf "$(dirname "$bin")"
[ "$a" -eq 1 ] && echo PASS || echo "FAIL: unknown command blocked on stdin (exit $a)"
