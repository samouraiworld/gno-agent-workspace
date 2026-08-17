/* Run: from a gno checkout:
gh pr checkout 6068 -R gnolang/gno && git checkout 304f09a7a
curl -fsSL -o examples/gno.land/r/gov/dao/v3/impl/filetests/executor_string_double_dispatch_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6068-gov-dao-allowlist-lockdown/1-304f09a7a/tests/executor_string_double_dispatch_filetest.gno
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gov/dao/v3/impl/...
rm examples/gno.land/r/gov/dao/v3/impl/filetests/executor_string_double_dispatch_filetest.gno
*/

// renderProposalPage evaluates p.ExecutorString() once for the != "" test and
// again for the Sprintf, so an executor's method body runs twice per page
// render. At 304f09a7a the counter reads 2; the golden asserts 1.

// PKGPATH: gno.land/r/test/dispatch
package dispatch

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

var calls int

// counted stands in for any third-party executor: String() is its own code,
// dispatched through the public dao.Executor interface on the render path.
type counted struct{}

func (e *counted) Execute(cur realm) error { return nil }

func (e *counted) String() string {
	calls++
	return "metadata"
}

func (e *counted) CreationRealm() string { return "gno.land/r/test/dispatch" }

func main(cur realm) {
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewUserRealm(user))
	pid := dao.MustCreateProposal(cross(cur), dao.NewProposalRequest("T", "d", &counted{}))

	calls = 0
	dao.Render(cross(cur), pid.String())
	println("Executor.String() dispatches per proposal-page render:", calls)
}

// Output:
// Executor.String() dispatches per proposal-page render: 1

// dispatches: 2   // IS:     the executor's body is paid for twice
// dispatches: 1   // SHOULD: hoisted into a local and reused
