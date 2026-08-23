package main

// A case block wider than nameIndexThreshold (32) resolves names through
// StaticBlock.nameIndex instead of the linear scan, so last-wins has to hold
// there too. The first switch declares the shadow after the map exists
// (incremental insert); the second declares it before, so buildNameIndex sees
// both slots and must keep the later one.
func main() {
	switch v := 1; v {
	case 1:
		a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 := 0, 1, 2, 3, 4, 5, 6, 7, 8, 9
		b0, b1, b2, b3, b4, b5, b6, b7, b8, b9 := 0, 1, 2, 3, 4, 5, 6, 7, 8, 9
		c0, c1, c2, c3, c4, c5, c6, c7, c8, c9 := 0, 1, 2, 3, 4, 5, 6, 7, 8, 9
		d0, d1, d2 := 0, 1, 2
		println(a0 + a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + a9 + b0 + b1 + b2 + b3 + b4 + b5 + b6 + b7 + b8 + b9 +
			c0 + c1 + c2 + c3 + c4 + c5 + c6 + c7 + c8 + c9 + d0 + d1 + d2)
		v := "late"
		println("late", v)
	}

	switch v := 1; v {
	case 1:
		v := "early"
		a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 := 0, 1, 2, 3, 4, 5, 6, 7, 8, 9
		b0, b1, b2, b3, b4, b5, b6, b7, b8, b9 := 0, 1, 2, 3, 4, 5, 6, 7, 8, 9
		c0, c1, c2, c3, c4, c5, c6, c7, c8, c9 := 0, 1, 2, 3, 4, 5, 6, 7, 8, 9
		d0, d1, d2 := 0, 1, 2
		println(a0 + a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + a9 + b0 + b1 + b2 + b3 + b4 + b5 + b6 + b7 + b8 + b9 +
			c0 + c1 + c2 + c3 + c4 + c5 + c6 + c7 + c8 + c9 + d0 + d1 + d2)
		println("early", v)
	}
}

// Output:
// 138
// late late
// 138
// early early
