/* Run: from a gno checkout:
gh pr checkout 6101 -R gnolang/gno && git checkout 911e1a57a
curl -fsSL -o examples/gno.land/p/demo/tokens/grc20/filetests/newtoken_rejects_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6101-realm-scoped-token-ids/1-911e1a57a/tests/newtoken_rejects_filetest.gno
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/p/demo/tokens/grc20
rm gno.land/p/demo/tokens/grc20/filetests/newtoken_rejects_filetest.gno
*/

// PKGPATH: gno.land/r/demo/grc20reject

// Replaces the constructor coverage TestNewTokenValidation carried before the
// VM-issued identifier landed. NewToken is unreachable from a /p/ test now,
// because runtime.NewRealmID needs a /r/ realm on the machine, so the
// rejection cases live here where PKGPATH supplies one.
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
