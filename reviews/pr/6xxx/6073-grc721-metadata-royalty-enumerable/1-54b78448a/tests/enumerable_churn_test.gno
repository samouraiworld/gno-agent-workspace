/* Run: from a gno checkout:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
P=examples/gno.land/p/demo/tokens/grc721/zprobe
mkdir -p $P
for f in gnomod.toml helpers_test.gno enumerable_churn_test.gno; do
  curl -fsSL -o $P/$f https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6073-grc721-metadata-royalty-enumerable/1-54b78448a/tests/$f
done
cd $P && gno test -v .
cd - && rm -r $P
*/

package zprobe

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/enumerable"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

// A deterministic mint/transfer/burn churn, checking the public invariants after every step.
func TestEnumerableChurn(cur realm, t *testing.T) {
	owners := []address{
		testutils.TestAddress("a"),
		testutils.TestAddress("b"),
		testutils.TestAddress("c"),
		testutils.TestAddress("d"),
	}
	ids := []grc721.TokenID{"1", "2", "3", "4", "5", "6", "7"}

	tok, coreLedger := newCore("Foo", "FOO", 0, cur)
	enum, _ := enumerable.NewEnumerable(coreLedger)

	seed := int64(12345)
	next := func(n int64) int64 { // deterministic, no randomness
		seed = (seed*1103515245 + 12345) % 2147483648
		if seed < 0 {
			seed = -seed
		}
		return seed % n
	}

	mints, transfers, burns := 0, 0, 0
	for step := 0; step < 400; step++ {
		id := ids[next(int64(len(ids)))]
		to := owners[next(int64(len(owners)))]

		owner, err := tok.OwnerOf(id)
		switch {
		case err != nil:
			if coreLedger.Mint(to, id) == nil {
				mints++
			}
		case next(3) == 0:
			if coreLedger.Burn(id) == nil {
				burns++
			}
		default:
			if owner != to && coreLedger.TransferFrom(owner, owner, to, id) == nil {
				transfers++
			}
		}

		// supply agrees with the core
		uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())

		// every position in the global list resolves to a live token
		for i := int64(0); i < enum.TotalSupply(); i++ {
			tid, err := enum.TokenByIndex(i)
			uassert.NoError(t, err)
			_, err = tok.OwnerOf(tid)
			uassert.NoError(t, err)
		}

		// each owner list is exactly as long as the core balance, and every entry is theirs
		total := int64(0)
		for _, o := range owners {
			n := int64(0)
			for {
				tid, err := enum.TokenOfOwnerByIndex(o, n)
				if err != nil {
					break
				}
				got, err := tok.OwnerOf(tid)
				uassert.NoError(t, err)
				uassert.Equal(t, o, got)
				n++
			}
			uassert.Equal(t, tok.BalanceOf(o), n)
			total += n
		}
		uassert.Equal(t, enum.TotalSupply(), total)
	}

	t.Log("400 steps:", mints, "mints,", transfers, "transfers,", burns, "burns; final supply", tok.TotalSupply())
}
