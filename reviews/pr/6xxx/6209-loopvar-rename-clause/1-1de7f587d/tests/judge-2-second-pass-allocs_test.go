package gnolang

// Counts allocations of a second initStaticBlocks1 pass over a file of N
// functions, each holding one func literal. The ADR says the second pass
// allocates no frame; FuncDecl and FuncLitExpr push a frame map each, so the
// count grows with N. Copy into gnovm/pkg/gnolang/ and run
//   go test -short -count=1 -v -run 'TestZZJudgeSecondPassAllocs$' .

import (
	"fmt"
	"strings"
	"testing"
)

func TestZZJudgeSecondPassAllocs(t *testing.T) {
	for _, n := range []int{1, 10, 100} {
		var sb strings.Builder
		sb.WriteString("package main\n")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&sb, "func f%d(a int) int { g := func(b int) int { return b }; return g(a) }\n", i)
		}
		m := NewMachine("main", nil)
		fn := m.MustParseFile("a.gno", sb.String())
		pn := NewPackageNode("main", "main", nil)
		initStaticBlocks1(nil, pn, fn)
		a := testing.AllocsPerRun(10, func() { initStaticBlocks1(nil, pn, fn) })
		t.Logf("funcs=%d second-pass allocs=%.0f", n, a)
	}
}
