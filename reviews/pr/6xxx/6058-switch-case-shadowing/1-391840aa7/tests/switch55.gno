package main

// Leaving a clause that shadows its switch's init name, by labelled break and
// by goto: the jump must not leave the shadow's slot live.
func main() {
L:
	switch v := 1; v {
	case 1:
		v := "shadow"
		println("inner", v)
		if len(v) > 0 {
			break L
		}
		println("unreachable")
	}

	switch v := 1; v {
	case 1:
		v := 10
		if v > 5 {
			goto done
		}
		println("no")
	}
done:
	println("done")
}

// Output:
// inner shadow
// done
