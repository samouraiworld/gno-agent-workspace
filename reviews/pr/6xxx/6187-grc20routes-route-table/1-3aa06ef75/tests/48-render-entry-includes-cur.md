# Candidate #48 — renderEntry prints `cur` as a call argument

What it asserts: `renderEntry` (grc20routes.gno:734-740) renders the entry
point of a proven/declared route as `Op: Func(cur, prefix..., tail...)`,
literally including the identifier `cur` as the first printed argument —
contradicting both the page's own header two blocks above (line 660-662,
"no leading arguments" for the unlisted/canonical case) and the only working
`gnokey maketx call` invocation the PR ships, in the integration txtar.

Repro (read-only, from a plain clone):

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
git clone https://github.com/gnolang/gno && cd gno
git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3

# 1. the rendered call shape, as literal source:
sed -n '734,740p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

Output (trimmed to the signal):

```
func renderEntry(e Entry, prefix []string) string {
	args := "cur"
	for _, p := range prefix {
		args += ", " + md.EscapeText(p)
	}
	return ufmt.Sprintf("- %s: `%s(%s%s)`\n", e.Op, e.Func, args, opTail(e.Op))
}
```

For a keyed token (`Prefix = ["FOO"]`) this renders:
`Approve: `Approve(cur, FOO, spender, amount)``

```bash
# 2. the only working transaction the PR itself demonstrates, same commit:
grep -n 'maketx call.*grc20factory -func Approve' gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar
```

Output:

```
gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20factory -func Approve -args FOO -args g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep -args 111 ...
```

The working call is `Approve(FOO, spender, amount)` — three `-args`, no `cur`.
`cur realm` is a crossing parameter the VM injects; `gnokey maketx call` cannot
supply it (stated by the package's own doc comment, grc20reg.gno:108-110, and
visible in the txtar: only 3 `-args` reach a 4-parameter method). A user who
copies `renderEntry`'s printed call verbatim into `-args` submits one argument
too many, mapped to the wrong parameters, and the transaction is rejected or
(worse, for a 3-arg Op like Approve where a stray leading `-args cur` would
shift every position by one) silently miscalls with `cur` bound to `spender`.

Band: Nit — the page's own header and the shipped recipe already tell a
careful reader the right shape; only the auto-rendered per-token page disagrees
with them.

Fix: `renderEntry` should build `args` from `prefix` alone, omitting the
literal `"cur"` token, e.g. `args := strings.Join(prefix, ", ")` (or the
existing loop without the `"cur"` seed), matching the header's own wording and
the txtar's call shape.
