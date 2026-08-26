/* Run: from a gno checkout, green at 42278bdce:
gh pr checkout 6075 -R gnolang/gno && git checkout 42278bdce
P=examples/gno.land/r/demo/zprobe75
mkdir -p $P
printf 'module = "gno.land/r/demo/zprobe75"\ngno = "0.9"\n' > $P/gnomod.toml
curl -fsSL -o $P/alias_filetest.gno https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6075-grc721reg-collection-registry/1-42278bdce/tests/alias_filetest.gno
cd $P && gno test -v .
cd - && rm -r $P
*/

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

// Output:
// one collection, two keys: gno.land/r/demo/zalias.first | gno.land/r/demo/zalias.second
// - **Same Collection** - [gno.land/r/demo/zalias](/r/demo/zalias).first - _core only_ - [info](/r/demo/defi/grc721reg:gno.land/r/demo/zalias.first)
// - **Same Collection** - [gno.land/r/demo/zalias](/r/demo/zalias).second - _core only_ - [info](/r/demo/defi/grc721reg:gno.land/r/demo/zalias.second)
