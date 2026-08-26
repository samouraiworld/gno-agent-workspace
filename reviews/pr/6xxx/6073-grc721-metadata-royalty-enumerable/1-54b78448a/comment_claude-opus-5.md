# Review: [#6073](https://github.com/gnolang/gno/pull/6073)
Posted: https://github.com/gnolang/gno/pull/6073#pullrequestreview-5029750455
Event: COMMENT

## Body
[AI review]

Automated pass over the three new extension packages, scoped to the delta over #6072. No design judgement on the wrap-rather-than-embed split and no merge verdict.

## examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno:11 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L11) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L11) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190076)
`NewEnumerable` accepts a core that has already minted, and the collection then carries a global list and per-owner lists that describe different token sets for the rest of its life, with no reindex call to repair it.

<details><summary>repro</summary>

A token minted before the attach never enters `allTokens`, since only `OnMint` calls `addToAll`, while `OnTransfer` still indexes that token under its new owner. The result is a supply that disagrees with the core and a token `TokenOfOwnerByIndex` returns and `TokenByIndex` never yields.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6073 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
printf 'module = "gno.land/p/demo/tokens/grc721/zprobe"\ngno = "0.9"\n' > $P/gnomod.toml
cat > $P/attach_test.gno <<'EOF'
package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/enumerable"
	"gno.land/p/nt/seqid/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func newCore(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))

	return
}

func TestAttachAfterMintKeepsOneSupply(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")

	tok, coreLedger := newCore("Foo", "FOO", 0, cur)
	urequire.NoError(t, coreLedger.Mint(alice, "1")) // minted before the extension is attached

	enum, _ := enumerable.NewEnumerable(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "2"))
	urequire.NoError(t, coreLedger.TransferFrom(alice, alice, bob, "1"))

	tid, err := enum.TokenOfOwnerByIndex(bob, 0)
	t.Log("bob holds", tid, err, "and the global list has", enum.TotalSupply(), "entries")
	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())
}
EOF
cd $P && gno test -v .
cd - && rm -r $P
```

The assertion is what the fixed code should hold, so it is red at this head:

```
=== RUN   TestAttachAfterMintKeepsOneSupply
bob holds 1 <nil> and the global list has 1 entries
uassert.Equal: same type but different value
	expected: 2
	actual:   1
--- FAIL: TestAttachAfterMintKeepsOneSupply
```

`coreLedger.ReadToken().TotalSupply()` is public, so the constructor can refuse a non-zero supply and turn this into a deploy-time failure.
</details>

## examples/gno.land/p/demo/tokens/grc721/enumerable/types.gno:20 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/enumerable/types.gno#L20) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/enumerable/types.gno#L20) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190078)
`allTokens` and `tokenList.ids` are realm-persisted slices, so each movement re-encodes the whole backing array and the extension's share of one mint grows from 1.45M gas in a 100-token collection to 7.26M in a 4000-token one, against a core mint that stays near 1.1M.

<details><summary>repro</summary>

A slice's backing array is a single realm object, so appending to it or writing one element rewrites every element. The four AVL trees beside it do not have that shape, which is why the core's own cost over the same range rises by only a third.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6073 -R gnolang/gno
B=examples/gno.land/r/zbench
mkdir -p $B
printf 'module = "gno.land/r/zbench"\ngno = "0.9"\n' > $B/gnomod.toml
for n in 100 4000; do
  for body in '_ = led' 'led.Mint(holder, pad(N))'; do
    name=$(echo "$body" | grep -q Mint && echo mint || echo noop)
    cat > $B/${name}_${n}_filetest.gno <<EOF
// PKGPATH: gno.land/r/zbench
package zbench

import (
	"strconv"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/enumerable"
)

const N = $n

const holder = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")

var led *grc721.PrivateLedger

func pad(i int) grc721.TokenID {
	s := strconv.Itoa(i)
	for len(s) < 8 {
		s = "0" + s
	}
	return grc721.TokenID(s)
}

func init(cur realm) {
	_, l := grc721.NewToken("Bench", "BNCH", 0, cur)
	led = l
	enumerable.NewEnumerable(led)
	for i := 0; i < N; i++ {
		led.Mint(holder, pad(i))
	}
}

func main(cur realm) {
	$body
}
EOF
  done
done
cd $B && gno test -v -p 1 .
cd - && rm -r $B
```

The mint's own cost is the mint file minus its no-op twin, which pays the same `init`:

```
--- PASS: ./mint_100_filetest.gno (gas: 127494215, storage: gno.land/r/zbench:+8706b)
--- PASS: ./noop_100_filetest.gno (gas: 125153490)
--- PASS: ./mint_4000_filetest.gno (gas: 9079931287, storage: gno.land/r/zbench:+8864b)
--- PASS: ./noop_4000_filetest.gno (gas: 9071515020)
```

| collection size | mint, core only | mint, enumerable attached | the extension's share |
| --- | --- | --- | --- |
| 100 | 889,245 | 2,340,725 | 1,451,480 |
| 4000 | 1,159,467 | 8,416,267 | 7,256,800 |

Near 1,490 gas per token already held, against a block cap of 3,000,000,000. The storage byte delta does not move, +8,706 against +8,864, so it is gas rather than deposit. An `avl.Tree` keyed by the index carries both lists in the shape the package already uses for `allIndex` and `ownedIndex`, and swap-and-pop becomes two `Set` calls and a `Remove`.
</details>

## examples/gno.land/p/demo/tokens/grc721/metadata/token.gno:45 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L45) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L45) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190086)
`TokenMetadata` returns a `Data` whose `Attributes` slice still points at the stored array, so code holding only the read view rewrites the record and no `MetadataUpdate` is emitted for the change.

<details><summary>repro</summary>

The struct copy isolates every scalar field and shares the one slice field, which is what makes it easy to miss: writing `got.Name` leaves the record alone.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6073 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
printf 'module = "gno.land/p/demo/tokens/grc721/zprobe"\ngno = "0.9"\n' > $P/gnomod.toml
cat > $P/alias_test.gno <<'EOF'
package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/nt/seqid/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func newCore(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))

	return
}

func TestReadViewCannotRewriteAttributes(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	meta, mled := metadata.NewMetadata(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))
	urequire.NoError(t, mled.SetTokenMetadata("1", metadata.Data{
		Name:       "Sword",
		Attributes: []metadata.Trait{{TraitType: "rarity", Value: "common"}},
	}))

	got, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	got.Name = "not stored"
	got.Attributes[0].Value = "legendary"

	again, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	t.Log("Name is", again.Name, "and the trait is", again.Attributes[0].Value)
	uassert.Equal(t, "common", again.Attributes[0].Value)
}
EOF
cd $P && gno test -v .
cd - && rm -r $P
```

```
=== RUN   TestReadViewCannotRewriteAttributes
Name is Sword and the trait is legendary
uassert.Equal: strings are different
	Diff: [-commo][+lege]n[+dary]
--- FAIL: TestReadViewCannotRewriteAttributes
```

A write from another realm stops at the readonly taint instead, `cannot directly modify readonly tainted object`, so the set is code inside the realm that owns the collection.
</details>

## examples/gno.land/p/demo/tokens/grc721/metadata/token.gno:81 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L81) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L81) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190092)
The setter stores the caller's slice rather than a copy, so the `Data` value an issuer passed in stays a live write handle into chain state after `SetTokenMetadata` returns, and copying on the read path alone leaves this open.

<details><summary>repro</summary>

Same package as the previous block, one more test:

```go
func TestSetterCopiesTheCallersSlice(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	meta, mled := metadata.NewMetadata(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))

	traits := []metadata.Trait{{TraitType: "rarity", Value: "common"}}
	urequire.NoError(t, mled.SetTokenMetadata("1", metadata.Data{Attributes: traits}))
	traits[0].Value = "legendary" // the caller keeps writing after the setter returned

	again, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	uassert.Equal(t, "common", again.Attributes[0].Value)
}
```

```
=== RUN   TestSetterCopiesTheCallersSlice
uassert.Equal: strings are different
	Diff: [-commo][+lege]n[+dary]
--- FAIL: TestSetterCopiesTheCallersSlice
```
</details>

## examples/gno.land/p/demo/tokens/grc721/royalty/token.gno:129 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L129) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L129) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190106)
`DeleteTokenRoyalty` announces an empty receiver and bps 0 while `resolve` then falls back to the collection default, so an indexer reconstructing policy from `RoyaltyUpdate` pays nothing on a token the chain still charges for.

<details><summary>repro</summary>

The event payload and the read disagree in the same transaction, which a filetest shows in one file since it prints both.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6073 -R gnolang/gno
F=examples/gno.land/p/demo/tokens/grc721/royalty/filetests
mkdir -p $F
cat > $F/zroyalty_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/zroyevents
package zroyevents

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/royalty"
)

func main(cur realm) {
	_, core := grc721.NewToken("Foo", "FOO", 0, cur)
	roy, led := royalty.NewRoyalty(core, 10000)
	me := cur.Address()

	core.Mint(me, "1")
	led.SetDefaultRoyalty(me, 500)    // collection default, 5%
	led.SetTokenRoyalty("1", me, 100) // per-token override, 1%

	led.DeleteTokenRoyalty("1")

	_, amount, _ := roy.RoyaltyInfo("1", 10000)
	println("RoyaltyInfo after the delete:", amount)
}
EOF
cd examples/gno.land/p/demo/tokens/grc721/royalty && gno test -v . -update-golden-tests
cd - && rm -r $F
```

The golden the run writes carries both halves. The last event before the read:

```
//     "type": "RoyaltyUpdate",
//         "key": "scope",    "value": "token"
//         "key": "tokenId",  "value": "1"
//         "key": "receiver", "value": ""
//         "key": "bps",      "value": "0"
```

and the output line beneath it:

```
// Output:
// RoyaltyInfo after the delete: 500
```

EIP-2981 makes `royaltyInfo` the authority for what a payer owes, so emitting the resolved state after the delete, meaning the default where one exists, is what keeps the stream usable.
</details>

## examples/gno.land/p/demo/tokens/grc721/royalty/token.gno:165 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L165) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L165) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190111)
`OnBurn` performs the same `perToken.Remove` as `DeleteTokenRoyalty` and emits nothing, so the last royalty an indexer holds for that id is one the chain no longer applies, and fixing the delete leaves this path untouched.

## examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno:19 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L19) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L19) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190117)
The constructor captures `core` and no hook ever reads it, so the same extension instance registered on a second core serves both collections and leaves the first with an index entry no burn can clear.

<details><summary>repro</summary>

The core rejects a duplicate kind per ledger, not a second ledger per extension, so the second registration is accepted. `metadata` and `royalty` carry the milder form, where a burn in one collection clears the other's record for that id.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6073 -R gnolang/gno
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
printf 'module = "gno.land/p/demo/tokens/grc721/zprobe"\ngno = "0.9"\n' > $P/gnomod.toml
cat > $P/shared_test.gno <<'EOF'
package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/enumerable"
	"gno.land/p/nt/seqid/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func newCore(name, symbol string, id seqid.ID, rlm realm) (tok *grc721.Token, led *grc721.PrivateLedger) {
	func(cur realm) {
		tok, led = grc721.NewToken(name, symbol, id, cur)
	}(cross(rlm))

	return
}

func TestExtensionServesOneCore(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	_, coreA := newCore("Foo", "FOO", 0, cur)
	_, coreB := newCore("Bar", "BAR", 1, cur)

	enum, led := enumerable.NewEnumerable(coreA)
	coreB.RegisterExtension(led) // a second collection, the same extension

	urequire.NoError(t, coreA.Mint(alice, "1"))
	urequire.NoError(t, coreB.Mint(alice, "1"))
	urequire.NoError(t, coreA.Burn("1"))
	urequire.NoError(t, coreB.Burn("1"))

	tid, err := enum.TokenByIndex(0)
	t.Log("both collections are empty, the index still answers", tid, err)
	uassert.Equal(t, int64(0), enum.TotalSupply())
}
EOF
cd $P && gno test -v .
cd - && rm -r $P
```

```
=== RUN   TestExtensionServesOneCore
both collections are empty, the index still answers 1 <nil>
uassert.Equal: same type but different value
	expected: 0
	actual:   1
--- FAIL: TestExtensionServesOneCore
```

Capturing `core.ID()` at construction and rejecting a hook from any other core closes it inside these packages.
</details>

## examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno:21 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L21) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L21) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190120)
Refactor: the returned `*Ledger` carries nothing but the three hooks the constructor already wired, and calling them by hand desyncs the index, so returning `*Enumerable` alone drops a value the series never uses, including [#6074](https://github.com/gnolang/gno/pull/6074)'s `NewCollection`, which takes read views only.

<details><summary>details</summary>

`metadata` and `royalty` both return a ledger carrying real setters; this one has none. `led.OnBurn(tid)` on a token the core still owns leaves the extension reporting 2 against the core's 3, and `led.OnTransfer(bob, alice, tid)` naming a sender who does not hold the token indexes past the end of that sender's list and faults with `slice index out of bounds: 1 (len=1)`, which gno `recover` catches.
</details>

## examples/gno.land/p/demo/tokens/grc721/royalty/token.gno:50 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L50) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/royalty/token.gno#L50) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190128)
Suggestion: the guard bounds `salePrice * Bps` rather than the result, so a sale price whose royalty is perfectly representable is refused with the same pair the doc four lines above calls the no-royalty signal.

<details><summary>details</summary>

At 10000 bps the royalty is the sale price itself, yet a price of 922337203685478 returns `ErrInvalidSalePrice` while 922337203685477 succeeds. Splitting the multiply removes both the branch and the `math` import:

```go
amount := salePrice/FeeDenominator*info.Bps + salePrice%FeeDenominator*info.Bps/FeeDenominator
```

Applied in a worktree and run both ways. The package suite stays green, and the table matches the current code everywhere the current code answers at all:

| bps | sale price | now | with the split multiply |
| --- | --- | --- | --- |
| 1 | 9999 | 0 | 0 |
| 1 | 10000 | 1 | 1 |
| 10000 | 922337203685477 | 922337203685477 | 922337203685477 |
| 10000 | 922337203685478 | error | 922337203685478 |
| 10000 | 9223372036854775807 | error | 9223372036854775807 |
</details>

## examples/gno.land/p/demo/tokens/grc721/metadata/types.gno:54 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L54) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L54) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190134)
Suggestion: the on-chain `Data` store has no reader anywhere, since nothing renders it, no `tokenURI` resolves to it, and neither [#6074](https://github.com/gnolang/gno/pull/6074) nor [#6075](https://github.com/gnolang/gno/pull/6075) calls `TokenMetadata`, so serving it from `TokenURI` when no URI is set is what makes the second store worth its bytes.

## examples/gno.land/p/demo/tokens/grc721/metadata/token.gno:76 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L76) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L76) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190139)
Suggestion: nothing caps the trait count or any string length here, where the core validates its own name and symbol, and one token carrying 100 traits measures 550,404 gas and 55,612 storage bytes, which is 5.5 GNOT locked at the default price.

## examples/gno.land/p/demo/tokens/grc721/metadata/token.gno:58 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L58) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L58) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190146)
Nit: `SetTokenURI` accepts the empty string, after which `HasTokenURI` answers true and `TokenURI` returns an empty URI with no error, where EIP-721 asks for an RFC 3986 URI.

## examples/gno.land/p/demo/tokens/grc721/enumerable/token_test.gno:267 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-extensions/examples/gno.land/p/demo/tokens/grc721/enumerable/token_test.gno#L267) · [↗](../../../../../.worktrees/gno-review-6073/examples/gno.land/p/demo/tokens/grc721/enumerable/token_test.gno#L267) [posted](https://github.com/gnolang/gno/pull/6073#discussion_r3862190150)
Missing test: no test attaches more than one extension to a single core ledger, so one burn fanning out to three hook sets, which is what the title claims, is uncovered.

<details><summary>test cases</summary>

Green at this head, and it belongs under `enumerable/`, the only one of the three that can import the other two without a cycle.

```go
func TestStackedExtensions(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	creator := testutils.TestAddress("creator")

	tok, coreLedger := newCore("Foo", "FOO", 0, cur)
	enum, _ := NewEnumerable(coreLedger)
	meta, mled := metadata.NewMetadata(coreLedger)
	roy, rled := royalty.NewRoyalty(coreLedger, 1000)

	urequire.NoError(t, coreLedger.Mint(alice, "1"))
	urequire.NoError(t, mled.SetTokenURI("1", "ipfs://one"))
	urequire.NoError(t, rled.SetDefaultRoyalty(creator, 250))
	urequire.NoError(t, rled.SetTokenRoyalty("1", creator, 500))

	// a transfer keeps every per-token record
	urequire.NoError(t, coreLedger.TransferFrom(alice, alice, bob, "1"))
	uassert.True(t, meta.HasTokenURI("1"))
	_, ok := roy.TokenRoyalty("1")
	uassert.True(t, ok)
	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())

	// one burn clears all three, and the collection default survives it
	urequire.NoError(t, coreLedger.Burn("1"))
	uassert.False(t, meta.HasTokenURI("1"))
	_, ok = roy.TokenRoyalty("1")
	uassert.False(t, ok)
	_, hasDefault := roy.DefaultRoyalty()
	uassert.True(t, hasDefault)
	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())

	// the same id re-minted carries no URI and no override
	urequire.NoError(t, coreLedger.Mint(bob, "1"))
	uassert.False(t, meta.HasTokenURI("1"))
	_, amount, err := roy.RoyaltyInfo("1", 10000)
	urequire.NoError(t, err)
	uassert.Equal(t, int64(250), amount)
	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())
}
```
</details>
