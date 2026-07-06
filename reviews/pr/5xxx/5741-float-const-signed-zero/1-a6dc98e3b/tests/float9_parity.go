// Go oracle for float9.gno: run under real Go and diff its stdout against the
// gno filetest's `// Output:` block. Confirms the signed-zero matrix matches Go
// line for line at a6dc98e3b.
//
//   go run float9_parity.go
//
// Output (identical to float9.gno):
//   var float64: +0
//   var float32: +0
//   conv float32: +0
//   conv typed const: +0
//   interface: +0
//   runtime narrowing: -0
//   runtime underflow: -0
//   nonzero conv: true true
//
// Overflow cases are compile errors in Go, so they can't live in this runnable
// file. Ran separately, float32 const overflow under real Go vs gno at a6dc98e3b:
//
//   float32(1e39)     untyped   Go: reject   gno: reject   (float11)
//   float32(-1e39)    untyped   Go: reject   gno: reject
//   float32(big=1e39) typed     Go: reject   gno: reject   (float12)
//   float32(big=-1e39) typed    Go: reject   gno: -Inf     <- only divergence
//
// Go message for the last case, matching gno's go/types checker but not its VM:
//   cannot convert big (constant -1e+39 of type float64) to type float32

package main

import "math"

var m = -1e-10000

const tiny float64 = -1e-50

func zero(f float64) string {
	if math.Signbit(f) {
		return "-0"
	}
	return "+0"
}

func main() {
	println("var float64:", zero(m))
	var f32 float32 = -1e-50
	println("var float32:", zero(float64(f32)))
	println("conv float32:", zero(float64(float32(-1e-50))))
	println("conv typed const:", zero(float64(float32(tiny))))
	var i interface{} = -1e-10000
	println("interface:", zero(i.(float64)))
	d := -math.SmallestNonzeroFloat64
	println("runtime narrowing:", zero(float64(float32(d))))
	println("runtime underflow:", zero(d/3))
	var v float32 = 0.3
	println("nonzero conv:", float32(0.3) == v, float32(tiny) == 0)
}
