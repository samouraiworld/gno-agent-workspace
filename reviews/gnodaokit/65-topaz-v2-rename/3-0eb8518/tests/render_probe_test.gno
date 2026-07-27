/* Run: from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08
cp <this file> gno/p/basedao/render_probe_test.gno
/tmp/gno-topaz test -v -run 'TestRenderProbeEveryPath' ./gno/p/basedao
rm gno/p/basedao/render_probe_test.gno
*/

// Walks every route the default render router registers, with valid and invalid
// input, and prints one line each rather than asserting.
// At 0eb8518 four abort: proposal/999, proposal/abc, proposal/-1 and
// role/nosuchrole. All four abort at the merge-base b8332969 too.

package basedao

import (
	"testing"

	"gno.land/p/samcrew/daocond"
)

func TestRenderProbeEveryPath(cur realm, t *testing.T) {
	members := []Member{
		{alice.String(), []string{"admin"}},
		{bob.String(), []string{}},
	}
	tdao := newTestingDAO(cur, t, 0.6, members)
	tdao.dao.Propose(tdao.mockProposalRequest, cur)
	tdao.setCaller(alice)
	tdao.dao.Vote(1, daocond.VoteYes, cur)

	paths := []string{
		"",
		"members",
		"proposals",
		"noprofile",
		"config",
		"history",
		"member/" + alice.String(),
		"member/g1v9kxjcm9ta047h6lta047h6lta047h6lzd40gh",
		"proposal/1",
		"proposal/999",
		"proposal/abc",
		"proposal/-1",
		"role/admin",
		"role/nosuchrole",
		"proposals/1",
		"roles",
		"nosuchpath",
	}

	for _, p := range paths {
		func() {
			defer func() {
				if r := recover(); r != nil {
					println("PANIC  " + p)
				}
			}()
			out := tdao.privdao.Render(p)
			if len(out) > 40 {
				out = out[:40]
			}
			println("ok     " + p + "  ->  " + out)
		}()
	}
}
