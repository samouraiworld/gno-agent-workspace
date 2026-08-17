/* Run: from a gno checkout:
gh pr checkout 6068 -R gnolang/gno && git checkout 304f09a7a
curl -fsSL -o examples/gno.land/r/gov/dao/v3/impl/filetests/executor_disclosure_spoof_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6068-gov-dao-allowlist-lockdown/1-304f09a7a/tests/executor_disclosure_spoof_filetest.gno
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gov/dao/v3/impl/...
rm examples/gno.land/r/gov/dao/v3/impl/filetests/executor_disclosure_spoof_filetest.gno
*/

// An executor returning "" from CreationRealm() and the disclosure line from
// String() leaves one "Executor created in:" line on the page, and it is the
// executor's own. At 304f09a7a the count is 1; the golden below asserts 2, the
// count once the realm always emits its own line.

// PKGPATH: gno.land/r/test/spoof
package spoof

import (
	"strings"
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

// String() is emitted raw, so it writes the disclosure line itself. CreationRealm()
// returns "" so the realm's own line is dropped by the `cr != ""` guard in render.gno.
type spoofExecutor struct{}

func (e *spoofExecutor) Execute(cur realm) error { return nil }

func (e *spoofExecutor) String() string {
	return "Nothing to see here.\n\nExecutor created in: `gno.land/r/gov/dao/v3/impl`\n"
}

func (e *spoofExecutor) CreationRealm() string { return "" }

func main(cur realm) {
	testing.SetOriginCaller(user)
	testing.SetRealm(testing.NewUserRealm(user))
	pid := dao.MustCreateProposal(cross(cur), dao.NewProposalRequest(
		"Innocent", "desc", &spoofExecutor{}))

	out := dao.Render(cross(cur), pid.String())
	// One line means the executor's forgery is the page's only provenance claim.
	println("disclosure lines on the page:", strings.Count(out, "Executor created in:"))
	println("names a realm the executor does not come from:",
		strings.Contains(out, "Executor created in: `gno.land/r/gov/dao/v3/impl`"))
}

// Output:
// disclosure lines on the page: 2
// names a realm the executor does not come from: true

// disclosure lines on the page: 1   // IS:     the executor's line is the only one
// disclosure lines on the page: 2   // SHOULD: the realm emits its own line regardless
