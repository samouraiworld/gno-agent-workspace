# Review: [#6273](https://github.com/gnolang/gno/pull/6273)

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Fourteen Warnings sit in code the branch adds: five title look-alikes that list beside the original, a front-page shelf and an activity feed that address namespaces and toolbox listings can take over, a Trending scan that drops the top app by slot order, an ownership check that fails open during a names pause, a hero and Spotlight pool cut before earned apps are counted, a page error that is never logged, a clipped focus ring, and a Claim snippet that fails as written.
Model: claude-opus-5-5, standard review: finders at xhigh, judges, reflector and triage at high, writers at medium, text pass at high
Overview: [overview](../overview.md)
Commit: 8ee3be106cd932e3156197c4ce3a080520493eb9
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6273 8ee3be106`
Open the code: [gh](https://github.com/gnolang/gno/blob/8ee3be106cd932e3156197c4ce3a080520493eb9)
Round: 1. 13 finders, one reflector, 31 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 2 refuted, none of them above Nit.

## examples/gno.land/r/gnoland/store/v0/front.gno:64 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/front.gno#L64) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/front.gno#L64) · Warning
`listShelf(mustList("updated", h))` reads `apps.updated`, which [`touch`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L244-L255) fills with every indexed app whatever its namespace. A g1 address-namespace app thus gets a Recently updated card by changing its tagline, a front-page slot the [`newApps` comment](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/front.gno#L39-L42) denies those namespaces.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > examples/gno.land/r/gnoland/store/v0/zz_anon_front_test.gno <<'EOF'
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

func TestZZAnonFrontPage(cur realm, t *testing.T) {
	testing.SetHeight(100 * epochBlocks)
	path := "gno.land/r/" + testutils.TestAddress("freeanon").String() + "/app"
	registerFrom(cur, path, "anon-app", "Anon App")

	testing.SkipHeights(updateCooldown)
	testing.SetRealm(testing.NewCodeRealm(path))
	Register(cross(cur), "", "Anon App", "Changed tagline", "utilities", 4)

	home := Render("api/v1/home")
	println("after update, New shelf:      ", shelfOf(home, "New"))
	println("after update, Recently upd.:  ", shelfOf(home, "Recently updated"))
	md := Render("")
	_, upd, _ := strings.Cut(md, "## Recently updated")
	upd, _, _ = strings.Cut(upd, "See all")
	println("markdown Recently updated:   ", strings.TrimSpace(upd))
}
EOF
(cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run TestZZAnonFrontPage -v 2>&1 | grep -v GAS:)
rm examples/gno.land/r/gnoland/store/v0/zz_anon_front_test.gno
```

The g1 app is absent from New and is the only card of Recently updated, in `api/v1/home` and in the markdown front:

```
after update, New shelf:       ..."slugs":[]
after update, Recently upd.:   ,"slugs":["anon-app"]
markdown Recently updated:    - **[Anon App](/r/g1veex2etpdehkuh6lta047h6lta047h6l2cyn54/app)** — Changed tagline
```

`newApps` reads `namedApps`, which [`index`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L301) fills only when `!address(l.Namespace).IsValid()`. The cooldown applies per listing and keys are free, so staggered g1 listings can hold every card gnoweb draws on that shelf.
</details>

## examples/gno.land/r/gnoland/store/v0/listing.gno:554 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L554) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/listing.gno#L554) · Warning
`titleKey` lowercases the title before [`foldRune`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L566) sees it, so Greek `Ν` and `Υ` [fold to `v` and `u`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L602-L605) and `Μ` and `Ζ` keep their own runes. An all-Greek `ΤΟΚΕΝ` thus keys as `tokev` and registers beside `TOKEN`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cd examples/gno.land/r/gnoland/store/v0
cat > zz_greek_fold_test.gno <<'EOF'
package store

import "testing"

// All-Greek capitals drawn exactly like the Latin word on the left.
var greekTwins = [][2]string{
	{"TOKEN", "ΤΟΚΕΝ"}, // Tau Omicron Kappa Epsilon Nu
	{"ZONE", "ΖΟΝΕ"},   // Zeta Omicron Nu Epsilon
	{"MINT", "ΜΙΝΤ"},   // Mu Iota Nu Tau
	{"KEY", "ΚΕΥ"},     // Kappa Epsilon Upsilon
	{"BOX", "ΒΟΧ"},     // control: every letter folds right
}

func TestB1GreekCapitalsFold(t *testing.T) {
	for _, p := range greekTwins {
		if !validText(p[1], maxTitle) {
			t.Logf("%s: refused by validText, no impersonation", p[0])
			continue
		}
		if titleKey(p[0]) != titleKey(p[1]) {
			t.Errorf("%s: Greek twin accepted with a distinct key: %q vs %q", p[0], titleKey(p[0]), titleKey(p[1]))
		}
	}
}

func TestB1GreekTwinRegisters(cur realm, t *testing.T) {
	registerFrom(cur, "gno.land/r/acme/token", "acme-token", "TOKEN")
	registerFrom(cur, "gno.land/r/mallory/token", "mallory-token", "ΤΟΚΕΝ")
	if lookup(&listings, "mallory-token") != nil {
		t.Errorf("a second app took a title drawn as TOKEN: %q", lookup(&listings, "mallory-token").Title)
	}
}
EOF
gno test . -run TestB1Greek -v
rm zz_greek_fold_test.gno
```

Both tests fail: four Greek twins key apart from their Latin original, the `BOX` control keys the same, and the second app titled `ΤΟΚΕΝ` lists.

```
TOKEN: Greek twin accepted with a distinct key: "token" vs "tokev"
ZONE: Greek twin accepted with a distinct key: "zone" vs "ζove"
MINT: Greek twin accepted with a distinct key: "mlnt" vs "μlvt"
KEY: Greek twin accepted with a distinct key: "key" vs "keu"
# …
a second app took a title drawn as TOKEN: "ΤΟΚΕΝ"
--- FAIL
```
</details>

## examples/gno.land/r/gnoland/store/v0/listing.gno:567 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L567) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/listing.gno#L567) · Warning
`foldRune` has no case for a Lisu or Cherokee letter, and [`validText`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/validate.gno#L151) refuses those scripts only beside a Latin letter. So an all-Lisu or all-Cherokee `ACME` keys apart from the Latin one and lists beside it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cd examples/gno.land/r/gnoland/store/v0
cat > zz_confusables_test.gno <<'EOF'
package store

import (
	"chain/runtime"
	"testing"
)

func TestB10ConfusableTitlesList(cur realm, t *testing.T) {
	h := runtime.ChainHeight()
	list := func(path, slug, title string) string {
		l := newListing(path, slug, title, "A tagline", "utilities", 4, kindApp)
		l.Confirmed = true
		return upsert(l, h)
	}
	if r := list("gno.land/r/latin/acme", "latin-acme", "ACME"); r != "" {
		t.Fatalf("setup: %s", r)
	}
	cases := []struct{ label, original, twin string }{
		{"all Lisu", "ACME", "ꓮꓚꓟꓰ"},
		{"all Cherokee", "ACME", "ᎪᏟᎷᎬ"},
		{"myanmar zero", "Boards", "B၀ards"},
		{"devanagari zero", "Gno.land Blog", "Gn०.land Blog"},
		{"divides as l", "Gno.land Blog", "Gno.∣and Blog"},
	}
	for i, tc := range cases {
		path := "gno.land/r/twin" + string(rune('a'+i)) + "/app"
		got := list(path, "twin-"+string(rune('a'+i)), tc.twin)
		t.Logf("%-16s validText=%v titleKey(%q)=%q titleKey(%q)=%q reason=%q",
			tc.label, validText(tc.twin, maxTitle), tc.original, titleKey(tc.original), tc.twin, titleKey(tc.twin), got)
		if got != reasonTitleTaken {
			t.Errorf("%s: %q lists beside %q: got reason %q, want %q", tc.label, tc.twin, tc.original, got, reasonTitleTaken)
		}
	}
}
EOF
gno test . -run TestB10Confusable -v
rm zz_confusables_test.gno
```

The Lisu and Cherokee cases fail: each twin passes `validText`, keys apart from `acme`, and lists with an empty reason after `ACME` was listed.

```
all Lisu         validText=true titleKey("ACME")="acme" titleKey("ꓮꓚꓟꓰ")="ꓮꓚꓟꓰ" reason=""
all Cherokee     validText=true titleKey("ACME")="acme" titleKey("ᎪᏟᎷᎬ")="ꭺꮯꮇꭼ" reason=""
# …
--- FAIL
```
</details>

## examples/gno.land/r/gnoland/store/v0/listing.gno:572-574 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L572-L574) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/listing.gno#L572) · Warning
The `'0'` and `'1', '|'` cases of `foldRune` fold only the ASCII digits and bar, and [`validText`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/validate.gno#L145-L146) flags a foreign script only on a letter. So `B၀ards`, with a Myanmar zero, lists beside the seeded [`Boards`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/seed.gno#L13), and a Devanagari zero or U+2223 DIVIDES does the same to [`Gno.land Blog`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/seed.gno#L14).

<details><summary>repro</summary>

The same test as the Lisu and Cherokee section on `foldRune`, its `myanmar zero`, `devanagari zero` and `divides as l` cases:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cd examples/gno.land/r/gnoland/store/v0
cat > zz_confusables_test.gno <<'EOF'
package store

import (
	"chain/runtime"
	"testing"
)

func TestB10ConfusableTitlesList(cur realm, t *testing.T) {
	h := runtime.ChainHeight()
	list := func(path, slug, title string) string {
		l := newListing(path, slug, title, "A tagline", "utilities", 4, kindApp)
		l.Confirmed = true
		return upsert(l, h)
	}
	if r := list("gno.land/r/latin/acme", "latin-acme", "ACME"); r != "" {
		t.Fatalf("setup: %s", r)
	}
	cases := []struct{ label, original, twin string }{
		{"all Lisu", "ACME", "ꓮꓚꓟꓰ"},
		{"all Cherokee", "ACME", "ᎪᏟᎷᎬ"},
		{"myanmar zero", "Boards", "B၀ards"},
		{"devanagari zero", "Gno.land Blog", "Gn०.land Blog"},
		{"divides as l", "Gno.land Blog", "Gno.∣and Blog"},
	}
	for i, tc := range cases {
		path := "gno.land/r/twin" + string(rune('a'+i)) + "/app"
		got := list(path, "twin-"+string(rune('a'+i)), tc.twin)
		t.Logf("%-16s validText=%v titleKey(%q)=%q titleKey(%q)=%q reason=%q",
			tc.label, validText(tc.twin, maxTitle), tc.original, titleKey(tc.original), tc.twin, titleKey(tc.twin), got)
		if got != reasonTitleTaken {
			t.Errorf("%s: %q lists beside %q: got reason %q, want %q", tc.label, tc.twin, tc.original, got, reasonTitleTaken)
		}
	}
}
EOF
gno test . -run TestB10Confusable -v
rm zz_confusables_test.gno
```

The three digit and symbol cases fail: each twin passes `validText`, keeps its look-alike rune in the key, and lists with an empty reason.

```
# …
myanmar zero     validText=true titleKey("Boards")="boards" titleKey("B၀ards")="b၀ards" reason=""
# … devanagari zero: titleKey("Gn०.land Blog")="gn०landblog" reason=""
# … divides as l:    titleKey("Gno.∣and Blog")="gno∣andblog" reason=""
--- FAIL
```
</details>

## examples/gno.land/r/gnoland/store/v0/pulse.gno:62 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/pulse.gno#L62) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/pulse.gno#L62) · Warning
`activity.push` runs for services and packages as well as apps, and [`writeActivity`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/api.gno#L188) serves app events only. So 20 service or package events after the newest app event fill the [20-slot ring](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/pulse.gno#L5), and the home activity feed shows nothing.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > examples/gno.land/r/gnoland/store/v0/zz_activity_flood_test.gno <<'EOF'
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
EOF
(cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run TestActivityFloodedByServices -v)
rm examples/gno.land/r/gnoland/store/v0/zz_activity_flood_test.gno
```

The test fails because the app event the feed would serve has been evicted by service events it never serves:

```
activity before: [{"kind":"listed","slug":"feed-app","stars":0},{"kind":"listed","slug":"valopers",...}]
activity after: []
--- FAIL: TestActivityFloodedByServices (0.21s)
```

`indexed()` is `!l.hidden && l.Confirmed` and lets every kind through; `shownApp()` adds `l.Kind == kindApp`. From 16 non-app events after the newest app events the feed shows fewer than 5 items, and from 20 it shows none.
</details>

## examples/gno.land/r/gnoland/store/v0/shelves.gno:144 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/shelves.gno#L144) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/shelves.gno#L144) · Warning
`range appTrend.slots` walks the slots in array order under one [`trendingScan`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/shelves.gno#L16) budget of 200, so 200 two-star apps in an earlier slot end the scan before it reaches a later slot. Trending then drops the week's top app for the day it was starred, not for its stars.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > examples/gno.land/r/gnoland/store/v0/zz_slot_order_test.gno <<'EOF'
package store

import (
	"chain/runtime"
	"strconv"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

// The hot app (5 ranked stars) sits in slot 6, 200 two-star apps in slot 0.
func TestTrendingSlotOrderHidesHotApp(cur realm, t *testing.T) {
	const e = 1105 // e % 7 == 6
	testing.SetHeight(e * epochBlocks)
	registerFrom(cur, "gno.land/r/hotslot/app", "hotslot-app", "Hot Slot")
	starFrom(cur, "hotslot-app", "hsa", "hsb", "hsc", "hsd", "hse")
	hot := bySlugOrPanic("hotslot-app")
	testing.SetHeight((e + 1) * epochBlocks) // slot 0, walked first
	for i := 0; i < trendingScan; i++ {
		ns := "slotfill" + strconv.Itoa(i)
		fan := "fill" + strconv.Itoa(i/10)
		registerFrom(cur, "gno.land/r/"+ns+"/app", ns, "Slot Fill "+strconv.Itoa(i))
		starFrom(cur, ns, fan+"a", fan+"b")
	}
	h := runtime.ChainHeight()
	uassert.Equal(t, 5, hot.stars7d(h))
	top := trending(h, 10)
	t.Log("trending(h, 10): " + jsonSlugs(top.items))
	uassert.True(t, len(top.items) > 0 && top.items[0].Slug == "hotslot-app")
}

// Control: the same data with the slots swapped.
func TestTrendingSlotOrderControl(cur realm, t *testing.T) {
	const e = 1106 // e % 7 == 0
	testing.SetHeight(e * epochBlocks)
	registerFrom(cur, "gno.land/r/hotctl/app", "hotctl-app", "Hot Control")
	starFrom(cur, "hotctl-app", "hca", "hcb", "hcc", "hcd", "hce")
	hot := bySlugOrPanic("hotctl-app")
	testing.SetHeight((e + 6) * epochBlocks)
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
EOF
(cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run 'TestTrendingSlotOrderHidesHotApp$' -v)
(cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run 'TestTrendingSlotOrderControl$' -v)
rm examples/gno.land/r/gnoland/store/v0/zz_slot_order_test.gno
```

The first run fails because the 5-star app is missing from Trending while 2-star apps fill it; the control, identical apart from the slot layout, passes:

```
trending(h, 10): ["slotfill0","slotfill1","slotfill10",...]
--- FAIL: TestTrendingSlotOrderHidesHotApp
# …
trending(h, 10): ["hotctl-app","ctlfill0",...]
--- PASS: TestTrendingSlotOrderControl
```

The pruning test is `bound < best.scores[best.n-1]`, so entries tied with the n-th best score are never pruned and each one spends the budget. Reaching this needs 200 apps at 2 or more ranked stars filed in one epoch.
</details>

## examples/gno.land/r/gnoland/store/v0/store.gno:117 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store.gno#L117) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/store.gno#L117) · Warning
`ownsNamespace` returns `names.IsAuthorizedAddressForNamespace`, and that call [answers false for every address](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/sys/names/verifier.gno#L231) during a GovDAO pause of `r/sys/names`. During a pause, [`Star`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store.gno#L170) therefore stores an owner's star on their own app as ranked, and that star keeps counting after the pause.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > examples/gno.land/r/gnoland/store/v0/filetests/zz_names_paused_selfstar_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/test/selfstar
package selfstar

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
	daoinit.InitWithUsers(cross(cur), c)
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

	testing.SetRealm(testing.NewUserRealm(owner))
	store.Star(cross(cur), "blog")
	testing.SkipHeights(3 * 17280)

	testing.SetRealm(testing.NewUserRealm(owner))
	store.Star(cross(cur), "own-a")
	println("live owns:", names.IsAuthorizedAddressForNamespace(owner, ns))
	println("live badge:", store.Badge(pathA))

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
EOF
(cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0/ -v 2>&1 | grep -A2 'RUN.*zz_names_paused')
rm examples/gno.land/r/gnoland/store/v0/filetests/zz_names_paused_selfstar_filetest.gno
```

The filetest passes, which pins the defect: the owner's own star reads ★ 0 while names is live and ★ 1 once it is paused.

```
--- PASS: ./gno.land/r/gnoland/store/v0/zz_names_paused_selfstar_filetest.gno
```

The ranked star counts toward Top, Trending, builder totals and one of the 3 stars that confirm the owner's listing. The same call makes [`Claim`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store.gno#L101) refuse every owner for the length of the pause.
</details>

## SKIP examples/gno.land/r/gnoland/store/v0/validate.gno:145 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/validate.gno#L145) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/validate.gno#L145) · Warning
The script-mixing check sets `foreign` only for a letter outside `mixesWithLatin`, and [`foldRune`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L572-L575) leaves Bengali `০` and Hebrew paseq `׀` unfolded while folding ASCII `0` and `|`. `Gn০land Swap` thus lists beside `Gnoland Swap`, where `Gn0land Swap` collides with it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > examples/gno.land/r/gnoland/store/v0/zz_homoglyph_test.gno <<'EOF'
package store

import (
	"strconv"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

// Asserts the defect: each look-alike is listed beside the original.
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
EOF
(cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run 'TestB8TitleHomoglyphs$' -v)
rm examples/gno.land/r/gnoland/store/v0/zz_homoglyph_test.gno
```

The test asserts that every copy lists, and it passes: each look-alike gets a key distinct from `gnolandswap`.

```
0 Gn০land Swap validText: true listed: true key: gn০landswap
1 Gn०land Swap validText: true listed: true ...
2 Gnoاand Swap validText: true listed: true ...
3 Gnסland Swap validText: true listed: true ...
4 Gno׀and Swap validText: true listed: true key: gno׀andswap
--- PASS: TestB8TitleHomoglyphs
```

The Arabic alef and Hebrew samekh rows are letters of scripts allowed next to Latin, which ADR-004 T6 leaves to a later confusables skeleton. The digit and paseq rows are not letters at all, and they defeat the `0` and `|` folds `foldRune` already ships.
</details>

Not posted: the same defect as the section on listing.gno:572-574; folding non-ASCII digits and bars in `foldRune`, or flagging a foreign non-letter in `validText`, closes both.

## examples/gno.land/r/gnoland/store/v0/validate.gno:177 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/validate.gno#L177) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/validate.gno#L177) · Warning
`commonLatin` admits every Latin Extended-B letter below U+0250 except the four click letters, and [`foldRune`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L572-L573) folds neither LATIN CAPITAL LETTER IOTA `Ɩ` nor WYNN `ƿ`. `GnoƖand Pay` and `Gnoland ƿay` thus list beside `Gnoland Pay`, where `GnoIand Pay` with an ASCII `I` collides.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > examples/gno.land/r/gnoland/store/v0/zz_latin_extb_test.gno <<'EOF'
package store

import (
	"strconv"
	"testing"

	"gno.land/p/nt/uassert/v0"
)

// Asserts the defect: both copies are listed beside the original.
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
EOF
(cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run 'TestB8LatinExtBLookalikes$' -v)
rm examples/gno.land/r/gnoland/store/v0/zz_latin_extb_test.gno
```

The test asserts that both copies list, and it passes: neither key matches `gnolandpay`.

```
0 GnoƖand Pay validText: true listed: true key: gnoɩandpay orig key: gnolandpay
1 Gnoland ƿay validText: true listed: true key: gnolandƿay orig key: gnolandpay
--- PASS: TestB8LatinExtBLookalikes
```

The same expression already drops U+01C0 to U+01C3 for this look-alike reason, and its comment names Romanian and Pinyin as the Extended-B letters it is for.
</details>

## examples/gno.land/r/gnoland/store/v0/validate.gno:185 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/validate.gno#L185) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/validate.gno#L185) · Warning
`isMimic` stops at the Mathematical Alphanumeric Symbols block. So `validText` accepts the double-struck, script and fraktur letters of Letterlike Symbols, U+2102 to U+214F, and each keeps its own key: `ℤℴℴℳ` lists beside `Zoom`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > examples/gno.land/r/gnoland/store/v0/zz_letterlike_test.gno <<'EOF'
package store

import "testing"

func TestB1LetterlikeMathLetters(t *testing.T) {
	for _, s := range []string{
		"ℤℴℴℳ", // script/double-struck "Zoom"
		"ℂℍℕ",  // double-struck "CHN"
	} {
		if validText(s, maxTitle) {
			t.Errorf("%q accepted though T6 says mathematical alphanumerics are refused", s)
		}
	}
	// control: the block isMimic covers is refused
	if validText("\U0001D400\U0001D401", maxTitle) {
		t.Errorf("control failed: U+1D400 accepted")
	}
}
EOF
(cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run TestB1LetterlikeMathLetters -v)
rm examples/gno.land/r/gnoland/store/v0/zz_letterlike_test.gno
```

The test fails because both Letterlike titles are accepted, while the U+1D400 control is refused:

```
"ℤℴℴℳ" accepted though T6 says mathematical alphanumerics are refused
"ℂℍℕ" accepted though T6 says mathematical alphanumerics are refused
--- FAIL: TestB1LetterlikeMathLetters
```

These letters are script Common, so the Latin branch never runs, and a title with no Latin letter passes `alnum && (!latin || !foreign)`. Mixing one with ASCII or with U+1D400 is refused, so only words spelled from about 25 such letters get through.
</details>

## gno.land/pkg/gnoweb/feature/store/api.go:208 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/api.go#L208) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/api.go#L208) · Warning
`paged` builds the `answered` error after `decode` returns, so the [deferred `Warn`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/api.go#L145-L149) inside `decode` never logs it, and [`servePage`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/handler.go#L142-L145) drops it silently. The page then falls back to markdown with no log line whenever the realm answers with the wrong key, page or page count.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > gno.land/pkg/gnoweb/feature/store/zz_paged_unlogged_test.go <<'EOF'
package store

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
)

func TestB3PagedMismatchIsNotLogged(t *testing.T) {
	var logs bytes.Buffer
	home := `{"version":1,"categories":[{"key":"defi","label":"DeFi","count":30}],"shelves":[],"listings":[]}`
	c := &fakeClient{responses: map[string]string{
		"api/v1/home": home,
		// The realm answers page 1 of another category: paged() refuses it.
		"api/v1/category/defi/1": `{"version":1,"category":{"key":"games","label":"Games","count":30},"page":1,"pages":2,"listings":[]}`,
		// Control: a truncated body, refused inside decode().
		"api/v1/category/defi/2": `{"version":1,"category":`,
	}}
	h := New(Deps{
		Client:    c,
		RealmPath: "/r/gnoland/store",
		Domain:    "gno.land",
		Trusted:   func(string) bool { return false },
		Logger:    slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	for _, target := range []string{":c/defi", ":c/defi?page=2"} {
		u, err := weburl.Parse("/r/gnoland/store" + target)
		if err != nil {
			t.Fatal(err)
		}
		view, m := h.View(context.Background(), u)
		t.Logf("%s: view nil=%v status=%d", target, view == nil, m.Status)
	}
	out := logs.String()
	t.Logf("log output:\n%s", out)
	if !strings.Contains(out, "category/defi/2") {
		t.Fatalf("control: the decode failure was not logged either")
	}
	if !strings.Contains(out, "answered") {
		t.Errorf("echo mismatch on category/defi/1 left no log line")
	}
}
EOF
go test -count=1 -run TestB3PagedMismatchIsNotLogged -v ./gno.land/pkg/gnoweb/feature/store/
rm gno.land/pkg/gnoweb/feature/store/zz_paged_unlogged_test.go
```

The test fails because both pages fall back, and the only log line is the control's decode failure:

```
:c/defi: view nil=true status=0
:c/defi?page=2: view nil=true status=0
level=WARN msg="store: realm query failed" endpoint=category/defi/2 error="store: decode category/defi/2: unexpected end of JSON input"
--- FAIL: TestB3PagedMismatchIsNotLogged
```

[`cached`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/cache.go#L64-L65) stores the error without logging it, so the mismatch recurs silently in every cache window.
</details>

## gno.land/pkg/gnoweb/feature/store/frontend/store.css:708 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L708) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L708) · Warning
`overflow-x: auto` on an unpadded box makes the Spotlight row clip the card's [focus ring](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L275-L279) on its top, bottom and left below the `--xl` breakpoint. The category strip has padding only at its [block end](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L160-L163), and it cuts the pills' ring on top and left at every width.

<details><summary>steps</summary>

Open the store home in Chromium at a width below the `--xl` breakpoint and press Tab until a Spotlight card takes focus: only the ring's right edge shows. Tab to the `All` category pill: its ring has no top or left edge.

At that width the Spotlight container computes `overflow-x: auto; overflow-y: auto`, and its clip box sits inside the ring on the top, bottom and left. The category strip computes `overflow-x: scroll; overflow-y: scroll` with no top padding. The card link's own outline is [turned off](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L441-L443), so the card ring is the only focus cue.
</details>

## gno.land/pkg/gnoweb/feature/store/templates/parts.html:150 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/templates/parts.html#L150) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/templates/parts.html#L150) · Warning
The Claim snippet asks for `-gas-wanted 20000000`, below the 25.6M gas `Claim` uses on a fresh chain. It also carries no `-chainid`, so gnokey signs for its [default chain ID `dev`](https://github.com/gnolang/gno/blob/8ee3be106/tm2/pkg/crypto/keys/client/maketx.go#L123-L125) and any other chain refuses it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_store_claim_snippet.txtar <<'EOF'
loadpkg gno.land/r/gnoland/store/v0

gnoland start

# The snippet's flags, with only the chain's ID added: Claim runs out of gas.
! gnokey maketx call -pkgpath gno.land/r/gnoland/store/v0 -func Claim -args gno.land/p/$test1_user_addr/mylib -args mylib -args 'My lib' -args 'What it does, in one line' -args dev-tools -args 0 -args package -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
stdout 'GAS WANTED: 20000000'
stderr 'out of gas'

# Enough gas, and no -chainid as the snippet shows: signed for "dev", refused.
! gnokey maketx call -pkgpath gno.land/r/gnoland/store/v0 -func Claim -args gno.land/p/$test1_user_addr/mylib -args mylib -args 'My lib' -args 'What it does, in one line' -args dev-tools -args 0 -args package -gas-fee 1000000ugnot -gas-wanted 40000000 -broadcast test1
stderr 'signature verification failed'

# Both fixed: the claim lands.
gnokey maketx call -pkgpath gno.land/r/gnoland/store/v0 -func Claim -args gno.land/p/$test1_user_addr/mylib -args mylib -args 'My lib' -args 'What it does, in one line' -args dev-tools -args 0 -args package -gas-fee 1000000ugnot -gas-wanted 40000000 -broadcast -chainid=tendermint_test test1
stdout 'OK!'
EOF
go test ./gno.land/pkg/integration -run 'TestTestdata/zz_store_claim_snippet' -v
rm gno.land/pkg/integration/testdata/zz_store_claim_snippet.txtar
```

The txtar passes, which pins all three steps: the snippet's gas runs out, a missing `-chainid` fails the signature, and both fixed land:

```
GAS WANTED: 20000000
GAS USED:   25606611
# …
Data: out of gas error
# …
signature verification failed; verify correct account, sequence, and chain-id
# …
OK!
GAS WANTED: 40000000
GAS USED:   25606591
--- PASS: TestTestdata
```

[`newTemplates`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/template.go#L19-L25) passes only the package path, domain and store URL into the store templates, so this snippet cannot print the configured chain ID and remote the way `ui/command.html` does.
</details>

## gno.land/pkg/gnoweb/feature/store/view.go:263 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L263) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/view.go#L263) · Warning
`eligible` cuts the pool at the first 12 eligible apps in shelf order, and the realm sends [New first](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/front.gno#L55), so 12 trusted apps on New keep every earned app on Top uncounted. [`capped`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L312) then stays false, and the hero and every Spotlight place go to operator apps.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > gno.land/pkg/gnoweb/feature/store/zz_featured_cap_test.go <<'EOF'
package store

import (
	"fmt"
	"testing"
)

// 4 apps earned their place on Top, behind 12 trusted apps on New.
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
			t.Errorf("day %d: %d operator apps among hero and Spotlight, want at most 1", day, trusted)
		}
	}
}
EOF
go test ./gno.land/pkg/gnoweb/feature/store/ -run TestB7FeaturedCapBehindTrustedNew -v
rm gno.land/pkg/gnoweb/feature/store/zz_featured_cap_test.go
```

The test fails because every day shows four operator apps although four apps have earned their place:

```
day 0: hero+spot=[t01 t02 t03 t04] trusted=4
# …
day 3: hero+spot=[t04 t10 t11 t12] trusted=4
--- FAIL: TestB7FeaturedCapBehindTrustedNew
```

Apps under the default trusted paths are not seeds, so they land on New: 12 of them there is a plausible launch state. The [doc comment](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L248-L250) calls shelf order quality first, while New is ordered by age.
</details>

## gno.land/pkg/gnoweb/feature/store/view.go:338 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L338) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/view.go#L338) · Warning
`slices.Contains(pool, l)` checks the 12-app pool, filled from New first, rather than eligibility. 12 eligible apps on New keep the week's leading earned app out of it, and the hero falls back to the [daily rotation](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L300) labelled 'In the spotlight today'.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6273 -R gnolang/gno
cat > gno.land/pkg/gnoweb/feature/store/zz_trending_hero_test.go <<'EOF'
package store

import (
	"fmt"
	"testing"
)

// An earned app leads the trending shelf with 50 stars this week,
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
EOF
go test ./gno.land/pkg/gnoweb/feature/store/ -run TestB7TrendingHeroBehindTrustedNew -v
rm gno.land/pkg/gnoweb/feature/store/zz_trending_hero_test.go
```

The test fails because the hero is the rotation's pick, not the app with 50 stars this week:

```
hero=t01 label="In the spotlight today"
--- FAIL: TestB7TrendingHeroBehindTrustedNew
```

An earned app is at least `earnAge` old, so on an active chain it has usually left New's window and sits outside the pool.
</details>

## SKIP examples/gno.land/r/gnoland/store/v0/admin.gno:54 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/admin.gno#L54) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/admin.gno#L54) · Missing test
Missing test: no test runs the Pause, Unpause or Unhide executor through GovDAO. The pause test sets `paused` directly, and [`store_govdao.txtar`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/integration/testdata/store_govdao.txtar#L1) executes only a hide and a pick.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP examples/gno.land/r/gnoland/store/v0/admin.gno:181 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/admin.gno#L181) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/admin.gno#L181) · Missing test
Missing test: `dropPick` ends a pick by pointer identity, and no test hides or renames the picked app in a transaction after the one that set the pick.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the txtar case that would settle it.

## SKIP examples/gno.land/r/gnoland/store/v0/filetests/register_rejected_filetest.gno:6 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/filetests/register_rejected_filetest.gno#L6) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/filetests/register_rejected_filetest.gno#L6) · Missing test
Missing test: no test lists a realm calling `store.Register` from its `init`, and this refused call is the only external `Register` call in the tests.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the txtar case that would settle it.

## SKIP examples/gno.land/r/gnoland/store/v0/front.gno:81 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/front.gno#L81) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/front.gno#L81) · Missing test
Missing test: no test fills the trending shelf past `gridPool` or compares the length of `stars_7d` with the length of `slugs`, so dropping `s.Counts[:gridPool]` stays green.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP examples/gno.land/r/gnoland/store/v0/listing.gno:334 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L334) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/listing.gno#L334) · Missing test
Missing test: no test hides or re-kinds a namespace's oldest listing, so the `builder.First` re-derivation through `firstCreated` is unpinned.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP examples/gno.land/r/gnoland/store/v0/store.gno:117 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store.gno#L117) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/store.gno#L117) · Missing test
Missing test: no test reaches the registered-name branch of `ownsNamespace`, since every Claim and own-star test uses a personal-address namespace.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP examples/gno.land/r/gnoland/store/v0/store_test.gno:508 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store_test.gno#L508) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/store_test.gno#L508) · Missing test
Missing test: no assertion of `stars7d` follows the `Unstar` of a ranked star older than the trend window. No test fails once the [`pulse.gno` guard](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/pulse.gno#L98-L100) for epochs older than the window is removed.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP gno.land/cmd/gnoweb/main.go:331 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/cmd/gnoweb/main.go#L331) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/cmd/gnoweb/main.go#L331) · Missing test
Missing test: no test runs `setupWeb` with a valid `-store-realm`, so the line that turns the store on in the binary is unpinned.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP gno.land/pkg/gnoweb/feature/store/cache.go:91 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/cache.go#L91) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/cache.go#L91) · Missing test
Missing test: no test pins `errorTTL`, the shorter lifetime of a failed load. The suite stays green when errors are cached for the full `cacheTTL`.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP gno.land/pkg/gnoweb/feature/store/store_test.go:263 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/store_test.go#L263) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/store_test.go#L263) · Missing test
Missing test: `TestCacheCoalesces` renders three times in sequence, so it pins the cache hit and never the coalescing of concurrent loads its name claims.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP gno.land/pkg/gnoweb/feature/store/templates/parts.html:147 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/templates/parts.html#L147) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/templates/parts.html#L147) · Missing test
Missing test: no test runs either published snippet against the realm, and `TestSnippetsUseTheConfiguredRealm` asserts only the import string.

Not posted: the `parts.html:150` Warning carries the txtar that runs the Claim snippet, and this section adds no second case.

## SKIP gno.land/pkg/gnoweb/handler_http.go:528 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/handler_http.go#L528) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/handler_http.go#L528) · Missing test
Missing test: no `pkg/gnoweb` test reaches the store branch of `GetPackageView` with a store response that decodes.

Not posted: PLAUSIBLE on the finder's read, and no verifier ran the mutation that would settle it.

## SKIP gno.land/pkg/integration/testdata/store_govdao.txtar:1 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/integration/testdata/store_govdao.txtar#L1) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/integration/testdata/store_govdao.txtar#L1) · Missing test
Missing test: only `ProposeHide` and `ProposeSpotlight` are executed through GovDAO, while the [store test comment](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store_test.gno#L875-L876) says GovDAO execution is covered by this txtar.

Not posted: the same gap as the `admin.gno:54` section, PLAUSIBLE on the finder's read with no verifier run.

## examples/gno.land/r/gnoland/store/v0/admin.gno:53 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/admin.gno#L53) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/admin.gno#L53) · Nit
Nit: `ProposePause` promises voters a stop to listing updates, while [`SubmitRich`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store.gno#L135) has no `paused` check and keeps changing a listing's icon and cover during a pause.

## gno.land/pkg/gnoweb/feature/store/api.go:207 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/api.go#L207) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/api.go#L207) · Nit
Nit: `paged` refuses only a page count below the page asked for, and a fresh page 1 can report more pages than the cached home count right after an app joins. Its [Next link](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/handler.go#L161) then leads to a Not found page from [`pageParam`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/handler.go#L123) until the home entry expires.

## gno.land/pkg/gnoweb/feature/store/cover.go:101 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/cover.go#L101) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/cover.go#L101) · Nit
Nit: `utf8.DecodeRuneInString(p)` takes each word's first rune rather than its first letter or digit, and `(Beta) Swap` gets the icon `(S` rather than `BS`.

<details><summary>cases</summary>

Added to [`TestInitials`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/store_test.go#L355), these two cases fail with `(S` and `#W`:

```go
		"(Beta) Swap": "BS",
		"#1 Wallet":   "1W",
```
</details>

## gno.land/pkg/gnoweb/feature/store/frontend/store.css:145-147 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L145-L147) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L145) · Nit
Nit: On a phone this rule hides `.clock` and `.moment`. The [pulse](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/templates/pages.html#L11) then shows only its [live dot](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L112) on a chain with no recent listing or star, with no text beside it.

## gno.land/pkg/gnoweb/feature/store/frontend/store.css:592-594 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L592-L594) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L592) · Nit
Nit: Only `.b-store-meta a` is lifted above the card's [stretched link](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L434-L437). So hover never shows the `title=` explanation on a card's [trust badge](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/frontend/store.css#L605), 'community' label or shortened path.

## gno.land/pkg/gnoweb/feature/store/templates/parts.html:144 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/templates/parts.html#L144) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/templates/parts.html#L144) · Nit
Nit: The promise holds only after 3 ranked stars. [`Claim`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store.gno#L87-L89) leaves a package unconfirmed, and [`index`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L296) keeps it off every [Build shelf](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/shelves.gno#L88-L93) and builder total until then.

## SKIP gno.land/pkg/gnoweb/feature/store/view.go:281 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L281) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/view.go#L281) · Nit
Nit: the `featured` comment calls the hero trending only when it leads "outright", while [`leads`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L355) compares against `slices.Max` and returns true on a tie, as its own comment says.

Not posted: a finding about a code comment's own wording.

## gno.land/pkg/gnoweb/feature/store/view.go:454 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L454) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/view.go#L454) · Nit
Nit: Activity [events](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/pulse.gno#L20-L24) carry no height. This line then keeps announcing that a seed app 'just joined' on an idle store, long after the deploy and until the next listing, update or milestone.

## examples/gno.land/r/gnoland/store/v0/admin.gno:171 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/admin.gno#L171) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/admin.gno#L171) · Suggestion
Suggestion: `picked` binds a pick to the folded title alone, and the app can rewrite its tagline and swap its cover after the vote while still under 'Picked by GovDAO'. I think `setPick` should record a hash of the tagline and effective cover, for `picked` to compare.

<details><summary>repro</summary>

A test that picks an app, re-registers it with the tagline `Official GovDAO airdrop, claim yours now`, then calls `SubmitRich` and skips `richDelay`, passes asserting the swapped tagline under `## Picked by GovDAO` and `picked(now)` still returning the app with the new cover. [`update`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L229) calls `dropPick` only when the folded title changes, which matches ADR-004's end triggers: hide, no longer a shown app, rename.
</details>

## examples/gno.land/r/gnoland/store/v0/api.gno:188 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/api.gno#L188) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/api.gno#L188) · Suggestion
Suggestion: `shownApp()` lets an app in a free g1 address namespace become the front page's 'just joined' moment, the slot the [`newApps` comment](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/front.gno#L39-L42) says those namespaces never earn. I think `writeActivity` should skip events from address namespaces, to match that comment.

<details><summary>repro</summary>

Registering `gno.land/r/<g1 address>/app` leaves it out of New and makes `{"kind":"listed","slug":"anon-app"}` the first activity event in `api/v1/home`, which gnoweb's [`latestMoment`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L454) renders as `Anon App just joined`. ADR-004 defines the feed as the most recent events of confirmed, visible listings with no namespace rule, so the code matches its spec and only the comment disagrees.
</details>

## examples/gno.land/r/gnoland/store/v0/listing.gno:212 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L212) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/listing.gno#L212) · Suggestion
Suggestion: `visibleInNamespace(l.Namespace) >= maxPerNamespace` counts the three seeded listings under [`gnoland`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/seed.gno#L13-L15) and the three under [`nt`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/seed.gno#L21-L23) against the [cap of 5](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/store.gno#L44), leaving room for two more core realms in each. I think [`isCoreNamespace`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/front.gno#L98) should exempt these namespaces from the cap, as it does from the [builder rankings](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L362-L364).

## examples/gno.land/r/gnoland/store/v0/listing.gno:488 [gh](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L488) · [↗](../../../../../.worktrees/gno-review-6273/examples/gno.land/r/gnoland/store/v0/listing.gno#L488) · Suggestion
Suggestion: `l.restar` drops a ranked star without rechecking [`confirmStars`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L466-L468), and [`confirm`](https://github.com/gnolang/gno/blob/8ee3be106/examples/gno.land/r/gnoland/store/v0/listing.gno#L259-L263) never resets `Confirmed`, so a claimed listing stays on the shelves after its three backers unstar. I think `removeStar` should clear `Confirmed` once a claimed, not self-listed listing drops below `confirmStars`.

## gno.land/pkg/gnoweb/feature/store/cache.go:57 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/cache.go#L57) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/cache.go#L57) · Suggestion
Suggestion: `c.group.DoChan` makes every caller a waiter, and singleflight [re-raises a panic in `load` with `go panic(e)`](https://github.com/golang/sync/blob/v0.21.0/singleflight/singleflight.go#L166-L167) on a goroutine no recover reaches, killing gnoweb. No input is known to reach such a panic, and I think the closure should recover it and return it as an error.

## gno.land/pkg/gnoweb/feature/store/view.go:493 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L493) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/feature/store/view.go#L493) · Suggestion
Suggestion: `TruncMiddle(ns, 6, 4)` keeps few enough characters for a vanity key search to find an address with another app's short path, against the [comment](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/view.go#L487-L488) saying truncation never hides a look-alike. I think showing more characters of each end puts the search out of reach.

<details><summary>count</summary>

The short path keeps `g1`, four data characters and the last four checksum characters: 8 characters of 5 bits each.

```bash
python3 -c 'print(2**(8*5))'
```

```
1099511627776
```
</details>

## gno.land/pkg/gnoweb/handler_http.go:126 [gh](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/handler_http.go#L126) · [↗](../../../../../.worktrees/gno-review-6273/gno.land/pkg/gnoweb/handler_http.go#L126) · Suggestion
Suggestion: `validate` checks neither `Meta.Domain` nor the leading slash of `StoreRealm`, so [`store.New`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/pkg/gnoweb/feature/store/feature.go#L57) panics without a domain and a slashless path never matches. I think `validate` should make both checks, a job only [`setupWeb`](https://github.com/gnolang/gno/blob/8ee3be106/gno.land/cmd/gnoweb/main.go#L326-L329) does today, and return the error from `NewHTTPHandler`.
