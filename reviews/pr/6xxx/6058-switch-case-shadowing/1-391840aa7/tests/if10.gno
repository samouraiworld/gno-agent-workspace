package main

// An `else if` opens its own faux block, so its init statement's name may be
// shadowed in that branch, and a branch of the outer `if` may shadow the outer
// init name from inside the `else`.
func main() {
	if v := 1; v == 0 {
		println("no")
	} else if v := "two"; v == "two" {
		v := 3
		println("elseif", v)
	} else {
		println("no", v)
	}

	if v := 1; v == 0 {
		println("no")
	} else if v == 1 {
		v := "inner"
		println("chain", v)
	}
}

// Output:
// elseif 3
// chain inner
