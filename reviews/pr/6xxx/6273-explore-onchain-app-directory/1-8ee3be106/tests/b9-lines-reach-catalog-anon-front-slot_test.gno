// Repro, from a plain clone of github.com/gnolang/gno:
//
//	git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//	cp b9-lines-reach-catalog-anon-front-slot_test.gno examples/gno.land/r/gnoland/store/v0/zz_anon_front_test.gno
//	cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run TestZZAnonFrontPage -v 2>&1 | grep -v GAS:
//
// Observed at 8ee3be106: the g1… app is absent from New ("slugs":[] with
// "List yours from a registered namespace"), yet it is the newest activity
// event at once ({"kind":"listed","slug":"anon-app"}) and, after
// updateCooldown, the only slug of the Recently updated shelf, in
// api/v1/home and in the markdown front.

package store

import (
	"strings"
	"testing"

	"gno.land/p/nt/testutils/v0"
)

func shelfOf(home, title string) string {
	_, s, _ := strings.Cut(home, `{"title":"`+title+`"`)
	s, _, _ = strings.Cut(s, `,"more"`)
	return s
}

func activityOf(home string) string {
	_, a, _ := strings.Cut(home, `"activity":`)
	a, _, _ = strings.Cut(a, `]`)
	return a + "]"
}

// A free g1… address namespace: New and Rediscover leave it out, but
// Recently updated and the activity feed (gnoweb's pulse moment) do not.
func TestZZAnonFrontPage(cur realm, t *testing.T) {
	testing.SetHeight(100 * epochBlocks)
	path := "gno.land/r/" + testutils.TestAddress("freeanon").String() + "/app"
	registerFrom(cur, path, "anon-app", "Anon App")

	home := Render("api/v1/home")
	println("after deploy, New shelf:      ", shelfOf(home, "New"))
	println("after deploy, activity:       ", activityOf(home))

	testing.SkipHeights(updateCooldown)
	testing.SetRealm(testing.NewCodeRealm(path))
	Register(cross(cur), "", "Anon App", "Changed tagline", "utilities", 4)

	home = Render("api/v1/home")
	println("after update, New shelf:      ", shelfOf(home, "New"))
	println("after update, Recently upd.:  ", shelfOf(home, "Recently updated"))
	println("after update, activity:       ", activityOf(home))
	md := Render("")
	_, upd, _ := strings.Cut(md, "## Recently updated")
	upd, _, _ = strings.Cut(upd, "See all")
	println("markdown Recently updated:   ", strings.TrimSpace(upd))
}
