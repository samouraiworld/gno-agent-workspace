// PKGPATH: gno.land/r/nt/commondao/v0/filetests/z_dissolve_sweep_dead_parent_filetest

/* Run: from a gno checkout:
gh pr checkout 6012 -R gnolang/gno && git checkout bb9f82f69
curl -fsSL -o examples/quarantined/gno.land/r/nt/commondao/v0/filetests/z_dissolve_sweep_dead_parent_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6012-commondao-council-tally-treasuries/1-bb9f82f69/tests/z_dissolve_sweep_dead_parent_filetest.gno
cd examples/quarantined/gno.land/r/nt/commondao/v0
gno test -v . 2>&1 | grep -A 12 'z_dissolve_sweep_dead_parent_filetest.gno'
rm filetests/z_dissolve_sweep_dead_parent_filetest.gno
*/

// dissolvePropDefinition.execute sweeps to p.dao.Parent().Address() without checking
// whether that parent is itself dissolved. At bb9f82f69 the leaf's 500ugnot lands in the
// dissolved middle DAO and stays there after the root dissolves, beyond every governance path.
package z_dissolve_sweep_dead_parent_filetest

import (
	"chain"
	"chain/banker"
	"testing"

	pdao "gno.land/p/nt/commondao/v0"
	"gno.land/r/nt/commondao/v0"
)

const (
	owner = address("g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq") // @devx
	user  = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5") // @test1
)

const (
	rootID = uint64(2)
	midID  = uint64(3)
	leafID = uint64(4)
)

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	commondao.Invite(cross(cur), user)

	testing.SetRealm(testing.NewUserRealm(user))
	commondao.New(cross(cur), "Root", "Purpose", "", "")

	// Root -> Mid -> Leaf
	pID := commondao.CreateSubDAOProposal(cross(cur), rootID, "Mid", "Purpose", "", string(user))
	commondao.Vote(cross(cur), rootID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), rootID, pID)

	pID = commondao.CreateSubDAOProposal(cross(cur), midID, "Leaf", "Purpose", "", string(user))
	commondao.Vote(cross(cur), midID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), midID, pID)

	testing.IssueCoins(commondao.GetView(leafID).Address(), chain.NewCoins(chain.NewCoin("ugnot", 500)))

	// Dissolve Mid first. Mid holds nothing, so nothing is swept.
	pID = commondao.CreateDissolutionProposal(cross(cur), midID, "")
	commondao.Vote(cross(cur), rootID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), rootID, pID)
}

func main(cur realm) {
	testing.SetRealm(testing.NewUserRealm(user))

	// Leaf's dissolution is hosted in Root, the nearest live ancestor.
	pID := commondao.CreateDissolutionProposal(cross(cur), leafID, "")
	commondao.Vote(cross(cur), rootID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), rootID, pID)

	b := banker.NewReadonlyBanker()
	println("mid dissolved:", commondao.GetView(midID).IsDeleted())
	println("leaf treasury:", b.GetCoins(commondao.GetView(leafID).Address()).String())
	println("mid treasury:", b.GetCoins(commondao.GetView(midID).Address()).String())
	println("root treasury:", b.GetCoins(commondao.GetView(rootID).Address()).String())

	// Root dissolves itself and names a destination. Whatever sits in Mid is
	// now beyond every governance path: Mid rejects proposals, and a clawback
	// on Mid needs a live proper ancestor that no longer exists.
	pID = commondao.CreateDissolutionProposal(cross(cur), rootID, owner)
	commondao.Vote(cross(cur), rootID, pID, pdao.ChoiceYes, "")
	commondao.Execute(cross(cur), rootID, pID)

	println("after root dissolution, mid treasury:", b.GetCoins(commondao.GetView(midID).Address()).String())
	println("destination:", b.GetCoins(owner).String())
}

// Output:
// mid dissolved: true
// leaf treasury:
// mid treasury:
// root treasury: 500ugnot
// after root dissolution, mid treasury:
// destination: 500ugnot
