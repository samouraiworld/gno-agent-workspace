# Candidate #56 — `renderToken` parses the raw token key instead of routing through `mustToken`/`PkgPath`

Anchor: `examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:700-701`

```bash
# from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
sed -n '505,520p;699,715p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
sed -n '1,44p;59,79p' examples/gno.land/p/nt/fqname/v0/fqname.gno
```

Confirmed by reading, no run needed — trace the two paths a token key can take:

```
grc20routes.gno:505-513: mustToken's own comment: "The '#subpath' a realm carries while operating
                          under a sub identity is stripped, because the result is handed to
                          consumers as a MsgCall destination and a sub path is not one."
grc20routes.gno:514-519: func mustToken(tokenKey string) string {
                              ...
                              pkgPath, _ := fqname.Parse(tokenKey)
                              return hostOf(pkgPath)   // <- strips "#subpath" via SplitPkgSubPath
                          }
grc20routes.gno:453:      func PkgPath(tokenKey string) string { return mustToken(tokenKey) }
grc20routes.gno:700-701:  func renderToken(tokenKey string) string {
                              r, source := Get(tokenKey)
                              pkgPath, symbol := fqname.Parse(tokenKey)   // <- raw Parse, no hostOf
```

`fqname.Parse` (fqname.gno:17-44) does no sub-path stripping at all — it only splits on the last
`/` and the following `.`, so any `#subpath` suffix in the package-path component of `tokenKey`
survives verbatim into `pkgPath`. `renderToken` feeds that unstripped `pkgPath` straight into
`fqname.RenderLink` (fqname.gno:62-79), which also performs no stripping — it only decides
`gno.land`-prefixed vs. not and builds the markdown link from whatever `pkgPath` it is given.

So for a token registered from a sub identity (a realm key of the shape
`gno.land/r/x#admin.SYM`), the two documented "destination" answers for the same token diverge:

- `PkgPath(key)` / `JSON(key)`'s `"pkg_path"` field → `gno.land/r/x` (through `mustToken`, stripped)
- `renderToken(key)`'s `"- realm:"` line and its markdown link → built from `gno.land/r/x#admin`
  (through raw `fqname.Parse`, unstripped)

The human-facing `Render` output is the one that keeps the uncallable form: `mustToken`'s own
comment states the reason the stripping exists ("the result is handed to consumers as a MsgCall
destination and a sub path is not one"), and `renderToken` is exactly a place the result is handed
to a consumer, but it bypasses the function that does the stripping.
