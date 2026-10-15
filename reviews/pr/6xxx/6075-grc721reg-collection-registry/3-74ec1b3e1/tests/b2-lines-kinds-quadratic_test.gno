// gnolang/gno#6075 round 3, finder b2-lines: GetExtensionKinds walks every
// kind (publishedKinds(v, 0)) and pairs each with Token.ExtensionView, whose
// lookupExtension is a linear scan, so one read costs ~n^2/2 iterations over
// an extension count the issuer alone chooses.
//
// Repro, from a plain clone of github.com/gnolang/gno:
//
//	git checkout 74ec1b3e1ad9406d645443fffbca8ae277b8edf7
//	go install ./gnovm/cmd/gno
//	cp b2-lines-kinds-quadratic_test.gno examples/gno.land/r/demo/defi/grc721reg/b2lines_cost_test.gno
//	cd examples && gno test -v -print-runtime-metrics -run TestB2Cost ./gno.land/r/demo/defi/grc721reg/
//
// Observed at 74ec1b3e1 (GAS line under each test; read cost = Read - Setup):
//
//	TestB2CostSetup200  33571967   TestB2CostRead200  61714332   read  28.1M
//	TestB2CostSetup400 118332789   TestB2CostRead400 225632077   read 107.3M
//
// Doubling n multiplies the read by 3.81. maxGasQuery is 3_000_000_000
// (gno.land/pkg/sdk/vm/keeper.go:52); extrapolated, ~2100 hooks put one
// GetExtensionKinds call over it, while each further registration costs the
// issuer O(n) in its own transaction.
package grc721reg

import (
	"strconv"
	"testing"

	"gno.land/p/demo/tokens/grc721"
)

type costHook struct{ kind string }

func (h *costHook) ExtensionKind() string                       { return h.kind }
func (h *costHook) OnMint(address, grc721.TokenID)              {}
func (h *costHook) OnTransfer(address, address, grc721.TokenID) {}
func (h *costHook) OnBurn(grc721.TokenID)                       {}

func setupHooks(cur realm, path string, n int) string {
	testing.SetRealm(testing.NewCodeRealm(path))
	token, ledger := grc721.NewToken("Cost", "COST", 0, cur)
	for i := 0; i < n; i++ {
		ledger.RegisterExtension(&costHook{"h" + strconv.Itoa(i)}, nil)
	}
	return Register(cross(cur), token, "")
}

func TestB2CostSetup200(cur realm, t *testing.T) { setupHooks(cur, "gno.land/r/demo/b2_s200", 200) }
func TestB2CostRead200(cur realm, t *testing.T) {
	k := setupHooks(cur, "gno.land/r/demo/b2_r200", 200)
	GetExtensionKinds(k)
}
func TestB2CostSetup400(cur realm, t *testing.T) { setupHooks(cur, "gno.land/r/demo/b2_s400", 400) }
func TestB2CostRead400(cur realm, t *testing.T) {
	k := setupHooks(cur, "gno.land/r/demo/b2_r400", 400)
	GetExtensionKinds(k)
}
