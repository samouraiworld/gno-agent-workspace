# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
# git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
# sed -n '650,656p;505,520p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
# grep -n 'recover\|strings.Cut(path\|strings.Split(path' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno

Candidate #15 (grc20routes.gno:653): confirmed behaviorally by read, no mutation needed.

Render's only branch is empty path vs non-empty (line 653: `if path != "" { return renderToken(path) }`),
and renderToken/mustToken (line 514-520) panics unconditionally when
`grc20reg.Get(tokenKey) == nil`, with no recover anywhere in the file (grep for
`recover|strings.Cut(path|strings.Split(path` over the whole file returns nothing)
and no path normalisation before the lookup. Any non-empty path that is not an
exact registered token key — a typo, a stale token, a query-string suffix —
reaches Render → renderToken → mustToken and aborts with
"grc20routes: unknown token: <path>" instead of rendering any page, index, or
link home.
