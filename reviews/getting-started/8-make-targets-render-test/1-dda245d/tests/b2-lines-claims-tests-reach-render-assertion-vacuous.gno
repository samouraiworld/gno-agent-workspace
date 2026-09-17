// Repro for: TestRender's assertion is satisfied by construction once the
// realm message is the empty string, so it passes against a Render that
// returns no realm state at all.
//
// From a plain clone (gno built from gnolang/gno master, on PATH):
//
//	git clone https://github.com/gnolang/getting-started
//	cd getting-started && git checkout dda245ddc4556d0b83e471b67830d17c0df7b666
//	cp <this file> ./zz_vacuous_test.gno
//	gno test -v .
//
// Expected: PASS, with the log line showing the shipped assertion holding
// against output that carries none of the realm's state.
//
// The direct form of the same repro, without this file:
//
//	sed -i 's/var message string = "Hello, Gno!"/var message string = ""/' hello.gno
//	perl -0pi -e 's/return `# ` \+ message \+ `/return `# oops/' hello.gno
//	gno test -v . -run TestRender    # => PASS, though Render dropped the message
//	git checkout -- hello.gno
//
// Measured on gno built from gnolang/gno master bbd9b2ffe (the PR body reports
// master.184+393b6f92a, a different sha).

package hello

import (
	"strings"
	"testing"
)

func TestRenderAssertionVacuousOnEmptyMessage(cur realm, t *testing.T) {
	Set(cross(cur), "")

	// Stand-in for a Render that lost the realm's state entirely.
	broken := "# oops\n\nno realm state in this output\n"

	// The assertion hello_test.gno:34 ships, verbatim.
	if !strings.Contains(broken, Get()) {
		t.Errorf("assertion caught the broken output; the vacuity claim is wrong")
	} else {
		t.Logf("shipped assertion passes against output carrying no realm state: Get()=%q", Get())
	}

	Set(cross(cur), "Hello, Gno!")
}
