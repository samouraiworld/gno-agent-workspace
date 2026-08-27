# Differential harness: `UnassignableNames` vs `NameSources[i].Type == NSFuncDecl`

`equivalence-harness.patch` applies to gnolang/gno at add66f24b, the merge base of PR 6089, where both
representations still exist and can be compared on the same run. Three probes:

1. `IsAssignable` computes the old answer and PR 6089's answer and records every call where they differ.
2. `Preprocess` sweeps every block it just built, plus the `ctx` chain, comparing
   `slices.Contains(UnassignableNames, name)` against `NameSources[i].Type == NSFuncDecl` for every name in
   every block. It also flags any block where `len(NameSources) != len(Names)`, which would make PR 6089's
   index read panic.
3. `Define2` records any redefinition overwriting an `NSFuncDecl` entry with another `NSType`, which would
   desynchronise the two representations after the fact.

Run:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6089 -R gnolang/gno
git checkout $(git merge-base origin/master HEAD)
git apply equivalence-harness.patch
go test ./gnovm/pkg/gnolang/ -run TestFiles -timeout 40m -count=1
cat /tmp/zz6089.txt
```

Result over the whole `TestFiles` corpus, 193s:

```
ZZ-REPORT calls=24107 blocks=808021 names=56860035 diffs=0
```

24107 real call sites, 56860035 name comparisons, no disagreement, no length mismatch, no clobbered
`NSFuncDecl` entry.
