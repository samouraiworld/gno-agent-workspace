# Review: PR #5737
Event: REQUEST_CHANGES

## Body
Correct for the interface-boxed nil receiver. Verified on 4c57c37e4: reverting the new deferred-panic path to master's eager panic makes a concrete `defer pt.M()` filetest print Go's `0` instead of the PR's `1`, and a value method bound to a nil pointer through an interface, once persisted and reloaded, runs on a zero receiver and returns instead of panicking.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/1-4c57c37e4/claude-opus-4-8_davd-gzl.md [↗](claude-opus-4-8_davd-gzl.md)

Repros run at 4c57c37e4.

*(AI Agent)*

## gnovm/pkg/gnolang/values.go:1835 [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1835)
A concrete `defer pt.M()` or `g := pt.M` on a nil `*T` value method now panics at call time, but Go and master panic eagerly when the method value is formed (the deferred `r = 1` never runs, so `f()` returns `0`). The `VPDerefValMethod`-nil path can't tell a concrete `*T` from an interface unwrapped to `*T`, so it defers both, while only the interface case should defer. The distinction has to be made where the interface is unwrapped (interface method dispatch) or marked upstream in the preprocessor, so this single site can't be correct for both cases.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5737 -R gnolang/gno
# `// Output: 0` is Go's behaviour (eager panic at the defer stmt; r=1 never runs).
# master prints 0; this PR prints 1, so the filetest FAILS on the PR.
cat > gnovm/tests/files/zz_concrete_defer.gno <<'EOF'
package main

type T struct{ x int }

func (T) M() {}

var pt *T

func f() (r int) {
	defer func() { recover() }()
	defer pt.M()
	r = 1
	return
}

func main() {
	println(f())
}

// Output:
// 0
EOF
go test -run 'TestFiles/zz_concrete_defer.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zz_concrete_defer.gno
```
```
--- FAIL: TestFiles/zz_concrete_defer.gno (0.00s)
    files_test.go:111: Output diff:
        --- Expected
        +++ Actual
        @@ -1 +1 @@
        -0
        +1
FAIL
```
</details>

*(AI Agent)*

## gnovm/pkg/gnolang/values.go:655 [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L655)
A value method bound to a nil pointer through an interface (`G := i.M`) is storable, but the unexported `nilReceiverPanic` flag is dropped by amino and `exportValue`, so after a store round-trip `G` runs on a zero receiver and returns instead of panicking. It panics in the tx that bound it and succeeds when called in a later tx: deterministic across nodes, so no consensus split, but a Go-semantics divergence. If the deferred-panic model stays, the panic-at-call property must survive serialization (a persisted marker, or deterministic reconstruction on load) rather than living in a transient unexported bool.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5737 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_nil_persist.txtar <<'EOF'
gnoland start

gnokey maketx addpkg -pkgdir $WORK/myrealm -pkgpath gno.land/r/test/myrealm -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
stdout OK!

# same tx as the bind: in-memory flag intact -> nil deref panic (matches Go)
! gnokey maketx call -pkgpath gno.land/r/test/myrealm -func BindAndCall -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
stderr 'nil pointer dereference'

# persist G = i.M (amino drops the unexported flag)
gnokey maketx call -pkgpath gno.land/r/test/myrealm -func Bind -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
stdout OK!

# later tx: G reloaded, flag gone -> no panic, method runs on a zero receiver
gnokey maketx call -pkgpath gno.land/r/test/myrealm -func CallLater -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test test1
stdout ZERO-RECEIVER
stdout OK!

-- myrealm/gnomod.toml --
module = "gno.land/r/test/myrealm"
gno = "0.9"

-- myrealm/myrealm.gno --
package myrealm

type I interface{ M() string }

type T struct{ x int }

func (T) M() string { return "ZERO-RECEIVER" }

var pt *T
var G func() string

func BindAndCall(cur realm) string { var i I = pt; G = i.M; return G() }
func Bind(cur realm)               { var i I = pt; G = i.M }
func CallLater(cur realm) string   { return G() }
EOF
go test -run 'TestTestdata/zz_nil_persist$' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_nil_persist.txtar
```
```
ok  	github.com/gnolang/gno/gno.land/pkg/integration	3.0s
# PASS == bug present: BindAndCall (same tx) panics with nil deref,
# but CallLater (later tx) returns ("ZERO-RECEIVER" string) + OK! instead of panicking.
```
</details>

*(AI Agent)*
