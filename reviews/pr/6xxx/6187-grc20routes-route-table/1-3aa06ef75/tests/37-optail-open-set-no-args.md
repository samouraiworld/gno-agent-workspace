# Candidate #37 — opTail collapses every non-canonical op to ", …"

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
sed -n '722,739p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

Output (the finding is visible without running anything):

```go
// opTail is the argument list an operation takes after the prefix. Unknown
// operations render without one rather than guessing.
func opTail(op string) string {
	switch op {
	case OpApprove:
		return ", spender, amount"
	case OpTransfer:
		return ", to, amount"
	case OpTransferFrom:
		return ", from, to, amount"
	}
	return ", …"
}

func renderEntry(e Entry, prefix []string) string {
	args := "cur"
	for _, p := range prefix {
		args += ", " + md.EscapeText(p)
	}
	return ufmt.Sprintf("- %s: `%s(%s%s)`\n", e.Op, e.Func, args, opTail(e.Op))
}
```

`renderEntry` calls `opTail(e.Op)` unconditionally, and `opTail`'s switch only
covers the three canonical ops (`OpApprove`, `OpTransfer`, `OpTransferFrom`,
lines 176-178). For the wugnot example the package doc comment itself gives —
`Canonical().With("deposit", "Deposit").With("withdraw", "Withdraw")` (line
714-716) — the `deposit` entry falls through to `, …` and renders as:

```
- deposit: `Deposit(cur, …)`
```

That is the exact scenario `Route.Funcs`'s doc comment (695-707) argues the
open set exists for: "wugnot has Deposit and Withdraw, which a wallet needs to
find the same way it finds Approve." But the rendered line for `Deposit`
carries no argument names at all — the reader already knows the function name
`Deposit` from the JSON `Func` field; `opTail` was supposed to be the thing
that adds value (the callable signature) and for every operation outside the
canonical three it adds nothing.

CONFIRMED by reading 722-739: the switch is exhaustive over the three
constants declared at lines 176-178, and every other `Op` string — including
`"deposit"`/`"withdraw"`, the PR body's own extension example — takes the
`, …` default. No run needed; the mapping is total and syntactic.

Band: Nit — `opTail` still renders something, it degrades gracefully, and the
raw `Func` name plus off-chain documentation remains available to the
integrator. What is lost is exactly what the doc comment (711-716) claims the
open set is for: a route that renders a copyable call for a declared
non-canonical operation renders one with the argument list dropped.
</content>
