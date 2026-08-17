/* Run: from a gno checkout:
gh pr checkout 6068 -R gnolang/gno && git checkout 304f09a7a
curl -fsSL -o examples/gno.land/r/gov/dao/v3/impl/filetests/description_40b_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6068-gov-dao-allowlist-lockdown/1-304f09a7a/tests/description_40b_filetest.gno
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gov/dao/v3/impl/...
rm examples/gno.land/r/gov/dao/v3/impl/filetests/description_40b_filetest.gno
*/

// Gas cost of one proposal-page render with a 40-byte description, which the
// change leaves unclamped and unescaped. Run beside its twin for the slope.
// Measured at 304f09a7a: 1mb 25,115,675, 40b 10,931,522.

// PKGPATH: gno.land/r/test/desc40b
package desc40b

import (
	"strings"
	"testing"

	"gno.land/r/gov/dao"
	"gno.land/r/gov/dao/v3/impl"
	"gno.land/r/gov/dao/v3/memberstore"
)

const user address = "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"

var pid dao.ProposalID

func init(cur realm) {
	memberstore.Get(0, cur).DeleteAll()
	memberstore.Get(0, cur).SetTier(memberstore.T1)
	memberstore.Get(0, cur).SetMember(memberstore.T1, user, memberstore.NewMember(3))
	dao.UpdateImpl(cross(cur), dao.NewUpdateRequest(impl.NewGovDAO(), nil))
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewUserRealm(user))
	pid = dao.MustCreateProposal(cross(cur), dao.NewProposalRequest("T", strings.Repeat("d", 40), nil))
}

func main(cur realm) {
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewUserRealm(user))
	println(len(dao.Render(cross(cur), pid.String())) > 0)
}

// Output:
// true
