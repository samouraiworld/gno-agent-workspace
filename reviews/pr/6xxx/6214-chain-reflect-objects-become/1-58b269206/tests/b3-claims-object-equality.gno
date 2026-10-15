// Repro (gnolang/gno#6214 at 58b2692067d6545410b99c71f33bdf506caadaa0):
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 58b2692067d6545410b99c71f33bdf506caadaa0
//   cp <this file> gnovm/tests/files/zrealm_b3claims_equality.gno
//   cd gnovm/pkg/gnolang && go test -run 'TestFiles/zrealm_b3claims_equality' -count=1 .
//
// Claim checked: docs/resources/gno-stdlibs.md "Opaque, comparable and safe as
// a map key", the ADR's "It stays comparable and map-key safe", and reflect.gno's
// "Two Objects are equal if and only if they name the same object".
// Observed: one persisted object, reached through *Token, *Alias and
// *struct{ n int }, yields three Objects with one address but three Type()
// values, so they compare unequal and take three map slots. The Output block
// below is what the head prints; a fix flips "a == b" and "a == c" to true and
// "map entries for one object" to 1.

// PKGPATH: gno.land/r/demo/b3eq
package b3eq

import (
	"chain/reflect"
)

type Token struct{ n int }
type Alias Token

var tok = &Token{n: 1}

func main(cur realm) {
	a, aok := reflect.Of(tok)
	b, bok := reflect.Of((*Alias)(tok))
	c, cok := reflect.Of((*struct{ n int })(tok))
	println("ok:", aok, bok, cok)
	println("same address:", a.Address() == b.Address(), a.Address() == c.Address())
	println("a == b:", a == b, "a == c:", a == c)
	println("types:", a.Type(), "|", b.Type(), "|", c.Type())
	seen := map[reflect.Object]bool{}
	seen[a] = true
	seen[b] = true
	seen[c] = true
	println("map entries for one object:", len(seen))
}

// Output:
// ok: true true true
// same address: true true
// a == b: false a == c: false
// types: gno.land/r/demo/b3eq.Token | gno.land/r/demo/b3eq.Alias |
// map entries for one object: 3
