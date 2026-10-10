// PKGPATH: gno.land/r/test/selfstar
package selfstar

// Repro (from a plain clone of gnolang/gno at 8ee3be106cd932e3156197c4ce3a080520493eb9):
//   cp b12-lines-reach-catalog-names-paused-selfstar_filetest.gno \
//      examples/gno.land/r/gnoland/store/v0/filetests/b12_names_paused_selfstar_filetest.gno
//   export GNOROOT=$PWD; cd examples
//   gno test ./gno.land/r/gnoland/store/v0/ -v 2>&1 | grep -A2 'RUN.*b12_names_paused'
// (-run does not select filetests; the whole package runs.) PASS pins the
// defect: the second badge reads 1, the owner's own star ranked.
//
// Star's own-namespace guard is !ownsNamespace(caller, l.Namespace). While
// r/sys/names is enabled and paused, IsAuthorizedAddressForNamespace answers
// false for every address, so the owner's own star ranks.

import (
	"chain/runtime/unsafe"
	"testing"

	"gno.land/p/nt/testutils/v0"
	store "gno.land/r/gnoland/store/v0"
	"gno.land/r/gov/dao"
	daoinit "gno.land/r/gov/dao/init/v0"
	"gno.land/r/sys/names"
	"gno.land/r/sys/users"
)

var c address = unsafe.OriginCaller()

func init(cur realm) {
	daoinit.InitWithUsers(cross(cur), c) // GovDAO with c as its only T1 member
}

func main(cur realm) {
	owner := testutils.TestAddress("selfowner")
	ns := owner.String()
	pathA := "gno.land/r/" + ns + "/a"
	pathB := "gno.land/r/" + ns + "/b"

	testing.SetRealm(testing.NewUserRealm("g1skl80cuz8zq3lul9pgz5pc35l2pfzgxgfpsqkx")) // r/sys/names admin
	names.Enable(cross(cur))

	testing.SetRealm(testing.NewCodeRealm("gno.land/r/sys/users/init"))
	if err := users.RegisterUserIgnoreCanonical(cross(cur), "selfowner", owner); err != nil {
		panic(err)
	}

	testing.SetRealm(testing.NewCodeRealm(pathA))
	store.Register(cross(cur), "own-a", "Own A", "A tagline", "utilities", 0)
	testing.SetRealm(testing.NewCodeRealm(pathB))
	store.Register(cross(cur), "own-b", "Own B", "A tagline", "utilities", 0)

	// The store sees the owner's name, then the name matures (nameMaturity).
	testing.SetRealm(testing.NewUserRealm(owner))
	store.Star(cross(cur), "blog")
	testing.SkipHeights(3 * 17280)

	// Control: names enabled and live, the owner's star on its own app does not rank.
	testing.SetRealm(testing.NewUserRealm(owner))
	store.Star(cross(cur), "own-a")
	println("live owns:", names.IsAuthorizedAddressForNamespace(owner, ns))
	println("live badge:", store.Badge(pathA))

	// GovDAO pauses r/sys/names, the deploy-only emergency halt.
	testing.SetOriginCaller(c)
	testing.SetRealm(testing.NewUserRealm(c))
	pid := dao.MustCreateProposal(cross(cur), names.ProposeSetPaused(cross(cur), true))
	dao.MustVoteOnProposal(cross(cur), dao.NewVoteRequest(dao.YesVote, pid))
	dao.ExecuteProposal(cross(cur), pid)

	testing.SetOriginCaller(owner)
	testing.SetRealm(testing.NewUserRealm(owner))
	store.Star(cross(cur), "own-b")
	println("paused owns:", names.IsAuthorizedAddressForNamespace(owner, ns))
	println("paused badge:", store.Badge(pathB))
}

// Output:
// live owns: true
// live badge: ★ 0 on Explore · [Star](/r/gnoland/store/v0$help&func=Star&slug=own-a)
// paused owns: false
// paused badge: ★ 1 on Explore · [Star](/r/gnoland/store/v0$help&func=Star&slug=own-b)
