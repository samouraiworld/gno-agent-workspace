/* Run: from a gno checkout:
gh pr checkout 6028 -R gnolang/gno && git checkout 37182a315
curl -fsSL -o examples/gno.land/p/demo/tokens/grc20/nil_generator_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6028-registry-owned-id-generator/1-37182a315/tests/grc20_nil_generator_test.gno
cd examples && go run ../gnovm/cmd/gno test -v -run TestNewTokenRejectsNilGenerator ./gno.land/p/demo/tokens/grc20
rm gno.land/p/demo/tokens/grc20/nil_generator_test.gno
*/

// NewToken gained a nil-generator guard and ErrNilGenerator, neither of which
// any shipped test reaches. A nil *identifier.Generator is the shape a realm
// declaring "var gen *identifier.Generator" and forgetting to construct it
// hands in. This passes at 37182a315.
package grc20

import (
	"testing"

	"gno.land/p/nt/urequire/v0"
)

func TestNewTokenRejectsNilGenerator(cur realm, t *testing.T) {
	urequire.AbortsContains(t, cur, ErrNilGenerator.Error(), func() {
		func(cur realm) {
			NewToken("Nil", "NIL", 6, nil, cur)
		}(cross(cur))
	})
}
