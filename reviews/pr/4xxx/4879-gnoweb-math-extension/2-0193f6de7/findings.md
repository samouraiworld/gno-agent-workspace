# Findings in posting order, from round assemble: 4 to post, 0 SKIP, 7 refuted kept out

## gno.land/pkg/gnoweb/markdown/ext_math.go:169 [gh](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/ext_math.go#L169) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: A line of unclosed inline $ openers renders in quadratic time: 1 MiB of '$a ' takes about 56 s per render against 41 ms at the merge base
Check: cp tests/solo-finder-2-inline-dollar-quadratic.go gno.land/pkg/gnoweb/markdown/zz_quad_repro_test.go && go test -v -count=1 -timeout 600s -run TestInlineDollarQuadratic ./gno.land/pkg/gnoweb/markdown/ at 0193f6de7 and at 87f0357fe: time grows 4x per doubling at head, linearly at base
Details: findDollarClose (ext_math.go:169) skips every $ preceded by a space, so each '$a ' opener scans to the end of its line and then the next line before giving up; time grows 4x per doubling. gnoweb's maxMarkdownRenderBytes (render.go:29) is 1 << 20, so a 1 MiB realm Render output is accepted.
Evidence: head 0193f6de7: n=8192 33.0ms, n=16384 126.1ms, n=32768 498.4ms, n=349525 (1048575 bytes) 55.67s. Same file at merge base 87f0357fe: 1.12ms, 1.95ms, 3.79ms, 40.95ms.
Artifact: tests/solo-finder-2-inline-dollar-quadratic.go

## gno.land/pkg/gnoweb/markdown/ext_math.go:284 [gh](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/ext_math.go#L284) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: An unclosed $$ line opens a display block whenever any later line within 8 KiB contains $$, so the headings, paragraphs and code fences in between render as math letters
Check: cp tests/solo-finder-1-unclosed-display-swallows_test.go gno.land/pkg/gnoweb/markdown/finder1_swallow_test.go && go test -run TestFinder1UnclosedDisplaySwallows -v ./gno.land/pkg/gnoweb/markdown/ at head 0193f6de7: expect PASS if closed, observe 4/4 FAIL
Details: hasClosingLine at ext_math.go:284 accepts the first later line containing the close tag, with no stop at a blank line or a block boundary. '$$\noops forgot to close\n\n## Section 2\n\nMore text with $$y$$ here.' renders the heading and paragraph as <mi> letters with two '?#' merrors, leaving '<p>y$$ here.</p>'. With the later $$ inside a fenced block, the fence opener is eaten and the closing ``` opens a fence that runs to the end of the page. Inside a blockquote or list item the container end closes the block, so only that container's own text becomes math.
Evidence: TestFinder1UnclosedDisplaySwallows rerun at 0193f6de7: 4/4 FAIL; case 2 output '<math ...><mi>u</mi>...<mo>`</mo><mo>`</mo><mo>`</mo><mi>g</mi><mi>o</mi>...</math> <pre><code> last paragraph </code></pre>'. Merge base 87f0357fe: ls gno.land/pkg/gnoweb/markdown | grep -c math -> 0, NewGnoExtension carries no math parser.
Artifact: tests/solo-finder-1-unclosed-display-swallows_test.go
State: CONFIRMED, band: Warning, angle: lines
TL;DR: A stray $$ line swallows every block up to a later inline $$x$$, the same lookahead defect as candidate 1
Check: cp tests/solo-finder-2-block-swallow.go gno.land/pkg/gnoweb/markdown/zz_swallow_repro_test.go && go test -v -run TestBlockSwallow ./gno.land/pkg/gnoweb/markdown/: case 1 output has no <h1> or <pre>
Details: Same mechanism as candidate 1 at ext_math.go:284: the heading, paragraph and fenced code between a lone $$ line and a later inline $$x$$ become one <math>; without the later $$ the page renders correctly.
Evidence: TestBlockSwallow at 0193f6de7: case 1 output '<math ...><mi>f</mi>...<merror title="Unexpanded macro argument">?#</merror> <mi>H</mi>...' with no <h1> and no <pre>; case 2 '<p>$$ forgot to close</p> <h1>Heading</h1> <p>A paragraph with no closer.</p>'.
Artifact: tests/solo-finder-2-block-swallow.go

## gno.land/pkg/gnoweb/markdown/ext_math.go:356 [gh](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/ext_math.go#L356) · Warning
State: CONFIRMED, band: Warning, angle: claims
TL;DR: An aligned environment of bare & renders at 107x its input, past FuzzMathRender's own 64x+4096 bound, and a 1 MiB page of such expressions renders 112 MB of HTML and allocates 2.5 GB
Check: cp tests/solo-finder-2-page-amplification.go gno.land/pkg/gnoweb/markdown/zz_amp_repro_test.go && go test -v -run TestPageAmplification ./gno.land/pkg/gnoweb/markdown/; then add the aligned corpus file from the header and run go test -run FuzzMathRender ./gno.land/pkg/gnoweb/markdown/ (expect FAIL at ext_math_test.go:137)
Details: Each & emits an <mtd> with text-align and padding CSS; the 8 KiB cap at ext_math.go:356 bounds one expression and not their count per page. The worst non-math shape at 1 MiB measured here, nested blockquotes, produces 28 MB and allocates 225 MB.
Evidence: TestPageAmplification at 0193f6de7: aligned-amp in=1048576 out=111811584 ratio=107 alloc=2538MB time=1.26s; pmatrix-amp ratio=54 alloc=1772MB; blockquote-base out=28200960 alloc=225MB. FuzzMathRender with the aligned_amp corpus entry: 'ext_math_test.go:137: output too large: 872458 bytes for 8180 bytes of input', FAIL.
Artifact: tests/solo-finder-2-page-amplification.go

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:32 [gh](https://github.com/gnolang/gno/blob/0193f6de7fc5823e3c01d4a5e580e377cde98659/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L32) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: writeEscaped passes author-typed entity references through, so \text{&lt;b&gt;} displays '<b>' and the x-tex annotation no longer carries the source as typed
Check: render $\text{&lt;b&gt;}$ through NewGnoExtension: observe '<mtext>&lt;b&gt;</mtext>' and '<annotation ...>\text{&lt;b&gt;}</annotation>' (decoded by the browser as '<b>')
Details: The entityRef pass-through at mmlnode.go:32-33 exists for symbol-table strings such as '&OverBrace;' but runs on every node text, \text{} content and the annotation included.
Evidence: IN: $\text{&lt;b&gt;} \text{&#34;}$ OUT: '<mtext>&lt;b&gt;</mtext> <mtext>&amp;34;</mtext> ... <annotation encoding="application/x-tex">\text{&lt;b&gt;} \text{&#34;}</annotation>' (TestJudgeR1Escaping).
Artifact: tests/judge-r1-and-reflector_test.go
