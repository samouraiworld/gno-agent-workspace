# Node budget trace, this container, a4d6089

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6061 -R gnolang/gno
GNO_TEST_TRACE_BUDGET=1 go test ./gno.land/pkg/integration/ \
  -run 'TestTestdata/(addpkg|adduser|balances|gnokey_query|restart|simple_call|gnoland_start|logged_in|maketx)' -v
```

```
nodeBudget: limit=3 running=0 max=6 avail=1.87GiB reserve=2.50GiB
nodeBudget: limit=2 running=3 max=6 avail=1.79GiB reserve=2.50GiB
nodeBudget: limit=3 running=2 max=6 avail=3.15GiB reserve=2.50GiB
nodeBudget: limit=2 running=3 max=6 avail=2.49GiB reserve=2.50GiB
ok  	github.com/gnolang/gno/gno.land/pkg/integration	89.416s
```

Host `MemAvailable` during the run: 5.1 GiB. The cgroup clamp reports 1.87 GiB because
`memory.current` charges the page cache, so the allowance falls to the floor before the
first node starts (`limit=3 running=0`).

## Full TestTestdata, same container, a4d6089

```bash
# from a local clone of gnolang/gno:
# run this inside a container with a memory.max set.
gh pr checkout 6061 -R gnolang/gno
GNO_TEST_TRACE_BUDGET=1 go test -run TestTestdata -timeout 45m ./gno.land/pkg/integration/
```

All 188 scripts, timestamps added to the trace:

```
t+   2.6s nodeBudget: limit=6 running=5 max=6 avail=4.53GiB reserve=2.50GiB
t+  21.4s nodeBudget: limit=5 running=5 max=6 avail=2.42GiB reserve=2.50GiB
t+  22.4s nodeBudget: limit=4 running=5 max=6 avail=2.26GiB reserve=2.50GiB
t+  23.4s nodeBudget: limit=3 running=4 max=6 avail=2.26GiB reserve=2.50GiB
t+  24.5s nodeBudget: limit=2 running=3 max=6 avail=2.08GiB reserve=2.50GiB
t+ 262.8s peak RSS 4031 MiB, wall 262.8s
```

Nothing follows t+24.5s: the allowance reached its floor in 22 seconds and stayed
there for the remaining 91% of the run. Master `7547806` finishes the same 188
scripts in 185.0s at 5011 MiB on this container.

Host `MemAvailable` stayed above 4 GiB throughout, and at one instant during the
run: `memory.max` 10.00 GiB, `memory.current` 9.77 GiB, `inactive_file` plus
`slab_reclaimable` 3.26 GiB, so `clampToCgroup` reports `Available` 0.23 GiB
where a reclaim-aware figure is 3.49 GiB.
