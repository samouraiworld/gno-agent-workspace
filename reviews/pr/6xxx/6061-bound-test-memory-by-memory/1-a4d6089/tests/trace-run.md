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
