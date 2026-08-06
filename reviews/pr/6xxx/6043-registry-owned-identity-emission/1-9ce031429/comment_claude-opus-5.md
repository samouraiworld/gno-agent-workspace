# Review: PR [#6043](https://github.com/gnolang/gno/pull/6043)
Event: REQUEST_CHANGES

## Body
[`Emitter()`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69) exports `IssueToken` to every realm with no gate, so a live `TokenEmitter` is one line away from anyone, and every finding here goes through that door. Keeping the handle out of `Token`'s accessors is not a defence against that, and neither is calling `IssueToken` inside the constructor. The fix has to answer who may obtain a handle and what a handle entitles its holder to do.

The two balance retunes in [`storage_deposit_price_change.txtar`](https://github.com/gnolang/gno/blob/9ce031429/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L37) and [line 72](https://github.com/gnolang/gno/blob/9ce031429/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L72) are deployment-size drift: both move by the same 1176600 ugnot, 11766 bytes at the default price, against 11471 bytes of added non-test source in the two genesis-loaded packages.

Repros run at 9ce031429.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6043-registry-owned-identity-emission/1-9ce031429/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## examples/gno.land/r/demo/defi/grc20reg/emitter.gno:101-103 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L101-L103)
Critical: `Emit` never reads `t.id`, and [`Emitter()`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69) hands a live handle to any realm, so a third party emits a `Transfer` naming any token under `pkg_path: gno.land/r/demo/defi/grc20reg`. A consumer following [line 29](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L29) and filtering on `pkg_path` credits the forged rows.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6043 -R gnolang/gno
cat > examples/gno.land/r/demo/defi/grc20reg/filetests/forge_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/grc20forge
package grc20forge

import "gno.land/r/demo/defi/grc20reg"

func main(cur realm) {
	handle := grc20reg.Emitter().IssueToken(cross(cur), "X")
	handle.Emit("Transfer",
		"token", "gno.land/r/demo/victim.VIC.1",
		"from", "g1victim000000000000000000000000000000",
		"to", "g1attacker0000000000000000000000000000",
		"value", "1000000")
}

// Events:
// [
//   {
//     "type": "Transfer",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/grc20forge.X.1"
//       },
//       {
//         "key": "from",
//         "value": "g1victim000000000000000000000000000000"
//       },
//       {
//         "key": "to",
//         "value": "g1attacker0000000000000000000000000000"
//       },
//       {
//         "key": "value",
//         "value": "1000000"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc20reg"
//   }
// ]
EOF
go run ./gnovm/cmd/gno test -C examples ./gno.land/r/demo/defi/grc20reg
rm examples/gno.land/r/demo/defi/grc20reg/filetests/forge_filetest.gno
```

```
--- FAIL: ./gno.land/r/demo/defi/grc20reg/forge_filetest.gno
Events diff:
--- Expected
+++ Actual
@@ -6,3 +6,3 @@
         "key": "token",
-        "value": "gno.land/r/demo/grc20forge.X.1"
+        "value": "gno.land/r/demo/victim.VIC.1"
       },
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/emitter.gno:102 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L102)
Critical: `kind` is forwarded too, so a handle emits a `register` row byte-identical to one [`Register`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L42-L48) would produce, for a token in a namespace the caller does not own and under a `token_path` already taken. No state is written, so neither the own-realm check nor the already-registered check fires, and an indexer keyed on the event sees a registration the contract would have rejected. Binding the `token` attribute to the handle's identifier leaves this open.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6043 -R gnolang/gno
cat > examples/gno.land/r/demo/defi/grc20reg/filetests/regforge_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/grc20regforge
package grc20regforge

import "gno.land/r/demo/defi/grc20reg"

func main(cur realm) {
	handle := grc20reg.Emitter().IssueToken(cross(cur), "junk")
	handle.Emit("register",
		"token_path", "gno.land/r/gnoland/wugnot.wugnot",
		"token_id", "gno.land/r/gnoland/wugnot.wugnot.1",
		"pkgpath", "gno.land/r/gnoland/wugnot",
		"slug", "",
		"symbol", "wugnot",
		"emitter", "gno.land/r/demo/defi/grc20reg")
	println("registry has wugnot:", grc20reg.Get("gno.land/r/gnoland/wugnot.wugnot") != nil)
}

// Output:
// x

// Events:
// x
EOF
go run ./gnovm/cmd/gno test -C examples -update-golden-tests ./gno.land/r/demo/defi/grc20reg
sed -n '/^\/\/ Output:/,$p' examples/gno.land/r/demo/defi/grc20reg/filetests/regforge_filetest.gno
rm examples/gno.land/r/demo/defi/grc20reg/filetests/regforge_filetest.gno
```

```
// Output:
// registry has wugnot: false

// Events:
// [
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/gnoland/wugnot.wugnot"
//       },
// …
//       {
//         "key": "emitter",
//         "value": "gno.land/r/demo/defi/grc20reg"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc20reg"
//   }
// ]
```
</details>

## examples/gno.land/p/demo/tokens/grc20/token.gno:105-108 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L105-L108)
Critical: the check tests the identifier's namespace and never that the handle is fresh, so a realm that fetches one handle from [`Emitter().IssueToken`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74) and returns it from two `IssueToken` calls gets two ledgers behind one identifier, both emitting under the registry's `pkg_path`. [`IssuedTo`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L107-L113) then confirms the registry issued it, so one of the pair registers as registry-owned.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6043 -R gnolang/gno
cat > examples/gno.land/r/demo/defi/grc20reg/filetests/reuse_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/grc20reuse
package grc20reuse

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

type relay struct{ handle grc20.TokenEmitter }

func (r *relay) IssueToken(cur realm, symbol string) grc20.TokenEmitter { return r.handle }

func main(cur realm) {
	r := &relay{handle: grc20reg.Emitter().IssueToken(cross(cur), "DUP")}
	first, _ := grc20.NewTokenWithEmitter("A", "DUP", 6, r, cur)
	second, _ := grc20.NewTokenWithEmitter("B", "DUP", 6, r, cur)
	println("first :", first.ID())
	println("second:", second.ID())
}

// Output:
// first : gno.land/r/demo/grc20reuse.DUP.1
// second: gno.land/r/demo/grc20reuse.DUP.2
EOF
go run ./gnovm/cmd/gno test -C examples ./gno.land/r/demo/defi/grc20reg
rm examples/gno.land/r/demo/defi/grc20reg/filetests/reuse_filetest.gno
```

```
--- FAIL: ./gno.land/r/demo/defi/grc20reg/reuse_filetest.gno
Output diff:
--- Expected
+++ Actual
@@ -1,2 +1,2 @@
 first : gno.land/r/demo/grc20reuse.DUP.1
-second: gno.land/r/demo/grc20reuse.DUP.2
+second: gno.land/r/demo/grc20reuse.DUP.1
```
</details>

## examples/gno.land/p/demo/tokens/grc20/token.gno:106 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L106)
The prefix test covers the realm and not the symbol, so a token reporting `GetSymbol() == "BBB"` emits under the identifier `<realm>.AAA.1`. The doc claim that this check preserves what [`NewToken`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L53) gives holds for the realm half only. [`Register`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40-L45) rejects such a token, but an unregistered one emits under the wrong symbol indefinitely.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6043 -R gnolang/gno
cat > examples/gno.land/r/demo/defi/grc20reg/filetests/symbol_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/grc20sym
package grc20sym

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

type relay struct{ handle grc20.TokenEmitter }

func (r *relay) IssueToken(cur realm, symbol string) grc20.TokenEmitter { return r.handle }

func main(cur realm) {
	r := &relay{handle: grc20reg.Emitter().IssueToken(cross(cur), "AAA")}
	token, _ := grc20.NewTokenWithEmitter("Bee", "BBB", 6, r, cur)
	println("symbol:", token.GetSymbol())
	println("id    :", token.ID())
}

// Output:
// symbol: BBB
// id    : gno.land/r/demo/grc20sym.BBB.1
EOF
go run ./gnovm/cmd/gno test -C examples ./gno.land/r/demo/defi/grc20reg
rm examples/gno.land/r/demo/defi/grc20reg/filetests/symbol_filetest.gno
```

```
--- FAIL: ./gno.land/r/demo/defi/grc20reg/symbol_filetest.gno
Output diff:
--- Expected
+++ Actual
@@ -1,2 +1,2 @@
 symbol: BBB
-id    : gno.land/r/demo/grc20sym.BBB.1
+id    : gno.land/r/demo/grc20sym.AAA.1
```
</details>

## examples/gno.land/p/demo/tokens/grc20/token.gno:126-132 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L126-L132)
An `Emitter` implementer decides whether a ledger write emits at all, so supply and balances move against an empty event stream. At registration such a token is stamped `emitter: ""`, the same stamp a `NewToken` token gets, and that one does emit. A typed nil reaches the same silence by accident, since [line 97](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L97-L99) and [line 102](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L102-L104) catch an untyped nil literal only.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6043 -R gnolang/gno
cat > examples/gno.land/p/demo/tokens/grc20/filetests/silent_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/grc20silent
package grc20silent

import "gno.land/p/demo/tokens/grc20"

type typedNil struct{}

func (t *typedNil) IssueToken(cur realm, symbol string) grc20.TokenEmitter {
	var handle *typedNil
	return handle
}
func (t *typedNil) TokenID() string                   { return "gno.land/r/demo/grc20silent.QUI.1" }
func (t *typedNil) Emit(kind string, attrs ...string) {}

func main(cur realm) {
	token, ledger := grc20.NewTokenWithEmitter("Quiet", "QUI", 6, &typedNil{}, cur)
	println("mint ok:", ledger.Mint(cur.Address(), 1000) == nil)
	println("burn ok:", ledger.Burn(cur.Address(), 400) == nil)
	println("supply :", token.TotalSupply())
}

// Output:
// x

// Events:
// x
EOF
go run ./gnovm/cmd/gno test -C examples -update-golden-tests ./gno.land/p/demo/tokens/grc20
sed -n '/^\/\/ Output:/,$p' examples/gno.land/p/demo/tokens/grc20/filetests/silent_filetest.gno
rm examples/gno.land/p/demo/tokens/grc20/filetests/silent_filetest.gno
```

```
// Output:
// mint ok: true
// burn ok: true
// supply : 600

// Events:
// null
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/emitter.gno:75-78 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L75-L78)
`rlmPath == ""` is [`IsUserCall()`](https://github.com/gnolang/gno/blob/9ce031429/gnovm/stdlibs/chain/runtime/frame.gno#L103-L107), true for `maketx call` alone, so a `maketx run` script passes this gate and takes identifiers out of the shared counter. The predicate that covers both user forms is [`IsUser`](https://github.com/gnolang/gno/blob/9ce031429/gnovm/stdlibs/chain/runtime/frame.gno#L78-L84). Such a token registers with `emitter: gno.land/r/demo/defi/grc20reg` on it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6043 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/ephemeral_issue.txtar <<'EOF'
loadpkg gno.land/r/demo/defi/grc20reg

gnoland start

gnokey maketx run -gas-fee 1000000ugnot -gas-wanted 40000000 -broadcast -chainid=tendermint_test test1 $WORK/run/issue.gno
stdout OK!
stdout 'run\.EPH\.1'
stdout 'run\.EPH\.2'

-- run/issue.gno --
package main

import "gno.land/r/demo/defi/grc20reg"

func main(cur realm) {
	println(grc20reg.Emitter().IssueToken(cross(cur), "EPH").TokenID())
	println(grc20reg.Emitter().IssueToken(cross(cur), "EPH").TokenID())
}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/ephemeral_issue' -v
rm gno.land/pkg/integration/testdata/ephemeral_issue.txtar
```

```
gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run.EPH.1
gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run.EPH.2
--- PASS: TestTestdata/ephemeral_issue (8.61s)
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:57-60 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L57-L60)
[`IssuedTo`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L107-L113) matches the identifier string, not the handle the token holds, so a token built on a counterfeit `Emitter` returning an identifier [`IssueToken`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74) handed out is stamped `emitter: gno.land/r/demo/defi/grc20reg` while its events never touch this realm. Recording consumption at construction would make the stamp name what the token actually holds.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6043 -R gnolang/gno
cat > examples/gno.land/r/demo/defi/grc20reg/filetests/provenance_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/grc20prov
package grc20prov

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

type fake struct{ id string }

func (f *fake) IssueToken(cur realm, symbol string) grc20.TokenEmitter { return f }
func (f *fake) TokenID() string                                        { return f.id }
func (f *fake) Emit(kind string, attrs ...string)                      {}

func main(cur realm) {
	handle := grc20reg.Emitter().IssueToken(cross(cur), "VIC")
	token, ledger := grc20.NewTokenWithEmitter("Victim", "VIC", 6, &fake{id: handle.TokenID()}, cur)
	grc20reg.Register(cross(cur), token, "")
	ledger.Mint(cur.Address(), 1)
}
EOF
go run ./gnovm/cmd/gno test -C examples -update-golden-tests ./gno.land/r/demo/defi/grc20reg
sed -n '/^\/\/ Events:/,$p' examples/gno.land/r/demo/defi/grc20reg/filetests/provenance_filetest.gno
rm examples/gno.land/r/demo/defi/grc20reg/filetests/provenance_filetest.gno
```

```
// Events:
// [
//   {
//     "type": "register",
// …
//       {
//         "key": "emitter",
//         "value": "gno.land/r/demo/defi/grc20reg"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc20reg"
//   }
// ]
```

One register event and nothing else. The `Mint` on the last line emitted nothing.
</details>

## examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno:42-53 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno#L42-L53)
[`type emitter struct{}`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L41) has no fields, so `snapshot := *canonicalEmitter` copies nothing and `&snapshot` dispatches to the same `sequences` tree. The assertion left standing is the one [`TestEmitterNeverReissuesAnIdentifier`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno#L15-L24) already makes, so the test would pass with the copy deleted. The copyable value in this design is the returned `TokenEmitter`.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:69 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L69)
Missing test: nothing asserts the `emitter` attribute. [`TestRegisterReportsWhetherTheRegistryIssuedTheIdentifier`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno#L57-L71) calls `IssuedTo` and never reads the event, so the test named for the report does not cover it. That attribute is what a consumer reads to tell a registry-owned token from a `NewToken` one.

<details><summary>test cases</summary>

```go
// PKGPATH: gno.land/r/demo/grc20regattr

// Register's emitter attribute is the only thing that tells a consumer, at
// registration time, which kind of token it is looking at. One realm registers
// one token of each kind, so the two rows sit side by side.
package grc20regattr

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

func main(cur realm) {
	owned, _ := grc20.NewTokenWithEmitter("Owned", "OWN", 6, grc20reg.Emitter(), cur)
	grc20reg.Register(cross(cur), owned, "")

	legacy, _ := grc20.NewToken("Legacy", "LEG", 6, 0, cur)
	grc20reg.Register(cross(cur), legacy, "")
}
```

The golden carries `emitter: gno.land/r/demo/defi/grc20reg` on the first register event and `emitter: ""` on the second.
</details>

## examples/gno.land/r/demo/defi/grc20reg/emitter.gno:80-88 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L80-L88)
Missing test: the counter surviving a transaction boundary, which [`emitter.gno:19`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L19) names as the reason a `/p/` package cannot own it. Unit tests and filetests both run inside one transaction, so the premise of the design holds untested. It does hold.

<details><summary>test cases</summary>

```
# The registry counter has to survive a transaction boundary, which no unit
# test or filetest crosses.

loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/demo/seqtok $WORK

gnoland start

gnokey maketx call -pkgpath gno.land/r/demo/seqtok -func New -args AAA -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
stdout OK!
stdout '\("gno.land/r/demo/seqtok.AAA.1" string\)'

gnokey maketx call -pkgpath gno.land/r/demo/seqtok -func New -args BBB -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
stdout OK!
stdout '\("gno.land/r/demo/seqtok.BBB.2" string\)'

-- gnomod.toml --
module = "gno.land/r/demo/seqtok"
gno = "0.9"
-- seqtok.gno --
package seqtok

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

func New(cur realm, symbol string) string {
	tok, _ := grc20.NewTokenWithEmitter("T", symbol, 6, grc20reg.Emitter(), cur)
	return tok.ID()
}
```
</details>

## examples/gno.land/r/demo/defi/grc20reg/filetests/emitter_pkgpath_filetest.gno:24-25 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/filetests/emitter_pkgpath_filetest.gno#L24-L25)
Missing test: only `Mint` is exercised through a non-nil emitter, so a routing regression on [`Transfer`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L281), [`Approve`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L331) or [`Burn`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L396) passes CI. [`ErrNilEmitter`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/types.gno#L132) and [`ErrNilTokenEmitter`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/types.gno#L133) have no assertion anywhere either.

## examples/gno.land/p/demo/tokens/grc20/token.gno:68-69 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L68-L69)
Nit: the caller does hold a `TokenEmitter`, through the exported [`Emitter()`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69), and still will after either Critical is fixed, since neither closes that route. The neighbouring claims do resolve with the code: [line 38](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L38-L42) holds once `Emit` is bound to the handle's identifier, and [line 44](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L44-L46) holds once an identifier is single-use.

## examples/gno.land/p/demo/tokens/grc20/token.gno:116 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L116)
Nit: there is no setter and no other assignment to `Token.emitter`, so moving a token onto a registry emitter later, which the body defers as "a separate, deliberate decision per token", needs a new package path. [`ErrPkgAlreadyExists`](https://github.com/gnolang/gno/blob/9ce031429/gno.land/pkg/sdk/vm/keeper.go#L665-L667) rejects re-adding a non-private one. Say whether the target is a new chain or a `grc20/v2`.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:51-56 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51-L56)
Nit: this says two ledgers behind one identifier remain possible, and [line 28](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L28-L29) says the trailing sequence id keeps identities unique. [Line 17](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L17) says construction lives in `NewToken`.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:59 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L59)
Nit: `cur.PkgPath()` is always `gno.land/r/demo/defi/grc20reg` here, since `Register` is this realm's own crossing function. Written as a frame read, it suggests the value varies with the caller.

## examples/gno.land/r/demo/defi/grc20reg/emitter.gno:85 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L85)
Nit: `sequences` is keyed by realm path, so a realm's first two tokens come out `.AAA.1` and `.BBB.2`. The identifier puts the symbol beside a number that is not the symbol's own.

## examples/gno.land/r/demo/defi/grc20reg/emitter.gno:85-88 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L85-L88)
Suggestion: every `IssueToken` call writes two permanent tree entries whether or not a token is ever built, and no path in the realm removes either. Twenty calls from one `maketx run`, creating no token, took this realm from 15134 to 58599 bytes and locked 4346500 ugnot nobody can reclaim. Recording on consumption rather than on issuance would cut the wasted half and close the reuse hole in the same move.

## examples/gno.land/r/demo/defi/grc20reg/emitter.gno:87 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L87)
Suggestion: reached directly rather than through `NewTokenWithEmitter`, `symbol` never meets [`validSymbol`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L152-L162), so dots, slashes, newlines and unbounded length land in the `issued` key. The identifier stops parsing: `gno.land/r/demo/x.A.1.2` reads as symbol `A` sequence `1.2` or symbol `A.1` sequence `2`. No collision follows, since a package path element cannot contain a dot and the sequence is always trailing digits.

## examples/gno.land/p/demo/tokens/grc20/emitter.gno:71-73 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L71-L73)
Suggestion: no adapter, versioned wrapper or multi-registry aggregator can return an accepted identifier, since `IssueToken` reads the namespace from `cur.Previous()` and only an implementation entered straight from `NewTokenWithEmitter`'s frame passes. The caller sees `emitter issued an id outside the calling realm's namespace`, which names the emitter as the offender when the shape is the constraint.

## examples/gno.land/p/demo/tokens/grc20/token.gno:101 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L101)
Suggestion: a factory writing `func Deploy(cur realm, name, symbol string, reg grc20.Emitter)` lets its caller choose the emitter, and the identifier is built in the factory's namespace so the prefix check passes. The caller then holds a permanent hook on every ledger write of a token carrying the factory's path. [The capability note](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L48-L57) covers handing a `TokenEmitter` out and says nothing about taking an `Emitter` in.

## examples/gno.land/p/demo/tokens/grc20/token.gno:81-96 [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L81-L96)
Suggestion: five checks byte-identical to [`NewToken:35-49`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L35-L49), with nothing linking them, so a check added to one constructor is absent from the other. The shared part is a pure function of `(name, symbol, decimals, rlm)`.
