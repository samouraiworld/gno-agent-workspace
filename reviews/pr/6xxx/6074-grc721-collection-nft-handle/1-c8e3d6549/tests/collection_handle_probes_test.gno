/* Run: from a gno checkout, green at c8e3d6549; the three logs are the findings:
gh pr checkout 6074 -R gnolang/gno && git checkout c8e3d6549
P=examples/gno.land/p/demo/tokens/grc721/collection
curl -fsSL -o $P/zprobe_test.gno https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6074-grc721-collection-nft-handle/1-c8e3d6549/tests/collection_handle_probes_test.gno
cd $P && gno test -v -run ZMine .
cd - && rm $P/zprobe_test.gno
*/

package collection

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/royalty"
	"gno.land/p/nt/seqid/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func mkToken(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))
	return
}

type forgedRoyalty struct{ id string }

func (f *forgedRoyalty) ExtensionKind() string { return royalty.Kind }
func (f *forgedRoyalty) TokenID() string       { return f.id }

// Wrap pairs any token with any ledger.
func TestZMineWrapMismatch(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	tokA, _ := mkToken("A", "AAA", 0, cur)
	tokB, ledB := mkToken("B", "BBB", 1, cur)

	nft := Wrap(tokA, ledB)
	urequire.NoError(t, nft.Mint(alice, "1"))

	t.Log("published token supply:", nft.Token().TotalSupply(), "| the ledger's own token supply:", tokB.TotalSupply())
	_, err := nft.Collection().Token().OwnerOf("1")
	t.Log("the published collection on the token it just minted:", err)
}

// Token.ID() is not unique, so the attach guard compares a string two tokens can share.
func TestZMineAttachIDCollision(cur realm, t *testing.T) {
	artist := testutils.TestAddress("artist")

	tokA, ledA := mkToken("Collection A", "FOO", 7, cur)
	tokB, ledB := mkToken("Collection B", "FOO", 7, cur)
	t.Log("two independent ledgers, one id:", tokA.ID(), "==", tokB.ID(), tokA.ID() == tokB.ID())

	royB, royBLedger := royalty.NewRoyalty(ledB, 10000)
	urequire.NoError(t, royBLedger.SetDefaultRoyalty(artist, 1000))

	nftA := Wrap(tokA, ledA)
	nftA.Attach(royB) // B's view, A's collection
	v, ok := nftA.Collection().Extension(royalty.Kind)
	t.Log("attached to A:", ok, "| is it B's view:", v.(*royalty.Royalty) == royB)
}

// A holder of a published Collection can build a lookalike around the genuine token.
func TestZMineForgedCollection(cur realm, t *testing.T) {
	artist := testutils.TestAddress("artist")

	tok, led := mkToken("Real", "REAL", 0, cur)
	nft := Wrap(tok, led)
	roy, royLedger := royalty.NewRoyalty(led, 10000)
	urequire.NoError(t, royLedger.SetDefaultRoyalty(artist, 1000))
	nft.Attach(roy)
	published := nft.Collection()

	fake := NewCollection(published.Token(), &forgedRoyalty{id: published.Token().ID()})
	t.Log("same token pointer:", fake.Token() == published.Token(), "| fake advertises royalty:", fake.HasExtension(royalty.Kind))

	v, _ := fake.Extension(royalty.Kind)
	_, isReal := v.(*royalty.Royalty)
	t.Log("the assert is the only difference, real royalty type on the forgery:", isReal)
	uassert.False(t, isReal)
}
