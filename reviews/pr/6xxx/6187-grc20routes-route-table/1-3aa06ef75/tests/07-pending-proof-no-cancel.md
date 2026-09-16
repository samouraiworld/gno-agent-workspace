# Candidate 7 — abandoned BeginKeyedProof entries are never removed

Confirmed by reading, not by running: a Nit anchored on the package's own
doc comment (`grc20routes.gno:160`, "in-flight proofs"), so no heredoc
reproducer per Calibration.

```
# from a local clone of gnolang/gno, examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:
grep -n 'pending\.' grc20routes.gno
```

Confirmed behaviorally:

```
377:	pending.Set(pendingKey(prover, pathA), &proof{
396:	v := pending.Get(pk)
415:	pending.Remove(pk)
```

`pending.Remove` fires only at line 415, reached only after
`FinishKeyedProof` passes both nonce checks (lines 408-412). No cancel entry
point and no expiry path exist anywhere in the package — a `BeginKeyedProof`
that is never finished leaves its node under `"<prover>/<path>"` permanently.
