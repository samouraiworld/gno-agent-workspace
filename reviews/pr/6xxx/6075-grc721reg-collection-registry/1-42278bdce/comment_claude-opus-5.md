# Review: [#6075](https://github.com/gnolang/gno/pull/6075)
Posted: https://github.com/gnolang/gno/pull/6075#pullrequestreview-5029954687
Event: COMMENT

## Body
[AI review]

Automated pass over the registry realm, scoped to the delta over #6074, with the realm audit checklist walked against it. No design judgement on storing extension views as interface values and no merge verdict.

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:152 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L152) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L152) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3862365272)
Critical: the kind is a string the registering realm chooses and it reaches the page unescaped, where the name and symbol beside it go through `md.EscapeText`, so one registration writes a heading and a link of the registrant's choosing into the index every visitor reads, and no call removes it.

<details><summary>repro</summary>

The three in-tree kinds are constants, but accepting kinds the registry does not import is the package's stated premise, so the string is caller-chosen by design.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
P=examples/gno.land/r/demo/zprobe75
mkdir -p $P
printf 'module = "gno.land/r/demo/zprobe75"\ngno = "0.9"\n' > $P/gnomod.toml
cat > $P/deface_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/zdeface
package zdeface

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/defi/grc721reg"
)

type view struct{ kind, id string }

func (v *view) ExtensionKind() string { return v.kind }
func (v *view) TokenID() string       { return v.id }

func main(cur realm) {
	tok, _ := grc721.NewToken("Bored Ape Yacht Club", "SCAM", 0, cur)

	grc721reg.Register(cross(cur), tok, "BAYC", &view{
		kind: "x`\n\n# Official collection\n\n[Mint here](https://evil.example/drain)\n\n`",
		id:   tok.ID(),
	})

	println(grc721reg.Render(""))
}
EOF
cd $P && gno test -v .
cd - && rm -r $P
```

The file carries no golden, so the run prints the page and reports the mismatch. The index page the registry then serves:

```
- **Bored Ape Yacht Club** - [gno.land/r/demo/zdeface](/r/demo/zdeface).BAYC - `x`

# Official collection

[Mint here](https://evil.example/drain)

`` - [info](/r/demo/defi/grc721reg:gno.land/r/demo/zdeface.BAYC)
```

`md.InlineCode` is already in the imported package and sizes the fence to contain internal backticks while folding newlines, so it replaces the hand-built span in one line.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:109 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L109) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L109) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3862365287)
The index walks every entry and registration is open to any realm, so the home page has a ceiling it cannot come back from: 54.8M gas at 100 entries and 165.4M at 300, near 553,000 per entry against a 3,000,000,000 query cap, with no call that removes an entry.

<details><summary>repro</summary>

Each pair of files differs only in whether `main` renders, so the difference is the render's own cost.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
B=examples/gno.land/r/demo/zbench75
mkdir -p $B
printf 'module = "gno.land/r/demo/zbench75"\ngno = "0.9"\n' > $B/gnomod.toml
for n in 100 300; do
  for kind in render noop; do
    body='println("home page bytes:", len(grc721reg.Render("")))'
    [ $kind = noop ] && body='println("home page bytes: 0")'
    cat > $B/${kind}_${n}_filetest.gno <<EOF
// PKGPATH: gno.land/r/demo/z${kind}${n}
package z${kind}${n}

import (
	"strconv"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/defi/grc721reg"
)

const N = $n

func init(cur realm) {
	tok, _ := grc721.NewToken("Filler", "FILL", 0, cur)
	for i := 0; i < N; i++ {
		grc721reg.Register(cross(cur), tok, "s"+strconv.Itoa(i))
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

```
--- ./render_100_filetest.gno (gas: 96744552)
--- ./noop_100_filetest.gno   (gas: 41952115)
--- ./render_300_filetest.gno (gas: 319205157)
--- ./noop_300_filetest.gno   (gas: 153799260)
```

| entries | `Render("")` gas |
| --- | --- |
| 100 | 54,792,437 |
| 300 | 165,405,897 |

Neither file carries a golden, so each run prints its line and reports a mismatch; the gas figure in the summary is what the measurement uses. The loop above also shows the filling: one collection under N slugs, near 559,000 gas per entry, which is the section below.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:42 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L42) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L42) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3862365295)
Keying on a caller-chosen slug drops the guard `grc20reg` documents on its own key line, so one collection takes as many entries as it likes, each carrying different extension views, and the suffix a reader treats as the symbol is free text.

<details><summary>repro</summary>

`grc20reg` builds its key from the token's own symbol, which is why a second registration of the same token always collides there and its `TestRegisterRejectsAliasedTokenPaths` passes.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
P=examples/gno.land/r/demo/zprobe75
mkdir -p $P
printf 'module = "gno.land/r/demo/zprobe75"\ngno = "0.9"\n' > $P/gnomod.toml
cat > $P/alias_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/zalias
package zalias

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/defi/grc721reg"
)

func main(cur realm) {
	token, _ := grc721.NewToken("Same Collection", "SAME", 0, cur)

	k1 := grc721reg.Register(cross(cur), token, "first")
	k2 := grc721reg.Register(cross(cur), token, "second")
	println("one collection, two keys:", k1, "|", k2)

	println(grc721reg.Render(""))
}
EOF
cd $P && gno test -v .
cd - && rm -r $P
```

```
one collection, two keys: gno.land/r/demo/zalias.first | gno.land/r/demo/zalias.second
- **Same Collection** - [gno.land/r/demo/zalias](/r/demo/zalias).first - _core only_ - [info](/r/demo/defi/grc721reg:gno.land/r/demo/zalias.first)
- **Same Collection** - [gno.land/r/demo/zalias](/r/demo/zalias).second - _core only_ - [info](/r/demo/defi/grc721reg:gno.land/r/demo/zalias.second)
```

The repro in the first section shows the other half: a collection whose symbol is `SCAM` renders under the key `…zdeface.BAYC`, in the slot where the sibling puts a symbol it verified.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:47 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L47) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L47) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3862365305)
Building a new `Collection` here rather than storing the caller's makes the entry a snapshot, so an extension the issuer attaches afterwards never reaches any consumer and the realm cannot correct it, since the key is taken and nothing updates or removes an entry.

<details><summary>repro</summary>

#6074's `Collection()` returns the live object, and its doc says a later `Attach` mutates it too. That holds for the issuer's own handle and not for the registry's copy.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
P=examples/gno.land/r/demo/zprobe75
mkdir -p $P
printf 'module = "gno.land/r/demo/zprobe75"\ngno = "0.9"\n' > $P/gnomod.toml
cat > $P/stale_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/zstale
package zstale

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/collection"
	"gno.land/p/demo/tokens/grc721/royalty"
	"gno.land/r/demo/defi/grc721reg"
)

var nft *collection.NFT

func main(cur realm) {
	token, ledger := grc721.NewToken("Late", "LATE", 0, cur)
	nft = collection.Wrap(token, ledger)

	key := grc721reg.Register(cross(cur), token, "") // no extension yet
	println("registered kinds:", len(grc721reg.MustGet(key).Kinds()))

	roy, _ := royalty.NewRoyalty(ledger, 1000)
	nft.Attach(roy)

	println("the realm's own handle now lists:", len(nft.Collection().Kinds()))
	println("the registry still lists:", len(grc721reg.MustGet(key).Kinds()))
	println(grc721reg.Render(key))
}
EOF
cd $P && gno test -v .
cd - && rm -r $P
```

```
registered kinds: 0
the realm's own handle now lists: 1
the registry still lists: 0
# Late
- symbol: **LATE**
- realm: [gno.land/r/demo/zstale](/r/demo/zstale)
- total supply: 0
- extensions: _core only_
```

Storing the caller's `*collection.Collection` keeps the two in step, and a call that replaces an entry for its own realm covers the rest.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:114 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L114) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L114) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3862365318)
`testing.SetRealm` inside the `t.Run` closure does not reach the `cur` the subtest closes over, so all three cases register from the registry realm itself and the `realmPath` each carries changes nothing.

<details><summary>repro</summary>

The same two shapes side by side, the second being what `TestRegisterAborts` does by accident with its `func(cur realm, ...)` case.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6075 -R gnolang/gno
G=examples/gno.land/r/demo/defi/grc721reg
cat > $G/zprobe_test.gno <<'EOF'
package grc721reg

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
)

func TestSetRealmScope(cur realm, t *testing.T) {
	t.Run("inside the subtest closure", func(t *testing.T) {
		testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/zprobe_inside"))
		tok, _ := grc721.NewToken("Inside", "INS", 0, cur)
		t.Log("key:", Register(cross(cur), tok, "inside"))
	})

	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/zprobe_outside"))
	tok, _ := grc721.NewToken("Outside", "OUT", 0, cur)
	t.Log("key:", Register(cross(cur), tok, "outside"))
}
EOF
cd $G && gno test -v -run TestSetRealmScope .
cd - && rm $G/zprobe_test.gno
```

```
=== RUN   TestSetRealmScope/inside_the_subtest_closure
key: gno.land/r/demo/defi/grc721reg.inside
--- PASS: TestSetRealmScope/inside_the_subtest_closure
key: gno.land/r/demo/zprobe_outside.outside
--- PASS: TestSetRealmScope
```

Hoisting the `SetRealm` and the build above `t.Run` restores the three paths, and asserting the key equals the case's realm path plus its slug pins it, where `NotEqual(t, "", key)` passes for any string.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:52 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L52) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L52) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3862365327)
Nit: the key goes out as `token_key` where the sibling emits `token_path`, though `consts.gno` says the event mirrors grc20reg for cross-registry consistency, and one line below the kinds go out as a single comma-joined attribute that a kind containing a comma splits wrongly and a kind may contain a comma, so an indexer splitting the field reads one kind as two; nothing bounds the kind's length either, and the only ceiling is the 4096 bytes `chain.Emit` allows, which aborts `Register` with a message that names the event rather than the registry.

## examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:388 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L388) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L388) [posted](https://github.com/gnolang/gno/pull/6075#discussion_r3862365335)
Missing test: every extension these tests register has a fixed benign kind, so nothing exercises a kind the registrant chose, and nothing registers one collection under two keys, which are the two behaviours the first and third sections turn on.

<details><summary>test cases</summary>

Both fail at this head, the first on the injected heading and the second because the aborts never comes.

```go
type badView struct{ kind, tid string }

func (b *badView) ExtensionKind() string { return b.kind }
func (b *badView) TokenID() string       { return b.tid }

func TestRenderEscapesExtensionKind(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_escape"))
	tok, _ := grc721.NewToken("Esc", "ESC", 0, cur)
	key := Register(cross(cur), tok, "esc", &badView{"a`\n# H\n`b", tok.ID()})

	urequire.False(t, strings.Contains(Render(""), "\n# H"))
	urequire.False(t, strings.Contains(Render(key), "\n# H"))
}

func TestOneTokenTakesOneKey(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_alias"))
	tok, _ := grc721.NewToken("Alias", "ALIAS", 0, cur)
	Register(cross(cur), tok, "a")

	urequire.AbortsContains(t, cur, "already registered", func() {
		Register(cross(cur), tok, "b")
	})
}
```
</details>
