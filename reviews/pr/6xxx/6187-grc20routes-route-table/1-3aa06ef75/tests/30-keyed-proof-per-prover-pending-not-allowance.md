# Candidate #30 — `TestKeyedProofIsPerProver` proves the `pending` map is per-prover, never that allowances are

Anchor: `examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:254`

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
sed -n '254,268p' examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno
sed -n '355,410p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

Confirmed by reading, no run needed. The test never calls `approveAs`/`Approve` for either key;
bob's only action is `FinishKeyedProof(cross(cur), keyA, keyB)` right after alice's `Begin`, and it
aborts with `"grc20routes: no proof in progress for "+selfPath`. Tracing `FinishKeyedProof`:

```
pk := pendingKey(prover, pathA)   // prover = bob here
v := pending.Get(pk)
if v == nil {
    panic("grc20routes: no proof in progress for " + pathA)   // <- this is what fires
}
```

and `pendingKey(prover, pkgPath) = prover.String() + "/" + pkgPath` — the `pending` map is already
keyed per-prover, so bob's lookup misses regardless of what alice parked. The abort the test
observes fires at this `pending.Get` miss, before either `allowanceOf` call the load-bearing check
(`grc20routes.gno` around line 403-409) makes. The comment's "Alice's allowances are not his to
borrow" is a second, distinct property — whether bob could piggyback if he opened his own proof
with alice's already-parked nonces — that this test's sequence cannot reach, since it never gives
bob a pending entry to begin with.

Confirmed behaviorally by construction: extending the test with `alice` parking both nonces via
`approveAs`, then having `bob` call `BeginKeyedProof` with the *same* keys and nonces, would
exercise the actual borrow question (`Begin`'s pre-existing check at grc20routes.gno:373 compares
`allowanceOf(...) == nonceA`, which is alice's parked value, against nonces bob himself supplies)
and currently runs in no test. This is the check the finder named; adding it is beyond this
read-only batch.
