#!/bin/sh
# gno run on the #6051 repro: does the host process still exit through a Go panic?
# From a plain clone of gnolang/gno at 51b1e76fb3c69b6ff52e27d950fdd7392d20da2b:
#   go build -o /tmp/gno-head ./gnovm/cmd/gno && sh <this file> /tmp/gno-head
# Observed at head: "panic: runtime error: index out of range [0] with length 0",
# "goroutine 1 [running]:", a host stack into machine.go:3285 (pushPanic), exit 2.
# The init() control takes the same Go-panic exit at head and at the merge base.
set -u
GNO=${1:-gno}
d=$(mktemp -d)
mkdir -p "$d/var" "$d/fn"
printf 'package main\n\nvar (\n\ta = ""\n\tA = a[0]\n)\n\nfunc main() {}\n' > "$d/var/main.gno"
printf 'package main\n\nfunc init() {\n\ta := ""\n\t_ = a[0]\n}\n\nfunc main() {}\n' > "$d/fn/main.gno"
echo '### var initializer'; "$GNO" run "$d/var/main.gno" 2>&1 | head -5
echo '### init() control'; "$GNO" run "$d/fn/main.gno" 2>&1 | head -5
