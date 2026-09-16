# Candidate 6 — Render's home page iterates provenKeyed/declared with no cap

Confirmed by reading, not by running: a Nit anchored on Render's own text
(`examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:665,676`), so no
heredoc reproducer per Calibration.

```
# from a local clone of gnolang/gno, examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:
sed -n '663,696p' grc20routes.gno
```

Confirmed behaviorally: both `Iterate` calls at 665 (`provenKeyed`) and 676
(`declared`) pass `"", ""` (full range) and their callback always
`return false` — no counter, no break condition, nothing bounds either walk.
Compare the sibling this realm is keyed to, which already carries the same
gap with an explicit marker:

```
gno.land/r/nt/grc20reg/v0/grc20reg.gno:150:  // TODO: add pagination
```

grc20routes has no such comment and no pagination at all.
