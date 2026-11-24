// Repro from a plain clone of gnolang/gno at a40c1fbf7 (PR 6058 head):
//   go install ./gnovm/cmd/gno && cp solo-finder-const-shadow-paths.gno /tmp/p.gno && cd /tmp && gno run p.gno
// Compare: cp solo-finder-const-shadow-paths.gno /tmp/p.go && go run /tmp/p.go (outputs must match).
// GetIsConstAt with a const shadow in a faux case block: use and assignment before, closure before/after, nested if, inner block, fallthrough. Head output matched Go line for line.

package main

func main() {
	switch v := 1; v {
	case 1:
		f := func() int { return v }
		v = 2
		const v = "c"
		println("closure-before", f(), v)
		g := func() string { return v }
		println("closure-after", g())
		if x := 0; x == 0 {
			println("nested-if", v)
		}
		{
			println("inner-before", v)
			const v = 7
			println("inner-after", v)
		}
		fallthrough
	case 2:
		println("fell", v)
		v = 9
		println("fell2", v)
	}
}
