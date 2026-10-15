// Repro (gnolang/gno#6214 at 58b2692067d6545410b99c71f33bdf506caadaa0), from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6214/head && git checkout 58b2692067d6545410b99c71f33bdf506caadaa0
//   cp <this file> gnovm/tests/files/zrealm_reflect_conversion.gno
//   go test ./gnovm/pkg/gnolang -run 'TestFiles/zrealm_reflect_conversion.gno' -count=1
//
// Expected Output below is what reflect.Object's doc promises ("Two Objects
// are equal if and only if they name the same object", "safe as a map key").
// At 58b269206 the test fails: Of(p) and Of((*Mine)(p)) carry the same id and
// address but different typ, because declaredTypeName reads the caller's
// static pointer type rather than the object, so the two Objects compare
// unequal, a map keyed by Object holds two entries for one object, and Type()
// reports whatever type the caller converted to.

// PKGPATH: gno.land/r/demo/reflectconv
package reflectconv

import (
	"chain/reflect"
	"strings"
)

type Token struct{ n int }

// Same underlying type as Token, so *Token converts to *Alias.
type Alias Token

// A type declared here over a type declared in another package.
type Mine strings.Builder

var (
	tok = &Token{n: 1}
	b   = &strings.Builder{}
)

func main(cur realm) {
	o1, _ := reflect.Of(tok)
	o2, _ := reflect.Of((*Alias)(tok))
	println("same object equal:", o1 == o2)
	seen := map[reflect.Object]int{}
	seen[o1]++
	seen[o2]++
	println("map entries for one object:", len(seen))
	println("type via alias:", o2.Type())

	x1, _ := reflect.Of(b)
	x2, _ := reflect.Of((*Mine)(b))
	println("cross-package equal:", x1 == x2)
	println("type via Mine:", x2.Type())
}

// Output:
// same object equal: true
// map entries for one object: 1
// type via alias: gno.land/r/demo/reflectconv.Token
// cross-package equal: true
// type via Mine: strings.Builder
