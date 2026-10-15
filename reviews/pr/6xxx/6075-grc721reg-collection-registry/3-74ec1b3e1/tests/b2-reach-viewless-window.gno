// b2-reach-viewless-window: ExtensionKindsUpTo(limit) counts view-less hooks,
// and publishedKinds drops them only after the window is cut, so a view-less
// hook registered first hides published kinds from the badges and the
// register event, and lets Register past its maxEventKinds guard.
//
// Repro from a plain clone of gnolang/gno at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7:
//
//   cp b2-reach-viewless-window.gno examples/gno.land/r/demo/defi/grc721reg/zz_reach_window_test.gno
//   cd examples && gno test ./gno.land/r/demo/defi/grc721reg -run 'TestReachViewless' -v
//
// Each test asserts the correct behaviour; a FAIL is the defect.
package grc721reg

import (
	"strconv"
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/urequire/v0"
)

func reachStack(led *grc721.PrivateLedger, id, prefix string, n int, withView bool) {
	for i := 0; i < n; i++ {
		v := &stubView{prefix + strconv.Itoa(i), id}
		if withView {
			led.RegisterExtension(v, v)
		} else {
			led.RegisterExtension(v, nil)
		}
	}
}

// One view-less hook ahead of maxBadges+1 published kinds: the page must mark
// the ninth published kind elided.
func TestReachViewlessHidesEllipsis(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reach_ellipsis"))
	token, ledger := grc721.NewToken("Ell", "ELL", 0, cur)
	reachStack(ledger, token.ID(), "hook", 1, false)
	reachStack(ledger, token.ID(), "kind", maxBadges+1, true)
	key := Register(cross(cur), token, "")

	t.Logf("published kinds: %d", len(GetExtensionKinds(key)))
	out := Render(key)
	t.Logf("badges rendered: %d, ellipsis: %v", strings.Count(out, "`kind"), strings.Contains(out, "…"))
	urequire.True(t, strings.Contains(out, "…"), "9 published kinds, 8 badges: the rest must be marked elided")
}

// maxBadges+1 view-less hooks ahead of one published kind: the page must not
// say the collection is core only.
func TestReachViewlessShowsCoreOnly(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reach_coreonly"))
	token, ledger := grc721.NewToken("Core", "CORE", 0, cur)
	reachStack(ledger, token.ID(), "hook", maxBadges+1, false)
	reachStack(ledger, token.ID(), "kind", 1, true)
	key := Register(cross(cur), token, "")

	t.Logf("published kinds: %d", len(GetExtensionKinds(key)))
	out := Render(key)
	urequire.False(t, strings.Contains(out, "_core only_"), "a collection with a published kind must not render as core only")
}

// One view-less hook ahead of maxEventKinds+1 published kinds: Register must
// abort with "too many extensions", as it does without the hook.
func TestReachViewlessPassesEventCap(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/reach_eventcap"))
	token, ledger := grc721.NewToken("Cap", "CAP", 0, cur)
	reachStack(ledger, token.ID(), "hook", 1, false)
	reachStack(ledger, token.ID(), "kind", maxEventKinds+1, true)

	urequire.AbortsContains(t, cur, "too many extensions", func() {
		key := Register(cross(cur), token, "")
		t.Logf("registered %s with %d published kinds", key, len(GetExtensionKinds(key)))
	})
}
