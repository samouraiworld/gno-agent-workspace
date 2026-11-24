// Repro from a plain clone of gnolang/gno at a40c1fbf7 (PR 6058 head).
// gno run rejects it at typecheck ("no new variables on left side of :="), as Go does.
// The filetest runner preprocesses despite a TypeCheckError (see gnovm/tests/files/redefine6.gno),
// so as gnovm/tests/files/blank_tsvar.gno this reaches defineSwitchVar -> defineFauxCopy(numFauxCopiedNames(), "_"):
//   go test ./gnovm/pkg/gnolang -run TestFiles/blank_tsvar.gno
// expected at head: Error "faux copy slot 0 holds x, not _" (body declares x); bodyless clause: index out of range.
// NOT RUN as a filetest this round (Nit).

package main

func main() {
	var i any = 42
	switch _ := i.(type) {
	case int:
		x := 3
		println("int", x)
	}
}
