package store

// Repro, from a plain clone (gnolang/gno#6273, head 8ee3be106cd932e3156197c4ce3a080520493eb9):
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//   make -C gnovm install   # or any gno built at that sha
//   cp <this file> examples/gno.land/r/gnoland/store/v0/b8_test.gno
//   cd examples/gno.land/r/gnoland/store/v0
//   gno test -v -run 'TestB8' .
// Measured: ok, four PASS (TestB8TitleHomoglyphs, TestB8PausedSubmitRich,
// TestB8PickKeepsSwappedTagline, TestB8LatinExtBLookalikes).
// Each test asserts the behaviour the finding describes; a PASS confirms it.

import (
	"chain/runtime"
	"strconv"
	"strings"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

// TestB8TitleHomoglyphs: validText admits a mixing-script letter or any
// non-letter beside Latin, and titleKey folds none of them, so a title one
// glyph away from a visible one is listed beside it.
func TestB8TitleHomoglyphs(cur realm, t *testing.T) {
	registerFrom(cur, "gno.land/r/beereal/swap", "bee-real", "Gnoland Swap")
	uassert.True(t, byPath.Has("gno.land/r/beereal/swap"), "real listed")
	for i, title := range []string{
		"Gn০land Swap", // Bengali digit zero for o
		"Gn०land Swap", // Devanagari digit zero for o
		"Gnoاand Swap", // Arabic alef for l
		"Gnסland Swap", // Hebrew samekh for o
		"Gno׀and Swap", // Hebrew paseq for l
	} {
		n := strconv.Itoa(i)
		path := "gno.land/r/beefake" + n + "/swap"
		registerFrom(cur, path, "bee-fake-"+n, title)
		println(n, title, "validText:", validText(title, maxTitle), "listed:", byPath.Has(path), "key:", titleKey(title))
		uassert.True(t, byPath.Has(path), title)
	}
}

// TestB8PausedSubmitRich: paused refuses Register and Claim updates, but
// SubmitRich never reads it, so a listed realm still swaps its icon and
// cover during a pause.
func TestB8PausedSubmitRich(cur realm, t *testing.T) {
	path := "gno.land/r/beepause/app"
	registerFrom(cur, path, "bee-pause", "Bee Pause")
	paused = true // the body of ProposePause's executor
	registerFrom(cur, path, "bee-pause", "Bee Pause Renamed")
	uassert.Equal(t, "Bee Pause", bySlugOrPanic("bee-pause").Title, "a title update is refused while paused")
	testing.SetRealm(testing.NewCodeRealm(path))
	SubmitRich(cross(cur), testIcon, testIcon)
	l := bySlugOrPanic("bee-pause")
	uassert.True(t, l.next != nil, "a rich change is accepted while paused")
	testing.SkipHeights(richDelay)
	r, _ := l.effective(runtime.ChainHeight())
	println("paused:", paused, "effective cover after richDelay:", r.Cover)
	uassert.Equal(t, testIcon, r.Cover)
	paused = false
}

// TestB8PickKeepsSwappedTagline: a pick is bound to the title alone, so the
// app swaps its tagline after the vote and keeps the GovDAO label.
func TestB8PickKeepsSwappedTagline(cur realm, t *testing.T) {
	path := "gno.land/r/beehero/app"
	registerFrom(cur, path, "bee-hero", "Bee Hero")
	h := runtime.ChainHeight()
	uassert.NoError(t, pickBound("bee-hero", path, titleKey("Bee Hero"), 7, h))
	const swapped = "Official GovDAO airdrop, claim yours now"
	testing.SetRealm(testing.NewCodeRealm(path))
	Register(cross(cur), "bee-hero", "Bee Hero", swapped, "utilities", 4)
	uassert.Equal(t, swapped, bySlugOrPanic("bee-hero").Tagline)
	uassert.True(t, picked(runtime.ChainHeight()) == bySlugOrPanic("bee-hero"), "still picked")
	home := Render("")
	i := strings.Index(home, "## Picked by GovDAO")
	uassert.True(t, i >= 0, "pick section")
	section := home[i:]
	if j := strings.Index(section[3:], "## "); j >= 0 {
		section = section[:j+3]
	}
	println(section)
	contains(t, section, swapped)

	// The cover too: a rich change shows a day later, still under the pick.
	testing.SetRealm(testing.NewCodeRealm(path))
	SubmitRich(cross(cur), testIcon, testIcon)
	testing.SkipHeights(richDelay)
	now := runtime.ChainHeight()
	r, _ := bySlugOrPanic("bee-hero").effective(now)
	println("cover in effect:", r.Cover, "picked:", picked(now) != nil)
	uassert.Equal(t, testIcon, r.Cover)
	uassert.True(t, picked(now) == bySlugOrPanic("bee-hero"), "still picked with the new cover")
}

// TestB8LatinExtBLookalikes: commonLatin admits Latin Extended-B below
// U+0250 but the click letters, and titleKey folds none of it, so a pure
// Latin title with LATIN CAPITAL LETTER IOTA for l, or WYNN for p, is
// listed beside the title it copies.
func TestB8LatinExtBLookalikes(cur realm, t *testing.T) {
	registerFrom(cur, "gno.land/r/beeorig/app", "bee-orig", "Gnoland Pay")
	for i, title := range []string{
		"GnoƖand Pay", // LATIN CAPITAL LETTER IOTA for l
		"Gnoland ƿay", // LATIN LETTER WYNN for p
	} {
		n := strconv.Itoa(i)
		path := "gno.land/r/beeextb" + n + "/app"
		registerFrom(cur, path, "bee-extb-"+n, title)
		println(n, title, "validText:", validText(title, maxTitle), "listed:", byPath.Has(path), "key:", titleKey(title), "orig key:", titleKey("Gnoland Pay"))
		uassert.True(t, byPath.Has(path), title)
	}
}
