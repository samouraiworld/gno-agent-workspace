// Asserts RegisterExtension reports a kind collision for an extension type whose
// dynamic type is uncomparable. Fails at f151f475d: the `e == ext` screen is a
// runtime error before the kind check runs.
//
// Run: from a gno checkout:
//   gh pr checkout 6072 -R gnolang/gno && git checkout f151f475d
//   cp <this-file> examples/gno.land/p/demo/tokens/grc721/zz_ext_uncomparable_test.gno
//   ( cd examples && go run ../gnovm/cmd/gno test -v -run TestRegisterExtension ./gno.land/p/demo/tokens/grc721/ )
//   rm examples/gno.land/p/demo/tokens/grc721/zz_ext_uncomparable_test.gno
package grc721

import (
	"testing"

	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

// journalExtension is an ordinary hook set written on a value receiver, holding a
// slice of what it has seen. Its dynamic type is therefore uncomparable.
type journalExtension struct {
	seen []TokenID
}

func (e journalExtension) ExtensionKind() string                   { return "journal" }
func (e journalExtension) OnMint(to address, tid TokenID)           {}
func (e journalExtension) OnTransfer(from, to address, tid TokenID) {}
func (e journalExtension) OnBurn(tid TokenID)                       {}

// RegisterExtension screens duplicates with `e == ext`, an interface comparison.
// Go only compares the dynamic values when the dynamic types match, so an
// uncomparable extension type survives until a second instance of that same type
// arrives — at which point the comparison is a runtime error, and the issuer gets
// a VM abort instead of the documented kind-collision panic.
func TestRegisterExtensionRejectsDuplicateKindOfUncomparableType(cur realm, t *testing.T) {
	_, led := newTestToken("Foo", "FOO", 0, cur)
	led.RegisterExtension(journalExtension{seen: []TokenID{"1"}})

	uassert.PanicsContains(t, cur, "grc721: extension kind already registered: journal", func() {
		led.RegisterExtension(journalExtension{seen: []TokenID{"2"}})
	})
}

// A single uncomparable extension, and extensions of differing dynamic types,
// are unaffected: the comparison short-circuits on the type mismatch.
func TestRegisterExtensionUncomparableTypesOfDifferentKindsAreFine(cur realm, t *testing.T) {
	_, led := newTestToken("Foo", "FOO", 0, cur)
	led.RegisterExtension(journalExtension{})
	uassert.NotPanics(t, cur, func() {
		led.RegisterExtension(&mockExtension{})
	})
	urequire.NoError(t, led.Mint(alice, "1"))
}
