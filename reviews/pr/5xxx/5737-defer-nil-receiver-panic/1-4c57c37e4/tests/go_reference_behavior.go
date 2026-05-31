// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: establishes the Go reference semantics PR #5737 claims to match.
go run reviews/pr/5xxx/5737-defer-nil-receiver-panic/1-4c57c37e4/tests/go_reference_behavior.go
*/

// Ground-truth Go behavior for a value-receiver method bound to a nil pointer,
// across four binding sites. PR #5737 makes the GnoVM defer the nil-deref to
// CALL time for *all* of them (the VPDerefValMethod path). Go only defers for
// the interface-boxed cases; for a concrete *T it derefs EAGERLY when the
// method value is formed.
package main

import "fmt"

type I interface{ M() string }

type T struct{ x int }

func (T) M() string { return "called" } // value receiver

var pt *T

// A: defer pt.M() — concrete nil *T.
func deferConcrete() (r int) {
	defer func() { recover() }()
	defer pt.M()
	r = 1
	return
}

// B: defer i.M() — interface holding typed-nil *T (the PR's filetest shape).
func deferIface() (r int) {
	defer func() { recover() }()
	var i I = pt
	defer i.M()
	r = 1
	return
}

func catch(label string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%-28s PANIC: %v\n", label, r)
		}
	}()
	fn()
}

func main() {
	fmt.Printf("A deferConcrete()  = %d   (Go: 0 => eager panic at defer-stmt)\n", deferConcrete())
	fmt.Printf("B deferIface()     = %d   (Go: 1 => panic at deferred-call time)\n", deferIface())

	// C: concrete method-value assignment, then call.
	catch("C g := pt.M (concrete)", func() {
		g := pt.M
		fmt.Printf("C g := pt.M (concrete)       bound OK (no eager panic)\n")
		_ = g
		g()
		fmt.Printf("C g := pt.M (concrete)       g() returned, no panic\n")
	})

	// D: interface method-value assignment, then call.
	catch("D g := i.M (interface)", func() {
		var i I = pt
		g := i.M
		fmt.Printf("D g := i.M (interface)       bound OK (no eager panic)\n")
		_ = g
		s := g()
		fmt.Printf("D g := i.M (interface)       g() returned %q, no panic\n", s)
	})
}
