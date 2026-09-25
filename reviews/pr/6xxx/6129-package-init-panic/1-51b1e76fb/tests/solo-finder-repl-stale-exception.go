package repl

// A REPL Machine reused after a package-level runtime panic.
// Repro from a plain clone of gnolang/gno at 51b1e76fb3c69b6ff52e27d950fdd7392d20da2b:
//   cp <this file> gnovm/pkg/repl/solo_finder_stale_test.go
//   go test ./gnovm/pkg/repl -run TestSoloFinderStale -count=1 -v | grep STEP
//   git checkout 1fc4c140ec4064e8d66cb9828ddea280a723cca6 -- gnovm/pkg/gnolang/machine.go && rerun

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestSoloFinderStale(t *testing.T) {
	seqs := map[string][]string{
		"direct": {`a := ""`, `b := a[0]`, `z := 0`, `d := 1/z`,
			`func f() (r any) { defer func() { r = recover() }(); return 1 }`, `println(f())`},
		"viafunc": {`func g() { panic("first") }`, `g()`, `z := 0`, `d := 1/z`,
			`func f() (r any) { defer func() { r = recover() }(); return 1 }`, `println(f())`},
		"plain": {`a := ""`, `b := a[0]`, `func h() int { return 2 }`, `println(h())`, `println(3)`,
			`func k() int { defer println("d"); return 4 }`, `println(k())`},
	}
	for _, name := range []string{"direct", "viafunc", "plain"} {
		out, errb := new(bytes.Buffer), new(bytes.Buffer)
		r := NewRepl(WithIO(os.Stdin, out, errb))
		for i, s := range seqs[name] {
			r.RunStatements(s)
			r.rw.Flush()
			fmt.Printf("STEP %s %d %-12q out=%q err=%q\n", name, i, s, out.String(), errb.String())
			out.Reset()
			errb.Reset()
		}
	}
}
