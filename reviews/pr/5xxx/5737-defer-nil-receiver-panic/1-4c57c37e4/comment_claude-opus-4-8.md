# Review: PR #5737
Posted: https://github.com/gnolang/gno/pull/5737#pullrequestreview-4515999355
Event: REQUEST_CHANGES

## Body
This is a design problem, not a tweak. The root cause sits a level deeper than the single flag site, so as written the change fixes the interface case but regresses the concrete one. Go has two distinct timings: a concrete `*T(nil)` derefs `*pt` when the method value is formed, so it panics eagerly at bind, while an interface-boxed `*T` only derefs inside the call, so its panic lands at call time. The `nilReceiverPanic` flag is read at the single `VPDerefValMethod` site, but that site runs after the interface is unwrapped to a bare `*T`, so it can't tell the two apart and defers both, when only the interface case should defer. To hold both, the decision needs to live where the receiver kind is still known (interface dispatch, or marked in the preprocessor) rather than at that one late site.

```
receiver form        Go            master         this PR
-------------------  ------------  -------------  --------------
defer pt.M() concrete eager(=0)    eager(=0) ✓    call-time(=1) ✗   <- regressed
defer i.M()  iface    call-time(=1) eager(=0) ✗   call-time(=1) ✓   <- fixed
G=i.M; persist; G()   panic         (binds eager)  no panic ✗        <- new hole
```

The current filetest covers the interface case; the concrete case and the persistence hole below are both unexercised today, so they'd be worth adding. An ADR pinning the intended semantics would also help anchor the fix.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/1-4c57c37e4/claude-opus-4-8_davd-gzl.md [↗](claude-opus-4-8_davd-gzl.md)

Repros run at 4c57c37e4.


## gnovm/pkg/gnolang/values.go:1835 [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L1835) [posted](https://github.com/gnolang/gno/pull/5737#discussion_r3428467430)
A concrete `defer pt.M()` or `g := pt.M` on a nil `*T` value method now panics at call time, whereas Go and master panic eagerly when the method value is formed (the deferred `r = 1` never runs, so `f()` returns `0`). The `VPDerefValMethod`-nil path can't tell a concrete `*T` from an interface unwrapped to `*T`, so it defers both, while only the interface case should defer; the concrete path needs to keep panicking eagerly at bind.

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


## gnovm/pkg/gnolang/values.go:655 [↗](../../../../../.worktrees/gno-review-5737/gnovm/pkg/gnolang/values.go#L655) [posted](https://github.com/gnolang/gno/pull/5737#discussion_r3428467444)
Store a nil-receiver method value like `G := i.M` and the unexported `nilReceiverPanic` flag is dropped by amino and `exportValue`, so after a reload `G` runs on a zero receiver instead of panicking. It panics when called in the tx that bound it but succeeds in a later tx: same on every node, so no consensus split, but still a Go-semantics divergence. If the deferred-panic model stays, the panic-at-call property needs to survive serialization rather than living in a transient bool.

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

