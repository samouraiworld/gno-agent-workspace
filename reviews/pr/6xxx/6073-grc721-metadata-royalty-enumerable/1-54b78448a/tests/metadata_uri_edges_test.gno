/* Run: from a gno checkout:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno metadata_uri_edges_test.gno; do
  curl -fsSL -o $P/$f https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6073-grc721-metadata-royalty-enumerable/1-54b78448a/tests/$f
done
cd $P && gno test -v .
cd - && rm -r $P
*/

package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestEmptyURIAndUnknownToken(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	meta, mled := metadata.NewMetadata(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))

	err := mled.SetTokenURI("1", "")
	uri, readErr := meta.TokenURI("1")
	t.Log("SetTokenURI(\"\") err:", err, "| TokenURI:", uri, readErr, "| HasTokenURI:", meta.HasTokenURI("1"))

	_, errNever := meta.TokenURI("never-minted")
	_, errUnset := meta.TokenURI("2")
	t.Log("never minted:", errNever, "| minted-with-no-uri:", errUnset)

	// a minted token with no uri, to make the pair honest
	urequire.NoError(t, coreLedger.Mint(alice, "2"))
	_, errUnset2 := meta.TokenURI("2")
	t.Log("after minting 2:", errUnset2)
}
