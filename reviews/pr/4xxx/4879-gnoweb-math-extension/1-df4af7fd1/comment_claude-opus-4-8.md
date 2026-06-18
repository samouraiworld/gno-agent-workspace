# Review: PR #4879
Event: REQUEST_CHANGES

## Body
The math extension renders into the markdown pipeline behind every realm and board page, and writes its output straight to the page, bypassing the safe writer that [`handler_http.go:315`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/handler_http.go#L315) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/handler_http.go#L315) assumes sanitizes realm HTML, so escaping has to cover both the serializer and the error fallback. Verified on df4af7fd1: adversarial LaTeX emits raw `<script>` and `onclick=` through the default goldmark config; `go test -race` flags concurrent writes to the one converter shared across requests; nested `\sqrt` expands 56KB of input to 128MB of output with no bound.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/4xxx/4879-gnoweb-math-extension/1-df4af7fd1/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:144-165 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L144)
The serializer writes element text and attribute values without HTML-escaping, so `\text{</math><script>…}` and `\class{x" onclick="…}{y}` inject live script and event handlers into any rendered page. Fix: HTML-escape `n.Text` and attribute values in `MMLNode.Write`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/xss_repro_test.go <<'EOF'
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestXSSRepro(t *testing.T) {
	for _, src := range []string{
		`$$\text{</math><script>alert('XSS')</script>}$$`,
		`$\class{x" onclick="alert(1)}{y}$`,
	} {
		var buf bytes.Buffer
		_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(src), &buf)
		t.Logf("IN : %s\nOUT: %s", src, strings.Join(strings.Fields(buf.String()), " "))
	}
}
EOF
go test -run TestXSSRepro -v ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/xss_repro_test.go
```

```
IN : $$\text{</math><script>alert('XSS')</script>}$$
OUT: <p> <math ...> <semantics> <mrow> <mtext></math><script>alert('XSS')</script></mtext> </mrow> <annotation encoding="application/x-tex">\text{&lt;/math>&lt;script>alert('XSS')&lt;/script>}</annotation> </semantics> </math> </p>
IN : $\class{x" onclick="alert(1)}{y}$
OUT: <p> <math ...> <semantics> <mrow> <mi class="x" onclick="alert(1)">y</mi> </mrow> <annotation ...>\class{x" onclick="alert(1)}{y}</annotation> </semantics> </math> </p>
```
The in-content `</math><script>` and the injected `onclick` attribute appear verbatim in the page DOM.
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:294-304 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L294)
On a conversion error the renderer writes the raw LaTeX into the wrapper unescaped, and a mismatched delimiter triggers it: `$\begin{matrix}</span><script>…$` runs the script. This is a separate sink, so fixing the serializer does not close it. Fix: HTML-escape `tex` in both fallback branches.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/fallback_repro_test.go <<'EOF'
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestFallbackXSS(t *testing.T) {
	src := `$\begin{matrix}</span><script>alert(1)</script>$`
	var buf bytes.Buffer
	_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(src), &buf)
	t.Logf("%s", strings.TrimSpace(buf.String()))
}
EOF
go test -run TestFallbackXSS -v ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/fallback_repro_test.go
```

```
<p><span class="math-inline">\begin{matrix}</span><script>alert(1)</script></span></p>
```
The `</span>` closes the wrapper and `<script>alert(1)</script>` is live HTML.
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:192-204 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L192)
The block parser opens a math block on any line starting with `\` or `$` and returns a node even when no delimiter matched, so the whole line is consumed and rendered as an empty `<math>` with its text dropped. Lines like `\alpha …`, escaped `\_`, or `$100 is the price` silently vanish. Fix: return `nil` from `Open` when no valid math region was found.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/contentloss_test.go <<'EOF'
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestContentLoss(t *testing.T) {
	for _, src := range []string{`\alpha is greek`, `$100 is the price`} {
		var buf bytes.Buffer
		_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(src), &buf)
		t.Logf("IN : %s\nOUT: %s", src, strings.Join(strings.Fields(buf.String()), " "))
	}
}
EOF
go test -run TestContentLoss -v ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/contentloss_test.go
```

```
IN : \alpha is greek
OUT: <math ...> <semantics> <none></none> <annotation encoding="application/x-tex"></annotation> </semantics> </math>
IN : $100 is the price
OUT: <math ...> <semantics> <none></none> <annotation encoding="application/x-tex"></annotation> </semantics> </math>
```
The line text is gone; the output is an empty math element.
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:261-265 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L261)
The converter is built once per renderer and shared across all requests, but each conversion mutates its fields, so concurrent page views data-race on it (`go test -race` confirms). Fix: build the converter per render, or use the stateless `InlineStyle`/`DisplayStyle`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/race_repro_test.go <<'EOF'
package markdown

import (
	"bytes"
	"sync"
	"testing"

	"github.com/yuin/goldmark"
)

func TestSharedConverterRace(t *testing.T) {
	gm := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
	in := [][]byte{[]byte(`$E=mc^2$`), []byte(`$$\int_0^1 x^2 dx$$`)}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) { defer wg.Done(); var b bytes.Buffer; _ = gm.Convert(in[n%2], &b) }(i)
	}
	wg.Wait()
}
EOF
go test -race -run TestSharedConverterRace ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/race_repro_test.go
```

```
WARNING: DATA RACE
Write at 0x... by goroutine 13:
  mathml.(*MathMLConverter).render()
      gno.land/pkg/gnoweb/markdown/mathml/mathml.go:122
  mathml.(*MathMLConverter).ConvertInline()
Previous write at 0x... by goroutine 58:
  mathml.(*MathMLConverter).render()
      gno.land/pkg/gnoweb/markdown/mathml/mathml.go:122
Found 1 data race(s)
FAIL
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:129-179 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L129)
Rendered output grows quadratically with nesting depth and nothing caps it: 56KB of nested `\sqrt` produces 128MB, and ~200KB OOM-kills the gnoweb process. Fix: cap the input length and/or output size per math block.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/amp_repro_test.go <<'EOF'
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestAmplification(t *testing.T) {
	for _, depth := range []int{1000, 2000, 4000, 8000} {
		src := "$" + strings.Repeat(`\sqrt{`, depth) + "x" + strings.Repeat("}", depth) + "$"
		var buf bytes.Buffer
		_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(src), &buf)
		t.Logf("depth=%-5d input=%-6d output=%-10d ratio=%.0fx", depth, len(src), buf.Len(), float64(buf.Len())/float64(len(src)))
	}
}
EOF
go test -run TestAmplification -v ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/amp_repro_test.go
```

```
depth=1000  input=7003   output=2036279    ratio=291x
depth=2000  input=14003  output=8072279    ratio=576x
depth=4000  input=28003  output=32144279   ratio=1148x
depth=8000  input=56003  output=128288279  ratio=2291x
```
The ratio doubles as depth doubles; ~200KB of nested math exhausts memory.
</details>
