# PR #5811: perf(gnovm): parallelize test suites and add gno test -p

URL: https://github.com/gnolang/gno/pull/5811
Author: thehowl | Base: master | Files: 3 | +281 -144
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `d668a22c2` (stale — +18 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5811 d668a22c2`

**TL;DR:** Reworks the GnoVM test suites so they run in parallel, and adds a `-p N` flag to `gno test` (like `go test -p`) that tests N packages concurrently, each worker on its own store. Goal is faster CI: `pkg/gnolang` test time and the `gno test` stdlib/examples jobs both drop substantially.

**Verdict: REQUEST CHANGES** — the parallelization is functionally correct (`-p 1` vs `-p N` give identical result sets), but it is not race-free: `go test -race ./gnovm/pkg/gnolang/` is clean on master and fails on this PR. Concurrent VM runs write two process-global variables without synchronization. Both are benign value-wise today, but they are real data races (the same class the PR already fixed for `gnoBuiltinsCache`), and the same unsynchronized writes now run in the shipped `gno test` default-parallel path.

## Summary

`gno test` was single-threaded and the `pkg/gnolang` filetest/stdlib suites ran mostly sequentially. This PR parallelizes all three: `TestFiles` short tests now draw a store from a `GOMAXPROCS` pool and run as parallel subtests, `TestStdlibs` runs every package as a parallel subtest with its own store, and `gno test -p N` (default `GOMAXPROCS`, so parallel by default) runs N package-workers each owning a reused store. Output is buffered per package and flushed in completion order, matching `go test`. The PR found and fixed one latent race this exposes (`gnoBuiltinsCache`, now a `sync.OnceValue`); two others remain.

## Glossary
- **fallbackAllocator** — package-global `*Allocator` (`MaxInt64` budget, never enforced) used on the few value paths that have no Machine/store allocator, e.g. realm-save map copies.
- **filetest** — a `*_filetest.gno` run by the GnoVM and asserted against `// Output:`/`// Error:` golden directives.

## Fix
No fix proposed by the PR for the remaining races. The two racing globals predate this PR and are only made load-bearing by running multiple VMs in one process: `fallbackAllocator` (`alloc.go:45`) is mutated by `copyValueWithRefs` via `MapList.Append` (`realm.go:1695`), and the debug `enabled` flag (`debug.go:204-212`) is toggled by the realm filetest path (`filetest.go:443`/`:486`). On-chain execution is single-threaded so neither matters in production; they only race under this PR's test/`-p` parallelism.

## Benchmarks / Numbers
Author-reported, not re-measured here:

| | before | after |
|---|---|---|
| CI `stdlibs / test` | 8m33s | ~5m30s |
| CI examples `gno-checks / test` | 3m44s | ~3m10s |
| local `gno test gnovm/stdlibs/...` (4 cores) | 455s | 270s (`-p 4`) |
| local `go test ./pkg/gnolang/` long, 16 cores | 283.7s | 245.0s |

## Critical (must fix)
None.

## Warnings (should fix)
- **[parallelization makes `-race` fail; was clean on master]** [`gnovm/pkg/gnolang/files_test.go:116`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/gnolang/files_test.go#L116) · [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/files_test.go#L116) — Running the previously-sequential short filetests (and every stdlib package, and `gno test -p`) in parallel writes two process-global variables without synchronization, so `go test -race ./gnovm/pkg/gnolang/` now fails.
  <details><summary>details</summary>

  Two distinct global-state races, both surfaced only by this PR's concurrency:

  - **`fallbackAllocator`** — [`copyValueWithRefs`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/gnolang/realm.go#L1695) · [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/realm.go#L1695) calls `MapList.Append(fallbackAllocator, …)`, which does the non-atomic `alloc.bytes += size` in [`Allocator.Allocate`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/gnolang/alloc.go#L303-L327) · [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/alloc.go#L303) on the single [package-global](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/gnolang/alloc.go#L45) · [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/alloc.go#L45). Every worker/store hits it through `loadStdlib` → realm finalize, so two parallel runs race. Seen in both `TestFiles` and `TestStdlibs`.
  - **debug `enabled` flag** — the realm filetest path calls [`gno.DisableDebug()`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/test/filetest.go#L443) · [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/test/filetest.go#L443) then [`gno.EnableDebug()`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/test/filetest.go#L486) · [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/test/filetest.go#L486), flipping the package-global `enabled` in [`debug.go:203-209`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/pkg/gnolang/debug.go#L203-L209) · [↗](../../../../../.worktrees/gno-review-5811/gnovm/pkg/gnolang/debug.go#L203). Parallel filetests toggle it concurrently.

  Both are benign value-wise: `fallbackAllocator`'s budget is `MaxInt64` and never enforced/read for correctness, and `enabled` only matters when the compile-time `debug` flag is on (off in normal builds). But they are real data races (undefined behavior), and the same unsynchronized writes execute in the shipped `gno test` now that `-p` defaults to `GOMAXPROCS` — including the CI `gno test` steps this PR points at `GOMAXPROCS`. The regression matters beyond cosmetics: with the suite no longer `-race`-clean, `-race` can't be used to catch *real* races in `pkg/gnolang`.

  This is the same failure class as the `gnoBuiltinsCache` race the PR already fixed ("with type-checks now running concurrently from the start, that latent race becomes load-bearing"); these two globals are the analogous cases that were missed.

  Repro and observed output: [repro](comment_claude-opus-4-8.md).

  Fix: make parallel execution race-free — give the no-accounting/`fallbackAllocator` and debug-`enabled` paths per-worker (or synchronized) state instead of a shared global, or keep `-p`/the suites from exercising them concurrently. Same outcome as the `gnoBuiltinsCache` fix: no process-global mutated under concurrency.
  </details>

## Nits
- [`gnovm/cmd/gno/test.go:181-189`](https://github.com/gnolang/gno/blob/d668a22c2/gnovm/cmd/gno/test.go#L181-L189) · [↗](../../../../../.worktrees/gno-review-5811/gnovm/cmd/gno/test.go#L181) — `-debug-addr` is dropped from `gno test`. It was a dead flag (registered but never read on master, confirmed: `git show origin/master:gnovm/cmd/gno/test.go` only references it in the struct and `RegisterFlags`), so this is safe cleanup; the only behavior change is that `gno test -debug-addr …` now errors instead of being silently ignored. No action needed; flagging the user-visible CLI delta.

## Missing Tests
- None blocking. The new `-p` path has no direct unit test in `gnovm/cmd/gno` (failfast-with-`-p`, the `[test system panic]` isolation path, completion-order flush), but these are integration-shaped and CI exercises `-p GOMAXPROCS` end-to-end. Worth a follow-up, not a blocker.

## Suggestions
- None.

## Open questions
- Parallel-by-default (`-p 0` → `GOMAXPROCS`) changes `gno test`'s default output from package order to completion order, and surfaces any latent global race (like the two above) on ordinary `gno test` runs. Intended and documented in the PR; noting because it widens the blast radius of the race finding to normal usage. Not posted separately.
- `gno test -p >1` flushes a package's captured stdout and stderr as two separate blocks (all out, then all err) rather than interleaved as `-p 1` does. Cosmetic; not posted.

## Verification

Built `gno` from the PR worktree (`go build ./gnovm/cmd/gno`) and ran live; all checks at d668a22c2:

- **`-p 1` vs `-p 4` equivalence**: identical result sets (durations stripped, sorted) over 8 example packages including a deliberate setup-failure (`svg/filetests`, no `gnomod.toml`) that both report as `FAIL … [setup failed]`. Confirms parallel scheduling doesn't change pass/fail.
- **Race regression (the Warning)**: `go test -race -short ./gnovm/pkg/gnolang/` — master passes (0 races, `ok`), this PR fails. `TestFiles` (subset, 66s): race on `fallbackAllocator`. `TestStdlibs -short` (663s): 6 races, all `fallbackAllocator`. `TestFiles -short` full (237s): 12 races, `fallbackAllocator` + debug `enabled`. Same `go test -race` command on the master worktree (`origin/master`, sequential short filetests): 0 races, `ok`.
- **CI**: `docs` job red — unrelated, the docs URL linter (`make lint` in `misc/docs/tools/linter`, `treat-urls-as-err=true`) hitting a dead external link; this PR touches no docs. All other jobs green.
- **gotypecheck.go**: the `gnoBuiltinsCache` → `sync.OnceValue` change is correct; the returned `*MemPackage` is read-only during type-check (same sharing semantics as the old lazy cache), so no new race there.
