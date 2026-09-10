// PKGPATH: gno.land/r/gnoland/boards2/v1/filetests/z_open_board_repost_gate_filetest

/* Run: from a gno checkout, about 2 minutes for the whole boards2 suite:
gh pr checkout 6132 -R gnolang/gno && git checkout 02ac71476
curl -fsSL -o examples/gno.land/r/gnoland/boards2/v1/filetests/z_open_board_repost_gate_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6132-boards2-opendiscussions-board/2-02ac71476/tests/z_open_board_repost_gate_filetest.gno
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gnoland/boards2/v1
rm examples/gno.land/r/gnoland/boards2/v1/filetests/z_open_board_repost_gate_filetest.gno
*/

// On the OpenDiscussions board init() seeds, CreateThread refuses a caller
// holding zero ugnot (pinned by the repo's own z_create_thread_06_filetest)
// while CreateRepost accepts the same caller and writes a thread carrying a
// caller-chosen title and body. RequiredAccountAmount is 3_000_000_000 ugnot.
// IS:     the Output: directive below, which the reviewed head produces.
// SHOULD, if the balance gate is meant to cover every public write:
//   // Output:
//   // threads before: 1
//   // repost refused: caller is not allowed to repost: account amount is lower than 3000 GNOT
//   // threads after: 1

package z_open_board_repost_gate_filetest

import (
	"testing"

	boards2 "gno.land/r/gnoland/boards2/v1"
)

const (
	owner address = "g1skl80cuz8zq3lul9pgz5pc35l2pfzgxgfpsqkx" // GovDAO T1 multisig, seeded board owner
	user  address = "g1us8428u2a5satrlxzagqqa5m6vmuze025anjlj" // @test2, never issued any ugnot
	openD         = 1                                          // board ID that init() consumes
)

func init(cur realm) {
	// Owners are exempt from the balance gate, so the owner seeds the thread
	// that the unfunded caller then reposts.
	testing.SetRealm(testing.NewUserRealm(owner))
	boards2.CreateThread(cross(cur), openD, "Seed", "Seed body")
}

func main(cur realm) {
	testing.SetRealm(testing.NewUserRealm(user))

	println("threads before:", len(boards2.GetThreads(openD, 0, 50)))

	// No ugnot was ever issued to user, and this call is not refused.
	rid := boards2.CreateRepost(cross(cur), openD, 1, openD, "Spam title", "Spam body")
	println("repost accepted, thread id:", rid)

	println("threads after:", len(boards2.GetThreads(openD, 0, 50)))
}

// Output:
// threads before: 1
// repost accepted, thread id: 2
// threads after: 2
