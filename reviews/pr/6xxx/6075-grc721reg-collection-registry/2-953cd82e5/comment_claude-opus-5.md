# Review: [#6075](https://github.com/gnolang/gno/pull/6075)
Posted: https://github.com/gnolang/gno/pull/6075#pullrequestreview-5040023768
Event: COMMENT

## Body
[AI review, not manually verified]

<details><summary>checks that held</summary>

- The index's cost no longer tracks the number of entries, which was the last round's finding: page one measures 9,374,904 gas at 25 entries against 10,034,786 at 300, and the same page-one figure at offset 5 of 100 entries, because the tree walk skips whole subtrees by size.
- A visitor cannot widen a page: with the size query parameter left unset, `Render("?size=1000")` and `Render("")` return the same bytes.
- A realm cannot name another realm's key in either direction, and nothing published grants a write: `rotree.ReadOnlyTree` panics on `Set` and `Remove`, `*collection.Collection` has no exported mutator, and the tellers `Token` hands out are readonly or bound to the caller's own address.
</details>

Verified on 953cd82e5c3e3defdbd1af40c3a99067ff3a9aa1

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:57 [gh](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L57) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L57) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3871114384)
Critical: a `gnokey maketx run` script mints a token and lists it in the same frame, so any funded address puts a row of its own wording on the public index, pointing at a realm that does not exist.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zrun_gate.txtar <<'TXTAR'
loadpkg gno.land/p/demo/tokens/grc721
loadpkg gno.land/p/demo/tokens/grc721/collection
loadpkg gno.land/r/demo/defi/grc721reg

gnoland start

gnokey maketx run -gas-fee 1000000ugnot -gas-wanted 200000000 -chainid=tendermint_test test1 $WORK/plant.gno
stdout OK!

# what the public index shows now
gnokey query vm/qrender --data 'gno.land/r/demo/defi/grc721reg:'
stdout 'Definitely Not A Scam'

# and whether the realm it links to exists
! gnokey query vm/qfile --data 'gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run'
stdout 'not available'

-- plant.gno --
package main

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/collection"
	"gno.land/r/demo/defi/grc721reg"
)

func main(cur realm) {
	token, _ := grc721.NewToken("Definitely Not A Scam", "GNOT", 0, cur)
	col := collection.NewCollection(token)
	println(grc721reg.Register(cross(cur), col, ""))
}
TXTAR
go test -v -run 'TestTestdata/zrun_gate' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/zrun_gate.txtar
```

The fixture passing is the finding: every assertion in it describes the entry landing.

```
> gnokey maketx run … test1 $WORK/plant.gno
gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run.GNOT
OK!
EVENTS: [… {"type":"register","attrs":[… {"key":"pkgpath","value":"gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run"} …]} …]
> gnokey query vm/qrender --data 'gno.land/r/demo/defi/grc721reg:'
> stdout 'Definitely Not A Scam'
> ! gnokey query vm/qfile --data 'gno.land/e/g1jg8…/run'
> stdout 'not available'
--- PASS: TestTestdata/zrun_gate
```

The entry is also frozen: the `PrivateLedger` died with the ephemeral package, so nobody can mint into it, and only another `maketx run` from the same address reaches `Unregister`. The predicate that separates the two cases is `IsUser()`; `IsUserCall()` is false for a run realm.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:237 [gh](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L237) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L237) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3871114398)
`v.Kinds()` materialises the whole extension set before the loop that stops at `maxBadges`, and the stored collection keeps growing after listing, so one full page costs 3,536,584,412 gas against a 3,000,000,000 query ceiling.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
P=examples/gno.land/r/demo/defi/grc721reg
cat > $P/zprobe_test.gno <<'EOF'
package grc721reg

import (
	"testing"

	"gno.land/p/demo/tokens/grc721/collection"
	"gno.land/p/nt/ufmt/v0"
)

type vView struct{ k, id string }

func (v *vView) ExtensionKind() string { return v.k }
func (v *vView) TokenID() string       { return v.id }

// one whole index page, each entry grown after listing so maxEventKinds is behind us
func fillPage(cur realm, kinds int) {
	for e := 0; e < pageSize; e++ {
		testing.SetRealm(testing.NewCodeRealm(ufmt.Sprintf("gno.land/r/demo/zv%02d", e)))

		nft, _ := collection.New("E", ufmt.Sprintf("E%02d", e), 0, cur)
		Register(cross(cur), nft.Collection(), "")

		for j := 0; j < kinds; j++ {
			nft.Attach(&vView{k: ufmt.Sprintf("k%05d", j), id: nft.Token().ID()})
		}
	}
}

func TestZFillOnly0(cur realm, t *testing.T)   { fillPage(cur, 0) }
func TestZFillRender0(cur realm, t *testing.T) { fillPage(cur, 0); _ = Render("") }
func TestZFillOnly2k(cur realm, t *testing.T)  { fillPage(cur, 2000) }
func TestZFillRender2k(cur realm, t *testing.T) {
	fillPage(cur, 2000)
	t.Log("page bytes:", len(Render("")))
}
EOF
cd $P
for t in TestZFillOnly0 TestZFillRender0 TestZFillOnly2k TestZFillRender2k; do
  printf '%-18s ' $t; gno test -v -run "$t\$" . 2>&1 | grep -- '--- GAS' | sed 's/--- GAS: *//' | sort -rn | head -1
done
cd - && rm $P/zprobe_test.gno
```

Each render figure minus its fill-only twin is the page's own cost, and the second one is over the ceiling.

```
TestZFillOnly0             18954079
TestZFillRender0           27812453     ->  8,858,374
TestZFillOnly2k         25124560727
TestZFillRender2k       28661145139     ->  3,536,584,412
page bytes: 3350
```

`maxGasQuery` is 3,000,000,000. The page is 3,350 bytes either way, so the constants bound what is printed and not what is read, and only each listing realm can remove its own row. The registry cannot reach `Collection.exts`, so the bound belongs on `*collection.Collection`.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:185 [gh](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L185) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L185) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3871114408)
`page.TotalItems` counts the whole tree rather than the page, so a page number past the last one clears this gate and the row loop writes nothing.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
P=examples/gno.land/r/demo/defi/grc721reg
cat > $P/zprobe_test.gno <<'EOF'
package grc721reg

import (
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/collection"
)

func TestZOutOfRangePage(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/zoob"))
	token, _ := grc721.NewToken("Oob", "OOB", 0, cur)
	Register(cross(cur), collection.NewCollection(token), "")

	out := Render("?page=999")
	t.Log("rows rendered:", strings.Count(out, "info"), "| body:", strings.TrimSpace(out))
}
EOF
cd $P && gno test -v -run TestZOutOfRangePage .
cd - && rm $P/zprobe_test.gno
```

A visitor arriving on a stale bookmark gets a page with nothing on it.

```
=== RUN   TestZOutOfRangePage
rows rendered: 0 | body:
--- PASS: TestZOutOfRangePage
```

Past one page the body is a picker built from the requested number, so it offers `[1](?page=1) | … | [997](?page=997) | [998](?page=998) | _999_`. `r/gov/dao/v3/impl` gates on the other value after the identical fallback.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:57 [gh](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L57) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L57) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3871114419)
This branch carries an older `collection` package than #6074's head, and building the two together leaves `grc721reg.gno` with no errors and this file with twenty, nineteen on `collection.NewCollection(token, views...)` and one on `nft.Attach`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6074 -R gnolang/gno
SIX=$(git rev-parse HEAD)
gh pr checkout 6075 -R gnolang/gno
rm -rf examples/gno.land/p/demo/tokens/grc721
git archive $SIX examples/gno.land/p/demo/tokens/grc721 | tar -x
cd examples/gno.land/r/demo/defi/grc721reg && gno test . 2>&1 | grep -c 'grc721reg_test.gno:'
cd - && git checkout -- examples/gno.land/p/demo/tokens/grc721
```

Every error lands in the test file; the registry itself needs no edit.

```
20
```

The behaviour they assert survives the port: written in the new flow, `collection.New` then `metadata.NewMetadata(led)` then `nft.Collection()`, the registry entry still reports the later extension. The live-entry assertion is the one worth keeping deliberately, since a port that took a snapshot would go quiet rather than red.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:65 [gh](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L65) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L65) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3871114426)
Nit: `registry.Set` runs above the two kind bounds, so an aborted registration stays in the tree wherever the abort is caught, which is this suite.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
P=examples/gno.land/r/demo/defi/grc721reg
cat > $P/zprobe_test.gno <<'EOF'
package grc721reg

import (
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/collection"
	"gno.land/p/nt/urequire/v0"
)

type vView struct{ k, id string }

func (v *vView) ExtensionKind() string { return v.k }
func (v *vView) TokenID() string       { return v.id }

func TestZSetBeforeChecks(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/zset"))
	token, _ := grc721.NewToken("Set", "SET", 0, cur)
	long := &vView{k: strings.Repeat("k", maxKindLen+1), id: token.ID()}

	urequire.AbortsContains(t, cur, "extension kind too long", func() {
		Register(cross(cur), collection.NewCollection(token, long), "")
	})

	t.Log("registry holds it:", Get("gno.land/r/demo/zset.SET") != nil,
		"| index renders it:", strings.Contains(Render(""), "Set"))
}
EOF
cd $P && gno test -v -run TestZSetBeforeChecks .
cd - && rm $P/zprobe_test.gno
```

The entry the registry refused is on the index.

```
=== RUN   TestZSetBeforeChecks
registry holds it: true | index renders it: true
--- PASS: TestZSetBeforeChecks
```

On chain the abort takes the transaction with it, so the ordering costs nothing there; it costs the suite, where `TestRenderIndexPaginates` counts entries and `extensionBadges`' truncation branch is reached by this accident rather than by a test.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:138 [gh](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L138) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L138) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3871114431)
Missing test: nothing asserts that finding a collection here grants no authority over its tokens, which is the property `grc20reg` spends two tests on.

<details><summary>test cases</summary>

Passes at this head. Needs `"chain"` and `uassert` in the test imports.

```go
func TestRegistryLookupGrantsNoTransferAuthority(cur realm, t *testing.T) {
	const (
		issuerPath   = "gno.land/r/demo/reg_auth_issuer"
		consumerPath = "gno.land/r/demo/reg_auth_consumer"
	)
	consumer := chain.PackageAddress(consumerPath)

	testing.SetRealm(testing.NewCodeRealm(issuerPath))
	token, ledger := grc721.NewToken("Authority", "AUTH", 0, cur)
	urequire.NoError(t, ledger.Mint(alice, "1"))
	urequire.NoError(t, ledger.Mint(alice, "2"))
	key := Register(cross(cur), collection.NewCollection(token), "")

	// alice approves the consumer realm for token 1 only
	urequire.NoError(t, ledger.ImpersonateTeller(alice).Approve(0, cur, consumer, "1"))

	// the consumer moves token 1 as itself, over a collection it does not own
	testing.SetRealm(testing.NewCodeRealm(consumerPath))
	teller := GetToken(key).RealmTeller(0, cur)
	urequire.NoError(t, teller.TransferFrom(0, cur, alice, artist, "1"))
	owner, err := MustGet(key).Token().OwnerOf("1")
	urequire.NoError(t, err)
	uassert.Equal(t, artist, owner, "the approved token moved")

	// token 2 was never approved, so the lookup grants nothing over it
	uassert.ErrorContains(t, teller.TransferFrom(0, cur, alice, artist, "2"),
		"caller is not token owner or approved")
	owner2, err := MustGet(key).Token().OwnerOf("2")
	urequire.NoError(t, err)
	uassert.Equal(t, alice, owner2, "the unapproved token did not")
}
```
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:220 [gh](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L220) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L220) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3871114436)
Refactor: `abortCase.substr` is filled in all six cases and read in none, since each `run` hardcodes its own substring, and dropping the field takes the file from 538 lines to 531.
