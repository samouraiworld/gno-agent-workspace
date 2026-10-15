// PKGPATH: gno.land/r/demo/reflectprobe
package reflectprobe

// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/6214/head && git checkout 58b2692067d6545410b99c71f33bdf506caadaa0
//   cp <this file> gnovm/tests/files/zrealm_reflect_zprobe_shapes.gno
//   cd gnovm/pkg/gnolang && go test . -run 'TestFiles/zrealm_reflect_zprobe_shapes' -v
// Records which shapes chain/reflect answers for today. Unpinned by the PR:
// new(T), &packageVar, a stored bound method, a top-level func, a /p/ func and
// a stdlib func all answer ok=true (the last two with a non-realm PkgPath);
// a map answers no identity although it is a reference like a func value.

import (
	"chain/reflect"
	"strings"

	"gno.land/p/nt/ufmt/v0"
)

type inner struct{ n int }

func (i *inner) Get() int { return i.n }

func helper() int { return 1 }

var (
	standalone = &inner{n: 1}
	newed      = new(inner)
	gv         inner
	mp         = map[string]int{"a": 1}
	bm         = standalone.Get
	cb         = func() int { return 2 }
	cb2        = cb
	pfn        = ufmt.Sprintf
	sfn        = strings.ToUpper
)

func show(label string, v any) {
	obj, ok := reflect.Of(v)
	println(label, "ok:", ok, "has:", reflect.HasIdentity(v), "pkg:", obj.PkgPath(), "type:", obj.Type(), "str:", obj.String())
}

func main(cur realm) {
	show("new(inner)     ", newed)
	show("&gv            ", &gv)
	show("map            ", mp)
	show("bound method   ", bm)
	show("top-level func ", helper)
	show("p/ func        ", ufmt.Sprintf)
	show("p/ func var    ", pfn)
	show("stdlib func    ", strings.ToUpper)
	show("stdlib func var", sfn)
	a, _ := reflect.Of(cb)
	b, _ := reflect.Of(cb2)
	println("cb==cb2 addr:", a.Address() == b.Address())
	x := inner{n: 5}
	p := &x
	show("&local fresh   ", p)
}

// Output:
// new(inner)      ok: true has: true pkg: gno.land/r/demo/reflectprobe type: gno.land/r/demo/reflectprobe.inner str: gno.land/r/demo/reflectprobe.inner#9 (g13x3cyqhl0w67urkzq49fd82u2588u8pwf4mary)
// &gv             ok: true has: true pkg: gno.land/r/demo/reflectprobe type: gno.land/r/demo/reflectprobe.inner str: gno.land/r/demo/reflectprobe.inner#11 (g14cvel3u5pa4eewux9l8l2atcg5gldxje4409qh)
// map             ok: false has: false pkg:  type:  str: <no object>
// bound method    ok: true has: true pkg: gno.land/r/demo/reflectprobe type:  str: <untyped>#16 (g1qvxvlgkkn3g7l5r9e6jx6kep9t2864lyqvcmwu)
// top-level func  ok: true has: true pkg: gno.land/r/demo/reflectprobe type:  str: <untyped>#3 (g1laxtq6h8c7sffagg9dvvnvr83h90jntspw5z0r)
// p/ func         ok: true has: true pkg: gno.land/p/nt/ufmt/v0 type:  str: <untyped>#6 (g1xuuyyx8xj4lj0c9lpj0au2meh25lccw34s9jca)
// p/ func var     ok: true has: true pkg: gno.land/p/nt/ufmt/v0 type:  str: <untyped>#6 (g1xuuyyx8xj4lj0c9lpj0au2meh25lccw34s9jca)
// stdlib func     ok: true has: true pkg: strings type:  str: <untyped>#46 (g1q37e4qy20davf35rwzuuu3662fnhf7f3v7npzr)
// stdlib func var ok: true has: true pkg: strings type:  str: <untyped>#46 (g1q37e4qy20davf35rwzuuu3662fnhf7f3v7npzr)
// cb==cb2 addr: true
// &local fresh    ok: false has: true pkg:  type:  str: <no object>
