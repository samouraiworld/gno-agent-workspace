package repl

// recover() at the REPL prompt, fresh and after a package-level runtime panic.
// Repro from a plain clone of gnolang/gno at 51b1e76fb3c69b6ff52e27d950fdd7392d20da2b:
//   cp <this file> gnovm/pkg/repl/judge_recover_test.go
//   go test ./gnovm/pkg/repl -run TestJudgeRecoverSibling -count=1 -v | grep JSTEP
//   git checkout 1fc4c140ec4064e8d66cb9828ddea280a723cca6 -- gnovm/pkg/gnolang/machine.go && rerun
// Observed at head: `x := recover()` after `b := a[0]` fails with a host nil
// pointer dereference; at the merge base and on a fresh REPL it returns nil.

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestJudgeRecoverSibling(t *testing.T) {
	seqs := map[string][]string{
		"fresh":   {`x := recover()`, `println(x)`},
		"direct":  {`a := ""`, `b := a[0]`, `x := recover()`},
		"viafunc": {`func g() { panic("first") }`, `g()`, `x := recover()`},
	}
	for _, name := range []string{"fresh", "direct", "viafunc"} {
		out, errb := new(bytes.Buffer), new(bytes.Buffer)
		r := NewRepl(WithIO(os.Stdin, out, errb))
		for i, s := range seqs[name] {
			r.RunStatements(s)
			r.rw.Flush()
			fmt.Printf("JSTEP %s %d %-16q out=%q err=%q\n", name, i, s, out.String(), errb.String())
			out.Reset()
			errb.Reset()
		}
	}
}
