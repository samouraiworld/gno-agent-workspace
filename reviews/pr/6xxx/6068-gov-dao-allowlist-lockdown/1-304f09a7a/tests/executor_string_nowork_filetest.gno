/* Run: from a gno checkout:
gh pr checkout 6068 -R gnolang/gno && git checkout 304f09a7a
curl -fsSL -o examples/gno.land/r/gov/dao/v3/impl/filetests/executor_string_nowork_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6068-gov-dao-allowlist-lockdown/1-304f09a7a/tests/executor_string_nowork_filetest.gno
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gov/dao/v3/impl/...
rm examples/gno.land/r/gov/dao/v3/impl/filetests/executor_string_nowork_filetest.gno
*/

// Gas cost of one proposal-page render when the executor's String() loops
// 0 times and returns four bytes. Run beside its twin to read the
// cost of the executor body alone. The gas is the `gno test -v` line for this
// file; maxGasQuery is 3,000,000,000.

// PKGPATH: gno.land/r/test/esnowork
package esnowork

import (
	"testing"

	"gno.land/r/gov/dao"
	"gno.land/r/gov/dao/v3/impl"
	"gno.land/r/gov/dao/v3/memberstore"
)

const user address = "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"

func init(cur realm) {
	memberstore.Get(0, cur).DeleteAll()
	memberstore.Get(0, cur).SetTier(memberstore.T1)
	memberstore.Get(0, cur).SetMember(memberstore.T1, user, memberstore.NewMember(3))
	dao.UpdateImpl(cross(cur), dao.NewUpdateRequest(impl.NewGovDAO(), nil))
}

// Short return value, one int field of stored state, unbounded work.
type ex struct{ N int }

func (e *ex) Execute(cur realm) error { return nil }

func (e *ex) String() string {
	acc := 0
	for i := 0; i < e.N; i++ {
		acc += i
	}
	if acc < 0 {
		return "neg"
	}
	return "meta"
}

func (e *ex) CreationRealm() string { return "gno.land/r/test/esnowork" }

func main(cur realm) {
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewUserRealm(user))
	pid := dao.MustCreateProposal(cross(cur), dao.NewProposalRequest("T", "d", &ex{N: 0}))
	println(len(dao.Render(cross(cur), pid.String())) > 0)
}

// Output:
// true
