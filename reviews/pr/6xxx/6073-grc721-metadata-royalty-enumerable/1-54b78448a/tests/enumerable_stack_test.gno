/* Run: from a gno checkout:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno enumerable_stack_test.gno; do
  curl -fsSL -o $P/$f https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6073-grc721-metadata-royalty-enumerable/1-54b78448a/tests/$f
done
cd $P && gno test -v .
cd - && rm -r $P
*/

package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721/enumerable"
	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/demo/tokens/grc721/royalty"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

// All three attached to one ledger, then a full mint/transfer/burn/re-mint cycle.
func TestStackAllThree(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	creator := testutils.TestAddress("creator")

	tok, coreLedger := newCore("Foo", "FOO", 0, cur)
	enum, _ := enumerable.NewEnumerable(coreLedger)
	meta, mled := metadata.NewMetadata(coreLedger)
	roy, rled := royalty.NewRoyalty(coreLedger, 1000)

	urequire.NoError(t, coreLedger.Mint(alice, "1"))
	urequire.NoError(t, coreLedger.Mint(alice, "2"))
	urequire.NoError(t, mled.SetTokenURI("1", "ipfs://one"))
	urequire.NoError(t, rled.SetDefaultRoyalty(creator, 250))
	urequire.NoError(t, rled.SetTokenRoyalty("1", creator, 500))

	urequire.NoError(t, coreLedger.TransferFrom(alice, alice, bob, "1"))
	uri, _ := meta.TokenURI("1")
	info, ok := roy.TokenRoyalty("1")
	t.Log("after transfer: uri", uri, "| royalty override", ok, info.Bps, "| enum supply", enum.TotalSupply(), "| core supply", tok.TotalSupply())

	// burn clears per-token state in every extension
	urequire.NoError(t, coreLedger.Burn("1"))
	_, uriErr := meta.TokenURI("1")
	_, royOK := roy.TokenRoyalty("1")
	t.Log("after burn: uri err", uriErr, "| royalty override", royOK, "| enum supply", enum.TotalSupply())

	// re-mint the same id to a different holder
	urequire.NoError(t, coreLedger.Mint(bob, "1"))
	_, uriErr = meta.TokenURI("1")
	_, amt, _ := roy.RoyaltyInfo("1", 10000)
	t.Log("after re-mint: uri err", uriErr, "| royalty amount", amt, "(default applies)", "| enum supply", enum.TotalSupply())

	order := []string{}
	for i := int64(0); i < enum.TotalSupply(); i++ {
		v, err := enum.TokenByIndex(i)
		urequire.NoError(t, err)
		order = append(order, v.String())
	}
	t.Log("global order after re-mint:", order)

	bobOrder := []string{}
	for i := int64(0); ; i++ {
		v, err := enum.TokenOfOwnerByIndex(bob, i)
		if err != nil {
			break
		}
		bobOrder = append(bobOrder, v.String())
	}
	t.Log("bob order:", bobOrder)
	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())
}

// A second extension of the same kind is rejected by the core.
func TestDuplicateKind(cur realm, t *testing.T) {
	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	enumerable.NewEnumerable(coreLedger)

	uassert.PanicsWithMessage(t, cur, "grc721: extension kind already registered: enumerable", func() {
		enumerable.NewEnumerable(coreLedger)
	})
}
