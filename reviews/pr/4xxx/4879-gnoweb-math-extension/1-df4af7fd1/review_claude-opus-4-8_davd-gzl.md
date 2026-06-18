# PR #4879: feat(gnoweb): math extension

URL: https://github.com/gnolang/gno/pull/4879
Author: alexiscolin | Base: master | Files: 33 | +14927 -1
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `df4af7fd1` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4879 df4af7fd1`

**TL;DR:** Adds server-side LaTeX math to gnoweb: write `$E=mc^2$` in a realm's markdown and gnoweb turns it into MathML the browser draws as a formula. The engine is a ~10k-line LaTeX-to-MathML converter vendored from TreeBlood, plus a goldmark extension that wires it into the markdown pipeline that renders every realm and board page.

**Verdict: REQUEST CHANGES** — two independent stored-XSS sinks (the MathML serializer writes text/attributes unescaped; the error fallback writes raw LaTeX) let any realm or board run script in the gnoweb origin; plus a confirmed data race on the shared converter and a quadratic-blowup OOM that crashes the gnoweb process. All four verified end-to-end on `df4af7fd1`.

## Summary

The extension registers unconditionally in the default gnoweb render path ([`render_config.go:47`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/render_config.go#L47) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/render_config.go#L47)), so it processes every realm `Render()` output and every board post, all attacker-controllable. A custom goldmark `NodeRenderer` calls `MathMLConverter.ConvertInline/Display(tex)` and writes the result straight to the page with `w.WriteString(mml)`. The serializer ([`mmlnode.go:116-184`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L116-L184) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L116-L184)) does zero HTML escaping on element text or attribute values, and the renderer bypasses goldmark's safe writer (the layer [`handler_http.go:315`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/handler_http.go#L315) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/handler_http.go#L315) assumes is sanitizing the output). The result is two XSS sinks, a process-shared mutable converter that races under concurrent requests, and a quadratic-expansion renderer with no size bound. The vendored converter was written for trusted static-site authoring; nothing about adversarial input was added on the way in.

## Examples

Delimiter behavior (the `\\`-prefixed forms are 3-byte literals `\`,`\`,`(` in the source, not standard LaTeX). Confirmed by running each through the pipeline:

| Markdown input | Renders as math? |
|----------------|------------------|
| `$E=mc^2$` | yes |
| `$$E=mc^2$$` | yes |
| `\(E=mc^2\)` (standard LaTeX) | no — emitted as literal text |
| `\[E=mc^2\]` (standard LaTeX) | no — emitted as literal text |
| `\\(E=mc^2\\)` | yes |

## Glossary

- `MathRenderer` — goldmark `NodeRenderer` that writes MathML directly to the page's `util.BufWriter`.
- `MathMLConverter` — the LaTeX-to-MathML AST builder vendored from TreeBlood; holds per-conversion mutable state.
- `MMLNode.Write` — serializer that emits `<tag attr="val">text</tag>` with no escaping.
- `ConvertInline`/`ConvertDisplay` — the converter methods the renderer calls; both mutate `currentExpr`.
- stored XSS — script that lives in saved content (here, a realm's render output) and runs in every viewer's browser.

## Fix

Before-state has no math support; after-state renders `$...$`/`$$...$$`. The load-bearing constraint the vendored code never had to satisfy: its output is interpolated into a user-facing HTML page, so every byte that originates in user LaTeX must be HTML-escaped before serialization, and the error fallback must escape too. Two distinct sinks need fixing: `MMLNode.Write` ([`mmlnode.go:149`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L149) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L149), [`:165`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L165) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L165)) and the raw-`tex` fallback ([`ext_math.go:294-304`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L294-L304) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L294-L304)). Separately, the shared converter must become per-render, and conversion must be bounded by input/output size.

## Critical (must fix)

- **[any realm or board can run script in your browser]** [`gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:144-165`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L144-L165) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L144-L165) — the MathML serializer writes element text and attribute values with no HTML escaping, so crafted LaTeX injects live markup. Fix: HTML-escape `n.Text` and every attribute value in `MMLNode.Write`.
  <details><summary>details</summary>

  `MMLNode.Write` emits attribute values via `w.WriteString(val)` ([mmlnode.go:149](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L149) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L149)) and element text via `w.WriteString(n.Text)` ([mmlnode.go:165](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L165) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L165)), neither escaped. Two families of user LaTeX reach these sinks:

  - **Text**: `\text{...}` ([commands_defs.go:283-288](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L283-L288) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L283-L288)) places its body in `<mtext>` via `stringifyTokensHtml` ([tokenize.go:462-472](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L462-L472) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L462-L472)), which only swaps space→`&nbsp;`. An in-body `</math>` closes the math element; the browser then parses the rest as HTML.
  - **Attributes**: `\class{...}{...}` ([commands_defs.go:132-139](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L132-L139) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L132-L139)) writes attacker bytes into a `class` value via `StringifyTokens`; a `"` ends the attribute and the rest becomes new attributes such as `onclick`. Same gadget in `cmd_textcolor` (`mathcolor`, [commands_defs.go:107](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L107) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L107)), `cmd_raisebox` (`voffset`, [commands_defs.go:145](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L145) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L145)), `cmd_multirow` (`rowspan`/`columnspan`, [commands_defs.go:16](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L16) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L16)), `cmd_mathop` (`mo` text, [commands_defs.go:174](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L174) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L174)).

  Centralizing the escape in `MMLNode.Write` closes every command at once and needs no per-command discipline (tags and attribute *names* come from the package, not the user, so they need no escaping). Reachable wherever gnoweb renders user markdown: an attacker deploys a realm whose `Render()` returns the payload, the victim opens it through gnoweb, and the script runs in the gnoweb origin (session theft, injected tx-signing forms). Verified end-to-end on df4af7fd1; [repro](comment_claude-opus-4-8.md).
  </details>

- **[error fallback prints raw LaTeX — second XSS, survives the serializer fix]** [`gno.land/pkg/gnoweb/markdown/ext_math.go:294-304`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L294-L304) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L294-L304) — when conversion returns an error, the renderer writes the original LaTeX into `<span>`/`<div>` unescaped, and a mismatched delimiter is enough to take that branch. Fix: HTML-escape `tex` in both fallback branches.
  <details><summary>details</summary>

  On a conversion error, `renderMath` writes `<span class="math-inline">` + raw `tex` + `</span>` ([ext_math.go:296-299](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L296-L299) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L296-L299)), the `tex` fully attacker-controlled. This is a distinct sink from the serializer: the bytes never pass through `MMLNode.Write`, so escaping there does not close it. The error branch is trivially reachable — an unterminated environment like `\begin{matrix}` or a mismatched `{`/`}` makes `ConvertInline` return a non-nil error (`mismatched environment` / `mismatched curly brace`), so `$\begin{matrix}</span><script>...</script>$` renders the `</span><script>` live. Verified end-to-end on df4af7fd1; [repro](comment_claude-opus-4-8.md).
  </details>

## Warnings (should fix)

- **[concurrent page views corrupt each other's math / risk a crash]** [`gno.land/pkg/gnoweb/markdown/ext_math.go:261-265`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L261-L265) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L261-L265) — one converter instance is shared by every request and mutated mid-render, so concurrent renders data-race on its state. Fix: build the converter per render (or call the stateless `InlineStyle`/`DisplayStyle`).
  <details><summary>details</summary>

  `NewMathRenderer` stores a single `mathml.NewMathMLConverter()` ([ext_math.go:263](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L263) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L263)), and `HTMLRenderer` keeps one goldmark instance reused across all requests ([render.go:42-56](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/render.go#L42-L56) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/render.go#L42-L56)). Each conversion writes shared fields: `converter.currentExpr` and `currentIsDisplay` ([mathml.go:122](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L122) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L122)). gnoweb serves requests concurrently, so two viewers loading math pages at once race on these. `go test -race` reports a write/write race on `currentExpr`. No attacker needed; CI runs no `-race` so it is invisible there. Racing on the `currentExpr` slice header is undefined behavior — at minimum cross-request output corruption, at worst a torn read. The package already ships stateless entry points (`InlineStyle`/`DisplayStyle` build a fresh converter per call); using one of those, or constructing the converter inside `renderMath`, removes the shared state. Verified with `-race` on df4af7fd1; [repro](comment_claude-opus-4-8.md).
  </details>

- **[a small realm page can exhaust gnoweb's memory]** [`gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:129-179`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L129-L179) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L129-L179) — output grows quadratically with nesting depth and nothing bounds it, so tens of KB of nested math render to hundreds of MB and crash the process. Fix: cap the input length and/or rendered byte count per math block.
  <details><summary>details</summary>

  The serializer pretty-prints with `2*indent` spaces per level ([mmlnode.go:130-133](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L130-L133) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L130-L133)) and the gnoweb path renders with indentation on (`PrintOneLine` is false). A nesting of depth N (e.g. `\sqrt{\sqrt{...}}`) emits N levels each indented up to 2N spaces, so output is O(N²). Measured end-to-end through the goldmark pipeline on df4af7fd1: 7KB→2MB (291×) at depth 1000, 28KB→32MB (1148×) at depth 4000, 56KB→128MB (2291×) at depth 8000 — the ratio doubles as depth doubles. ~200KB of nested math (well within a realm's render output) extrapolates to multi-GB and OOM-kills gnoweb on every view. A per-block input or output cap (KATeX caps both) keeps normal pages working and bounds the blast. Verified end-to-end on df4af7fd1; [repro](comment_claude-opus-4-8.md).
  </details>

- **[standard `\(...\)` / `\[...\]` silently don't render]** [`gno.land/pkg/gnoweb/markdown/ext_math.go:41-44`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L41-L44) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L41-L44) — the delimiters are the 3-byte literals `\\(` (two backslashes + paren), so the standard one-backslash LaTeX forms are not recognized. Fix: accept the single-backslash forms, or drop these delimiters and document only `$...$`/`$$...$$`.
  <details><summary>details</summary>

  `_inlineopen` etc. are Go raw-string literals `` `\\(` ``, which are literally `\`,`\`,`(`. A user typing `\(E=mc^2\)` (the universal LaTeX/KaTeX/MathJax form, and the form this PR's description advertises) gets unprocessed text; only `\\(E=mc^2\\)` renders. Math copied from other tools fails silently. The shipped `r/docs/markdown` example sidesteps it by documenting only the `$` forms. Confirmed by running both forms through the pipeline on df4af7fd1.
  </details>

- **[every panic is silently swallowed into "invalid math"]** [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:105-121`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L105-L121) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L105-L121) — `render`/`TexToMML` recover every panic into a generic `<merror>` with no log, and `SemanticsOnly` discards the panic entirely. Fix: log the panic and stack before returning the error node.
  <details><summary>details</summary>

  `render` ([mathml.go:105](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L105) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L105)) and `TexToMML` ([mathml.go:12](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L12) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L12)) wrap the whole conversion in `defer recover()` and return a generic error with no stack, log, or metric. A malformed input that crashes the parser degrades for every viewer with no operator signal. Worse, `SemanticsOnly` recovers with `_ = r` ([mathml.go:207-211](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L207-L211) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L207-L211)) and continues with a half-built AST. At minimum log the panic + stack at warn level.
  </details>

- **[vendored TreeBlood sources carry no per-file MIT attribution]** [`gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD) — the MIT licence file ships, but none of the ~10 derived `.go` files name the upstream. Fix: add a one-line upstream + MIT notice to each derived file.
  <details><summary>details</summary>

  MIT requires the copyright + permission notice in "all copies or substantial portions of the Software". A single `LICENCE.MD` in the package dir is a common reading, but ~10k LOC of vendored, locally-modified code is exactly where a per-file header is worth the one line, and where the upstream delta (the PR description says TreeBlood was "updated and fixed for our gnoweb needs") should be recorded so future syncs have a baseline.
  </details>

- **[annotation half-escapes — only `<`, not `>`/`&`/`"`]** [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:176`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L176) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L176) — the `<annotation>` child stores the raw tex with only `<`→`&lt;`. Fix: use `html.EscapeString`, which the global serializer fix would supply anyway.
  <details><summary>details</summary>

  Both `wrapInMathTag` variants ([mathml.go:58](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L58) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L58), [mathml.go:176](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L176) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L176)) write the original tex into `<annotation>` escaping only `<`. Not an XSS breakout on its own (closing `</annotation>` needs a `<`, which is escaped), but `&` and `"` pass through and produce malformed markup that strict XML/AT/screen-reader tooling mishandles. Folds into the same escaping fix.
  </details>

## Nits

- [`gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:152-161`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L152-L161) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L152-L161) — attributes are sorted before output ([mmlnode.go:142](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L142) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L142)) but the `CSS` map is iterated unsorted, so a node with 2+ inline-style props serializes in random order. `\varliminf` produced two distinct `style="..."` orderings across 300 renders. Non-deterministic HTML; flaky golden tests. Sort the CSS keys too.
- [`gno.land/pkg/gnoweb/markdown/ext_math.go:117`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L117) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L117) and [`:122`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L122) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L122) — stray commented debug lines (`// fmt.Println(...)`, `// count := 0`). Delete.
- [`gno.land/pkg/gnoweb/markdown/ext_math.go:36-37`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L36-L37) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L36-L37) — `delimeter_ams`/`delimeter_tex` misspelling, should be `delimiter_*`.
- [`gno.land/pkg/gnoweb/markdown/mathml/environnements.go`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/environnements.go) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/environnements.go) — filename typo (French double-n), should be `environments.go`.
- [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:92-96`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L92-L96) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L92-L96) — `NewMathMLConverter(macros ...map[string]string)` accepts but discards `macros`, and `newCommand` ([commands.go:427-468](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L427-L468) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L427-L468)) parses then drops the definition, so `\newcommand`/`\def` never take effect. Either wire macros through or remove the parameter and advertise the limitation.
- [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:82`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L82) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L82) — `unknownCommandsAsOps` is read at [commands.go:354](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L354) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L354) but never set anywhere, so that branch is dead. Drop the field+branch or expose a way to enable it.
- [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:8-9`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L8-L9) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L8-L9) — `TexToMML`, `DisplayStyle`, `InlineStyle`, `SemanticsOnly`, `NewDocument` are exported but have no callers in or out of the package. Either make the renderer use the stateless pair (which also fixes the data race) or unexport the unused surface.

## Missing Tests

- **[no adversarial / XSS coverage]** [`gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go) — the 3146-line suite covers only well-formed LaTeX. After the escaping fix, add tests asserting `\text{</math>...}`, `\class{x" onclick=...}{y}`, `\textcolor`, `\raisebox`, and the error-fallback path (`$\begin{matrix}</span><script>...$`) never emit raw `<script>`, `onclick=`, or an unescaped `</math>`/`</span>`.
  <details><summary>details</summary>

  `gno.land/pkg/gnoweb/markdown/golden/ext_math/error_cases_test.txtar` is the natural home for the fallback cases. Pair each with the desired post-fix output (escaped), not the current buggy bytes.
  </details>

- **[no fuzz target, no concurrency test]** none — package has no `Fuzz*` and no `-race` test.
  <details><summary>details</summary>

  A `FuzzConvert` over random bytes into `ConvertInline` would quickly surface panics, hangs, and blow-ups in a ~140-command recursive parser fed user input. A table test that renders math from N goroutines through one shared converter, run under `-race`, would catch the shared-state race in CI (which otherwise runs no `-race`).
  </details>

## Suggestions

- [`gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:116-184`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L116-L184) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L116-L184) — make `MMLNode.Write` the single escaping chokepoint: `html.EscapeString` on `n.Text` and on each attribute value. One audit point, no risk of a future `SetAttr` site reintroducing the hole. Tags and attribute names are package-internal, so they don't need escaping but could be asserted against an XML-name pattern in debug builds.

## Open questions

- Is math meant to render arbitrary board/forum content, or only `r/docs`-style realms from trusted authors? It renders everything today; the answer sets how urgent the escaping is but not whether it's needed. Not posted — the fix is required regardless of the answer.
- Should the converter expose a streaming/one-line mode for gnoweb to avoid the O(N²) indentation entirely? Setting `PrintOneLine` drops the quadratic whitespace term, though a size cap is still needed. Not posted — a size cap is the load-bearing fix; this is an optimization.
