// Asserts the statements that declare a name through an AssignStmt (if-init,
// switch-init, a nested for-init, a func literal's body) read the enclosing loop's
// variable in their own clause and the new name after it, as Go does.
// Passes at head 1de7f587d; expected values are Go's scoping rules for each line.
// cp <this file> gnovm/tests/files/zz_judge_siblings.gno
// cd gnovm/pkg/gnolang && go test -short -run 'TestFiles/zz_judge_siblings.gno' .
package main

func main() {
	for x := 1; x < 2; x++ {
		if x := x + 10; x > 0 {
			println("if", x)
		}
		switch x := x + 20; x {
		default:
			println("switch", x)
		}
		g := func() int { x := x + 30; return x }
		println("funclit", g())
		var fs []func() int
		for x := x + 40; x < 43; x++ {
			fs = append(fs, func() int { return x })
		}
		for _, f := range fs {
			println("closure", f())
		}
		println("loop", x)
	}
}

// Output:
// if 11
// switch 21
// funclit 31
// closure 41
// closure 42
// loop 1
