// Judge check for candidate 2: counts how often the \color fallback after
// GetNextExpr (commands.go:333, b.Unget()) runs, over the package's own tests
// and a random corpus of \color shapes mixed with brackets, braces, cell
// separators and comments. Asserts zero. Mutation applied before the run:
//
//	sed -i '333s/^\t\t\tb.Unget()$/\t\t\tzzColorFallback++; b.Unget()/' gno.land/pkg/gnoweb/markdown/mathml/commands.go
//	cp judge-2-color-fallback-unreached.go gno.land/pkg/gnoweb/markdown/mathml/zz_judge2_test.go
//	go test -count=1 -skip 'TestZZ(DiffFuzz|Solo1|Judge1|LeftNil|SwitchCommentRun)' ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"math/rand"
	"strings"
	"testing"
)

var zzColorFallback int

func TestZZJudge2ColorFallback(t *testing.T) {
	t.Logf("fallback hits from the package tests before this one: %d", zzColorFallback)
	units := []string{
		`\color`, `\color[RGB]`, `\color{red}`, `\color[RGB]{1,2,3}`, `\color[rgb]`,
		`[`, `]`, `{`, `}`, `&`, `\\`, `\cr`, "%c\n", " ", "x", `\begin{matrix}`,
		`\end{matrix}`, `\left(`, `\right)`, `\over`, `\bf`, `{}`, `[]`, `\{`, `\}`,
	}
	r := rand.New(rand.NewSource(6377))
	n := 0
	for k := 0; k < 300000; k++ {
		var sb strings.Builder
		for j := r.Intn(9) + 1; j > 0; j-- {
			sb.WriteString(units[r.Intn(len(units))])
		}
		tex := sb.String()
		if !strings.Contains(tex, `\color`) {
			continue
		}
		n++
		NewMathMLConverter().ConvertInline(tex)
		NewMathMLConverter().ConvertInline(`\begin{matrix}` + tex + `\end{matrix}`)
	}
	t.Logf("corpus inputs with \\color: %d, fallback hits total: %d", n, zzColorFallback)
	if zzColorFallback != 0 {
		t.Errorf("fallback reached %d times", zzColorFallback)
	}
}
