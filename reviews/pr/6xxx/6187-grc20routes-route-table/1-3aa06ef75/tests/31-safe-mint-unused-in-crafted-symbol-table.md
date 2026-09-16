# Candidate #31 — the `SAFE` mint in `TestRegisterRejectsCraftedSymbols` is dead setup

Anchor: `examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:155`

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
grep -n SAFE examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno
sed -n '295,306p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

Confirmed by reading, no run needed:

```
grc20routes_test.gno:155:  mint(0, cur, "SAFE")     # the only occurrence of "SAFE" in the file
```

```
grc20routes.gno:300:  validateSymbol(symbol)                                    // aborts first
grc20routes.gno:305:  if grc20reg.Get(key) == nil { panic(...) }                // never reached by any table case
```

Every case in the table (`a.b`, `a/b`, `a"b`, `a:b`, `a#b`, `""`) fails `validateSymbol` at line
300 and panics before `Register` ever calls `grc20reg.Get` at line 305. The `SAFE` token minted at
line 155 is never named by, looked up by, or otherwise connected to any of the six table cases, so
its presence or absence cannot change the test's outcome — it documents nothing the table's own
symbols don't already establish, and reads as if the registry lookup were part of what the table
proves when in fact `validateSymbol` alone decides every case.

Deleting line 155 needs no mutation to confirm: the call graph above already shows `Register`
returns via the `validateSymbol` panic in all six cases, so `grc20reg.Get` (and hence any minted
token) is provably unreachable code for this test. Running
`gno test -run TestRegisterRejectsCraftedSymbols` in a copy with the line removed, per the finder's
named check, would only reconfirm what the read already settles.
