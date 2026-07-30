// PKGPATH: gno.land/r/nt/commondao/v0/filetests/z_tally_order_dependence_filetest

/* Run: from a gno checkout:
gh pr checkout 6012 -R gnolang/gno && git checkout bb9f82f69
curl -fsSL -o examples/quarantined/gno.land/r/nt/commondao/v0/filetests/z_tally_order_dependence_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6012-commondao-council-tally-treasuries/1-bb9f82f69/tests/z_tally_order_dependence_filetest.gno
cd examples/quarantined/gno.land/r/nt/commondao/v0
gno test -v . 2>&1 | grep -A 12 'z_tally_order_dependence_filetest.gno'
rm filetests/z_tally_order_dependence_filetest.gno
*/

// TallyDefault subtracts abstains from the denominator and CommonDAO.Vote finalizes on the
// result, so two abstains can finalize a proposal while a third member is still on ABSTAIN.
// At bb9f82f69 order 1 reads passed and order 2 dismissed from the same three intentions.
package z_tally_order_dependence_filetest

import (
	"testing"

	pdao "gno.land/p/nt/commondao/v0"
	"gno.land/r/nt/commondao/v0"
)

const (
	owner = address("g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq") // @devx
	a     = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5") // @test1
	b     = address("g1us8428u2a5satrlxzagqqa5m6vmuze025anjlj") // @test2
	c     = address("g1manfred47kzduec920z88wfr64ylksmdcedlf5") // @moul
)

const (
	daoOne = uint64(2)
	daoTwo = uint64(3)
)

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	commondao.Invite(cross(cur), a)

	// Two DAOs with the same three member council.
	testing.SetRealm(testing.NewUserRealm(a))
	commondao.New(cross(cur), "One", "Purpose", "", string(b)+"\n"+string(c))
	commondao.New(cross(cur), "Two", "Purpose", "", string(b)+"\n"+string(c))
}

func vote(cur realm, daoID, pID uint64, m address, choice pdao.VoteChoice) {
	testing.SetRealm(testing.NewUserRealm(m))
	commondao.Vote(cross(cur), daoID, pID, choice, "")
}

func status(daoID, pID uint64) string {
	p, _ := commondao.GetView(daoID).GetProposal(pID)
	return string(p.Status())
}

func main(cur realm) {
	testing.SetRealm(testing.NewUserRealm(a))
	p1 := commondao.CreateTextProposal(cross(cur), daoOne, "T", "B", 7)
	testing.SetRealm(testing.NewUserRealm(a))
	p2 := commondao.CreateTextProposal(cross(cur), daoTwo, "T", "B", 7)

	// Same council, same supermajority proposal, and b ends on NO in both.
	// Only the cast order differs.

	// Order 1: c abstains before b settles. The two abstains cut the
	// denominator to 1, so a's single YES clears two thirds and the
	// proposal finalizes while b is still on ABSTAIN.
	vote(cur, daoOne, p1, a, pdao.ChoiceYes)
	vote(cur, daoOne, p1, b, pdao.ChoiceAbstain)
	vote(cur, daoOne, p1, c, pdao.ChoiceAbstain)
	println("order 1, b has not settled yet:", status(daoOne, p1))

	// Order 2: c votes NO, the denominator stays at 3, and b's revision
	// to NO still lands.
	vote(cur, daoTwo, p2, a, pdao.ChoiceYes)
	vote(cur, daoTwo, p2, b, pdao.ChoiceAbstain)
	vote(cur, daoTwo, p2, c, pdao.ChoiceNo)
	vote(cur, daoTwo, p2, b, pdao.ChoiceNo)
	println("order 2, b settled on NO:", status(daoTwo, p2))
}

// Output:
// order 1, b has not settled yet: active
// order 2, b settled on NO: dismissed
