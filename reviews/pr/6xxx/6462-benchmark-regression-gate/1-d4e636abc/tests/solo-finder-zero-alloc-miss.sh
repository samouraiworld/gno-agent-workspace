#!/usr/bin/env bash
# check.sh passes a 0 -> N allocs/op regression: benchstat prints "?" (no
# ratio from a zero base) with p=0.000, and check.sh skips every delta that
# does not start with "+".
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git checkout d4e636abca2dd92070e6cf84da811aee3c12a0c4
#   go install golang.org/x/perf/cmd/benchstat@v0.0.0-20260112171951-5abaabe9f1bd
#   bash <this file> .
# Expect: "REGRESSION Slow-4 ..." lines only; no line for ZeroAlloc-4,
# whose allocs/op goes 0 -> 1 on every sample.
# Observed (benchstat v0.0.0-20260709024250, CSV layout identical):
#   ZeroAlloc-4,0,0%,1,0%,?,p=0.000 n=10
#   REGRESSION Slow-4 sec/op: +49.73% ...
#   REGRESSION Slow-4 allocs/op: +50.00% ...   (no ZeroAlloc-4 row)
set -u
repo=$(cd "${1:-.}" && pwd)
t=$(mktemp -d)
hdr=$'goos: linux\ngoarch: amd64\npkg: example.com/x\ncpu: X'
{ echo "$hdr"; for i in $(seq 10); do
  printf 'BenchmarkZeroAlloc-4 \t 1000000\t %d ns/op\t 0 B/op\t 0 allocs/op\n' $((1000+i))
  printf 'BenchmarkSlow-4 \t 1000000\t %d ns/op\t 64 B/op\t 2 allocs/op\n' $((1000+i)); done; } >"$t/base.txt"
{ echo "$hdr"; for i in $(seq 10); do
  printf 'BenchmarkZeroAlloc-4 \t 1000000\t %d ns/op\t 16 B/op\t 1 allocs/op\n' $((1000+i))
  printf 'BenchmarkSlow-4 \t 1000000\t %d ns/op\t 96 B/op\t 3 allocs/op\n' $((1500+i)); done; } >"$t/head.txt"
(cd "$t" && benchstat -format csv base.txt head.txt 2>/dev/null >bs.csv)
grep -E '^ZeroAlloc' "$t/bs.csv"
bash "$repo/misc/benchci/check.sh" "$t/bs.csv"; echo "check.sh exit=$?"
