# Allocation accounting on a fallthrough chain

`fallthrough-alloc_test.go` beside this file, dropped into `gnovm/pkg/gnolang/`
and run at each commit. Toolchain go1.25.9. `zzChain(n, k)` builds a switch of
`n` clauses chained by `fallthrough`, each declaring `k` locals; every clause
declares the same number so the merge base does not hit its block-shrinkage
panic.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6058 -R gnolang/gno
cp <this-directory>/fallthrough-alloc_test.go gnovm/pkg/gnolang/zz_ftgas_test.go
go test ./gnovm/pkg/gnolang/ -run 'TestZZFallthroughAlloc' -count=1 -v
rm gnovm/pkg/gnolang/zz_ftgas_test.go
```

| Program | `391840aa7` | `0cf310707` master | `754780601` merge base |
|---|---|---|---|
| 1 clause, 4 locals | 2760 | 2760 | 2760 |
| 2 clauses, 4 locals | 2920 | 2920 | 2760 |
| 10 clauses, 4 locals | 4200 | 4200 | 2760 |
| 50 clauses, 4 locals | 10600 | 10600 | 2760 |

Bytes accounted by the allocator across `RunMain`. Gas over the same runs, in
the same order: 153270, 153270, 150379 for the 50-clause case, and the VM cycle
count is identical on all three, so the whole delta is allocation accounting.

`b.Values = b.Values[:ss.GetNumNames()]` lowers `len(b.Values)`, so
`ExpandWith` computes `newNames` against the switch's own name count rather
than the previous clause's and calls `alloc.AllocateBlockItems` for the target
clause's whole set. `growBlockValues` re-slices inside the existing capacity on
that path, so nothing is allocated for the charge. `Allocator.Allocate` both
charges gas and counts toward `maxBytes`, which drives the GC callback.

The head and current master columns are identical because the line arrives
through [#6056](https://github.com/gnolang/gno/pull/6056), merged as
`0cf310707`, whose hunks are byte-identical to this branch's first commit:

```bash
diff <(git show 0cf310707 -- gnovm/pkg/gnolang/ | grep -E '^[+-]' | grep -v '^[+-][+-]') \
     <(git show f854aef45 -- gnovm/pkg/gnolang/ | grep -E '^[+-]' | grep -v '^[+-][+-]')
```

So the branch carries the change rather than causing it.
