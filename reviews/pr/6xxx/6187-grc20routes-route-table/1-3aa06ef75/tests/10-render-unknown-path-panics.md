# Candidate 10 — Render on an unknown path panics through mustToken

Confirmed by reading, not by running: a Nit anchored on Render's own dispatch
(`grc20routes.gno:652-655`), so no heredoc reproducer per Calibration.

```
# from a local clone of gnolang/gno, examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:
sed -n '652,655p;699,701p;514,519p' grc20routes.gno
```

Confirmed behaviorally:

```
652:func Render(path string) string {
653:	if path != "" {
654:		return renderToken(path)
655:	}
...
699:func renderToken(tokenKey string) string {
700:	r, source := Get(tokenKey)
...
514:func mustToken(tokenKey string) string {
515:	if grc20reg.Get(tokenKey) == nil {
516:		panic("grc20routes: unknown token: " + tokenKey)
```

`renderToken` calls `Get` (line 700), which calls `mustToken` (line 437),
which panics on any `tokenKey` not already in `grc20reg`. `Render` passes any
non-empty path straight through with no recover and no existence check of
its own, so `/r/nt/grc20routes/v0:<anything unregistered>` panics instead of
rendering an error page. The diff's own `TestRejectsUnknownToken`
(`grc20routes_test.gno:310`) pins this panic for `Get` directly; nothing in
the suite exercises `Render` with an unknown path.
