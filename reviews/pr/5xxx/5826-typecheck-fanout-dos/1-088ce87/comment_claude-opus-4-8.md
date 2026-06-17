# Review: PR #5826
Event: REQUEST_CHANGES

## Body
The value-containment fan-out fix is correct and the guard is genuinely linear, but the same exponential `validType` DoS is still reachable through three type shapes the guard under-counts: generic instantiations, interface type-set unions, and imported types. Verified on 088ce87 end-to-end through `addpkg` on an in-memory gnoland node with drop-in txtars in the form of the PR's own `addpkg_typecheck_fanout.txtar`: a generic-instantiation fan-out (`A_n` embeds `W[A_{n-1}]`) and an interface union-doubling chain (`I_n = [0]I_{n-1} | [1]I_{n-1}`) each pass the guard and then hang the node past 60s on the unmetered deploy path instead of producing the `denial-of-service` rejection the guard gives the direct case; and a value-doubling chain split across a deploy chain of imported packages keeps every package's gas flat (~5M) while wall-clock `validType` time doubles each hop (0.1s, 0.3s, 0.9s, 3.5s, 14s) until the next deploy kills the node. The other arms hold: pointer/slice/map/chan/func breaks complete in microseconds, matching `validType`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/typecheck_bound.go:143-146 [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L143)
A value fan-out routed through a generic instantiation (`W[A{n-1}]`) is counted as `cost(W)`, a constant, because this arm drops the type argument, so the guard accepts it and `go/types` validType still hangs the unmetered deploy path. The base type does not bound the cost when the type argument drives the expansion. Fix: reject or conservatively over-count generic instantiations whose type arguments can drive value-containment fan-out, so a generic doubling chain hits the budget like the direct-value one.

<details><summary>repro (drop-in txtar, same form as the PR's own addpkg_typecheck_fanout.txtar)</summary>

Save as `gno.land/pkg/integration/testdata/addpkg_typecheck_fanout_generic.txtar`, then `go test ./gno.land/pkg/integration/ -run TestTestdata/addpkg_typecheck_fanout_generic`. On 088ce87 the guard does not reject the package, so the deploy reaches the unmetered `validType` walk and the node stops responding (the RPC client gives up with `context deadline exceeded`) instead of the clean `denial-of-service` rejection the test asserts. Swap `W[A%d]` for the direct `[0]T%d` form and the same depth is rejected at the guard in milliseconds, which is the difference this finding is about.

```txtar
# Adding a package whose types form an exponential value-containment fan-out
# routed through a generic instantiation must ALSO be rejected cleanly at
# type-check time, not hang the node. Each A_n embeds W[A_{n-1}] by value and
# W holds its type parameter twice, so the validType walk still doubles, but
# checkTypeExpansionBound's cost() drops the IndexExpr type argument.

# start a new node
gnoland start

# adding the generic fan-out package must fail with a denial-of-service rejection
! gnokey maketx addpkg -pkgdir $WORK/fanout -pkgpath gno.land/r/foobar/fanout -gas-fee 350001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test $test1_user_addr
stderr 'denial-of-service'

-- fanout/gnomod.toml --
module = "gno.land/r/foobar/fanout"
gno = "0.9"

-- fanout/fanout.gno --
package fanout

type W[P any] struct{ a, b [0]P }
type A0 struct{ v int }

type A1 struct{ x W[A0] }
type A2 struct{ x W[A1] }
type A3 struct{ x W[A2] }
type A4 struct{ x W[A3] }
type A5 struct{ x W[A4] }
type A6 struct{ x W[A5] }
type A7 struct{ x W[A6] }
type A8 struct{ x W[A7] }
type A9 struct{ x W[A8] }
type A10 struct{ x W[A9] }
type A11 struct{ x W[A10] }
type A12 struct{ x W[A11] }
type A13 struct{ x W[A12] }
type A14 struct{ x W[A13] }
type A15 struct{ x W[A14] }
type A16 struct{ x W[A15] }
type A17 struct{ x W[A16] }
type A18 struct{ x W[A17] }
type A19 struct{ x W[A18] }
type A20 struct{ x W[A19] }
type A21 struct{ x W[A20] }
type A22 struct{ x W[A21] }
type A23 struct{ x W[A22] }
type A24 struct{ x W[A23] }
type A25 struct{ x W[A24] }
type A26 struct{ x W[A25] }
type A27 struct{ x W[A26] }
type A28 struct{ x W[A27] }
type A29 struct{ x W[A28] }
type A30 struct{ x W[A29] }
type A31 struct{ x W[A30] }
type A32 struct{ x W[A31] }
type A33 struct{ x W[A32] }
type A34 struct{ x W[A33] }
type A35 struct{ x W[A34] }
type A36 struct{ x W[A35] }
type A37 struct{ x W[A36] }
type A38 struct{ x W[A37] }
type A39 struct{ x W[A38] }
type A40 struct{ x W[A39] }

var Sink A40
```

Observed on 088ce87 (the guard never fires; the node hangs until the client deadline):
```
# adding the generic fan-out package must fail with a denial-of-service rejection (65.00s)
> ! gnokey maketx addpkg ... -pkgpath gno.land/r/foobar/fanout ...
"gnokey" error: unable to call RPC method abci_query, unable to send request, Post "http://127.0.0.1:...": context deadline exceeded
> stderr 'denial-of-service'
FAIL: testdata/addpkg_typecheck_fanout_generic.txtar:12: no match for `denial-of-service` found in stderr
```
</details>

*(AI Agent)*

## gnovm/pkg/gnolang/typecheck_bound.go:127-138 [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L127)
The `*ast.InterfaceType` arm only recurses embedded named types; an interface type-set union (`[0]X | [1]X`) is a single field whose `Type` is an `*ast.BinaryExpr` (and `~T` is an `*ast.UnaryExpr`), both of which fall through to `default: return 1` at line 147-148. But `validType` does walk interface type-set terms, so a union-doubling chain is scored as a constant by the guard yet drives the same exponential walk. Fix: recurse both operands of a `|` `*ast.BinaryExpr` and the operand of a `~` `*ast.UnaryExpr` inside the interface/type-elem handling, mirroring the struct/array value-containment recursion.

<details><summary>repro (drop-in txtar, same form as the PR's own addpkg_typecheck_fanout.txtar)</summary>

Save as `gno.land/pkg/integration/testdata/addpkg_typecheck_fanout_union.txtar`, then `go test ./gno.land/pkg/integration/ -run TestTestdata/addpkg_typecheck_fanout_union`. The union term `[0]I{n-1} | [1]I{n-1}` is an `*ast.BinaryExpr` the guard scores as `1`, so `checkTypeExpansionBound` returns nil and the deploy reaches the exponential `validType` walk; on 088ce87 the node hangs until the client deadline rather than rejecting.

```txtar
# Adding a package whose interface type-set unions form an exponential
# value-containment fan-out must be rejected cleanly at type-check time, not
# hang the node. Each I_n unions two array types over I_{n-1}, so go/types'
# validType walk over the type set still doubles per level, but
# checkTypeExpansionBound's *ast.InterfaceType arm does not recurse the
# *ast.BinaryExpr union term: it falls through to default and returns 1.

# start a new node
gnoland start

# adding the union fan-out package must fail with a denial-of-service rejection
! gnokey maketx addpkg -pkgdir $WORK/fanout -pkgpath gno.land/r/foobar/fanout -gas-fee 350001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test $test1_user_addr
stderr 'denial-of-service'

-- fanout/gnomod.toml --
module = "gno.land/r/foobar/fanout"
gno = "0.9"

-- fanout/fanout.gno --
package fanout

type I0 interface{ m() }

type I1 interface{ [0]I0 | [1]I0 }
type I2 interface{ [0]I1 | [1]I1 }
type I3 interface{ [0]I2 | [1]I2 }
type I4 interface{ [0]I3 | [1]I3 }
type I5 interface{ [0]I4 | [1]I4 }
type I6 interface{ [0]I5 | [1]I5 }
type I7 interface{ [0]I6 | [1]I6 }
type I8 interface{ [0]I7 | [1]I7 }
type I9 interface{ [0]I8 | [1]I8 }
type I10 interface{ [0]I9 | [1]I9 }
type I11 interface{ [0]I10 | [1]I10 }
type I12 interface{ [0]I11 | [1]I11 }
type I13 interface{ [0]I12 | [1]I12 }
type I14 interface{ [0]I13 | [1]I13 }
type I15 interface{ [0]I14 | [1]I14 }
type I16 interface{ [0]I15 | [1]I15 }
type I17 interface{ [0]I16 | [1]I16 }
type I18 interface{ [0]I17 | [1]I17 }
type I19 interface{ [0]I18 | [1]I18 }
type I20 interface{ [0]I19 | [1]I19 }
type I21 interface{ [0]I20 | [1]I20 }
type I22 interface{ [0]I21 | [1]I21 }
type I23 interface{ [0]I22 | [1]I22 }
type I24 interface{ [0]I23 | [1]I23 }
type I25 interface{ [0]I24 | [1]I24 }
type I26 interface{ [0]I25 | [1]I25 }
type I27 interface{ [0]I26 | [1]I26 }
type I28 interface{ [0]I27 | [1]I27 }
type I29 interface{ [0]I28 | [1]I28 }
type I30 interface{ [0]I29 | [1]I29 }
type I31 interface{ [0]I30 | [1]I30 }
type I32 interface{ [0]I31 | [1]I31 }
type I33 interface{ [0]I32 | [1]I32 }
type I34 interface{ [0]I33 | [1]I33 }
type I35 interface{ [0]I34 | [1]I34 }
type I36 interface{ [0]I35 | [1]I35 }
type I37 interface{ [0]I36 | [1]I36 }
type I38 interface{ [0]I37 | [1]I37 }
type I39 interface{ [0]I38 | [1]I38 }
type I40 interface{ [0]I39 | [1]I39 }

type Use struct{ x I40 }
```

Observed on 088ce87:
```
# adding the union fan-out package must fail with a denial-of-service rejection (65.00s)
> ! gnokey maketx addpkg ... -pkgpath gno.land/r/foobar/fanout ...
"gnokey" error: unable to call RPC method abci_query, unable to send request, Post "http://127.0.0.1:...": context deadline exceeded
> stderr 'denial-of-service'
FAIL: testdata/addpkg_typecheck_fanout_union.txtar:13: no match for `denial-of-service` found in stderr
```
</details>

*(AI Agent)*

## gnovm/pkg/gnolang/typecheck_bound.go:141-142 [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L141)
`*ast.SelectorExpr` (an imported type `pkg.T`) is scored as a leaf (`return 1`). That is sound within one package, but `validType` follows value containment across package boundaries and does not memoize (golang/go#65711), so it re-expands imported types in full. The guard runs per package at each `AddPackage`, while `validType` at that same deploy walks the whole transitive type graph. A value-doubling chain split across a deploy-chain of packages, each with a trivial local cost (`pkg.T` = 1), makes the Nth package's `validType` walk expand 2^N while every package individually passes the per-package budget; the deploy that first crosses the time threshold hangs the unmetered path. This is an architectural gap, not a single-package miscount: the guard cannot see the imported cost. Fix: persist each package's max named-type expansion cost (e.g. in `TypeCheckCache`) and add it for `SelectorExpr`, instead of treating imported types as constant leaves.

<details><summary>repro (drop-in txtar: a deploy chain, each package passes the guard, the last one falls over)</summary>

Save as `gno.land/pkg/integration/testdata/addpkg_typecheck_fanout_imported.txtar`, then `go test ./gno.land/pkg/integration/ -run TestTestdata/addpkg_typecheck_fanout_imported`. `p0` is a depth-16 value-doubling chain (cost 2^16, under the 1M budget); `p1..p5` each embed the previous package's `T` four times by value. Per package the guard cost stays tiny because `p{k-1}.T` is a `SelectorExpr` scored as `1`, so every package passes. But `validType` re-expands the imported chain each deploy, so the deploys succeed with flat gas while wall-clock time doubles per hop, and `p5` takes the node down.

```txtar
# A value-containment fan-out SPLIT ACROSS A DEPLOY CHAIN of packages. Each
# package passes checkTypeExpansionBound on its own: imported types are scored as
# a constant leaf (*ast.SelectorExpr -> return 1), so every package's cost stays
# far under the per-package budget. But go/types' validType follows value
# containment across import boundaries and does not memoize, so the walk doubles
# per package and the final deploy hangs/kills the unmetered addpkg path even
# though no single package ever crosses the budget.

# start a new node
gnoland start

# deploy p0
gnokey maketx addpkg -pkgdir $WORK/p0 -pkgpath gno.land/r/foobar/p0 -gas-fee 350001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test $test1_user_addr

# deploy p1
gnokey maketx addpkg -pkgdir $WORK/p1 -pkgpath gno.land/r/foobar/p1 -gas-fee 350001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test $test1_user_addr

# deploy p2
gnokey maketx addpkg -pkgdir $WORK/p2 -pkgpath gno.land/r/foobar/p2 -gas-fee 350001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test $test1_user_addr

# deploy p3
gnokey maketx addpkg -pkgdir $WORK/p3 -pkgpath gno.land/r/foobar/p3 -gas-fee 350001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test $test1_user_addr

# deploy p4
gnokey maketx addpkg -pkgdir $WORK/p4 -pkgpath gno.land/r/foobar/p4 -gas-fee 350001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test $test1_user_addr

# deploy p5 (passes the guard, but its validType walk re-expands the whole imported chain)
! gnokey maketx addpkg -pkgdir $WORK/p5 -pkgpath gno.land/r/foobar/p5 -gas-fee 350001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test $test1_user_addr
stderr 'denial-of-service'

-- p0/gnomod.toml --
module = "gno.land/r/foobar/p0"
gno = "0.9"

-- p0/p0.gno --
package p0

type t0 struct{ v int }
type t1 struct{ a, b [0]t0 }
type t2 struct{ a, b [0]t1 }
type t3 struct{ a, b [0]t2 }
type t4 struct{ a, b [0]t3 }
type t5 struct{ a, b [0]t4 }
type t6 struct{ a, b [0]t5 }
type t7 struct{ a, b [0]t6 }
type t8 struct{ a, b [0]t7 }
type t9 struct{ a, b [0]t8 }
type t10 struct{ a, b [0]t9 }
type t11 struct{ a, b [0]t10 }
type t12 struct{ a, b [0]t11 }
type t13 struct{ a, b [0]t12 }
type t14 struct{ a, b [0]t13 }
type t15 struct{ a, b [0]t14 }
type T struct{ a, b [0]t15 }

-- p1/gnomod.toml --
module = "gno.land/r/foobar/p1"
gno = "0.9"

-- p1/p1.gno --
package p1

import "gno.land/r/foobar/p0"

type T struct{ a, b, c, d [0]p0.T }

-- p2/gnomod.toml --
module = "gno.land/r/foobar/p2"
gno = "0.9"

-- p2/p2.gno --
package p2

import "gno.land/r/foobar/p1"

type T struct{ a, b, c, d [0]p1.T }

-- p3/gnomod.toml --
module = "gno.land/r/foobar/p3"
gno = "0.9"

-- p3/p3.gno --
package p3

import "gno.land/r/foobar/p2"

type T struct{ a, b, c, d [0]p2.T }

-- p4/gnomod.toml --
module = "gno.land/r/foobar/p4"
gno = "0.9"

-- p4/p4.gno --
package p4

import "gno.land/r/foobar/p3"

type T struct{ a, b, c, d [0]p3.T }

-- p5/gnomod.toml --
module = "gno.land/r/foobar/p5"
gno = "0.9"

-- p5/p5.gno --
package p5

import "gno.land/r/foobar/p4"

type T struct{ a, b, c, d [0]p4.T }
```

Observed on 088ce87 (gas stays flat while wall-clock doubles each hop, then the node dies):
```
# deploy p0 (0.118s)   GAS USED: 3055141
# deploy p1 (0.297s)   GAS USED: 4126720
# deploy p2 (0.914s)   GAS USED: 4349353
# deploy p3 (3.467s)   GAS USED: 4598450
# deploy p4 (13.556s)  GAS USED: 4847547
# deploy p5 (28.079s)  "gnokey" error: ... Post "http://127.0.0.1:...": EOF   <- node down
> stderr 'denial-of-service'
FAIL: testdata/addpkg_typecheck_fanout_imported.txtar:29: no match for `denial-of-service` found in stderr
```
</details>

*(AI Agent)*
