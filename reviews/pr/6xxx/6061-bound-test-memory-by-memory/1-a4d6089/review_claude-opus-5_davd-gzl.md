# PR [#6061](https://github.com/gnolang/gno/pull/6061): fix(tests): bound test memory by memory, not by core count

URL: https://github.com/gnolang/gno/pull/6061
Author: thehowl | Base: master | Files: 11 | +758 -5
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: a4d6089 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6061 a4d6089`
Overview: [visual overview](../overview.html)

## Overview

Running the gno test suites on a workstation can exhaust its memory, because the heavyweight suites size their worker count off the CPU count while each worker costs hundreds of megabytes of heap. This change caps the expensive thing in each of the three suites instead of the parallelism: the filetest store pool and the `examples` worker count take a static cap of four, and the integration suite's nodes are admitted against a live memory reading that ramps the allowance up while the machine reports room to spare and walks it back down when it stops doing so. The reading comes from `/proc/meminfo` on Linux, narrowed by the process's cgroup memory limit, and from a new dependency elsewhere. Two environment variables pin the count and trace what the controller decided.

## Verdict: REQUEST CHANGES

The cgroup reading counts reclaimable page cache as used, so the suite this change protects runs 42% longer inside a container than it does on master, and the controller stops responding at the one reading that means the machine really is out of memory. 3 Warnings, 7 Nits, 3 Missing tests, 5 Suggestions; 14 posted, 4 held as comment-wording or unreachable.

## Verify first

- [`tm2/pkg/testutils/parallel_linux.go:64`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L64) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L64) — the free figure a container gets. Run the integration suite under `GNO_TEST_TRACE_BUDGET=1` in any cgroup with a `memory.max` and compare the `avail=` field against `/proc/meminfo`'s own `MemAvailable`.
- [`gno.land/pkg/integration/testscript_gnoland.go:201`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L201) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L201) — the shrink arm is the only one that can reach the floor, and the floor is half the concurrency CI validates. Time a full `TestTestdata` run against master on the machine you develop on.

## Summary

Three suites lose their dependence on `GOMAXPROCS`. [`gnovm/pkg/gnolang/files_test.go:66`](https://github.com/gnolang/gno/blob/a4d6089/gnovm/pkg/gnolang/files_test.go#L66) · [↗](../../../../../.worktrees/gno-review-6061/gnovm/pkg/gnolang/files_test.go#L66) sizes its store pool with the new [`testutils.MaxParallel`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L40) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel.go#L40), a static `min(GOMAXPROCS, 4)`, and bounds the tests that build their own store with a second semaphore of the same size. [`examples/Makefile:28`](https://github.com/gnolang/gno/blob/a4d6089/examples/Makefile#L28) · [↗](../../../../../.worktrees/gno-review-6061/examples/Makefile#L28) passes a flat `-p 4`. The integration suite gets a feedback controller, [`nodeBudget`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L114) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L114), whose allowance starts at four and steps once per second towards `GOMAXPROCS` while the reading shows room for another node, and down towards two when the reading falls under a quarter of the machine's memory.

On this six-core host the cap is a clean win for the filetest pool and for `examples`, and a loss for the integration suite, whose controller reads a container as starved when it is not.

The branch is red on [`ci / codegen-verify`](https://github.com/gnolang/gno/actions/runs/31622314104): `make tidy VERIFY_MOD_SUMS=true` reports `contribs/gnobro`, `contribs/gnodev` and `contribs/gnogenesis` out of date, and `gnodev` and `gnobro` stop building for darwin and windows until that lands. The cause is the shape rather than the omission: a non-test file in `tm2/pkg/testutils` imports a third-party library, and every module reaching that package inherits it. `make tidy` from the repo root covers all three.

## Benchmarks / Numbers

Full `TestTestdata`, 188 scripts, this host, cgroup limit 10.00 GiB:

| | master `7547806` | head `a4d6089` |
|---|---|---|
| peak RSS | 5011 to 5155 MiB | 3345 to 4031 MiB |
| wall | 167.6 to 185.0s | 243.5 to 262.8s |
| time at or under the floor of two | | 238.3s of 262.8s |

`TestFiles`, filetests `m` through `t`, pool of six against pool of four, medians of three runs:

| | pool 6 | pool 4 |
|---|---|---|
| peak RSS | 2004 MiB | 1521 MiB |
| wall | 74.8s | 70.7s |

`gno test` over 34 `examples` packages on six cores, where the flat `-p 4` is a cap:

| | `-p 6` | `-p 4` |
|---|---|---|
| peak RSS | 1945 MiB | 1353 MiB |
| wall | 37.5s | 35.5s |

The same flag on `./gno.land/p/moul/...` with `GOMAXPROCS=2`, where it is a raise. Six-core container first, then the same block re-run on a 16-core host:

| | `-p 2` | `-p 4` |
|---|---|---|
| peak RSS, six-core container | 602 MiB | 1097 MiB |
| wall, six-core container | 7s | 11s |
| peak RSS, 16-core host | 658 MiB | 1113 MiB |
| wall, 16-core host | 2s | 2s |

The memory penalty reproduces on both; the wall-time penalty is the container's alone, so only the memory claim is posted.

## Warnings (should fix)

- **[reads pressure that is not there]** `tm2/pkg/testutils/parallel_linux.go:64` — `memory.max - memory.current` counts the reclaimable page cache as used, so inside a container the allowance falls to its floor and the suite takes 42% longer than it does on master.
  <details><summary>details</summary>

  `memory.current` is anon plus page cache plus kernel memory, while the figure this clamp narrows [accounts for reclaimable caches](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L52-L55) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel.go#L52-L55) and counts them as free. A suite that reads its own testdata therefore charges every cached byte against its own budget. Sampled every two seconds across a run of the same suite on this host, the cgroup's `file` counter grew from 0.70 GiB to 1.43 GiB, the clamped reading fell under the reserve in 33 of 133 samples, and the kernel's own `MemAvailable` fell under it in none. Fix: subtract the reclaimable pages, `inactive_file` and `slab_reclaimable` from `memory.stat`, before treating the remainder as used.

  **Repro:** [`tests/cgroup_cache_test.go`](tests/cgroup_cache_test.go) reads the live figures in a container with a `memory.max`, and fails once the cache it ignores is a large share of `memory.current`:

  ```
  cgroup limit=10.00GiB current=9.77GiB inactive_file+slab_reclaimable=3.26GiB
  ReadMemInfo Available=0.23GiB, cache-corrected=3.49GiB
  reserve=Total/4=2.50GiB shrinks=true (corrected would shrink=false)
  ```

  What the controller then does, from [`tests/trace-run.md`](tests/trace-run.md), its own trace over the full suite with timestamps added:

  ```
  t+   2.6s nodeBudget: limit=6 running=5 max=6 avail=4.53GiB reserve=2.50GiB
  t+  21.4s nodeBudget: limit=5 running=5 max=6 avail=2.42GiB reserve=2.50GiB
  t+  22.4s nodeBudget: limit=4 running=5 max=6 avail=2.26GiB reserve=2.50GiB
  t+  23.4s nodeBudget: limit=3 running=4 max=6 avail=2.26GiB reserve=2.50GiB
  t+  24.5s nodeBudget: limit=2 running=3 max=6 avail=2.08GiB reserve=2.50GiB
  t+ 262.8s PEAK_TREE_RSS_MiB=4031  WALL_S=262.8
  ```

  Nothing after t+24.5s: the allowance reached the floor in 22 seconds and stayed there for the remaining 91% of the run. Master finishes the same 188 scripts in 185.0s at 5011 MiB on the same box.
  </details>

- **[the feedback loop stops where it is needed most]** `tm2/pkg/testutils/parallel_linux.go:51` — a cgroup sitting at its limit reports `Available == 0`, which `ReadMemInfo` returns as a failed reading, and [`resize`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L193-L195) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L193-L195) treats a failed reading as a reason to leave the allowance alone.
  <details><summary>details</summary>

  `clampToCgroup` yields `Available == 0` whenever `memory.current` reaches `memory.max`, which the branch's own [`TestClampToCgroup/usage_at_the_limit`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux_test.go#L39-L44) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux_test.go#L39-L44) asserts. One byte of headroom walks the allowance to the floor; zero bytes freezes it at whatever it had reached. Fix: let `ok` mean the figures were read, since the shrink arm already handles a reading of zero.

  **Repro:** two budgets identical but for the last available byte, both saturated, both ticked 25 times with the settle window expired:

  ```
  zero-available:     limit=8 running=8 min=2 readMem calls=33
  one-byte-available: limit=2 running=7 min=2 readMem calls=26
  ```
  </details>

- **[a cap that raises the count]** `examples/Makefile:28` — the flat `GNOTEST_JOBS ?= 4` is above `GOMAXPROCS` on any host under four cores, so it raises worker count and memory where the rest of the change lowers them.
  <details><summary>details</summary>

  [`gno test -p`](https://github.com/gnolang/gno/blob/a4d6089/gnovm/cmd/gno/test.go#L282-L286) clamps the count only against the number of packages, never against the core count, so a two-core laptop goes from two workers on master to four here. The filetest pool in the same change takes `min(GOMAXPROCS, 4)`. The env var the change establishes does not reach this suite either, `gno test` never reading it. Fix: derive the default from `testutils.MaxParallel` rather than writing a literal, which also gives the CI invocation the same bound.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno, standing in for a two-core host:
  gh pr checkout 6061 -R gnolang/gno
  go build -o /tmp/gno ./gnovm/cmd/gno
  cd examples
  for p in 2 4; do
    start=$SECONDS
    GOMAXPROCS=2 /tmp/gno test -p $p ./gno.land/p/moul/... >/dev/null 2>&1 &
    pid=$!; peak=0
    while kill -0 $pid 2>/dev/null; do
      v=$(awk '/VmHWM/{print $2}' /proc/$pid/status 2>/dev/null)
      [ -n "$v" ] && [ "$v" -gt "$peak" ] && peak=$v
      sleep 0.05
    done
    echo "-p $p: peak $((peak / 1024)) MiB, wall $((SECONDS - start))s"
  done
  ```

  ```
  -p 2: peak 658 MiB, wall 2s
  -p 4: peak 1113 MiB, wall 2s
  ```
  </details>

## Nits

- **[unpaced readings on the failure path]** `gno.land/pkg/integration/testscript_gnoland.go:193` — the early return skips `b.lastCheck = time.Now()`, so a reading that fails is retaken on every poll by every blocked script, which is the cost the pacing at [line 196](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L196-L199) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L196-L199) exists to avoid, a `vm_stat` subprocess per reading on darwin. Fix: stamp `lastCheck` before the read rather than after it.
  <details><summary>details</summary>

  Twenty polls of a budget sitting inside the hysteresis band, against twenty polls of one whose reading fails:

  ```
  readable-in-band=1 reads, unreadable=20 reads
  ```
  </details>

- **[token returned before the node it paid for]** `gno.land/pkg/integration/testscript_gnoland.go:271` — the node teardown is registered during setup and the release later, and testscript runs deferred functions in reverse, so a waiting script is admitted while the finishing node is still alive.
  <details><summary>details</summary>

  [`Env.Defer`](https://github.com/rogpeppe/go-internal/blob/v1.15.0/testscript/testscript.go#L88-L90) delegates to [`TestScript.Defer`](https://github.com/rogpeppe/go-internal/blob/v1.15.0/testscript/testscript.go#L912-L918), which chains each new function in front of the previous one. The stop and the `nodesManager.Delete` sit in the setup callback at [line 378](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L378-L389) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L378-L389), so they run last. The overshoot is bounded by how long `Stop` takes and the store is collectable only at the next GC either way, so this is accounting slack rather than a leak. Fix: give the token back inside the same deferred function that stops the node.

  **Repro:** [`tests/slot_release_order_test.go`](tests/slot_release_order_test.go), which registers the two functions at the two positions the code uses:

  ```
  expected: []string{"node stop", "budget release"}
  actual  : []string{"budget release", "node stop"}
  ```
  </details>

- **[unfloored counter]** `gno.land/pkg/integration/testscript_gnoland.go:220` — `release` decrements without a floor, so a second release for one slot drives `running` negative and admits a node past `max` for the rest of the run. Unreachable today, `slot.held` at [line 266](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L266-L268) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L266-L268) gating the acquire, and the guard and the counter sit in different functions with no test between them. Fix: floor the decrement at zero. Not posted: nothing reaches it today.

- **[the comment names a reading the function never takes]** `tm2/pkg/testutils/parallel.go:14` — `defaultMaxParallel` is documented as the count used "when the machine's memory can't be read", while `MaxParallel` returns `min(GOMAXPROCS, 4)` unconditionally and calls `ReadMemInfo` nowhere. Fix: call it the static cap the suites that must fix their pool size up front take. Not posted: comment wording, with no behaviour behind it.

- **[the pool is not sized by memory]** `gnovm/pkg/gnolang/files_test.go:61` — "the pool is sized by memory rather than by GOMAXPROCS" describes a feedback loop that lives only in `nodeBudget`; the pool takes `min(GOMAXPROCS, 4)`. Not posted: comment wording, with no behaviour behind it, and fixing the `parallel.go:14` wording leaves this line untouched.

- **[a constant quoted from the wrong end of the range]** `gno.land/pkg/integration/testscript_gnoland.go:74` — `nodeMemCost` is described as "~580 MiB measured over the range above, rounded up", which is the average across the endpoints; the marginal cost is 973 MiB per node between two and four nodes, the band the shrunken allowance actually lives in.
  <details><summary>details</summary>

  From the six-point table in the description: 973 MiB per node from two to four, 717 from four to six, 512 from six to eight, 307 from eight to twelve, 666 from twelve to sixteen, 592 across the whole span. A growth threshold set below the local marginal cost admits a node the machine cannot pay for. Fix: quote it as the average across the measured range, or take it from the low end where the growth decision is made.
  </details>

- **[the reason given is not the reason]** `gno.land/pkg/integration/testscript_gnoland.go:259` — the comment justifies holding the token to the end of the script by a deadlock in which "every holder waiting for one more would deadlock once the budget is full", which cannot happen: [`gnoland start`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L447-L451) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L447-L451) rejects a second node for the same script, so no script ever needs two tokens. The behaviour is right and the token is genuinely held to the end. Fix: give the real reason, that a script stopping and restarting would otherwise queue behind fresh scripts. Not posted: the behaviour is right and only the comment's stated reason is wrong.

## Missing Tests

- **[the branch a Warning rides on]** `gno.land/pkg/integration/testscript_gnoland.go:193` — no test reaches `resize` with a reading that fails.
  <details><summary>details</summary>

  [`TestNodeBudgetStaticWhenMemoryUnknown`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/node_budget_test.go#L107-L120) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/node_budget_test.go#L107-L120) reads as coverage of it but leaves `reserve` zero, so `resize` returns at its first line and `readMem` is never called. Coverage confirms it: `testscript_gnoland.go:193.9,195.3` has count 0. Fix: add the case below, which fails while the reading is discarded and passes once `ok` means the figures were read.

  ```go
  func TestNodeBudgetShrinksWhenReadingFails(t *testing.T) {
  	t.Parallel()

  	// A cgroup at its memory.max clamps Available to 0, which ReadMemInfo
  	// reports as ok=false. That is maximum pressure, not a missing platform.
  	b := &nodeBudget{
  		min: 2, max: 8, limit: 8, reserve: 8 << 30,
  		readMem: func() (testutils.MemInfo, bool) {
  			return testutils.MemInfo{Total: 32 << 30, Available: 0}, false
  		},
  	}
  	for range 8 {
  		require.True(t, b.tryAcquire())
  	}
  	for range 10 {
  		elapse(b)
  		b.tryAcquire()
  	}
  	assert.Equal(t, 2, b.limit, "held its ramp with no memory left")
  }
  ```
  </details>

- **[the entry point the suite calls]** `gno.land/pkg/integration/testscript_gnoland.go:162` — every case drives `tryAcquire`, so `acquire`, the blocking poll loop, is at zero coverage on a type whose whole job is bounding concurrent admission.
  <details><summary>details</summary>

  Also untested: that the allowance holds while slots sit free, which is what [line 203](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L203-L207) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L203-L207) restricts growth for. Fix: add a blocking case and a contention case, which take `acquire` from 0% to 100% under `-race`.

  ```go
  // acquire blocks rather than returning, and wakes once a holder releases.
  func TestNodeBudgetAcquireBlocksUntilRelease(t *testing.T) {
  	t.Parallel()

  	b := &nodeBudget{min: 1, max: 1, limit: 1, readMem: memOf(0)}
  	require.True(t, b.tryAcquire())

  	done := make(chan struct{})
  	go func() {
  		b.acquire()
  		close(done)
  	}()
  	select {
  	case <-done:
  		t.Fatal("acquire returned with the budget full")
  	case <-time.After(3 * nodeBudgetPoll):
  	}
  	b.release()
  	select {
  	case <-done:
  	case <-time.After(5 * time.Second):
  		t.Fatal("acquire did not wake after release")
  	}
  	assert.Equal(t, 1, b.running)
  }

  // The admitted count never exceeds the allowance under concurrent scripts.
  func TestNodeBudgetConcurrentAdmission(t *testing.T) {
  	t.Parallel()

  	b := &nodeBudget{min: 2, max: 4, limit: 4, reserve: 8 << 30, readMem: memOf(64 << 30)}
  	var live, peak atomic.Int64
  	var wg sync.WaitGroup
  	for range 64 {
  		wg.Go(func() {
  			b.acquire()
  			n := live.Add(1)
  			for {
  				p := peak.Load()
  				if n <= p || peak.CompareAndSwap(p, n) {
  					break
  				}
  			}
  			time.Sleep(time.Millisecond)
  			live.Add(-1)
  			b.release()
  		})
  	}
  	wg.Wait()
  	assert.LessOrEqual(t, peak.Load(), int64(4), "admitted more nodes than the allowance")
  	assert.Equal(t, 0, b.running, "running did not return to zero")
  }
  ```
  </details>

- **[unreachable from any test]** `tm2/pkg/testutils/parallel_linux.go:89` — `cgroupDir` and the `/proc/meminfo` parser bake their paths in as literals, so deleting the whole host-namespace fallback at [line 108](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L100-L109) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L100-L109) leaves the suite green.
  <details><summary>details</summary>

  The 63.6% coverage `cgroupDir` reports is incidental, from `TestReadMemInfo` walking whichever branch this machine has. The load-bearing untested case is a kernel with no `MemAvailable` line, where the reading must be reported absent rather than as no memory free. Fix: thread a root prefix through `ReadMemInfo`, `cgroupMemory` and `cgroupDir`, twelve changed lines, then point table cases at a `t.TempDir()` fixture. That takes `cgroupDir` to 100% and the file from 82.1% to 96.6%.

  ```go
  func ReadMemInfo() (MemInfo, bool) { return readMemInfo("/") }

  // readMemInfo reads the figures under root, "/" outside tests.
  func readMemInfo(root string) (MemInfo, bool) {
  	bz, err := os.ReadFile(filepath.Join(root, "proc/meminfo"))
  	...
  	if limit, used, ok := cgroupMemory(root); ok {

  func cgroupMemory(root string) (limit, used uint64, ok bool) {
  	dir, ok := cgroupDir(root)
  	...

  func cgroupDir(root string) (string, bool) {
  	mount := filepath.Join(root, cgroupMount)
  	bz, err := os.ReadFile(filepath.Join(root, "proc/self/cgroup"))
  	...
  		if dir := filepath.Join(mount, rel); fileExists(filepath.Join(dir, "memory.max")) {
  			return dir, true
  		}
  		return mount, fileExists(filepath.Join(mount, "memory.max"))
  ```

  The cases that close it are in [`tests/parallel_linux_cases_test.go`](tests/parallel_linux_cases_test.go).
  </details>

## Suggestions

- **[the floor is where a run stays]** `gno.land/pkg/integration/testscript_gnoland.go:72` — `nodeMinParallel` is half the count CI validates, and climbing back out needs a reading 0.625 GiB above the reserve, so a transient dip pins the suite for the rest of the run rather than costing it a few seconds. Measured over one full suite: the allowance reached two 22 seconds in and the remaining 91% of the run was spent there. Fix: stop the shrink at `MaxParallel()` and go below it only after several consecutive readings under the reserve.

- **[a limit on an ancestor is invisible]** `tm2/pkg/testutils/parallel_linux.go:79` — `cgroupMemory` reads `memory.max` from this process's own directory only, while cgroup limits are hierarchical: a systemd slice or a Kubernetes pod cgroup carries the limit while the leaf reads `max`. The clamp is then skipped entirely and the suite sizes itself against the host under a limit that will kill it, which is the failure the clamp exists to prevent. Fix: walk from the leaf to the mount root and take the smallest limit found.

- **[cgroup v1 gets the host's memory]** `tm2/pkg/testutils/parallel_linux.go:97` — `cgroupDir` keeps only the unified `0::` line, so under cgroup v1 no limit is found, yet `ReadMemInfo` still returns the host figures with `ok` true and the budget ramps against them. The description says the budget is static there. It is no worse than master, which used `GOMAXPROCS` outright. Fix: read `memory.limit_in_bytes`, or report no reading under v1, either of which stops the ramp.

- **[a dead dependency for 212 lines]** `go.mod:29` — `github.com/pbnjay/memory` last changed in July 2021, carries no tag, so dependabot can never move it, forks a `vm_stat` subprocess per reading on darwin, and the container limits this change writes by hand have been [an open issue](https://github.com/pbnjay/memory/issues/12) on it since February 2025. [`PHILOSOPHY.md:7`](https://github.com/gnolang/gno/blob/a4d6089/PHILOSOPHY.md?plain=1#L7) sets the bar at audited and minimal. Fix: inline the darwin and windows readings using `golang.org/x/sys`, already in the graph, which also replaces the library's own `unsafe` reinterpretation of a `sysctl` string.

- **[operating-system introspection in a test-helper package]** `tm2/pkg/testutils/parallel.go:47` — `MemInfo` and the [readings that fill it](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L16-L52) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L16-L52) describe the machine rather than any test, and this package's placement is what puts a memory library into the module graph of `gnodev` and `gnobro`. Fix: move the type and the platform files to a package about the operating system and leave the worker-count helpers here.

## Verified

Re-verified against the head on a 16-core host with no cgroup limit, every finding re-run where the host allows it:

- The cgroup clamp charges reclaimable cache: under a real `systemd-run --user --scope -p MemoryMax=2G` holding 1.81 GiB of page cache, `ReadMemInfo` reports `Available` 0.154 GiB where a reclaim-aware figure is 1.960 GiB, and the reserve of 0.50 GiB makes the controller shrink. This replaces the container-only repro in the posted comment.
- Zero available freezes the ramp: `limit=8` after 25 saturated ticks with `ok=false`, against `limit=2` with one spare byte.
- The proposed `TestNodeBudgetShrinksWhenReadingFails` fails on this head: `limit` 8 where 2 is correct.
- `TestNodeBudgetStaticWhenMemoryUnknown` leaves `reserve` zero and calls `readMem` zero times.
- Unpaced readings on the failure path: 1 read over 20 polls in the hysteresis band, 20 reads over 20 polls when the reading fails.
- `release` without a floor drives `running` to -1 after one acquire and two releases.
- Coverage on the head: `acquire` 0.0%, `resize` 83.3%, with block `193.9,195.3` at count 0.
- Deferred order: the token is released before the node stops, `[budget release, node stop]` against the required `[node stop, budget release]`.
- Deleting the host-namespace fallback at line 108 and replacing it with `return "", false` leaves `go test ./tm2/pkg/testutils/` green.
- `go mod why -m github.com/pbnjay/memory` from `contribs/gnodev` and `contribs/gnobro` both resolve through `gno.land/pkg/integration` and `tm2/pkg/testutils`.
- The `examples` `-p` block re-run verbatim: 658 MiB at `-p 2` against 1113 MiB at `-p 4`, wall time equal.

Not re-runnable here, carried from the original six-core container: the full `TestTestdata` wall times and the trace, since this host has no `memory.max`.


- The budget admits no more than its allowance under contention: 64 goroutines against a limit of four peaked at four admitted and drained to zero, under `-race`.
- `go test -race` is clean on `tm2/pkg/testutils` and on the new `gno.land/pkg/integration` cases.
- No deadlock between the node token and the `-no-parallel` write lock: a token is taken before `sequentialMu` at every site, and the two scripts using the flag pass with the budget active.
- `restart` correctly does not take a second token, and a script that stops its node keeps the one it has.
- Memory freed by the suite does not return to the reading it is measured by. After `runtime.GC()` released 1.5 GiB, `heapFree=1.50GiB heapReleased=0.00GiB`, resident size flat and `MemAvailable` unmoved 30 seconds later.
- The allocator filetests, which take their own store because pooled reuse moves their counts, pass at pool sizes one, two, four and eight. Filetests `m` through `t` hold their wall time from two stores to eight and are fastest at four, so the cap costs nothing here.

## Open questions

- The controller measures the whole machine while the thing it bounds is one process's heap, so the suite's own retained-but-idle heap reads as somebody else's pressure. Adding `runtime/metrics`' `/memory/classes/heap/free:bytes` back into the reading would make the signal mean what the controller assumes. Not posted: it is a design direction, and the container reading is the defect that bites first.
- `max` is `GOMAXPROCS` while the number of scripts that can run at once is `go test -parallel`. Lowering `-parallel` below the current allowance silently stops the ramp, growth requiring `running >= limit`. Not posted: neither case is wrong, and the trace line is the only place it shows.
- The description says the opt-out stores mean the total "cannot quietly double", while the comment beside them says the total is bounded at twice the pool. The comment is right and the change still cuts the bound from `2 * GOMAXPROCS` to `2 * MaxParallel()`. Carried into the posted body, since the wording is what a maintainer reads before the code.
