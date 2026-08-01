/* Run: from a gno checkout:
gh pr checkout 6027 -R gnolang/gno && git checkout 854b03529
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/alias_identity_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6027-slug-alias-registrations/1-854b03529/tests/alias_identity_test.gno
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/r/demo/defi/grc20reg/
rm gno.land/r/demo/defi/grc20reg/alias_identity_test.gno
*/

// Multiple aliases store one pointer under several keys. If persistence ever
// splits that object, each alias gets its own ledger and balances drift.
// Passes at 854b03529; asserts the property the alias feature rests on.

package grc20reg

import (
	"testing"

	"gno.land/p/demo/tokens/grc20"
	"gno.land/p/nt/urequire/v0"
)

func TestAliasesShareOneObject(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/grc20reg_alias_identity"))
	token, ledger := grc20.NewToken("Aliased", "ALI", 4, 0, cur)
	ledger.Mint(cur.Address(), 42)

	first := Register(cross(cur), token, "first")
	second := Register(cross(cur), token, "second")

	urequire.True(t, Get(first) == Get(second), "both aliases must resolve to one token object")
	urequire.Equal(t, int64(42), Get(second).BalanceOf(cur.Address()))
}
