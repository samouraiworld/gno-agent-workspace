# Candidate 40 — the five length/count caps have no test

What it asserts: `maxIdentLen`, `maxOpLen`, `maxSymbolLen`, `maxEntries` and
`maxPrefixArg` in `examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:192`
can each be raised by orders of magnitude with the package's own suite
staying green, because no test in `grc20routes_test.gno` ever supplies a
value that trips any of the five.

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
git worktree add --detach /tmp/gno6187-40 3aa06ef757b87d731a3e554e96c7540a209fd9a3
cd /tmp/gno6187-40/examples/gno.land/r/nt/grc20routes/v0
sed -i \
  -e 's/maxIdentLen  = 64/maxIdentLen  = 10000/' \
  -e 's/maxOpLen     = 32/maxOpLen     = 10000/' \
  -e 's/maxSymbolLen = 64/maxSymbolLen = 10000/' \
  -e 's/maxEntries   = 16/maxEntries   = 1000/' \
  -e 's/maxPrefixArg = 128/maxPrefixArg = 100000/' \
  grc20routes.gno
cd /tmp/gno6187-40/examples
gno test ./gno.land/r/nt/grc20routes/v0   # gno from the branch's own source
```

Observed (this run, head 3aa06ef75):
```
ok      ./gno.land/r/nt/grc20routes/v0 	3.31s
```

All five constants raised 150x-3000x, suite still green: `TestValidationRejectsUnsafeValues`'s
eleven cases are all character-class or structural, and `TestRegisterRejectsCraftedSymbols`'s
six are the same, so none of the five numeric bounds is on any test's critical path.

Missing test: one case each for `maxEntries+1` entry points, a symbol
`maxSymbolLen+1` bytes long, an op name `maxOpLen+1` bytes long, a prefix
argument `maxPrefixArg+1` bytes long, and a `validateFuncName` identifier
`maxIdentLen+1` bytes long, each asserting the specific panic message at
grc20routes.gno:541/588/605/563/616.
