/* Run: from a gno checkout:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno enumerable_shared_ledger_test.gno; do
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

// One extension instance registered on two different core ledgers.
func TestSharedLedgerAcrossCores(cur realm, t *testing.T) {
	alice := testutils.TestAddress("alice")

	tokA, coreA := newCore("Foo", "FOO", 0, cur)
	tokB, coreB := newCore("Bar", "BAR", 1, cur)

	enum, led := enumerable.NewEnumerable(coreA)
	coreB.RegisterExtension(led) // accepted: the kind is free on coreB

	urequire.NoError(t, coreA.Mint(alice, "1"))
	urequire.NoError(t, coreB.Mint(alice, "1")) // same id, different collection
	t.Log("after two mints: enum supply", enum.TotalSupply(), "| core A", tokA.TotalSupply(), "| core B", tokB.TotalSupply())

	urequire.NoError(t, coreA.Burn("1"))
	urequire.NoError(t, coreB.Burn("1"))
	t.Log("after both burns: enum supply", enum.TotalSupply(), "| core A", tokA.TotalSupply(), "| core B", tokB.TotalSupply())

	tid, err := enum.TokenByIndex(0)
	t.Log("TokenByIndex(0):", tid, err)
	tid, err = enum.TokenOfOwnerByIndex(alice, 0)
	t.Log("TokenOfOwnerByIndex(alice,0):", tid, err)
}
