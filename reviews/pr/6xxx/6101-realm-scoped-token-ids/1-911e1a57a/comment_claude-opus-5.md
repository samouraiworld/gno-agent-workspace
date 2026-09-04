# Review: PR [#6101](https://github.com/gnolang/gno/pull/6101)
Event: REQUEST_CHANGES

## Body
No test in the branch pins a token's identifier against the realm the token reports as its origin.

## examples/gno.land/p/demo/tokens/grc20/token.gno:46 [gh](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L46) · [↗](../../../../../.worktrees/gno-review-6101/examples/gno.land/p/demo/tokens/grc20/token.gno#L46)
The prefix here comes from [`m.Realm`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/chain/runtime/native.go#L59) and [`origRealm`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L43) from the realm value [`IsCurrent`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L23-L25) verified, so a token built through [a plain function in another `/r/` package](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/pkg/gnolang/machine.go#L2545-L2548) registers under one realm and names another in every event. Reject the identifier here when it does not start with `origRealm` and a colon.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6101 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/idprefix_probe.txtar <<'EOF'
loadpkg gno.land/r/demo/defi/grc20reg
loadpkg gno.land/r/demo/tests/bwrap $WORK/bwrap
loadpkg gno.land/r/demo/tests/amint $WORK/amint

gnoland start

gnokey maketx call -pkgpath gno.land/r/demo/tests/amint -func Mint -gas-fee 2000000ugnot -gas-wanted 20_000_000 -chainid=tendermint_test test1
stdout OK!
stdout 'origin=gno.land/r/demo/tests/amint id=gno.land/r/demo/tests/bwrap:[0-9]+ key=gno.land/r/demo/tests/amint\.AMT'

-- bwrap/gnomod.toml --
module = "gno.land/r/demo/tests/bwrap"
gno = "0.9"
-- bwrap/bwrap.gno --
package bwrap

import "gno.land/p/demo/tokens/grc20"

// Build forwards the caller's own realm to grc20.NewToken. Nothing here is
// privileged, and bwrap never sees amint's ledger.
func Build(name, symbol string, decimals int, rlm realm) (*grc20.Token, *grc20.PrivateLedger) {
	return grc20.NewToken(name, symbol, decimals, rlm)
}
-- amint/gnomod.toml --
module = "gno.land/r/demo/tests/amint"
gno = "0.9"
-- amint/amint.gno --
package amint

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
	"gno.land/r/demo/tests/bwrap"
)

var (
	Token  *grc20.Token
	ledger *grc20.PrivateLedger
)

func Mint(cur realm) string {
	Token, ledger = bwrap.Build("Amint", "AMT", 4, cur)
	key := grc20reg.Register(cross(cur), Token, "")
	ledger.Mint(cur.Address(), 1000)
	return "origin=" + Token.GetOriginRealm() + " id=" + Token.ID() + " key=" + key
}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/idprefix_probe' -v -timeout 900s
rm gno.land/pkg/integration/testdata/idprefix_probe.txtar
```

[`Register`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40) accepts the token under `amint`, and both the `NewToken` and the `Transfer` event carry `bwrap:5`.

```
("origin=gno.land/r/demo/tests/amint id=gno.land/r/demo/tests/bwrap:5 key=gno.land/r/demo/tests/amint.AMT" string)
EVENTS:     [{"type":"NewToken","attrs":[{"key":"token","value":"gno.land/r/demo/tests/bwrap:5"},…],"pkg_path":"gno.land/p/demo/tokens/grc20"},{"type":"register","attrs":[{"key":"token_path","value":"gno.land/r/demo/tests/amint.AMT"},{"key":"pkgpath","value":"gno.land/r/demo/tests/amint"},…],"pkg_path":"gno.land/r/demo/defi/grc20reg"},{"type":"Transfer","attrs":[{"key":"token","value":"gno.land/r/demo/tests/bwrap:5"},…]}]
--- PASS: TestTestdata/idprefix_probe (6.04s)
```

The `grc20reg` suite already mints across the split: a token whose `GetOriginRealm` reads `gno.land/r/demo/foo` comes back with `ID` `gno.land/r/demo/defi/grc20reg:39`, because [`testing.SetRealm`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L15-L16) moves the threaded realm value and leaves `m.Realm` on `grc20reg`. Adding the prefix check leaves `p/demo/tokens/grc20` with its four filetests, `grc20factory` and `wugnot` green and reddens only those tests.
</details>

## gnovm/stdlibs/chain/runtime/native.go:59 [gh](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/chain/runtime/native.go#L59) · [↗](../../../../../.worktrees/gno-review-6101/gnovm/stdlibs/chain/runtime/native.go#L59)
The number is the realm's object counter, so a realm redeployed after an unrelated edit mints a different identifier and nothing off chain can pin one. Say in the ADR's Consequences and in `NewToken`'s doc that an identifier is fixed at deployment and specific to it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6101 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/idcount_probe.txtar <<'EOF'
loadpkg gno.land/r/demo/tests/idlean $WORK/idlean
loadpkg gno.land/r/demo/tests/idpad $WORK/idpad

gnoland start

gnokey query vm/qeval --data 'gno.land/r/demo/tests/idlean.First()'
stdout '"gno.land/r/demo/tests/idlean:7" string'

gnokey query vm/qeval --data 'gno.land/r/demo/tests/idpad.First()'
stdout '"gno.land/r/demo/tests/idpad:17" string'

-- idlean/gnomod.toml --
module = "gno.land/r/demo/tests/idlean"
gno = "0.9"
-- idlean/idlean.gno --
package idlean

import "chain/runtime"

var first string

func init(cur realm) {
	first = runtime.NewRealmID()
}

func First() string { return first }
-- idpad/gnomod.toml --
module = "gno.land/r/demo/tests/idpad"
gno = "0.9"
-- idpad/idpad.gno --
package idpad

import "chain/runtime"

type filler struct{ a, b, c string }

var (
	pad1 = &filler{"a", "b", "c"}
	pad2 = &filler{"d", "e", "f"}
	pad3 = []*filler{{"g", "h", "i"}}

	first string
)

func init(cur realm) {
	first = runtime.NewRealmID()
}

func First() string { return first }
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/idcount_probe' -v -timeout 900s
rm gno.land/pkg/integration/testdata/idcount_probe.txtar
```

The two realms' identifier code is byte-identical and their answers are ten apart.

```
data: ("gno.land/r/demo/tests/idlean:7" string)
data: ("gno.land/r/demo/tests/idpad:17" string)
--- PASS: TestTestdata/idcount_probe (3.50s)
```

The branch's own archives show the same spread on one realm: `foo20` is [`foo20:22`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L26) loaded at genesis and [`foo20:24`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/integration/testdata/grc20_id_persists_cross_realm.txtar#L17) deployed by transaction with one file added. Master's `gno.land/r/demo/defi/foo20.FOO.0000000` was derivable from the source alone.
</details>

## gnovm/stdlibs/native_gas.go:143 [gh](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/native_gas.go#L143) · [↗](../../../../../.worktrees/gno-review-6101/gnovm/stdlibs/native_gas.go#L143)
This base borrows [`chain/params.SetString`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/native_gas.go#L121)'s price, nothing in `gnovm/cmd/calibrate` measures it, and [the header](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/native_gas.go#L71-L74) promises the fitter reproduces this table verbatim. Add `BenchmarkNative_Runtime_NewRealmID` and refit, or argue the borrowed base in the header the way [the six `chain/params` Get rows](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/native_gas.go#L88-L102) are argued.

## examples/gno.land/p/demo/tokens/grc20/token_test.gno:15-27 [gh](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L15-L27) · [↗](../../../../../.worktrees/gno-review-6101/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L15)
Missing test: [`newTokenFixture`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L15-L27) builds the struct directly, so no test in `examples/` calls [`grc20.NewToken`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L22) and its [`ErrInvalidDecimals`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L36-L38), [`ErrNotRealm`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L26-L28) and [`ErrSpoofedRealm`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L23-L25) paths lost every assertion. They have to move to `filetests/`, since a `/p/` `PKGPATH` makes the constructor abort with [`realm ID issuance requires a persistent realm`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/chain/runtime/native.go#L49-L51), `testing.SetRealm` or not.

<details><summary>test cases</summary>

Sibling [`grc721`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc721/token_test.gno#L106-L114) still covers the same set. Drop this in `examples/gno.land/p/demo/tokens/grc20/filetests/newtoken_rejects_filetest.gno` and run `gno test ./gno.land/p/demo/tokens/grc20` from `examples/`. It passes at this head.

```go
// PKGPATH: gno.land/r/demo/grc20reject
package grc20reject

import (
	"strings"

	"gno.land/p/demo/tokens/grc20"
)

// NewToken is a /p/ call from this realm's own frame, so its panic is not
// cross-realm and recover sees it.
func mustPanic(label string, want error, fn func()) {
	defer func() {
		r := recover()
		if r == nil {
			println(label, "no panic")
		} else if r != want {
			println(label, "wrong error")
		} else {
			println(label, "ok")
		}
	}()
	fn()
}

func main(cur realm) {
	mustPanic("empty name", grc20.ErrInvalidName, func() {
		grc20.NewToken("", "OK", 4, cur)
	})
	mustPanic("long name", grc20.ErrInvalidName, func() {
		grc20.NewToken(strings.Repeat("a", grc20.MaxNameLen+1), "OK", 4, cur)
	})
	mustPanic("control char in name", grc20.ErrInvalidName, func() {
		grc20.NewToken("bad\x01name", "OK", 4, cur)
	})
	mustPanic("empty symbol", grc20.ErrInvalidSymbol, func() {
		grc20.NewToken("Name", "", 4, cur)
	})
	mustPanic("dot in symbol", grc20.ErrInvalidSymbol, func() {
		grc20.NewToken("Name", "BA.D", 4, cur)
	})
	mustPanic("slash in symbol", grc20.ErrInvalidSymbol, func() {
		grc20.NewToken("Name", "BA/D", 4, cur)
	})
	mustPanic("negative decimals", grc20.ErrInvalidDecimals, func() {
		grc20.NewToken("Name", "OK", -1, cur)
	})
	mustPanic("decimals over cap", grc20.ErrInvalidDecimals, func() {
		grc20.NewToken("Name", "OK", grc20.MaxDecimals+1, cur)
	})

	tok, _ := grc20.NewToken(strings.Repeat("a", grc20.MaxNameLen), strings.Repeat("A", grc20.MaxSymbolLen), grc20.MaxDecimals, cur)
	println("boundary accepted", tok != nil)
	utf8Tok, _ := grc20.NewToken("Доллар", "RUB", 2, cur)
	println("utf8 name accepted", utf8Tok != nil)
}

// Output:
// empty name ok
// long name ok
// control char in name ok
// empty symbol ok
// dot in symbol ok
// slash in symbol ok
// negative decimals ok
// decimals over cap ok
// boundary accepted true
// utf8 name accepted true
```
</details>

## gno.land/adr/prxxxx_grc20_realm_ids.md:1 [gh](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/adr/prxxxx_grc20_realm_ids.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6101/gno.land/adr/prxxxx_grc20_realm_ids.md#L1)
Nit: the filename still carries the `prxxxx` placeholder, while the twenty-nine other PR-scoped files in `gno.land/adr/` carry their number, [`pr6025_prod_only_typecheck_at_addpackage.md`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/adr/pr6025_prod_only_typecheck_at_addpackage.md?plain=1#L1) for one.

## examples/gno.land/r/demo/tests/grc20xfer/grc20xfer.gno:16-20 [gh](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/r/demo/tests/grc20xfer/grc20xfer.gno#L16-L20) · [↗](../../../../../.worktrees/gno-review-6101/examples/gno.land/r/demo/tests/grc20xfer/grc20xfer.gno#L16)
Suggestion: this realm exists only to feed one archive, every chain then loads it, and `Transfer` spends the realm's own balance through [`RealmTeller`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L59) for any caller. Move both files into `grc20_id_persists_cross_realm.txtar` the way it already inlines [`idprobe.gno`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/integration/testdata/grc20_id_persists_cross_realm.txtar#L42-L49), and delete the realm.
