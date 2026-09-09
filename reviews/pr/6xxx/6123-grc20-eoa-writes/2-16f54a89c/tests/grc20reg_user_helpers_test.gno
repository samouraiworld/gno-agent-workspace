// Unit coverage for the three functions PR 6123 adds to grc20reg. The package's
// own suite names none of them: UserTransfer, UserApprove and UserTransferFrom
// reach a test binary only through the integration txtar, so the actor binding
// regresses minutes away from the package. Passes at 16f54a89c.
//
/* Run: from a gno checkout:
gh pr checkout 6123 -R gnolang/gno && git checkout 16f54a89c
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/grc20reg_user_helpers_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6123-grc20-eoa-writes/2-16f54a89c/tests/grc20reg_user_helpers_test.gno
cd examples && go run ../gnovm/cmd/gno test -v -run TestUserHelpers ./gno.land/r/demo/defi/grc20reg
rm gno.land/r/demo/defi/grc20reg/grc20reg_user_helpers_test.gno
*/

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
