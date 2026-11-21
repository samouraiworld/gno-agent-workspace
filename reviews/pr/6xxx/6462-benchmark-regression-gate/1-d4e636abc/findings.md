# Findings in posting order, from round assemble: 6 to post, 2 SKIP, 0 refuted kept out

## misc/benchci/check.sh:26 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/check.sh#L26) · Warning
State: CONFIRMED, band: Warning, angle: silent-miss
TL;DR: a benchmark going from 0 to N allocs/op passes the alloc gate, since benchstat prints '?' and check.sh skips every delta not starting with '+'
Check: bash tests/solo-finder-zero-alloc-miss.sh <head checkout>
Details: check.sh:26 'if (delta == "~" || delta !~ /^\+/) next' drops benchstat's '?' delta, which the pinned benchstat prints when the base is 0 (no ratio). Benchdata/loop.gno is 0 allocs/op in the PR's own CI run, so one allocation per loop iteration there passes the allocs/op floor with p=0.000.
Evidence: pinned benchstat: 'ZeroAlloc-4,0,0%,1,0%,?,p=0.000 n=10'; check.sh printed only Slow-4 rows, 'benchci: 2 regression(s) above the limits'. CI log: 'Benchdata/loop.gno-4 0.000 +- 0% 0.000 +- 0%' under allocs/op
Artifact: tests/solo-finder-zero-alloc-miss.sh

## misc/benchci/suite.txt:14 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/suite.txt#L14) · Warning
State: CONFIRMED, band: Warning, angle: flaky-red
TL;DR: the cache line runs CacheStoreGet* at 20 iterations, so the gate fails A/A comparisons of an unchanged tree
Check: REPS=250 bash tests/solo-finder-aa-cache-20x.sh <judge worktree>
Details: suite.txt:14 gives tm2/pkg/store/cache '20x' for every benchmark on the line, including CacheStoreGetKeyFound/NoKeyFound, which take 50-400 ns per op; 20 iterations time a few microseconds, and the PR's own CI run shows them at +-66%/+-141% and +-20%/+-37%. With the same tree on both sides, check.sh still exited 1 in 3 of 250 runs on this machine, each delta above the 20% floor. Raising MAX_TIME_REGRESSION does not bound it, since the finder's runs reached +78.95%.
Evidence: rep 56: REGRESSION CacheStoreGetKeyFound-16 sec/op: +23.63% (limit 20%, p=0.041 n=10); rep 132: REGRESSION CacheStoreGetNoKeyFound-16 sec/op: +28.27% (p=0.035); rep 225: REGRESSION CacheStoreGetKeyFound-16 sec/op: +20.02% (p=0.016); red 3/250. CI 42262327632: 'CacheStoreGetKeyFound-4 89.15n +- 66% 51.85n +- 141%'
Artifact: tests/judge-1-aa-cache-20x.out

## SKIP gnovm/cmd/benchstore/store_test.go:445 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/gnovm/cmd/benchstore/store_test.go#L445) · Nit
State: PLAUSIBLE, band: Nit, angle: removed
TL;DR: BenchmarkStoreGetCacheSweep still skips silently without -cache-sweep
Check: cd gnovm/cmd/benchstore && go test -run '^$' -bench CacheSweep -db=pebbledb .
Details: Line 445 'b.Skip("use -cache-sweep=500,1024,... to run")' is the silent-skip shape requireDB replaced, but failing there would turn every '-bench . -db=pebbledb' run red, so keeping the skip is a defensible design; the run from the package directory is still to do.

## misc/benchci/run.sh:22 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/run.sh#L22) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: run.sh creates $out/bin without clearing it, so a reused output directory can run a binary left by an earlier run
Check: Read run.sh lines 22, 35-39 and 48
Details: Line 22 'mkdir -p "$out/bin/base" "$out/bin/head"' keeps old files; lines 36-38 let a base build fail with only a message, and line 48 '[[ -x $bin ]] || return 0' then runs whatever $i.test an earlier run left, possibly for another suite line. CI starts from a fresh out/, so only the README's local flow reaches it; and with a stale file, a package missing on base makes run()'s cd fail and aborts the run instead of benchmarking head only.
Evidence: sed -n '22p;35,40p;48p' run.sh at head

## .github/workflows/ci-bench.yml:12 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/.github/workflows/ci-bench.yml#L12) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: the gno.land/**/*.go trigger starts the job for changes no suite benchmark depends on
Check: go list -deps -test ... | grep -c gnolang/gno/gno.land
Details: None of the seven suite packages imports a gno.land package, so a gno.land-only pull request pays the job's runner time for a comparison of identical code, which can only go red from noise (candidate 1).
Evidence: go list -deps -test ./gnovm/pkg/gnolang ./tm2/pkg/sdk/auth ./tm2/pkg/store/cache ./tm2/pkg/bptree/benchmarks ./tm2/pkg/bft/wal ./tm2/pkg/crypto/secp256k1 ./tm2/pkg/amino | grep -c gnolang/gno/gno.land: 0

## misc/benchci/check.sh:23 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/check.sh#L23) · Suggestion
State: CONFIRMED, band: Suggestion, angle: silent-miss
TL;DR: a benchmark present on one side only yields a 3-field CSV row that check.sh drops at NF < 7, so the gate reports no regression with that row uncompared
Check: Edit suite.txt or rename one benchmark on head, run run.sh + check.sh: expect a failure or warning, observe exit 0
Details: Re-anchored from run.sh:48: the run shows the row reaching check.sh and dying at check.sh:23 '$1 == "" || $1 == "geomean" || NF < 7 { next }'. A rename, a move or a regexp that stops matching on head turns a benchmark into a base-only row and the job stays green.
Evidence: CSV 'WalRead10KB-16,2.7974e-05,inf' (NF=3); benchstat table 'WalRead10KB-16 27.97u +- inf' with an empty head column; check.sh 'benchci: no regression above the limits', exit 0

## SKIP misc/benchci/check.sh:28 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/check.sh#L28) · Suggestion
State: PLAUSIBLE, band: Suggestion, angle: catalog
TL;DR: B/op rows never fail the gate
Check: Read check.sh:28
Details: check.sh:28 'limit = unit == "sec/op" ? max_time : unit == "allocs/op" ? max_alloc : -1' gives B/op limit -1. Whether a B/op floor earns its false-red cost is unmeasured.

## tm2/pkg/bft/wal/wal_test.go:275 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/tm2/pkg/bft/wal/wal_test.go#L275) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: open gnolang/gno#6003 makes the same maxSize fix in the same helper and keeps the 10MB and 100MB sizes this branch deletes
Check: gh pr diff 6003 -R gnolang/gno
Details: #6003 is OPEN and MERGEABLE, touches only wal_test.go, adds the same 'maxSize := int64(n) + 64' writer/reader pair, adds TestWALWriterReaderLargeMessage and gates the largest size on testing.Short(). The two branches conflict in benchmarkWalRead and the deleted benchmarks.
Evidence: state OPEN, mergeable MERGEABLE, files [tm2/pkg/bft/wal/wal_test.go]; diff '+func TestWALWriterReaderLargeMessage', '+\tmaxSize := int64(n) + 64 // n + overhead.', '+\tif testing.Short() {'
