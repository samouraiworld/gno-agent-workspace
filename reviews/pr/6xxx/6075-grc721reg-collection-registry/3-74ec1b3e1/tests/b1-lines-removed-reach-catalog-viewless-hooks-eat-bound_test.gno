// Repro: view-less hooks count against the registry's kind bounds before the
// view filter runs, so the register event and the index badges under-report
// the kinds a collection publishes, and the "too many extensions" abort is
// skipped, while GetExtensionKinds and GetExtension still see every view.
//
// From a plain clone of gnolang/gno at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7:
//
//   cp <this file> examples/gno.land/r/demo/defi/grc721reg/zz_viewless_bound_test.gno
//   cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/demo/defi/grc721reg -run 'TestViewlessHooks' -v
//
// Uses stubView from grc721reg_test.gno in the same package. Each case asserts
// the behaviour the registry claims (abort on an oversized published set, a
// badge for every published kind up to maxBadges, an ellipsis past it) and
// fails on head with the observed figure.
package grc721reg

import (
	"strconv"
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/uassert/v0"
)

func stackViews(led *grc721.PrivateLedger, id, prefix string, n int) {
	for i := 0; i < n; i++ {
		v := &stubView{prefix + strconv.Itoa(100+i), id}
		led.RegisterExtension(v, v)
	}
}

func stackHooks(led *grc721.PrivateLedger, id string, n int) {
	for i := 0; i < n; i++ {
		led.RegisterExtension(&stubView{"hook" + strconv.Itoa(100+i), id}, nil)
	}
}

// maxEventKinds+1 published kinds abort on their own (TestExtensionCountIsCapped);
// one view-less hook registered first lets the same set register, and the event
// carries maxEventKinds of them.
func TestViewlessHooksSkipEventCap(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_hook_event"))
	token, ledger := grc721.NewToken("HookEvent", "HEV", 0, cur)
	stackHooks(ledger, token.ID(), 1)
	stackViews(ledger, token.ID(), "kind", maxEventKinds+1)

	aborted := uassert.AbortsContains(t, cur, "too many extensions", func() {
		Register(cross(cur), token, "")
	})

	published := len(GetExtensionKinds("gno.land/r/demo/reg_hook_event.HEV"))
	inEvent := len(publishedKinds(token, maxEventKinds+1))
	println("published views:", published, "kinds the register event carries:", inEvent, "aborted:", aborted)
}

// maxBadges+1 view-less hooks registered first hide a published view from the
// index: the badge reads "_core only_" while GetExtension resolves the view.
func TestViewlessHooksHideBadges(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_hook_badge"))
	token, ledger := grc721.NewToken("HookBadge", "HBG", 0, cur)
	stackHooks(ledger, token.ID(), maxBadges+1)
	stackViews(ledger, token.ID(), "pub", 1)
	key := Register(cross(cur), token, "")

	_, ok := GetExtension(key, "pub100")
	out := Render(key)
	println("GetExtension(pub100) ok:", ok, "| render extensions line core-only:", strings.Contains(out, "_core only_"))
	if ok && strings.Contains(out, "_core only_") {
		t.Errorf("published view pub100 resolves but the page renders _core only_")
	}
}

// One view-less hook plus maxBadges+1 published views: the page shows maxBadges
// badges and no ellipsis, as if the set were complete.
func TestViewlessHooksDropEllipsis(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reg_hook_ellip"))
	token, ledger := grc721.NewToken("HookEllip", "HEL", 0, cur)
	stackHooks(ledger, token.ID(), 1)
	stackViews(ledger, token.ID(), "pub", maxBadges+1)
	key := Register(cross(cur), token, "")

	out := Render(key)
	println("published:", len(GetExtensionKinds(key)), "badges:", strings.Count(out, "`pub"), "ellipsis:", strings.Contains(out, "…"))
	if !strings.Contains(out, "…") {
		t.Errorf("%d published kinds render %d badges with no ellipsis", len(GetExtensionKinds(key)), strings.Count(out, "`pub"))
	}
}
