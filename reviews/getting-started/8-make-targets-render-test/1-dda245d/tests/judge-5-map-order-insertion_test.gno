// judge-5-map-order-insertion_test.gno — AGENTS.md:49 says "Map iteration
// order is unspecified". GnoVM walks a map's MapList in insertion order:
// insertion order verbatim, an overwrite keeps a key's slot, a delete plus
// re-insert moves it to the tail, integer keys ignore numeric order.
// Measured on gno built from gnolang/gno master bbd9b2ffe (the PR body reports
// verification on master.184+393b6f92a, a different master sha).
//
// Repro from a plain clone — no checkout of this PR needed, the claim is about
// the toolchain:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout bbd9b2ffe && make -C gnovm install   # or any gno on PATH
//	mkdir /tmp/judge5 && cd /tmp/judge5
//	printf 'module = "gno.land/r/example/probe"\ngno = "0.9"\n' > gnomod.toml
//	printf 'package probe\n\nfunc Render(path string) string { return "probe" }\n' > probe.gno
//	cp <this file> ./probe_test.gno
//	gno test -v .
//
// Observed, identical on three consecutive runs:
//
//	INSERT="mqaze"      // insertion order, not hashed and not sorted
//	OVERWRITE="mqaze"   // m["q"]=99 does not move "q"
//	REINSERT="mqzea"    // delete("a") then m["a"]=7 appends at the tail
//	INTKEYS=[9 3 7 1]   // insertion order, not numeric order
//	--- PASS: TestJudge5MapOrder
package probe

import "testing"

// The mechanism: MapValue is a doubly-linked MapList (Head/Tail) plus a vmap
// index, gnovm/pkg/gnolang/values.go:931-1016, so range walks the list.
// gnolang/gno docs/resources/gno-data-structures.md:132 states it outright and
// still tells realms not to lean on it for public Render output, which is the
// rule AGENTS.md draws from the wrong premise.
func TestJudge5MapOrder(t *testing.T) {
	m := map[string]int{}
	for i, k := range []string{"m", "q", "a", "z", "e"} {
		m[k] = i
	}
	got := ""
	for k := range m {
		got += k
	}
	t.Logf("INSERT=%q", got)
	if got != "mqaze" {
		t.Errorf("expected insertion order %q, got %q", "mqaze", got)
	}

	m["q"] = 99 // overwriting a key must not move it
	got2 := ""
	for k := range m {
		got2 += k
	}
	t.Logf("OVERWRITE=%q", got2)
	if got2 != "mqaze" {
		t.Errorf("overwrite moved a key: %q", got2)
	}

	delete(m, "a")
	m["a"] = 7 // delete then re-insert must append at the tail
	got3 := ""
	for k := range m {
		got3 += k
	}
	t.Logf("REINSERT=%q", got3)
	if got3 != "mqzea" {
		t.Errorf("expected %q after delete+re-insert, got %q", "mqzea", got3)
	}

	n := map[int]string{}
	for _, k := range []int{9, 3, 7, 1} {
		n[k] = "x"
	}
	ints := []int{}
	for k := range n {
		ints = append(ints, k)
	}
	t.Logf("INTKEYS=%v", ints)
	if len(ints) != 4 || ints[0] != 9 || ints[3] != 1 {
		t.Errorf("integer keys did not come back in insertion order: %v", ints)
	}
}
