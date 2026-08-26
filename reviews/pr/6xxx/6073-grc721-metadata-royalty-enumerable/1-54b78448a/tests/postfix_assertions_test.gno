/* Run: from a gno checkout, red at 54b78448a and green once the four Warnings are fixed:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno postfix_assertions_test.gno; do
  curl -fsSL -o $P/$f https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6073-grc721-metadata-royalty-enumerable/1-54b78448a/tests/$f
done
cd $P && gno test -v .
cd - && rm -r $P
*/

package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721/enumerable"
	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

// Attaching after the first mint must not be accepted silently.
func TestAttachAfterMintKeepsOneSupply(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	tok, coreLedger := newCore("Foo", "FOO", 0, cur)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))

	enum, _ := enumerable.NewEnumerable(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "2"))

	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())
}

// The read view must not be able to rewrite a stored attribute.
func TestReadViewCannotRewriteAttributes(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	meta, mled := metadata.NewMetadata(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))
	urequire.NoError(t, mled.SetTokenMetadata("1", metadata.Data{
		Attributes: []metadata.Trait{{TraitType: "rarity", Value: "common"}},
	}))

	got, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	got.Attributes[0].Value = "legendary"

	again, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	uassert.Equal(t, "common", again.Attributes[0].Value)
}

// The slice handed to the setter must stop being a write handle once it returns.
func TestSetterCopiesTheCallersSlice(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	meta, mled := metadata.NewMetadata(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))

	traits := []metadata.Trait{{TraitType: "rarity", Value: "common"}}
	urequire.NoError(t, mled.SetTokenMetadata("1", metadata.Data{Attributes: traits}))
	traits[0].Value = "legendary"

	again, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	uassert.Equal(t, "common", again.Attributes[0].Value)
}

// One extension instance must not serve two collections.
func TestExtensionBoundToOneCore(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	_, coreA := newCore("Foo", "FOO", 0, cur)
	_, coreB := newCore("Bar", "BAR", 1, cur)

	enum, led := enumerable.NewEnumerable(coreA)
	uassert.PanicsWithMessage(t, cur, "enumerable: extension already bound to another token", func() {
		coreB.RegisterExtension(led)
	})

	urequire.NoError(t, coreA.Mint(alice, "1"))
	urequire.NoError(t, coreA.Burn("1"))
	uassert.Equal(t, int64(0), enum.TotalSupply())
}
