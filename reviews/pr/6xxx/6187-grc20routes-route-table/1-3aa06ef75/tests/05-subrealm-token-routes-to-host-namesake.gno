// Asserts that a token registered under a realm's "#sub" identity gets exactly
// the MsgCall route of the host realm's own token of the same symbol, so a
// wallet following it approves the wrong token: measured here as 500 on
// zone5.BAR and 0 on zone5#treasury.BAR. Pins the reviewed head's behaviour in
// the Output block; the post-fix assertion is the commented line in main.
//
// # from a local clone of github.com/gnolang/gno:
//   git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
//   mkdir -p examples/gno.land/r/demo/zone5
//   printf 'module = "gno.land/r/demo/zone5"\n\ngno = "0.9"\n' \
//     > examples/gno.land/r/demo/zone5/gnomod.toml
//   cp 05-subrealm-token-routes-to-host-namesake.gno \
//      examples/gno.land/r/demo/zone5/main_filetest.gno
//   (cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/r/demo/zone5)
//   rm -rf examples/gno.land/r/demo/zone5

// PKGPATH: gno.land/r/demo/zone5
package zone5

import (
	"chain"

	"gno.land/p/nt/grc20/v0"
	"gno.land/r/nt/grc20reg/v0"
	"gno.land/r/nt/grc20routes/v0"
)

var (
	ledgers = map[string]*grc20.PrivateLedger{}

	fooKey, barKey, subBarKey string
)

func init(cur realm) {
	foo, fl := grc20.NewToken("Foo", "FOO", 4, 0, cur)
	ledgers["FOO"] = fl
	fooKey = grc20reg.Register(cross(cur), foo, "")

	bar, bl := grc20.NewToken("Bar", "BAR", 4, 1, cur)
	ledgers["BAR"] = bl
	barKey = grc20reg.Register(cross(cur), bar, "")
}

// Approve is this realm's honestly keyed entry point: the symbol first, then
// GRC20's own arguments. Exactly the shape grc20routes.Keyed() describes, so
// the proof below is genuine and no forged allowance is involved.
func Approve(cur realm, symbol string, spender address, amount int64) {
	l := ledgers[symbol]
	if l == nil {
		panic("zone5: unknown symbol " + symbol)
	}
	if err := l.CallerTeller().Approve(0, cur, spender, amount); err != nil {
		panic(err)
	}
}

func main(cur realm) {
	probe := grc20routes.ProbeAddress()
	owner := cur.Address() // whom the frame-relative teller credits
	spender := chain.PackageAddress("gno.land/r/demo/zspender5")

	// A fully honest keyed proof: this realm really does route by symbol.
	grc20routes.BeginKeyedProof(cross(cur), fooKey, barKey, 11, 22)
	Approve(cross(cur), "FOO", probe, 11)
	Approve(cross(cur), "BAR", probe, 22)
	grc20routes.FinishKeyedProof(cross(cur), fooKey, barKey)

	// The same realm now mints a SECOND BAR under its own "treasury" sub
	// identity. grc20reg keys on the raw sub path, so the key differs and
	// grc20reg.Register's one-token-per-realm+symbol guard never fires.
	sub := cur.Sub("treasury")
	tbar, _ := grc20.NewToken("Treasury Bar", "BAR", 4, 2, sub)
	subBarKey = grc20reg.Register(cross(sub), tbar, "")

	println("host BAR key    :", barKey)
	println("treasury BAR key:", subBarKey)
	println("route(host BAR)    :", grc20routes.JSON(barKey))
	println("route(treasury BAR):", grc20routes.JSON(subBarKey))

	// A wallet reads the treasury BAR route and follows it verbatim: its
	// pkg_path plus its prefix spell zone5.Approve("BAR", spender, 500).
	Approve(cross(cur), "BAR", spender, 500)

	hostAllow := grc20reg.MustGet(barKey).Allowance(owner, spender)
	subAllow := grc20reg.MustGet(subBarKey).Allowance(owner, spender)
	println("allowance on host BAR    :", hostAllow)
	println("allowance on treasury BAR:", subAllow)

	// mustToken strips "#treasury" but Get never puts it back, so the sub
	// token is proven-keyed through a host name that does not answer for it.
	println("IsKeyedRealm(host)    :", grc20routes.IsKeyedRealm("gno.land/r/demo/zone5"))
	println("IsKeyedRealm(host#sub):", grc20routes.IsKeyedRealm("gno.land/r/demo/zone5#treasury"))

	// The pair. Active: what the reviewed head does.
	if hostAllow == 500 && subAllow == 0 {
		println("HEAD: the treasury BAR route moved the allowance onto zone5.BAR")
	}
	// Post-fix, uncomment once a sub-realm token either routes somewhere that
	// can address it or is refused a route at all:
	// if subAllow != 500 {
	// 	panic("route did not reach the token it names")
	// }
}

// Output:
// host BAR key    : gno.land/r/demo/zone5.BAR
// treasury BAR key: gno.land/r/demo/zone5#treasury.BAR
// route(host BAR)    : {"token_key":"gno.land/r/demo/zone5.BAR","pkg_path":"gno.land/r/demo/zone5","prefix":["BAR"],"source":"proven","funcs":{"approve":"Approve","transfer":"Transfer","transfer_from":"TransferFrom"}}
// route(treasury BAR): {"token_key":"gno.land/r/demo/zone5#treasury.BAR","pkg_path":"gno.land/r/demo/zone5","prefix":["BAR"],"source":"proven","funcs":{"approve":"Approve","transfer":"Transfer","transfer_from":"TransferFrom"}}
// allowance on host BAR    : 500
// allowance on treasury BAR: 0
// IsKeyedRealm(host)    : true
// IsKeyedRealm(host#sub): false
// HEAD: the treasury BAR route moved the allowance onto zone5.BAR
