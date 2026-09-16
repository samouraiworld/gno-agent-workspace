# provenKeyed has no removal path, and Register only ever corrects one token

Asserts: `provenKeyed` (grc20routes.gno:158) is written only at :416
(`FinishKeyedProof`) and read at :442/:456/:665; no `Remove` call exists
anywhere in the package. `Register` (:295) keys on
`cur.Previous().PkgPath()`, so only the realm whose own deployed code calls it
can correct one of its own tokens — and only that one token, per
`TestDeclaredBeatsProven` (grc20routes_test.gno:272-293), whose own assertion
at :292 is `"and does not disturb its siblings"`: after `Register` runs for
`keyA`, `Get(keyB)` still returns `SourceProven`.

Measured at head 3aa06ef75:
```
grep -n 'provenKeyed' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
# 158: provenKeyed = avl.NewTree()      (init)
# 416: provenKeyed.Set(pathA, true)     (the only write)
# 442: if provenKeyed.Has(pkgPath) {    (Get)
# 456: func IsKeyedRealm ... provenKeyed.Has(pkgPath)
# 665: provenKeyed.Iterate(...)         (Render, read-only)
grep -rn 'grc20routes' examples/gno.land/r/demo/defi/grc20factory
# (no output — grc20factory, named in the package doc :30-32 as live and
#  un-redeployable, imports nothing from this package and cannot call Register)
```

# from a local clone of github.com/gnolang/gno:
git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
(cd examples && go run ../gnovm/cmd/gno test -v -run 'TestDeclaredBeatsProven' ./gno.land/r/nt/grc20routes/v0)

# observed output (trimmed):
```
=== RUN   TestDeclaredBeatsProven
--- PASS: TestDeclaredBeatsProven (0.00s)
ok      ./gno.land/r/nt/grc20routes/v0 	...s
```
The pass itself is the proof: the test's own comment ("and does not disturb
its siblings") and its assertion that `keyB` stays `SourceProven` after
`Register(keyA, ...)` runs is the diff's own test demonstrating that a wrong
`provenKeyed` mark cannot be cleared in bulk, only patched token by token, and
only by a realm whose code can still call `Register` at all.

Consequence: a realm marked keyed by mistake (via the already-confirmed forged
proof, or via the grief case the package doc names at :81-84 — two
separately-named approves) has no path back to `SourceConvention` or a correct
declared shape for any token it has not individually re-`Register`ed, and a
realm deployed before grc20routes existed (grc20factory is the doc's own
example) cannot call `Register` at all without being redeployed. The mark is
permanent for that population's entire present and future token set; nothing
in this package can revoke it.
