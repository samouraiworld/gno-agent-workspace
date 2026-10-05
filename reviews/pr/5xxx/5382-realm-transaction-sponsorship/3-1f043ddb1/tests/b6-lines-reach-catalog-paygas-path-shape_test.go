// Repro: does the payGas/payStorage gas slope, fitted on one realm-path shape
// ("gno.land/r/a" + "/a"*k), hold for other valid realm paths of the same length?
//
// From a plain clone of gnolang/gno at 1f043ddb131e54e1410aebabd41d04ed49e37e1e:
//
//   cp <this file> gnovm/cmd/calibrate/zz_paygas_shape_test.go
//   go test -count=1 -run 'TestPayGasPathShapes' -v ./gnovm/cmd/calibrate/
//   go test -count=1 -run '^$' -bench 'PayGasShape' -benchtime 20000x -count 5 ./gnovm/cmd/calibrate/
//   rm gnovm/cmd/calibrate/zz_paygas_shape_test.go
//
// Every shape is checked to be a 256-byte realm path that payGas commits for,
// then timed through the same dispatch harness as BenchmarkNative_Runtime_PayGas_256.
package calibrate

import (
	"strings"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/gnovm/stdlibs"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store"
)

func shapePad(prefix, unit string, n int) string {
	return prefix + fillWorstCase(n-len(prefix), unit)
}

var payGasShapes = map[string]string{
	"SlashA":    shapePad("gno.land/r/a", "/a", 256),  // the committed bench shape
	"OneUser":   shapePad("gno.land/r/", "a", 256),    // one long user name
	"OneRepo":   shapePad("gno.land/r/a/", "a", 256),  // one long repo segment
	"Underscore": shapePad("gno.land/r/a/a", "_a", 256), // separators inside one name
	"Dash":      shapePad("gno.land/r/a/a", "-a", 256),
	"Digits":    shapePad("gno.land/r/a/a", "1", 256),
	"SlashAB1":  shapePad("gno.land/r/a", "/a_b", 256),
}

func TestPayGasPathShapes(t *testing.T) {
	for name, p := range payGasShapes {
		if len(p) != 256 || !gno.IsRealmPath(p) || strings.IndexByte(p, '#') >= 0 {
			t.Errorf("%s: %q (len %d) is not a 256-byte realm path", name, p, len(p))
		}
	}
}

func benchPayGasShape(b *testing.B, pkgPath string) {
	m := newDispatchMachine(2)
	addContextAndFrames(m, pkgPath)
	pgi := &sdk.PayGasInfo{Eligible: true}
	ctx := m.Context.(stdlibs.ExecContext)
	ctx.PayGasInfo = pgi
	ctx.GasPrice = std.GasPrice{Gas: 1000, Price: std.Coin{Denom: "ugnot", Amount: 1}}
	m.Context = ctx
	m.GasMeter = store.NewGasMeter(10_000_000)
	setBlockValueFromGo(m, 0, pkgPath)
	setBlockValueFromGo(m, 1, int64(1_000_000))
	h := &dispatchHarness{m: m, wrapper: resolveWrapper(b, "chain/runtime", "payGas"), nReturns: 0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pgi.MaxFee = 0
		h.call()
	}
	if pgi.MaxFee == 0 {
		b.Fatal("payGas did not commit")
	}
}

func BenchmarkPayGasShape_SlashA(b *testing.B)     { benchPayGasShape(b, payGasShapes["SlashA"]) }
func BenchmarkPayGasShape_OneUser(b *testing.B)    { benchPayGasShape(b, payGasShapes["OneUser"]) }
func BenchmarkPayGasShape_OneRepo(b *testing.B)    { benchPayGasShape(b, payGasShapes["OneRepo"]) }
func BenchmarkPayGasShape_Underscore(b *testing.B) { benchPayGasShape(b, payGasShapes["Underscore"]) }
func BenchmarkPayGasShape_Dash(b *testing.B)       { benchPayGasShape(b, payGasShapes["Dash"]) }
func BenchmarkPayGasShape_Digits(b *testing.B)     { benchPayGasShape(b, payGasShapes["Digits"]) }
func BenchmarkPayGasShape_SlashAB1(b *testing.B)   { benchPayGasShape(b, payGasShapes["SlashAB1"]) }
