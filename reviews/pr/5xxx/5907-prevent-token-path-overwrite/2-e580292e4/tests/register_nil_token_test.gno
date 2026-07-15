/* Run: from a gno checkout:
gh pr checkout 5907 -R gnolang/gno && git checkout e580292e4
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/zz_nil_token_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5907-prevent-token-path-overwrite/2-e580292e4/tests/register_nil_token_test.gno
cd examples && gno test -v -run TestRegisterRejectsNilToken ./gno.land/r/demo/defi/grc20reg/
rm gno.land/r/demo/defi/grc20reg/zz_nil_token_test.gno
*/
// Register's nil guard (grc20reg.gno:24) is untested. Removing it makes a nil
// token abort deep in GetSymbol as "value method ...Token.GetSymbol called
// using nil *Token pointer" instead of the clean "grc20reg: nil token".
// This test locks the early reject and its message.
package grc20reg

import (
	"testing"

	"gno.land/p/demo/tokens/grc20"
	"gno.land/p/nt/urequire/v0"
)

func TestRegisterRejectsNilToken(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/grc20reg_nil"))
	urequire.AbortsContains(t, cur, "nil token", func() {
		Register(cross(cur), nil, "")
	})
}

// Baseline: a well-formed token from the same realm still registers, so the
// guard rejects only the nil case, not valid input.
func TestRegisterAcceptsValidToken(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/grc20reg_nil_ok"))
	token, _ := grc20.NewToken(0, cur, "Ok", "OK", 4)
	Register(cross(cur), token, "")
	urequire.Equal(t, "OK", Get(token.ID()).GetSymbol())
}
