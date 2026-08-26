/* Run: from a gno checkout, green at 42278bdce:
gh pr checkout 6075 -R gnolang/gno && git checkout 42278bdce
P=examples/gno.land/r/demo/zprobe75
mkdir -p $P
printf 'module = "gno.land/r/demo/zprobe75"\ngno = "0.9"\n' > $P/gnomod.toml
curl -fsSL -o $P/stale_extensions_filetest.gno https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6075-grc721reg-collection-registry/1-42278bdce/tests/stale_extensions_filetest.gno
cd $P && gno test -v .
cd - && rm -r $P
*/

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

	key := grc721reg.Register(cross(cur), token, "") // registered with no extension yet
	println("registered kinds:", len(grc721reg.MustGet(key).Kinds()))

	// the issuing realm attaches royalty afterwards, on its own handle
	roy, _ := royalty.NewRoyalty(ledger, 1000)
	nft.Attach(roy)

	println("the realm's own handle now lists:", len(nft.Collection().Kinds()))
	println("the registry still lists:", len(grc721reg.MustGet(key).Kinds()))
	println(grc721reg.Render(key))
}

// Output:
// registered kinds: 0
// the realm's own handle now lists: 1
// the registry still lists: 0
// # Late
// - symbol: **LATE**
// - realm: [gno.land/r/demo/zstale](/r/demo/zstale)
// - total supply: 0
// - extensions: _core only_
