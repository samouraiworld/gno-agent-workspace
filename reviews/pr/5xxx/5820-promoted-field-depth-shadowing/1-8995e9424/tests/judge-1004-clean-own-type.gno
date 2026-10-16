// Asserts: T{Amb; Clean}, Amb{A; B}, Clean{C}, with A, B and C each declaring M,
// is rejected, since Go finds three M at depth 2 (ambiguous selector T.M).
// Head 8995e9424 and base c775354dd both accept it and run C.M: the
// embed_method_shadow_internal_ambiguous fixture rejects only because Clean embeds A.
package main

type I interface{ M() string }

type A struct{}

func (A) M() string { return "A" }

type B struct{}

func (B) M() string { return "B" }

type C struct{}

func (C) M() string { return "C" }

type Amb struct {
	A
	B
}

type Clean struct{ C }

type T struct {
	Amb
	Clean
}

func main() {
	var i I = T{}
	println(i.M())
}

// Error:
// main/judge-1004-clean-own-type.gno:35:6-15: main.T does not implement main.I (missing method M)
