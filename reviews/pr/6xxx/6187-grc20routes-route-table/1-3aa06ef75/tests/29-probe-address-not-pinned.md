# Candidate #29 — `approveAs`/`allowanceOf` share `probe`, never pinning it to the real deployed address

Anchor: `examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:39`

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
sed -n '39,43p;280p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno
```

Confirmed by reading, no run needed — the two functions dereference the same package-level identifier:

```
grc20routes_test.gno:40:  if err := ledgers[symbol].Approve(owner, ProbeAddress(), amount); err != nil {
grc20routes.gno:280:      func ProbeAddress() address { return probe }
grc20routes.gno:426-427:  func allowanceOf(tokenKey string, owner address) int64 {
                              return grc20reg.MustGet(tokenKey).Allowance(owner, probe)
                          }
```

`approveAs` parks the allowance at `ProbeAddress()`, and `allowanceOf` (which the keyed-proof
tests call transitively through `Begin`/`FinishKeyedProof`) reads it back at the package var
`probe` — the same value `ProbeAddress()` returns. Every unit-suite assertion about "the probe
address" is therefore true for whatever `probe` happens to be initialized to; the unit suite
cannot distinguish `probe == this realm's own address` (what the body's safety claim rests on)
from any other value. Only the txtar, which hardcodes the literal address
`g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep`, pins that.

Confirmed behaviorally by construction: swapping `probe`'s init value would leave the package
suite green (both sides move together) while reddening the txtar — this needs a mutated copy of
`examples/`, out of this batch's read-only scope, and is exactly the check the finder named.
