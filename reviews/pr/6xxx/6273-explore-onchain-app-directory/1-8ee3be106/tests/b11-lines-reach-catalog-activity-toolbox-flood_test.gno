// Repro, gnolang/gno#6273 at 8ee3be106: track() admits every indexed listing
// into the 20-slot activity ring, services and packages included, while
// writeActivity serves app events only. 20 service listings empty the home
// activity feed although app events exist.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//   cp b11-lines-reach-catalog-activity-toolbox-flood_test.gno \
//      examples/gno.land/r/gnoland/store/v0/zz_activity_flood_test.gno
//   cd examples/gno.land/r/gnoland/store/v0 && gno test . -run TestActivityFloodedByServices -v
//
// Expect on the head: FAIL, "activity":[] after the services are listed.
package store

import (
	"strconv"
	"strings"
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func activityOf(home string) string {
	_, a, _ := strings.Cut(home, `"activity":`)
	a, _, _ = strings.Cut(a, `,"listings":`)
	return a
}

func TestActivityFloodedByServices(cur realm, t *testing.T) {
	registerFrom(cur, "gno.land/r/feedapp/app", "feed-app", "Feed App")
	before := activityOf(Render("api/v1/home"))
	t.Log("activity before: " + before)
	uassert.True(t, strings.HasPrefix(before, `[{"kind":"listed","slug":"feed-app"`))

	for i := 0; i < activitySize; i++ {
		owner := testutils.TestAddress("svcflood" + strconv.Itoa(i))
		path := "gno.land/r/" + owner.String() + "/svc"
		title := "Flood Service " + strconv.Itoa(i)
		testing.SetRealm(testing.NewUserRealm(owner))
		Claim(cross(cur), path, "flood-svc-"+strconv.Itoa(i), title, "A tagline", "infrastructure", 1, kindService)
		registerIn(cur, path, "ignored", title, "infrastructure")
		uassert.Equal(t, kindService, bySlugOrPanic("flood-svc-"+strconv.Itoa(i)).Kind)
	}
	after := activityOf(Render("api/v1/home"))
	t.Log("activity after: " + after)
	uassert.True(t, strings.HasPrefix(after, `[{"kind":"listed","slug":"feed-app"`),
		"the newest app event should survive service listings, which the feed never serves")
}
