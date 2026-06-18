# Review: PR #4879
Event: REQUEST_CHANGES

## Body
The math extension renders into the markdown pipeline behind every realm and board page, and writes its output straight to the page, bypassing the safe writer that [`handler_http.go:315`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/handler_http.go#L315) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/handler_http.go#L315) assumes sanitizes realm HTML, so escaping has to cover both the serializer and the error fallback. Verified on df4af7fd1: adversarial LaTeX emits raw `<script>` and `onclick=` through the default goldmark config; `go test -race` flags concurrent writes to the one converter shared across requests; nested `\sqrt` expands 56KB of input to 128MB of output with no bound.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/4xxx/4879-gnoweb-math-extension/1-df4af7fd1/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:144-165 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L144)
The MathML serializer writes element text and attribute values with no HTML escaping, so LaTeX like `\text{</math><script>...}` or `\class{x" onclick="...}{y}` injects live script or event-handler attributes into any rendered realm or board page. The same gadget reaches `mathcolor`, `voffset`, and `rowspan` through `\textcolor`/`\raisebox`/`\multirow`. Fix: HTML-escape `n.Text` and every attribute value in `MMLNode.Write`.

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
When conversion fails the renderer writes the original LaTeX into the wrapper `<span>`/`<div>` unescaped, and a mismatched delimiter is enough to trigger it: `$\begin{matrix}</span><script>...$` closes the span and runs the script. This is a separate sink from the serializer, so escaping there does not close it. Fix: HTML-escape `tex` in both fallback branches.

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

## gno.land/pkg/gnoweb/markdown/ext_math.go:261-265 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L261)
Each conversion mutates the converter's `currentExpr`/`currentIsDisplay` fields, but one converter instance is built per renderer and shared across every request, so concurrent page views data-race on it and `go test -race` reports a write/write race. The effect is cross-request output corruption with no attacker needed. Fix: build the converter per render, or call the stateless `InlineStyle`/`DisplayStyle`.

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
  markdown.(*MathRenderer).renderMath()
Previous write at 0x... by goroutine 58:
  mathml.(*MathMLConverter).render()
      gno.land/pkg/gnoweb/markdown/mathml/mathml.go:122
# ...
Found 1 data race(s)
FAIL
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:129-179 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L129)
Output grows quadratically with nesting depth because each level is pretty-printed with `2*indent` spaces, and nothing caps it: 56KB of nested `\sqrt` renders to 128MB (2291x), and ~200KB OOM-kills the gnoweb process on every view. Fix: cap the input length and/or rendered byte count per math block.

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
The ratio doubles as depth doubles; extrapolating, ~200KB of nested math exhausts memory.
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:41-44 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L41)
These delimiters are the literal byte sequences `\\(`/`\\[` (two backslashes), so the standard one-backslash LaTeX forms `\(...\)` and `\[...\]` — the forms the PR description advertises — render as plain text. Fix: accept the single-backslash forms, or drop them and document only `$...$`/`$$...$$`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/delim_repro_test.go <<'EOF'
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestDelim(t *testing.T) {
	for _, src := range []string{`\(E=mc^2\)`, `\[E=mc^2\]`, `\\(E=mc^2\\)`} {
		var buf bytes.Buffer
		_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(src), &buf)
		t.Logf("math=%-5v  in=%q", strings.Contains(buf.String(), "<math"), src)
	}
}
EOF
go test -run TestDelim -v ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/delim_repro_test.go
```

```
math=false  in="\\(E=mc^2\\)"      # actual bytes: \(E=mc^2\)  (standard LaTeX)
math=false  in="\\[E=mc^2\\]"      # actual bytes: \[E=mc^2\]  (standard LaTeX)
math=true   in="\\\\(E=mc^2\\\\)"  # actual bytes: \\(E=mc^2\\) (two backslashes)
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/mathml.go:105-121 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L105)
Conversion recovers every panic into a generic error node with no log, stack, or metric, so a crashing input degrades silently with no operator signal. Fix: log the panic and stack before returning the error node.

## gno.land/pkg/gnoweb/markdown/mathml/mathml.go:176 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L176)
The `<annotation>` child stores the raw tex escaping only `<`, so `>`, `&`, and `"` pass through and can produce malformed markup. Folding this into the global serializer-escaping fix covers it.

## gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD:1 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD#L1)
The MIT licence file ships, but none of the ~10 TreeBlood-derived `.go` files carry the upstream copyright/permission notice. Fix: add a one-line upstream + MIT header to each derived file.

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:152-161 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L152)
Attribute keys are sorted before output but the `CSS` map is iterated unsorted, so a node with two or more inline-style properties serializes its `style` in random order. This yields non-deterministic HTML and flaky golden tests; sort the CSS keys too.

## gno.land/pkg/gnoweb/markdown/ext_math.go:117-122 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L117)
Stray commented-out debug lines (`// fmt.Println(string(line))` and `// count := 0`). Delete.

## gno.land/pkg/gnoweb/markdown/ext_math.go:36-37 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L36)
`delimeter_ams`/`delimeter_tex` are misspelled; should be `delimiter_*`.

## gno.land/pkg/gnoweb/markdown/mathml/environnements.go:1 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L1)
Filename typo (French double-n): `environnements.go` should be `environments.go`.

## gno.land/pkg/gnoweb/markdown/mathml/mathml.go:92-96 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L92)
`NewMathMLConverter` accepts but discards its `macros` argument, and `\newcommand`/`\def` are parsed then dropped, so user-defined macros never take effect. Wire them through, or remove the parameter and document the limitation.

## gno.land/pkg/gnoweb/markdown/mathml/mathml.go:82 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L82)
`unknownCommandsAsOps` is read at `commands.go:354` but never set anywhere, so that branch is dead. Remove the field and branch, or add a way to enable it.

## gno.land/pkg/gnoweb/markdown/mathml/mathml.go:64-72 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L64)
`TexToMML`, `DisplayStyle`, `InlineStyle`, `SemanticsOnly`, and `NewDocument` are exported but have no callers. Either make the renderer use the stateless pair (which also fixes the shared-state race) or unexport the unused surface.

## gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go:1 [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go#L1)
The 3146-line suite covers only well-formed LaTeX: nothing asserts that adversarial input stays escaped, and there is no fuzz target or `-race` concurrency test. Add these after the escaping and shared-state fixes land.
