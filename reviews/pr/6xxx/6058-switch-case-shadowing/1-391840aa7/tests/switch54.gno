package main

// Shadowing a switch or if init name from a default clause, and with two init
// names of which only one is shadowed.
func main() {
	switch v := 1; v {
	case 9:
		println("no")
	default:
		println("pre", v)
		v := "d"
		println("post", v)
	}

	switch a, b := 1, 2; a {
	case 1:
		println("outer", a, b)
		b := "shadow"
		println("inner", a, b)
	}

	if a, b := 1, 2; a == 1 {
		println("outer", a, b)
		b := "shadow"
		println("inner", a, b)
	}
}

// Output:
// pre 1
// post d
// outer 1 2
// inner 1 shadow
// outer 1 2
// inner 1 shadow
