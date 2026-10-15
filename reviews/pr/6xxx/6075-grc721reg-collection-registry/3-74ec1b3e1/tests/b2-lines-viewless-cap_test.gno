// gnolang/gno#6075 round 3, finder b2-lines: a view-less hook inside the
// publishedKinds walk window bypasses the register event's kind cap and the
// page's elision mark, so both under-report the published set with no signal.
//
// Repro, from a plain clone of github.com/gnolang/gno:
//
//	git checkout 74ec1b3e1ad9406d645443fffbca8ae277b8edf7
//	go install ./gnovm/cmd/gno
//	cp b2-lines-viewless-cap_test.gno examples/gno.land/r/demo/defi/grc721reg/b2lines_cap_test.gno
//	cd examples && gno test -v -run TestB2Viewless ./gno.land/r/demo/defi/grc721reg/
//
// Observed at 74ec1b3e1 (both PASS, asserting the defect):
//
//	event kinds=16 published kinds=17 maxEventKinds=16
//	badges=8 elided=false published=10
//
// Uses stubView from grc721reg_test.gno in the same package.
package grc721reg

import (
	"strconv"
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/urequire/v0"
)

// hookOnly is an Extension with no view, registered with a nil view.
type hookOnly struct{ kind string }

func (h *hookOnly) ExtensionKind() string                       { return h.kind }
func (h *hookOnly) OnMint(address, grc721.TokenID)              {}
func (h *hookOnly) OnTransfer(address, address, grc721.TokenID) {}
func (h *hookOnly) OnBurn(grc721.TokenID)                       {}

func stackViews(led *grc721.PrivateLedger, id string, n int) {
	for i := 0; i < n; i++ {
		v := &stubView{"kind" + strconv.Itoa(i), id}
		led.RegisterExtension(v, v)
	}
}

// One view-less hook ahead of maxEventKinds+1 viewed kinds: the cap that
// aborts an all-viewed set of the same size lets this one through, and the
// event set is silently one short of what GetExtensionKinds reports.
func TestB2ViewlessHookBypassesEventCap(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/b2_cap_event"))
	token, ledger := grc721.NewToken("Over", "OVER", 0, cur)
	ledger.RegisterExtension(&hookOnly{"silent"}, nil)
	stackViews(ledger, token.ID(), maxEventKinds+1)

	key := Register(cross(cur), token, "")
	inEvent := publishedKinds(token, maxEventKinds+1)
	all := GetExtensionKinds(key)
	t.Logf("event kinds=%d published kinds=%d maxEventKinds=%d", len(inEvent), len(all), maxEventKinds)
	urequire.Equal(t, maxEventKinds+1, len(all), "17 viewed kinds are published")
	urequire.Equal(t, maxEventKinds, len(inEvent), "the event carries 16 and no error")
}

// One view-less hook ahead of maxBadges+2 viewed kinds: the page renders
// maxBadges badges and drops the elision mark although more are published.
func TestB2ViewlessHookHidesElision(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/b2_cap_page"))
	token, ledger := grc721.NewToken("Badges", "BADGE", 0, cur)
	ledger.RegisterExtension(&hookOnly{"silent"}, nil)
	stackViews(ledger, token.ID(), maxBadges+2)

	key := Register(cross(cur), token, "")
	out := Render(key)
	t.Logf("badges=%d elided=%v published=%d", strings.Count(out, "`kind"), strings.Contains(out, "…"), len(GetExtensionKinds(key)))
	urequire.Equal(t, maxBadges+2, len(GetExtensionKinds(key)), "10 viewed kinds are published")
	urequire.False(t, strings.Contains(out, "…"), "but the page shows no elision mark")
}
