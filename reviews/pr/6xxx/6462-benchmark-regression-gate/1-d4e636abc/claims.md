# Claims: gnolang/gno#6462 round 1, d4e636abc, claude-opus-5-5, quick review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`check.sh:26`](https://github.com/gnolang/gno/blob/d4e636abc/misc/benchci/check.sh#L26) | 2 | Warning | CONFIRMED | differs | posts |
| [`suite.txt:14`](https://github.com/gnolang/gno/blob/d4e636abc/misc/benchci/suite.txt#L14) | 1 | Warning | CONFIRMED | differs | posts |
| [`store_test.go:445`](https://github.com/gnolang/gno/blob/d4e636abc/gnovm/cmd/benchstore/store_test.go#L445) | 7 | Nit | CONFIRMED | same | SKIP |
| [`run.sh:22`](https://github.com/gnolang/gno/blob/d4e636abc/misc/benchci/run.sh#L22) | 6 | Nit | CONFIRMED | differs | posts |
| [`ci-bench.yml:12`](https://github.com/gnolang/gno/blob/d4e636abc/.github/workflows/ci-bench.yml#L12) | 4 | Suggestion | CONFIRMED | differs | posts |
| [`check.sh:23`](https://github.com/gnolang/gno/blob/d4e636abc/misc/benchci/check.sh#L23) | 3 | Nit | CONFIRMED | differs | posts |
| [`check.sh:28`](https://github.com/gnolang/gno/blob/d4e636abc/misc/benchci/check.sh#L28) | 8 | Suggestion | PLAUSIBLE | differs | SKIP |
| [`wal_test.go:275`](https://github.com/gnolang/gno/blob/d4e636abc/tm2/pkg/bft/wal/wal_test.go#L275) | 5 | Suggestion | CONFIRMED |  | SKIP |

### 2 · Warning · CONFIRMED · `misc/benchci/check.sh:26`

A benchmark going from 0 to N allocs/op passes the alloc gate, since benchstat prints '?' and check.sh skips every delta not starting with '+'

- Check: bash tests/solo-finder-zero-alloc-miss.sh <head checkout>
- Observed: pinned benchstat: 'ZeroAlloc-4,0,0%,1,0%,?,p=0.000 n=10'; check.sh printed only Slow-4 rows, 'benchci: 2 regression(s) above the limits'. CI log: 'Benchdata/loop.gno-4 0.000 +- 0% 0.000 +- 0%' under allocs/op
- Test: [`tests/solo-finder-zero-alloc-miss.sh`](tests/solo-finder-zero-alloc-miss.sh)

### 1 · Warning · CONFIRMED · `misc/benchci/suite.txt:14`

The cache line runs CacheStoreGet* at 20 iterations, so the gate fails A/A comparisons of an unchanged tree

- Check: REPS=250 bash tests/solo-finder-aa-cache-20x.sh <judge worktree>
- Observed: rep 56: REGRESSION CacheStoreGetKeyFound-16 sec/op: +23.63% (limit 20%, p=0.041 n=10); rep 132: REGRESSION CacheStoreGetNoKeyFound-16 sec/op: +28.27% (p=0.035); rep 225: REGRESSION CacheStoreGetKeyFound-16 sec/op: +20.02% (p=0.016); red 3/250. CI 42262327632: 'CacheStoreGetKeyFound-4 89.15n +- 66% 51.85n +- 141%'
- Test: [`tests/judge-1-aa-cache-20x.out`](tests/judge-1-aa-cache-20x.out)

### 7 · Nit · CONFIRMED · `gnovm/cmd/benchstore/store_test.go:445`

BenchmarkStoreGetCacheSweep still skips silently without -cache-sweep

- Check: cd gnovm/cmd/benchstore && go test -count=1 -timeout 2m -run '^$' -bench CacheSweep -db=pebbledb . (go1.25.9, parent, after the round)
- Observed: 'PASS' / 'ok  github.com/gnolang/gno/gnovm/cmd/benchstore 0.011s', no benchmark line; line 445 'b.Skip("use -cache-sweep=500,1024,... to run")' sits outside the diff hunk (head 437-443), so the section stays SKIP

### 6 · Nit · CONFIRMED · `misc/benchci/run.sh:22`

run.sh creates $out/bin without clearing it, so a reused output directory can run a binary left by an earlier run

- Check: Read run.sh lines 22, 35-39 and 48
- Observed: sed -n '22p;35,40p;48p' run.sh at head

### 4 · Suggestion · CONFIRMED · `.github/workflows/ci-bench.yml:12`

The gno.land/**/*.go trigger starts the job for changes no suite benchmark depends on

- Check: go list -deps -test ... \| grep -c gnolang/gno/gno.land
- Observed: go list -deps -test ./gnovm/pkg/gnolang ./tm2/pkg/sdk/auth ./tm2/pkg/store/cache ./tm2/pkg/bptree/benchmarks ./tm2/pkg/bft/wal ./tm2/pkg/crypto/secp256k1 ./tm2/pkg/amino \| grep -c gnolang/gno/gno.land: 0

### 3 · Nit · CONFIRMED · `misc/benchci/check.sh:23`

A benchmark present on one side only yields a 3-field CSV row that check.sh drops at NF < 7, so the gate reports no regression with that row uncompared

- Check: Edit suite.txt or rename one benchmark on head, run run.sh + check.sh: expect a failure or warning, observe exit 0
- Observed: CSV 'WalRead10KB-16,2.7974e-05,inf' (NF=3); benchstat table 'WalRead10KB-16 27.97u +- inf' with an empty head column; check.sh 'benchci: no regression above the limits', exit 0

### 8 · Suggestion · PLAUSIBLE · `misc/benchci/check.sh:28`

B/op rows never fail the gate

- Check: Read check.sh:28
- Observed: 

### 5 · Suggestion · CONFIRMED · `tm2/pkg/bft/wal/wal_test.go:275`

Open gnolang/gno#6003 makes the same maxSize fix in the same helper and keeps the 10MB and 100MB sizes this branch deletes

- Check: gh pr diff 6003 -R gnolang/gno
- Observed: state OPEN, mergeable MERGEABLE, files [tm2/pkg/bft/wal/wal_test.go]; diff '+func TestWALWriterReaderLargeMessage', '+\tmaxSize := int64(n) + 64 // n + overhead.', '+\tif testing.Short() {'

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`ci-bench.yml:44`](https://github.com/gnolang/gno/blob/d4e636abc/.github/workflows/ci-bench.yml#L44) | wrong-base | merge-base HEAD origin/$BASE_REF on the pull_request merge commit returns the master tip, not the fork point | gh run view 42262327632 --log: 'HEAD is now at 6eecfa5f6 Merge d4e636abc... into e918a057d...' then 'base: e918a057d2c1dc0f2a7d45d6dfc70c81036ee437', the merge commit's first parent: the job compares base-branch tip against the merge result, the right pair; 'merge base' in README/ADR is wording only |
| [`check.sh:20`](https://github.com/gnolang/gno/blob/d4e636abc/misc/benchci/check.sh#L20) | silent-miss | check.sh may read the wrong CSV column for the pinned benchstat | benchstat -format csv on synthetic input: ',sec/op,CI,sec/op,CI,vs base,P' and 'Slow-4,...,+49.73%,p=0.000 n=10'; check.sh printed 'REGRESSION Slow-4 sec/op: +49.73% (limit 20%, ...)'; pinned run's stderr notes 'F20: all samples are equal' put the delta in column F ($6) |
| [`suite.txt:20`](https://github.com/gnolang/gno/blob/d4e636abc/misc/benchci/suite.txt#L20) | silent-miss | a suite.txt regexp may match no benchmark | run 42262327632 log has rows for all seven packages: Benchdata/*, CreateNewMachine, Encode/Decode x ComplexSt/FuzzBlock/InterfaceHeavy, WalRead512B/10KB/100KB, SaveVersion/bptree-fast/1k, Verification, AccountMapper* x4, AccountLoad1000, CacheStoreIterator1000, CacheStoreGet{NoKeyFound,KeyFound} |
| [`store_test.go:59`](https://github.com/gnolang/gno/blob/d4e636abc/gnovm/cmd/benchstore/store_test.go#L59) | fallout | b.Fatal without -db may turn an existing -bench job red | grep -rn -bench over Makefile/*.mk/*.yml/*.sh: only gnovm/Makefile calibrate targets (./$(CALIBRATE_DIR)/, ./pkg/gnolang/), tm2/pkg/iavl/Makefile, run-disk-bench.sh and misc/benchci; none reaches gnovm/cmd/benchstore |
| [`wal_test.go:315`](https://github.com/gnolang/gno/blob/d4e636abc/tm2/pkg/bft/wal/wal_test.go#L315) | claims | claim that consensus opens its WAL at maxMsgSize = 1MB | tm2/pkg/bft/consensus/state.go:400 'wal, err := walm.NewWAL(walFile, maxMsgSize)'; reactor.go:29 'maxMsgSize = 1048576 // 1MB' |
| [`wal_test.go:317`](https://github.com/gnolang/gno/blob/d4e636abc/tm2/pkg/bft/wal/wal_test.go#L317) | claims | BenchmarkWalRead1MB may still fail after the helper fix | go test -run '^$' -bench 'WalRead1MB$' -benchtime 3x ./tm2/pkg/bft/wal at head: 'BenchmarkWalRead1MB-16 3 2007982 ns/op 8439250 B/op 368 allocs/op', ok |
| [`wal_test.go:317`](https://github.com/gnolang/gno/blob/d4e636abc/tm2/pkg/bft/wal/wal_test.go#L317) | removed | 10MB/100MB/1GB WAL benchmarks may have been added at someone's request | git log -S BenchmarkWalRead1GB at merge base: 'c4c0d85b3 2021-04-03 backup, so anyone can continue the good work', the initial import |
| [`check.sh:26`](https://github.com/gnolang/gno/blob/d4e636abc/misc/benchci/check.sh#L26) | catalog | invariant catalog classes against the diff | diff touches CI tooling, benchmark and test files only (no GnoVM, stdlib or .gno code); the catalog applies to gno code per the review rules, no class touched |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | misc/benchci/suite.txt:14 | REPS=250 bash tests/solo-finder-aa-cache-20x.sh <judge worktree> | hot | fixed-iteration-noise-floor |
| 2 | CONFIRMED | Warning | misc/benchci/check.sh:26 | bash tests/solo-finder-zero-alloc-miss.sh <head checkout> | warm | zero-base-delta-skipped |
| 3 | CONFIRMED | Nit | misc/benchci/check.sh:23 | Edit suite.txt or rename one benchmark on head, run run.sh + check.sh: expect a failure or warning, observe exit 0 | warm |  |
| 4 | CONFIRMED | Suggestion | .github/workflows/ci-bench.yml:12 | go list -deps -test ... \| grep -c gnolang/gno/gno.land | hot |  |
| 5 | CONFIRMED | Suggestion | tm2/pkg/bft/wal/wal_test.go:275 | gh pr diff 6003 -R gnolang/gno | cold |  |
| 6 | CONFIRMED | Nit | misc/benchci/run.sh:22 | Read run.sh lines 22, 35-39 and 48 | warm |  |
| 7 | CONFIRMED | Nit | gnovm/cmd/benchstore/store_test.go:445 | cd gnovm/cmd/benchstore && go test -run '^$' -bench CacheSweep -db=pebbledb . | cold |  |
| 8 | PLAUSIBLE | Suggestion | misc/benchci/check.sh:28 | Read check.sh:28 | warm |  |

</details>

Hit rate per tier, from the record: hot 2/2 confirmed over 2 files, warm 3/4 confirmed over 2 files, cold 1/2 confirmed over 4 files.

## Completeness

- Catalog: `skills/invariant-catalog.md` covers GnoVM, stdlib and `.gno` code; the diff touches a workflow, two shell scripts, a suite file, docs and two test files, so no class applies.
- Siblings swept by shape at head: `b.Skip("TODO` over every `*_test.go` has no hit left; `b.Skip("use -` hits only `gnovm/cmd/benchstore/store_test.go:445` (row 7, PLAUSIBLE Nit, SKIP); the `NewWALWriter`/`NewWALReader` pairs in `wal_test.go` are 55/59 on `maxTestMsgSize` and 277/294 on `maxSize`, no mismatched pair left.
- Reflector added rows 7 (silent `-cache-sweep` skip) and 8 (`B/op` has no floor), both SKIP. The parent ran row 7 from the package directory after the round: `PASS` in 0.011s with no benchmark line, so CONFIRMED; it stays SKIP because line 445 sits outside the diff and failing there would redden every `-bench . -db=pebbledb` run. Row 8 stays PLAUSIBLE, its same-tree `B/op` run not made.
- Parent's final check: row 3 moved from Suggestion to Nit, since its fix, failing on a one-sided row, was not applied and run, and a `Suggestion:` naming no edit is the wrong band. Row 4 keeps its deletion of line 12 as a suggestion block: the `go list -deps -test` run over every suite package is the check that a `gno.land/` change cannot move a suite benchmark.
- Row 3 re-anchored from `run.sh:48` to `check.sh:23`: the base-only CSV row is `WalRead10KB-16,2.7974e-05,∞`, NF=3, dropped by `NF < 7`.
- Row 1 rerun by the judge: 3 of 250 same-tree runs red on this machine (`tests/judge-1-aa-cache-20x.out`), deltas +20.02% to +28.27%; the finder's +78.95% is from its own run and stays out of the draft.
- Row 2 rerun with the pinned benchstat `v0.0.0-20260112171951-5abaabe9f1bd`: same CSV and `check.sh` output as the newer build.
- Row 5 (gnolang/gno#6003 overlap) ships SKIP: a posted comment never names a competing pull request; the user raises it.
- Not done this round: the draft's repro blocks were not extracted and rerun verbatim; they match the `tests/` scripts that were run, minus the deltas log. The fixes the findings imply (a 100ms line for `CacheStoreGet.*`, failing on `?` and one-sided rows) were not applied or run, so no section states a fix.
- PR body and threads: only the bot comment; the body's "raise `MAX_TIME_REGRESSION` if flaky" remedy is answered by row 1's deltas.

## Retro

- What failed: a first launch went out without the user's four topics and was stopped before any stage wrote a file. The judge left row 7 PLAUSIBLE with its check "still to run"; the parent ran it afterwards in 0.011s and it confirmed. No agent died, no cap was hit, both agents ran on claude-opus-5-5 with no fallback.
- What worked: the topics carried in one solo finder settled all four angles: wrong base cleared from the run log, silent misses gave the zero-alloc Warning and the one-sided-row Nit, flaky red gave the 20x Warning, fallout cleared the `-bench` callers and found the #6003 overlap. Both Warnings held under the judge's own rerun.
- Hit rate per tier: hot 2/2 over 2 files, warm 3/4 over 2 files, cold 1/2 over 4 files; hot above warm above cold, so the weights hold here.
- Upgrade, not applied: a judge whose check runs in seconds runs it rather than leaving a PLAUSIBLE "still to run", priced at one more judge turn, about 7k output. It changes a stage prompt, which waits for the review loop of *Editing this corpus* in `skills/authoring.md`.
- Measured: 2 agents, 38 turns, 102k output, 4.7M cache read, 0.27M cache write, about 19 minutes, about $5.14 by `./scripts/review-retro.py`.
