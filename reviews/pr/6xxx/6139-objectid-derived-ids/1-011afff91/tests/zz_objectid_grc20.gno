// PKGPATH: gno.land/r/objidgrc20
package objidgrc20

import (
	"gno.land/p/demo/tokens/grc20"
	tests "gno.land/r/tests/vm"
)

type pair struct {
	A grc20.Token
	B grc20.Token
}

var p pair

func main(cur realm) {
	t1, l1 := grc20.NewToken("One", "ONE", 4, cur)
	t2, l2 := grc20.NewToken("Two", "TWO", 4, cur)
	p.A, p.B = *t1, *t2

	_ = tests.CurrentRealmPath(cross(cur))

	l1.Mint("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5", 111)
	l2.Mint("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5", 222)

	println("t1.ID   ", t1.ID())
	println("t2.ID   ", t2.ID())
	println("p.A.ID  ", p.A.ID())
	println("p.B.ID  ", p.B.ID())
	println("A==B    ", p.A.ID() == p.B.ID())
	println("A supply", p.A.TotalSupply(), "B supply", p.B.TotalSupply())
	println("A path  ", p.A.TokenPath(), "B path", p.B.TokenPath())
}

// Output:
// x
