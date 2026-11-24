// A blank type switch variable with a clause that declares no names.
// gno run and Go reject it at typecheck ("no new variables on left side of :=");
// the filetest runner preprocesses it anyway. Head panics in defineFauxCopy
// with index out of range; the merge base preprocesses and runs it.
package main

func main() {
	var i any = 42
	switch _ := i.(type) {
	case int:
	}
	println("done")
}
