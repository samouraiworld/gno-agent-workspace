# Review: PR [#5923](https://github.com/gnolang/gno/pull/5923)
Posted: https://github.com/gnolang/gno/pull/5923#pullrequestreview-4790426069
Event: REQUEST_CHANGES

## Body
[AI bot]

Verified on 37466ba3c: two node runs that differ only by one simulated `addpkg` reach opposite outcomes on the same later transaction, and both runs behave identically at the merge-base d14a03770. Warm and cold gas match exactly on two realm shapes beyond the one [`typecache_restart_gas.txtar`](https://github.com/gnolang/gno/blob/37466ba3c/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar) covers, so nothing in the memo's reach shows up in billed gas today.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/2-37466ba3c/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gnovm/pkg/gnolang/realm.go:1293-1296 [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1293-L1296) [posted](https://github.com/gnolang/gno/pull/5923#discussion_r3660029942)
Critical: a verdict computed inside a discarded transaction is kept, so simulating a public build of a path and then deploying a private package at that path leaves [the enforcement walk skipped](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1420) for those types for the life of the process. An array of nil pointers to the private type then persists into a public realm, with no private object for [`assertObjectIsPublic`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1179-L1183) to catch. Simulation is [an ABCI query](https://github.com/gnolang/gno/blob/37466ba3c/tm2/pkg/crypto/keys/client/broadcast.go#L243) answered by one node and never replicated, so a validator that served it accepts a transaction that a validator that did not will panic on.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5923 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/zz_poison.txtar <<'EOF'
gnoland start

gnokey maketx addpkg -pkgdir $WORK/pub -pkgpath gno.land/r/zz/pub -gas-fee 1000001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1
stdout OK!

# The only line that differs from the control below.
gnokey maketx addpkg -pkgdir $WORK/priv_public -pkgpath gno.land/r/zz/priv -simulate only -gas-fee 1000001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1

gnokey maketx addpkg -pkgdir $WORK/priv -pkgpath gno.land/r/zz/priv -gas-fee 1000001ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1
stdout OK!

! gnokey maketx call -pkgpath gno.land/r/zz/priv -func Leak -gas-fee 5000000ugnot -gas-wanted 50000000 -chainid=tendermint_test test1
stderr 'cannot persist object of type defined in the private realm gno.land/r/zz/priv'

-- pub/gnomod.toml --
module = "gno.land/r/zz/pub"
gno = "0.9"

-- pub/pub.gno --
package pub

var Slot any

func Save(cur realm, v any) {
	Slot = v
}

-- priv/gnomod.toml --
module = "gno.land/r/zz/priv"
gno = "0.9"
private = true

-- priv/priv.gno --
package priv

import "gno.land/r/zz/pub"

type Data struct {
	N int
}

// Persisted at deploy time, so the deploy's own commit memoizes Data.
var keep = &Data{N: 0}

// Carries the private type but no object from the private realm.
func Leak(cur realm) {
	var arr [3]*Data
	pub.Save(cross(cur), arr)
}

-- priv_public/gnomod.toml --
module = "gno.land/r/zz/priv"
gno = "0.9"

-- priv_public/priv.gno --
package priv

import "gno.land/r/zz/pub"

type Data struct {
	N int
}

var keep = &Data{N: 0}

func Leak(cur realm) {
	var arr [3]*Data
	pub.Save(cross(cur), arr)
}
EOF

# The control: the same file with the simulated addpkg removed.
grep -v 'simulate only' gno.land/pkg/integration/testdata/zz_poison.txtar \
  > gno.land/pkg/integration/testdata/zz_control.txtar

go test ./gno.land/pkg/integration/ -run 'TestTestdata/zz_control$' -count=1
go test ./gno.land/pkg/integration/ -run 'TestTestdata/zz_poison$' -count=1

rm gno.land/pkg/integration/testdata/zz_poison.txtar gno.land/pkg/integration/testdata/zz_control.txtar
```

```
ok  	github.com/gnolang/gno/gno.land/pkg/integration	1.354s

    OK!
    GAS USED:   2105566
    FAIL: testdata/zz_poison.txtar:14: unexpected "gnokey" command success
    FAIL: testdata/zz_poison.txtar:15: no match for `cannot persist object of type defined in the private realm gno.land/r/zz/priv` found in stderr
--- FAIL: TestTestdata/zz_poison (1.49s)
FAIL	github.com/gnolang/gno/gno.land/pkg/integration	1.433s
```

Both files pass at the merge-base d14a03770.
</details>

## gnovm/pkg/gnolang/store.go:192-193 [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L192-L193) [posted](https://github.com/gnolang/gno/pull/5923#discussion_r3660029956)
`TypeID` does not distinguish a type declared in a private package from a structurally identical one declared in a public package at the same path, and privacy is read from the [`PackageValue`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L2333-L2339) at walk time rather than from anything the key carries. The same claim at [`realm.go:1421-1423`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1421-L1423) is what licenses skipping the enforcement walk, so it is the argument the optimization rests on. State which transactions a verdict may be drawn from instead.
