// PKGPATH: gno.land/r/nt/commondao/v0/filetests/z_invite_gate_origin_filetest

/* Run: from a gno checkout:
gh pr checkout 6012 -R gnolang/gno && git checkout bb9f82f69
curl -fsSL -o examples/quarantined/gno.land/r/nt/commondao/v0/filetests/z_invite_gate_origin_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6012-commondao-council-tally-treasuries/1-bb9f82f69/tests/z_invite_gate_origin_filetest.gno
cd examples/quarantined/gno.land/r/nt/commondao/v0
gno test -v . 2>&1 | grep -A 12 'z_invite_gate_origin_filetest.gno'
rm filetests/z_invite_gate_origin_filetest.gno
*/

// New gates DAO creation on unsafe.OriginCaller but seats cur.Previous().Address()
// on the council, so an intermediate realm burns the invited user's invite and
// takes the only council seat. At bb9f82f69 the council holds the realm, not the user.
package z_invite_gate_origin_filetest

import (
	"chain"
	"strconv"
	"testing"

	"gno.land/r/nt/commondao/v0"
)

const (
	owner = address("g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq") // @devx
	user  = address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5") // @test1
)

const evilPath = "gno.land/r/x/evil"

var daoID uint64

func init(cur realm) {
	testing.SetRealm(testing.NewUserRealm(owner))
	commondao.Invite(cross(cur), user)

	// The invited EOA is the transaction origin, but an intermediate realm
	// is the immediate caller.
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewCodeRealm(evilPath))
	daoID = commondao.New(cross(cur), "Hijacked", "Purpose", "", "")
}

func main() {
	view := commondao.GetView(daoID)
	println("invite still held by user:", commondao.IsInvited(user))
	println("user is council member:", view.Council().Has(user))
	println("intermediate realm is council member:", view.Council().Has(chain.PackageAddress(evilPath)))
	println("dao id:", strconv.FormatUint(daoID, 10))
}

// Output:
// invite still held by user: true
// user is council member: true
// intermediate realm is council member: false
// dao id: 2
