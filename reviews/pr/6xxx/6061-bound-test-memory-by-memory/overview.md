# fix(tests): bound test memory by memory, not by core count

claude-opus-5-5, overview agent, high effort

PR: [gnolang/gno#6061](https://github.com/gnolang/gno/pull/6061) · Review files: [6061-bound-test-memory-by-memory](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6061-bound-test-memory-by-memory)

## TLDR

Several test suites start one heavy worker per CPU core, so a 16-core
workstation runs four times the memory a 4-core CI runner does and can run out.
The change caps those workers at four by default, and lets the integration
suite add more only while the machine reports free memory to spare. A big
machine stays responsive, and CI runs as it did.

## The static cap, `MaxParallel`

After the change, every suite that sizes its workers up front asks
[`testutils.MaxParallel`](https://github.com/gnolang/gno/pull/6061/files#diff-3d9a5edfccddbd6fe7f56ec0928fb1fc6ff370aa16c3ec348c60b7b1eff77dadR41-R46)
how many to run. It answers the smaller of the core count and
[four](https://github.com/gnolang/gno/pull/6061/files#diff-3d9a5edfccddbd6fe7f56ec0928fb1fc6ff370aa16c3ec348c60b7b1eff77dadR19),
the core count of the GitHub runners the suites are tuned on. Setting
`GNO_TEST_MAX_PARALLEL` to a positive integer replaces that answer outright;
[any other value is ignored](https://github.com/gnolang/gno/pull/6061/files#diff-3d9a5edfccddbd6fe7f56ec0928fb1fc6ff370aa16c3ec348c60b7b1eff77dadR23-R29).

Four call sites take it, all after the change:

- `TestFiles` sizes its pool of GnoVM stores with it, where it
  [used `GOMAXPROCS` before](https://github.com/gnolang/gno/blob/8580ae38e/gnovm/pkg/gnolang/files_test.go#L60),
  and its tests that skip the pool
  [draw on the same count](https://github.com/gnolang/gno/pull/6061/files#diff-8c86191f2df542141df7a2007e92276da7d71ee50c26f32bc25b12a46f20e2b2R76).
- `TestStdlibs` had no bound before: every stdlib package ran as a parallel
  subtest with its own fresh store. It now
  [waits for one of `MaxParallel` tokens](https://github.com/gnolang/gno/pull/6061/files#diff-8c86191f2df542141df7a2007e92276da7d71ee50c26f32bc25b12a46f20e2b2R186).
- `examples/Makefile` and `gnovm/Makefile` pass `-p $(GNOTEST_JOBS)` to
  `gno test`, which is
  [`GNO_TEST_MAX_PARALLEL` when it is a positive integer, else 4](https://github.com/gnolang/gno/pull/6061/files#diff-ae8bb09b4bbb919309d4c8bb8ffc46e031bf63b7274502171a183a4dc1138f53R31).
  The shell line mirrors `MaxParallelOverride` in `make`.
- The integration suite's node budget starts from it and never drops below it.

## `gno test -p` stops at the core count

Before, `gno test -p n` ran
[`n` packages at once whatever the core count](https://github.com/gnolang/gno/blob/8580ae38e/gnovm/cmd/gno/test.go#L282-L285).
After, a value above `GOMAXPROCS` is
[lowered to `GOMAXPROCS`](https://github.com/gnolang/gno/pull/6061/files#diff-2e39c5441249018f694183a5710aa7adcad09ec17e870507d20a42184b07e1a7R259-R263).
A worker past the core count cannot run at the same time as the others, yet
still holds a store, so it adds memory and no speed. This is what lets the
Makefiles pass a flat `-p 4` to a 2-core machine without raising its workers.

## Reading free memory, `ReadMemInfo`

The ramp needs one number: how much more memory this process may take.
[`MemInfo`](https://github.com/gnolang/gno/pull/6061/files#diff-3d9a5edfccddbd6fe7f56ec0928fb1fc6ff370aa16c3ec348c60b7b1eff77dadR49-R58)
carries it as `Available`, beside `Total`, both in bytes. Each platform fills it
its own way.

- **Linux** starts from `MemTotal` and `MemAvailable` in `/proc/meminfo`, then
  [narrows both](https://github.com/gnolang/gno/pull/6061/files#diff-9640c076e375af84a8219815eaac2ac6d056ff22b013ff32dcd0832ac4f9dcc8R57-R61)
  by every cgroup v2 limit from the process's own cgroup
  [up to the mount](https://github.com/gnolang/gno/pull/6061/files#diff-9640c076e375af84a8219815eaac2ac6d056ff22b013ff32dcd0832ac4f9dcc8R128-R142).
  `/proc/meminfo` describes the host even inside a container, so without the
  narrowing a container would size itself against memory it cannot have. A
  cgroup's free memory is `memory.max` less `memory.current`,
  [less `inactive_file`](https://github.com/gnolang/gno/pull/6061/files#diff-9640c076e375af84a8219815eaac2ac6d056ff22b013ff32dcd0832ac4f9dcc8R94-R96),
  page cache the kernel drops first. Memory under a cgroup v1 controller
  [returns no reading](https://github.com/gnolang/gno/pull/6061/files#diff-9640c076e375af84a8219815eaac2ac6d056ff22b013ff32dcd0832ac4f9dcc8R53-R56),
  since v1 limits are not read.
- **macOS** reads `hw.memsize` for the total, and counts free pages plus
  [file-backed pages](https://github.com/gnolang/gno/pull/6061/files#diff-383c26ee6d674fa570cbb90f8c794c44e616521295f8283eefa28f9ad23dae5aR23),
  the cache the kernel can drop, as available.
- **Every other platform** returns no reading, which keeps its suites on the
  static cap.

## The node budget, `nodeBudget`

Each integration `.txtar` script runs as a parallel subtest and starts its own
in-memory node. Before the change, the number of nodes alive at once was
whatever `go test -parallel` allowed, `GOMAXPROCS` by default. After, a script
takes a slot from one shared
[`nodeBudget`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R118-R144)
before its first node starts, and waits while none is free.

The budget holds a `limit` between a floor `min` and a ceiling `max`, set by
[`newNodeBudget`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R157-R176):

| After the change | `min` | `max` | starting `limit` | `reserve` |
| --- | --- | --- | --- | --- |
| memory readable, no override | `MaxParallel()` | `GOMAXPROCS` | `MaxParallel()` | a quarter of `Total` |
| `GNO_TEST_MAX_PARALLEL` set, or no reading | `MaxParallel()` | same | same | 0, never moves |

At most once a second,
[`resize`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R211-R262)
takes a reading and moves `limit` by one step:

- **Raise** when every slot is taken and the reading leaves room for one more
  node on top of the reserve:
  [`Available ≥ reserve + nodeMemCost + owed`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R249).
  `nodeMemCost` is
  [640 MiB](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R72),
  the author's measured peak per node rounded up. `owed` is what the running
  nodes have yet to grow into: `running × nodeMemCost` less what the Go
  runtime already holds from the OS. New nodes have not shown in the reading
  yet, and without `owed` the budget would keep admitting against memory they
  are about to take.
- **Cut** when
  [`Available + returning < reserve`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R242-R248),
  never below `min`. `returning` counts one `nodeMemCost` for each cut of the
  last
  [5 seconds](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R87)
  that left a running node unreplaced, the time that node's memory takes to
  reach the reading. Without it the budget would cut once a second down to the
  floor before the first cut showed.
- **Hold** in between. The two thresholds sit a whole node apart, so the limit
  settles instead of flapping.

A worked case, computed from the formulas above: on a 32 GiB, 16-core machine
with no cgroup limit, the reserve is 8 GiB. Four nodes always run. A fifth
needs at least 8.625 GiB available plus `owed`, and the limit drops back
toward four once available memory falls below 8 GiB. The project's own
`TestNodeBudget*` tests build this 8 GiB reserve and pass at the head.

Setting `GNO_TEST_TRACE_BUDGET` prints each change of limit to stderr.

## One slot per script, `acquireSlot`

[`acquireSlot`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R311-R320)
is called by
[`gnoland start`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R515),
before the genesis transactions are built, and is a no-op once the script holds
a slot. The slot goes back when the script ends, not on `gnoland stop`. A
script that stops and starts its node again therefore never waits for a second
slot, which would deadlock once every slot belonged to a script waiting for
one more.

## Before and after

**Before:** the stdlib, examples, file and integration suites each ran one
store or node per core, through `GOMAXPROCS` in `TestFiles`, `gno test` and
`go test -parallel`, and `TestStdlibs` ran every package at once.

**After:** each runs at most `MaxParallel()`, four by default, and the
integration suite alone ramps from four toward the core count through
`nodeBudget` while memory allows. The author's figures for the integration
suite on a 16-core host:
[3.9 GiB peak at two nodes, 12.0 GiB at sixteen, 145 s against 43 s](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R65-R68).

## What a user notices

- On a machine with more than four cores, `make test` in `examples/` and
  `gnovm/` runs four `gno test` workers, whatever the core count, unless
  `GNO_TEST_MAX_PARALLEL` says otherwise.
- `gno test -p 32` on an 8-core machine runs eight packages at once, per
  `GOMAXPROCS` of that process.
- The integration suite's node count now depends on the machine's free memory
  at run time, so two runs on one host can admit different counts.

## What it leaves open

- A cgroup v1 memory limit: nothing reads it, so those hosts stay on the static
  cap.
- A `memory.high` throttle below `memory.max`:
  [not considered](https://github.com/gnolang/gno/pull/6061/files#diff-9640c076e375af84a8219815eaac2ac6d056ff22b013ff32dcd0832ac4f9dcc8R83-R84),
  so the ramp can grow into it.
- The BSDs and Windows: no reading, so the budget never ramps there.
- Only the integration suite ramps; the GnoVM suites keep the fixed count.

## Words used here

| Name | What it is |
| --- | --- |
| `GOMAXPROCS` | Go's count of threads running Go code at once, the core count by default. |
| `MaxParallel()` | `GNO_TEST_MAX_PARALLEL` if a positive integer, else the smaller of `GOMAXPROCS` and 4. |
| `GNO_TEST_MAX_PARALLEL` | Environment variable that pins every count above and stops the ramp. |
| `GNOTEST_JOBS` | Makefile variable passed as `gno test -p`; `GNO_TEST_MAX_PARALLEL` if a positive integer, else 4. |
| `ReadMemInfo` | Returns `Total` and `Available` in bytes, and `false` where nothing can be read. |
| `MemAvailable` | Linux's estimate, in `/proc/meminfo`, of memory a process can take without swapping. |
| `memory.max`, `memory.current` | A cgroup v2 directory's limit, `max` when none, and its current usage. |
| `inactive_file` | The line of `memory.stat` counting page cache the kernel reclaims first. |
| `nodeBudget` | The shared counter deciding how many integration nodes run at once. |
| `limit` | The current allowance of nodes, between `min` and `max`, moved one step per reading. |
| `reserve` | Memory left to the rest of the machine: a quarter of `Total`, 0 when the budget is static. |
| `nodeMemCost` | 640 MiB, the memory one more running node is assumed to add. |
| `owed` | Memory running nodes have yet to take: `running × nodeMemCost` less what the Go runtime holds. |
| `returning` | `nodeMemCost` per cut made in the last 5 s that left a node unreplaced. |
| `acquireSlot` | Takes a script's one slot on `gnoland start`; released when the script ends. |
| `GNO_TEST_TRACE_BUDGET` | Non-empty prints each change of `limit` to stderr. |

<details>
<summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a
generated file. Every excerpt is the after side.

### 1. `hot` · The resize step, [`gno.land/pkg/integration/testscript_gnoland.go`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R211-R262)

Decides, once a second, whether the node allowance goes up, down or stays.
**If wrong:** the suite admits nodes into memory it does not have, or sinks to
the floor on a machine with room.

```go
switch {
case mi.Available+returning < b.reserve && b.limit > b.min:
	b.limit--
	if b.running > b.limit {
		b.cuts = append(b.cuts, now)
	}
case mi.Available >= b.reserve+nodeMemCost+owed && b.limit < b.max && b.running >= b.limit:
	b.limit++
```

### 2. `hot` · The slot per script, [`gno.land/pkg/integration/testscript_gnoland.go`](https://github.com/gnolang/gno/pull/6061/files#diff-d2f03f15d5209aec6341d1c301259f2282e6bfb60e15464d07464b77e0be1237R311-R320)

Takes one budget slot the first time a script starts a node and gives it back
when the script ends. **If wrong:** a restarting script waits on itself, or a
slot leaks and the suite stalls.

```go
slot := ts.Value(envKeyNodeSlot).(*nodeSlot)
if slot.held {
	return
}
nm.budget.acquire()
slot.held = true
ts.Defer(nm.budget.release)
```

### 3. `hot` · The `-p` clamp, [`gnovm/cmd/gno/test.go`](https://github.com/gnolang/gno/pull/6061/files#diff-2e39c5441249018f694183a5710aa7adcad09ec17e870507d20a42184b07e1a7R259-R263)

Bounds `gno test` workers by the core count. **If wrong:** a Makefile's `-p 4`
raises a small machine's worker count.

```go
if cmd.parallel <= 0 || cmd.parallel > runtime.GOMAXPROCS(0) {
	cmd.parallel = runtime.GOMAXPROCS(0)
}
```

### 4. `warm` · The Linux reading, [`tm2/pkg/testutils/parallel_linux.go`](https://github.com/gnolang/gno/pull/6061/files#diff-9640c076e375af84a8219815eaac2ac6d056ff22b013ff32dcd0832ac4f9dcc8R87-R98)

Turns one cgroup directory into a limit and a usage, page cache left out.
**If wrong:** a container sizes itself against the host, or reclaimable cache
reads as used.

```go
if limit, ok = readCgroupUint(filepath.Join(dir, "memory.max")); !ok {
	return 0, 0, false
}
if used, ok = readCgroupUint(filepath.Join(dir, "memory.current")); !ok {
	return 0, 0, false
}
if inactive, ok := readCgroupStat(filepath.Join(dir, "memory.stat"), "inactive_file"); ok {
	used -= min(used, inactive)
}
```

### 5. `warm` · The macOS reading, [`tm2/pkg/testutils/parallel_darwin.go`](https://github.com/gnolang/gno/pull/6061/files#diff-383c26ee6d674fa570cbb90f8c794c44e616521295f8283eefa28f9ad23dae5aR17-R32)

Counts free and file-backed pages as available. **If wrong:** the ramp on a Mac
reads memory as free that the heap already holds.

```go
Available: uint64(pageSize) * (uint64(free) + uint64(fileBacked)),
```

### 6. `warm` · The static cap, [`tm2/pkg/testutils/parallel.go`](https://github.com/gnolang/gno/pull/6061/files#diff-3d9a5edfccddbd6fe7f56ec0928fb1fc6ff370aa16c3ec348c60b7b1eff77dadR41-R46)

The one count every fixed-size suite shares. **If wrong:** CI and workstations
diverge again.

```go
if n, ok := MaxParallelOverride(); ok {
	return n
}
return min(runtime.GOMAXPROCS(0), defaultMaxParallel)
```

### 7. `warm` · The Makefile count, [`examples/Makefile`](https://github.com/gnolang/gno/pull/6061/files#diff-ae8bb09b4bbb919309d4c8bb8ffc46e031bf63b7274502171a183a4dc1138f53R31)

Mirrors the override rule in shell, the same line as
[`gnovm/Makefile`](https://github.com/gnolang/gno/pull/6061/files#diff-9db7ffebcca3a100c6cf7c0b65b1ec86dbdb6fcf80721a326aa5e968b01a70ceR28).
**If wrong:** `make test` and `go test` disagree on one setting.

```make
GNOTEST_JOBS ?= $(or $(shell echo '$(GNO_TEST_MAX_PARALLEL)' | grep -xE '0*[1-9][0-9]*' | sed 's/^0*//'),4)
```

### 8. `cold` · The store pools, [`gnovm/pkg/gnolang/files_test.go`](https://github.com/gnolang/gno/pull/6061/files#diff-8c86191f2df542141df7a2007e92276da7d71ee50c26f32bc25b12a46f20e2b2R186)

Bounds `TestStdlibs`' fresh stores, as `TestFiles` bounds its pool. **If
wrong:** the stdlib run holds one store per package at once.

```go
freshStore := make(chan struct{}, testutils.MaxParallel())
```

</details>
