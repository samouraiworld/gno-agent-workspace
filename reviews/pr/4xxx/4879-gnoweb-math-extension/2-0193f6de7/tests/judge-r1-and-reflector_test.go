// Round-1 Warnings re-run at the head, plus the judge's own probes: the entity
// pass-through (candidate 2) and a \newcommand doubling chain (reflector).
// Run: cp into gno.land/pkg/gnoweb/markdown/, then
//   go test -count=1 -run 'TestJudge' -v ./gno.land/pkg/gnoweb/markdown/
//   go test -race -count=1 -run 'TestJudgeRace' ./gno.land/pkg/gnoweb/markdown/
package markdown

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yuin/goldmark"
)

func judgeRender(src string) string {
	var buf bytes.Buffer
	_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(src), &buf)
	return buf.String()
}

func TestJudgeR1Escaping(t *testing.T) {
	for _, src := range []string{
		`$$\text{</math><script>alert('XSS')</script>}$$`,
		`$\class{x" onclick="alert(1)}{y}$`,
		`$\begin{matrix}</span><script>alert(1)</script>$`,
		`\alpha is greek`,
		`$100 is the price`,
		`$\text{&lt;b&gt;} \text{&#34;}$`,
	} {
		t.Logf("IN : %s\nOUT: %s", src, strings.Join(strings.Fields(judgeRender(src)), " "))
	}
}

func TestJudgeR1Amplification(t *testing.T) {
	for _, depth := range []int{1000, 2000, 4000, 8000} {
		src := "$" + strings.Repeat(`\sqrt{`, depth) + "x" + strings.Repeat("}", depth) + "$"
		out := judgeRender(src)
		t.Logf("depth=%-5d input=%-6d output=%-8d ratio=%.0fx", depth, len(src), len(out), float64(len(out))/float64(len(src)))
	}
}

func TestJudgeNewcommandChain(t *testing.T) {
	for _, n := range []int{8, 10, 12, 14} {
		var b strings.Builder
		b.WriteString(`$\newcommand{\za}{x}`)
		prev := `\za`
		for i := 1; i < n; i++ {
			name := fmt.Sprintf(`\z%c`, 'a'+i)
			fmt.Fprintf(&b, `\newcommand{%s}{%s%s}`, name, prev, prev)
			prev = name
		}
		b.WriteString(prev + "$")
		start := time.Now()
		out := judgeRender(b.String())
		t.Logf("levels=%-3d input=%-5d output=%-9d mi=%-6d time=%v", n, b.Len(), len(out), strings.Count(out, "<mi>"), time.Since(start))
	}
}

func TestJudgeRace(t *testing.T) {
	gm := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
	in := [][]byte{[]byte(`$E=mc^2$`), []byte(`$$\int_0^1 x^2 dx$$`)}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) { defer wg.Done(); var b bytes.Buffer; _ = gm.Convert(in[n%2], &b) }(i)
	}
	wg.Wait()
}
