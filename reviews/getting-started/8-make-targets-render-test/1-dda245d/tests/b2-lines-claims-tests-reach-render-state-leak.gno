// Repro for: realm state persists across test functions, so TestRender runs
// against the message TestSetAndGet wrote ("Hello, Test!") and nothing in the
// suite ever asserts what a freshly deployed realm renders ("Hello, Gno!").
//
// From a plain clone (gno built from gnolang/gno master, on PATH):
//
//	git clone https://github.com/gnolang/getting-started
//	cd getting-started && git checkout dda245ddc4556d0b83e471b67830d17c0df7b666
//	cp <this file> ./zz_state_leak_test.gno
//	gno test -v .
//
// Expected: PASS, with "message after the suite = \"Hello, Test!\"" and
// "Render() does NOT contain the initial message" in the log — the state
// TestRender saw.
//
// Run it alone to see the other half:
//
//	gno test -v . -run TestRenderSeesLeakedState   # => initial message present
//
// Measured on gno built from gnolang/gno master bbd9b2ffe (the PR body reports
// master.184+393b6f92a, a different sha).

package hello

import (
	"strings"
	"testing"
)

func TestRenderSeesLeakedState(t *testing.T) {
	t.Logf("message after the suite = %q", Get())

	if strings.Contains(Render(""), "Hello, Gno!") {
		t.Logf("Render() contains the initial message")
	} else {
		t.Logf("Render() does NOT contain the initial message; TestRender asserted against %q", Get())
	}
}
