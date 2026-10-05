# Review: [#4879](https://github.com/gnolang/gno/pull/4879)
Posted: https://github.com/gnolang/gno/pull/4879#pullrequestreview-5416357299
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Three Warnings the branch introduces block it: an unclosed `$$` line swallows every block up to a later `$$`, unclosed inline `$` openers render in quadratic time, and the 8 KiB cap leaves a page of `&` cells rendering at 107x its size.
Model: claude-opus-5-5, standard review, solo wider: finders xhigh, judge and writer high
Commit: 0193f6de7 (latest)
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-4879 0193f6de7`
Round: 2. Two finders, one reflector pass, 12 candidates: 5 from the finders and 7 from the reflector, five of those re-checking the round-1 Warnings. Every candidate, the one Nit included, was run from scratch by a judge that was not its finder; the finders dropped 15 more on their own read.

## Body

## gno.land/pkg/gnoweb/markdown/ext_math.go:284 [gh](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/ext_math.go#L284) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L284) · Warning [posted](https://github.com/gnolang/gno/pull/4879#discussion_r4185312540)
`bytes.Contains(line, closeTag)` accepts any later line holding `$$`, so an unclosed `$$` line followed by an inline `$$y$$` renders every block between them as math.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/swallow_repro_test.go <<'GO'
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestUnclosedDisplaySwallows(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"heading", "$$\noops forgot to close\n\n## Section 2\n\nMore text with $$y$$ here.\n", "<h2"},
		{"fence", "$$\nunclosed\n\n\x60\x60\x60go\n$$\n\x60\x60\x60\n\nlast paragraph\n", "<p>last paragraph</p>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(c.src), &buf)
			if out := buf.String(); !strings.Contains(out, c.want) {
				t.Errorf("want %q, got:\n%s", c.want, strings.Join(strings.Fields(out), " "))
			}
		})
	}
}
GO
go test -run TestUnclosedDisplaySwallows -v ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/swallow_repro_test.go
```

Both cases fail: the heading, and in the second case the fence opener, are rendered inside `<math>`, and the fence's closing line opens a code block that runs to the end of the page.

```text
swallow_repro_test.go:21: want "<h2", got:
    <math class="math-displaystyle" display="block" ...> <semantics> <mrow> <mi>o</mi> <mi>o</mi> <mi>p</mi> <mi>s</mi> <mi>f</mi> <mi>o</mi> <mi>r</mi> # …
swallow_repro_test.go:21: want "<p>last paragraph</p>", got:
    <math class="math-displaystyle" display="block" ...> <semantics> <mrow> <mi>u</mi> <mi>n</mi> <mi>c</mi> <mi>l</mi> <mi>o</mi> <mi>s</mi> <mi>e</mi> <mi>d</mi> <mo>`</mo> <mo>`</mo> <mo>`</mo> <mi>g</mi> <mi>o</mi> # …
--- FAIL: TestUnclosedDisplaySwallows (0.00s)
    --- FAIL: TestUnclosedDisplaySwallows/heading (0.00s)
    --- FAIL: TestUnclosedDisplaySwallows/fence (0.00s)
```

Inside a blockquote or a list item the container's end closes the block, so only that container's own text becomes math. With no later `$$` the page renders as plain text.
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:169 [gh](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/ext_math.go#L169) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L169) · Warning [posted](https://github.com/gnolang/gno/pull/4879#discussion_r4185312550)
`findDollarClose` skips every `$` after a space, so each unclosed `$a ` opener rescans the rest of its line, and a 1 MiB line of openers renders in 56 s.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/quad_repro_test.go <<'GO'
package markdown

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/yuin/goldmark"
)

func TestInlineDollarQuadratic(t *testing.T) {
	for _, n := range []int{1 << 13, 1 << 14, 1 << 15, (1 << 20) / 3} {
		src := strings.Repeat("$a ", n)
		var buf bytes.Buffer
		gm := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
		start := time.Now()
		_ = gm.Convert([]byte(src), &buf)
		t.Logf("n=%-7d input=%-8d time=%v", n, len(src), time.Since(start))
	}
}
GO
go test -count=1 -timeout 600s -run TestInlineDollarQuadratic -v ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/quad_repro_test.go
```

Render time grows fourfold per doubling of the input, which is the finding; the last row is a 1 MiB page, the size gnoweb's [`maxMarkdownRenderBytes`](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/render.go#L29) admits.

```text
quad_repro_test.go:19: n=8192    input=24576    time=31.650587ms
quad_repro_test.go:19: n=16384   input=49152    time=119.456544ms
quad_repro_test.go:19: n=32768   input=98304    time=458.956213ms
quad_repro_test.go:19: n=349525  input=1048575  time=55.632560958s
```

The same file at the merge base, where `$` is plain text, prints 1.1 ms, 2.0 ms, 3.8 ms and 41 ms. Unclosed `\\(a ` openers rescan the same way, through the [`bytes.Index`](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/ext_math.go#L129) call that looks for the closing `\\)`.
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:356 [gh](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/ext_math.go#L356) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L356) · Warning [posted](https://github.com/gnolang/gno/pull/4879#discussion_r4185312557)
`MaxMathInputLen` bounds one expression and not a page, so a 1 MiB page of `aligned` environments of bare `&` renders to 112 MB of HTML.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4879 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/amp_repro_test.go <<'GO'
package markdown

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func ampPage(t *testing.T, name, para string) {
	src := strings.Repeat(para, (1<<20)/len(para))
	runtime.GC()
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	var buf bytes.Buffer
	_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(src), &buf)
	runtime.ReadMemStats(&m1)
	t.Logf("%-16s in=%-8d out=%-10d ratio=%-4.0f alloc=%dMB", name, len(src), buf.Len(),
		float64(buf.Len())/float64(len(src)), (m1.TotalAlloc-m0.TotalAlloc)>>20)
}

func TestPageAmplification(t *testing.T) {
	ampPage(t, "aligned-amp", "$\\begin{aligned}"+strings.Repeat("&", MaxMathInputLen-32)+"\\end{aligned}$\n\n")
	ampPage(t, "blockquote-base", strings.Repeat(">", 4096)+"\n\n")
}
GO
go test -count=1 -run TestPageAmplification -v ./gno.land/pkg/gnoweb/markdown/
mkdir -p gno.land/pkg/gnoweb/markdown/testdata/fuzz/FuzzMathRender
python3 -c 'open("gno.land/pkg/gnoweb/markdown/testdata/fuzz/FuzzMathRender/aligned_amp","w").write("go test fuzz v1\nstring(\"\\\\begin{aligned}" + "&"*8150 + "\\\\end{aligned}\")\n")'
go test -count=1 -run FuzzMathRender ./gno.land/pkg/gnoweb/markdown/
rm -r gno.land/pkg/gnoweb/markdown/amp_repro_test.go gno.land/pkg/gnoweb/markdown/testdata/fuzz/FuzzMathRender
```

The first test prints the amplification, 107 times the input, against nested blockquotes as the heaviest non-math page of the same size; `FuzzMathRender` then fails its own output bound on one such expression.

```text
amp_repro_test.go:20: aligned-amp      in=1048576  out=111811584  ratio=107  alloc=2538MB
amp_repro_test.go:20: blockquote-base  in=1044990  out=28200960   ratio=27   alloc=225MB
# …
--- FAIL: FuzzMathRender/aligned_amp (0.01s)
    ext_math_test.go:137: output too large: 872458 bytes for 8180 bytes of input
```

Each `&` emits an `<mtd>` carrying `text-align` and `padding` CSS at a 32-space indent. A `pmatrix` of bare `&` renders at 54 times its input.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:32 [gh](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L32) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L32) · Nit [posted](https://github.com/gnolang/gno/pull/4879#discussion_r4185312570)
Nit: `writeEscaped` passes an entity reference the author typed through unescaped, so `$\text{&lt;b&gt;}$` displays `<b>` and the x-tex annotation decodes to `\text{<b>}`.

<details><summary>output</summary>

```text
IN : $\text{&lt;b&gt;} \text{&#34;}$
OUT: <mtext>&lt;b&gt;</mtext> <mtext>&amp;34;</mtext> # …
     <annotation encoding="application/x-tex">\text{&lt;b&gt;} \text{&#34;}</annotation>
```
</details>
