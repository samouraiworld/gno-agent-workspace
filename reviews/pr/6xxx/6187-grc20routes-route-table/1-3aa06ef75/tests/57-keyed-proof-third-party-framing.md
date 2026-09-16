# Candidate #57 — the doc comment's "nothing to race" claim ignores the token realm's own ledger power

Anchor: `examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:344-346`

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
sed -n '344,346p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
sed -n '35,43p' examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno
sed -n '275,292p' examples/gno.land/p/nt/grc20/v0/token.gno
```

Confirmed by reading, no run needed — three call sites establish the mechanism:

```
grc20routes.gno:344-346: "One transaction is tidy but not required: the open proof is held against
                          the prover, and only the prover can change the prover's own allowances, so
                          there is nothing for anyone else to race."
grc20routes_test.gno:39-42: func approveAs(symbol string, owner address, amount int64) {
                                if err := ledgers[symbol].Approve(owner, ProbeAddress(), amount); ...
token.gno:275-276: // Approve sets the allowance of the specified owner and spender.
                    func (led *PrivateLedger) Approve(owner, spender address, amount int64) error {
```

`PrivateLedger.Approve` takes an arbitrary `owner` parameter — it never checks that `owner` is the
calling realm or the transaction signer. The package's own test helper, `approveAs`, exploits
precisely this to fake a keyed realm's per-symbol `Approve(cur, symbol, spender, amount)` from
outside: it drives `ledgers[symbol].Approve(owner, ...)` directly, choosing `owner` freely. The
same ledger is reachable from any function the token's own realm exposes, so a realm holding two
registered tokens can set *any* address's probe allowance to *any* value — including an address
that is not the caller of that realm at all — without that address's consent or participation.

The doc comment's safety argument ("only the prover can change the prover's own allowances") is
therefore false as a description of the code: the token realm, not just the prover, can change the
prover's allowances, and it can do so for a prover it never interacted with. The consequence the
finder names — an honest `BeginKeyedProof`/`FinishKeyedProof` caller whose probe nonces are parked
by the realm itself, so `FinishKeyedProof` succeeds and `provenEvent` names them as the prover of a
realm they never called — follows directly from `Approve`'s open `owner` parameter; it needs no
mutation to see, only reading `PrivateLedger.Approve`'s signature against what the comment claims
happens between `Begin` and `Finish`.

Not verified here (would need a run per the finder's own check, out of this batch's read-only
scope): whether a deployed victim realm's own code path would ever call `Approve` with a
caller-chosen `owner` in a way reachable from `zattack`-style probing, versus needing a realm
written to specifically stage this. The mechanism (`Approve`'s arbitrary `owner`) is proven by the
signature and by the test helper's own use of it; the end-to-end framing scenario is PLAUSIBLE
pending that run.
