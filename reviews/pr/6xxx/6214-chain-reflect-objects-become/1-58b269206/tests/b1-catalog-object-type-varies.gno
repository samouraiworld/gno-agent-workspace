// Repro (gnolang/gno#6214 at 58b269206), from a plain clone:
//   git -C gno fetch origin pull/6214/head && git -C gno checkout 58b269206
//   cp b1-catalog-object-type-varies.gno gno/gnovm/tests/files/zrealm_reflect_b1_typevaries.gno
//   cd gno && go test ./gnovm/pkg/gnolang -run 'TestFiles/zrealm_reflect_b1_typevaries' -v
// Claim checked: reflect.gno says "Two Objects are equal if and only if they
// name the same object" and that Object is "safe as a map key". Object.typ is
// read from the static type of the pointer the caller passes, so a Go-legal
// pointer conversion yields a second, unequal Object for the same object.
// The Output below is what the head prints; the doc predicts true/true/1.

// PKGPATH: gno.land/r/demo/reflectb1
package reflectb1

import "chain/reflect"

type T struct{ X int }

type U T

var p = &T{X: 1}

func main(cur realm) {
	a, okA := reflect.Of(p)
	b, okB := reflect.Of((*U)(p))
	c, okC := reflect.Of((*struct{ X int })(p))
	println(okA, okB, okC)
	println("same address:", a.Address() == b.Address(), a.Address() == c.Address())
	println("same id:", a.ID() == b.ID(), a.ID() == c.ID())
	println("types:", a.Type(), "|", b.Type(), "|", c.Type())
	println("a == b:", a == b, "a == c:", a == c)

	// A realm deduplicating by Object, e.g. one vote per object.
	votes := map[reflect.Object]int{}
	votes[a]++
	votes[b]++
	votes[c]++
	println("distinct keys for one object:", len(votes))
}

// Output:
// true true true
// same address: true true
// same id: true true
// types: gno.land/r/demo/reflectb1.T | gno.land/r/demo/reflectb1.U |
// a == b: false a == c: false
// distinct keys for one object: 3
