/* Run: from a gno checkout:
gh pr checkout 6028 -R gnolang/gno && git checkout 37182a315
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/empty_slug_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6028-registry-owned-id-generator/1-37182a315/tests/grc20reg_empty_slug_test.gno
cd examples && go run ../gnovm/cmd/gno test -v -run TestRegisterEmptySlug ./gno.land/r/demo/defi/grc20reg
rm gno.land/r/demo/defi/grc20reg/empty_slug_test.gno
*/

// foo20, wugnot, test20 and bar20 all register with an empty slug, so the key
// they end up under is the bare realm path. No shipped test registers
// successfully with an empty slug. This pins that key shape and passes at
// 37182a315.
package grc20reg

import (
	"testing"

	"gno.land/p/demo/tokens/grc20"
	"gno.land/p/nt/urequire/v0"
)

func TestRegisterEmptySlug(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/grc20reg_empty_slug"))
	token, _ := grc20.NewToken("Empty", "EMT", 4, IdentifierGenerator(), cur)

	key := Register(cross(cur), token, "")
	urequire.Equal(t, "gno.land/r/demo/grc20reg_empty_slug", key)
	urequire.Equal(t, token.ID(), Get(key).ID())

	// The bare realm path is a single slot: a second empty-slug registration
	// from the same realm is rejected.
	other, _ := grc20.NewToken("Other", "OTH", 4, IdentifierGenerator(), cur)
	urequire.AbortsContains(t, cur, "token already registered", func() {
		Register(cross(cur), other, "")
	})
}
