// b2-reach-kinds-quadratic: GetExtensionKinds walks ExtensionKindsUpTo(0) and
// calls Token.ExtensionView per kind, whose lookupExtension is a linear scan,
// so one read costs O(n^2) in the extension count the collection's issuer sets,
// extensions added after Register included (the registry keeps the live token).
//
// Repro from a plain clone of gnolang/gno at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7:
//
//   cp b2-reach-kinds-quadratic.gno examples/gno.land/r/demo/defi/grc721reg/zz_reach_quad_test.gno
//   cd examples && gno test ./gno.land/r/demo/defi/grc721reg -run 'TestReachQuad' -v -print-runtime-metrics
//
// Read cost = GAS(Read<n>) - GAS(Setup<n>). Linear would double per doubling of n;
// quadratic quadruples.
package grc721reg

import (
	"strconv"
	"testing"

	"gno.land/p/demo/tokens/grc721"
)

func reachQuadSetup(cur realm, n int, path string) string {
	testing.SetRealm(testing.NewCodeRealm(path))
	token, ledger := grc721.NewToken("Quad", "QUAD", 0, cur)
	key := Register(cross(cur), token, "")
	for i := 0; i < n; i++ {
		v := &stubView{"kind" + strconv.Itoa(i), token.ID()}
		ledger.RegisterExtension(v, v)
	}
	return key
}

func TestReachQuadSetup100(cur realm, t *testing.T) { reachQuadSetup(cur, 100, "gno.land/r/demo/rq_s100") }
func TestReachQuadRead100(cur realm, t *testing.T) {
	t.Logf("kinds: %d", len(GetExtensionKinds(reachQuadSetup(cur, 100, "gno.land/r/demo/rq_r100"))))
}
func TestReachQuadSetup200(cur realm, t *testing.T) { reachQuadSetup(cur, 200, "gno.land/r/demo/rq_s200") }
func TestReachQuadRead200(cur realm, t *testing.T) {
	t.Logf("kinds: %d", len(GetExtensionKinds(reachQuadSetup(cur, 200, "gno.land/r/demo/rq_r200"))))
}
func TestReachQuadSetup400(cur realm, t *testing.T) { reachQuadSetup(cur, 400, "gno.land/r/demo/rq_s400") }
func TestReachQuadRead400(cur realm, t *testing.T) {
	t.Logf("kinds: %d", len(GetExtensionKinds(reachQuadSetup(cur, 400, "gno.land/r/demo/rq_r400"))))
}
func TestReachQuadSetup800(cur realm, t *testing.T) { reachQuadSetup(cur, 800, "gno.land/r/demo/rq_s800") }
func TestReachQuadRead800(cur realm, t *testing.T) {
	t.Logf("kinds: %d", len(GetExtensionKinds(reachQuadSetup(cur, 800, "gno.land/r/demo/rq_r800"))))
}
