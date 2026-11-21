# ci-bench: a benchmark regression gate on pull requests
Written by claude-opus-5-5.
PR: [gnolang/gno#6462](https://github.com/gnolang/gno/pull/6462)

## TLDR

Until now no CI job ran any of the repository's benchmarks, so a pull request could make the VM or the store much slower and still pass. This change adds a job that times a small set of benchmarks on the pull request and on the code it branches from, on the same machine, and fails when the pull request is clearly slower or allocates more. It also repairs two benchmarks that reported success while measuring nothing.

## The suite

`misc/benchci/suite.txt` lists what the job runs, one line per package: the package directory, a `-bench` regular expression and a `-benchtime`. It covers the GnoVM running toy programs and building a machine, the auth account mapper, the cache store, the B+ tree in memory, WAL decoding, secp256k1 signature checks and amino encoding, about 40 time rows in all.

The `-benchtime` is either a duration, `100ms`, or a fixed count, `20x`. A count suits a benchmark whose setup is expensive, since Go's `b.N` ramp reruns the setup at every step. The count applies to every benchmark the line's expression matches, so a fast benchmark on a `20x` line gets 20 iterations too.

## Measuring both sides on one runner

`misc/benchci/run.sh` takes two checkouts, base and head. `.github/workflows/ci-bench.yml` checks out the pull request and adds a second worktree at `git merge-base HEAD origin/<base branch>`, so both trees sit on the same runner.

The script compiles each package's test binary once per side with `go test -c`, then runs `COUNT` rounds (10 by default). Odd rounds run base first and even rounds head first, so drift during the job, a noisy neighbour or a warming CPU, lands on both sides. A package missing on base is run on head only. Each binary runs from its package directory, since some benchmarks read `testdata/`.

At the end, `benchstat` compares the two files of results and writes a text table, shown in the job summary, and a CSV that the next step reads.

## The significance test and the effect floor

`misc/benchci/check.sh` reads the CSV. benchstat runs a Mann-Whitney U test on the 10 samples of each side and writes `~` instead of a delta when p is 0.05 or more. A significant row then fails only if its slowdown also passes a floor: `MAX_TIME_REGRESSION` (20%) for `sec/op`, and `MAX_ALLOC_REGRESSION` (10%) for `allocs/op`, which does not depend on the machine. `B/op` is printed in the table but has no floor.

The wide time floor is the trade-off: on a shared runner a 3% drift is indistinguishable from noise, so the gate aims at the large regression nobody noticed. The floors are environment variables, so the workflow can loosen them without editing the script.

## The benchstore `-db` check

The benchmarks in `gnovm/cmd/benchstore` need a database backend named by `-db`. Before, `requireDB` called `b.Skip` without it, and a skip prints nothing without `-v`, so `go test -bench .` printed `PASS` after timing nothing. After, `requireDB` calls `b.Fatal` with the flag to pass, and `BenchmarkStoreGetCacheSweep` goes through the same check before its own pebbledb-only and `-cache-sweep` skips. Defaulting to pebbledb was set aside because the key sizes reach 1e9 keys on disk.

## The WAL read benchmark's size limit

`benchmarkWalRead` in `tm2/pkg/bft/wal/wal_test.go` writes one message of `n` bytes and decodes it `b.N` times. Before, the writer got a limit of `n+64` and the reader `maxTestMsgSize` (64KiB), so every size above 64KiB failed to decode, and five sizes were skipped with a TODO. After, one `maxSize` serves both, the write error is checked, 100KB and 1MB run again, and 10MB, 100MB and 1GB are deleted: consensus opens its WAL with a 1MB `maxMsgSize`, so those sizes never reach a production WAL.

## What a user notices

- A pull request changing Go code under `gnovm/`, `tm2/` or `gno.land/`, `go.mod`, `go.sum` or the gate's own files runs one more job, `ci / bench`, about 4 minutes on a GitHub runner for this pull request.
- That job fails when a suite row is significantly slower by 20% or more, or allocates 10% more, and its summary carries the full benchstat table either way.
- `go test -bench` on `gnovm/cmd/benchstore` without `-db` now fails with a message naming the flag.

## Not fixed

- Benchmarks outside `suite.txt` still run in no CI job; adding one is a line in that file.
- The IAVL benchmarks and their Makefile targets are left alone; the description defers them to a separate issue.

## Overlap with gnolang/gno#6003

The open [gnolang/gno#6003](https://github.com/gnolang/gno/pull/6003) fixes the same writer/reader limit in `benchmarkWalRead`, keeps the 10MB and 100MB sizes behind `testing.Short()`, and adds `TestWALWriterReaderLargeMessage`, a plain test of a 100KB round trip. Both edit the same lines of `tm2/pkg/bft/wal/wal_test.go`, so whichever merges second resolves that file: the helper's `maxSize` lines agree, while the size functions this change deletes are the ones #6003 keeps.

[Review files](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6462-benchmark-regression-gate)

## Words used here

| Name | What it is |
| --- | --- |
| `suite.txt` | The list the job runs: `<package dir> <-bench regexp> <-benchtime>` per line, `#` for comments. |
| `run.sh` | Compiles both sides' test binaries once and runs them in `COUNT` interleaved rounds, then calls benchstat. |
| `check.sh` | Reads benchstat's CSV and exits 1 on a significant `sec/op` or `allocs/op` slowdown above its floor. |
| `COUNT` | Rounds per side, default 10; benchstat needs at least 6 for a confidence interval. |
| `MAX_TIME_REGRESSION` | The `sec/op` floor in percent, default 20. |
| `MAX_ALLOC_REGRESSION` | The `allocs/op` floor in percent, default 10. |
| benchstat | `golang.org/x/perf/cmd/benchstat`, pinned in the workflow; compares two sets of Go benchmark results. |
| `requireDB` | The benchstore helper that now fails a benchmark run without `-db`. |
| `maxSize` | The single size limit `benchmarkWalRead` gives the WAL writer and reader, `n + 64` bytes. |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a generated file.

### 1. `hot` · What runs, [`misc/benchci/suite.txt`](https://github.com/gnolang/gno/pull/6462/files#diff-3890f0322f81815780801cc4c32451a8914017d58bcb3dccd9f6bc96cf810316R9-R24)
Decides which benchmarks the gate times and for how long.
**If wrong:** a row too short to measure fails pull requests that changed nothing, or a regression in an unlisted path passes.
```text
# Cache store, including the iterator path that went quadratic before.
tm2/pkg/store/cache        ^Benchmark(AccountLoad1000|CacheStoreIterator1000|CacheStoreGet.*)$  20x
```

### 2. `hot` · When it runs, [`.github/workflows/ci-bench.yml`](https://github.com/gnolang/gno/pull/6462/files#diff-c0d8bb86d6a1aa2b01fbd72483d2b2380bd3344a56b386cff7a751410e340f41R16-R61)
Sets the path filter, checks out the merge base beside the pull request and runs the two scripts.
**If wrong:** the job compares the wrong pair of trees, or runs on changes that cannot move a benchmark.
```yaml
      - name: Run benchmarks
        run: head/misc/benchci/run.sh base head out
      - name: Check for regressions
        run: head/misc/benchci/check.sh out/benchstat.csv
```

### 3. `warm` · Interleaved rounds, [`misc/benchci/run.sh`](https://github.com/gnolang/gno/pull/6462/files#diff-d56b912275221f8d7bc442ace04ee4b6135ff0c97118a508b374d58ecb40084eR44-R64)
Runs each compiled binary per round, alternating which side goes first.
**If wrong:** drift during the job lands on one side and reads as a regression.
```bash
for ((r = 1; r <= count; r++)); do
	if ((r % 2)); then order=(base head); else order=(head base); fi
```

### 4. `warm` · The decision, [`misc/benchci/check.sh`](https://github.com/gnolang/gno/pull/6462/files#diff-d96551e47f6f8854b4b89e966c9f0e7f9a9427a94f733f35c8a51dfa26020505R20-R37)
Turns benchstat's CSV into pass or fail.
**If wrong:** the job is green on a regression, or red on noise.
```awk
	delta = $6
	if (delta == "~" || delta !~ /^\+/) next
```

### 5. `cold` · The `-db` check, [`gnovm/cmd/benchstore/store_test.go`](https://github.com/gnolang/gno/pull/6462/files#diff-110edc42d2df1acd865b70aa1d7c27c9ccf26147c017fb5ccb1c0b58fa5da727R54-R61)
Fails a benchmark run that names no backend.
**If wrong:** `go test -bench` prints `PASS` after measuring nothing.

### 6. `cold` · One size limit, [`tm2/pkg/bft/wal/wal_test.go`](https://github.com/gnolang/gno/pull/6462/files#diff-a67939d034b0ba68adc089bf47c84f4ce139abd6fdc28f8a8a6e1cf7aaee158bR273-R296)
Gives the WAL writer and reader the same limit.
**If wrong:** every size above 64KiB fails to decode.

</details>
