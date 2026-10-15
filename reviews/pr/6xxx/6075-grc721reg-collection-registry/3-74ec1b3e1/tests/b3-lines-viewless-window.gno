// b3-lines-viewless-window: publishedKinds cuts the ExtensionKindsUpTo window
// before it drops view-less hooks, then sorts what is left. Register's
// maxEventKinds guard, the badge row and the badge order all answer from that
// window, not from the kinds the collection publishes.
//
// Repro from a plain clone of gnolang/gno at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7:
//
//   cp b3-lines-viewless-window.gno examples/gno.land/r/demo/defi/grc721reg/zz_lines_window_test.gno
//   cd examples && gno test ./gno.land/r/demo/defi/grc721reg -run 'TestLines' -v
//   rm gno.land/r/demo/defi/grc721reg/zz_lines_window_test.gno
//
// Each test asserts the correct behaviour; a FAIL is the defect. stubView is
// the package's own test double, from grc721reg_test.gno.
package grc721reg

import (
	"strconv"
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/urequire/v0"
)

func linesStack(led *grc721.PrivateLedger, id, prefix string, n int, withView bool) {
	for i := 0; i < n; i++ {
		v := &stubView{prefix + strconv.Itoa(i), id}
		if withView {
			led.RegisterExtension(v, v)
		} else {
			led.RegisterExtension(v, nil)
		}
	}
}

func linesBadges(out string) string {
	i := strings.Index(out, "- extensions: ")
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(out[i:])
}

// maxEventKinds view-less hooks ahead of maxEventKinds+1 published kinds: the
// collection publishes more kinds than the event may carry, so Register must
// abort exactly as it does when the same kinds are registered alone.
func TestLinesEventCapCountsHooks(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/lines_cap"))
	token, ledger := grc721.NewToken("Cap", "CAP", 0, cur)
	linesStack(ledger, token.ID(), "hook", maxEventKinds+1, false)
	linesStack(ledger, token.ID(), "kind", maxEventKinds+1, true)

	aborted := true
	func() {
		defer func() { recover() }()
		Register(cross(cur), token, "")
		aborted = false
	}()
	t.Logf("Register aborted: %v; published kinds now: %d (cap %d)",
		aborted, len(GetExtensionKinds("gno.land/r/demo/lines_cap.CAP")), maxEventKinds)
	urequire.True(t, aborted, "17 published kinds must trip the maxEventKinds guard")
}

// maxBadges+1 view-less hooks ahead of one published kind: the registry must
// not report the collection as core only while GetExtensionKinds lists a kind.
func TestLinesCoreOnlyWithAKind(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/lines_core"))
	token, ledger := grc721.NewToken("Core", "LCORE", 0, cur)
	linesStack(ledger, token.ID(), "hook", maxBadges+1, false)
	linesStack(ledger, token.ID(), "kind", 1, true)
	key := Register(cross(cur), token, "")

	t.Logf("GetExtensionKinds: %v; page: %q", GetExtensionKinds(key), linesBadges(Render(key)))
	urequire.False(t, strings.Contains(Render(key), "_core only_"),
		"a collection with a published kind must not render as core only")
}

// Two collections publishing the same ten kinds, registered in two orders:
// publishedKinds says its sort makes the output independent of registration
// order, so the two badge rows must match.
func TestLinesBadgesIgnoreRegistrationOrder(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/lines_order_a"))
	first, firstLed := grc721.NewToken("Order", "ORDA", 0, cur)
	linesStack(firstLed, first.ID(), "a", 1, true)
	linesStack(firstLed, first.ID(), "b", maxBadges+1, true)
	firstKey := Register(cross(cur), first, "")

	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/lines_order_b"))
	second, secondLed := grc721.NewToken("Order", "ORDB", 0, cur)
	linesStack(secondLed, second.ID(), "b", maxBadges+1, true)
	linesStack(secondLed, second.ID(), "a", 1, true)
	secondKey := Register(cross(cur), second, "")

	a, b := linesBadges(Render(firstKey)), linesBadges(Render(secondKey))
	t.Logf("a registered first: %s", a)
	t.Logf("a registered last:  %s", b)
	urequire.Equal(t, a, b, "same kinds, same badges")
}
