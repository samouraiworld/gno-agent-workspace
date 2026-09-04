// PKGPATH: gno.land/r/gnoland/boards2/v1/filetests/z_open_board_gate_00_filetest

/* Run: from a gno checkout, about 5 minutes for the whole boards2 suite:
gh pr checkout 6132 -R gnolang/gno && git checkout 93a0bea09
curl -fsSL -o examples/gno.land/r/gnoland/boards2/v1/filetests/z_open_board_gate_00_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6132-boards2-opendiscussions-board/1-93a0bea09/tests/z_open_board_gate_00_filetest.gno
go run ./gnovm/cmd/gno test -C examples -v ./gno.land/r/gnoland/boards2/v1
rm examples/gno.land/r/gnoland/boards2/v1/filetests/z_open_board_gate_00_filetest.gno
*/

// init() leaves exactly one board, and a holder of 100 GNOT is refused on it.
// RequiredAccountAmount is 3_000_000_000 ugnot, so the caller is 2,900 GNOT short.
// IS:     the Error: directive below, which the reviewed head produces.
// SHOULD, if ordinary users are meant to post without a GovDAO transaction:
//   // Output:
//   // boards after init: 1
//   // thread 1

package z_open_board_gate_00_filetest

import (
	"chain"
	"testing"

	boards2 "gno.land/r/gnoland/boards2/v1"
)

const user address = "g1us8428u2a5satrlxzagqqa5m6vmuze025anjlj"

func main(cur realm) {
	// OpenDiscussions is board 1, the only board init() created.
	println("boards after init:", boards2.BoardCount())

	testing.IssueCoins(user, chain.Coins{{"ugnot", 100_000_000}})
	testing.SetRealm(testing.NewUserRealm(user))
	println("thread", boards2.CreateThread(cross(cur), 1, "hello", "body"))
}

// Output:
// boards after init: 1

// Error:
// caller is not allowed to create threads: account amount is lower than 3000 GNOT
