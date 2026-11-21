# Review: [#6462](https://github.com/gnolang/gno/pull/6462)
Posted: https://github.com/gnolang/gno/pull/6462#pullrequestreview-6004593948

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Two Warnings in the new gate block it: the `20x` cache line fails same-tree comparisons by noise alone, and `check.sh` passes any allocs/op regression from a zero base.
Model: claude-opus-5-5, quick review
Commit: d4e636abca2dd92070e6cf84da811aee3c12a0c4
Overview: [overview](../overview.md)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6462 d4e636abc`
Open the code: [misc/benchci](https://github.com/gnolang/gno/tree/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci)
Round: 1. One finder, a reflector pass, 8 candidates, each run or read from scratch by a judge that was not its finder; 0 refuted, 2 left PLAUSIBLE and skipped.

## Body

## misc/benchci/check.sh:26 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/check.sh#L26) · [↗](../../../../../.worktrees/gno-review-6462/misc/benchci/check.sh#L26) · Warning [posted](https://github.com/gnolang/gno/pull/6462#discussion_r4651965128)

This check skips the `?` benchstat prints for a zero base, so a benchmark going from 0 to 1 allocs/op passes at p=0.000.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6462 -R gnolang/gno
go install golang.org/x/perf/cmd/benchstat@v0.0.0-20260112171951-5abaabe9f1bd
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
bash misc/benchci/check.sh "$t/bs.csv"; echo "check.sh exit=$?"
rm -rf "$t"
```

`check.sh` reports only the `Slow-4` rows: `ZeroAlloc-4` goes from 0 to 1 allocs/op on every sample and never reaches the limit check.

```
ZeroAlloc-4,1.0055e-06,0%,1.0055e-06,0%,~,p=1.000 n=10
ZeroAlloc-4,0,0%,16,0%,?,p=0.000 n=10
ZeroAlloc-4,0,0%,1,0%,?,p=0.000 n=10
REGRESSION Slow-4 sec/op: +49.73% (limit 20%, p=0.000 n=10)
REGRESSION Slow-4 allocs/op: +50.00% (limit 10%, p=0.000 n=10)
benchci: 2 regression(s) above the limits
check.sh exit=1
```

The `allocs/op` table of this pull request's own `ci / bench` run shows `Benchdata/loop.gno-4 0.000 ± 0%` on both sides.
</details>

## misc/benchci/suite.txt:14 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/suite.txt#L14) · [↗](../../../../../.worktrees/gno-review-6462/misc/benchci/suite.txt#L14) · Warning [posted](https://github.com/gnolang/gno/pull/6462#discussion_r4651965131)

The `20x` here also covers both `CacheStoreGet` benchmarks, whose samples last only microseconds, so 3 of 250 same-tree runs failed the gate.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6462 -R gnolang/gno
go install golang.org/x/perf/cmd/benchstat@v0.0.0-20260112171951-5abaabe9f1bd
t=$(mktemp -d)
printf '%s\n' 'tm2/pkg/store/cache ^Benchmark(CacheStoreGet.*)$ 20x' >"$t/suite.txt"
red=0
for ((k = 1; k <= 250; k++)); do
  SUITE=$t/suite.txt COUNT=10 bash misc/benchci/run.sh . . "$t/out" >/dev/null 2>&1
  if ! res=$(bash misc/benchci/check.sh "$t/out/benchstat.csv"); then
    red=$((red + 1)); echo "rep $k: $(grep REGRESSION <<<"$res")"
  fi
done
echo "red $red/250"
rm -rf "$t"
```

Base and head are the same tree, so every `REGRESSION` line is a false red.

```
rep 56: REGRESSION CacheStoreGetKeyFound-16 sec/op: +23.63% (limit 20%, p=0.041 n=10)
rep 132: REGRESSION CacheStoreGetNoKeyFound-16 sec/op: +28.27% (limit 20%, p=0.035 n=10)
rep 225: REGRESSION CacheStoreGetKeyFound-16 sec/op: +20.02% (limit 20%, p=0.016 n=10)
red 3/250
```

The pull request's own `ci / bench` run shows the spread on a GitHub runner: `CacheStoreGetKeyFound-4 89.15n ± 66% 51.85n ± 141%` and `CacheStoreGetNoKeyFound-4 422.8n ± 20% 440.9n ± 37%`.
</details>

## SKIP gnovm/cmd/benchstore/store_test.go:445 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/gnovm/cmd/benchstore/store_test.go#L445) · [↗](../../../../../.worktrees/gno-review-6462/gnovm/cmd/benchstore/store_test.go#L445) · Nit

Nit: the sweep benchmark still skips silently without its sweep flag, so it reports `PASS` after measuring nothing.

Not posted: line 445 sits outside the diff, and failing there would turn every `-bench . -db=pebbledb` run red, since the sweep is opt-in.

## misc/benchci/check.sh:23 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/check.sh#L23) · [↗](../../../../../.worktrees/gno-review-6462/misc/benchci/check.sh#L23) · Nit [posted](https://github.com/gnolang/gno/pull/6462#discussion_r4651965136)

Nit: this `NF < 7` guard drops the three-field CSV row of a benchmark renamed on one side, so the gate stays green without comparing it.

<details><summary>check</summary>

With `BenchmarkWalRead10KB` renamed on head and the suite line `tm2/pkg/bft/wal ^BenchmarkWalRead(512B|10KB)$ 100ms`, `COUNT=4`:

```
WalRead10KB-16,2.7974e-05,∞
benchci: no regression above the limits
```
</details>

## misc/benchci/run.sh:22 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/run.sh#L22) · [↗](../../../../../.worktrees/gno-review-6462/misc/benchci/run.sh#L22) · Nit [posted](https://github.com/gnolang/gno/pull/6462#discussion_r4651965139)

Nit: this `mkdir -p` leaves earlier binaries in a reused output directory, so a failed base build benchmarks a stale one.

## .github/workflows/ci-bench.yml:12 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/.github/workflows/ci-bench.yml#L12) · [↗](../../../../../.worktrees/gno-review-6462/.github/workflows/ci-bench.yml#L12) · Suggestion [posted](https://github.com/gnolang/gno/pull/6462#discussion_r4651965142)

Suggestion: no suite package imports `gno.land`, so a change confined to `gno.land/` runs the job on identical code that can only fail on noise.

```suggestion
```

<details><summary>check</summary>

```bash
go list -deps -test ./gnovm/pkg/gnolang ./tm2/pkg/sdk/auth ./tm2/pkg/store/cache \
  ./tm2/pkg/bptree/benchmarks ./tm2/pkg/bft/wal ./tm2/pkg/crypto/secp256k1 ./tm2/pkg/amino \
  | grep -c 'gnolang/gno/gno.land'
```

```
0
```
</details>

## SKIP misc/benchci/check.sh:28 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/misc/benchci/check.sh#L28) · [↗](../../../../../.worktrees/gno-review-6462/misc/benchci/check.sh#L28) · Suggestion

Suggestion: give `B/op` a floor too, since this line maps it to `-1` and a tenfold larger allocation passes unless time also moves 20%.

Not posted: whether a `B/op` floor is worth its false reds takes a same-tree run of the `B/op` deltas, which this round did not make.

## SKIP tm2/pkg/bft/wal/wal_test.go:275 [gh](https://github.com/gnolang/gno/blob/d4e636abca2dd92070e6cf84da811aee3c12a0c4/tm2/pkg/bft/wal/wal_test.go#L275) · [↗](../../../../../.worktrees/gno-review-6462/tm2/pkg/bft/wal/wal_test.go#L275) · Suggestion

Suggestion: the open [gnolang/gno#6003](https://github.com/gnolang/gno/pull/6003) makes the same `maxSize` fix here, keeps the 10MB and 100MB sizes and adds a large-message test, so the branches conflict.

Not posted: it names a competing pull request, which a posted comment never does; the overlap is described in `overview.md` for the user to raise.
