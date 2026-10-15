// Repro (from a plain clone):
//   git clone https://github.com/gnolang/gno && cd gno && git checkout 58b2692067d6545410b99c71f33bdf506caadaa0
//   cp <this file> gnovm/tests/files/zrealm_reflect_lines_probe.gno
//   cd gnovm/pkg/gnolang && go test -short -run 'TestFiles/^zrealm_reflect_lines_probe.gno$' -update-golden-tests .
//   sed -n '/^\/\/ Output:/,$p' ../../tests/files/zrealm_reflect_lines_probe.gno
// Shows: strings.ToUpper is Of-ok with PkgPath "strings" (not a realm); h (type Handler func() int)
// is Of-ok with Type() "" and String() "<untyped>#N" although its dynamic type is the declared Handler.
// PKGPATH: gno.land/r/probe/reflectlines
package reflectlines

import (
	"chain/reflect"
	"strings"
)

type T struct{ n int }

func (t *T) M() int { return t.n }

type Handler func() int

type P *T

var (
	pkgVar T
	ptr        = &T{n: 1}
	bm         = ptr.M
	h  Handler = func() int { return 1 }
	mp         = map[string]int{"a": 1}
	declp   P  = &T{n: 2}
)

func top() int { return 1 }

func show(label string, v any) {
	obj, ok := reflect.Of(v)
	println(label, "ok:", ok, "has:", reflect.HasIdentity(v), "pkg:", obj.PkgPath(), "typ:", obj.Type(), "str:", obj.String())
}

func main(cur realm) {
	show("&pkgVar", &pkgVar)
	show("ptr", ptr)
	show("bm", bm)
	show("h", h)
	show("mp", mp)
	show("top", top)
	show("strings.ToUpper", strings.ToUpper)
	show("(*T).M", (*T).M)
	show("declp", declp)
}

// Output:
// &pkgVar ok: true has: true pkg: gno.land/r/probe/reflectlines typ: gno.land/r/probe/reflectlines.T str: gno.land/r/probe/reflectlines.T#3 (g1uaxk6wwrsm5563c5z9qly2l5c4058hz2c5meyc)
// ptr ok: true has: true pkg: gno.land/r/probe/reflectlines typ: gno.land/r/probe/reflectlines.T str: gno.land/r/probe/reflectlines.T#6 (g1ktqwxe2y5ru93yygceuaqltnekej0776e2s0nj)
// bm ok: true has: true pkg: gno.land/r/probe/reflectlines typ:  str: <untyped>#9 (g1x949c9l6eqdcue20rj0ukskvf849xwndhkk250)
// h ok: true has: true pkg: gno.land/r/probe/reflectlines typ:  str: <untyped>#13 (g1ajre539psr4cuphysnuznew4w0zsnlgl3sh055)
// mp ok: false has: false pkg:  typ:  str: <no object>
// top ok: true has: true pkg: gno.land/r/probe/reflectlines typ:  str: <untyped>#19 (g1hwnpc8qs8te826laann3ygy8u9zpd56tavqpsu)
// strings.ToUpper ok: true has: true pkg: strings typ:  str: <untyped>#46 (g1q37e4qy20davf35rwzuuu3662fnhf7f3v7npzr)
// (*T).M ok: false has: true pkg:  typ:  str: <no object>
// declp ok: true has: true pkg: gno.land/r/probe/reflectlines typ: gno.land/r/probe/reflectlines.T str: gno.land/r/probe/reflectlines.T#17 (g13yudmclt44gwh4ukyewhz0jy3ds2fy2f6nfke3)
