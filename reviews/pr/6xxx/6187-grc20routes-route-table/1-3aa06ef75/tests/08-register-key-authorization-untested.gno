// Register's authorization for the DECLARED path rests entirely on
// `rlmPath := cur.Previous().PkgPath()` at grc20routes.gno:296, which the doc
// (:68-72) and this same file's TestRouteAlwaysPointsAtItsOwnRealm (:395) call
// out as the reason "a realm can only ever write keys beneath its own path".
// But every one of the 20 Test funcs in grc20routes_test.gno runs inside
// `package grc20routes` and mints its own tokens with mint(), so
// cur.Previous().PkgPath() is always selfPath: no test ever calls Register from
// a second, different realm attempting a foreign key. Measured at head
// 3aa06ef75: replacing the caller-path-derived key with an argument-derived one
// (a symbol shaped "victimPath|SYM" overrides rlmPath) leaves the full package
// suite green, 20/20 — the exact authorization break the doc's "cannot be
// forged" section claims is impossible goes undetected.
//
// # from a local clone of github.com/gnolang/gno:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// G=examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
// python3 - <<'PYEOF'
// f = "examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno"
// s = open(f).read()
// old = '''func Register(cur realm, symbol string, r Route) string {
// 	rlmPath := cur.Previous().PkgPath()
// 	if rlmPath == "" {
// 		panic("grc20routes: caller is not a realm")
// 	}'''
// new = '''func Register(cur realm, symbol string, r Route) string {
// 	rlmPath := cur.Previous().PkgPath()
// 	if rlmPath == "" {
// 		panic("grc20routes: caller is not a realm")
// 	}
// 	if i := strings.IndexByte(symbol, '|'); i >= 0 {
// 		rlmPath = symbol[:i]
// 		symbol = symbol[i+1:]
// 	}'''
// assert old in s
// open(f, "w").write(s.replace(old, new))
// PYEOF
// (cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/nt/grc20routes/v0)   # still ok, 20/20
// git checkout "$G"
//
// # observed output (trimmed):
// # ok      ./gno.land/r/nt/grc20routes/v0 	3.12s
// # (no test failed; none of the 20 tests ever pass a "|"-shaped symbol or calls
// #  Register from outside package grc20routes, so the authorization the doc
// #  describes is asserted by comment, not by a test)
package grc20routes
