# Math in gnoweb pages: LaTeX turned into MathML on the server

Written by claude-opus-5-5, high effort.
PR: [gnolang/gno#4879](https://github.com/gnolang/gno/pull/4879)

## TLDR

Before, gnoweb showed `$E = mc^2$` in a realm page as typed, dollar signs
included. After, gnoweb converts LaTeX between `$…$`, `$$…$$`, `\\(…\\)` or
`\\[…\\]`, or in a fenced code block whose info string is `math`, into MathML
on the server, through a
[Go package](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L97-L105)
ported from [TreeBlood](https://github.com/Wyatt915/treeblood), and the browser
draws it [natively](https://developer.mozilla.org/en-US/docs/Web/MathML). An
expression over [8,192 bytes](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L21),
nested [64 levels](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L11)
deep, failing to convert, or met once the page's
[MathML budget](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L49-L56)
is spent shows as its escaped LaTeX source.

## What it is for

The [PR description](https://github.com/gnolang/gno/pull/4879) asks for LaTeX
support done entirely on the server. MathML is drawn by the browser itself, so
a page needs neither a front-end plugin nor generated images. A realm author
writes the same TeX that
[GitHub renders](https://docs.github.com/en/get-started/writing-on-github/working-with-advanced-formatting/writing-mathematical-expressions)
in Markdown.

## How it works today

The before state, read at the merge base.

- gnoweb renders a realm's Markdown with one
  [goldmark](https://github.com/yuin/goldmark) instance, built from
  [`NewGnoExtension`](https://github.com/gnolang/gno/blob/87f0357fe/gno.land/pkg/gnoweb/render_config.go#L42)
  and used for every request in
  [`RenderRealm`](https://github.com/gnolang/gno/blob/87f0357fe/gno.land/pkg/gnoweb/render.go#L135-L136).
- [`GnoExtension.Extend`](https://github.com/gnolang/gno/blob/87f0357fe/gno.land/pkg/gnoweb/markdown/ext.go#L67-L97)
  installs gnoweb's own syntax: foreign blocks, columns, alerts, links, forms
  and mentions. None of them reads `$`.
- A `<math>` tag written by hand is raw HTML, which goldmark drops unless the
  node runs with
  [`UnsafeHTML`](https://github.com/gnolang/gno/blob/87f0357fe/gno.land/pkg/gnoweb/app.go#L150-L153).
  The stylesheet already gives `math` elements a
  [monospace font](https://github.com/gnolang/gno/blob/87f0357fe/gno.land/pkg/gnoweb/frontend/css/05-composition.css#L406-L408).

## What the change does

### Before and after, measured

Each row is one Markdown input rendered through goldmark with
`NewGnoExtension()`, the setup the PR's own tests use, once in a merge-base
tree for Before and once in a pull request head tree for After. Neither run
turned on `UnsafeHTML`.

| Markdown input | Before | After |
| --- | --- | --- |
| `$E = mc^2$` | the text `$E = mc^2$` | inline `<math>`: `E`, `=`, `m`, and `c` with a superscript `2` |
| `$$\frac{a}{b}$$` on one line | the text | `<math display="block">` holding an `<mfrac>`, inside the paragraph |
| `$$`, `\frac{a}{b}`, `$$` on three lines | the text `$$\frac{a}{b}$$` | the same block `<math>`, with no paragraph around it |
| ```` ```math ````, `\frac{a}{b}`, ```` ``` ```` on three lines | a code block of class `language-math` | the same block `<math>`, with no code block around it |
| `\\(x^2\\)` | the text `\(x^2\)` | inline `<math>` |
| `\\[x^2\\]` | the text `\[x^2\]` | `<math display="block">`, inside the paragraph |
| `$5 and $10` | the text | the text, unchanged |
| `a $ 5 fee` | the text | the text, unchanged |
| `costs \$5` | `costs $5` | `costs $5` |
| `$$` never closed, a blank line, `next paragraph` | two paragraphs of text | two paragraphs of text, unchanged |
| `$$`, `x`, `- y`, `$$` on four lines | a paragraph `$$ x` and a list item `y $$` | the same paragraph and list item: the list line ends the math before the second `$$` |
| `$\text{<b>hi</b>}$` | the text, each tag replaced by `<!-- raw HTML omitted -->` | `<mtext>` holding `&lt;b&gt;hi&lt;/b&gt;`, shown as literal characters |
| `$\class{warn}{x}$` | the text | `<math>` holding `x`; the class name is dropped |
| `$\color{red}{x}$` | the text | `x` inside `<mstyle class="math-color-red">` |
| `$\color{#ff0000}{x}$` | the text | `x` inside an `<mstyle>` with no class, in the text colour |
| 63 nested `\sqrt{` | the text | `<math>` |
| 64 nested `\sqrt{` | the text | `<span class="math-inline">` holding the escaped LaTeX |
| `$` + 8,192 bytes + `$` | the text | `<math>` |
| `$` + 8,193 bytes + `$` | the text | `<span class="math-inline">` holding the escaped LaTeX |

### The path an expression takes

The after state, from Markdown source to the page.

```mermaid
flowchart TD
  MD["realm Markdown"] --> IP["texInlineRegionParser: $…$, $$…$$ on one line, \\(…\\), \\[…\\]"]
  MD --> BP["texBlockRegionParser: $$ or \\[ opening a line, closed by a later line"]
  MD --> FC["fenced code block with info string math"]
  FC --> TR["mathTransformer"]
  IP --> N["mathInlineNode or mathBlockNode, holding the LaTeX and the page's budget"]
  BP --> N
  TR --> N
  N --> R["MathRenderer.renderMath"]
  R -- "up to 8,192 bytes, budget left" --> C["a new MathMLConverter: tokenize, ParseTex, MMLNode.Write"]
  R -- "over 8,192 bytes, or budget spent" --> FB["escaped LaTeX in span.math-inline or div.math-display; a fence stays a code block"]
  C -- "converted, output within both bounds" --> OUT["MathML math element in the page"]
  C -- "error, nesting past 63, or output past a bound" --> FB
```

### Finding the math in Markdown

- The inline parser,
  [`texInlineRegionParser.Parse`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L143-L227),
  runs on every
  [`\` and `$`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L139-L141).
  It takes `$…$` and `\\(…\\)` as inline math, and `$$…$$` and `\\[…\\]` as
  display math when both delimiters sit on one line. An expression may run onto
  [one following line](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L200-L210).
- A single `$` follows
  [Pandoc's rule](https://pandoc.org/MANUAL.html#extension-tex_math_dollars)
  so prices stay text. The opening `$` must be followed by a
  [non-space](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L159-L163),
  and
  [`findDollarClose`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L333-L347)
  accepts a closing `$` only when no space or escaping backslash precedes it
  and no digit follows it.
- [`findCloseCached`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L286-L318)
  keeps the last search for a closing delimiter on the opener's line and on the
  next one,
  [one cache per delimiter](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L266-L271),
  so a line of unclosed `$a ` openers is searched once rather than once per
  opener.
- An expression that is
  [empty or holds a footnote reference](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L211-L218)
  stays text:
  [`refersToFootnote`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L237-L264)
  reads `$5[^1] or 4$` as two prices around a reference to a footnote the page
  defines.
- The backslash delimiters are written with
  [two backslashes](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L88-L95)
  in the Markdown source, `\\(x^2\\)`, the form plain gnoweb already showed as
  `\(x^2\)`.
- The block parser,
  [`texBlockRegionParser.Open`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L353-L388),
  opens a block only on a line
  [starting with `$$` or `\\[`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L364-L367)
  whose closing delimiter is
  [absent from that line](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L374-L376)
  and present on a later closing line, checked by
  [`hasClosingLine`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L554-L609).
  A [closing line](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L416-L426)
  ends with the delimiter and holds it once.
- A math block is read like a paragraph:
  [`endsMath`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L542-L544)
  ends it at a blank line or at any line that would open a list, a quote, a
  heading, a fence or an HTML block, and
  [a line outside the block's quote or list item](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L598-L601)
  ends it too. An opener with no closing line before that stays a paragraph and
  leaves the rest of the page alone.
- `hasClosingLine` stores how far its last scan reached on the parse context,
  [one entry per closing delimiter](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L555-L558)
  and [per container](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L566-L582),
  so a page of unclosed openers is scanned once overall rather than once per
  opener.
- The lines that end display math include the blocks gnoweb's other
  extensions open:
  [`GnoExtension.Extend`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext.go#L119)
  passes their block parsers to
  [`NewExtMath`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L873-L879),
  and gnoweb's render config adds goldmark's footnote parser through
  [`WithPeerBlockParsers`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/render_config.go#L44-L45).
- [`mathTransformer`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L734-L778)
  runs once the inline parsers are done. It turns a
  [fenced code block whose info string is `math`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L764-L777)
  into a display math block, and a `$$` block whose lines refer to a footnote
  back into
  [the paragraph it would be](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L755-L763).

### Converting LaTeX to MathML

The `mathml` package is a port of TreeBlood, with its MIT licence kept in
[`LICENCE.MD`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD?plain=1#L1-L3).
One conversion runs these steps:

1. [`tokenize`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L439)
   splits the LaTeX into
   [tokens](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L26-L50):
   commands, letters, numbers, braces and the rest.
2. [`ParseTex`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L82)
   builds a tree of
   [`MMLNode`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L45-L53)
   elements. Commands such as `\frac` go through
   [`ProcessCommand`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L206),
   environments such as `pmatrix` through
   [`processEnv`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L258),
   and a name such as `\alpha` becomes `α` through
   [`symbolTable`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L108).
3. [`wrapInMathTag`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L68-L85)
   puts the tree in a `<math>` element and keeps the LaTeX source beside it in
   an
   [`<annotation encoding="application/x-tex">`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L81-L83).
4. [`MMLNode.Write`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L154-L207)
   prints the tree on one line, since
   [whitespace around inline math would show as spaces](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L59-L61).

A panic anywhere in these steps is
[recovered](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L42-L50)
and returned as an error, and
[`renderMath`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L847-L861)
then writes the escaped LaTeX instead of the MathML.

### Writing the markup into the page

The renderer
[writes its markup straight to the output](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L836).
goldmark's raw-HTML filter therefore does not touch it: the measured runs
produced `<math>` with `UnsafeHTML` off. What reaches the page is controlled
by the converter instead.

- Every text and attribute value passes through
  [`writeEscaped`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L12-L27),
  which escapes `<`, `>`, `"` and every `&`, so an entity the author typed,
  `&lt;`, displays as typed.
- An attribute whose name falls outside letters, digits, `-`, `_` and `:` is
  [skipped](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L178-L181).
- [`\class{name}{x}`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L244-L251)
  renders `x` and drops `name`, so a page cannot apply the site's own CSS
  classes.
- `\color` and `\textcolor` accept only the
  [theme colours](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L196-L204),
  red, blue, green, orange, purple and gray, each as a
  [`math-color-` class](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L207-L211)
  the stylesheet
  [colours](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/frontend/css/05-composition.css#L457-L480).
  Any other colour leaves the text colour.
- The fallback escapes the LaTeX with
  [`html.EscapeString`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L860).
- Attributes are printed in
  [sorted order](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L170-L177),
  so one input always yields the same bytes.

### Limits

The after state. Each limit caps what one expression or one page can cost the
server, or how far math can draw over the page around it.

| Limit | Value | Past it |
| --- | --- | --- |
| Expression length | [`MaxMathInputLen`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L21), 8,192 bytes | [not converted](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L820); the escaped LaTeX is shown |
| Parser nesting | [`MaxParseDepth`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L11), 64 levels, the outermost one included | [`ParseTex` panics](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L85-L89), the panic is recovered, and the escaped LaTeX is shown |
| MathML of one expression | [`maxMathOutputLen`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L33), 64 bytes per input byte plus 4,096 | [the output is discarded](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L831); the escaped LaTeX is shown |
| MathML of one page | [`mathBudgetFrom`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L53), `maxMathOutputLen` of the page's size, at most [`MaxMathPageOutput`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L27), 2 MiB | [each later expression](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L820) shows its escaped LaTeX |
| Block lookahead | [8,192 bytes](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L563) past the opening line | the opener stays a paragraph |
| Size switches | [`minSize` and `maxSize`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L361-L364), `\tiny` to `\Huge`, nesting included | the size is [clamped](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L374) |
| `\raisebox` shift | [`maxRaisePt`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L50), 2em, nesting included | [the shift is ignored](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L72-L74) |
| `\multirow` and `\multicolumn` span | [`maxCellSpan`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L15), 64 | [the span is ignored](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L95-L97) |

### One converter per expression

A `MathMLConverter` holds
[state for the expression it is converting](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L16-L21),
and gnoweb serves every request from
[one goldmark instance](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/render.go#L114).
[`renderMath`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L820-L823)
therefore builds a new converter for every expression.
[`TestMathConcurrentRender`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math_test.go#L269-L281)
renders 50 expressions at once, and the race workflow
[runs it under `-race`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/.github/workflows/ci-race.yml#L61-L62),
triggered by changes under
[`gno.land/pkg/gnoweb/markdown/`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/.github/workflows/ci-race.yml#L29).

### Where math renders

The after state, per gnoweb surface.

| Surface | Math renders | Reason |
| --- | --- | --- |
| Realm pages | yes | [`GnoExtension.Extend`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext.go#L119) calls `NewExtMath(...).Extend` |
| `<gno-foreign>` blocks | no | the inner instance, [`buildInnerForeignMarkdown`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_foreign.go#L445-L464), loads no math extension |
| Package documentation | no | the documentation instance, [`docOpts`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/render.go#L101-L108), loads no `GnoExtension` |

### Styling, documentation and tests

- The stylesheet sets the
  [math font, size, colours and block scrolling](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/frontend/css/05-composition.css#L413-L534)
  in em and theme tokens, compiled into
  [`public/main.css`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/public/main.css#L1). Math left
  unconverted
  [shows as code](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/frontend/css/05-composition.css#L606-L612).
- The `r/docs/markdown` realm gains a
  [Mathematics section](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/examples/quarantined/gno.land/r/docs/markdown/markdown.gno?plain=1#L917-L1142)
  showing each syntax beside its rendering.
- [`pr4879_gnoweb_math_extension.md`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/adr/pr4879_gnoweb_math_extension.md?plain=1#L1)
  records the design: parsing rules, security model, denial-of-service model
  and styling.
- [19 golden files](https://github.com/alexiscolin/gno/tree/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/golden/ext_math)
  pin the rendered HTML through gnoweb's existing
  [golden runner](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_test.go#L63-L75),
  which [walks subdirectories](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_test.go#L66).
  [`mathml_test.go`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go#L12)
  tests the converter alone in 84 test functions, and
  [`FuzzMathRender`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math_test.go#L463-L501)
  fuzzes the extension for script tags, `on…` attributes, `javascript:` URLs,
  any `<svg>` but gnoweb's own icons, a render over
  [10 seconds](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math_test.go#L486)
  and output past
  [`maxMathOutputLen` of the input](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math_test.go#L493)
  plus [16 bytes per input byte](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math_test.go#L461).

## Concepts

| Term | Meaning here |
| --- | --- |
| LaTeX | The markup mathematicians write formulas in: `\frac{a}{b}` for a fraction, `x^2` for a power, `\alpha` for α. |
| MathML | The [W3C markup for mathematics](https://developer.mozilla.org/en-US/docs/Web/MathML). A browser draws a `<math>` element itself, as it draws a table. |
| Inline and display math | Inline math sits in a line of text, `display="inline"`; display math is a block of its own, `display="block"`. |
| Delimiter | The marks around an expression: `$…$` and `\\(…\\)` for inline, `$$…$$` and `\\[…\\]` for display. |
| Math fence | A fenced code block whose info string is exactly `math`, which [GitHub also renders](https://docs.github.com/en/get-started/writing-on-github/working-with-advanced-formatting/writing-mathematical-expressions) as display math. |
| Pandoc's dollar rule | The convention, from the [Pandoc document converter](https://pandoc.org/MANUAL.html#extension-tex_math_dollars), that `$` opens math only before a non-space and closes it only after a non-space and before a non-digit. |
| goldmark | The [Go Markdown library](https://github.com/yuin/goldmark) gnoweb renders with; an extension adds parsers that turn text into nodes and renderers that print those nodes. |
| TreeBlood | The open-source Go LaTeX-to-MathML converter the `mathml` package is ported from. |
| MathML budget | The bytes of MathML one render has left to write, [set from the page's size](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L49-L56) the first time an expression is parsed; [each conversion spends its output](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L832-L834) from it, whether the output is kept or discarded. |
| Fallback | What a failed or refused expression becomes: its LaTeX, HTML-escaped, in `<span class="math-inline">` or `<div class="math-display">`; a math fence stays the code block it was. |

## Review files

[Review files for this PR](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/4xxx/4879-gnoweb-math-extension)
