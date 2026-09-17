// b1-claims-gno-semantics_test.gno — the four runnable claims AGENTS.md makes
// in its "Gno is not quite Go" section, each asserted against what the VM does.
//
// Repro from a plain clone (no checkout of this PR needed; the file is
// self-contained and the claims are about the toolchain, not the repo):
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout bbd9b2ffe && make -C gnovm install   # or any gno on PATH
//	mkdir /tmp/b1probe && cd /tmp/b1probe
//	printf 'module = "gno.land/r/example/probe"\ngno = "0.9"\n' > gnomod.toml
//	printf 'package probe\n\nfunc Render(path string) string { return "probe" }\n' > probe.gno
//	cp <this file> ./probe_test.gno
//	gno test -v .
//
// Measured on gno built from gnolang/gno master bbd9b2ffe (the PR body states
// it was verified on master.184+393b6f92a, a different master sha).
//
// Each test asserts the claim exactly as AGENTS.md words it. Today the file is
// RED on TestUfmtDashFlagResidue (the claim omits the "5s" the formatter copies
// out), while TestMapIterationOrderIsDeterministic passes — and its passing is
// what refutes AGENTS.md: order is insertion-stable, not unspecified.
package probe

import (
	"testing"

	"gno.land/p/nt/ufmt/v0"
)

// AGENTS.md: "Render must be deterministic. Map iteration order is
// unspecified, so never build output by ranging a map."
//
// GnoVM's MapValue is a doubly-linked MapList plus an index map
// (gnovm/pkg/gnolang/values.go:935-1016), so `range` walks insertion order and
// is stable across runs and across the amino round-trip. The rule may still be
// worth keeping, but the reason given for it is a Go fact, not a Gno one.
func TestMapIterationOrderIsDeterministic(t *testing.T) {
	keys := []string{"z", "a", "m", "b", "q", "c", "y", "d"}
	m := map[string]int{}
	for i, k := range keys {
		m[k] = i
	}
	first := ""
	for k := range m {
		first += k
	}
	second := ""
	for k := range m {
		second += k
	}
	if first != second {
		t.Errorf("iteration order varies within one run: %q then %q", first, second)
	}
	if first != "zambqcyd" {
		t.Errorf("expected insertion order %q, got %q", "zambqcyd", first)
	}
	t.Logf("MAPORDER=%q (insertion order, not unspecified)", first)
}

// AGENTS.md: "`%-5s` comes back as `(unhandled verb: %-)`".
// The verb switch defaults on '-' and writes that marker, then the loop
// resumes at '5' and copies the rest of the spec out verbatim, so the caller
// gets the marker plus a trailing "5s".
func TestUfmtDashFlagResidue(t *testing.T) {
	got := ufmt.Sprintf("%-5s", "ab")
	if got != "(unhandled verb: %-)" {
		t.Errorf("AGENTS.md says %q, got %q", "(unhandled verb: %-)", got)
	}
	t.Logf("DASH5S=%q", got)
	t.Logf("DASH5S_IN_CONTEXT=%q", ufmt.Sprintf("[%-5s]", "ab"))
}

// AGENTS.md: "`ufmt.Sprintf(\"%03d\", 7)` returns `\"7\"`". Holds: the width is
// parsed and then dropped, %d never receives it.
func TestUfmtZeroPadIgnored(t *testing.T) {
	if got := ufmt.Sprintf("%03d", 7); got != "7" {
		t.Errorf("expected %q, got %q", "7", got)
	}
}

// AGENTS.md: "`Example*` tests need an `// Output:` block or `gno test` skips
// them silently". Holds: under `gno test -v` only ExampleWithOutput appears in
// the RUN list and the sentinel below never prints.
func ExampleNoOutput() {
	println("SENTINEL_EXAMPLE_WITHOUT_OUTPUT_RAN")
}

func ExampleWithOutput() {
	println("hi")
	// Output:
	// hi
}
