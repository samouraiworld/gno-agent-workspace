# Review: [#6061](https://github.com/gnolang/gno/pull/6061)
Event: REQUEST_CHANGES

## Body
- The description says the total cannot quietly double, while [the second semaphore](https://github.com/gnolang/gno/blob/a4d6089/gnovm/pkg/gnolang/files_test.go#L76) bounds it at exactly twice the pool size.

## tm2/pkg/testutils/parallel_linux.go:64 [gh](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L64) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L64)
`memory.max - memory.current` charges the reclaimable page cache as used, so a container reading its own testdata starves itself: the suite takes 262.8s against 185.0s on master. Take `inactive_file` and `slab_reclaimable` off `memory.current` and the container figure matches [`MemAvailable`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L52-L55).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
# run this inside a container with a memory.max set.
gh pr checkout 6061 -R gnolang/gno
GNO_TEST_TRACE_BUDGET=1 go test -run TestTestdata -timeout 45m ./gno.land/pkg/integration/
grep -E '^(file|inactive_file|slab_reclaimable) ' /sys/fs/cgroup/memory.stat
awk '/MemAvailable/' /proc/meminfo
```

The allowance collapses to its floor while the kernel still reports 4 GiB available, and nothing follows t+24.5s. Timestamps added to the trace:

```
t+   2.6s nodeBudget: limit=6 running=5 max=6 avail=4.53GiB reserve=2.50GiB
t+  21.4s nodeBudget: limit=5 running=5 max=6 avail=2.42GiB reserve=2.50GiB
t+  22.4s nodeBudget: limit=4 running=5 max=6 avail=2.26GiB reserve=2.50GiB
t+  23.4s nodeBudget: limit=3 running=4 max=6 avail=2.26GiB reserve=2.50GiB
t+  24.5s nodeBudget: limit=2 running=3 max=6 avail=2.08GiB reserve=2.50GiB
t+ 262.8s peak RSS 4031 MiB, wall 262.8s
```

Sampled every two seconds across a run of the same suite on this host, the cgroup's `file` counter grew from 0.70 GiB to 1.43 GiB from the suite reading its own testdata, the clamped reading fell under the reserve in 33 of 133 samples, and `/proc/meminfo`'s `MemAvailable` fell under it in none.

On this host at one instant: `memory.max` 10.00 GiB, `memory.current` 9.77 GiB, `inactive_file` plus `slab_reclaimable` 3.26 GiB, so `Available` reads 0.23 GiB where a reclaim-aware figure is 3.49 GiB and the reserve is 2.50 GiB.
</details>

## examples/Makefile:28 [gh](https://github.com/gnolang/gno/blob/a4d6089/examples/Makefile#L28) · [↗](../../../../../.worktrees/gno-review-6061/examples/Makefile#L28)
A flat `4` raises the worker count on any host under four cores, since [`gno test -p`](https://github.com/gnolang/gno/blob/a4d6089/gnovm/cmd/gno/test.go#L282-L286) clamps only against the package count: two cores peak at 1097 MiB against 602 MiB on master. Deriving it from [`testutils.MaxParallel()`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L40-L45) holds it to `min(GOMAXPROCS, 4)`.

<details><summary>repro</summary>

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
rm -f /tmp/gno
```

The new default costs 82% more memory than the old one on two cores, and is slower:

```
-p 2: peak 602 MiB, wall 7s
-p 4: peak 1097 MiB, wall 11s
```
</details>

## tm2/pkg/testutils/parallel_linux.go:51 [gh](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L51) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L51)
The allowance freezes at exactly the pressure it exists for: a cgroup at its limit clamps `Available` to zero, which this line reports as a failed reading for [`resize`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L193-L195) to discard. The shrink arm already handles a reading of zero, so `ok` need only mean the figures were read.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6061 -R gnolang/gno
cat > gno.land/pkg/integration/zz_freeze_test.go <<'EOF'
package integration

import (
	"fmt"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/testutils"
)

func TestZeroAvailableFreeze(t *testing.T) {
	walk := func(avail uint64, ok bool) int {
		b := &nodeBudget{min: 2, max: 8, limit: 8, reserve: 8 << 30}
		b.readMem = func() (testutils.MemInfo, bool) {
			return testutils.MemInfo{Total: 32 << 30, Available: avail}, ok
		}
		for range 8 {
			b.tryAcquire()
		}
		for range 25 {
			elapse(b)
			b.tryAcquire()
		}
		return b.limit
	}
	// A cgroup at memory.max yields Available 0, which ReadMemInfo reports as
	// ok=false; one byte of headroom is a reading like any other.
	fmt.Printf("zero-available: limit=%d\none-byte-available: limit=%d\n", walk(0, false), walk(1, true))
}
EOF
go test -run TestZeroAvailableFreeze -v ./gno.land/pkg/integration/
rm gno.land/pkg/integration/zz_freeze_test.go
```

Both budgets are saturated and ticked 25 times with the settle window expired, and the zero reading holds the allowance at 8 where one spare byte walks it to the floor of 2:

```
zero-available: limit=8
one-byte-available: limit=2
```
</details>

## gno.land/pkg/integration/testscript_gnoland.go:162-166 [gh](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L162-L166) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L162-L166)
Missing test: the blocking poll loop in `acquire` and the count admitted under contention, every existing case driving `tryAcquire` directly.

<details><summary>test cases</summary>

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

## gno.land/pkg/integration/testscript_gnoland.go:193-195 [gh](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L193-L195) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L193-L195)
Missing test: a `resize` whose reading fails, [`TestNodeBudgetStaticWhenMemoryUnknown`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/node_budget_test.go#L107-L120) leaving `reserve` zero so `readMem` is never called.

<details><summary>test cases</summary>

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

## tm2/pkg/testutils/parallel_linux.go:89 [gh](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L89) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L89)
Missing test: `cgroupDir` and the `/proc/meminfo` parser hard-code their paths, so deleting [the host-namespace fallback](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L108) leaves the package green. Twelve lines thread a root prefix through `ReadMemInfo`, `cgroupMemory` and `cgroupDir`, taking the file from 82.1% to 96.6%.

<details><summary>test cases</summary>

The seam:

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

The cases, which pass with the seam and fail when the fallback at line 108 is replaced by `return "", false`:

```go
// fakeRoot builds a /proc and /sys/fs/cgroup tree under a temp dir. Keys are
// paths relative to the root, so "proc/meminfo" and "sys/fs/cgroup/memory.max".
func fakeRoot(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	return root
}

func TestCgroupDir(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		files map[string]string
		want  string // relative to the root, "" when ok is false
		ok    bool
	}{
		{
			// Under a cgroup namespace the unified path is already the mount.
			name: "namespaced, path is the mount",
			files: map[string]string{
				"proc/self/cgroup":         "0::/\n",
				"sys/fs/cgroup/memory.max": "max\n",
			},
			want: "sys/fs/cgroup", ok: true,
		},
		{
			name: "nested path exists under the mount",
			files: map[string]string{
				"proc/self/cgroup":                              "0::/user.slice/app.scope\n",
				"sys/fs/cgroup/memory.max":                      "max\n",
				"sys/fs/cgroup/user.slice/app.scope/memory.max": "max\n",
			},
			want: "sys/fs/cgroup/user.slice/app.scope", ok: true,
		},
		{
			// A container sharing the host's cgroup namespace: the unified
			// path names a directory that does not exist under the mount this
			// process sees, and the mount root is the reading that applies.
			name: "host namespace, nested path absent, falls back to the mount",
			files: map[string]string{
				"proc/self/cgroup":         "0::/system.slice/docker-deadbeef.scope\n",
				"sys/fs/cgroup/memory.max": "max\n",
			},
			want: "sys/fs/cgroup", ok: true,
		},
		{
			name: "no memory controller anywhere",
			files: map[string]string{
				"proc/self/cgroup":           "0::/system.slice/docker-deadbeef.scope\n",
				"sys/fs/cgroup/cgroup.procs": "\n",
			},
			ok: false,
		},
		{
			// v1 lines name a controller; there is no unified entry to use.
			name: "cgroup v1 only",
			files: map[string]string{
				"proc/self/cgroup":         "12:memory:/user.slice\n11:cpu,cpuacct:/user.slice\n",
				"sys/fs/cgroup/memory.max": "max\n",
			},
			ok: false,
		},
		{
			name:  "proc file absent",
			files: map[string]string{"sys/fs/cgroup/memory.max": "max\n"},
			ok:    false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := fakeRoot(t, tt.files)
			dir, ok := cgroupDir(root)
			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, filepath.Join(root, tt.want), dir)
			}
		})
	}
}

const sampleMeminfo = `MemTotal:       32950272 kB
MemFree:         1048576 kB
MemAvailable:   20971520 kB
Buffers:          123456 kB
`

func TestReadMemInfoMeminfo(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name              string
		meminfo           string
		wantTotal, wantAv uint64
		ok                bool
	}{
		{
			name: "kB fields become bytes", meminfo: sampleMeminfo,
			wantTotal: 32950272 << 10, wantAv: 20971520 << 10, ok: true,
		},
		{
			// Kernels before 3.14 omit MemAvailable. Zero must read as no
			// figure, not as no memory free.
			name:    "no MemAvailable line",
			meminfo: "MemTotal:       32950272 kB\nMemFree:         1048576 kB\n",
			ok:      false,
		},
		{
			name:    "no MemTotal line",
			meminfo: "MemAvailable:   20971520 kB\n",
			ok:      false,
		},
		{
			name:      "malformed values are skipped, not fatal",
			meminfo:   "MemTotal:\nMemTotal:       32950272 kB\nMemAvailable: nonsense kB\nMemAvailable: 4 kB\n",
			wantTotal: 32950272 << 10, wantAv: 4 << 10, ok: true,
		},
		{name: "empty file", meminfo: "", ok: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := fakeRoot(t, map[string]string{"proc/meminfo": tt.meminfo})
			mi, ok := readMemInfo(root)
			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.wantTotal, mi.Total)
				assert.Equal(t, tt.wantAv, mi.Available)
			}
		})
	}
}

func TestReadMemInfoClampsToCgroup(t *testing.T) {
	t.Parallel()

	root := fakeRoot(t, map[string]string{
		"proc/meminfo":                 sampleMeminfo,
		"proc/self/cgroup":             "0::/\n",
		"sys/fs/cgroup/memory.max":     "4294967296\n", // 4 GiB
		"sys/fs/cgroup/memory.current": "1073741824\n", // 1 GiB
	})
	mi, ok := readMemInfo(root)
	require.True(t, ok)
	assert.Equal(t, uint64(4<<30), mi.Total, "host total must narrow to the cgroup limit")
	assert.Equal(t, uint64(3<<30), mi.Available)
}
```
</details>

## gno.land/pkg/integration/testscript_gnoland.go:74-76 [gh](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L74-L76) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L74-L76)
Nit: `nodeMemCost` at 640 MiB under-charges the two-to-four band a shrunken allowance sits in, where a node costs 973 MiB rather than the ~580 MiB averaged over the whole range.

<details><summary>the table's own marginal costs</summary>

```
2->4: 973 MiB/node
4->6: 717 MiB/node
6->8: 512 MiB/node
8->12: 307 MiB/node
12->16: 666 MiB/node
2->16: 592 MiB/node
```
</details>

## gno.land/pkg/integration/testscript_gnoland.go:199 [gh](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L199) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L199)
Nit: a failed reading skips this stamp, so every blocked script retakes the reading five times a second, a `vm_stat` subprocess each on darwin. Stamping before the read covers the failed reading too.

## gno.land/pkg/integration/testscript_gnoland.go:220 [gh](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L220) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L220)
Nit: `release` decrements without a floor, so a second release for one slot drives `running` negative and admits a node past the allowance for the rest of the run, [`slot.held`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L266-L268) the only guard against that today.

## gno.land/pkg/integration/testscript_gnoland.go:259-263 [gh](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L259-L263) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L259-L263)
Nit: the comment blames a deadlock that cannot happen, [`gnoland start`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L447-L451) rejecting a second node for the same script. Only the reason is wrong: a script stopping and restarting would otherwise queue behind fresh scripts.

## gno.land/pkg/integration/testscript_gnoland.go:271 [gh](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L271) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L271)
Nit: [testscript runs deferred functions in reverse](https://github.com/rogpeppe/go-internal/blob/v1.15.0/testscript/testscript.go#L912-L918) and the node teardown registers before this line, so the token goes back with the node still running. The release belongs inside the deferred function that stops the node.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6061 -R gnolang/gno
cat > gno.land/pkg/integration/zz_order_test.go <<'EOF'
package integration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeSlotReleaseOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string
	note := func(s string) func() {
		return func() {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, s)
		}
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "order.txtar"),
		[]byte("acquire\n-- placeholder --\n"), 0o600))

	// testscript runs each script as a parallel subtest, so the order is only
	// complete once they have all finished.
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"node stop", "budget release"}, order,
			"the token must be given back after the node it accounts for is stopped")
	})

	testscript.Run(t, testscript.Params{
		Dir: dir,
		Setup: func(env *testscript.Env) error {
			// Same position as SetupGnolandTestscript's node-stop defer.
			env.Defer(note("node stop"))
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			// Same position as acquireSlot's ts.Defer(nm.budget.release).
			"acquire": func(ts *testscript.TestScript, neg bool, args []string) {
				ts.Defer(note("budget release"))
			},
		},
	})
}
EOF
go test -run TestNodeSlotReleaseOrder ./gno.land/pkg/integration/
rm gno.land/pkg/integration/zz_order_test.go
```

The two functions are registered at the two positions the code uses, and the token comes back first:

```
expected: []string{"node stop", "budget release"}
actual  : []string{"budget release", "node stop"}
```
</details>

## gnovm/pkg/gnolang/files_test.go:61 [gh](https://github.com/gnolang/gno/blob/a4d6089/gnovm/pkg/gnolang/files_test.go#L61) · [↗](../../../../../.worktrees/gno-review-6061/gnovm/pkg/gnolang/files_test.go#L61)
Nit: the pool is sized by [`min(GOMAXPROCS, 4)`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L40-L45), and the memory-driven sizing this describes lives only in [`nodeBudget`](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L114).

## tm2/pkg/testutils/parallel.go:14-15 [gh](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L14-L15) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel.go#L14-L15)
Nit: the comment calls `defaultMaxParallel` the count used when memory cannot be read, while [`MaxParallel`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L40-L45) returns `min(GOMAXPROCS, 4)` and reads no memory.

## gno.land/pkg/integration/testscript_gnoland.go:72 [gh](https://github.com/gnolang/gno/blob/a4d6089/gno.land/pkg/integration/testscript_gnoland.go#L72) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L72)
Suggestion: a transient dip pins the suite at two nodes for the rest of the run, since climbing back out needs a reading 0.625 GiB above the reserve. A dip costs seconds instead once the shrink stops at [`MaxParallel()`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L40-L45) and goes below it only after several consecutive readings under the reserve.

## go.mod:29 [gh](https://github.com/gnolang/gno/blob/a4d6089/go.mod#L29) · [↗](../../../../../.worktrees/gno-review-6061/go.mod#L29)
Suggestion: [`github.com/pbnjay/memory`](https://github.com/pbnjay/memory) last changed in July 2021, forks a `vm_stat` subprocess per reading on darwin, and carries [an open issue](https://github.com/pbnjay/memory/issues/12) for the container limits this branch writes by hand. `golang.org/x/sys` is already in the graph and covers the darwin and windows readings in two calls.

## tm2/pkg/testutils/parallel.go:47-48 [gh](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel.go#L47-L48) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel.go#L47-L48)
Suggestion: this type describes the machine rather than any test, so every module reaching the worker-count helpers beside it takes a memory library into its graph, `contribs/gnodev` and `contribs/gnobro` included. Its own package takes the type and [the readings that fill it](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L16-L52), leaving the worker-count helpers in `testutils`.

## tm2/pkg/testutils/parallel_linux.go:79 [gh](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L79) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L79)
Suggestion: the suite sizes itself against the whole host when an ancestor cgroup carries the `memory.max` and the leaf this reads says `max`. Take the smallest limit found from the leaf up to the mount root, which covers a systemd slice and a Kubernetes pod alike.

## tm2/pkg/testutils/parallel_linux.go:97 [gh](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L97) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L97)
Suggestion: a cgroup v1 container has no unified `0::` line, so [`ReadMemInfo`](https://github.com/gnolang/gno/blob/a4d6089/tm2/pkg/testutils/parallel_linux.go#L48-L51) returns the host figures with `ok` true and the budget ramps against memory it does not have. Reading `memory.limit_in_bytes` under v1 keeps the budget static as the description promises, and so does reporting no reading at all.
