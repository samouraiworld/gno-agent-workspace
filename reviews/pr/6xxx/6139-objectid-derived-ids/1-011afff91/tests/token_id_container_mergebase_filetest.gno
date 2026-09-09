// Repro, from a plain clone of github.com/gnolang/gno at the merge base
// d4bb7ab93:
//
//	git checkout d4bb7ab93
//	mkdir -p examples/gno.land/r/demo/grc20container
//	printf 'module = "gno.land/r/demo/grc20container"\ngno = "0.9"\n' \
//	    > examples/gno.land/r/demo/grc20container/gnomod.toml
//	cp token_id_container_mergebase_filetest.gno \
//	    examples/gno.land/r/demo/grc20container/
//	cd examples/gno.land/r/demo/grc20container
//	GNOROOT=$(git rev-parse --show-toplevel) go run ../../../../../gnovm/cmd/gno test -v .
//
// The baseline for token_id_container_filetest.gno: the same four shapes at the
// merge base, with the base's id seqid.ID argument fed from one seqid.ID as its
// doc comment prescribes. All four answer true, so the branch is what turns a
// distinguishable pair into an indistinguishable one. It passes at d4bb7ab93.

// PKGPATH: gno.land/r/demo/grc20container
package grc20container

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/p/nt/seqid/v0"
)

type Pair struct {
	A grc20.Token
	B grc20.Token
}

var (
	nextTokenID  seqid.ID
	origA, origB *grc20.Token
	pair         Pair
	arr          [2]grc20.Token
	varA, varB   grc20.Token
)

func init(cur realm) {
	origA, _ = grc20.NewToken("Alpha", "ALPHA", 4, nextTokenID.Next(), cur)
	origB, _ = grc20.NewToken("Beta", "BETA", 4, nextTokenID.Next(), cur)

	pair.A, pair.B = *origA, *origB
	arr[0], arr[1] = *origA, *origB
	varA, varB = *origA, *origB
}

func main(cur realm) {
	println("pointers differ:", origA.ID() != origB.ID())
	println("separate vars differ:", varA.ID() != varB.ID())
	println("struct fields differ:", pair.A.ID() != pair.B.ID())
	println("array elements differ:", arr[0].ID() != arr[1].ID())
	println("A id:", pair.A.ID())
	println("B id:", pair.B.ID())
}

// Output:
// pointers differ: true
// separate vars differ: true
// struct fields differ: true
// array elements differ: true
// A id: gno.land/r/demo/grc20container.ALPHA.0000001
// B id: gno.land/r/demo/grc20container.BETA.0000002
