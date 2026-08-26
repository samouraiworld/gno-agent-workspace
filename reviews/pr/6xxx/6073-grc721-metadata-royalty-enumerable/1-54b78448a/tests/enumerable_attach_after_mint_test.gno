/* Run: from a gno checkout:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno enumerable_attach_after_mint_test.gno; do
  curl -fsSL -o $P/$f https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6073-grc721-metadata-royalty-enumerable/1-54b78448a/tests/$f
done
cd $P && gno test -v .
cd - && rm -r $P
*/

package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721/enumerable"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/urequire/v0"
)


// Attach enumerable after a token already exists.
func TestAttachAfterMint(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")

	tok, coreLedger := newCore("Foo", "FOO", 0, cur)
	urequire.NoError(t, coreLedger.Mint(alice, "1")) // minted before the extension is attached

	enum, _ := enumerable.NewEnumerable(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "2"))

	t.Log("core TotalSupply:", tok.TotalSupply(), "enumerable TotalSupply:", enum.TotalSupply())

	// move the pre-attach token: the hook indexes it under the new owner only
	urequire.NoError(t, coreLedger.TransferFrom(alice, alice, bob, "1"))

	tid, err := enum.TokenOfOwnerByIndex(bob, 0)
	t.Log("TokenOfOwnerByIndex(bob,0):", tid, "err:", err)

	global := []string{}
	for i := int64(0); i < enum.TotalSupply(); i++ {
		v, err := enum.TokenByIndex(i)
		urequire.NoError(t, err)
		global = append(global, v.String())
	}
	t.Log("global list:", global, "core TotalSupply:", tok.TotalSupply(), "enumerable TotalSupply:", enum.TotalSupply())

	// burning the pre-attach token from bob
	urequire.NoError(t, coreLedger.Burn("1"))
	_, err = enum.TokenOfOwnerByIndex(bob, 0)
	t.Log("after burn, TokenOfOwnerByIndex(bob,0) err:", err, "enumerable TotalSupply:", enum.TotalSupply(), "core:", tok.TotalSupply())
}
