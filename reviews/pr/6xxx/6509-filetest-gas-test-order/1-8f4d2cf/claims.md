# Claims: gnolang/gno#6509 round 1, 8f4d2cfa3, claude-opus-5-5, quick review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`files_test.go:128`](https://github.com/gnolang/gno/blob/8f4d2cfa3/gnovm/pkg/gnolang/files_test.go#L128) | 3 | Suggestion | CONFIRMED | differs | posts |
| [`filetest.go:471`](https://github.com/gnolang/gno/blob/8f4d2cfa3/gnovm/pkg/test/filetest.go#L471) | 2 | Suggestion | CONFIRMED | same | posts |
| [`imports.go:248`](https://github.com/gnolang/gno/blob/8f4d2cfa3/gnovm/pkg/test/imports.go#L248) | 1 | Suggestion | CONFIRMED | differs | posts |

### 3 · Suggestion · CONFIRMED · `gnovm/pkg/gnolang/files_test.go:128`

The alloc_* fresh-store carve-out in TestFiles is no longer needed: the alloc_* goldens that are not _long pass on pooled stores at head and fail at the merge base

- Check: Drop the `strings.HasPrefix(subTestName, "alloc")` clause at files_test.go:128. Run `go test ./gnovm/pkg/gnolang -run 'TestFiles' -count=3` at head. Green means the carve-out is dead. Run the same at base to confirm it was needed there.
- Observed: sed 's/if isLong \|\| strings.HasPrefix(subTestName, "alloc") {/if isLong {/' files_test.go; go test -count=1 -run 'TestFiles/alloc' .: head ok 18.9s; base FAIL alloc_4, alloc_defer_gc, alloc_7, alloc_7a, alloc_6a, alloc_6, alloc_3, alloc_5, alloc_1, alloc_10c (first 10 shown). Head -count=3: ok 57.4s.

### 2 · Suggestion · CONFIRMED · `gnovm/pkg/test/filetest.go:471`

Realm filetests preprocess on a gasless forked allocator, so the preprocessing charge the PR adds skips every PKGPATH under /r/

- Check: Write a filetest with `// PKGPATH: gno.land/r/test/x` whose main has one `make([]byte, 10)` site, then a copy with two sites, and RunFiletest both. Do the same pair with no PKGPATH. Expected: the main pair's per-site gas delta includes the 113 preprocessing charge and the realm pair's does not. Add `txs.SetAllocator(m.Alloc)` after filetest.go:471 and the realm delta should gain it.
- Observed: tests/judge-2-realm-preprocess-gas_test.go, go test -run TestJudge2RealmGas -v in gnovm/pkg/test: head main literal=4172 make=7097 delta=2925, realm literal=9222 make=12022 delta=2800; base main delta=2812, realm delta=2800. The realm delta is the same at head and base, and 125 short of the head main delta.
- Test: [`tests/judge-2-realm-preprocess-gas_test.go`](tests/judge-2-realm-preprocess-gas_test.go)

### 1 · Suggestion · CONFIRMED · `gnovm/pkg/test/imports.go:248`

Gno run's machine and its store stop sharing an allocator, so store-side allocations leave the 500 MB cap and the GC

- Check: In both worktrees, add `fmt.Fprintln(os.Stderr, m.Store.GetAllocator() == m.Alloc)` after the machine is built at run.go:240. Run `go run ./gnovm/cmd/gno run x.gno` on a main-package file, and again with `-pkgpath gno.land/r/demo/x`. Expected: base prints true both times, head prints false both times.
- Observed: run.go patched to print `SHARED m.Store.GetAllocator() == m.Alloc`, GNOROOT set to the tree, go1.27.3: head `SHARED false` for plain and `-pkgpath gno.land/r/demo/x`; base `SHARED true` for both.

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`filetest.go:85`](https://github.com/gnolang/gno/blob/8f4d2cfa3/gnovm/pkg/test/filetest.go#L85) | removed | Replacing MaxAllocBytes with Alloc: NewAllocator(maxAlloc) changes behaviour for a zero or negative MAXALLOC: before, it silently became unlimited. | filetest.go:40-42 maps 0 to math.MaxInt64 before NewAllocator, so zero behaves as before. `grep -rn 'MAXALLOC: -\\|MAXALLOC: 0' gnovm/tests/files` returned no hits, so no filetest uses a non-positive cap, and a negative one now fails loudly instead of silently. |
| [`imports.go:248`](https://github.com/gnolang/gno/blob/8f4d2cfa3/gnovm/pkg/test/imports.go#L248) | catalog | A non-nil root allocator, written by root-store loads and read by every BeginTransaction Fork, could race once tests run in parallel. | cmd/gno/test.go:302-305 'One TestOptions (and store) per worker'. files_test.go:139-140 draws each store from optsPool and returns it only when the subtest ends, so no store is shared concurrently. preprocess_test.go:66-69 runs its shared-store subtests sequentially, with no t.Parallel. |
| [`imports.go:248`](https://github.com/gnolang/gno/blob/8f4d2cfa3/gnovm/pkg/test/imports.go#L248) | catalog | The PR body's claim that the root store gets its own allocator 'as VMKeeper.Initialize does', and whether on-chain gas moves. | keeper.go:275-276 `alloc := gno.NewAllocator(maxAllocTx)` and `vm.gnoStore = gno.NewStore(alloc, baseStore, iavlStore)`, so the parity claim holds. The diff header shows 22 files, all under gnovm/pkg/test and gnovm/tests/files, so no keeper or VM code changes. |
| [`test.go:345`](https://github.com/gnolang/gno/blob/8f4d2cfa3/gnovm/pkg/test/test.go#L345) | reach | Package tests run by `gno test` may report different gas now that tgs forks a non-nil root allocator. | test.go:351-363: at base, m2 (MaxAllocBytes MaxInt64, nil GasMeter) installed its allocator on tgs through machine.go:189. At head, tgs holds a gasless MaxInt64 fork. In both cases, runTestFiles overwrites m.Alloc (test.go:532, 557), so no gas meter ever saw the store-side allocations and the reported GAS line is unchanged. |
| [`preprocess.go:1979`](https://github.com/gnolang/gno/blob/8f4d2cfa3/gnovm/pkg/gnolang/preprocess.go#L1979) | catalog | With the root's own allocator, FuncValue copies made while preprocessing on the root store could get a different PkgID. | `grep -n 'func (fv \*FuncValue) Copy' -A 25 values.go` shows only `alloc.AllocateFunc()`, with no stampPkgID call, so the copy's ownership is independent of which allocator it gets. |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | gnovm/pkg/test/imports.go:248 | In both worktrees, add `fmt.Fprintln(os.Stderr, m.Store.GetAllocator() == m.Alloc)` after the machine is built at run.go:240. Run `go run ./gnovm/cmd/gno run x.gno` on a main-package file, and again with `-pkgpath gno.land/r/demo/x`. Expected: base prints true both times, head prints false both times. | cold |  |
| 2 | CONFIRMED | Suggestion | gnovm/pkg/test/filetest.go:471 | Write a filetest with `// PKGPATH: gno.land/r/test/x` whose main has one `make([]byte, 10)` site, then a copy with two sites, and RunFiletest both. Do the same pair with no PKGPATH. Expected: the main pair's per-site gas delta includes the 113 preprocessing charge and the realm pair's does not. Add `txs.SetAllocator(m.Alloc)` after filetest.go:471 and the realm delta should gain it. | cold |  |
| 3 | CONFIRMED | Suggestion | gnovm/pkg/gnolang/files_test.go:128 | Drop the `strings.HasPrefix(subTestName, "alloc")` clause at files_test.go:128. Run `go test ./gnovm/pkg/gnolang -run 'TestFiles' -count=3` at head. Green means the carve-out is dead. Run the same at base to confirm it was needed there. |  |  |

</details>

Hit rate per tier, from the record: hot 0/0 confirmed over 0 files, warm 0/0 confirmed over 0 files, cold 2/2 confirmed over 22 files.

## Completeness

- Angles thin: none of the three candidates was a Warning; the tests angle did not run as a finder, and the parent's mutations m1 and m2 cover it: the new test pins the `imports.go` half through `require.Same`, the 19 `gas/` goldens pin the `filetest.go` half.
- Catalog: Gas and Determinism are the classes the diff touches, both covered by candidates 1-3. No VM op, realm, banker, deposit, caller or decoder code changes, so the other classes have no material. Global mutable state: the root allocator is per store, and the finder's drop shows no store is shared across parallel subtests.
- Sibling sweep by shape: `NewStore(nil,` over non-test Go in gnovm, gno.land, misc and contribs returned no hit; every hit is a gnolang unit test building a bare store with no filetest runner. The machine-beside-a-separate-store-allocator shape in `gnovm/pkg/test/test.go` is the PR body's own follow-up. `gno run` and `gno repl` share the `StoreWithOptions` root, which is candidate 1.
- Toolchain: the judge's runs used go1.27.3 from PATH, not the pinned go1.25.9. The parent re-ran all three on go1.25.9 in fresh worktrees at head and base, results below.
- Parent re-run, go1.25.9: (1) `gno run` built from each tree with the `SHARED` print, plain and `-pkgpath gno.land/r/demo/x`: head `SHARED false` twice, base `SHARED true` twice. (2) The draft's realm repro block, extracted from the draft verbatim, at head: main delta 2925, realm delta 2800; the artifact test at base: main 2812, realm 2800. (3) `go test -count=1 -timeout 2m -run 'TestFiles/alloc'` with the `alloc` clause dropped: head ok; base 8 failures, `alloc_1`, `alloc_3`, `alloc_5`, `alloc_6`, `alloc_6a`, `alloc_7`, `alloc_7a`, `alloc_10c`, against the judge's 10, so the failing set varies by schedule and the draft says eight to ten.

## Retro

- Failed: the finder spent all 40 of its tool calls, the solo cap, on a 94-line diff. The judge ran its checks on go1.27.3 from PATH rather than the pinned go1.25.9, so the parent re-ran all three on the pinned toolchain; every figure held.
- Worked: all 3 candidates confirmed, 0 refuted. The `gno run` allocator regression is one the parent's own pass missed, found by the reach angle following `StoreWithOptions` out to its other callers.
- Hit rate: cold 2/2 over 22 cold files, plus candidate 3 on `files_test.go`, a file outside the diff with no tier; no hot or warm tier in the risk table.
- Upgrade, proposed and not applied: the judge's prompt carries the pinned toolchain's `PATH` export from the project's `AGENTS.md`, since a stage starts on `/usr/bin/go` otherwise. Estimate: no added cost; it removes the parent's re-run, three checks here.
- Cost: 2 agents, 60 turns, 94k output, 7.7M cache read, 0.36M cache write, about $6.27 measured by `./scripts/review-retro.py`, against $5 projected; 17 minutes wall clock.
