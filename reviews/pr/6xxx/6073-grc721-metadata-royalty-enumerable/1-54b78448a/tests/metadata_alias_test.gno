/* Run: from a gno checkout:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno metadata_alias_test.gno; do
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

// The stored Data is handed back by value, but its Attributes slice is shared.
func TestMetadataAttributeAliasing(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	meta, mled := metadata.NewMetadata(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))

	stored := metadata.Data{
		Name:       "Sword",
		Attributes: []metadata.Trait{{TraitType: "rarity", Value: "common"}},
	}
	urequire.NoError(t, mled.SetTokenMetadata("1", stored))

	// read path: mutate what the getter returned
	got, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	got.Attributes[0].Value = "legendary"
	got.Name = "not stored"

	again, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	t.Log("after mutating the returned value: Name =", again.Name, "| Attributes[0].Value =", again.Attributes[0].Value)

	// write path: mutate the slice the setter was given
	stored.Attributes[0].Value = "mutated by the caller after the write"
	third, err := meta.TokenMetadata("1")
	urequire.NoError(t, err)
	t.Log("after mutating the setter's own slice: Attributes[0].Value =", third.Attributes[0].Value)
}
