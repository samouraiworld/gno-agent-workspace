package main

// Both slots of a shadowed name heap-captured in the same clause, then a
// fallthrough out of it; and the same pair inside a loop, where the faux block
// is re-created per iteration.
func main() {
	switch v := 1; v {
	case 1:
		f := func() int { return v }
		v := 99
		g := func() int { return v }
		println("outer-cap", f(), "shadow-cap", g())
		fallthrough
	case 2:
		println("next", v)
	}

	fs := []func() int{}
	for i := 0; i < 2; i++ {
		if v := i; v >= 0 {
			fs = append(fs, func() int { return v })
			v := v + 10
			fs = append(fs, func() int { return v })
		}
	}
	for _, f := range fs {
		println(f())
	}
}

// Output:
// outer-cap 1 shadow-cap 99
// next 1
// 0
// 10
// 1
// 11
