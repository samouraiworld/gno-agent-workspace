/* Run: from a gno checkout:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno enumerable_direct_hooks_test.gno; do
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

// The *Ledger NewEnumerable hands back exposes only the three core hooks.
func TestEnumerableLedgerCalledDirectly(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")

	_, coreLedger := newCore("Foo", "FOO", 0, cur)
	enum, led := enumerable.NewEnumerable(coreLedger)
	urequire.NoError(t, coreLedger.Mint(alice, "1"))
	urequire.NoError(t, coreLedger.Mint(alice, "2"))
	urequire.NoError(t, coreLedger.Mint(bob, "3"))

	// "2" belongs to alice; naming bob as the sender indexes past the end of bob's list
	func() {
		defer func() { t.Log("OnTransfer with the wrong sender:", recover()) }()
		led.OnTransfer(bob, alice, "2")
	}()

	// OnBurn on a token the core still owns drops it from the index
	led.OnBurn("1")
	t.Log("after a direct OnBurn: enumerable supply", enum.TotalSupply())
	tid, err := enum.TokenByIndex(0)
	t.Log("TokenByIndex(0):", tid, err)
}
