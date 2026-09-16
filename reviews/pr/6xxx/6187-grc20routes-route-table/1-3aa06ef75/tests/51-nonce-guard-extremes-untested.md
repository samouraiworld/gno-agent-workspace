# Candidate #51 — nonce sign guard exercised only at zero

What it asserts: `BeginKeyedProof`'s nonce validation (`nonceA <= 0 ||
nonceB <= 0`, grc20routes.gno:367) already rejects negative and
`math.MaxInt64`-range nonces in the shipped code. The package's own test suite,
however, exercises that guard at `0` only — never at a negative value, never at
`math.MaxInt64` — so nothing in the suite would catch a future edit that
narrowed the guard to `== 0` (admitting negatives) or that overflowed on the
extreme.

Repro (read-only, from a plain clone):

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
git clone https://github.com/gnolang/gno && cd gno
git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3

sed -n '367,369p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
grep -n 'nonceA\|nonceB\|MaxInt64\|BeginKeyedProof(' \
  examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno
```

Output (trimmed to the signal):

```
	if nonceA <= 0 || nonceB <= 0 {
		panic("grc20routes: nonces must be positive")
	}
```

```
grc20routes_test.gno:245:		func() { BeginKeyedProof(cross(cur), keyA, keyA, 1, 2) })
grc20routes_test.gno:247:		func() { BeginKeyedProof(cross(cur), keyA, keyB, 7, 7) })
grc20routes_test.gno:249:		func() { BeginKeyedProof(cross(cur), keyA, keyB, 0, 2) })
grc20routes_test.gno:261:		BeginKeyedProof(cross(cur), keyA, keyB, 11, 22)
```

Every nonce literal the suite ever passes is one of `0, 1, 2, 7, 11, 22` — no
call anywhere in `grc20routes_test.gno` passes a negative value or
`math.MaxInt64`. The "nonces must be positive" assertion (line 249) covers the
boundary `0` and nothing beyond it in either direction.

Band: Nit — the production guard is correct today (`<= 0` rejects negative and
zero alike), so this is a test-coverage gap, not a live defect: it earns a Nit
because the guard is one keystroke (`<=` to `==`) from silently regressing with
the suite still green.

Fix: add two cases to `TestKeyedProofRejectsMismatchedInputs` —
`BeginKeyedProof(cross(cur), keyA, keyB, -1, 2)` and
`BeginKeyedProof(cross(cur), keyA, keyB, math.MaxInt64, 2)` (the second also
exercises `allowanceOf`'s int64 comparison at the top of its range) — both
asserting the same `"grc20routes: nonces must be positive"` panic (or, for
`MaxInt64`, whatever the correct behavior is once traced through to the
allowance comparison at lines 408-413).

Check still open, beyond this batch: whether `nonceA == math.MaxInt64` can
ever equal a real GRC20 allowance set through `grc20reg`'s `Approve`, i.e.
whether `allowanceOf` and the underlying token's allowance type can carry that
value at all — unresolved by this read; would need tracing grc20reg's
`Allowance`/`Approve` amount type and a probe call, not a further read of this
file.
