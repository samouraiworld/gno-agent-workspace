// Repro, from a plain clone of github.com/gnolang/gno:
//
//	mkdir -p examples/gno.land/r/demo/grc20container
//	printf 'module = "gno.land/r/demo/grc20container"\ngno = "0.9"\n' \
//	    > examples/gno.land/r/demo/grc20container/gnomod.toml
//	cp token_id_container_filetest.gno examples/gno.land/r/demo/grc20container/
//	cd examples/gno.land/r/demo/grc20container
//	GNOROOT=$(git rev-parse --show-toplevel) go run ../../../../../gnovm/cmd/gno test -v .
//
// Two tokens with different symbols report the same Token.ID() when they are
// held as fields of one struct or as elements of one array, because
// runtime.ObjectID resolves a pointer or a slice to its container object. It
// fails at the reviewed head on the two "differ" lines below.

// PKGPATH: gno.land/r/demo/grc20container
package grc20container

import "gno.land/p/demo/tokens/grc20"

type Pair struct {
	A grc20.Token
	B grc20.Token
}

var (
	origA, origB *grc20.Token
	pair         Pair
	arr          [2]grc20.Token
	varA, varB   grc20.Token
)

func init(cur realm) {
	origA, _ = grc20.NewToken("Alpha", "ALPHA", 4, cur)
	origB, _ = grc20.NewToken("Beta", "BETA", 4, cur)

	pair.A, pair.B = *origA, *origB
	arr[0], arr[1] = *origA, *origB
	varA, varB = *origA, *origB
}

func main(cur realm) {
	// The two ledgers the events are emitted from. These are the shape
	// NewToken hands back and they are correctly distinct.
	println("pointers differ:", origA.ID() != origB.ID())

	// One copy per package-level var: also distinct, which is the case
	// NewToken's doc comment describes.
	println("separate vars differ:", varA.ID() != varB.ID())

	// Two copies inside one container. ObjectID resolves each field pointer
	// to the Pair, and each array element to the array, so both tokens answer
	// with the container's address and the identifier stops naming a ledger.
	println("struct fields differ:", pair.A.ID() != pair.B.ID())
	println("array elements differ:", arr[0].ID() != arr[1].ID())
}

// Output:
// pointers differ: true
// separate vars differ: true
// struct fields differ: true
// array elements differ: true
