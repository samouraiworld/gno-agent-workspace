# Review: [#6074](https://github.com/gnolang/gno/pull/6074)
Posted: https://github.com/gnolang/gno/pull/6074#pullrequestreview-5037920979
Event: COMMENT

## Body
[AI review, not manually verified]

I am doing a manual review today.

<details><summary>checks that held</summary>

- A `*Collection` a second realm stored in one transaction reports the extension the issuer registers two transactions later, driven through four `gnokey maketx call` messages against a running node.
- The three views a collection can hand out carry no exported mutator, and each extension package's write half has no `TokenID()`, so it does not satisfy `ExtensionView` and cannot be registered as one.
- The one-argument `RegisterExtension` has no caller left anywhere: `git grep -nE 'RegisterExtension\([^,)]*\)' -- '*.gno'` returns nothing, and the four realms under `examples/quarantined/` that import grc721 call it nowhere.
</details>

Verified on 6350eccafee9b5bcef45532897dfa288772b54a9

## examples/gno.land/p/demo/tokens/grc721/token.gno:166 [gh](https://github.com/gnolang/gno/blob/6350eccaf/examples/gno.land/p/demo/tokens/grc721/token.gno#L166) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/token.gno#L166) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3869322204)
`RegisterExtension` files the view under the hook's kind and calls neither of the view's own two methods, so a collection can advertise `metadata` and hand back a royalty view, or advertise `royalty` with a view naming another token.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6074 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/collection
cat > $P/zprobe_test.gno <<'EOF'
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

// takes its kind from a field, so the probe can hand it any kind
type probeHook struct{ kind string }

func (p probeHook) ExtensionKind() string                        { return p.kind }
func (p probeHook) OnMint(to address, tid grc721.TokenID)        {}
func (p probeHook) OnTransfer(f, to address, tid grc721.TokenID) {}
func (p probeHook) OnBurn(tid grc721.TokenID)                    {}

func TestViewKindIsNeverChecked(cur realm, t *testing.T) {
	_, ledA := mkPair("A", "AAA", 0, cur)
	_, ledB := mkPair("B", "BBB", 1, cur)
	royB, _ := royalty.NewRoyalty(ledB, 1000)

	ledA.RegisterExtension(probeHook{kind: metadata.Kind}, royB)

	v, ok := NewCollection(ledA).Extension(metadata.Kind)
	_, isMeta := v.(*metadata.Metadata)
	t.Log("A advertises metadata:", ok,
		"| asserts to *metadata.Metadata:", isMeta,
		"| is B's royalty view:", v.(*royalty.Royalty) == royB,
		"| its own kind:", v.ExtensionKind())
}

func TestViewFromAnotherToken(cur realm, t *testing.T) {
	tokA, ledA := mkPair("A", "AAA", 0, cur)
	_, ledB := mkPair("B", "BBB", 1, cur)
	royB, royBLedger := royalty.NewRoyalty(ledB, 1000)

	ledA.RegisterExtension(royBLedger, royB) // B's hooks and B's view, onto A

	v, ok := NewCollection(ledA).Extension(royalty.Kind)
	t.Log("A advertises royalty:", ok,
		"| the view names token:", v.TokenID(),
		"| A's token is:", tokA.ID())
}
EOF
cd $P && gno test -v -run 'TestViewKindIsNeverChecked|TestViewFromAnotherToken' .
cd - && rm $P/zprobe_test.gno
```

Both logs are the finding: the collection answers a kind with a value that is neither that kind nor its own token's.

```
=== RUN   TestViewKindIsNeverChecked
A advertises metadata: true | asserts to *metadata.Metadata: false | is B's royalty view: true | its own kind: royalty
--- PASS: TestViewKindIsNeverChecked
=== RUN   TestViewFromAnotherToken
A advertises royalty: true | the view names token: gno.land/p/demo/tokens/grc721/collection.BBB.0000001 | A's token is: gno.land/p/demo/tokens/grc721/collection.AAA.0000000
--- PASS: TestViewFromAnotherToken
```

`ExtensionView` is exactly `ExtensionKind()` and `TokenID()`, both free to call at registration. `TokenID()` is what the guard deleted this round compared, and `git grep '\.TokenID()' -- examples/` now returns three hits, all of them test assertions.
</details>

## examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:23 [gh](https://github.com/gnolang/gno/blob/6350eccaf/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L23) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L23) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3869322215)
`Collection` copies the ledger's views into a tree of its own instead of reading them, so a published collection stops tracking its ledger:

- a `NewCollection` value keeps no ledger, so it never refreshes;
- [`Wrap`](https://github.com/gnolang/gno/blob/6350eccaf/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L43) builds a fresh `*Collection` for a persisted token and ledger, so the documented refresh lands on an object nobody published.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6074 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/collection
cat > $P/zprobe_test.gno <<'EOF'
package collection

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/nt/seqid/v0"
)

func mkPair(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))

	return
}

func TestNewCollectionIsFrozen(cur realm, t *testing.T) {
	_, led := mkPair("Foo", "FOO", 0, cur)
	c := NewCollection(led)

	metadata.NewMetadata(led)

	t.Log("NewCollection kinds:", len(c.Kinds()), "| ledger kinds:", len(led.ExtensionKinds()))
}

func TestRewrapOrphansThePublished(cur realm, t *testing.T) {
	tok, led := mkPair("Foo", "FOO", 0, cur)

	published := Wrap(tok, led).Collection() // what the registry stores
	metadata.NewMetadata(led)                // a later transaction
	fresh := Wrap(tok, led).Collection()     // the issuer re-wraps its persisted pair

	t.Log("same object:", published == fresh,
		"| published kinds:", len(published.Kinds()),
		"| fresh kinds:", len(fresh.Kinds()))
}
EOF
cd $P && gno test -v -run 'TestNewCollectionIsFrozen|TestRewrapOrphansThePublished' .
cd - && rm $P/zprobe_test.gno
```

Both published values sit on zero while their ledger carries one.

```
=== RUN   TestNewCollectionIsFrozen
NewCollection kinds: 0 | ledger kinds: 1
--- PASS: TestNewCollectionIsFrozen
=== RUN   TestRewrapOrphansThePublished
same object: false | published kinds: 0 | fresh kinds: 1
--- PASS: TestRewrapOrphansThePublished
```

`Token` already holds an unexported `*PrivateLedger` with no accessor, so a `*Collection` reaches the ledger transitively today and naming the field costs no reachability. Holding it and reading the views through it takes `collection.gno` and `nft.gno` from 135 lines to 112, deletes `sync`, closes both cases, drops 100 lookups through `NFT.Extension` from 7,069,863 gas to 2,591,245, and leaves every test in `collection_test.gno` passing unchanged. It moves one behaviour: `Kinds()` then answers in registration order rather than sorted.
</details>

## examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:18 [gh](https://github.com/gnolang/gno/blob/6350eccaf/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L18) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L18) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3869322228)
`NewCollection` checks the pointer and not the pair, so a zero-value `*grc721.PrivateLedger` yields a collection that lists kinds and returns canonical views over a nil token, where `Wrap` refuses the same ledger.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6074 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/collection
cat > $P/zprobe_test.gno <<'EOF'
package collection

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/demo/tokens/grc721/royalty"
)

func TestNilTokenCollection(cur realm, t *testing.T) {
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

func TestWrapRefusesIt(cur realm, t *testing.T) {
	var tok *grc721.Token
	func(cur realm) { tok, _ = grc721.NewToken("Real", "REAL", 0, cur) }(cross(cur))

	defer func() { t.Log("Wrap with that ledger:", recover()) }()
	_ = Wrap(tok, &grc721.PrivateLedger{})
}
EOF
cd $P && gno test -v -run 'TestNilTokenCollection|TestWrapRefusesIt' .
cd - && rm $P/zprobe_test.gno
```

The collection passes every check a consumer can make and aborts the transaction on the first identity read.

```
=== RUN   TestNilTokenCollection
kinds: [metadata royalty] | Extension(royalty) ok: true | asserts to *royalty.Royalty: true | Token() == nil: true
a consumer reading c.Token().ID(): runtime error: nil pointer dereference
--- PASS: TestNilTokenCollection
=== RUN   TestWrapRefusesIt
Wrap with that ledger: collection: ledger does not belong to this token
--- PASS: TestWrapRefusesIt
```

The check does not close the shape, since `&collection.Collection{}` is equally constructible and a registry has to guard its own inputs either way. The two constructors added in one diff disagreeing on the same argument is the finding.
</details>

## examples/gno.land/p/demo/tokens/grc721/token_test.gno:698 [gh](https://github.com/gnolang/gno/blob/6350eccaf/examples/gno.land/p/demo/tokens/grc721/token_test.gno#L698) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/token_test.gno#L698) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3869322235)
Missing test: `TestExtensionViewLookup` registers only a view whose kind and token both match its hook, so nothing pins a mismatch in either.

<details><summary>test cases</summary>

Both fail at this head and pass once `RegisterExtension` compares the view's own two methods.

```go
func TestRegisterExtensionRejectsAForeignView(cur realm, t *testing.T) {
	tokA, ledA := newTestToken("A", "AAA", 0, cur)
	_, ledB := newTestToken("B", "BBB", 1, cur)

	uassert.PanicsWithMessage(t, cur, "grc721: view kind does not match the extension: mock", func() {
		ledA.RegisterExtension(journalExtension{seen: []TokenID{"1"}}, &mockView{core: tokA})
	})

	uassert.PanicsWithMessage(t, cur, "grc721: view belongs to another token: "+ledB.ReadToken().ID(), func() {
		ledA.RegisterExtension(&mockExtension{}, &mockView{core: ledB.ReadToken()})
	})
}
```
</details>

## examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno:134 [gh](https://github.com/gnolang/gno/blob/6350eccaf/examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno#L134) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno#L134) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3869322242)
Missing test: "Kinds returns sorted registered kinds" runs on `metadata` then `royalty`, a sequence identical sorted and unsorted, so it holds whether `Kinds` sorts or not.

<details><summary>test cases</summary>

`ExtensionKinds` documents registration order and `Kinds` documents none; with royalty registered first they answer `[royalty metadata]` and `[metadata royalty]` at this head. Green as written.

```go
func TestKindsOrderIsIndependentOfRegistrationOrder(cur realm, t *testing.T) {
	nft := newNFT("Foo NFT", "FNFT", 0, cur)
	royalty.NewRoyalty(nft.ledger, 1000) // registered first, sorts second
	metadata.NewMetadata(nft.ledger)

	kinds := nft.ledger.ExtensionKinds()
	urequire.Equal(t, 2, len(kinds))
	uassert.Equal(t, royalty.Kind, kinds[0], "the ledger keeps registration order")

	ks := nft.Collection().Kinds()
	urequire.Equal(t, 2, len(ks))
	uassert.Equal(t, metadata.Kind, ks[0], "Collection.Kinds is the tree walk, so it is sorted")
	uassert.Equal(t, royalty.Kind, ks[1])
}
```
</details>

## examples/gno.land/p/demo/tokens/grc721/token.gno:169 [gh](https://github.com/gnolang/gno/blob/6350eccaf/examples/gno.land/p/demo/tokens/grc721/token.gno#L169) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/token.gno#L169) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3869322255)
Nit: `PrivateLedger.HasExtension` is new exported API whose only caller is `RegisterExtension`, and it answers the opposite of `Collection.HasExtension` for a kind registered with a nil view, so a realm that trusts the collection's `false` aborts on `grc721: extension kind already registered`.

## examples/gno.land/p/demo/tokens/grc721/token.gno:195 [gh](https://github.com/gnolang/gno/blob/6350eccaf/examples/gno.land/p/demo/tokens/grc721/token.gno#L195) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/token.gno#L195) [posted](https://github.com/gnolang/gno/pull/6074#discussion_r3869322260)
Suggestion: `lookupExtension` calls `e.hook.ExtensionKind()` on every lookup instead of storing the kind at registration, so an entry re-keys itself whenever its hook's answer changes, freeing the kind it was registered under and taking one it was not.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6074 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/collection
cat > $P/zprobe_test.gno <<'EOF'
package collection

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/nt/seqid/v0"
)

// reads its kind through a pointer, so it can re-key after registration
type shifty struct{ kind *string }

func (s shifty) ExtensionKind() string                        { return *s.kind }
func (s shifty) OnMint(to address, tid grc721.TokenID)        {}
func (s shifty) OnTransfer(f, to address, tid grc721.TokenID) {}
func (s shifty) OnBurn(tid grc721.TokenID)                    {}

func TestKindNotFrozen(cur realm, t *testing.T) {
	var led *grc721.PrivateLedger
	func(cur realm) { _, led = grc721.NewToken("Foo", "FOO", seqid.ID(0), cur) }(cross(cur))

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
EOF
cd $P && gno test -v -run TestKindNotFrozen .
cd - && rm $P/zprobe_test.gno
```

The entry squats the kind it moved to, and the extension that owns that kind can no longer be registered.

```
=== RUN   TestKindNotFrozen
registered as: aux
now reports: metadata | HasExtension(aux): false | HasExtension(metadata): true
the real metadata extension can no longer register: grc721: extension kind already registered: metadata
--- PASS: TestKindNotFrozen
```

Putting `kind` on `registeredExtension` at registration and keying every lookup off it also drops one interface call per entry per lookup.
</details>
