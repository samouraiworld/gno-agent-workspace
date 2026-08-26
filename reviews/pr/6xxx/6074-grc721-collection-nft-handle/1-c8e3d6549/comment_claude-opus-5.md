# Review: [#6074](https://github.com/gnolang/gno/pull/6074)
Posted: https://github.com/gnolang/gno/pull/6074#pullrequestreview-5029953006
Event: COMMENT

## Body
[AI review]

Automated pass over the collection package, scoped to the delta over #6073. No design judgement on the two-handle split and no merge verdict.

## examples/gno.land/p/demo/tokens/grc721/collection/nft.gno:26 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L26) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L26) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3862363696)
`Wrap` rejects nils and nothing else, so a transposed pair gives an NFT that mints into one collection and publishes the other, with no error at the wrap and none at the mint.

<details><summary>repro</summary>

`ledger.ReadToken()` is public, so the check costs one line beside the nil guard.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6074 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/collection
cat > $P/zprobe_test.gno <<'EOF'
package collection

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/seqid/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/urequire/v0"
)

func mkToken(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))

	return
}

func TestWrapPairsWhatItIsGiven(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	tokA, _ := mkToken("A", "AAA", 0, cur)
	tokB, ledB := mkToken("B", "BBB", 1, cur)

	nft := Wrap(tokA, ledB) // transposed
	urequire.NoError(t, nft.Mint(alice, "1"))

	_, err := nft.Collection().Token().OwnerOf("1")
	t.Log("published supply", nft.Token().TotalSupply(), "| ledger's own supply", tokB.TotalSupply(), "| published collection on the token it minted:", err)
}
EOF
cd $P && gno test -v -run TestWrapPairsWhatItIsGiven .
cd - && rm $P/zprobe_test.gno
```

```
=== RUN   TestWrapPairsWhatItIsGiven
published supply 0 | ledger's own supply 1 | published collection on the token it minted: invalid token id
--- PASS: TestWrapPairsWhatItIsGiven
```
</details>

## examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:35 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L35) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L35) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3862363706)
Two collections built in one realm with the same symbol and sequence id carry the same `Token.ID()`, so a view wired to the other ledger passes this check and the panic below claims an ownership the comparison never established.

<details><summary>repro</summary>

The core's own `NewToken` doc treats a repeated id as a case indexers must detect rather than one it prevents. The existing test varies both the symbol and the id, so it never meets the collision.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6074 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/collection
cat > $P/zprobe_test.gno <<'EOF'
package collection

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/royalty"
	"gno.land/p/nt/seqid/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/urequire/v0"
)

func mkToken(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))

	return
}

func TestAttachGuardComparesAnIDTwoTokensShare(cur realm, t *testing.T) {
	artist := testutils.TestAddress("artist")

	tokA, ledA := mkToken("Collection A", "FOO", 7, cur)
	tokB, ledB := mkToken("Collection B", "FOO", 7, cur)
	t.Log("one id, two ledgers:", tokA.ID(), tokA.ID() == tokB.ID())

	royB, royBLedger := royalty.NewRoyalty(ledB, 10000)
	urequire.NoError(t, royBLedger.SetDefaultRoyalty(artist, 1000))

	nftA := Wrap(tokA, ledA)
	nftA.Attach(royB) // B's view onto A's collection

	v, ok := nftA.Collection().Extension(royalty.Kind)
	t.Log("attached to A:", ok, "| it is B's view:", v.(*royalty.Royalty) == royB)
}
EOF
cd $P && gno test -v -run TestAttachGuardComparesAnIDTwoTokensShare .
cd - && rm $P/zprobe_test.gno
```

```
=== RUN   TestAttachGuardComparesAnIDTwoTokensShare
one id, two ledgers: gno.land/p/demo/tokens/grc721/collection.FOO.0000007 true
attached to A: true | it is B's view: true
--- PASS: TestAttachGuardComparesAnIDTwoTokensShare
```

A's royalty answers then come from B's storage, and A's own mints never reach that extension's hooks.
</details>

## examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:16 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L16) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L16) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3862363718)
`NewCollection` is exported and `Token()` returns the genuine token, so a holder of a published collection hands on a lookalike carrying the real token and views of its own, and only the concrete type assert tells the two apart.

<details><summary>repro</summary>

`ExtensionView` is two exported methods, so any package satisfies it, and the comma-ok form a reader naturally writes turns a forged royalty into no royalty rather than an abort.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6074 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/collection
cat > $P/zprobe_test.gno <<'EOF'
package collection

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/royalty"
	"gno.land/p/nt/seqid/v0"
	"gno.land/p/nt/testutils/v0"
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

func TestAPublishedCollectionCanBeImitated(cur realm, t *testing.T) {
	artist := testutils.TestAddress("artist")

	tok, led := mkToken("Real", "REAL", 0, cur)
	nft := Wrap(tok, led)
	roy, royLedger := royalty.NewRoyalty(led, 10000)
	urequire.NoError(t, royLedger.SetDefaultRoyalty(artist, 1000))
	nft.Attach(roy)
	published := nft.Collection()

	fake := NewCollection(published.Token(), &forgedRoyalty{id: published.Token().ID()})
	v, _ := fake.Extension(royalty.Kind)
	_, isReal := v.(*royalty.Royalty)

	t.Log("same token pointer:", fake.Token() == published.Token(),
		"| advertises royalty:", fake.HasExtension(royalty.Kind),
		"| passes the concrete assert:", isReal)
}
EOF
cd $P && gno test -v -run TestAPublishedCollectionCanBeImitated .
cd - && rm $P/zprobe_test.gno
```

```
=== RUN   TestAPublishedCollectionCanBeImitated
same token pointer: true | advertises royalty: true | passes the concrete assert: false
--- PASS: TestAPublishedCollectionCanBeImitated
```

The core package beside this one already answers this shape for another type, with `IsCanonicalTeller`.
</details>

## examples/gno.land/p/demo/tokens/grc721/collection/nft.gno:48 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L48) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L48) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3862363728)
Suggestion: an extension is registered on the ledger by its own constructor and recorded here by a second call, so a realm that makes the first call and forgets this one ships an extension that is live, with hooks firing and events emitted, while every consumer reads a collection that has none and the view cannot be recovered, since re-running the constructor panics on the duplicate kind.

## examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:31 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L31) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L31) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3862363732)
Nit: a nil interface is skipped in silence here while a typed nil pointer walks past the guard and faults on the next line, so `NewCollection(tok, (*royalty.Royalty)(nil))` aborts with `runtime error: nil pointer dereference` where every other bad input to this file yields a `collection:` message.

## examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno:166 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno#L166) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno#L166) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3862363741)
Missing test: nothing attaches an extension after `Collection()` has been handed out, which is both the hazard the doc names and the only way a realm adds an extension after publication, so a later change returning a copy would pass this suite and break every realm that upgrades that way.

<details><summary>test cases</summary>

Green at this head.

```go
func TestLateAttachMutatesAPublishedCollection(cur realm, t *testing.T) {
	nft := newNFT("Foo", "FOO", 0, cur)
	published := nft.Collection() // the pointer a registry would already hold
	urequire.Equal(t, 0, len(published.Kinds()))

	meta, _ := metadata.NewMetadata(nft.Ledger())
	nft.Attach(meta)

	uassert.True(t, published == nft.Collection())
	urequire.Equal(t, 1, len(published.Kinds()))
	uassert.True(t, published.HasExtension(metadata.Kind))

	// a kind can be added but never replaced
	uassert.PanicsWithMessage(t, cur, "collection: duplicate extension kind: metadata", func() {
		nft.Attach(meta)
	})
}
```
</details>
