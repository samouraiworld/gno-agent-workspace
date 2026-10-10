// Repro: gnoweb Spotlight pool truncated in shelf order (view.go:263, :338).
// eligible() keeps the first spotlightPool (12) eligible apps in the realm's
// shelf order, and the realm sends New first, so 12 eligible apps on New
// hide every app of Top and Trending from the operator cap and the trending
// hero.
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//   cp <this file> gno.land/pkg/gnoweb/feature/store/zz_b7_featured_test.go
//   go test ./gno.land/pkg/gnoweb/feature/store/ -run 'TestB7FeaturedCapBehindTrustedNew|TestB7TrendingHeroBehindTrustedNew' -v
//
// Expected at 8ee3be106: both FAIL, "day 0: hero+spot=[t01 t02 t03 t04] trusted=4"
// and "hero=t01 label=\"In the spotlight today\"".
package store

import (
	"fmt"
	"testing"
)

// The doc of featured: once spotlightSize apps have earned their place, the
// operator's own apps take one place at most, hero included. Here 4 apps have
// earned theirs, on the second shelf (Top), behind 12 trusted apps on New.
func TestB7FeaturedCapBehindTrustedNew(t *testing.T) {
	var ls []listing
	var newSlugs, topSlugs []string
	for i := 1; i <= 12; i++ {
		s := fmt.Sprintf("t%02d", i)
		ls = append(ls, listing{Slug: s, Kind: kindApp, tier: tierTrusted})
		newSlugs = append(newSlugs, s)
	}
	for i := 1; i <= 4; i++ {
		s := fmt.Sprintf("e%d", i)
		ls = append(ls, listing{Slug: s, Kind: kindApp, tier: tierRegistered, Earned: true})
		topSlugs = append(topSlugs, s)
	}
	bySlug := indexListings(ls)
	for day := int64(0); day < 4; day++ {
		res := &homeResponse{Shelves: []shelf{{Slugs: newSlugs}, {Slugs: topSlugs}}}
		res.Height = day * blocksPerDay
		hero, _, spot := featured(res, bySlug)
		shown := append([]*listing{hero}, spot...)
		trusted := 0
		var names []string
		for _, l := range shown {
			names = append(names, l.Slug)
			if l.tier == tierTrusted {
				trusted++
			}
		}
		t.Logf("day %d: hero+spot=%v trusted=%d", day, names, trusted)
		if trusted > 1 {
			t.Errorf("day %d: %d operator apps among hero and Spotlight, want at most 1 once 4 apps earned their place", day, trusted)
		}
	}
}

// The doc of featured: else the eligible app with the most momentum this week.
// An earned app leads the trending shelf with 50 stars this week, but sits
// behind 12 trusted apps of New in shelf order.
func TestB7TrendingHeroBehindTrustedNew(t *testing.T) {
	var ls []listing
	var newSlugs []string
	for i := 1; i <= 12; i++ {
		s := fmt.Sprintf("t%02d", i)
		ls = append(ls, listing{Slug: s, Kind: kindApp, tier: tierTrusted})
		newSlugs = append(newSlugs, s)
	}
	ls = append(ls, listing{Slug: "hot", Kind: kindApp, tier: tierRegistered, Earned: true})
	trendingShelf := shelf{Slugs: []string{"hot"}, Stars7d: []int{50}, More: []listRef{{Key: "trending"}}}
	res := &homeResponse{Shelves: []shelf{{Slugs: newSlugs}, trendingShelf}}
	hero, label, _ := featured(res, indexListings(ls))
	t.Logf("hero=%s label=%q", hero.Slug, label)
	if hero.Slug != "hot" {
		t.Errorf("hero %s (%q), want the eligible app with the most momentum, hot", hero.Slug, label)
	}
}
