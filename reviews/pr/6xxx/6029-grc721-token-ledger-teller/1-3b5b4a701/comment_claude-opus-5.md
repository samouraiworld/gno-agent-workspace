# Review: PR [#6029](https://github.com/gnolang/gno/pull/6029)
Event: REQUEST_CHANGES

## Body
The consumer migrations delete about 110 comment lines the API change does not touch. That includes the doc comments on [`PlantTree`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/quarantined/gno.land/r/demo/btree_dao/btree_dao.gno#L39), [`GetToken`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/getters.gno#L34) and [`RegisterMultiToken`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L63), and most of the counterfeit-token warning in [`eventix`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/quarantined/gno.land/r/jjoptimist/eventix/eventix.gno#L22-L24). Restore the ones the migration does not need.

Verified on 3b5b4a701: a second realm that pulls the [`metadata.Metadata`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L23-L26) view through [`grc721reg.GetExtension`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L84-L91) and writes to the returned `Attributes` slice is rejected on chain with `cannot directly modify readonly tainted object`. That slice cannot be written across a realm boundary.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6029-grc721-token-ledger-teller/1-3b5b4a701/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## examples/gno.land/r/demo/grc721reg/grc721reg.gno:146 [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L146)
Critical: the kind string comes from a caller-supplied [`grc721.ExtensionView`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/types.gno#L38-L41) and reaches the shared listing raw, while the name and symbol beside it go through `md.EscapeText`. An injected heading and link land on the page users browse to find collections, and the same string reaches the [`register` event](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L51) joined by commas. Hold the kind to the charset [`validateSlug`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L152-L162) already enforces, at [`attach`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L39-L44), so the event is covered too.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6029 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/kind_injection.txtar <<'EOF'
loadpkg gno.land/p/demo/tokens/grc721
loadpkg gno.land/p/demo/tokens/grc721/collection
loadpkg gno.land/r/demo/grc721reg
loadpkg gno.land/r/evil $WORK/evil

gnoland start

gnokey maketx call -pkgpath gno.land/r/evil -func Go -gas-fee 1000000ugnot -gas-wanted 40000000 -chainid=tendermint_test test1

gnokey query vm/qrender --data 'gno.land/r/demo/grc721reg:'
! stdout 'INJECTED HEADING'

-- evil/gnomod.toml --
module = "gno.land/r/evil"
gno = "0.9"

-- evil/evil.gno --
package evil

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/grc721reg"
)

type badView struct{ id string }

func (b *badView) ExtensionKind() string {
	return "x`\n# INJECTED HEADING\n[click me](http://phish.example)\n`y"
}

func (b *badView) TokenID() string { return b.id }

var tok *grc721.Token

func init(cur realm) {
	tok, _ = grc721.NewToken("Evil Coll", "EVIL", 0, cur)
}

func Go(cur realm) string {
	return grc721reg.Register(cross(cur), tok, "evil", &badView{id: tok.ID()})
}
EOF
go test -run 'TestTestdata/kind_injection' ./gno.land/pkg/integration/ 2>&1 | grep -A6 qrender
rm gno.land/pkg/integration/testdata/kind_injection.txtar
```

```
> gnokey query vm/qrender --data 'gno.land/r/demo/grc721reg:'
[stdout]
height: 0
data: - **Evil Coll** - [gno.land/r/evil](/r/evil).evil - `x`
# INJECTED HEADING
[click me](http://phish.example)
`y` - [info](/r/demo/grc721reg:gno.land/r/evil.evil)
```
</details>

## examples/gno.land/p/demo/tokens/grc721/collection/nft.gno:34-36 [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L34)
[`Register`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L22) takes a token plus views and [builds its own collection](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L41), so following this comment leaves a realm with two extension lists and nothing comparing them. [`foo721`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/quarantined/gno.land/r/demo/foo721/foo721.gno#L30-L37) writes the list twice, and `Collection()` has no non-test caller in `examples/`. Either have `Register` take the `*Collection` or drop [`Attach`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L48-L51).

## examples/gno.land/p/demo/tokens/grc721/token.gno:114-118 [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/token.gno#L114)
Nothing stops an extension being attached after tokens exist, so the ordering rule lives only in the [enumerable](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L10) and [metadata](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L12) doc comments. An enumerable attached after one mint never catches up: [`TokenOfOwnerByIndex`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L44-L51) answers with a token the global list does not hold, and the registry still advertises the collection as `enumerable`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6029 -R gnolang/gno
cat > examples/gno.land/p/demo/tokens/grc721/enumerable/probe_test.gno <<'EOF'
package enumerable

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestProbe(cur realm, t *testing.T) {
	a := testutils.TestAddress("a")
	b := testutils.TestAddress("b")
	tok, core := newCore("Late", "LATE", 0, cur)
	urequire.NoError(t, core.Mint(a, "1"))
	enum, _ := NewEnumerable(core)
	urequire.NoError(t, core.Mint(a, "2"))
	t.Logf("core supply=%d  enumerable supply=%d", tok.TotalSupply(), enum.TotalSupply())
	_, err := enum.TokenByIndex(1)
	t.Logf("TokenByIndex(1)=%v", err)
	urequire.NoError(t, core.TransferFrom(a, a, b, "1"))
	tid, err := enum.TokenOfOwnerByIndex(b, 0)
	t.Logf("TokenOfOwnerByIndex(new owner,0)=%q err=%v", tid.String(), err)
	g0, _ := enum.TokenByIndex(0)
	t.Logf("global list = [%q] of length %d", g0.String(), enum.TotalSupply())
}
EOF
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/p/demo/tokens/grc721/enumerable -run TestProbe 2>&1 | grep -E 'supply=|TokenByIndex|TokenOfOwner|global list'
rm gno.land/p/demo/tokens/grc721/enumerable/probe_test.gno
```

```
core supply=2  enumerable supply=1
TokenByIndex(1)=index out of range
TokenOfOwnerByIndex(new owner,0)="1" err=<nil>
global list = ["2"] of length 1
```
</details>

## examples/gno.land/p/demo/tokens/grc721/token.gno:207-210 [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/token.gno#L207)
Rejecting the empty address leaves an owner no way to clear a per-token approval, which EIP-721 does by approving the zero address. The approved account keeps its right to move the token until the token itself moves. [`token_test.gno:317`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/token_test.gno#L317-L322) pins the rejection, so the carry-over from `basic_nft.gno` becomes a choice this PR makes.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6029 -R gnolang/gno
cat > examples/gno.land/p/demo/tokens/grc721/probe_test.gno <<'EOF'
package grc721

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestProbeApprove(cur realm, t *testing.T) {
	owner := testutils.TestAddress("owner")
	spender := testutils.TestAddress("spender")
	tok, led := newTestToken("Probe", "PROBE", 0, cur)
	urequire.NoError(t, led.Mint(owner, "1"))
	urequire.NoError(t, led.Approve(owner, spender, "1"))
	t.Logf("clear attempt  = %v", led.Approve(owner, address(""), "1"))
	got, _ := tok.GetApproved("1")
	t.Logf("still approved = %s", got.String())
	t.Logf("spender moves it = %v", led.TransferFrom(spender, owner, spender, "1"))
}
EOF
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/p/demo/tokens/grc721 -run TestProbeApprove 2>&1 | grep -E 'clear attempt|still approved|spender moves'
rm gno.land/p/demo/tokens/grc721/probe_test.gno
```

```
clear attempt  = invalid address
still approved = g1wdcx2mnyv4e97h6lta047h6lta047h6l735n06
spender moves it = <nil>
```
</details>

## examples/gno.land/r/demo/grc721reg/grc721reg.gno:32-36 [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg.gno#L32)
The key is the caller's slug and the prefix check binds only the realm, so the symbol constrains nothing: one realm can register the same token under two keys, or register a token whose symbol is `REAL` under a key ending `.FAKE`. [grc20reg](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40-L45) builds the key from `token.GetSymbol()`, so a caller can resolve a token from the realm and symbol it already knows. Say whether symbol-keyed lookup is meant to be dropped.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6029 -R gnolang/gno
cat > examples/gno.land/r/demo/grc721reg/probe_test.gno <<'EOF'
package grc721reg

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
)

func TestProbeKeys(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/probe"))
	tok, _ := grc721.NewToken("Probe", "REAL", 0, cur)
	t.Logf("token id      = %s", tok.ID())
	t.Logf("first key     = %s", Register(cross(cur), tok, "one"))
	t.Logf("second key    = %s", Register(cross(cur), tok, "two"))
	t.Logf("unrelated key = %s", Register(cross(cur), tok, "FAKE"))
}
EOF
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/r/demo/grc721reg -run TestProbeKeys 2>&1 | grep -E 'token id|key '
rm gno.land/r/demo/grc721reg/probe_test.gno
```

```
token id      = gno.land/r/demo/probe.REAL.0000000
first key     = gno.land/r/demo/probe.one
second key    = gno.land/r/demo/probe.two
unrelated key = gno.land/r/demo/probe.FAKE
```
</details>

## examples/gno.land/p/demo/tokens/grc721/enumerable/token_test.gno:267 [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/enumerable/token_test.gno#L267)
Missing test: an extension attached after the ledger already holds a token. These cases only cover the extension being ahead of the ledger, and it is the other order that breaks.

<details><summary>test cases</summary>

```go
func TestAttachAfterMint(cur realm, t *testing.T) {
	owner := testutils.TestAddress("attach-owner")
	next := testutils.TestAddress("attach-next")

	tok, core := newCore("Late", "LATE", 0, cur)
	urequire.NoError(t, core.Mint(owner, "1"))

	enum, _ := NewEnumerable(core)
	urequire.NoError(t, core.Mint(owner, "2"))

	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())

	first, err := enum.TokenByIndex(0)
	urequire.NoError(t, err)
	uassert.Equal(t, "1", first.String())

	urequire.NoError(t, core.TransferFrom(owner, owner, next, "1"))
	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())
}
```

Full file with its run header: [`attach_after_mint_test.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6029-grc721-token-ledger-teller/1-3b5b4a701/tests/attach_after_mint_test.gno).
</details>

## examples/gno.land/r/demo/grc721reg/grc721reg_test.gno:40-111 [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/r/demo/grc721reg/grc721reg_test.gno#L40)
Missing test: a view the extension packages did not build. Every case passes `*metadata.Metadata`, `*royalty.Royalty` or `*enumerable.Enumerable`, and all three report a constant kind, so nothing here exercises a kind string a foreign realm chose.

<details><summary>test cases</summary>

The full fixture, an integration test that registers a foreign `ExtensionView` and asserts the listing stays clean: [`grc721reg_kind_injection.txtar`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6029-grc721-token-ledger-teller/1-3b5b4a701/tests/grc721reg_kind_injection.txtar).
</details>

## gno.land/pkg/integration/testdata/grc721_emit.txtar:14 [↗](../../../../../.worktrees/gno-review-6029/gno.land/pkg/integration/testdata/grc721_emit.txtar#L14)
Nit: `.*` on both sides of `pkg_path` lets this match a `pkg_path` belonging to any event in the array, not the one carrying `TokenURIUpdate`, and [line 18](https://github.com/gnolang/gno/blob/3b5b4a701/gno.land/pkg/integration/testdata/grc721_emit.txtar#L18) is the same. `FNFT[^"]*` accepts any suffix too, where the sequence id is a fixed `0000000` that can be written out.

## examples/quarantined/gno.land/r/demo/foo721/foo721.gno:33 [↗](../../../../../.worktrees/gno-review-6029/examples/quarantined/gno.land/r/demo/foo721/foo721.gno#L33)
Nit: this drops the returned error while every other ledger call in the file panics on one, and the file is what a realm author copies.

## examples/gno.land/p/demo/tokens/grc721/metadata/token.gno:76-81 [↗](../../../../../.worktrees/gno-review-6029/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L76)
Suggestion: [`Data.Attributes`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/metadata/types.gno#L34-L44) is a slice, so the stored struct shares its backing array with the caller and with whatever [`TokenMetadata`](https://github.com/gnolang/gno/blob/3b5b4a701/examples/gno.land/p/demo/tokens/grc721/metadata/token.gno#L39-L46) returns. Inside the issuing realm, a later write to either changes stored metadata without emitting `MetadataUpdate`, the signal EIP-4906 consumers follow.
