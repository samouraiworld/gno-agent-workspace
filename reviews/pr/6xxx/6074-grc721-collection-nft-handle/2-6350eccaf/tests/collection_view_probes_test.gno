/* Run: from a gno checkout, green at 6350eccaf; the logs and the two GAS lines are the findings:
gh pr checkout 6074 -R gnolang/gno && git checkout 6350eccaf
P=examples/gno.land/p/demo/tokens/grc721/collection
curl -fsSL -o $P/zprobe_test.gno https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6074-grc721-collection-nft-handle/2-6350eccaf/tests/collection_view_probes_test.gno
cd $P && gno test -v -run ZMine .
cd - && rm $P/zprobe_test.gno
*/

package collection

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/demo/tokens/grc721/royalty"
	"gno.land/p/nt/seqid/v0"
)

func mkPair(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))

	return
}

func mkNFT(name, symbol string, id seqid.ID, rlm realm) (nft *NFT, led *grc721.PrivateLedger) {
	func(cur realm) {
		nft, led = New(name, symbol, id, cur)
	}(cross(rlm))

	return
}

// probeHook takes its kind from a field, so a probe can hand it any kind.
type probeHook struct{ kind string }

func (p probeHook) ExtensionKind() string                        { return p.kind }
func (p probeHook) OnMint(to address, tid grc721.TokenID)        {}
func (p probeHook) OnTransfer(f, to address, tid grc721.TokenID) {}
func (p probeHook) OnBurn(tid grc721.TokenID)                    {}

// RegisterExtension files the view under the hook's kind and never asks the view
// its own kind.
func TestZMineViewKindIsNeverChecked(cur realm, t *testing.T) {
	_, ledA := mkPair("A", "AAA", 0, cur)
	_, ledB := mkPair("B", "BBB", 1, cur)
	royB, _ := royalty.NewRoyalty(ledB, 1000)

	ledA.RegisterExtension(probeHook{kind: metadata.Kind}, royB)

	c := NewCollection(ledA)
	v, ok := c.Extension(metadata.Kind)
	_, isMeta := v.(*metadata.Metadata)
	t.Log("A advertises metadata:", ok,
		"| asserts to *metadata.Metadata:", isMeta,
		"| is B's royalty view:", v.(*royalty.Royalty) == royB,
		"| its own kind:", v.ExtensionKind(),
		"| its own token:", v.TokenID())
}

// nor its token, which is what the guard deleted this round used to compare.
func TestZMineViewFromAnotherToken(cur realm, t *testing.T) {
	tokA, ledA := mkPair("A", "AAA", 0, cur)
	_, ledB := mkPair("B", "BBB", 1, cur)
	royB, royBLedger := royalty.NewRoyalty(ledB, 1000)

	ledA.RegisterExtension(royBLedger, royB) // B's hooks and B's view, onto A

	v, ok := NewCollection(ledA).Extension(royalty.Kind)
	t.Log("A advertises royalty:", ok,
		"| the view names token:", v.TokenID(),
		"| A's token is:", tokA.ID())
}

// NewCollection copies the views once and keeps no ledger.
func TestZMineNewCollectionIsFrozen(cur realm, t *testing.T) {
	_, led := mkPair("Foo", "FOO", 0, cur)
	c := NewCollection(led)

	metadata.NewMetadata(led)

	t.Log("NewCollection kinds:", len(c.Kinds()),
		"| ledger kinds:", len(led.ExtensionKinds()),
		"| HasExtension(metadata):", c.HasExtension(metadata.Kind))
}

// and Wrap mints a fresh one per call, so re-wrapping a persisted pair refreshes
// an object nobody published.
func TestZMineRewrapOrphansThePublished(cur realm, t *testing.T) {
	tok, led := mkPair("Foo", "FOO", 0, cur)

	published := Wrap(tok, led).Collection()
	metadata.NewMetadata(led)
	fresh := Wrap(tok, led).Collection()

	t.Log("same object:", published == fresh,
		"| published kinds:", len(published.Kinds()),
		"| fresh kinds:", len(fresh.Kinds()))
}

// NewCollection's only input check is ledger == nil.
func TestZMineNilTokenCollection(cur realm, t *testing.T) {
	zero := &grc721.PrivateLedger{} // every field unexported, so any package can build it
	metadata.NewMetadata(zero)
	royalty.NewRoyalty(zero, 10000)

	c := NewCollection(zero)
	v, ok := c.Extension(royalty.Kind)
	_, canonical := v.(*royalty.Royalty)
	t.Log("kinds:", c.Kinds(), "| Extension(royalty) ok:", ok,
		"| asserts to *royalty.Royalty:", canonical,
		"| Token() == nil:", c.Token() == nil)

	defer func() { t.Log("a consumer reading c.Token().ID():", recover()) }()
	_ = c.Token().ID()
}

// Wrap refuses the same ledger.
func TestZMineWrapRefusesIt(cur realm, t *testing.T) {
	tok, _ := mkPair("Real", "REAL", 0, cur)

	defer func() { t.Log("Wrap with that ledger:", recover()) }()
	_ = Wrap(tok, &grc721.PrivateLedger{})
}

// shifty reads its kind through a pointer, so it can re-key after registration.
type shifty struct{ kind *string }

func (s shifty) ExtensionKind() string                        { return *s.kind }
func (s shifty) OnMint(to address, tid grc721.TokenID)        {}
func (s shifty) OnTransfer(f, to address, tid grc721.TokenID) {}
func (s shifty) OnBurn(tid grc721.TokenID)                    {}

// lookupExtension re-derives the kind on every call rather than storing it.
func TestZMineKindNotFrozen(cur realm, t *testing.T) {
	_, led := mkPair("Foo", "FOO", 0, cur)

	k := "aux"
	led.RegisterExtension(shifty{kind: &k}, nil)
	t.Log("registered as:", led.ExtensionKinds()[0])

	k = metadata.Kind

	t.Log("now reports:", led.ExtensionKinds()[0],
		"| HasExtension(aux):", led.HasExtension("aux"),
		"| HasExtension(metadata):", led.HasExtension(metadata.Kind))

	defer func() { t.Log("the real metadata extension can no longer register:", recover()) }()
	metadata.NewMetadata(led)
}

// probeViewless has hooks and publishes nothing.
type probeViewless struct{}

func (probeViewless) ExtensionKind() string                        { return "probeviewless" }
func (probeViewless) OnMint(to address, tid grc721.TokenID)        {}
func (probeViewless) OnTransfer(f, to address, tid grc721.TokenID) {}
func (probeViewless) OnBurn(tid grc721.TokenID)                    {}

// the two HasExtension methods answer differently for a viewless kind.
func TestZMineHasExtensionDisagrees(cur realm, t *testing.T) {
	_, led := mkPair("Foo", "FOO", 0, cur)
	led.RegisterExtension(probeViewless{}, nil)
	c := NewCollection(led)

	t.Log("ledger.HasExtension:", led.HasExtension("probeviewless"),
		"| collection.HasExtension:", c.HasExtension("probeviewless"),
		"| ledger kinds:", len(led.ExtensionKinds()),
		"| collection kinds:", len(c.Kinds()))

	defer func() { t.Log("registering it again:", recover()) }()
	led.RegisterExtension(probeViewless{}, nil)
}

// the two kind listings this PR adds use different orders.
func TestZMineKindsOrder(cur realm, t *testing.T) {
	_, led := mkPair("Ord", "ORD", 9, cur)
	royalty.NewRoyalty(led, 1000) // registered first, sorts second
	metadata.NewMetadata(led)

	t.Log("ledger.ExtensionKinds():", led.ExtensionKinds(),
		"| collection.Kinds():", NewCollection(led).Kinds())
}

// 100 lookups through NFT.Extension, which re-syncs on every call.
func TestZMineLookupThroughNFT(cur realm, t *testing.T) {
	nft, led := mkNFT("Foo", "FOO", 0, cur)
	metadata.NewMetadata(led)
	royalty.NewRoyalty(led, 1000)

	for i := 0; i < 100; i++ {
		nft.Extension(metadata.Kind)
	}
}

// the same 100 lookups on the *Collection it returns.
func TestZMineLookupThroughCollection(cur realm, t *testing.T) {
	nft, led := mkNFT("Foo", "FOO", 0, cur)
	metadata.NewMetadata(led)
	royalty.NewRoyalty(led, 1000)

	col := nft.Collection()
	for i := 0; i < 100; i++ {
		col.Extension(metadata.Kind)
	}
}
