package main

type I interface{ M() string }

type A struct{}

func (A) M() string { return "A.M" }

type B struct{}

func (B) M() string { return "B.M" }

type R struct{}

func (R) M() string { return "R.M" }

type Z struct{ R }

type S struct {
	A
	B
	Z
}

type T struct {
	S
	R
}

func main() {
	var i I = T{}
	println(i.M())
}

// Output:
// R.M
