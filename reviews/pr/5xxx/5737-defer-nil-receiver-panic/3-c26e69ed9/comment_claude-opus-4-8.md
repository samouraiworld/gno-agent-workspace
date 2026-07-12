# Review: PR [#5737](https://github.com/gnolang/gno/pull/5737)
Event: APPROVE

## Body
Verified on c26e69ed9 that the struct-value cycle terminates on the persisted tx path and the unbounded vm/qrender path, not only the in-memory filetest: each raises the fatal cyclic panic in about 3.5s instead of hanging. The reload keeps the struct operand identity cache-stable, so the seen-set fires. The `MethodPkg` field is wired through proto field 5 and the `init()` size self-check confirms `_allocBoundMethodValue` = 232 matches the struct.

<details><summary>cyclic fix on the persisted + unbounded paths</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5737 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/z_cyc_render.txtar <<'EOF'
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
EOF
go test -run 'TestTestdata/z_cyc_render$' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/z_cyc_render.txtar
```

```
ok  	github.com/gnolang/gno/gno.land/pkg/integration	3.5s
# qrender fails with: cyclic embedded interface in method-value dispatch (no hang)
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/3-c26e69ed9/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values.go:739 [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L739)
A method value bound over a function-local type panics `unexpected type with id ...S` on reload, where master returned the value. The lazy bind saves the operand's own type and re-derives the trail at the call through `Store.GetType`, but a function-local type is never written to the type store; master eager-bound the promoted package-level receiver and dodged this. A raw interface value over a local type already fails on both master and this PR, so resolve eagerly for a non-persistable operand type or reject the bind with a clear error rather than the internal panic.

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
Missing test: the struct-carried cycle is pinned only in an in-memory filetest. On the persisted, unbounded query and `Render()` paths a non-terminating walk would hang the node, and termination there relies on the struct operand's identity staying cache-stable on reload, which no test guards.

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
