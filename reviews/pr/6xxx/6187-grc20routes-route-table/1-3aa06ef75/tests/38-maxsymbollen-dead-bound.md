# Candidate #38 — maxSymbolLen (64) is dead relative to grc20's own cap (11)

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
grep -n "MaxSymbolLen" examples/gno.land/p/nt/grc20/v0/types.gno
sed -n '190,196p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
sed -n '296,309p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

Output:

```
142:	MaxSymbolLen = 11
```
```go
	maxIdentLen  = 64
	maxOpLen     = 32
	maxSymbolLen = 64
	maxEntries   = 16
	maxPrefixLen = 4
	maxPrefixArg = 128
```
```go
func Register(cur realm, symbol string, r Route) string {
	rlmPath := cur.Previous().PkgPath()
	if rlmPath == "" {
		panic("grc20routes: caller is not a realm")
	}
	validateSymbol(symbol)                 // line 300: local length/charset check first
	key := fqname.Construct(rlmPath, symbol)

	// Bind this table to grc20reg's: no routes for tokens that do not exist,
	// so a key here always resolves to a token there.
	if grc20reg.Get(key) == nil {          // line 305: the registry lookup runs second
		panic("grc20routes: not registered in grc20reg: " + key)
	}
```

CONFIRMED by reading: `grc20/v0/types.gno:142` fixes `MaxSymbolLen = 11` for
every GRC20 token, while `grc20routes.gno:194` sets its own, unrelated
`maxSymbolLen = 64`. `validateSymbol` (called at line 300) runs before the
`grc20reg.Get` registry lookup (line 305), so for any `symbol` between 12 and
64 characters the local bound never fires — `grc20reg.Get` is what rejects
those, since no such key can exist — and for any `symbol` over 64 characters
the caller is told `"grc20routes: symbol too long"`, a diagnosis that is
accurate about the input but describes a state (`grc20reg` holding a
65+-character symbol) that `grc20/v0`'s own 11-character cap makes impossible
system-wide. The two constants are never reconciled or derived from one
another; `maxSymbolLen` at 64 is a bound copied from the identical-looking
`maxIdentLen`/`maxOpLen` constants above it (64 and 32, both used for realm
paths and op names, which have no GRC20-imposed cap) rather than from GRC20's
own type, and the ordering (validate-then-lookup) means the local bound can
only ever preempt the registry error, never add a check the registry lookup
doesn't already make.

Band: Nit — the code fails safe either way (both paths panic and reject
registration), and reordering the checks or tightening the constant changes
only which error message a misbehaving caller sees, not whether registration
succeeds. Fix: derive `maxSymbolLen` from `grc20.MaxSymbolLen` (11) or drop the
local length check and let `grc20reg.Get` be the sole authority, so the error
message matches the actual constraint.
</content>
