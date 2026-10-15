// Finder b3-catalog, gnolang/gno#6075 at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7.
// publishedKinds caps the raw walk before dropping view-less kinds, so a
// view-less hook inside the window hides a published kind: the badge list
// loses its elision mark, Register's "too many extensions" cap lets 17 through
// with 16 in the event, and the badge subset follows registration order.
//
// Repro, from a plain clone of github.com/gnolang/gno:
//   git fetch origin pull/6075/head && git checkout --detach 74ec1b3e1ad9406d645443fffbca8ae277b8edf7
//   cp <this file> examples/gno.land/r/demo/defi/grc721reg/zz_catalog_test.gno
//   cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/demo/defi/grc721reg -v -run TestCatalog
//
// Observed at 74ec1b3e1 (all three FAIL; each asserts the correct behaviour):
//   Render shows 8 of 9 published kinds with no elision mark
//   Register accepted 17 published kinds (cap 16); event lists 16
//   sorted-first kind0 missing from badges: ... `kind1` ... `kind8` …
package grc721reg

import (
	"strconv"
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
)

// stack registers one view-less hook first, then n published views.
func catStack(led *grc721.PrivateLedger, id string, n int) {
	led.RegisterExtension(&stubView{"hook", id}, nil)
	for i := 0; i < n; i++ {
		v := &stubView{"kind" + strconv.Itoa(i), id}
		led.RegisterExtension(v, v)
	}
}

// Badges: maxBadges+1 published kinds must render maxBadges badges and "…".
func TestCatalogBadgesHideKindsBehindHook(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/cat_badges"))
	token, ledger := grc721.NewToken("Cat", "CAT", 0, cur)
	catStack(ledger, token.ID(), maxBadges+1)
	key := Register(cross(cur), token, "")
	out := Render(key)
	published := len(GetExtensionKinds(key))
	shown := strings.Count(out, "`kind")
	elided := strings.Contains(out, "…")
	println("badges: published", published, "shown", shown, "elided", elided)
	if published > shown && !elided {
		t.Errorf("Render shows %d of %d published kinds with no elision mark", shown, published)
	}
}

// Event cap: maxEventKinds+1 published kinds must abort "too many extensions".
func TestCatalogEventCapBypassedByHook(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/cat_event"))
	token, ledger := grc721.NewToken("Over", "OVER", 0, cur)
	catStack(ledger, token.ID(), maxEventKinds+1)
	key := Register(cross(cur), token, "")
	published := len(GetExtensionKinds(key))
	inEvent := len(publishedKinds(token, maxEventKinds+1))
	println("event: registered", key, "published", published, "kinds in register event", inEvent)
	if published > maxEventKinds {
		t.Errorf("Register accepted %d published kinds (cap %d); event lists %d", published, maxEventKinds, inEvent)
	}
}

// Order: the badge set must not depend on registration order.
func TestCatalogBadgeSetDependsOnOrder(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/cat_order"))
	token, ledger := grc721.NewToken("Ord", "ORD", 0, cur)
	for i := maxBadges + 1; i >= 0; i-- { // kind9 .. kind0, kind0 registered last
		v := &stubView{"kind" + strconv.Itoa(i), token.ID()}
		ledger.RegisterExtension(v, v)
	}
	out := Render(Register(cross(cur), token, ""))
	println("order: kind0 shown", strings.Contains(out, "`kind0`"))
	if !strings.Contains(out, "`kind0`") {
		t.Errorf("sorted-first kind0 missing from badges: %s", out)
	}
}
