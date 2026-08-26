/* Run: from a gno checkout:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno royalty_delete_and_math_test.gno; do
  curl -fsSL -o $P/$f https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6073-grc721-metadata-royalty-enumerable/1-54b78448a/tests/$f
done
cd $P && gno test -v .
cd - && rm -r $P
*/

package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721/royalty"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/urequire/v0"
)

// DeleteTokenRoyalty announces receiver "" bps 0 while the token falls back to the collection default.
func TestDeleteTokenRoyaltyEventVsState(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")
	creator := testutils.TestAddress("creator")
	other := testutils.TestAddress("other")

	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	roy, rled := royalty.NewRoyalty(coreLedger, 10000)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))

	urequire.NoError(t, rled.SetDefaultRoyalty(creator, 500))
	urequire.NoError(t, rled.SetTokenRoyalty("1", other, 100))

	recv, amt, err := roy.RoyaltyInfo("1", 10000)
	t.Log("with override: receiver", recv, "amount", amt, "err", err)

	rled.DeleteTokenRoyalty("1") // emits RoyaltyUpdate scope=token receiver="" bps=0

	recv, amt, err = roy.RoyaltyInfo("1", 10000)
	t.Log("after DeleteTokenRoyalty: receiver", recv, "amount", amt, "err", err)

	info, ok := roy.TokenRoyalty("1")
	t.Log("TokenRoyalty override present:", ok, info)
}

// rounding and overflow edges
func TestRoyaltyMath(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")
	creator := testutils.TestAddress("creator")

	_, coreLedger := newCore("Bar", "BAR", 0, cur)
	roy, rled := royalty.NewRoyalty(coreLedger, 10000)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))
	urequire.NoError(t, rled.SetDefaultRoyalty(creator, 1))

	for _, price := range []int64{0, 1, 9999, 10000, 19999} {
		_, amt, err := roy.RoyaltyInfo("1", price)
		t.Log("bps=1 price", price, "->", amt, err)
	}

	urequire.NoError(t, rled.SetDefaultRoyalty(creator, 10000))
	_, amt, err := roy.RoyaltyInfo("1", 922337203685477) // MaxInt64/10000
	t.Log("bps=10000 price MaxInt64/10000 ->", amt, err)
	_, amt, err = roy.RoyaltyInfo("1", 922337203685478)
	t.Log("bps=10000 price MaxInt64/10000+1 ->", amt, err)

	// royalty on a token that does not exist
	_, amt, err = roy.RoyaltyInfo("nope", 10000)
	t.Log("unminted token ->", amt, err)
	// per-token override on a burned token
	urequire.NoError(t, coreLedger.Burn("1"))
	_, amt, err = roy.RoyaltyInfo("1", 10000)
	t.Log("burned token ->", amt, err)
}
