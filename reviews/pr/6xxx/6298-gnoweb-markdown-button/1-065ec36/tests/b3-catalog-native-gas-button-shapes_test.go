// gnolang/gno#6298 at 065ec369b: does the new <gno-button escape in the
// chain/markdown native push EscapeBlockHazards(Rich) past the shape its gas
// row in gnovm/stdlibs/native_gas.go was fitted on (`[a]: u\n(x\n`,
// Base 136, Slope 27361/1024 ns per input byte)? native_gas.go is not in the diff.
//
// Repro from a plain clone (Go 1.25.9):
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6298/head && git checkout --detach 065ec369b321c207815f82b0c1dbf2b0f9f10b01
//   cp <this file> gnovm/cmd/calibrate/zz_b3_button_bench_test.go
//   cd gnovm/cmd/calibrate && go test -run '^$' -bench 'BenchmarkB3Button/EscapeBlockHazards/(ref|combo_lines|btn_lines)/100000' -count=4 -benchtime=300ms .
//   rm zz_b3_button_bench_test.go
//
// Measured (linux/amd64, 16 threads), ns per input byte at n=100000:
//   EscapeBlockHazards      ref 27.6-30.1  combo_lines 25.5-27.2  btn_lines 11.7-11.9
//   EscapeBlockHazardsRich  ref 29.6-42.8  combo_lines 28.5-35.4  btn_lines 12.2-18.8 (noisy)
// No button shape exceeds the fitted worst case on the same machine: refuted.
package calibrate

import (
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

// Shapes fed to the EscapeBlockHazards native through the same dispatch
// harness the calibration table was fitted with. "ref" is the shape the
// native_gas.go row names as the worst case.
var b3Shapes = map[string]string{
	"ref":          "[a]: u\n(x\n",
	"btn_oneline":  "<gno-button",
	"btn_lines":    "<gno-button\n",
	"lt_oneline":   "<",
	"combo_lines":  "[a]: u\n(<gno-button\n",
	"combo_inline": "[a]: <gno-button\n(x\n",
}

func benchB3(b *testing.B, fn, shape string, n int) {
	b.Helper()
	s := fillWorstCase(n, b3Shapes[shape])
	m := newDispatchMachine(1)
	setBlockValueFromGo(m, 0, s)
	h := &dispatchHarness{m: m, wrapper: resolveWrapper(b, "chain/markdown", gno.Name(fn)), nReturns: 1}
	b.ResetTimer()
	b.SetBytes(int64(n))
	for i := 0; i < b.N; i++ {
		h.call()
	}
}

func BenchmarkB3Button(b *testing.B) {
	for _, fn := range []string{"EscapeBlockHazards", "EscapeBlockHazardsRich"} {
		for _, shape := range []string{"ref", "btn_oneline", "btn_lines", "lt_oneline", "combo_lines", "combo_inline"} {
			for _, n := range []int{10000, 100000} {
				b.Run(fn+"/"+shape+"/"+itoaB3(n), func(b *testing.B) { benchB3(b, fn, shape, n) })
			}
		}
	}
}

func itoaB3(n int) string {
	if n == 10000 {
		return "10000"
	}
	return "100000"
}
