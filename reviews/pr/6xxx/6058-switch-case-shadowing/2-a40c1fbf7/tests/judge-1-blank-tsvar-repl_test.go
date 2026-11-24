// The REPL parses with SkipObjectResolution, so the parser's
// "no new variables on left side of :=" check never runs and a blank type
// switch variable reaches preprocess. Prints what each statement does.
// Run: copy to gnovm/pkg/repl/, go test -v -run TestJudgeBlankTSVar ./gnovm/pkg/repl/
package repl

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestJudgeBlankTSVar(t *testing.T) {
	for _, c := range []string{
		`switch _ := i.(type) { case int: x := 3; println("blank-body", x) }`,
		`switch _ := i.(type) { case int: }; println("blank-empty ok")`,
		`switch _ := i.(type) { case int: println("blank-stmt") }`,
		`switch y := i.(type) { case int: x := 3; println("named", x, y) }`,
	} {
		out, errb := new(bytes.Buffer), new(bytes.Buffer)
		r := NewRepl(WithIO(os.Stdin, out, errb))
		r.RunStatements(`var i any = 42`)
		r.RunStatements(c)
		e := errb.String()
		if i := strings.IndexByte(e, '\n'); i > 0 {
			e = e[:i]
		}
		t.Logf("%q\n  out=%q\n  err=%q", c, strings.TrimSpace(out.String()), e)
	}
}
