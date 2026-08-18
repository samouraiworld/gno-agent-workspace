// Every form PR 5981 newly rejects, written in Go and run by the Go compiler.
//
// Run: needs only a Go toolchain, no gno clone:
//
//	mkdir -p /tmp/p && cp <this file> /tmp/p/main.go && cd /tmp/p
//	printf 'module p\n\ngo 1.25\n' > go.mod && go run .
//
// Go compiles and runs all nine; gno at 055d85cbc rejects all nine at the
// declaration. The divergence is deliberate, per issue 5876.
package main

type T int

func fParamUnused(iota int)    { println("param_unused ok") }
func fParamSecond(a, iota int) { println("param_second ok") }
func fResultNamed() (iota int) { return 5 }
func (iota T) M()              { println("recv ok") }

func main() {
	for iota := 0; iota < 2; iota++ {
		println("forinit", iota)
	}
	for iota := 0; ; {
		println("forinit_nocond", iota)
		break
	}
	for iota, j := 0, 1; iota < 1; iota++ {
		println("forinit_multi", iota+j)
	}
	for iota := 0; ; {
		_ = iota
		println("forinit_unused ok")
		break
	}
	g := func(iota int) { println("funclit_param ok") }
	g(1)
	fParamUnused(1)
	fParamSecond(1, 2)
	println("result_named", fResultNamed())
	var t T
	t.M()
}
