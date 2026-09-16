# Candidate 11 — renderToken's empty-Funcs branch: REFUTED as dead code

Confirmed unreachable by reading, not by running: a Nit anchored on
`grc20routes.gno:711`, so no heredoc reproducer per Calibration (the finder's
own suggested check, a sentinel panic, is a mutation and out of scope for a
Nit settled by reading).

```
# from a local clone of gnolang/gno, examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:
sed -n '246,260p;295,312p;436,447p;536,539p' grc20routes.gno
```

Proving lines. `Get` (436-447) has exactly three return paths:

```
439:	if v := declared.Get(tokenKey); v != nil {
440:		return *(v.(*Route)), SourceDeclared
442:	if provenKeyed.Has(pkgPath) {
444:		return Keyed(symbol), SourceProven
446:	return Canonical(), SourceConvention
```

`Canonical()` (246-253) and `Keyed()` (255-260) each hardcode exactly three
`Entry` values in `Funcs` — never zero. The only other source, `declared`,
is populated solely by `Register` (295-311), which calls `validate(r)`
(line 309) before `declared.Set`, and `validate` panics on
`len(r.Funcs) == 0` (536-539). So no route stored anywhere in this package
can ever carry an empty `Funcs`, and `renderToken`'s `len(r.Funcs) == 0`
branch at line 711 is unreachable — REFUTED as a live bug; it is dead
protective code, not a false claim about a real one.
