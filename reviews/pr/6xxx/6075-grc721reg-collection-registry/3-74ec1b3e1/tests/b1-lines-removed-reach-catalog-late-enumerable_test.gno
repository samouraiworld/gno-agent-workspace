// Repro: NewEnumerable attached after the first mint registers its view with
// the core, and the registry now publishes that view to every consumer, but
// the index never saw the earlier mints: the published enumerable view answers
// TotalSupply 0 and TokenByIndex out of range while the core reports the
// supply, and a later transfer leaves the owner index ahead of the global one.
// Nothing refuses the late attach; the doc's "attach it before the first mint"
// is the only guard.
//
// From a plain clone of gnolang/gno at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7:
//
//   cp <this file> examples/gno.land/r/demo/defi/grc721reg/zz_late_enumerable_test.gno
//   cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/demo/defi/grc721reg -run 'TestLateEnumerable' -v
//
// Asserts that the published view agrees with the core (or that the late
// attach is refused); fails on head with the observed figures.
package grc721reg

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/enumerable"
	"gno.land/p/nt/testutils/v0"
)

func TestLateEnumerableViewDisagreesWithCore(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_late_enum"))
	token, ledger := grc721.NewToken("Late", "LATE", 0, cur)
	bob := testutils.TestAddress("bob")
	carol := testutils.TestAddress("carol")

	ledger.Mint(bob, "1")
	ledger.Mint(bob, "2")

	refused := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				refused = true
			}
		}()
		enumerable.NewEnumerable(ledger)
	}()
	if refused {
		return
	}

	key := Register(cross(cur), token, "")
	v, ok := GetExtension(key, enumerable.Kind)
	if !ok {
		t.Fatalf("enumerable view not published")
	}
	en := v.(*enumerable.Enumerable)

	ledger.TransferFrom(bob, bob, carol, "1")

	_, errIdx := en.TokenByIndex(0)
	ownTid, errOwn := en.TokenOfOwnerByIndex(carol, 0)
	_, errBob := en.TokenOfOwnerByIndex(bob, 0)
	println("core TotalSupply:", token.TotalSupply(), "| published enumerable TotalSupply:", en.TotalSupply())
	println("TokenByIndex(0) err:", errIdx != nil, "| TokenOfOwnerByIndex(carol,0):", string(ownTid), errOwn == nil, "| TokenOfOwnerByIndex(bob,0) err:", errBob != nil)

	if en.TotalSupply() != token.TotalSupply() {
		t.Errorf("published enumerable view reports TotalSupply %d, core reports %d", en.TotalSupply(), token.TotalSupply())
	}
}
