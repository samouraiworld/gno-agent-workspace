/* Run: from a gno checkout:
gh pr checkout 5983 -R gnolang/gno && git checkout 7d9a11104
curl -fsSL -o examples/gno.land/r/zeycan1/tba/tba_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5983-token-bound-account-realm/1-7d9a11104/tests/tba_test.gno
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/r/zeycan1/tba
rm gno.land/r/zeycan1/tba/tba_test.gno
*/

// The realm ships with no tests. These cover the admin gate, the owner
// gate, and vault control following the token, plus the three properties
// the current code does not hold: Withdraw accepting a zero amount,
// TransferFrom reporting an authorization failure as a return value, and
// the selfPath constant being unchecked against the realm's real path.
// At the pinned hash TestWithdrawRejectsNonPositive and
// TestTransferFromAbortsWhenUnauthorized fail; the other four pass.
package tba

import (
	"testing"

	"chain"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/uassert/v0"
)

const (
	adminAddr = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
	aliceAddr = address("g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0")
	bobAddr   = address("g10ddhfxv47lhyhpj426u57ckxwl0qtrk6c9rs5v")
	sinkAddr  = address("g13ek2zz9qurzynzvssyc4sthwppnruhnp0gdz8n")
)

// setup makes the realm usable from a unit test. init() reads
// unsafe.PreviousRealm(), which is the empty address under `gno test`,
// so admin has to be assigned before any admin-gated call.
func setup(cur realm, t *testing.T) grc721.TokenID {
	t.Helper()
	admin = adminAddr
	testing.SetRealm(testing.NewUserRealm(adminAddr))
	if nft == nil {
		Setup(cross(cur))
	}
	return Mint(cross(cur), aliceAddr, "art", "ipfs://img")
}

func TestSetupAndMintAreAdminOnly(cur realm, t *testing.T) {
	admin = adminAddr
	nft = nil

	testing.SetRealm(testing.NewUserRealm(aliceAddr))
	uassert.Equal(t, "only admin can initialize", revive(func() { Setup(cross(cur)) }))

	testing.SetRealm(testing.NewUserRealm(adminAddr))
	Setup(cross(cur))
	uassert.Equal(t, "already initialized", revive(func() { Setup(cross(cur)) }))

	testing.SetRealm(testing.NewUserRealm(aliceAddr))
	uassert.Equal(t, "only admin can mint", revive(func() {
		Mint(cross(cur), aliceAddr, "art", "ipfs://img")
	}))
}

func TestWithdrawIsOwnerOnly(cur realm, t *testing.T) {
	tid := setup(cur, t)
	testing.IssueCoins(VaultAddress(tid), chain.NewCoins(chain.NewCoin("ugnot", 1000)))

	testing.SetRealm(testing.NewUserRealm(bobAddr))
	uassert.Equal(t, "caller is not the current owner of this token", revive(func() {
		Withdraw(cross(cur), tid, bobAddr, 100)
	}))
	uassert.Equal(t, int64(1000), Balance(tid))

	testing.SetRealm(testing.NewUserRealm(aliceAddr))
	Withdraw(cross(cur), tid, sinkAddr, 400)
	uassert.Equal(t, int64(600), Balance(tid))
}

func TestVaultFollowsTheToken(cur realm, t *testing.T) {
	tid := setup(cur, t)
	testing.IssueCoins(VaultAddress(tid), chain.NewCoins(chain.NewCoin("ugnot", 500)))

	testing.SetRealm(testing.NewUserRealm(aliceAddr))
	uassert.NoError(t, TransferFrom(cross(cur), aliceAddr, bobAddr, tid))

	// The old owner loses the vault; the new owner gains it, with the
	// balance untouched by the transfer.
	testing.SetRealm(testing.NewUserRealm(aliceAddr))
	uassert.Equal(t, "caller is not the current owner of this token", revive(func() {
		Withdraw(cross(cur), tid, aliceAddr, 100)
	}))

	testing.SetRealm(testing.NewUserRealm(bobAddr))
	Withdraw(cross(cur), tid, sinkAddr, 500)
	uassert.Equal(t, int64(0), Balance(tid))
}

func TestWithdrawRejectsNonPositive(cur realm, t *testing.T) {
	tid := setup(cur, t)
	testing.IssueCoins(VaultAddress(tid), chain.NewCoins(chain.NewCoin("ugnot", 1000)))
	testing.SetRealm(testing.NewUserRealm(aliceAddr))

	// A zero-amount withdraw currently commits a no-op transaction.
	abort := revive(func() { Withdraw(cross(cur), tid, bobAddr, 0) })
	uassert.True(t, abort != nil, "zero amount was accepted")
	uassert.Equal(t, int64(1000), Balance(tid))
}

func TestTransferFromAbortsWhenUnauthorized(cur realm, t *testing.T) {
	tid := setup(cur, t)

	// bob owns neither the token nor an approval. The call must not
	// report success back to the caller.
	testing.SetRealm(testing.NewUserRealm(bobAddr))
	abort := revive(func() {
		TransferFrom(cross(cur), aliceAddr, bobAddr, tid)
	})
	uassert.True(t, abort != nil, "unauthorized transfer returned instead of aborting")

	owner, err := OwnerOf(tid)
	uassert.NoError(t, err)
	uassert.Equal(t, aliceAddr.String(), owner.String())
}

func TestVaultAddressMatchesTheMintedSubRealm(cur realm, t *testing.T) {
	tid := setup(cur, t)

	// Withdraw spends from cur.Sub(vaultSubpath(tid)); VaultAddress and
	// Balance derive from the selfPath constant. Depositors follow
	// VaultAddress, so the two must name the same account.
	uassert.Equal(t, cur.Sub(vaultSubpath(tid)).Address().String(), VaultAddress(tid).String())
}
