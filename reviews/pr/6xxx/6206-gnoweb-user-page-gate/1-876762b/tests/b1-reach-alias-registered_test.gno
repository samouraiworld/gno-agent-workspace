// b1-reach-alias-registered_test.gno — PR 6206, head 876762bdf2ea6635b27e2b0a42f26f9fdc54af24.
//
// The chain-side half of the alias finding: after a rename, the old name is
// still a live, permanently-held registration, and ResolveName reports it with
// data != nil and isCurrent == false -- a second return value byte-identical to
// a name that was never registered at all. gnoweb's userExists()
// (gno.land/pkg/gnoweb/handler_http.go:566-575) reads only that second value.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	make -C gnovm build           # or use any gno binary built from this tree
//	cp <this file> examples/gno.land/r/sys/users/b1_reach_alias_registered_test.gno
//	cd examples/gno.land/r/sys/users
//	gno test . -run TestB1ReachAliasStaysRegistered -v
//
// Observed at head: --- PASS: TestB1ReachAliasStaysRegistered (0.00s), GAS 3355729.
// Every assertion below holds, including uassert.False(t, oldIsCurrent) for a
// name whose owner is very much alive.
package users

import (
	"testing"

	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestB1ReachAliasStaysRegistered(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm(initControllerPath))
	cleanStore(t)

	urequire.NoError(t, RegisterUser(cross(cur), alice, aliceAddr))
	data, _ := ResolveName(alice)
	urequire.NoError(t, data.UpdateName(0, cur, "alicerenamed"))

	// The old name still resolves to the same account.
	old, oldIsCurrent := ResolveName(alice)
	urequire.NotEqual(t, nil, old)
	uassert.Equal(t, aliceAddr, old.Addr())
	uassert.Equal(t, "alicerenamed", old.Name())
	uassert.False(t, oldIsCurrent) // the only value gnoweb reads

	// A name nobody ever took produces the same second return value.
	gone, goneIsCurrent := ResolveName("neverregistered")
	uassert.Equal(t, nil, gone)
	uassert.False(t, goneIsCurrent)
}
