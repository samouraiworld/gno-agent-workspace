#!/usr/bin/env bash
# A/A false-red rate of the suite's tm2/pkg/store/cache line at -benchtime 20x.
# Same tree on both sides, so every REGRESSION check.sh prints is noise.
# CacheStoreGet{NoKeyFound,KeyFound} time ~50-400ns per op, so 20 iterations
# measure 1-8us in total: the PR's own CI run (42262327632) shows them at
# +-20%/+-37% and +-66%/+-141%.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git checkout d4e636abca2dd92070e6cf84da811aee3c12a0c4
#   go install golang.org/x/perf/cmd/benchstat@v0.0.0-20260112171951-5abaabe9f1bd
#   REPS=60 bash <this file> .
# Expect: "red N/REPS" with N > 0 means check.sh fails a pull request that
# changed nothing in the package.
set -u
repo=$(cd "${1:-.}" && pwd)
reps=${REPS:-60}
t=$(mktemp -d)
printf '%s\n' 'tm2/pkg/store/cache ^Benchmark(CacheStoreGet.*)$ 20x' >"$t/suite.txt"
red=0
for ((k = 1; k <= reps; k++)); do
  SUITE=$t/suite.txt COUNT=10 bash "$repo/misc/benchci/run.sh" "$repo" "$repo" "$t/out" >/dev/null 2>&1 || { echo "run.sh failed"; exit 2; }
  if ! res=$(bash "$repo/misc/benchci/check.sh" "$t/out/benchstat.csv"); then
    red=$((red + 1)); echo "rep $k: $(grep REGRESSION <<<"$res" | tr '\n' ' ')"
  fi
  grep -E '^CacheStoreGet' "$t/out/benchstat.csv" | head -2 | cut -d, -f1,6,7 | tr '\n' ' ' >>"$t/deltas.txt"; echo >>"$t/deltas.txt"
done
echo "red $red/$reps"
echo "sec/op deltas (first 8 reps):"; head -8 "$t/deltas.txt"
