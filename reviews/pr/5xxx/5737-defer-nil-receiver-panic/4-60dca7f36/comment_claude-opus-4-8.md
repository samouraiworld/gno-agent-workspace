# Review: PR [#5737](https://github.com/gnolang/gno/pull/5737)
Event: APPROVE

## Body
Verified on 60dca7f36 by booting gnodev over a realm holding a package-level `G = i.Get`. The state page for that object renders; rebuilt with the `IsLazy()` branch deleted, the same URL turns into a 500 with a recovered nil-pointer dereference in the log. The `c5` nil deref recovers as an `error`, matching real Go on go1.26.5; reverting that guard to `typedString` flips the filetest back to `c5 1`.

The red `docs` check is a dead link in `docs/MANIFESTO.md`, not a code problem.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/4-60dca7f36/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## SKIP gnovm/pkg/gnolang/values.go:784 [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L784)
Already raised: https://github.com/gnolang/gno/pull/5737#discussion_r3513857885

A method value bound over a function-local type panics `unexpected type with id ...S` on reload, where master returned the value. Local types were never persistable in general, so the trigger is narrow, but the failure surfaces as an internal panic rather than a clear error.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5737 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/z_lt_repro.txtar <<'EOF'
gnoland start
gnokey maketx addpkg -pkgdir $WORK/lt -pkgpath gno.land/r/test/lt -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
stdout OK!
gnokey maketx call -pkgpath gno.land/r/test/lt -func Bind -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
stdout OK!
! gnokey maketx call -pkgpath gno.land/r/test/lt -func Call -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
stderr 'unexpected type with id'
-- lt/gnomod.toml --
module = "gno.land/r/test/lt"
gno = "0.9"
-- lt/lt.gno --
package lt
type T struct{ x int }
func (t T) Get() int { return t.x }
type I interface{ Get() int }
var G func() int
func Bind(cur realm) {
	type S struct{ T }
	var i I = S{T{7}}
	G = i.Get
}
func Call(cur realm) int { return G() }
EOF
go test -run 'TestTestdata/z_lt_repro$' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/z_lt_repro.txtar
```

```
# this PR: reload tx fails with
Data: unexpected type with id gno.land/r/test/lt[gno.land/r/test/lt/lt.gno:6:1-10:2].S
# master (a19f13f90): the same Call returns (7 int)
```
</details>

## gnovm/tests/files/method_iface_cyclic_value.gno:1 [↗](../../../../../.worktrees/gno-review-5737/gnovm/tests/files/method_iface_cyclic_value.gno)
Missing test: the struct-carried cycle is pinned only by an in-memory filetest. The existing [`method_iface_cyclic_persist.txtar`](https://github.com/gnolang/gno/blob/60dca7f36/gno.land/pkg/integration/testdata/method_iface_cyclic_persist.txtar) · [↗](../../../../../.worktrees/gno-review-5737/gno.land/pkg/integration/testdata/method_iface_cyclic_persist.txtar) covers the pointer cycle `s.IG = s`, so no test guards that the struct-carried shape still terminates after a store round-trip.

<details><summary>test case</summary>

Stores the cycle and reaches it through qrender; terminates with the fatal panic. Add as `gno.land/pkg/integration/testdata/method_iface_cyclic_value_persist.txtar`.

```
gnoland start
gnokey maketx addpkg -pkgdir $WORK/cyc -pkgpath gno.land/r/test/cyc -gas-fee 1000000ugnot -gas-wanted 20000000 -chainid=tendermint_test test1
stdout OK!
gnokey maketx call -pkgpath gno.land/r/test/cyc -func Bind -gas-fee 1000000ugnot -gas-wanted 20000000 -chainid=tendermint_test test1
stdout OK!
! gnokey query vm/qrender --data 'gno.land/r/test/cyc:'
stdout 'cyclic embedded interface'
-- cyc/gnomod.toml --
module = "gno.land/r/test/cyc"
gno = "0.9"
-- cyc/cyc.gno --
package cyc
import "strconv"
type IG interface{ Get() int }
type Box struct{ IG }
type W struct{ *Box }
var G func() int
func Bind(cur realm) { s := &Box{}; s.IG = W{s}; var o IG = s; G = o.Get }
func Render(path string) string { return strconv.Itoa(G()) }
```
</details>
