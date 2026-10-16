// solo-finder-2: `:=` redeclaration to a type differing only in tags (or
// variadicity) is accepted, and the variable's static type silently changes.
//
// Repro from a plain clone of gnolang/gno at b42a521b16655374e507bdb949ab5b8289bc8404:
//   gno run main.gno           (this file)
// Observed at head: prints "2 1" (accepted).
// Go: cannot use struct{A int "b"}{…} as struct{A int "a"} value in assignment.
// Variadic variant (same head):
//   f := func(xs []int) {}; y, f := 1, func(xs ...int) {}; f([]int{7, 8, 9})
//   -> accepted redeclaration, then "cannot use []int as int" at the call:
//      f's static type became func(...int) mid-scope.
// Non-variant control: x := 1; y, x := 1, "s"
//   -> "StaticBlock.Define2(x) cannot change .T; was int, new string"
// Cause: nodes.go:2389 `tv.T.TypeID() != old.T.TypeID()`, the sibling of the
// "Redefinition cannot change .T" check the diff moved to identicalTypes.
package main

func main() {
	x := struct {
		A int "a"
	}{}
	y, x := 1, struct {
		A int "b"
	}{2}
	println(x.A, y)
}
