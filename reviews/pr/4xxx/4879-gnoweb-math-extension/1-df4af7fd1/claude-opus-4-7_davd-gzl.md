# PR #4879: feat(gnoweb): math extension

URL: https://github.com/gnolang/gno/pull/4879
Author: alexiscolin | Base: master | Files: 33 | +14927 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]
Local worktree: `git -C gno worktree add .worktrees/gno-review-4879 df4af7fd1` (then `gh -R gnolang/gno pr checkout 4879` inside it)

**Verdict: REQUEST CHANGES** — Stored XSS via LaTeX in any user-rendered markdown: `\text{}` writes attacker bytes straight into the page DOM and `\class{}` injects arbitrary HTML attributes (including `onclick=`). Output goes through the MathML renderer's raw `BufWriter.WriteString`, bypassing every escaping layer. Reachable from boards, forms, anything that renders user markdown. Must fix before merge.

## Summary

Adds a server-side LaTeX→MathML extension to gnoweb's markdown pipeline, vendored from [TreeBlood](https://github.com/Wyatt915/treeblood) (~10k LOC under `gno.land/pkg/gnoweb/markdown/mathml/`). New `$...$`, `$$...$$`, `\\(...\\)`, `\\[...\\]` delimiters detected by an inline + block goldmark parser; a custom `MathRenderer` calls `MathMLConverter.ConvertInline/Display(tex)` and writes the resulting MathML string directly to the page via `w.WriteString(mml)`. Tests are golden-file txtars under `gno.land/pkg/gnoweb/markdown/golden/ext_math/`.

The MathML serializer ([`mmlnode.go:116-184`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L116-L184) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L116-L184)) does zero HTML escaping on tag text, attribute values, or attribute names — and the rendering path bypasses goldmark's safe writer entirely. Two distinct primitives turn this into a confirmed XSS:

1. `\text{...}` ([`commands_defs.go:283-288`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L283-L288) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L283-L288)) puts attacker-controlled bytes into `<mtext>...</mtext>` verbatim. `</math>` inside the braces closes the math context, anything after is parsed as HTML.
2. `\class{...}{...}` ([`commands_defs.go:132-139`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L132-L139) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L132-L139)) writes attacker bytes into an attribute value via `n.SetAttr("class", StringifyTokens(...))` with no escaping. A `"` ends the attribute; the rest becomes new attributes.

End-to-end reproduction below.

## Glossary

- `MathRenderer` — goldmark `NodeRenderer` writing MathML directly to `util.BufWriter`.
- `MathMLConverter` — LaTeX→MathML AST builder (vendored from TreeBlood).
- `MMLNode.Write` — serializer that emits `<tag attr="val">...</tag>` with no escaping.
- `stringifyTokensHtml` — token-to-string used for `<mtext>` body (only swaps space → `&nbsp;`).
- `StringifyTokens` — token-to-string used for attribute values like `class`, `mathcolor`, `voffset`, `rowspan`.

## Fix

The PR adds a brand-new extension; before-state has no math support, after-state renders `$...$`/`$$...$$` blocks. The load-bearing constraint missed by the implementation is that the MathML output is interpolated into a user-facing HTML page and any LaTeX command that ends up in `node.Text` or `node.Attrib[key]` must be HTML-escaped before serialization — `mmlnode.go:Write` currently does `w.WriteString(val)` for attributes and `w.WriteString(n.Text)` for content with no `html.EscapeString` or equivalent. Fixing this is mechanical (escape in `Write`) but every reachable command needs to be revisited because TreeBlood was built for static-site authoring, not adversarial input.

## Critical (must fix)

- **[stored XSS via LaTeX text — script execution from any rendered markdown]** [`gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go:283-288`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L283-L288) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L283-L288) — `\text{</math><script>alert(1)</script>}` lands verbatim in the page DOM.
  <details><summary>details</summary>

  `cmd_text` does `NewMMLNode("mtext", stringifyTokensHtml(args[0].Expr))` ([commands_defs.go:287](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L287) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L287)). `stringifyTokensHtml` ([`tokenize.go:462-472`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L462-L472) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L462-L472)) only swaps `" "` → `&nbsp;`; it does not escape `<`, `>`, `&`. `MMLNode.Write` ([`mmlnode.go:163-166`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L163-L166) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L163-L166)) then writes `n.Text` raw via `w.WriteString`. End-to-end run on `df4af7fd1`:

  ```
  Display: $$\text{</math><script>alert('XSS')</script>}$$
  →
  <math ...><semantics><mrow><mtext></math><script>alert('XSS')</script></mtext></mrow>...
  ```

  The browser HTML parser closes the `math` element at the in-content `</math>` and then sees a live `<script>` tag. Reachable from anywhere gnoweb renders user-controlled markdown — boards, forms, the `r/docs/markdown` realm this PR ships, etc. Severity is critical: attacker hosts a malicious realm, victim opens it through gnoweb, script runs in the gnoweb origin (steals session, signs txs via injected forms, etc.). **Fix:** escape `n.Text` in `MMLNode.Write` with `html.EscapeString` (or its `<>&"'` subset). Confirm by running the repro below; the `<script>` should appear as `&lt;script&gt;` in the output.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 4879 -R gnolang/gno
  cat > /tmp/xss.go <<'EOF'
  package main

  import (
  	"bytes"
  	"fmt"
  	"github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown"
  	"github.com/yuin/goldmark"
  )

  func main() {
  	m := goldmark.New(goldmark.WithExtensions(markdown.NewGnoExtension()))
  	var buf bytes.Buffer
  	_ = m.Convert([]byte(`Display: $$\text{</math><script>alert('XSS')</script>}$$`), &buf)
  	fmt.Println(buf.String())
  }
  EOF
  go run /tmp/xss.go
  rm /tmp/xss.go
  ```
  Look for `<script>alert('XSS')</script>` in the output — it appears verbatim, unescaped.
  </details>

- **[stored XSS via attribute injection — `\class{}` writes arbitrary attributes]** [`gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go:132-139`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L132-L139) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L132-L139) — `\class{x" onclick="alert(1)}{y}` produces `<mi class="x" onclick="alert(1)">y</mi>`.
  <details><summary>details</summary>

  `cmd_class` calls `n.SetAttr("class", StringifyTokens(args[0].Expr))` ([commands_defs.go:137](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L137) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L137)). `StringifyTokens` ([`tokenize.go:455-461`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L455-L461) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L455-L461)) concatenates token values with no escaping. `MMLNode.Write` ([`mmlnode.go:144-151`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L144-L151) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L144-L151)) emits `key="val"` with `w.WriteString(val)` — quotes in `val` are not escaped. Same gadget exists in `cmd_textcolor` (`mathcolor`), `cmd_raisebox` (`voffset`), `cmd_mathop` (`mo` text), `cmd_multirow` (`rowspan`/`columnspan`). Confirmed end-to-end:

  ```
  $\class{x" onclick="alert(1)}{y}$
  →
  <mi class="x" onclick="alert(1)">y</mi>
  ```

  This is simpler than the `\text{}` path because the attacker doesn't even need to break out of an element — `onclick` (or `onmouseover`, `onfocus`, etc.) on any rendered MathML node fires on user interaction. **Fix:** escape attribute values in `MMLNode.Write` (at minimum `"` and `&`); ideally use `html.EscapeString` on every attribute write. Also audit every `SetAttr` caller for an attacker-controlled value (grep shows ~15 sites — `class`, `mathcolor`, `rowspan`, `columnspan`, `voffset`, `title`, `href` if added later).
  </details>

- **[fallback path writes raw `tex` unescaped]** [`gno.land/pkg/gnoweb/markdown/ext_math.go:294-304`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L294-L304) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L294-L304) — error fallback emits the original LaTeX directly inside `<span>`/`<div>`.
  <details><summary>details</summary>

  When `ConvertInline/Display` returns a non-nil error, the renderer writes `<span class="math-inline">` + raw `tex` + `</span>` (or the `<div>` variant). `tex` is fully attacker-controlled at this point — the same `</span><script>...</script>` payload escapes the wrapper. Empirically the converter rarely returns a non-nil error (it recovers panics and returns a wrapped MathML buffer instead), but the path exists and would silently flip to "exploitable" if conversion error handling is changed later. **Fix:** wrap `tex` in `html.EscapeString` before writing in both fallback branches.
  </details>

## Warnings (should fix)

- **[delimiter contract mismatches PR description]** [`gno.land/pkg/gnoweb/markdown/ext_math.go:41-44`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L41-L44) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L41-L44) — `_inlineopen` is the 3-byte sequence `\\(` (two backslashes + paren), not the standard LaTeX `\(`.
  <details><summary>details</summary>

  The Go raw-string literal `` `\\(` `` is literally three bytes (`\`, `\`, `(`). Users typing `\(E=mc^2\)` get the unprocessed text `(E=mc^2)`; only `\\(E=mc^2\\)` (two backslashes on each side) is recognized. Same for `\\[...\\]`. This contradicts the PR description ("`\(...\)` and `\[...\]` support") and contradicts universal LaTeX/KaTeX/MathJax convention, so existing math copied from other tools will silently fail to render. Either change the literals to `[]byte{'\\', '('}` etc., or restate the contract in the PR description and the docs realm. The `r/docs/markdown` example already side-steps this by only documenting `$...$` and `$$...$$`. Confirmed by running the inputs `\(E=mc^2\)` vs `\\(E=mc^2\\)` through the pipeline.
  </details>

- **[`recover()` masks real bugs, swallows panics into "invalid math" UI]** [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:12-27, 105-121`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L12-L121) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L12-L121) — every panic is converted to a generic `<merror>` with no log.
  <details><summary>details</summary>

  `TexToMML` and `MathMLConverter.render` both wrap the entire conversion in `defer recover()` ([mathml.go:12](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L12) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L12), [mathml.go:105](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L105) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L105)) and on any panic return a "MathML encountered an unexpected error" error with no stack, no log, no metric. Useful for keeping the page rendering, but operationally blind: a malformed input that crashes the parser will silently degrade for every viewer with no signal to operators that something is wrong. At minimum, log the panic + stack at warn level so the operator can find the crashing inputs. The `SemanticsOnly` recover at [mathml.go:207-211](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L207-L211) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L207-L211) is worse — it discards the panic with `_ = r` and continues with a half-built AST.
  </details>

- **[per-file MIT attribution missing on TreeBlood-derived sources]** [`gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/LICENCE.MD) vs `*.go` headers — license file ships, but none of the 10 `.go` files carry the MIT notice.
  <details><summary>details</summary>

  TreeBlood is MIT, and the MIT terms require "the above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software". Shipping one `LICENCE.MD` next to the package code is the common interpretation, but the more conservative practice (and what gno's own `tm2/` vendored packages do — e.g. `tm2/pkg/db/` files cite their source) is to put a one-line attribution at the top of each derived `.go` file. Cost: a one-line comment per file. Worth doing for compliance hygiene given this is ~10k LOC of vendored code.
  </details>

- **[output amplification: ~21× expansion per byte of math input]** [`gno.land/pkg/gnoweb/markdown/mathml/`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/) — 800KB of LaTeX produces 17MB of HTML in 0.5s with no per-render bound.
  <details><summary>details</summary>

  Benchmarked on `df4af7fd1`: a single `$x_1+x_2+...$` with 800KB of expression rendered to 17MB of MathML in ~514ms; a 200×200 `\begin{matrix}` (160KB input) rendered to 2.4MB in ~131ms. Realm renderers can already produce large markdown, but math amplifies per-byte significantly (each digit/letter becomes a `<mi>...</mi>` or `<mn>...</mn>` element). For comparison, KaTeX's server-side renderer caps input length and node count to bound this. **Fix:** cap the rendered MML byte count (or the tex input length) per math block — log a clear error past the cap. Even a 100KB output cap would let normal pages render fine and prevent a tiny realm from rendering to a multi-MB response per page view.
  </details>

- **[`annotation` only escapes `<`, not `>`/`&`/`"`]** [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:58, 176`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L58) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L58) — `annotation := NewMMLNode("annotation", strings.ReplaceAll(tex, "<", "&lt;"))`.
  <details><summary>details</summary>

  Both `wrapInMathTag` functions write the original tex into a `<annotation>` child and only escape `<`. `>` and `&` are not entities-escaped, so an annotation with `&` in it produces malformed XML/HTML (browsers handle it leniently but RSS/AT/screen-reader tooling may not). Not directly exploitable for XSS since the parent path is `<annotation encoding="application/x-tex">...</annotation>` and `</annotation>` would itself need `<`, but the half-escaping is a code smell — switch to `html.EscapeString` to be consistent. Same fix applies once the global `MMLNode.Write` escaping lands.
  </details>

## Nits

- [`gno.land/pkg/gnoweb/markdown/ext_math.go:117`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L117) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L117) — Stray commented `// fmt.Println(string(line))` debug print. Delete.
- [`gno.land/pkg/gnoweb/markdown/ext_math.go:122`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L122) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L122) — Stray commented `// count := 0` next to a comment about linebreaks. Delete.
- [`gno.land/pkg/gnoweb/markdown/mathml/environnements.go`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/environnements.go) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/environnements.go) — filename typo: `environnements` (French double-n) → `environments`.
- [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:65-72`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L65-L72) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L65-L72) — `DisplayStyle`/`InlineStyle` package-level helpers are unused inside the gnoweb pipeline (the renderer uses `MathMLConverter.ConvertInline/Display`). Either drop them or document them as the public API.
- [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:86-90`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L86-L90) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L86-L90) — `NewDocument(macros, doNumbering)` calls `NewMathMLConverter(macros)` but `NewMathMLConverter` discards `macros...` (no `_ = macros`, no use). Macros are never wired through — `cmd_text`/`cmd_class`/etc. will not see user-defined macros. Either implement macro support or drop the parameter.
- [`gno.land/pkg/gnoweb/markdown/ext_math.go:34-38`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/ext_math.go#L34-L38) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L34-L38) — `delimeter_ams`/`delimeter_tex` misspelling — should be `delimiter_*`. Pervasive across the file.
- [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:80`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L80) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L80) — `unknownCommandsAsOps` field is declared on `MathMLConverter` but never read or written anywhere in the package. Dead field.

## Missing Tests

- **[XSS coverage]** [`gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go) — no test asserts that adversarial LaTeX cannot produce raw `<script>`, `onclick=`, `javascript:`, or close the math element prematurely.
  <details><summary>details</summary>

  The 3146-line test file covers a wide range of valid LaTeX → MathML cases but has zero adversarial inputs. After the escaping fix lands, add a focused test that asserts each of `\text{</math>...}`, `\class{x" onclick="...}{y}`, `\textcolor{x"}{y}`, `\raisebox{x"}{y}`, `\href{javascript:...}{y}` produces output where `<script>`, `onclick`, and unescaped `</math>` are absent (or properly escaped) — and that the escaping is invariant to a fuzz corpus of random byte sequences inside `\text{...}`. The golden-file txtars (`gno.land/pkg/gnoweb/markdown/golden/ext_math/error_cases_test.txtar` is the natural home) currently test only well-formed inputs.
  </details>

- **[fuzz target on `MathMLConverter.ConvertInline`]** none — package has no `Fuzz*` function.
  <details><summary>details</summary>

  Given the size of the parser (3 token kinds, ~140 commands, recursive environments), `go test -fuzz=FuzzConvert` over random byte input would catch panics, infinite loops, and memory blow-ups quickly. Mandatory in a package that processes user-controlled input for an HTML page.
  </details>

## Suggestions

- [`gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:116-184`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L116-L184) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L116-L184) — Single-point fix: centralize escaping in `MMLNode.Write`. One audit point, no per-command discipline required, no risk of forgetting an escape at a new `SetAttr` site.
  <details><summary>details</summary>

  Concretely: change line 149 to `w.WriteString(html.EscapeString(val))` and line 165 to `w.WriteString(html.EscapeString(n.Text))`. Then audit `Tag` and `Attrib` key writes (these come from the package itself, not user input, so they don't need escaping but should be guarded with a `validXMLName` regex assert in debug builds). This is the smallest defensible diff that closes both XSS gadgets.
  </details>

- [`gno.land/pkg/gnoweb/markdown/mathml/mathml.go:65-72`](https://github.com/gnolang/gno/blob/df4af7fd1/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L65-L72) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L65-L72) — Decide whether `TexToMML`/`DisplayStyle`/`InlineStyle` are part of the package's public API. If yes, document. If no, unexport (lower-case). Right now they're exported, untested at the package surface level, and duplicate the `MathMLConverter` methods.

## Questions for Author

- Is the math extension expected to render text from boards/forums/forms, or only from `r/docs` realms authored by trusted parties? The blast radius of the XSS in the first case is much larger.
- Why the `\\(...\\)` (two-backslash) delimiter form? Is this intentional vs. standard LaTeX `\(...\)`?
- TreeBlood upstream has been "updated and fixed for our gnoweb needs" (per PR description) — is the diff against upstream documented somewhere? Future merges from upstream will need a clear delta.
- `NewMathMLConverter(macros ...map[string]string)` accepts but ignores `macros`. Was macro support deliberately dropped, or should `needMacroExpansion` be populated from the first arg?
