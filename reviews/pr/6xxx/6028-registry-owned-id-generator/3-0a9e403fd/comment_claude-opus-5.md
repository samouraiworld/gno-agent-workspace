# Review: PR [#6028](https://github.com/gnolang/gno/pull/6028)
Posted: https://github.com/gnolang/gno/pull/6028#pullrequestreview-5027273175
Event: COMMENT

## Body
[Automatic AI review]

Automated pass over the registry-owned id generator, scoped to correctness, storage cost and test coverage. No merge verdict.

The description still specifies the previous design, a `gno.land/p/onbloc/identifier` package with a sha256 plus cford32 code and `slug` as an alias key, none of which the branch carries.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:32-34 [gh](https://github.com/gnolang/gno/blob/0a9e403fd/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32-L34) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32) [posted](https://github.com/gnolang/gno/pull/6028#discussion_r3860093506)
A caller that keeps this pointer roots the object under grc20reg rather than under itself, so fifty generators held in a caller's slice bill 40,051 bytes to grc20reg against 11,222 to the caller, and grc20reg references none of them and can never release that storage.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6028 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/hoard_probe.txtar <<'EOF'
loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/hoarder $WORK

gnoland start

gnokey maketx call -pkgpath gno.land/r/hoarder -func Grab -args '50' -gas-fee 1000000ugnot -gas-wanted 90000000 -chainid=tendermint_test test1
stdout OK!

-- gnomod.toml --
module = "gno.land/r/hoarder"
gno = "0.9"

-- hoarder.gno --
package hoarder

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

var kept []*grc20.IDGenerator

// Grab persists n generators in this realm's own state.
func Grab(cur realm, n int) {
	for i := 0; i < n; i++ {
		kept = append(kept, grc20reg.IdentifierGenerator(cross(cur)))
	}
}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/hoard_probe' -v -timeout 900s
rm gno.land/pkg/integration/testdata/hoard_probe.txtar
```

The realm that created and kept the generators takes 11,222 of the 51,273 bytes; the rest lands on a registry that was only asked for a pointer.

```
STORAGE DELTA:  51273 bytes
EVENTS:     [{"bytes_delta":40051,"fee_delta":{"denom":"ugnot","amount":4005100},"pkg_path":"gno.land/r/demo/defi/grc20reg"},{"bytes_delta":11222,"fee_delta":{"denom":"ugnot","amount":1122200},"pkg_path":"gno.land/r/hoarder"}]
--- PASS: TestTestdata/hoard_probe (1.35s)
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:19 [gh](https://github.com/gnolang/gno/blob/0a9e403fd/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L19) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L19) [posted](https://github.com/gnolang/gno/pull/6028#discussion_r3860093518)
One counter serves every realm, so the digits record the registry's own registration order rather than the issuing realm's: [`grc20_registry_emit.txtar`](https://github.com/gnolang/gno/blob/0a9e403fd/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L19) pins foo20 at `:0000001` only because that file loads nothing else that registers, and loading wugnot and test20 ahead of it moves foo20 to `:0000003`, so the id a realm gets is not derivable from the realm.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:173-189 [gh](https://github.com/gnolang/gno/blob/0a9e403fd/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L173-L189) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L173) [posted](https://github.com/gnolang/gno/pull/6028#discussion_r3860093521)
Missing test: no test here crosses a transaction boundary, so nothing pins the counter that keeps two registry-issued ids apart.

<details><summary>test cases</summary>

```
loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/idprobe $WORK

gnoland start

gnokey maketx call -pkgpath gno.land/r/idprobe -func Mint -args 'AAA' -gas-fee 1000000ugnot -gas-wanted 40000000 -chainid=tendermint_test test1
stdout OK!
stdout '"key":"id","value":"gno.land/r/idprobe.AAA.gno.land/r/demo/defi/grc20reg:0000001"'

gnokey maketx call -pkgpath gno.land/r/idprobe -func Mint -args 'BBB' -gas-fee 1000000ugnot -gas-wanted 40000000 -chainid=tendermint_test test1
stdout OK!
stdout '"key":"id","value":"gno.land/r/idprobe.BBB.gno.land/r/demo/defi/grc20reg:0000002"'

-- gnomod.toml --
module = "gno.land/r/idprobe"
gno = "0.9"

-- idprobe.gno --
package idprobe

import (
	"chain"

	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

func Mint(cur realm, symbol string) {
	tok, _ := grc20.NewToken("Probe", symbol, 4, grc20reg.IdentifierGenerator(cross(cur)), cur)
	chain.Emit("probe_id", "id", tok.ID())
}
```
</details>

## examples/gno.land/p/demo/tokens/grc20/token.gno:68 [gh](https://github.com/gnolang/gno/blob/0a9e403fd/examples/gno.land/p/demo/tokens/grc20/token.gno#L68) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L68) [posted](https://github.com/gnolang/gno/pull/6028#discussion_r3860093530)
Suggestion: the 30 characters spliced in here are the same for every registered token, since [`Register`](https://github.com/gnolang/gno/blob/0a9e403fd/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L67-L69) rejects any other issuer, so carrying the issuer as its own attribute on [the `NewToken` event](https://github.com/gnolang/gno/blob/0a9e403fd/examples/gno.land/p/demo/tokens/grc20/token.gno#L80-L86) would keep the id at 38 characters rather than 68 on every `Transfer`, `Approval`, `Mint` and `Burn`.

## SKIP examples/gno.land/p/demo/tokens/grc20/token.gno:27 [gh](https://github.com/gnolang/gno/blob/0a9e403fd/examples/gno.land/p/demo/tokens/grc20/token.gno#L27) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L27)
Already raised: https://github.com/gnolang/gno/pull/6028#pullrequestreview-3583044847
Suggestion: only grc20reg's own generator produces a token it accepts, so a `/p/` standard's useful contract runs through one named realm.
