# PR [#6187](https://github.com/gnolang/gno/pull/6187): feat(examples): add r/nt/grc20routes, a MsgCall route table for GRC20 tokens
Verdict: REQUEST CHANGES, on seven defects in the route this realm hands a wallet: a keyed proof needs no call to the realm whose shape it claims, the proven prefix is the registry symbol rather than the leading argument the proof observed, a realm marked keyed stays keyed for every token it will ever register, a sub identity's token and its host's token of the same symbol publish byte-identical routes, a dotted subpath rides in front of the symbol, `Func` alone cannot build a keyed call, and the rendered argument arrives markdown-escaped.
Event: REQUEST_CHANGES
Model: claude-opus-5[1m], standard review
Commit: 3aa06ef75 (latest)
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/3aa06ef757b87d731a3e554e96c7540a209fd9a3) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/3aa06ef757b87d731a3e554e96c7540a209fd9a3)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6187 3aa06ef75`
Round: 1. 6 finders, one critic, 58 candidates, each run from scratch by an agent that was not its finder.

## Body
- Every read here takes a `grc20reg` key, whose `#subpath` half [`grc20reg.Register` writes raw](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno#L39-L40), and this realm has no one place deciding what that half means.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:408-413 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L408-L413) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L408)
[`grc20reg.Approve`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno#L126-L128) sets the calling realm's own allowance on any registered token, so both comparisons pass for any realm hosting two of them, whatever shape its entry points take. Refusing a prover that is not a signing user, [`IsUserCall`](https://github.com/gnolang/gno/blob/3aa06ef75/gnovm/stdlibs/chain/runtime/frame.gno#L105) on `cur.Previous()`, leaves only allowances a token's own realm can move.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/zz_forged_proof.txtar <<'TXTAR'
# A keyed route is "proven" for a realm that exposes no write entry point at all.
# The package argues that only a realm whose Approve reads
# Approve(cur, SYMBOL, spender, amount) can make two of its tokens carry the
# prover's nonces at once. grc20reg.Approve(_ int, rlm realm, tokenKey, spender,
# amount) acts for the CALLING REALM on any registered token, so a prover that is
# itself a realm produces both allowances without ever calling the token realm.
# victim hosts two registered tokens and exposes nothing else. attacker runs the
# four proof calls. The route flips from source=convention to source=proven.

loadpkg gno.land/r/nt/grc20routes/v0

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid tendermint_test test1
stdout 'OK!'

gnokey maketx addpkg -pkgdir $WORK/victim -pkgpath gno.land/r/$alice_user_addr/victim -gas-fee 1000000ugnot -gas-wanted 40000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey maketx addpkg -pkgdir $WORK/attacker -pkgpath gno.land/r/$alice_user_addr/attacker -gas-fee 1000000ugnot -gas-wanted 40000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

# Before: the convention, which is the right answer — victim is not keyed.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/$alice_user_addr/victim.VICA\")"
stdout '\\"prefix\\":\[\],\\"source\\":\\"convention\\"'

# The forgery. Four calls, none of them to victim.
gnokey maketx call -pkgpath gno.land/r/$alice_user_addr/attacker -func Forge -args gno.land/r/$alice_user_addr/victim.VICA -args gno.land/r/$alice_user_addr/victim.VICB -args 111 -args 222 -gas-fee 1000000ugnot -gas-wanted 40000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

# After: victim reads as keyed, and every token it hosts is now routed
# Approve(cur, SYMBOL, spender, amount) — a shape victim does not have.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/$alice_user_addr/victim.VICA\")"
stdout '\\"prefix\\":\[\\"VICA\\"\],\\"source\\":\\"proven\\"'

gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/$alice_user_addr/victim.VICB\")"
stdout '\\"prefix\\":\[\\"VICB\\"\],\\"source\\":\\"proven\\"'

gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.IsKeyedRealm(\"gno.land/r/$alice_user_addr/victim\")"
stdout 'true'

-- victim/gnomod.toml --
module = "victim"
gno = "0.9"

-- victim/victim.gno --
// victim hosts two registered tokens and exposes no write entry point, so no
// Approve of any arity exists here.
package victim

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/p/nt/seqid/v0"

	"gno.land/r/nt/grc20reg/v0"
)

var id seqid.ID

func init(cur realm) {
	a, _ := grc20.NewToken("Victim A", "VICA", 6, id.Next(), cur)
	grc20reg.Register(cross(cur), a, "")
	b, _ := grc20.NewToken("Victim B", "VICB", 6, id.Next(), cur)
	grc20reg.Register(cross(cur), b, "")
}

-- attacker/gnomod.toml --
module = "attacker"
gno = "0.9"

-- attacker/attacker.gno --
package attacker

import (
	"gno.land/r/nt/grc20reg/v0"
	"gno.land/r/nt/grc20routes/v0"
)

// Forge marks the realm hosting keyA and keyB as keyed. The two allowances the
// proof reads are set through grc20reg, which acts for this realm on any
// registered token, so the target realm is never called.
func Forge(cur realm, keyA, keyB string, nonceA, nonceB int64) {
	grc20routes.BeginKeyedProof(cross(cur), keyA, keyB, nonceA, nonceB)
	grc20reg.Approve(0, cur, keyA, grc20routes.ProbeAddress(), nonceA)
	grc20reg.Approve(0, cur, keyB, grc20routes.ProbeAddress(), nonceB)
	grc20routes.FinishKeyedProof(cross(cur), keyA, keyB)
}
TXTAR

go test ./gno.land/pkg/integration/ -run 'TestTestdata/zz_forged_proof' -p 1 -parallel 1
rm gno.land/pkg/integration/testdata/zz_forged_proof.txtar
```

victim exposes no `Approve` of any arity, and its route carries a symbol prefix once the four calls land:

```
# before the proof
data: ("{"token_key":"gno.land/r/g1vszm.../victim.VICA","pkg_path":"gno.land/r/g1vszm.../victim","prefix":[],"source":"convention",...}" string)

# after the four calls, none of them to victim
data: ("{"token_key":"gno.land/r/g1vszm.../victim.VICA","pkg_path":"gno.land/r/g1vszm.../victim","prefix":["VICA"],"source":"proven",...}" string)
data: ("{"token_key":"gno.land/r/g1vszm.../victim.VICB","pkg_path":"gno.land/r/g1vszm.../victim","prefix":["VICB"],"source":"proven",...}" string)

# IsKeyedRealm("gno.land/r/g1vszm.../victim")
data: (true bool)
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:444 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L444) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L444)
[`FinishKeyedProof`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L408-L412) compares two nonces and never observes the leading argument that carried them, so the published prefix is the registry symbol and a realm keyed by [an IBC denom](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L130-L136) gets a route that aborts at every operation. Recording the leading argument the prover passed, instead of deriving it from the registry key, keeps the prefix to what the proof observed.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/grc20routes_denom_keyed.txtar <<'TXTAR'
loadpkg gno.land/r/nt/grc20routes/v0

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid tendermint_test test1
stdout 'OK!'

# zdenomv hosts two registered tokens behind grc20factory's exact Approve
# shape. Only the identifier space of the leading argument differs: an IBC
# denom, not the grc20 symbol.
gnokey maketx addpkg -pkgdir $WORK/zdenomv -pkgpath gno.land/r/demo/zdenomv -gas-fee 1000000ugnot -gas-wanted 40000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

# The four documented proof calls, run honestly: the prover passes the realm's
# real leading argument, so both allowances land through the realm's own entry
# point. Nothing is forged here.
gnokey maketx call -pkgpath gno.land/r/nt/grc20routes/v0 -func BeginKeyedProof -args gno.land/r/demo/zdenomv.VCHA -args gno.land/r/demo/zdenomv.VCHB -args 111 -args 222 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/demo/zdenomv -func Approve -args ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2 -args g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep -args 111 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/demo/zdenomv -func Approve -args ibc/9117A26BA81E29FA4F78F57DC2BD90CD3D26848101BA880445F119B22A1E254E -args g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep -args 222 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/nt/grc20routes/v0 -func FinishKeyedProof -args gno.land/r/demo/zdenomv.VCHA -args gno.land/r/demo/zdenomv.VCHB -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

# The route a wallet now reads. The assertion is the leading argument the proof
# observed.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/demo/zdenomv.VCHA\")"
stdout '\\"prefix\\":\[\\"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2\\"\],\\"source\\":\\"proven\\"'

# The consequence: MsgCall{Func: "Approve", Args: prefix + [spender, amount]}
# built from the route the head actually returns.
! gnokey maketx call -pkgpath gno.land/r/demo/zdenomv -func Approve -args VCHA -args $alice_user_addr -args 1 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stderr 'unknown denom VCHA'

-- zdenomv/gnomod.toml --
module = "gno.land/r/demo/zdenomv"
gno = "0.9"
-- zdenomv/zdenomv.gno --
package zdenomv

import (
	"gno.land/p/nt/avl/v0"
	"gno.land/p/nt/grc20/v0"
	"gno.land/p/nt/seqid/v0"

	"gno.land/r/nt/grc20reg/v0"
)

// The leading argument this realm accepts. Neither is a grc20 symbol.
const (
	DenomA = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	DenomB = "ibc/9117A26BA81E29FA4F78F57DC2BD90CD3D26848101BA880445F119B22A1E254E"
)

var (
	id      seqid.ID
	ledgers = avl.NewTree() // denom -> *grc20.PrivateLedger
)

func init(cur realm) {
	ta, la := grc20.NewToken("Voucher A", "VCHA", 6, id.Next(), cur)
	grc20reg.Register(cross(cur), ta, "")
	ledgers.Set(DenomA, la)

	tb, lb := grc20.NewToken("Voucher B", "VCHB", 6, id.Next(), cur)
	grc20reg.Register(cross(cur), tb, "")
	ledgers.Set(DenomB, lb)
}

// Approve is grc20factory.Approve with the instance looked up by denom
// instead of by symbol.
func Approve(cur realm, denom string, spender address, amount int64) {
	v := ledgers.Get(denom)
	if v == nil {
		panic("zdenomv: unknown denom " + denom)
	}
	caller := cur.Previous().Address()
	teller := v.(*grc20.PrivateLedger).ImpersonateTeller(caller)
	if err := teller.Approve(0, cur, spender, amount); err != nil {
		panic(err)
	}
}
TXTAR
go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20routes_denom_keyed' -p 1 -parallel 1
rm gno.land/pkg/integration/testdata/grc20routes_denom_keyed.txtar
```

The proof is honest and the realm really is keyed, and the route still comes back carrying the registry symbol rather than the denom the two approvals passed.

```
--- FAIL: TestTestdata (0.07s)
    --- FAIL: TestTestdata/grc20routes_denom_keyed (5.85s)
        # The four documented proof calls, run honestly: the prover passes the realm's
        # real leading argument, so both allowances land through the realm's own entry
        # point. Nothing is forged here. (0.681s)
        # The route a wallet now reads. (0.016s)
        > stdout '\\"prefix\\":\[\\"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2\\"\],\\"source\\":\\"proven\\"'
        FAIL: testdata/grc20routes_denom_keyed.txtar:50: no match for `\\"prefix\\":\[\\"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2\\"\],\\"source\\":\\"proven\\"` found in stdout
FAIL
FAIL	github.com/gnolang/gno/gno.land/pkg/integration	6.003s
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:416 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L416) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L416)
Nothing removes an entry from `provenKeyed`, so a realm marked keyed by a proof that does not describe its call shape stays marked for every token it hosts now or later. The only correction is [`Register`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L295), which [replaces one token's route and leaves its siblings proven](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L272-L294) and is unreachable to a realm deployed before this one, the population the proven path serves.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:443 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L443) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L443)
[`fqname.Parse`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/p/nt/fqname/v0/fqname.gno#L33) splits at the first dot after the last slash, and [`isValidSubpathSegment`](https://github.com/gnolang/gno/blob/3aa06ef75/gnovm/stdlibs/chain/address.gno#L74) allows a dot inside a subpath. A token registered under the sub identity `a.b` publishes `b.SUBTK` as its prefix, so the wallet's `Approve` carries a leading argument [the declared path refuses outright](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L590-L593).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
cat > examples/gno.land/r/nt/grc20routes/v0/zz_dot_subpath_test.gno <<'GNO'
package grc20routes

import (
	"strings"
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestProvenPrefixFromDotSubpath(cur realm, t *testing.T) {
	prover := testutils.TestAddress("subdot")

	// "a.b" is a legal subpath, so a realm may register tokens under it.
	sub := cur.Sub("a.b")
	keyA := mint(0, sub, "SUBTK") // keys in grc20reg as "<selfPath>#a.b.SUBTK"
	keyB := mint(0, cur, "SUBB")  // same host realm, primary identity

	urequire.Equal(t, selfPath+"#a.b.SUBTK", keyA)
	urequire.Equal(t, selfPath, PkgPath(keyA), "the MsgCall destination is the host realm")

	// The proven path emits a prefix the declared path refuses outright.
	uassert.AbortsWithMessage(t, cur, "grc20routes: invalid symbol character: .",
		func() { Register(cross(cur), "b.SUBTK", Keyed("b.SUBTK")) })

	testing.SetOriginCaller(prover)
	BeginKeyedProof(cross(cur), keyA, keyB, 777, 888)
	approveAs("SUBTK", prover, 777)
	approveAs("SUBB", prover, 888)
	FinishKeyedProof(cross(cur), keyA, keyB)

	r, source := Get(keyA)
	urequire.Equal(t, SourceProven, source)
	urequire.Equal(t, 1, len(r.Prefix))

	uassert.Equal(t, "SUBTK", r.Prefix[0], "the prefix is the symbol")
	uassert.True(t, strings.Contains(JSON(keyA), `"prefix":["SUBTK"]`),
		"JSON publishes the symbol")
}
GNO
(cd examples && gno test -v -run TestProvenPrefixFromDotSubpath ./gno.land/r/nt/grc20routes/v0)
rm examples/gno.land/r/nt/grc20routes/v0/zz_dot_subpath_test.gno
```

The two assertions on the published prefix fail: the subpath tail rides in front of the symbol, in the route and in the JSON a wallet reads.

```
=== RUN   TestProvenPrefixFromDotSubpath
uassert.Equal: strings are different
	Diff: [+b.]SUBTK - the prefix is the symbol
should be true - JSON publishes the symbol
--- FAIL: TestProvenPrefixFromDotSubpath (0.04s)
failed: "TestProvenPrefixFromDotSubpath"
FAIL    ./gno.land/r/nt/grc20routes/v0 	3.26s
FAIL: 0 build errors, 1 test errors
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:500 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L500) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L500)
`Func` drops the route's [`Prefix`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L502) and returns the entry point name alone. A proven-keyed token's realm entry point is [`Approve(cur, symbol, spender, amount)`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L88), so a `MsgCall` built from that one value is one argument short.

<details><summary>the two answers the shipped fixture asks for</summary>

The same token, queried twice in [`grc20routes_keyed_proof.txtar`](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar#L62-L69) after the realm is proven keyed:

```
# (c) After: both proven tokens, keyed by their own symbol.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/demo/defi/grc20factory.BAR\")"
stdout '\\"prefix\\":\[\\"BAR\\"\],\\"source\\":\\"proven\\"'

# The single-operation getter, which is all a wallet building one message needs.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.Func(\"gno.land/r/demo/defi/grc20factory.BAR\",\"approve\")"
stdout 'Approve'
```

The `BAR` that `JSON` reports as the prefix is the `symbol` parameter of the real entry point, and the only working call in the same fixture passes it: [`-func Approve -args BAR -args <spender> -args 222`](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar#L53). The exported surface has no `Prefix(tokenKey)` getter to pair with `Func`, so the complete answer is only reachable through [`Get`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L436) or [`JSON`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L466).
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:519 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L519) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L519)
`mustToken` drops the `#subpath` and [`Get`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L443-L444) never restores it, so a sub identity's token and its host's token of the same symbol publish byte-identical routes. A wallet following the sub token's route approves the host's token instead.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
mkdir -p examples/gno.land/r/demo/zone5
printf 'module = "gno.land/r/demo/zone5"\n\ngno = "0.9"\n' > examples/gno.land/r/demo/zone5/gnomod.toml
cat > examples/gno.land/r/demo/zone5/main_filetest.gno <<'GNO'
// PKGPATH: gno.land/r/demo/zone5
package zone5

import (
	"chain"

	"gno.land/p/nt/grc20/v0"
	"gno.land/r/nt/grc20reg/v0"
	"gno.land/r/nt/grc20routes/v0"
)

var (
	ledgers = map[string]*grc20.PrivateLedger{}

	fooKey, barKey, subBarKey string
)

func init(cur realm) {
	foo, fl := grc20.NewToken("Foo", "FOO", 4, 0, cur)
	ledgers["FOO"] = fl
	fooKey = grc20reg.Register(cross(cur), foo, "")

	bar, bl := grc20.NewToken("Bar", "BAR", 4, 1, cur)
	ledgers["BAR"] = bl
	barKey = grc20reg.Register(cross(cur), bar, "")
}

// This realm's honestly keyed entry point: the symbol first, then GRC20's own
// arguments. Exactly the shape Keyed() describes, so the proof below is real.
func Approve(cur realm, symbol string, spender address, amount int64) {
	l := ledgers[symbol]
	if l == nil {
		panic("zone5: unknown symbol " + symbol)
	}
	if err := l.CallerTeller().Approve(0, cur, spender, amount); err != nil {
		panic(err)
	}
}

func main(cur realm) {
	probe := grc20routes.ProbeAddress()
	owner := cur.Address()
	spender := chain.PackageAddress("gno.land/r/demo/zspender5")

	grc20routes.BeginKeyedProof(cross(cur), fooKey, barKey, 11, 22)
	Approve(cross(cur), "FOO", probe, 11)
	Approve(cross(cur), "BAR", probe, 22)
	grc20routes.FinishKeyedProof(cross(cur), fooKey, barKey)

	// The same realm mints a SECOND BAR under its own "treasury" sub identity.
	// grc20reg keys on the raw sub path, so the key differs and its
	// one-token-per-realm-and-symbol guard never fires.
	sub := cur.Sub("treasury")
	tbar, _ := grc20.NewToken("Treasury Bar", "BAR", 4, 2, sub)
	subBarKey = grc20reg.Register(cross(sub), tbar, "")

	println("host BAR key    :", barKey)
	println("treasury BAR key:", subBarKey)
	println("route(host BAR)    :", grc20routes.JSON(barKey))
	println("route(treasury BAR):", grc20routes.JSON(subBarKey))

	// A wallet reads the treasury BAR route and follows it verbatim: its
	// pkg_path plus its prefix spell zone5.Approve("BAR", spender, 500).
	Approve(cross(cur), "BAR", spender, 500)

	println("allowance on host BAR    :", grc20reg.MustGet(barKey).Allowance(owner, spender))
	println("allowance on treasury BAR:", grc20reg.MustGet(subBarKey).Allowance(owner, spender))
}
GNO
(cd examples && gno test -v ./gno.land/r/demo/zone5)
rm -rf examples/gno.land/r/demo/zone5
```

The two tokens hand out the same route, and the allowance the treasury route asked for landed on the host's BAR.

```
host BAR key    : gno.land/r/demo/zone5.BAR
treasury BAR key: gno.land/r/demo/zone5#treasury.BAR
route(host BAR)    : {"token_key":"gno.land/r/demo/zone5.BAR","pkg_path":"gno.land/r/demo/zone5","prefix":["BAR"],"source":"proven",…}
route(treasury BAR): {"token_key":"gno.land/r/demo/zone5#treasury.BAR","pkg_path":"gno.land/r/demo/zone5","prefix":["BAR"],"source":"proven",…}
allowance on host BAR    : 500
allowance on treasury BAR: 0
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:737 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L737) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L737)
[`md.EscapeText`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/p/moul/md/v0/md.gno#L414) runs over the prefix argument on its way into a code span, and [`inlineEscapeSet` escapes](https://github.com/gnolang/gno/blob/3aa06ef75/gnovm/stdlibs/chain/markdown/markdown.go#L86) both `_` and `-`, which [`validateSymbol` permits](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L591) in a symbol. The page prints `Approve(cur, A\_B, spender, amount)` for a user to copy out.

```suggestion
		args += ", " + p
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno

cat > examples/gno.land/r/nt/grc20routes/v0/escape_test.gno <<'EOF'
package grc20routes

import (
	"strings"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

func TestRenderedPrefixArgumentIsCopyPasteable(cur realm, t *testing.T) {
	key := mint(0, cur, "A_B")
	Register(cross(cur), "A_B", Keyed("A_B"))

	for _, line := range strings.Split(Render(key), "\n") {
		if strings.HasPrefix(line, "- approve:") {
			println(line)
		}
	}
	uassert.True(t, strings.Contains(Render(key), "Approve(cur, A_B, spender, amount)"),
		"the rendered call must carry the literal token symbol")
}
EOF

go build -o /tmp/gno-6187 ./gnovm/cmd/gno
(cd examples && /tmp/gno-6187 test -v -run TestRenderedPrefixArgumentIsCopyPasteable ./gno.land/r/nt/grc20routes/v0 2>&1 | grep -v '^--- GAS')
rm examples/gno.land/r/nt/grc20routes/v0/escape_test.gno /tmp/gno-6187
```

The printed line is the one a reader copies the argument out of, and the backslash is inside the backticks:

```
- approve: `Approve(cur, A\_B, spender, amount)`
=== RUN   TestRenderedPrefixArgumentIsCopyPasteable
should be true - the rendered call must carry the literal token symbol
--- FAIL: TestRenderedPrefixArgumentIsCopyPasteable (0.02s)
FAIL    ./gno.land/r/nt/grc20routes/v0 	3.11s
```

The heading at [`:703`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L703) escapes correctly, being a plain inline slot; [`JSON`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L475) emits the same value unescaped, so the two consumer surfaces disagree on the argument string. Every proven-path token is affected the same way, since `Get` returns [`Keyed(symbol)`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L444) verbatim.
</details>

## SKIP examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:192 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L192) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L192)
Missing test: no test exercises `maxIdentLen`, `maxOpLen`, `maxSymbolLen`, `maxEntries` or `maxPrefixArg`, so all five can be raised by three orders of magnitude with the suite green.

Skipped: the same edit closes it as the section anchored on the `maxEntries` guard.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:295 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L295) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L295)
Missing test: no test calls `Register` from another realm. [`sortEntries`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L310) reorders the caller's own `Route.Funcs` in place and [`declared.Set`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L311) stores that same backing array, so a realm that reads back the route it built sees a different order.

<details><summary>test cases</summary>

A second realm builds a route the way the package doc shows, with `Canonical().With(...)`, which lands `Funcs` out of `Op` order, keeps it in a package variable and registers it.

```go
// examples/gno.land/r/demo/zsort/zsort.gno
package zsort

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/p/nt/seqid/v0"

	"gno.land/r/nt/grc20reg/v0"
	"gno.land/r/nt/grc20routes/v0"
)

var (
	tok *grc20.Token
	led *grc20.PrivateLedger
	id  seqid.ID

	myRoute grc20routes.Route
)

func init(cur realm) {
	tok, led = grc20.NewToken("Sort", "SRT", 6, id.Next(), cur)
	grc20reg.Register(cross(cur), tok, "")
}

func DeclareOutOfOrder(cur realm) string {
	myRoute = grc20routes.Canonical().With("deposit", "Deposit").With("withdraw", "Withdraw")
	return grc20routes.Register(cross(cur), "SRT", myRoute)
}

func MyRouteOps() string {
	s := ""
	for _, e := range myRoute.Funcs {
		s += e.Op + ","
	}
	return s
}
```

```go
// examples/gno.land/r/demo/zsort/zsort_test.gno
package zsort

import (
	"testing"

	"gno.land/r/nt/grc20routes/v0"
)

func TestRegisterLeavesTheCallersRouteAlone(cur realm, t *testing.T) {
	built := grc20routes.Canonical().With("deposit", "Deposit").With("withdraw", "Withdraw")
	before := ""
	for _, e := range built.Funcs {
		before += e.Op + ","
	}
	DeclareOutOfOrder(cross(cur))
	if after := MyRouteOps(); after != before {
		t.Fatalf("Register reordered the caller's slice: %s -> %s", before, after)
	}
}
```

At this head the test fails on `approve,transfer,transfer_from,deposit,withdraw,` becoming `approve,deposit,transfer,transfer_from,withdraw,`.
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:317 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L317) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L317)
Missing test: this `register` event and [`proven`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L418) are the realm's only push-side output, and no package test or txtar step asserts either, so deleting both `chain.Emit` calls leaves the suite and [`grc20routes_keyed_proof.txtar`](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar) green. [`grc20reg.Register`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno#L50) emits its own event typed `register` in the same transaction, so an indexer filtering on the type name alone receives both and has to discriminate on `pkg_path`.

<details><summary>test cases</summary>

A filetest pins both events, including the `pkg_path` that separates them from grc20reg's. It passes at this head and reddens when either `chain.Emit` is deleted.

```go
// PKGPATH: gno.land/r/demo/routeevents
package routeevents

import (
	"gno.land/p/nt/grc20/v0"

	"gno.land/r/nt/grc20reg/v0"
	routes "gno.land/r/nt/grc20routes/v0"
)

func main(cur realm) {
	// Two tokens on one realm: the declared path needs one registered in
	// grc20reg, the keyed proof needs two at the same realm.
	tokA, ledA := grc20.NewToken("Alpha", "ALFA", 6, 0, cur)
	tokB, ledB := grc20.NewToken("Beta", "BETA", 6, 1, cur)
	keyA := grc20reg.Register(cross(cur), tokA, "")
	keyB := grc20reg.Register(cross(cur), tokB, "")

	// DECLARED path, emits "register". "deposit" is there so ops is not the
	// canonical three, which a byte-for-byte golden would otherwise hide.
	routes.Register(cross(cur), "ALFA", routes.Canonical().With("deposit", "Deposit"))

	// PROVEN path, four calls, emits "proven" on the last.
	prover := cur.Address()
	routes.BeginKeyedProof(cross(cur), keyA, keyB, 111, 222)
	if err := ledA.Approve(prover, routes.ProbeAddress(), 111); err != nil {
		panic(err)
	}
	if err := ledB.Approve(prover, routes.ProbeAddress(), 222); err != nil {
		panic(err)
	}
	routes.FinishKeyedProof(cross(cur), keyA, keyB)
	println("done")
}

// Output:
// done

// Events:
// [
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/routeevents.ALFA.0000000"
//       },
//       {
//         "key": "name",
//         "value": "Alpha"
//       },
//       {
//         "key": "symbol",
//         "value": "ALFA"
//       },
//       {
//         "key": "decimals",
//         "value": "6"
//       }
//     ],
//     "pkg_path": "gno.land/p/nt/grc20/v0"
//   },
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/routeevents.BETA.0000001"
//       },
//       {
//         "key": "name",
//         "value": "Beta"
//       },
//       {
//         "key": "symbol",
//         "value": "BETA"
//       },
//       {
//         "key": "decimals",
//         "value": "6"
//       }
//     ],
//     "pkg_path": "gno.land/p/nt/grc20/v0"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/routeevents.ALFA"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/routeevents"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "ALFA"
//       }
//     ],
//     "pkg_path": "gno.land/r/nt/grc20reg/v0"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/routeevents.BETA"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/routeevents"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "BETA"
//       }
//     ],
//     "pkg_path": "gno.land/r/nt/grc20reg/v0"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_key",
//         "value": "gno.land/r/demo/routeevents.ALFA"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/routeevents"
//       },
//       {
//         "key": "ops",
//         "value": "approve,deposit,transfer,transfer_from"
//       },
//       {
//         "key": "prefix",
//         "value": ""
//       }
//     ],
//     "pkg_path": "gno.land/r/nt/grc20routes/v0"
//   },
//   {
//     "type": "Approval",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/routeevents.ALFA.0000000"
//       },
//       {
//         "key": "owner",
//         "value": "g13s9pqxx2nza4v8ngw46veuss0sq6xdtrzrtx7f"
//       },
//       {
//         "key": "spender",
//         "value": "g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep"
//       },
//       {
//         "key": "value",
//         "value": "111"
//       }
//     ],
//     "pkg_path": "gno.land/p/nt/grc20/v0"
//   },
//   {
//     "type": "Approval",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/routeevents.BETA.0000001"
//       },
//       {
//         "key": "owner",
//         "value": "g13s9pqxx2nza4v8ngw46veuss0sq6xdtrzrtx7f"
//       },
//       {
//         "key": "spender",
//         "value": "g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep"
//       },
//       {
//         "key": "value",
//         "value": "222"
//       }
//     ],
//     "pkg_path": "gno.land/p/nt/grc20/v0"
//   },
//   {
//     "type": "proven",
//     "attrs": [
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/routeevents"
//       },
//       {
//         "key": "prover",
//         "value": "g13s9pqxx2nza4v8ngw46veuss0sq6xdtrzrtx7f"
//       }
//     ],
//     "pkg_path": "gno.land/r/nt/grc20routes/v0"
//   }
// ]
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:401-403 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L401-L403) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L401)
Missing test: every `FinishKeyedProof` in the suite and in [the keyed-proof fixture](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar) passes the keys its `BeginKeyedProof` opened. Deleting this pair check leaves both green and turns `tokenKeyB` into an argument the function reads nowhere.

<details><summary>test cases</summary>

The pending entry is keyed by prover and realm path, not by the token pair, so any registered token of that realm reaches it.

```go
func TestFinishKeyedProofRejectsADifferentPair(cur realm, t *testing.T) {
	prover := testutils.TestAddress("pair")
	keyA := mint(0, cur, "FPA")
	keyB := mint(0, cur, "FPB")
	keyC := mint(0, cur, "FPC") // a third token on the same realm, never in the proof
	testing.SetOriginCaller(prover)

	BeginKeyedProof(cross(cur), keyA, keyB, 1111, 2222)
	approveAs("FPA", prover, 1111)
	approveAs("FPB", prover, 2222)

	uassert.AbortsWithMessage(t, cur,
		"grc20routes: tokens do not match the proof in progress",
		func() { FinishKeyedProof(cross(cur), keyA, keyC) })
	uassert.AbortsWithMessage(t, cur,
		"grc20routes: tokens do not match the proof in progress",
		func() { FinishKeyedProof(cross(cur), keyC, keyB) })

	// The pair that was opened still finishes, so the two aborts above are the
	// guard and not a proof this test broke.
	FinishKeyedProof(cross(cur), keyA, keyB)
	uassert.True(t, IsKeyedRealm(selfPath))
}
```
</details>

## SKIP examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:408 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L408) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L408)
Missing test: no test reaches the token-A half of this check, so deleting it leaves the package suite and [`grc20routes_keyed_proof.txtar`](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar) green and a realm that moved only its second token comes out keyed.

Skipped: grc20routes_test.gno:211 carries the same gap and the same mirror test.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:415 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L415) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L415)
Missing test: no test asserts that a committed proof clears the [`pending`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L377) entry it consumed, so deleting `pending.Remove` leaves the whole suite green.

<details><summary>test cases</summary>

```go
func TestFinishKeyedProofClearsItsPendingEntry(cur realm, t *testing.T) {
	prover := testutils.TestAddress("clears")
	keyA := mint(0, cur, "CLA")
	keyB := mint(0, cur, "CLB")
	testing.SetOriginCaller(prover)

	BeginKeyedProof(cross(cur), keyA, keyB, 101, 202)
	pk := pendingKey(prover, selfPath)
	urequire.True(t, pending.Has(pk), "BeginKeyedProof opens exactly one entry")

	approveAs("CLA", prover, 101)
	approveAs("CLB", prover, 202)
	FinishKeyedProof(cross(cur), keyA, keyB)

	uassert.False(t, pending.Has(pk),
		"a committed proof leaves nothing behind in the pending tree")
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:540-542 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L540-L542) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L540)
Missing test: [`TestValidationRejectsUnsafeValues`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L321) has eleven cases and none exceeds a length cap. Deleting this `maxEntries` guard, or the ones on [`maxPrefixArg`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L562-L564), [`maxOpLen`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L604-L606) or [`maxIdentLen`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L616-L618), leaves the package suite green.

<details><summary>test cases</summary>

One row per cap, sized one byte over its constant, plus the at-cap acceptance case so the bounds stay inclusive. The [`maxSymbolLen`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L587-L589) row pins a message rather than a bound, since the `grc20reg` lookup in `Register` already refuses any symbol over `grc20.MaxSymbolLen`.

```go
// entries builds n distinct valid entries, so only the maxEntries guard can fire.
func entries(n int) []Entry {
	es := make([]Entry, n)
	for i := 0; i < n; i++ {
		es[i] = Entry{"op_" + strings.Repeat("x", i), "Fn" + strings.Repeat("X", i)}
	}
	return es
}

func TestValidationRejectsOverlongValues(cur realm, t *testing.T) {
	mint(0, cur, "CAPS")

	longOp := strings.Repeat("a", maxOpLen+1)
	longFunc := "A" + strings.Repeat("a", maxIdentLen) // exported, so only length can fail
	longSymbol := strings.Repeat("A", maxSymbolLen+1)
	longPrefix := strings.Repeat("a", maxPrefixArg+1)

	cases := []struct {
		name   string
		symbol string
		route  Route
		msg    string
	}{
		{
			"one entry over maxEntries", "CAPS",
			Route{Funcs: entries(maxEntries + 1)},
			"grc20routes: too many entry points",
		},
		{
			"operation name over maxOpLen", "CAPS",
			Route{Funcs: []Entry{{longOp, "Approve"}}},
			"grc20routes: operation name too long: " + longOp,
		},
		{
			"function name over maxIdentLen", "CAPS",
			Route{Funcs: []Entry{{OpApprove, longFunc}}},
			"grc20routes: approve name too long",
		},
		{
			"symbol over maxSymbolLen", longSymbol,
			Route{Funcs: []Entry{{OpApprove, "Approve"}}},
			"grc20routes: symbol too long",
		},
		{
			"prefix argument over maxPrefixArg", "CAPS",
			Route{Funcs: []Entry{{OpApprove, "Approve"}}, Prefix: []string{longPrefix}},
			"grc20routes: prefix argument too long",
		},
	}

	for _, tc := range cases {
		r, sym := tc.route, tc.symbol
		uassert.AbortsWithMessage(t, cur, tc.msg,
			func() { Register(cross(cur), sym, r) }, tc.name)
	}
}

func TestValidationAcceptsValuesAtTheCap(cur realm, t *testing.T) {
	mint(0, cur, "ATCAP")

	es := entries(maxEntries)
	es[0] = Entry{strings.Repeat("a", maxOpLen), "A" + strings.Repeat("a", maxIdentLen-1)}
	r := Route{Funcs: es, Prefix: []string{strings.Repeat("a", maxPrefixArg)}}

	uassert.NotAborts(t, cur, func() { Register(cross(cur), "ATCAP", r) },
		"a route exactly on every cap must register")
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:652 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L652) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L652)
Missing test: `Render` through [`renderEntry`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L734-L739) is 89 lines and the realm's only human-facing surface, and neither the package suite nor [the keyed-proof fixture](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar) calls `Render` once.

<details><summary>test cases</summary>

The first case, which fails at this head by aborting the whole call rather than printing a page:

```go
// examples/gno.land/r/nt/grc20routes/v0/render_filetest.gno
package main

import (
	"gno.land/r/nt/grc20routes/v0"
)

func main() {
	println(grc20routes.Render("gno.land/r/demo/nope.NOPE"))
}
```

```
panic: grc20routes: unknown token: gno.land/r/demo/nope.NOPE
mustToken<VPBlock(3,38)>(tokenKey<VPBlock(1,0)>)
    gno.land/r/nt/grc20routes/v0/grc20routes.gno:516
Get<VPBlock(3,33)>(tokenKey<VPBlock(1,0)>)
    gno.land/r/nt/grc20routes/v0/grc20routes.gno:437
renderToken<VPBlock(4,47)>(path<VPBlock(2,0)>)
    gno.land/r/nt/grc20routes/v0/grc20routes.gno:700
ref(gno.land/r/nt/grc20routes/v0).Render(gno.land/r/demo/nope.NOPE)
    gno.land/r/nt/grc20routes/v0/grc20routes.gno:654
```

The remaining cases: both trees empty, one proof and one declaration, a token page on each of the three sources, and a declared route carrying a prefix and a non-canonical operation, which is where `opTail` and `renderEntry` run.
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:136 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L136) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L136)
Missing test: every `With` in the suite starts from the prefix-less [`Canonical()`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L246-L251), so nothing pins the [`Prefix: r.Prefix` carry-over](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L267) the keyed builder rests on.

<details><summary>test cases</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
# save the test below as examples/gno.land/r/nt/grc20routes/v0/with_prefix_test.gno
(cd examples && gno test ./gno.land/r/nt/grc20routes/v0)

# the oracle: the suite is green without this file once the carry-over is dropped
sed -i '267s/.*/\tout := Route{}/' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
(cd examples && gno test ./gno.land/r/nt/grc20routes/v0)
git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

```go
// A keyed realm that exposes more than the GRC20 three builds its route with
// Keyed then With. Every With must carry the leading symbol argument over.
func TestWithKeepsThePrefix(cur realm, t *testing.T) {
	r := Keyed("WPX").With("deposit", "Deposit").With("withdraw", "Withdraw")

	urequire.Equal(t, 1, len(r.Prefix), "With drops the prefix Keyed set")
	uassert.Equal(t, "WPX", r.Prefix[0])
	uassert.Equal(t, "Deposit", r.Func("deposit"))
	uassert.Equal(t, "Approve", r.Func(OpApprove), "the standard three survive With")

	// Through the store, which is the copy a wallet actually reads.
	key := mint(0, cur, "WPX")
	Register(cross(cur), "WPX", r)

	got, _ := Get(key)
	urequire.Equal(t, 1, len(got.Prefix), "the stored route lost the prefix")
	uassert.Equal(t, "WPX", got.Prefix[0])
	uassert.True(t, strings.Contains(JSON(key), `"prefix":["WPX"]`),
		"a wallet builds the leading approve argument from this prefix")
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:174 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L174) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L174)
Missing test: this asserts `hostOf` against two literals, so no `<host>#<sub>.<SYM>` key reaches `PkgPath`, `JSON` or `Get`. Drop the [strip](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L519) and a wallet reads `gno.land/r/nt/grc20routes/v0#admin` as its MsgCall destination, the suite and the fixture still green.

<details><summary>test cases</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
# save the test below as examples/gno.land/r/nt/grc20routes/v0/subrealm_route_test.gno
(cd examples && gno test -v -run 'TestSubRealmTokenRoutesToCallableHost' ./gno.land/r/nt/grc20routes/v0)

# the oracle: the shipped suite and grc20routes_keyed_proof.txtar both survive this
sed -i 's|^\treturn hostOf(pkgPath)$|\treturn pkgPath|' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
(cd examples && gno test -v -run 'TestSubRealmTokenRoutesToCallableHost' ./gno.land/r/nt/grc20routes/v0)
git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

```go
func TestSubRealmTokenRoutesToCallableHost(cur realm, t *testing.T) {
	// grc20reg keys on the raw cur.Previous().PkgPath(), so a token minted
	// and registered while the realm runs under a sub identity keys with
	// the "#sub" attached.
	sub := cur.Sub("admin")
	tok, led := grc20.NewToken("Test SUBH", "SUBH", 6, nextID.Next(), sub)
	ledgers["SUBH"] = led
	key := grc20reg.Register(cross(sub), tok, "")
	urequire.Equal(t, selfPath+"#admin.SUBH", key)

	// "<host>#<sub>" is not a MsgCall destination; the host is.
	uassert.Equal(t, selfPath, PkgPath(key))
	uassert.True(t, strings.Contains(JSON(key), `"pkg_path":"`+selfPath+`"`),
		"JSON pkg_path must name the callable host, got "+JSON(key))

	// provenKeyed is keyed per host, so one proof run by the host realm has
	// to cover the token it registered under a sub identity.
	prover := testutils.TestAddress("subprover")
	keyA := mint(0, cur, "SUBA")
	keyB := mint(0, cur, "SUBB")
	testing.SetOriginCaller(prover)
	BeginKeyedProof(cross(cur), keyA, keyB, 901, 902)
	approveAs("SUBA", prover, 901)
	approveAs("SUBB", prover, 902)
	FinishKeyedProof(cross(cur), keyA, keyB)

	_, source := Get(key)
	uassert.Equal(t, SourceProven, source,
		"a proof on the host realm must cover its sub-registered tokens")
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:195 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L195) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L195)
Missing test: every keyed-proof test proves the realm it runs inside, which declares no `Approve` of any arity, so deleting the [same-realm guard](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L361-L363) or loosening [the exact-nonce comparison](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L408-L413) to any nonzero allowance keeps the suite green. The case nothing covers is a second realm hosting two registered tokens, exposing no keyed `Approve`, that must still read `source=convention`, which needs a txtar beside [`grc20routes_keyed_proof.txtar`](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
go build -o /tmp/gno-6187 ./gnovm/cmd/gno

PKG=gno.land/r/nt/grc20routes/v0
WORK="$(mktemp -d)"
cp -r examples "$WORK/examples"
SRC="$WORK/examples/$PKG/grc20routes.gno"
cp "$SRC" "$WORK/pristine.gno"

run() {
  local out
  out="$(cd "$WORK/examples" && /tmp/gno-6187 test "./$PKG" 2>&1)"
  printf '%-42s exit=%d  %s\n' "$1" "$?" "$(printf '%s' "$out" | tail -1)"
}

printf 'func Approve on the realm every test proves: '
grep -c '^func Approve' "$SRC"
run "baseline (unmutated)"

python3 - "$SRC" <<'PY'
import sys
p = sys.argv[1]
s = open(p).read()
old = '''\tif pathA != pathB {
\t\tpanic("grc20routes: both tokens must live in the same realm")
\t}
'''
assert old in s, "guard not found"
open(p, "w").write(s.replace(old, '\t_ = pathB // MUTATION 1\n'))
PY
run "mutation 1: same-realm guard deleted"
cp "$WORK/pristine.gno" "$SRC"

python3 - "$SRC" <<'PY'
import sys
p = sys.argv[1]
s = open(p).read()
for key in ("A", "B"):
	old = '\tif allowanceOf(p.key%s, prover) != p.nonce%s {' % (key, key)
	assert old in s, old
	s = s.replace(old, '\tif allowanceOf(p.key%s, prover) == 0 { // MUTATION 2' % key)
open(p, "w").write(s)
PY
run "mutation 2: exact nonce no longer required"
rm -rf "$WORK" /tmp/gno-6187
```

Neither mutation reddens anything, and the realm the suite marks keyed exposes no `Approve` at all:

```
func Approve on the realm every test proves: 0
baseline (unmutated)                       exit=0  ok  ./gno.land/r/nt/grc20routes/v0  10.69s
mutation 1: same-realm guard deleted       exit=0  ok  ./gno.land/r/nt/grc20routes/v0  20.92s
mutation 2: exact nonce no longer required exit=0  ok  ./gno.land/r/nt/grc20routes/v0  59.38s
```

Mutation 2 survives because the one rejection test, [`TestKeyedProofFailsWhenOnlyOneTokenMoves`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L211), leaves the second allowance at 0, where the weakened comparison still aborts.
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:211 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L211) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L211)
Missing test: `TestKeyedProofFailsWhenOnlyOneTokenMoves` moves token A and lands on [the token-B branch](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L411-L413), and every other `FinishKeyedProof` in the suite and in [the fixture](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar#L50-L56) has both allowances correct. Deleting [the token-A branch](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L408-L410) reddens nothing, and a realm whose second token alone moved is marked keyed.

<details><summary>test cases</summary>

The mirror, which passes at this head and fails once lines 408-410 are dropped:

```go
// The mirror of TestKeyedProofFailsWhenOnlyOneTokenMoves, which moves token A
// and lands on the keyB branch. Without this one, an unkeyed realm that moved
// token B alone would be marked keyed.
func TestKeyedProofFailsWhenOnlyTokenBMoves(cur realm, t *testing.T) {
	prover := testutils.TestAddress("halfb")
	keyA := mint(0, cur, "HBA")
	keyB := mint(0, cur, "HBB")
	testing.SetOriginCaller(prover)

	BeginKeyedProof(cross(cur), keyA, keyB, 333, 444)
	approveAs("HBB", prover, 444) // only token B: the side the suite never covers

	uassert.AbortsWithMessage(t, cur,
		"grc20routes: "+keyA+" does not carry its nonce",
		func() { FinishKeyedProof(cross(cur), keyA, keyB) })
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:226 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L226) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L226)
Missing test: this parks the stale nonce on token A alone, so the token-B half of the [already-parked guard](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L373) is unpinned. One ordinary `Approve` then keys the realm for a prover who parked nonceB in advance.

<details><summary>test cases</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
# save the test below as examples/gno.land/r/nt/grc20routes/v0/keyed_proof_parked_b_test.gno
(cd examples && gno test ./gno.land/r/nt/grc20routes/v0)

# the oracle: the suite is green without this file once the disjunct is dropped
sed -i '373s/ || allowanceOf(tokenKeyB, prover) == nonceB//' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
(cd examples && gno test ./gno.land/r/nt/grc20routes/v0)
git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

```go
// The mirror of TestKeyedProofRejectsPreExistingAllowance, which parks token A
// and lands on the tokenKeyA disjunct.
func TestKeyedProofRejectsPreExistingAllowanceOnTokenB(cur realm, t *testing.T) {
	prover := testutils.TestAddress("staleb")
	keyA := mint(0, cur, "SBA")
	keyB := mint(0, cur, "SBB")
	approveAs("SBB", prover, 666) // parked on token B alone, before the proof opens
	testing.SetOriginCaller(prover)

	uassert.AbortsWithMessage(t, cur,
		"grc20routes: nonce already parked on the probe; pick another",
		func() { BeginKeyedProof(cross(cur), keyA, keyB, 555, 666) })
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:238 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L238) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L238)
Missing test: no case here covers the cross-realm rejection, and every token in the suite registers under `selfPath`. Neutralize [`pathA != pathB`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L361) and any account pairs `wugnot.wugnot` with `grc20factory.FOO` to read wugnot back as proven.

<details><summary>test cases</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
# save the fixture below as
#   gno.land/pkg/integration/testdata/grc20routes_cross_realm_proof.txtar
go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20routes_cross_realm_proof' -p 1 -parallel 1

# the oracle: with the guard neutralized to `_ = pathB`, this fixture is the only red
```

```txtar
# BeginKeyedProof's cross-realm guard has no test at this head: the package suite
# mints every token under one realm and grc20routes_keyed_proof.txtar never pairs
# two realms. Green at this head, red with the guard neutralized to `_ = pathB`.

loadpkg gno.land/r/demo/defi/grc20factory
loadpkg gno.land/r/gnoland/wugnot
loadpkg gno.land/r/nt/grc20routes/v0

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid tendermint_test test1
stdout 'OK!'

# One token on the factory realm, which really is keyed by symbol.
gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20factory -func New -args Foo -args FOO -args 6 -args 1000000 -args 0 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

# The victim: wugnot hosts exactly one token and its Approve(cur, spender, amount)
# takes no leading symbol, so the convention is the right route for it.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/gnoland/wugnot.wugnot\")"
stdout '\\"prefix\\":\[\],\\"source\\":\\"convention\\"'

# The guard. Pairing the victim's only token with a token of an unrelated realm is
# what a prover needs to reach two distinct registry keys without a keyed Approve.
! gnokey maketx call -pkgpath gno.land/r/nt/grc20routes/v0 -func BeginKeyedProof -args gno.land/r/gnoland/wugnot.wugnot -args gno.land/r/demo/defi/grc20factory.FOO -args 111 -args 222 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stderr 'both tokens must live in the same realm'

# Everything else the pair needs is an ordinary call alice may already make: both
# allowances land on the probe address (grc20routes' own realm address) unaided.
gnokey maketx call -pkgpath gno.land/r/gnoland/wugnot -func Approve -args g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep -args 111 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20factory -func Approve -args FOO -args g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep -args 222 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

# So only the rejected open stands between that state and provenKeyed[wugnot].
! gnokey maketx call -pkgpath gno.land/r/nt/grc20routes/v0 -func FinishKeyedProof -args gno.land/r/gnoland/wugnot.wugnot -args gno.land/r/demo/defi/grc20factory.FOO -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stderr 'no proof in progress for gno.land/r/gnoland/wugnot'

# wugnot still routes by the convention. Were it proven, a wallet would build
# Approve("wugnot", spender, amount) against a two-argument entry point.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/gnoland/wugnot.wugnot\")"
stdout '\\"prefix\\":\[\],\\"source\\":\\"convention\\"'
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:248-249 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L248-L249) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L248)
Missing test: this case passes `(0, 2)`, so only the `nonceA` half of the [positivity guard](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L367) is reached. Dropping `|| nonceB <= 0` admits a negative nonceB, since zero still falls to the already-parked guard.

<details><summary>test cases</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
# save the test below as examples/gno.land/r/nt/grc20routes/v0/keyed_proof_nonce_b_test.gno
(cd examples && gno test ./gno.land/r/nt/grc20routes/v0)

# the oracle: the suite is green without this file once the disjunct is dropped
sed -i '367s/ || nonceB <= 0//' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
(cd examples && gno test ./gno.land/r/nt/grc20routes/v0)
git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

```go
// The mirror of the (0, 2) case in TestKeyedProofRejectsMismatchedInputs, which
// lands on the nonceA disjunct.
func TestKeyedProofRejectsNonPositiveNonceB(cur realm, t *testing.T) {
	prover := testutils.TestAddress("npos")
	keyA := mint(0, cur, "NPA")
	keyB := mint(0, cur, "NPB")
	testing.SetOriginCaller(prover)

	// Zero is checked here before the already-parked guard, which would
	// otherwise catch it with a different message.
	uassert.AbortsWithMessage(t, cur, "grc20routes: nonces must be positive",
		func() { BeginKeyedProof(cross(cur), keyA, keyB, 2, 0) })

	// Negative: nothing downstream of the positivity guard looks at it, so with
	// the disjunct dropped BeginKeyedProof returns normally here and parks a
	// proof whose keyB leg no Approve can ever satisfy.
	uassert.AbortsWithMessage(t, cur, "grc20routes: nonces must be positive",
		func() { BeginKeyedProof(cross(cur), keyA, keyB, 2, -1) })
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:321 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L321) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L321)
Missing test: the eleven cases here are all character-class, emptiness or duplicate rejections, so [`maxEntries`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L540), [`maxPrefixArg`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L562), [`maxSymbolLen`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L587), [`maxOpLen`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L604) and [`maxIdentLen`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L616) fire on no input the suite builds. Deleting any one of the five leaves the suite green on the one write path a third-party realm reaches.

<details><summary>test cases</summary>

One case per cap at one byte over, plus the accept-at-the-cap route that keeps the comparisons inclusive:

```go
// entries builds n distinct valid entries, so only the maxEntries guard can fire.
func entries(n int) []Entry {
	es := make([]Entry, n)
	for i := 0; i < n; i++ {
		es[i] = Entry{"op_" + strings.Repeat("x", i), "Fn" + strings.Repeat("X", i)}
	}
	return es
}

func TestValidationRejectsOverlongValues(cur realm, t *testing.T) {
	mint(0, cur, "CAPS")

	// Each value is one byte over its constant: 17>16, 33>32, 65>64, 129>128.
	longOp := strings.Repeat("a", maxOpLen+1)
	longFunc := "A" + strings.Repeat("a", maxIdentLen) // exported, so only length can fail
	longSymbol := strings.Repeat("A", maxSymbolLen+1)
	longPrefix := strings.Repeat("a", maxPrefixArg+1)

	cases := []struct {
		name   string
		symbol string
		route  Route
		msg    string
	}{
		{
			"one entry over maxEntries",
			"CAPS",
			Route{Funcs: entries(maxEntries + 1)},
			"grc20routes: too many entry points",
		},
		{
			"operation name over maxOpLen",
			"CAPS",
			Route{Funcs: []Entry{{longOp, "Approve"}}},
			"grc20routes: operation name too long: " + longOp,
		},
		{
			"function name over maxIdentLen",
			"CAPS",
			Route{Funcs: []Entry{{OpApprove, longFunc}}},
			"grc20routes: approve name too long",
		},
		{
			// Shadowed by the grc20reg gate in Register, which refuses any symbol
			// over grc20.MaxSymbolLen (11) anyway. Pins the message, not a bound.
			"symbol over maxSymbolLen",
			longSymbol,
			Route{Funcs: []Entry{{OpApprove, "Approve"}}},
			"grc20routes: symbol too long",
		},
		{
			"prefix argument over maxPrefixArg",
			"CAPS",
			Route{Funcs: []Entry{{OpApprove, "Approve"}}, Prefix: []string{longPrefix}},
			"grc20routes: prefix argument too long",
		},
	}

	for _, tc := range cases {
		r, sym := tc.route, tc.symbol
		uassert.AbortsWithMessage(t, cur, tc.msg,
			func() { Register(cross(cur), sym, r) }, tc.name)
	}
}

// TestValidationAcceptsValuesAtTheCap is the other half: the caps are inclusive,
// so a route sitting exactly on every one of them registers.
func TestValidationAcceptsValuesAtTheCap(cur realm, t *testing.T) {
	mint(0, cur, "ATCAP")

	es := entries(maxEntries)
	es[0] = Entry{strings.Repeat("a", maxOpLen), "A" + strings.Repeat("a", maxIdentLen-1)}
	r := Route{Funcs: es, Prefix: []string{strings.Repeat("a", maxPrefixArg)}}

	uassert.NotAborts(t, cur, func() { Register(cross(cur), "ATCAP", r) },
		"a route exactly on every cap must register")
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:395 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L395) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L395)
Missing test: every `Register` test runs inside `package grc20routes` and mints through `mint`, so [`cur.Previous().PkgPath()`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L296) is always `selfPath` and `TestRouteAlwaysPointsAtItsOwnRealm` restates the key `Register` just built from it. Letting the `symbol` argument override `rlmPath` keeps all 20 tests green, so the case with no coverage is a second realm calling `Register` with a key aimed at the first realm's token.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
go build -o /tmp/gno-6187 ./gnovm/cmd/gno
G=examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno

python3 - "$G" <<'PY'
import sys
f = sys.argv[1]
s = open(f).read()
old = '''func Register(cur realm, symbol string, r Route) string {
\trlmPath := cur.Previous().PkgPath()
\tif rlmPath == "" {
\t\tpanic("grc20routes: caller is not a realm")
\t}'''
new = old + '''
\tif i := strings.IndexByte(symbol, '|'); i >= 0 {
\t\trlmPath = symbol[:i]
\t\tsymbol = symbol[i+1:]
\t}'''
assert old in s
open(f, "w").write(s.replace(old, new))
PY

(cd examples && /tmp/gno-6187 test ./gno.land/r/nt/grc20routes/v0)
git checkout "$G" && rm /tmp/gno-6187
```

A realm path taken straight from the caller's argument passes every test:

```
ok      ./gno.land/r/nt/grc20routes/v0 	3.12s
```
</details>

## gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar:14 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar#L14) · [↗](../../../../../.worktrees/gno-review-6187/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar#L14)
Missing test: this fixture has no step that is expected to fail, so deleting the [two allowance comparisons](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L408-L413) in `FinishKeyedProof` keeps it green against grc20factory as deployed.

<details><summary>test cases</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
# save the fixture below as
#   gno.land/pkg/integration/testdata/grc20routes_keyed_proof_refused.txtar
go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20routes_keyed_proof_refused' -p 1 -parallel 1

# the oracle: with the guard gone the new fixture reddens and the shipped one does not
sed -i '408,413d' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20routes_keyed_proof' -p 1 -parallel 1
git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

```txtar
# A proof whose second allowance never moved must be refused, and every token on
# the realm must stay on the convention afterwards. Green at this head, red with
# the two allowance checks in FinishKeyedProof removed, which the shipped fixture
# survives.
#
# g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep is grc20routes' own realm address,
# the probe the proof parks its throwaway allowance on.

loadpkg gno.land/r/demo/defi/grc20factory
loadpkg gno.land/r/nt/grc20routes/v0

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid tendermint_test test1
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20factory -func New -args Foo -args FOO -args 6 -args 1000000 -args 0 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20factory -func New -args Bar -args BAR -args 6 -args 1000000 -args 0 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

# The proof opens, and only the FIRST token's allowance moves. This is the shape
# an unkeyed realm can produce: one token per call.
gnokey maketx call -pkgpath gno.land/r/nt/grc20routes/v0 -func BeginKeyedProof -args gno.land/r/demo/defi/grc20factory.FOO -args gno.land/r/demo/defi/grc20factory.BAR -args 111 -args 222 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20factory -func Approve -args FOO -args g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep -args 111 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

# The step the shipped fixture never takes: FinishKeyedProof must abort.
! gnokey maketx call -pkgpath gno.land/r/nt/grc20routes/v0 -func FinishKeyedProof -args gno.land/r/demo/defi/grc20factory.FOO -args gno.land/r/demo/defi/grc20factory.BAR -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stderr 'gno.land/r/demo/defi/grc20factory.BAR does not carry its nonce'

# ...and the refusal leaves nothing behind: the realm still reads as the
# convention, for every token it hosts.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/demo/defi/grc20factory.FOO\")"
stdout '\\"prefix\\":\[\],\\"source\\":\\"convention\\"'

gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/demo/defi/grc20factory.BAR\")"
stdout '\\"prefix\\":\[\],\\"source\\":\\"convention\\"'

# Completing the second allowance is the only difference between the refusal
# above and a proof: the same pending entry then commits.
gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20factory -func Approve -args BAR -args g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep -args 222 -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/nt/grc20routes/v0 -func FinishKeyedProof -args gno.land/r/demo/defi/grc20factory.FOO -args gno.land/r/demo/defi/grc20factory.BAR -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/demo/defi/grc20factory.FOO\")"
stdout '\\"prefix\\":\[\\"FOO\\"\],\\"source\\":\\"proven\\"'
```
</details>

## gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar:24 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar#L24) · [↗](../../../../../.worktrees/gno-review-6187/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar#L24)
Missing test: the realms loaded beside grc20routes never call `Register`, so [`cur.Previous().PkgPath()`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L296) evaluates to a foreign path nowhere in the repository. Replace it with the literal `gno.land/r/nt/grc20routes/v0` and the package suite and this fixture both stay green.

<details><summary>test cases</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
# save the fixture below as
#   gno.land/pkg/integration/testdata/grc20routes_declared.txtar
go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20routes_declared' -p 1 -parallel 1
```

The oracle, with `rlmPath := cur.Previous().PkgPath()` replaced by the literal `"gno.land/r/nt/grc20routes/v0"`:

| suite / fixture | head | mutant |
| --- | --- | --- |
| `gno test ./gno.land/r/nt/grc20routes/v0` | ok | ok 3.30s |
| `testdata/grc20routes_keyed_proof.txtar` | ok | ok 6.81s |
| the fixture below | ok | FAIL 5.03s, `not registered in grc20reg: gno.land/r/nt/grc20routes/v0.PRB` |

```txtar
# Runs the declared half of grc20routes with a real foreign caller, which nothing
# at this head does: grc20routes_keyed_proof.txtar loads only grc20factory and
# wugnot, neither of which can call Register, and every Register in the package
# suite runs with cur.Previous().PkgPath() == selfPath.

loadpkg gno.land/r/nt/grc20routes/v0
loadpkg gno.land/r/demo/zdeclare $WORK/zdeclare

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid tendermint_test test1
stdout 'OK!'

# Before: nothing recorded, so the canonical convention -- which is wrong for
# this realm, whose entry points take the symbol first and rename approve.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/demo/zdeclare.PRB\")"
stdout '\\"prefix\\":\[\],\\"source\\":\\"convention\\"'

# The declared path, with a real MsgCall frame: zdeclare builds the Route in its
# own realm and hands it across. cur.Previous().PkgPath() is gno.land/r/demo/zdeclare.
gnokey maketx call -pkgpath gno.land/r/demo/zdeclare -func Declare -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'
stdout 'gno.land/r/demo/zdeclare.PRB'

# After: the caller-built Route survived the cross-realm hand-off, the sort and
# the store, with its arbitrary entry point name intact.
gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/demo/zdeclare.PRB\")"
stdout '\\"prefix\\":\[\\"PRB\\"\],\\"source\\":\\"declared\\"'
stdout '\\"approve\\":\\"ApproveExact\\"'
stdout '\\"transfer\\":\\"Transfer\\",\\"transfer_from\\":\\"TransferFrom\\"'

gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.Func(\"gno.land/r/demo/zdeclare.PRB\",\"approve\")"
stdout 'ApproveExact'

# A second Register from the same realm overwrites its own key.
gnokey maketx call -pkgpath gno.land/r/demo/zdeclare -func Redeclare -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid tendermint_test alice
stdout 'OK!'

gnokey query vm/qeval --data "gno.land/r/nt/grc20routes/v0.JSON(\"gno.land/r/demo/zdeclare.PRB\")"
stdout '\\"prefix\\":\[\],\\"source\\":\\"declared\\"'
stdout '\\"approve\\":\\"Approve\\"'

-- zdeclare/gnomod.toml --
module = "gno.land/r/demo/zdeclare"
gno = "0.9"

-- zdeclare/zdeclare.gno --
package zdeclare

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/p/nt/seqid/v0"

	"gno.land/r/nt/grc20reg/v0"
	"gno.land/r/nt/grc20routes/v0"
)

var (
	tok *grc20.Token
	led *grc20.PrivateLedger
	id  seqid.ID
)

func init(cur realm) {
	tok, led = grc20.NewToken("Probe", "PRB", 6, id.Next(), cur)
	grc20reg.Register(cross(cur), tok, "")
}

// Declare is the cross-realm Register neither the package suite nor the
// testdata exercises: the Route is built in THIS realm and handed across.
func Declare(cur realm) string {
	r := grc20routes.Keyed("PRB").With(grc20routes.OpApprove, "ApproveExact")
	return grc20routes.Register(cross(cur), "PRB", r)
}

// Redeclare overwrites the realm's own key with the canonical shape.
func Redeclare(cur realm) string {
	return grc20routes.Register(cross(cur), "PRB", grc20routes.Canonical())
}
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:194 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L194) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L194)
Nit: `maxSymbolLen` is 64 while [`grc20.MaxSymbolLen`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/p/nt/grc20/v0/types.gno#L142) caps every token symbol at 11, so this bound can only fire on a symbol no registry key can hold. It also runs [ahead of the `grc20reg` lookup](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L300-L307) that would give the accurate reason.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:241 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L241) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L241)
Nit: `Func` returns `""` both for an operation a realm declared out and for one nothing was recorded about, and [its doc names only the first](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L492-L493), so [`Func("gno.land/r/gnoland/wugnot.wugnot", "deposit")`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L500-L503) reports no deposit entry point while wugnot exports [`Deposit`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L31). A second return value carrying whether the entry point was found separates the two, which `Get` already gives a caller willing to read the whole route.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:267 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L267) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L267)
Nit: `With` rebuilds `Funcs` into a fresh slice and copies only the `Prefix` slice header, so every route chained off one [`Keyed`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L256) call shares one backing array, which no exported call writes into today.

```suggestion
	out := Route{Prefix: append([]string(nil), r.Prefix...)}
```

## SKIP examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:345 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L345) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L345)
Nit: [`PrivateLedger.Approve`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/p/nt/grc20/v0/token.gno#L276) writes the allowance of any `owner` the token realm names and never checks the caller, so a token realm can park or clear a probe allowance on an address that never called it.

Skipped: a finding on a code comment's own wording, which changes no behaviour.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:367 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L367) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L367)
Test: no test passes a negative nonce or `math.MaxInt64`, so the nonce sign guard is reached only at `0` and narrowing `<= 0` to `== 0` would stay green.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:456 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L456) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L456)
Nit: `IsKeyedRealm` looks its argument up raw while [`Get`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L437) and [`PkgPath`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L453) both normalize through [`hostOf`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L523-L525), so the package-path half of a `<host>#<sub>.<SYM>` registry key answers false on a realm those two report as proven.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:470-484 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L470-L484) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L470)
Nit: the prefix and funcs fragments are built by two index-guarded concatenation loops, 15 lines, where [`strings.Join`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L321-L322) over two prebuilt slices is 11 and emits the same bytes.

```suggestion
	prefixParts := make([]string, len(r.Prefix))
	for i, p := range r.Prefix {
		prefixParts[i] = `"` + p + `"`
	}
	prefix := strings.Join(prefixParts, ",")

	funcParts := make([]string, len(r.Funcs))
	for i, e := range r.Funcs {
		funcParts[i] = `"` + e.Op + `":"` + e.Func + `"`
	}
	funcs := strings.Join(funcParts, ",")
```

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:549 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L549) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L549)
Nit: the duplicate-operation check rescans `r.Funcs[:i]` for every entry, 5 lines, where [sorting before validating](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L309-L310) makes it the one-line test `i > 0 && r.Funcs[i-1].Op == e.Op`. The cost is reordering the caller's slice before a route can be rejected.

## SKIP examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:573 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L573) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L573)
Nit: `validateSymbol`, [`validateOpName`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L600) and [`validateFuncName`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L615) each hand-roll a length check and a per-rune allowed-set loop over three slightly different sets, 58 lines for one skeleton.

Skipped: the shared-helper version measures 48 lines against 58, and it reddens `TestValidationRejectsUnsafeValues` and `TestRegisterRejectsCraftedSymbols`, because the three panic strings differ in shape and one shared message cannot carry them all.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:615-616 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L615-L616) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L615)
Nit: `validateFuncName` returns without panicking on `""`, its loop running zero times, so the exported-identifier contract holds only while [`validate`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L545-L547) guards the one call site. [`validateOpName`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L601-L603) rejects `""` itself.

```suggestion
func validateFuncName(field, name string) {
	if name == "" {
		panic("grc20routes: empty entry point for " + field)
	}
	if len(name) > maxIdentLen {
```

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:653 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L653) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L653)
Nit: `Render` hands every non-empty path to `renderToken`, which [panics with `grc20routes: unknown token:`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L515-L516) on any path that is not a live registry key. A mistyped or stale gnoweb link aborts the call instead of rendering a not-found page with a way back to the index.

## SKIP examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:654 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L654) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L654)
Nit: `renderToken` takes any non-empty path and aborts through `mustToken`, so an arbitrary user-supplied subpath under the realm takes the page down.

Skipped: the same edit closes it as the section anchored on the branch one line above.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:664-672 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L664-L672) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L664)
Nit: `n` is incremented across the whole walk and read once as a flag, here and again in [the declared block](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L675-L695), where `Size()` on each tree answers the same question and drops 4 lines.

```suggestion
	if provenKeyed.Size() == 0 {
		s += "_None yet._\n"
	}
	provenKeyed.Iterate("", "", func(pkgPath string, _ any) bool {
		s += "- " + md.InlineCode(pkgPath) + " — `Approve(cur, SYMBOL, spender, amount)`\n"
		return false
	})
```

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:665 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L665) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L665)
Nit: the home page walks `provenKeyed` here and [`declared`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L676) end to end, with no start key and a callback that always returns `false`. The page's gas cost grows with every realm proven and every route declared, and neither tree has a removal path.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:701 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L701) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L701)
Nit: `renderToken` parses `tokenKey` itself instead of reusing [`PkgPath`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L453), so for a token registered under a sub identity [the realm line and its link](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L705) name `<path>#<sub>`, the form [`mustToken` strips](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L505-L513) because it is not a `MsgCall` destination.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:722 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L722) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L722)
Nit: `opTail` returns `, …` for every operation outside the fixed three, so a route declaring [`deposit`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L265) renders as `Deposit(cur, …)` while `approve` on the same route renders its argument names. The open operation set is exactly what a reader comes to the page for.

## SKIP examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:731 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L731) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L731)
Nit: `opTail` returns `", …"` for every operation outside the canonical three, so a declared `deposit` renders as `Deposit(cur, …)` and carries nothing the `Func` name did not, on exactly [the open set the `Funcs` doc argues this realm exists for](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L217-L223).

Skipped: the same edit closes it as the section anchored on the `default` arm nine lines above.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:735 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L735) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L735)
Nit: `args` is seeded with `cur`, which the VM injects and no signer can pass, so the page prints `Approve(cur, FOO, spender, amount)`. The call that works takes three arguments, [`-func Approve -args FOO -args <spender> -args <amount>`](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar#L50), as [the same page's own header](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L659-L661) says.

## SKIP examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:39 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L39) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L39)
Test: the suite's keyed-proof assertions hold for any value of `probe`, since `approveAs` parks the allowance at `ProbeAddress()` and [`allowanceOf`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L426) reads it back at that same variable.

Skipped: nothing here asks the author for a change, and the integration fixture's literal address already pins what the suite cannot.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:60-64 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L60-L64) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L60)
Test: the convention cases assert nothing about `IsKeyedRealm(selfPath)`. They fail whenever a proof test declared above them marks the realm first.

```suggestion
// Every token in this file lives on this one realm, so a keyed proof below marks
// the whole realm. The convention cases therefore have to run before it — they
// are first in the file, and gno runs tests in declaration order.
func TestJSONFallsBackAndNamesTheSource(cur realm, t *testing.T) {
	urequire.False(t, IsKeyedRealm(selfPath),
		"a keyed-proof test declared above this one inverts every assertion below")
	key := mint(0, cur, "JSNUN")
```

<details><summary>run both ways</summary>

```
# with the guard, current declaration order
ok      ./gno.land/r/nt/grc20routes/v0 	3.25s

# with the guard, TestKeyedProofCoversEveryTokenOnTheRealm moved above line 47
=== RUN   TestKeyedProofCoversEveryTokenOnTheRealm
--- PASS: TestKeyedProofCoversEveryTokenOnTheRealm (0.03s)
=== RUN   TestUnrecordedTokenFallsBackToCanonical
--- FAIL: TestUnrecordedTokenFallsBackToCanonical (0.00s)
=== RUN   TestJSONFallsBackAndNamesTheSource
should be false - a keyed-proof test declared above this one inverts every assertion below
--- FAIL: TestJSONFallsBackAndNamesTheSource (0.00s)
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:155-156 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L155-L156) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L155)
Test: no case reaches this mint, since [`validateSymbol`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L300) aborts all six symbols in the table below before the [`grc20reg.Get` lookup](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L305).

```suggestion
```

## SKIP examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno:254 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L254) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L254)
Test: bob's `FinishKeyedProof` aborts at the prover-scoped pending lookup, so the allowance comparisons below it never run.

Skipped: the finding is the comment's own wording, which changes no behaviour.

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:296-299 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L296-L299) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L296)
Suggestion: `Register` takes a `Route` struct, and [`convertArgToGno`](https://github.com/gnolang/gno/blob/3aa06ef75/gno.land/pkg/sdk/vm/convert.go#L226) panics on any contract argument outside a primitive, array or slice, so no `MsgCall` can reach this function and `cur.Previous().PkgPath()` is never empty.

```suggestion
	rlmPath := cur.Previous().PkgPath()
```

<details><summary>both ways</summary>

4 lines to 1, and the file goes from 740 lines to 737. The guard's own path, a caller that is not a realm, cannot be constructed: `Route` is a struct parameter, so the transaction is refused before the VM enters `Register`. The package suite is green with the three lines gone.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
sed -i '297,299d' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
(cd examples && gno test ./gno.land/r/nt/grc20routes/v0)
git checkout -- examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
```

```
ok      ./gno.land/r/nt/grc20routes/v0 	3.19s
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:361-363 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L361-L363) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L361)
Suggestion: the same-realm check compares paths [`mustToken`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L514-L519) has already stripped of `#subpath`, so a realm's top-level token and one it registered under `cur.Sub("x")` pass as one realm, and the host's token comes out proven from two approvals through separate ledgers. Comparing the raw registry paths, before `hostOf` strips them, keeps a sub identity's token out of its host's proof.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6187 -R gnolang/gno
mkdir -p examples/gno.land/r/demo/zsub6
printf 'module = "gno.land/r/demo/zsub6"\n\ngno = "0.9"\n' > examples/gno.land/r/demo/zsub6/gnomod.toml
cat > examples/gno.land/r/demo/zsub6/main_filetest.gno <<'GNO'
// PKGPATH: gno.land/r/demo/zsub6
package zsub6

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/r/nt/grc20reg/v0"
	"gno.land/r/nt/grc20routes/v0"
)

var (
	hostLedger, subLedger *grc20.PrivateLedger
	keyHost, keySub       string
)

func init(cur realm) {
	// keyHost is minted and registered from this realm's own top-level frame.
	// It exposes no keyed Approve at all.
	host, hl := grc20.NewToken("Host", "HOST", 4, 0, cur)
	hostLedger = hl
	keyHost = grc20reg.Register(cross(cur), host, "")

	// keySub is minted and registered from this same realm's cur.Sub("x").
	sub := cur.Sub("x")
	subTok, sl := grc20.NewToken("Sub", "SUBT", 4, 1, sub)
	subLedger = sl
	keySub = grc20reg.Register(cross(sub), subTok, "")
}

func main(cur realm) {
	probe := grc20routes.ProbeAddress()
	prover := cur.Address()

	// The pair runs entirely through the two ledger objects, never through a
	// keyed entry point on the host token.
	grc20routes.BeginKeyedProof(cross(cur), keyHost, keySub, 111, 222)
	must(hostLedger.Approve(prover, probe, 111))
	must(subLedger.Approve(prover, probe, 222))
	grc20routes.FinishKeyedProof(cross(cur), keyHost, keySub)

	println("keyed(host)      :", grc20routes.IsKeyedRealm("gno.land/r/demo/zsub6"))
	rHost, sourceHost := grc20routes.Get(keyHost)
	println("route(host token):", sourceHost, rHost.Prefix)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
GNO
(cd examples && gno test -v ./gno.land/r/demo/zsub6)
rm -rf examples/gno.land/r/demo/zsub6
```

The host's own token comes back keyed, from a proof its own identity never joined.

```
keyed(host)      : true
route(host token): proven slice[("HOST" string)]
```
</details>

## examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno:467 [gh](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L467) · [↗](../../../../../.worktrees/gno-review-6187/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L467)
Suggestion: `mustToken` traverses `grc20reg` and [`Get`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L437) traverses it again on the next line, and [`BeginKeyedProof`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L359-L360) and [`FinishKeyedProof`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno#L393) make 4 and 3 traversals where 2 each answer the same questions. One `resolve(tokenKey) (*grc20.Token, string)` helper, plus a `getRoute` returning the `pkgPath` and `symbol` its callers already need, brings the three to 1, 2 and 2, at 757 lines against 740.

<details><summary>patch</summary>

Traversals per call, before and after: `JSON` 2 to 1, `BeginKeyedProof` 4 to 2, `FinishKeyedProof` 3 to 2, `renderToken` one lookup and two `fqname.Parse` to one lookup and at most one. The `grc20routes: unknown token: ` string [`TestRejectsUnknownToken`](https://github.com/gnolang/gno/blob/3aa06ef75/examples/gno.land/r/nt/grc20routes/v0/grc20routes_test.gno#L314) pins stays byte-identical, since `resolve` keeps it verbatim.

```diff
--- a/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
+++ b/examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
@@ -143,6 +143,7 @@ import (
 	"gno.land/p/moul/md/v0"
 	"gno.land/p/nt/avl/v0"
 	"gno.land/p/nt/fqname/v0"
+	"gno.land/p/nt/grc20/v0"
 	"gno.land/p/nt/ufmt/v0"
 
 	"gno.land/r/nt/grc20reg/v0"
@@ -356,8 +357,8 @@ type proof struct {
 func BeginKeyedProof(cur realm, tokenKeyA, tokenKeyB string, nonceA, nonceB int64) {
 	prover := cur.Previous().Address()
 
-	pathA := mustToken(tokenKeyA)
-	pathB := mustToken(tokenKeyB)
+	tokA, pathA := resolve(tokenKeyA)
+	tokB, pathB := resolve(tokenKeyB)
 	if pathA != pathB {
 		panic("grc20routes: both tokens must live in the same realm")
 	}
@@ -370,7 +371,7 @@ func BeginKeyedProof(cur realm, tokenKeyA, tokenKeyB string, nonceA, nonceB int6
 	if nonceA == nonceB {
 		panic("grc20routes: nonces must differ")
 	}
-	if allowanceOf(tokenKeyA, prover) == nonceA || allowanceOf(tokenKeyB, prover) == nonceB {
+	if tokA.Allowance(prover, probe) == nonceA || tokB.Allowance(prover, probe) == nonceB {
 		panic("grc20routes: nonce already parked on the probe; pick another")
 	}
 
@@ -390,7 +391,7 @@ func BeginKeyedProof(cur realm, tokenKeyA, tokenKeyB string, nonceA, nonceB int6
 // something else.
 func FinishKeyedProof(cur realm, tokenKeyA, tokenKeyB string) {
 	prover := cur.Previous().Address()
-	pathA := mustToken(tokenKeyA)
+	tokA, pathA := resolve(tokenKeyA)
 
 	pk := pendingKey(prover, pathA)
 	v := pending.Get(pk)
@@ -402,13 +403,15 @@ func FinishKeyedProof(cur realm, tokenKeyA, tokenKeyB string) {
 		panic("grc20routes: tokens do not match the proof in progress")
 	}
 
+	tokB, _ := resolve(tokenKeyB)
+
 	// The load-bearing check. Two distinct registry keys at one realm, each
 	// carrying the exact allowance claimed for it. An unkeyed Approve reaches
 	// one token, so it cannot produce both.
-	if allowanceOf(p.keyA, prover) != p.nonceA {
+	if tokA.Allowance(prover, probe) != p.nonceA {
 		panic("grc20routes: " + p.keyA + " does not carry its nonce")
 	}
-	if allowanceOf(p.keyB, prover) != p.nonceB {
+	if tokB.Allowance(prover, probe) != p.nonceB {
 		panic("grc20routes: " + p.keyB + " does not carry its nonce")
 	}
 
@@ -422,8 +425,17 @@ func pendingKey(prover address, pkgPath string) string {
 	return prover.String() + "/" + pkgPath
 }
 
-func allowanceOf(tokenKey string, owner address) int64 {
-	return grc20reg.MustGet(tokenKey).Allowance(owner, probe)
+// resolve returns the token at tokenKey and the realm it routes to, doing one
+// registry traversal and one fqname.Parse for both.
+//
+// Panics if tokenKey is not a registered token.
+func resolve(tokenKey string) (*grc20.Token, string) {
+	tok := grc20reg.Get(tokenKey)
+	if tok == nil {
+		panic("grc20routes: unknown token: " + tokenKey)
+	}
+	pkgPath, _ := fqname.Parse(tokenKey)
+	return tok, hostOf(pkgPath)
 }
 
 // ------------------------------------------------------------------ reading
@@ -434,16 +446,25 @@ func allowanceOf(tokenKey string, owner address) int64 {
 //
 // Panics if tokenKey is not a registered token.
 func Get(tokenKey string) (Route, string) {
-	pkgPath := mustToken(tokenKey)
+	_, _, r, source := getRoute(tokenKey)
+	return r, source
+}
+
+// getRoute is Get plus the pkgPath and symbol its callers already need, so a
+// caller wanting more than the route does one registry traversal, not two.
+// symbol is only computed on the proven path; other callers that need it
+// parse tokenKey themselves.
+func getRoute(tokenKey string) (pkgPath, symbol string, r Route, source string) {
+	pkgPath = mustToken(tokenKey)
 
 	if v := declared.Get(tokenKey); v != nil {
-		return *(v.(*Route)), SourceDeclared
+		return pkgPath, "", *(v.(*Route)), SourceDeclared
 	}
 	if provenKeyed.Has(pkgPath) {
-		_, symbol := fqname.Parse(tokenKey)
-		return Keyed(symbol), SourceProven
+		_, symbol = fqname.Parse(tokenKey)
+		return pkgPath, symbol, Keyed(symbol), SourceProven
 	}
-	return Canonical(), SourceConvention
+	return pkgPath, "", Canonical(), SourceConvention
 }
 
 // PkgPath returns the realm a user sends the MsgCall to, which is the package
@@ -464,8 +485,7 @@ func IsKeyedRealm(pkgPath string) bool { return provenKeyed.Has(pkgPath) }
 // registry key, or validated on the way in (see validate), so the output needs
 // no escaping and cannot be made to carry a quote or a backslash.
 func JSON(tokenKey string) string {
-	pkgPath := mustToken(tokenKey)
-	r, source := Get(tokenKey)
+	pkgPath, _, r, source := getRoute(tokenKey)
 
 	prefix := ""
 	for i, p := range r.Prefix {
@@ -512,11 +532,8 @@ func Func(tokenKey, op string) string {
 // here to "<path>" — the realm a user can actually call. This is the same
 // normalization grc20's guardHome applies for the same reason.
 func mustToken(tokenKey string) string {
-	if grc20reg.Get(tokenKey) == nil {
-		panic("grc20routes: unknown token: " + tokenKey)
-	}
-	pkgPath, _ := fqname.Parse(tokenKey)
-	return hostOf(pkgPath)
+	_, pkgPath := resolve(tokenKey)
+	return pkgPath
 }
 
 // hostOf drops the "#subpath" a realm carries under a sub identity.
@@ -697,8 +714,10 @@ func Render(path string) string {
 }
 
 func renderToken(tokenKey string) string {
-	r, source := Get(tokenKey)
-	pkgPath, symbol := fqname.Parse(tokenKey)
+	pkgPath, symbol, r, source := getRoute(tokenKey)
+	if symbol == "" {
+		_, symbol = fqname.Parse(tokenKey)
+	}
 
 	s := ufmt.Sprintf("# %s\n\n", md.EscapeText(symbol))
 	s += "- token key: " + md.InlineCode(tokenKey) + "\n"
```
</details>
