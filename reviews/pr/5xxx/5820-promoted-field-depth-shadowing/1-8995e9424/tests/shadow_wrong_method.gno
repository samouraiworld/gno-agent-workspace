package main

type I interface{ M() string }

type C struct{ V string }

func (C) M() string { return "C.M" }

type R struct{ V string }

func (R) M() string { return "R.M" }

type X struct{ R }

type S struct {
	C
	X
}

type T struct {
	S
	R
}

func main() {
	t := T{S{C{"c"}, X{R{"inner"}}}, R{"outer"}}
	var i I = t
	println(i.M(), t.V)
}

// Output:
// R.M outer
