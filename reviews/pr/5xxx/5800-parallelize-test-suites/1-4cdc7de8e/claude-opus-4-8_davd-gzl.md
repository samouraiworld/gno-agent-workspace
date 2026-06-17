# PR #5800: perf(gnovm): parallelize test suites to cut CI time

URL: https://github.com/gnolang/gno/pull/5800
Author: thehowl | Base: master | Files: 4 | +265 -140
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `4cdc7de8e` (stale — +8 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5800 4cdc7de8e`

**TL;DR:** Restructures the gnovm test suites (`TestFiles`, `TestStdlibs`) to run in parallel and adds a `-jobs N` flag to `gno test` so CI can test packages concurrently. No test content changes; the goal is wall-clock CI time (`stdlibs / test` 8m33s → 5m21s). The risk is concurrency safety, since the VM and its test harness lean on process-global state.

**Verdict: REQUEST CHANGES** — the design is sound and the CI win is real, but parallelizing `TestFiles`/`TestStdlibs` regresses `go test -race ./gnovm/pkg/gnolang` from clean (master) to 24 data races; only the `gnoBuiltinsCache` race was fixed, the `enabled` debug global and a shared `*Allocator` remain. Separately, `gno test -jobs -1` hangs forever. Both are fixable in-PR or can be consciously deferred with a note; flagging so the decision is explicit rather than silent.

## Summary
Two commits. First parallelizes the in-package test suites: `TestFiles`'s 2339 non-long filetests now run as parallel subtests drawing stores from a `GOMAXPROCS`-sized pool, and every `TestStdlibs` package runs as a parallel subtest with its own store. Second adds `gno test -jobs N` (default 1, unchanged): N workers, each owning a store/typecheck-cache reused across the packages it runs, with per-package output buffered and flushed in package order. CI passes `-jobs 4` on the stdlibs and examples jobs. The mechanics are correct: per-worker store isolation, deterministic in-order output, and a `proxyWriter` that swaps each package's writers cleanly. The hazard is that the suites were race-clean on master only because they ran sequentially; making them concurrent surfaces shared-global state the harness was silently relying on.

## Glossary
- `TestFiles` / `TestStdlibs` — the two heavyweight gnovm in-package test suites (`gnovm/pkg/gnolang/files_test.go`); the bulk of `main / test` CI time.
- filetest — a `*_filetest.gno` run by the VM and matched against `// Output:` / `Realm:` golden directives.
- store / `TestOptions` — per-test VM state; one store loads stdlib packages once and reuses them.
- `-jobs N` — new `gno test` flag: test up to N packages concurrently.
- `enabled` — process-global bool in `gnovm/pkg/gnolang/debug.go` gating debug + store-op logging.

## Fix
`gno test`'s old single loop over `pkgs` (shared store, live output) is split into a sequential path (`-jobs 1`, byte-identical to before) and a parallel path: `jobs` worker goroutines pull package indices off an `atomic.Int64`, each writing into a per-package `bytes.Buffer` pair that a collector drains in index order. The body that loads + type-checks + runs one package is extracted into `testPkg`. `gnoBuiltinsCache` moves from lazy-populated to eager `init`-time + read-only, removing one type-check race. The pieces are individually correct; the gap is that the suite-level parallelism (commit 1) wasn't validated under `-race`.

## Critical (must fix)
- **[`go test -race` goes from clean to 24 races]** [`files_test.go:114-127`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/pkg/gnolang/files_test.go#L114-L127) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/pkg/gnolang/files_test.go#L114-L127) — parallelizing the filetests exposes pre-existing shared-global races the sequential suite hid; only `gnoBuiltinsCache` was fixed.
  <details><summary>details</summary>

  Under an identical invocation, master reports 0 races and this branch reports 24. The PR body notes the `gnoBuiltinsCache` race was fixed "now that type-checks run concurrently from the start" — but at least two more shared-state races remain, both reachable from `RunFiletest` running concurrently across pool stores.

  Class 1 — the process-global `enabled` bool. The realm filetest path calls [`gno.DisableDebug()`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/pkg/test/filetest.go#L440) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/pkg/test/filetest.go#L440) then [`gno.EnableDebug()`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/pkg/test/filetest.go#L483) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/pkg/test/filetest.go#L483), which write `enabled` at [`debug.go:204`/`:208`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/pkg/gnolang/debug.go#L204-L208) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/pkg/gnolang/debug.go#L204-L208). It is read by [`(*defaultStore).SetLogStoreOps`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/pkg/gnolang/store.go#L1214-L1216) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/pkg/gnolang/store.go#L1214-L1216) to decide whether to capture store ops. That capture feeds the `Realm:` directive, so this is not purely cosmetic: a concurrent toggle can flip store-op logging mid-test and corrupt a filetest's realm golden output.

  Class 2 — a shared `*Allocator`. `alloc.bytes += size` at [`alloc.go:326`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/pkg/gnolang/alloc.go#L303-L326) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/pkg/gnolang/alloc.go#L303-L326) is a non-atomic int64 increment. Two filetests on *different* pool stores nonetheless reach the same `*Allocator` (same address in both race stacks) via `copyValueWithRefs → AllocateMapItem → Allocate` on the object-save / `loadStdlib` paths — so distinct stores are sharing allocator state somewhere below `StoreWithOptions`. Likely benign for the result (filetests use `MaxAllocBytes=MaxInt64`, so a miscount won't trip the limit), but it is genuine UB and the larger share of the 24.

  Scope: production is single-threaded per block, so this is test-harness-only and CI stays green (the gnovm job runs coverage mode, not `-race`). But `go test -race ./gnovm/pkg/gnolang` is a standard health check for a core VM package, and it now drowns in harness races — real VM races would be invisible underneath. Fix: make `enabled` not toggle per-test (or serialize the realm path's debug toggle), and trace why per-test stores share an `*Allocator`. If a full fix is out of scope, say so in the PR and note that `go test -race` on this package is knowingly broken until a follow-up.

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5800 -R gnolang/gno
  GNOROOT=$PWD go test -race -short -run 'TestFiles/a' ./gnovm/pkg/gnolang/ 2>&1 | grep -c 'DATA RACE'
  ```
  ```
  24
  ```
  Baseline, same clone on master:
  ```bash
  # from a local clone of gnolang/gno:
  git checkout master
  GNOROOT=$PWD go test -race -short -run 'TestFiles/a' ./gnovm/pkg/gnolang/ 2>&1 | grep -c 'DATA RACE'
  ```
  ```
  0
  ```
  One representative stack (the `enabled` global):
  ```
  WARNING: DATA RACE
  Write at 0x...a821 by goroutine 902:
    gnolang.DisableDebug()  debug.go:204
    test.(*TestOptions).runTest()  filetest.go:440
  Previous read at 0x...a821 by goroutine 989:
    gnolang.(*defaultStore).SetLogStoreOps()  store.go:1215
    test.(*TestOptions).runTest()  filetest.go:486
  ```
  </details>

## Warnings (should fix)
- **[silent infinite hang on bad input]** [`test.go:270-278`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/cmd/gno/test.go#L270-L278) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/cmd/gno/test.go#L270-L278) — `gno test -jobs -1` deadlocks forever with no output and no error.
  <details><summary>details</summary>

  A negative `-jobs` falls through the `== 0 → GOMAXPROCS` guard unchanged, so `jobs = min(-1, len(pkgs)) = -1`, and [`for range jobs`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/cmd/gno/test.go#L289) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/cmd/gno/test.go#L289) with a negative count spawns zero workers. The collector then blocks on `<-res.done` for index 0, which nothing will ever close. The flag help documents `0` but not negatives, and the failure mode is the worst kind (hang, not error). Fix: reject `jobs < 0` (or clamp to 1) right next to the existing `-jobs`/`-debug` validation a few lines up.

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5800 -R gnolang/gno
  GNOROOT=$PWD timeout 25 go run ./gnovm/cmd/gno test -C examples -jobs -1 \
    ./gno.land/p/moul/once/ ./gno.land/p/moul/fifo/
  echo "exit=$?  # 124 = hung (timed out); produced no output and no error"
  ```
  ```
  exit=124  # 124 = hung (timed out); produced no output and no error
  ```
  </details>

## Nits
- [`test.go:329-353`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/cmd/gno/test.go#L329-L353) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/cmd/gno/test.go#L329-L353) — head-of-line blocking in the collector: the loop drains results strictly in index order, so a slow package 0 holds every finished package's output buffered in memory until it completes. Fine for normal test output; for verbose `./...` over hundreds of packages it can pin a few MB. Acceptable tradeoff for deterministic ordering, just worth knowing.

## Missing Tests
- **[new scheduler is only validated by CI wall-clock]** [`test.go:259-318`](https://github.com/gnolang/gno/blob/4cdc7de8e/gnovm/cmd/gno/test.go#L259-L318) · [↗](../../../../../.worktrees/gno-review-5800/gnovm/cmd/gno/test.go#L259-L318) — no unit test covers the `-jobs` parallel path.
  <details><summary>details</summary>

  The worker pool, in-order buffered output, parallel `-failfast` (stop scheduling after first failure, in-flight finish), and the `-jobs`/`-debug` rejection are all new control flow with zero direct coverage. A small test in `gnovm/cmd/gno` asserting that `-jobs 4` and `-jobs 1` produce identical status-line output on a fixed package set, plus the `-jobs -1`/`-debug` guards, would lock the behavior and catch the hang above. I confirmed `-jobs 4` vs `-jobs 1` output ordering matches manually (both sort by package path), but that's not regression-protected.
  </details>

## Questions for Author
- Did you run `go test -race` on the parallelized suites? It reports 24 races on this branch vs 0 on master (repro above) — were these known/accepted, or missed?
- `gno test -update-golden-tests` with `-jobs >1` is not forced sequential, unlike `TestFiles` which sets `poolSize=1` under `*withSync`. Intentional? It looks safe since `gno test` golden writes are per-package (independent files), but the asymmetry with `files_test.go` is worth a one-line confirmation.
