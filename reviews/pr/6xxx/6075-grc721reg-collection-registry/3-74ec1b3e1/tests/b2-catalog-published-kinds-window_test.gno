// publishedKinds windows ExtensionKindsUpTo over every kind, then drops the
// view-less ones, so one view-less hook registered first lets a 17-kind
// collection past the maxEventKinds cap and hides the badge truncation mark.
//
// Repro, from a plain clone of gnolang/gno at 74ec1b3e1 with gno built from it on PATH,
// T being reviews/pr/6xxx/6075-grc721reg-collection-registry/3-74ec1b3e1/tests in gno-agent-workspace
//   cp $T/b2-catalog-published-kinds-window_test.gno examples/gno.land/r/demo/defi/grc721reg/z_b2_window_test.gno
//   cd examples && GNOROOT=$PWD/.. gno test ./gno.land/r/demo/defi/grc721reg/ -run TestB2PublishedKindsWindow -v
// PASS means the defect holds.

package grc721reg

import (
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/ufmt/v0"
	"gno.land/p/nt/urequire/v0"
)

func addViews(led *grc721.PrivateLedger, tok *grc721.Token, n int) {
	for i := 0; i < n; i++ {
		v := &stubView{ufmt.Sprintf("k%02d", i), tok.ID()}
		led.RegisterExtension(v, v)
	}
}

func TestB2PublishedKindsWindow(cur realm, t *testing.T) {
	// Control: 17 published kinds abort Register.
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_window_ctl"))
	ctl, ctlLed := grc721.NewToken("Ctl", "CTL", 0, cur)
	addViews(ctlLed, ctl, maxEventKinds+1)
	urequire.AbortsContains(t, cur, "too many extensions", func() { Register(cross(cur), ctl, "") })

	// Same 17 view kinds, one view-less hook registered first: Register passes.
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_window"))
	tok, led := grc721.NewToken("Win", "WIN", 0, cur)
	led.RegisterExtension(&stubView{"a-silent", tok.ID()}, nil)
	addViews(led, tok, maxEventKinds+1)
	key := Register(cross(cur), tok, "")
	println("cap bypass: registered", key, "with", len(GetExtensionKinds(key)), "published kinds; window saw", len(publishedKinds(tok, maxEventKinds+1)))

	// Badges: 9 views shows the truncation mark; one view-less hook first hides it.
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_badge_ctl"))
	bc, bcLed := grc721.NewToken("BCtl", "BCTL", 0, cur)
	addViews(bcLed, bc, maxBadges+1)
	bcKey := Register(cross(cur), bc, "")
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_badge"))
	bt, btLed := grc721.NewToken("Badge", "BADGE", 0, cur)
	btLed.RegisterExtension(&stubView{"a-silent", bt.ID()}, nil)
	addViews(btLed, bt, maxBadges+1)
	btKey := Register(cross(cur), bt, "")
	println("badge ctl: published", len(GetExtensionKinds(bcKey)), "truncation mark", strings.Contains(extensionBadges(bc), "…"))
	println("badge one-silent: published", len(GetExtensionKinds(btKey)), "truncation mark", strings.Contains(extensionBadges(bt), "…"))
	println("badge one-silent row:", extensionBadges(bt))

	urequire.Equal(t, maxEventKinds+1, len(GetExtensionKinds(key)), "17 published kinds registered past the cap")
	urequire.True(t, strings.Contains(extensionBadges(bc), "…"), "control shows the mark")
	urequire.False(t, strings.Contains(extensionBadges(bt), "…"), "one view-less hook hides it")
}
