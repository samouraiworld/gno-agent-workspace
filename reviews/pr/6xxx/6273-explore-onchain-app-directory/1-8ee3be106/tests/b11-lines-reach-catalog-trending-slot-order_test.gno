// Repro, gnolang/gno#6273 at 8ee3be106: trending() walks appTrend.slots in
// array order (epoch % trendEpochs), so trendingScan entries in an earlier
// slot spend the scan budget before a hotter app in a later slot is read.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//   cp b11-lines-reach-catalog-trending-slot-order_test.gno \
//      examples/gno.land/r/gnoland/store/v0/zz_slot_order_test.gno
//   cd examples/gno.land/r/gnoland/store/v0 && gno test . -run TestTrendingSlotOrderHidesHotApp -v
//
// Expect on the head: FAIL, the hot app (5 ranked stars this week) is absent
// from Trending while 2-star apps fill it.
package store

import (
	"chain/runtime"
	"strconv"
	"strings"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

func TestTrendingSlotOrderHidesHotApp(cur realm, t *testing.T) {
	const e = 1105 // e % 7 == 6: the hot app's slot is the last one walked
	testing.SetHeight(e * epochBlocks)
	registerFrom(cur, "gno.land/r/hotslot/app", "hotslot-app", "Hot Slot")
	starFrom(cur, "hotslot-app", "hsa", "hsb", "hsc", "hsd", "hse")
	hot := bySlugOrPanic("hotslot-app")
	uassert.Equal(t, int64(6), hot.trendEpoch%trendEpochs)

	testing.SetHeight((e + 1) * epochBlocks) // (e+1) % 7 == 0: slot 0, walked first
	for i := 0; i < trendingScan; i++ {
		ns := "slotfill" + strconv.Itoa(i)
		fan := "fill" + strconv.Itoa(i/10) // 10 stars per fan, under maxStarsPerEpoch
		registerFrom(cur, "gno.land/r/"+ns+"/app", ns, "Slot Fill "+strconv.Itoa(i))
		starFrom(cur, ns, fan+"a", fan+"b")
	}
	h := runtime.ChainHeight()
	uassert.Equal(t, 5, hot.stars7d(h), "the hot app is still inside the 7-epoch window")

	top := trending(h, 10)
	t.Log("trending(h, 10): " + jsonSlugs(top.items))
	_, shelf, _ := strings.Cut(Render("api/v1/home"), `{"title":"Trending this week"`)
	shelf, _, _ = strings.Cut(shelf, "]")
	t.Log("home Trending shelf: " + shelf)
	uassert.True(t, len(top.items) > 0 && top.items[0].Slug == "hotslot-app",
		"hot app (5 stars this week) should lead Trending over 2-star apps")
}

// Control, run alone with -run TestTrendingSlotOrderControl: the same 200
// two-star apps, but in slot 6 while the hot app sits in slot 0. Expect PASS:
// only the slot layout changed, and the hot app now leads.
func TestTrendingSlotOrderControl(cur realm, t *testing.T) {
	const e = 1106 // e % 7 == 0: the hot app's slot is walked first
	testing.SetHeight(e * epochBlocks)
	registerFrom(cur, "gno.land/r/hotctl/app", "hotctl-app", "Hot Control")
	starFrom(cur, "hotctl-app", "hca", "hcb", "hcc", "hcd", "hce")
	hot := bySlugOrPanic("hotctl-app")
	testing.SetHeight((e + 6) * epochBlocks) // slot 6, still inside the hot app's window
	for i := 0; i < trendingScan; i++ {
		ns := "ctlfill" + strconv.Itoa(i)
		fan := "ctlf" + strconv.Itoa(i/10)
		registerFrom(cur, "gno.land/r/"+ns+"/app", ns, "Ctl Fill "+strconv.Itoa(i))
		starFrom(cur, ns, fan+"a", fan+"b")
	}
	h := runtime.ChainHeight()
	uassert.Equal(t, 5, hot.stars7d(h))
	top := trending(h, 10)
	t.Log("trending(h, 10): " + jsonSlugs(top.items))
	uassert.Equal(t, "hotctl-app", top.items[0].Slug)
}
