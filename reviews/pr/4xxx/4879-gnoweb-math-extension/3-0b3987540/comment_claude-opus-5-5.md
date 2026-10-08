# Review: [#4879](https://github.com/gnolang/gno/pull/4879)
Posted: https://github.com/gnolang/gno/pull/4879#pullrequestreview-5454217425
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Two Criticals block it: a `\color` whose `{` closes outside its environment hangs gnoweb, and each pair of openers whose closer sits outside their group multiplies parse time by 4 to 5. 33 Warnings on rendering, empty arguments and the symbol table also reach users.
Model: claude-opus-5-5, standard review
Commit: 0b3987540 (latest)
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-4879 0b3987540`
Round: 3. 12 finders, one reflector, 69 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted.

## Body

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:282 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L282) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L282) · Critical
Critical: `b.GetNextN(i - b.idx)` discards its error, so a `\color` whose `{` closes outside its environment is re-read forever and hangs gnoweb.

<details><summary>repro</summary>

The subtest never returns: [`GetNextN`'s error branch](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L415-L416) leaves `b.idx` in place and [`b.Unget()`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L292) rewinds onto `\color`.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b4-lines-reach-rollout-catalog-switch-scope-hang_test.go gno.land/pkg/gnoweb/markdown/zz_switch_scope_hang_test.go
(cd gno.land/pkg/gnoweb/markdown && timeout 60 go test -run 'TestSwitchScopeCrossingBrace/env-color' -v -count=1 .); echo exit=$?
rm gno.land/pkg/gnoweb/markdown/zz_switch_scope_hang_test.go
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:416 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L416) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L416) · Critical
Critical: past the buffer's end, `GetNextN` returns the rest of it and leaves `b.idx` in place, so each `\left(` or `\begin` closing outside its `{group}` doubles the parse work.

A few hundred bytes of realm output then stall gnoweb or run it out of memory.

<details><summary>repro</summary>

Time grows about x4-5 per two openers: [`parse.go:210`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L210) and [`parse.go:233`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L233) call `GetNextN(tok.MatchOffset)` past the end of the group's buffer.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b8-lines-reach-catalog-cross-group-blowup_test.go gno.land/pkg/gnoweb/markdown/zz_cross_group_blowup_test.go
go test -run TestCrossGroupOpenerBlowup -v -count=1 ./gno.land/pkg/gnoweb/markdown
rm gno.land/pkg/gnoweb/markdown/zz_cross_group_blowup_test.go
```
</details>

## gno.land/pkg/gnoweb/frontend/css/05-composition.css:521 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/frontend/css/05-composition.css#L521) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/frontend/css/05-composition.css#L521) · Warning
Centred `mtd` cells get only `text-align`, and the [`* { padding: 0 }` reset](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/frontend/css/03-generic.css#L18) removes the browser's cell padding, so matrix columns touch and `\begin{bmatrix} 10 & 20 \end{bmatrix}` reads as `[1020]`.

An `array{rl}` gets no gap either, since the side padding only covers the `aligned` layout.

<details><summary>repro</summary>

Open `tests/b1-lines-removed-claims-reach-catalog-array-rl-gap.html` from the clone root in headless Chromium with `--virtual-time-budget=3000 --dump-dom`. Adjacent numeric cells measure 0 px apart. With the `main.css` `<link>` removed, the browser default padding separates them.
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:333 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L333) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L333) · Warning
`findDollarClose` accepts the `$` inside gnoweb's own `/r/x$help&func=` link as a closer.

One unpaired `$` earlier on the line, such as `[Send $10]`, then turns the realm's tx link into math glyphs.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB1UserDollarEatsRealmLink ./gno.land/pkg/gnoweb/
```
with `tests/judge-2-dollar-eats-link_test.go` copied in. The rendered HTML for a line holding `$5` and then a `$help` link has no `<a>` for that link; its label and URL prefix come out as `<mi>`/`<mo>`.
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:364-366 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L364-L366) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L364) · Warning
The `bytes.HasPrefix` opener checks skip no leading spaces, so a `$$` or `\[` opener indented by one to three spaces renders as a literal paragraph.

This hits a `$$` block in a list item, a footnote or a quote, where the closer and an indented math fence accept the same indentation.

<details><summary>repro</summary>

```
curl -fsSL -o gno.land/pkg/gnoweb/markdown/judge-5-indented-display-opener_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/judge-5-indented-display-opener_test.go
go test -count=1 -v -run TestIndentedDisplayOpener ./gno.land/pkg/gnoweb/markdown/
```
The five indented-opener cases report `math=false`; the unindented controls report `math=true`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:64 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L64) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L64) · Warning
`"boldsymbol": ctxVarBold` gives `\boldsymbol` the context of [`\mathbf`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L63), so `\boldsymbol{x}` renders bold upright. MathJax and KaTeX keep the bold italic that authors pick it for.

<details><summary>repro</summary>

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b12-lines-reach-catalog-boldsymbol-italic_test.go gno.land/pkg/gnoweb/markdown/mathml/zz_boldsymbol_italic_test.go
go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestBoldsymbolIsBoldItalic -v -count=1
rm gno.land/pkg/gnoweb/markdown/mathml/zz_boldsymbol_italic_test.go
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:219 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L219) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L219) · Warning
`NewMMLNode("mi")` carries `movablelimits`, which MathML Core honours only on `<mo>`.

Inline `\lim_{x \to 0}`, `\max` and `\sup` therefore set their limit underneath, where TeX sets a subscript.

<details><summary>repro</summary>

Open in Chrome: `inline_lim.mi_movablelimits.script_below_base` reads `true`, while the `mo_movablelimits` control reads `script_right_of_base: true`.

```bash
xdg-open reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b3-lines-reach-catalog-mathml-layout.html
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:222 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L222) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L222) · Warning
`lspace` on an `<mi>` draws no gap, since MathML Core defines it for `<mo>` only, so `$a\sin x$` reads `asinx` and `\sin \cos \tan` runs together.

<details><summary>repro</summary>

Open in Chrome: `lspace_gap_a_to_sin.mi_with_lspace` equals `mi_without` (0px), while the `mo` control is 2.44px.

```bash
xdg-open reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b3-lines-reach-catalog-mathml-layout.html
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:249 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L249) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L249) · Warning
`context|variant` ORs a nested font into the outer one, and the [`set_variants_from_context` switch](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/variant_transform.go#L29-L56) has no case or default for the pair, so both fonts are dropped.

`\boldsymbol{\mathbb{R}}` and `{\bf \mathbb{R}}` render a plain R, and `\mathbf{\mathfrak{g}}` a bare italic g.

<details><summary>repro</summary>

`\mathbb{\mathbf{R}}`, `\mathtt{\mathbf{x}}` and `\mathbf{\mathrm{x}}` lose both fonts the same way, and `\mathcal{L}` alone keeps its font.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b4-lines-reach-rollout-catalog-variant-or-combines_test.go gno.land/pkg/gnoweb/markdown/mathml/zz_variant_or_combines_test.go
go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestInnerFontSurvivesOuterFont -v -count=1
rm gno.land/pkg/gnoweb/markdown/mathml/zz_variant_or_combines_test.go
```

Every nested case of this second test fails with a bare `<mi>g</mi>`, `<mi>A</mi>`, `<mi>R</mi>` or `<mi>x</mi>`:

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_variant_nested_fonts_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestNestedFontCommandsKeepAVariant -v
package mathml

import (
	"regexp"
	"testing"
)

func TestNestedFontCommandsKeepAVariant(t *testing.T) {
	bare := regexp.MustCompile(`<mi>[A-Za-z]</mi>`)
	for _, tex := range []string{
		`\mathbf{\mathfrak{g}}`,
		`\mathbf{\mathcal{A}}`,
		`\mathbb{\mathbf{R}}`,
		`\mathtt{\mathbf{x}}`,
		`\mathrm{\mathbf{x}}`,
		`\mathbf{\mathrm{x}}`,
	} {
		out, err := NewMathMLConverter().ConvertInline(tex)
		if err != nil {
			t.Fatalf("%s: %v", tex, err)
		}
		if bare.MatchString(out) {
			t.Errorf("%s: both variants dropped, got %s", tex, bare.FindString(out))
		}
	}
}
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:274 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L274) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L274) · Warning
The scan for a style switch's scope jumps only over `tokCurly|tokOpen` groups, so it stops at the `&` inside a following environment.

`\displaystyle\begin{pmatrix}a&b\end{pmatrix}` shows a literal `&` and puts the later cells outside the table, with no error.

<details><summary>repro</summary>

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b4-lines-reach-rollout-catalog-switch-splits-environment_test.go gno.land/pkg/gnoweb/markdown/zz_switch_splits_env_test.go
go test ./gno.land/pkg/gnoweb/markdown/ -run TestSwitchBeforeEnvironmentKeepsCells -v -count=1
rm gno.land/pkg/gnoweb/markdown/zz_switch_splits_env_test.go
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:330 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L330) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L330) · Warning
`base.Tag` dereferences the nil that `ParseTex` returns for an empty accent argument.

`\hat{}`, `\overline{}` or a trailing `\hat` panics, and the whole formula falls back to raw LaTeX.

<details><summary>repro</summary>

The panic is recovered in [`mathml.go:42-48`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L42-L48) as `MathML encountered an unexpected error` for each accent.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b4-lines-reach-rollout-catalog-accent-empty-arg_test.go gno.land/pkg/gnoweb/markdown/zz_accent_empty_arg_test.go
go test ./gno.land/pkg/gnoweb/markdown/ -run TestAccentWithEmptyArgumentStillRenders -v -count=1
rm gno.land/pkg/gnoweb/markdown/zz_accent_empty_arg_test.go
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:389 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L389) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L389) · Warning
`context |= ctxVarNormal` lands on top of the outer font's bit, and `set_variants_from_context` has no case for that pair.

`\mathbf{\Gamma}` and `\boldsymbol{\Omega}` render a plain `<mi>Γ</mi>`, neither bold nor upright.

<details><summary>repro</summary>

Expect 𝚪, 𝛀 and 𝛁; the output is `<mi>Γ</mi>`, `<mi>Ω</mi>` and `<mi>∇</mi>`.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b12-lines-reach-catalog-variant-upright-symbol_test.go gno.land/pkg/gnoweb/markdown/mathml/zz_variant_upright_symbol_test.go
go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestVariantOnUprightSymbol -v -count=1
rm gno.land/pkg/gnoweb/markdown/mathml/zz_variant_upright_symbol_test.go
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go:72 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L72) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L72) · Warning
`safeRaise` checks the 2em bound on raw TeX lengths and ignores the enclosing size switch.

`\Huge\raisebox{2em}{x}` passes and moves 4.98em, so inline math can cover the page text above it.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB5RaiseUnderSize ./gno.land/pkg/gnoweb/markdown/mathml/
```
with `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` as `zz_b5_test.go`: both inputs convert with err nil. `tests/judge-22-raise-under-huge.html` in headless Chromium measures the 4.98em and 7.95em shifts. That page does not load gnoweb's stylesheet.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go:96 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L96) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L96) · Warning
`n.SetAttr` runs on the nil that `ParseTex` returns for an empty cell.

A blank spanning cell such as `\multicolumn{2}{c}{}` panics, and the page shows the whole table as raw TeX.

<details><summary>repro</summary>

```
cd gno.land && go test ./pkg/gnoweb/markdown/mathml/ -run TestB6EmptyArgumentConverts -v
```
with `tests/b6-lines-reach-catalog-empty-args_test.go`: both array inputs return `MathML encountered an unexpected error`. See also `tests/judge-21-multicell-empty-arg_test.go`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go:230 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L230) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L230) · Warning
`base.Tag` reads the nil that `ParseTex` returns for an empty brace group, so `\overset{a}{}` and `\underset{a}{}` panic.

The formula is then shown as TeX source, and `\substack{}` fails the same way.

<details><summary>repro</summary>

```
cd gno.land && go test ./pkg/gnoweb/markdown/mathml/ -run TestB6EmptyArgumentConverts -v
```
with `tests/b6-lines-reach-catalog-empty-args_test.go`: `\overset{}{}` and `\underset{}{}` return `MathML encountered an unexpected error`. See also `tests/judge-25-empty-base-nil-deref_test.go`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go:292 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L292) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L292) · Warning
`StringifyTokens(args[0].Expr)` joins the argument's token text instead of parsing it.

`\mathop{\mathrm{argmax}}_x f` renders `<mo>mathrm{argmax}</mo>`, and `\mathop{\sum}_i` renders the word `sum`.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB5MathopSource ./gno.land/pkg/gnoweb/markdown/mathml/
```
with `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` as `zz_b5_test.go`. See also `tests/judge-20-mathop-source-text_test.go`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go:316 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L316) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L316) · Warning
`converter.ParseTex(args[0], ctx)` reads an argument that TeX's `\bmod` does not take.

In `a \bmod \frac{p}{q}` it takes `\frac` alone, so the page shows an error box and stray `p` and `q`.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB5BmodTakesArg ./gno.land/pkg/gnoweb/markdown/mathml/
```
with `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` as `zz_b5_test.go`. See also `tests/judge-23-24-star-matrix-bmod_test.go`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/environnements.go:21 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L21) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L21) · Warning
A starred matrix reads its column alignment only from a brace group.

The mathtools spelling `\begin{pmatrix*}[r]` shows a literal `[r]` in its first cell and stays centred, with no error.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB5StarMatrixBracketOption ./gno.land/pkg/gnoweb/markdown/mathml/
```
with `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` as `zz_b5_test.go`. See also `tests/judge-23-24-star-matrix-bmod_test.go`.
</details>

## SKIP gno.land/pkg/gnoweb/markdown/mathml/environnements.go:127 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L127) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L127) · Warning
`processTable` treats a single-node body as the table and splits that node's children into cells.

`\begin{pmatrix} \frac{1}{2} \end{pmatrix}` renders `(1 2)`, with no fraction bar and no error.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB5SingleNodeEnv ./gno.land/pkg/gnoweb/markdown/mathml/
```
with `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` as `zz_b5_test.go`. Chromium draws 1 and 2 side by side at the same height. See also `tests/judge-19-env-single-node-body_test.go`.
</details>

Not posted: the `parse.go:211` section carries this case and its repro, and one fix closes both.

## gno.land/pkg/gnoweb/markdown/mathml/mathml.go:11 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L11) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L11) · Warning
`MaxParseDepth` counts `ParseTex` frames, and each unnested `\color` or `\bf` switch recurses for the rest of its group.

64 sequential `\bf x+` units fail to convert at nesting depth 1.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_judge45_test.go
// go test -run 'TestJudge45' -v -count=1 ./gno.land/pkg/gnoweb/markdown/mathml
package mathml

import (
	"strings"
	"testing"
)

func TestJudge45(t *testing.T) {
	for _, unit := range []string{`\color{red}x+`, `\bf x+`, `{\color{red}x}+`, `\mathbf{x}+`} {
		first := -1
		for n := 1; n <= 100; n++ {
			tex := strings.Repeat(unit, n) + "c"
			if _, err := NewMathMLConverter().ConvertDisplay(tex); err != nil {
				first = n
				t.Logf("%-18q first fails at n=%d len=%d: %v", unit, n, len(tex), err)
				break
			}
		}
		if first >= 0 {
			t.Errorf("%q x%d fails though nesting depth is 1", unit, first)
		}
	}
}
```

This fails at n=64 on the two switch units, and the grouped controls convert up to n=100.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:199 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L199) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L199) · Warning
`Write` skips a nil child, so an `mfrac` or `mroot` built from an empty argument is written one child short.

`\frac{}{b}` then reads as `b`, with no bar and no error.

<details><summary>repro</summary>

```
cd gno.land && go test ./pkg/gnoweb/markdown/mathml/ -run TestB6FixedArityChildren -v
```
with `tests/b6-lines-reach-catalog-empty-args_test.go` shows the missing child. See also `tests/judge-29-fixed-arity-nil-child_test.go` and `tests/judge-29-fixed-arity-layout.html`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/parse.go:137 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L137) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L137) · Warning
`if temp != nil` drops the script bit of an empty script group, whose nil `ParseTex` result is still appended as a sibling.

The next script then detaches from its base, so `\sum_{}^{n} k` loses its upper limit.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b7_emptyscript_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestB7EmptyScriptGroup -v
package mathml

import (
	"regexp"
	"strings"
	"testing"
)

func TestB7EmptyScriptGroup(t *testing.T) {
	for _, tex := range []string{`\sum_{}^{n} k`, `\int_{}^{1} f`, `x^{}_2`} {
		out, err := NewMathMLConverter().ConvertInline(tex)
		out = regexp.MustCompile(`(?s)<annotation.*</annotation>`).ReplaceAllString(out, "")
		if err != nil || strings.Contains(out, "<none></none>") {
			t.Errorf("%q: script detached from its base: %s", tex, out)
		}
	}
}
```
</details>

## SKIP gno.land/pkg/gnoweb/markdown/mathml/parse.go:210 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L210) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L210) · Warning
`GetNextN` advances by `MatchOffset` only after skipping leading comments.

A `%` comment right after `\begin{..}` therefore slides the window one token past `\end`, and a trailing `.` lands in the last cell.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b7_envcomment_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestB7EnvCommentShift -v
package mathml

import (
	"regexp"
	"testing"
)

func render(tex string) (string, error) {
	out, err := NewMathMLConverter().ConvertInline(tex)
	return regexp.MustCompile(`(?s)<annotation.*</annotation>`).ReplaceAllString(out, ""), err
}

func TestB7EnvCommentShift(t *testing.T) {
	pairs := [][2]string{
		{"\\begin{pmatrix}%c\na & b\\end{pmatrix}.", "\\begin{pmatrix}a & b\\end{pmatrix}."},
		{"\\left( \\begin{matrix}%c\na & b\\end{matrix}\\right)", "\\left( \\begin{matrix}a & b\\end{matrix}\\right)"},
		{"\\begin{matrix}%c\na\\end{matrix}", "\\begin{matrix}a\\end{matrix}"},
	}
	for _, p := range pairs {
		got, err1 := render(p[0])
		want, err2 := render(p[1])
		if err1 != nil || err2 != nil || got != want {
			t.Errorf("a %% comment changed the rendering of %q\n  with comment: %s\n  without:      %s", p[1], got, want)
		}
	}
}
```
</details>

Not posted: the `tokenize.go:418` section carries this case and its repro, and one fix closes both.

## gno.land/pkg/gnoweb/markdown/mathml/parse.go:211 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L211) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L211) · Warning
`processEnv` gets a single-node body as the node itself, since [`return siblings[0]`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L325) hands back a lone sibling unwrapped, and [`processTable`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L127) splits that node's children into cells.

`\begin{matrix} \frac{a}{b} \end{matrix}` renders as `a b` with no error, and `\begin{array}{|c|} x \end{array}` loses its column spec and its `mtd`.

<details><summary>repro</summary>

`\begin{pmatrix} \frac{1}{2} \end{pmatrix}` renders `(1 2)` the same way, with no fraction bar. Only the multi-sibling path sets `Option`, and a body of two cells or more keeps its structure.

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_env_single_node_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestEnvSingleNodeBody -count=1 -v
package mathml

import (
	"strings"
	"testing"
)

func TestEnvSingleNodeBody(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`\begin{matrix} \frac{a}{b} \end{matrix}`, "<mfrac>"},
		{`\begin{pmatrix} \frac{a}{b} \end{pmatrix}`, "<mfrac>"},
		{`\begin{aligned} \sqrt{2} \end{aligned}`, "<msqrt>"},
		// control: a second cell makes the body an mrow, and the fraction survives
		{`\begin{matrix} \frac{a}{b} & 1 \end{matrix}`, "<mfrac>"},
	} {
		out, err := NewMathMLConverter().ConvertInline(tc.in)
		if err != nil {
			t.Errorf("%s: err %v", tc.in, err)
			continue
		}
		body := out[:strings.Index(out, "<annotation")]
		if !strings.Contains(body, tc.want) {
			t.Errorf("%s: want %s in output, got %s", tc.in, tc.want, body)
		}
	}
}
```

The three single-node bodies fail, and the two-cell control passes.

The one-cell `array` loses its column spec in this second test's logged rows, and the two-cell controls keep it:

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_j33_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestJudge33EnvSingle -v
package mathml

import (
	"strings"
	"testing"
)

func TestJudge33EnvSingle(t *testing.T) {
	for _, tex := range []string{`\begin{pmatrix}\frac{a}{b}\end{pmatrix}`, `\begin{pmatrix}\frac{a}{b} & c\end{pmatrix}`, `\begin{pmatrix}{\frac{a}{b}}\end{pmatrix}`, `\begin{array}{|c|} x \end{array}`, `\begin{array}{|c|} x & y \end{array}`} {
		out, err := NewMathMLConverter().ConvertInline(tex)
		i := strings.Index(out, "<semantics>")
		j := strings.Index(out, "<annotation")
		if i >= 0 && j > i {
			out = out[i:j]
		}
		t.Logf("%-45s err=%v %s", tex, err, out)
	}
	out, _ := NewMathMLConverter().ConvertInline(`\begin{pmatrix}\frac{a}{b}\end{pmatrix}`)
	if !strings.Contains(out, "<mfrac>") {
		t.Errorf("single-cell pmatrix lost its fraction")
	}
}
```

`TestB5SingleNodeEnv` in `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` fails the same way on `\begin{pmatrix} \frac{1}{2} \end{pmatrix}` and `\begin{array}{c} \sqrt{x} \end{array}`. Chromium draws 1 and 2 side by side at the same height. See also `tests/judge-19-env-single-node-body_test.go`.
</details>

## SKIP gno.land/pkg/gnoweb/markdown/mathml/parse.go:325 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L325) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L325) · Warning
`return siblings[0]` hands back a lone sibling unwrapped, and only the multi-sibling path sets `Option`.

A one-cell `array` loses its column spec, and `\begin{array}{|c|} x \end{array}` has no `mtd`.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_j33_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestJudge33EnvSingle -v
package mathml

import (
	"strings"
	"testing"
)

func TestJudge33EnvSingle(t *testing.T) {
	for _, tex := range []string{`\begin{pmatrix}\frac{a}{b}\end{pmatrix}`, `\begin{pmatrix}\frac{a}{b} & c\end{pmatrix}`, `\begin{pmatrix}{\frac{a}{b}}\end{pmatrix}`, `\begin{array}{|c|} x \end{array}`, `\begin{array}{|c|} x & y \end{array}`} {
		out, err := NewMathMLConverter().ConvertInline(tex)
		i := strings.Index(out, "<semantics>")
		j := strings.Index(out, "<annotation")
		if i >= 0 && j > i {
			out = out[i:j]
		}
		t.Logf("%-45s err=%v %s", tex, err, out)
	}
	out, _ := NewMathMLConverter().ConvertInline(`\begin{pmatrix}\frac{a}{b}\end{pmatrix}`)
	if !strings.Contains(out, "<mfrac>") {
		t.Errorf("single-cell pmatrix lost its fraction")
	}
}
```

The logged rows show that the one-cell bodies lose their structure and the two-cell controls keep it.
</details>

Not posted: the `parse.go:211` section carries this case and its repro, and one fix closes both.

## gno.land/pkg/gnoweb/markdown/mathml/parse.go:340 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L340) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L340) · Warning
`postProcessInfix` runs before scripts are attached and pairs one node on each side.

`x^2 \over 2` renders as x times 2/2, and `1 \over \int\limits_0^1 f` panics.

<details><summary>repro</summary>

```
curl -fsSL -o gno.land/pkg/gnoweb/markdown/mathml/b7-lines-reach-catalog-infix-over-scope_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b7-lines-reach-catalog-infix-over-scope_test.go
go test -run TestB7InfixOverScope -v ./gno.land/pkg/gnoweb/markdown/mathml/
```
All five shapes fail. See also `tests/judge-32-infix-scope_test.go`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/parse.go:389 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L389) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L389) · Warning
The space-merge loop reads `n.Children[j].Tok` with no nil check, and an empty group `{}` stays a nil sibling.

Gnoweb then shows raw TeX for `a\,{}` and `T^{\mu}\,{}_{\nu}`, since both panic.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_j34_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestJudge34SpaceNil -v
package mathml

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestJudge34SpaceNil(t *testing.T) {
	for _, tex := range []string{`a\,{}`, `a{}`, `a\,{b}`, `a\,\,{}`, `T^{\mu}\,{}_{\nu}`} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					site := ""
					for _, l := range strings.Split(string(debug.Stack()), "\n") {
						if strings.Contains(l, "mathml/parse.go") {
							site = strings.TrimSpace(l)
							break
						}
					}
					t.Errorf("%-20s panic %v at %s", tex, r, site)
				}
			}()
			c := NewMathMLConverter()
			toks, _ := tokenize([]rune(tex))
			c.ParseTex(NewTokenBuffer(toks), ctxRoot)
			t.Logf("%-20s ok", tex)
		}()
	}
}
```

The three inputs with a space before `{}` panic at parse.go:389, and the controls `a{}` and `a\,{b}` pass.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/parse.go:478 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L478) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L478) · Warning
The script base is always the previous sibling, `n.Children[i-1]`, even when that sibling is a `&` or `\\` separator.

`a & ^2 b` collapses into one cell, and `a \\ _1 b` folds two rows into one.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b7_cellscript_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestB7CellStartScript -v
package mathml

import (
	"regexp"
	"strings"
	"testing"
)

func TestB7CellStartScript(t *testing.T) {
	for _, c := range []struct {
		tex      string
		mtr, mtd int
	}{
		{`\begin{matrix} a & ^2 b \\ c & d \end{matrix}`, 2, 4},
		{`\begin{matrix} a \\ _1 b \end{matrix}`, 2, 2},
	} {
		out, err := NewMathMLConverter().ConvertInline(c.tex)
		out = regexp.MustCompile(`(?s)<annotation.*</annotation>`).ReplaceAllString(out, "")
		if err != nil || strings.Count(out, "<mtr") != c.mtr || strings.Count(out, "<mtd") != c.mtd {
			t.Errorf("%q: want %d rows %d cells, got %d rows %d cells: %s", c.tex, c.mtr, c.mtd, strings.Count(out, "<mtr"), strings.Count(out, "<mtd"), out)
		}
	}
}
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:1346 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L1346) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L1346) · Warning
`\eth` maps to `ƪ`, U+01AA, where amssymb's `\eth` is `ð`, the glyph this table already gives [`\dh` at :3343](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3343).

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b11_eth_test.go
// go test -run TestB11Eth -v ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"strings"
	"testing"
)

func TestB11Eth(t *testing.T) {
	out, err := NewMathMLConverter().ConvertInline(`\eth`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out[:strings.Index(out, "<annotation")], "ð") {
		t.Errorf(`\eth: want ð (U+00F0), got %s`, out)
	}
}
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:1537 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L1537) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L1537) · Warning
`hspace` is a zero-argument symbol, so `a \hspace{1em} b` parses `{1em}` as math and shows `1em` on the page, with no error.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_judge63_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestJudge63 -v -count=1
package mathml

import (
	"strings"
	"testing"
)

func TestJudge63HspaceLength(t *testing.T) {
	c := NewMathMLConverter()
	for _, tex := range []string{`a \hspace{1em} b`, `a \hspace{2cm} b`, `x\hspace{0.5em}y`} {
		out, err := c.ConvertInline(tex)
		t.Logf("%s -> err=%v\n%s", tex, err, out)
		if strings.Contains(out, "<mi>e</mi><mi>m</mi>") || strings.Contains(out, "<mi>c</mi><mi>m</mi>") {
			t.Errorf("%s: length argument rendered as math letters", tex)
		}
	}
}
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:2983 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L2983) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L2983) · Warning
`upharpoonleft` gets `↾`, U+21BE with its barb to the right, and [`upharpoonright` at :2989](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L2988-L2989) gets `↿`, so each draws the other's arrow.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b11_harpoon_test.go
// go test -run TestB11HarpoonSwap -v ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"strings"
	"testing"
)

func TestB11HarpoonSwap(t *testing.T) {
	for _, c := range []struct{ tex, want string }{
		{`\upharpoonleft`, "↿"},
		{`\upharpoonright`, "↾"},
		{`\downharpoonleft`, "⇃"},  // control
		{`\downharpoonright`, "⇂"}, // control
	} {
		out, err := NewMathMLConverter().ConvertInline(c.tex)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out[:strings.Index(out, "<annotation")], c.want) {
			t.Errorf("%s: want %q", c.tex, c.want)
		}
	}
}
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:3359 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3359) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3359) · Warning
`\l` maps to capital `Ł`, the glyph [`\L` at :3288](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3288) also gets, while every other pair in this block keeps its case: `\o` gives `ø`.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b11_lstroke_test.go
// go test -run TestB11LStroke -v ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"strings"
	"testing"
)

func TestB11LStroke(t *testing.T) {
	for _, c := range []struct{ tex, want string }{{`\l`, "ł"}, {`\o`, "ø"}} {
		out, err := NewMathMLConverter().ConvertInline(c.tex)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out[:strings.Index(out, "<annotation")], c.want) {
			t.Errorf("%s: want %s", c.tex, c.want)
		}
	}
}
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:3434 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3434) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3434) · Warning
`underbar` is a nullary symbol whose char is the placeholder `X`, so `\underbar{x}` renders a stray `X`.

`id` and `data` likewise render an `x` the author never typed.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b11_placeholder_test.go
// go test -run TestB11Placeholders -v ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"strings"
	"testing"
)

func TestB11Placeholders(t *testing.T) {
	for _, c := range []struct{ tex, bad string }{
		{`\underbar{x}`, "<mi>X</mi>"},
		{`f = \id`, "<mi>x</mi>"},
		{`\data`, "<mi>x</mi>"},
	} {
		out, err := NewMathMLConverter().ConvertInline(c.tex)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out[:strings.Index(out, "<annotation")], c.bad) {
			t.Errorf("%s: placeholder %s in output", c.tex, c.bad)
		}
	}
}
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:3544 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3544) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3544) · Warning
`symbolTable["pmb"] = symbolTable["mu"]` aliases `pmb` to `mu`, and nothing else in the package handles `pmb`, so `\pmb{x}`, the amsmath bold command, renders a stray `μ` before a plain `x`.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b11_pmb_test.go
// go test -run TestB11Pmb -v ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"strings"
	"testing"
)

func TestB11Pmb(t *testing.T) {
	out, err := NewMathMLConverter().ConvertInline(`\pmb{x}`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out[:strings.Index(out, "<annotation")], "μ") {
		t.Errorf(`\pmb{x}: renders a μ that is not in the source`)
	}
}
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:176 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L176) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L176) · Warning
The escaped `|` keeps the rune `|`, so `\|v\|_2` and `\left\| x \right\|` render the absolute-value bar instead of the norm's U+2016, while `\Vert` renders U+2016.

<details><summary>repro</summary>

Expect a U+2016 `<mo>`; the output is `<mo stretchy="true" symmetric="true">|</mo>`. The existing [`double_vertical_bars_fence`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go#L2257) case asserts only `stretchy`.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/judge-7-norm-double-bar_test.go gno.land/pkg/gnoweb/markdown/mathml/zz_norm_double_bar_test.go
go test ./gno.land/pkg/gnoweb/markdown/mathml -run TestB3Repro/norm_glyph -v -count=1
rm gno.land/pkg/gnoweb/markdown/mathml/zz_norm_double_bar_test.go
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:188 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L188) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L188) · Warning
Matching `\\` against `char_reserved` makes the line break an escaped backslash.

`a = 1 \\ b = 2` outside an environment then renders on one line, with a visible `<mo>\</mo>`.

<details><summary>repro</summary>

Expect no `<mo>\</mo>`; it is present. A panic planted in the linebreak branch leaves the golden suite green.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/judge-8-linebreak-literal-backslash_test.go gno.land/pkg/gnoweb/markdown/mathml/zz_linebreak_test.go
go test ./gno.land/pkg/gnoweb/markdown/mathml -run TestB3Repro/linebreak_display -v -count=1
rm gno.land/pkg/gnoweb/markdown/mathml/zz_linebreak_test.go
```
</details>

## gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:418 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L418) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L418) · Warning
`GetNextN` skips comment tokens after its bounds check and still advances by `n`, so a `%` comment after an opener slides the [`parse.go:210`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L210) and [`:233`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L233) window past its closer.

A trailing `x` after `\end{matrix}` then renders inside the last cell, and `\left(` with `\right)` last fails the whole conversion.

<details><summary>repro</summary>

Each comment slides the window one token, since both lines size it by `MatchOffset`, which counts the comment. A trailing `.` lands in the last cell the same way.

Both subtests fail.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/b8-lines-reach-catalog-getnextn-comment_test.go gno.land/pkg/gnoweb/markdown/zz_getnextn_comment_test.go
go test -run TestGetNextNCommentShift -v -count=1 ./gno.land/pkg/gnoweb/markdown
rm gno.land/pkg/gnoweb/markdown/zz_getnextn_comment_test.go
```

Each pair below renders differently with the comment than without it:

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_b7_envcomment_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestB7EnvCommentShift -v
package mathml

import (
	"regexp"
	"testing"
)

func render(tex string) (string, error) {
	out, err := NewMathMLConverter().ConvertInline(tex)
	return regexp.MustCompile(`(?s)<annotation.*</annotation>`).ReplaceAllString(out, ""), err
}

func TestB7EnvCommentShift(t *testing.T) {
	pairs := [][2]string{
		{"\\begin{pmatrix}%c\na & b\\end{pmatrix}.", "\\begin{pmatrix}a & b\\end{pmatrix}."},
		{"\\left( \\begin{matrix}%c\na & b\\end{matrix}\\right)", "\\left( \\begin{matrix}a & b\\end{matrix}\\right)"},
		{"\\begin{matrix}%c\na\\end{matrix}", "\\begin{matrix}a\\end{matrix}"},
	}
	for _, p := range pairs {
		got, err1 := render(p[0])
		want, err2 := render(p[1])
		if err1 != nil || err2 != nil || got != want {
			t.Errorf("a %% comment changed the rendering of %q\n  with comment: %s\n  without:      %s", p[1], got, want)
		}
	}
}
```

`tests/judge-41-getnextn-comment_test.go` runs the same cases from the `mathml` package as `TestJudge41`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:616 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L616) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L616) · Warning
`toks[i+1]` is the whitespace token in `\left (`, which TeX reads as `\left(`.

The fence bits land on the space, so the parenthesis does not stretch, and `\big (` loses its size.

<details><summary>repro</summary>

`left-space` and `big-space` fail.

```bash
cp reviews/pr/4xxx/4879-gnoweb-math-extension/3-0b3987540/tests/judge-42-left-space_test.go gno.land/pkg/gnoweb/markdown/zz_left_space_test.go
go test -run TestLeftSpaceDelimiter -v -count=1 ./gno.land/pkg/gnoweb/markdown
rm gno.land/pkg/gnoweb/markdown/zz_left_space_test.go
```
</details>

## SKIP gno.land/pkg/gnoweb/markdown/mathml/variant_transform.go:29-56 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/variant_transform.go#L29-L56) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/variant_transform.go#L29) · Warning
The `isolateMathVariant(context)` switch has no case for nested font commands such as `Frak|Bold` or `Bb|Bold`, and no default.

`\mathbf{\mathfrak{g}}` drops both fonts and renders a bare italic letter.

<details><summary>repro</summary>

```go
// cp to gno.land/pkg/gnoweb/markdown/mathml/zz_variant_nested_fonts_test.go
// go test ./gno.land/pkg/gnoweb/markdown/mathml/ -run TestNestedFontCommandsKeepAVariant -v
package mathml

import (
	"regexp"
	"testing"
)

func TestNestedFontCommandsKeepAVariant(t *testing.T) {
	bare := regexp.MustCompile(`<mi>[A-Za-z]</mi>`)
	for _, tex := range []string{
		`\mathbf{\mathfrak{g}}`,
		`\mathbf{\mathcal{A}}`,
		`\mathbb{\mathbf{R}}`,
		`\mathtt{\mathbf{x}}`,
		`\mathrm{\mathbf{x}}`,
		`\mathbf{\mathrm{x}}`,
	} {
		out, err := NewMathMLConverter().ConvertInline(tex)
		if err != nil {
			t.Fatalf("%s: %v", tex, err)
		}
		if bare.MatchString(out) {
			t.Errorf("%s: both variants dropped, got %s", tex, bare.FindString(out))
		}
	}
}
```

Every case fails with a bare `<mi>g</mi>`, `<mi>A</mi>`, `<mi>R</mi>` or `<mi>x</mi>`.
</details>

Not posted: the `commands.go:249` section carries this case and its repro, and one fix closes both.

## SKIP .github/workflows/ci-race.yml:62 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/.github/workflows/ci-race.yml#L62) · [↗](../../../../../.worktrees/gno-review-4879/.github/workflows/ci-race.yml#L62) · Missing test
Missing test: the race step renders only two one-line inputs.

The block parser, the math fence and the footnote path never run under `-race`.

Not posted: no judge ran the check, which adds a `$$` block and a math fence to `TestMathConcurrentRender` and plants a field write in `texBlockRegionParser.Open`.

## SKIP gno.land/pkg/gnoweb/markdown/golden/ext_math/minimal_test.txtar:2 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/golden/ext_math/minimal_test.txtar#L2) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/golden/ext_math/minimal_test.txtar#L2) · Missing test
Missing test: `minimal_test.txtar` and `escaped_test.txtar` are byte-identical.

No golden covers the documented `\\(\pi r^2\\)` form.

Not posted: no judge ran the `cmp` and grep that settle it.

## SKIP gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go:883 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go#L883) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml_test.go#L883) · Missing test
Missing test: 52 of the 84 test functions in `mathml_test.go` assert nothing.

The render tests discard their result, and four command tests recover and discard any panic.

Not posted: no judge ran the mutation of `render` that would show these tests stay green.

## SKIP gno.land/pkg/gnoweb/markdown/mathml/mmlnode_test.go:324 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode_test.go#L324) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode_test.go#L324) · Missing test
Missing test: the deep case of `TestNestedSizeIsBounded` nests 40 `\Huge` groups, past the parse depth limit.

No size switch is ever written, so the case passes without reaching its bound.

Not posted: no judge ran `TestB6NestedSizeDeepCaseReachesSizes`.

## SKIP gno.land/pkg/gnoweb/markdown/mathml/symbols.go:2982 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L2982) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L2982) · Missing test
Missing test: no test pins any `symbolTable` glyph, so the harpoon swap, `\l`, `\pmb` and the placeholder entries all pass the package suite.

Not posted: no judge ran the table test that would redden on the harpoon pair.

## examples/quarantined/gno.land/r/docs/markdown/markdown.gno:1030 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/examples/quarantined/gno.land/r/docs/markdown/markdown.gno?plain=1#L1030) · [↗](../../../../../.worktrees/gno-review-4879/examples/quarantined/gno.land/r/docs/markdown/markdown.gno#L1030) · Nit
Nit: `\color` here is a switch that colours everything to the end of the group, so this example renders the `+` red and the `=` blue.

<details><summary>observed</summary>

Rendering the docs realm through `NewDefaultRenderConfig` puts the `+` inside the red `<mstyle>` and the `=` inside the blue one (`TestB1DocsRealm`).
</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:354 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/ext_math.go#L354) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/ext_math.go#L354) · Nit
Nit: the `parent.(*mathInlineNode)` guard can never be true.

A block parser's parent is always a block node, and the only `mathInlineNode` is built after block parsing ends.

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:272 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L272) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L272) · Nit
Nit: the scan counts from `b.idx` with comment tokens included, while `GetNextN` skips them.

A `%` comment after a style switch in a table cell merges two cells.

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:285 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L285) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L285) · Nit
Nit: the `\color` branch reads its spec with `GetNextExpr`, which rejects anything but `{`.

`\color[RGB]{255,0,0}{x}` shows an error box and prints the values as math.

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:349 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L349) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L349) · Nit
Nit: the fallback `merror` takes `tok.Value`, which carries no backslash, and sets no `title`, so `\unknowncommand` shows as `unknowncommand` in a box that reads like prose.

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:352 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L352) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L352) · Nit
Nit: `set_variants_from_context` also runs on the unknown-command `merror`, so `\mathbf{\foo}` shows `𝐟𝐨𝐨`, which the author cannot search the source for.

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:466 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L466) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L466) · Nit
Nit: `newCommand` never reads the `[n]` argument count, so `GetNextN(1, true)` takes `[` as the definition.

`\newcommand{\f}[1]{#1^2}` leaks `1]` into the formula.

## gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go:182 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L182) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands_defs.go#L182) · Nit
Nit: `getScripts` never pads a repeated script after the last pair.

`\sideset{_a_b}{}` panics, and `\sideset{^a^b}{}` drops `b` with no error.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB5SidesetRepeatedScript ./gno.land/pkg/gnoweb/markdown/mathml/
```
with `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` as `zz_b5_test.go`. See also `tests/judge-26-sideset-repeated-script_test.go`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/environnements.go:87 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L87) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L87) · Nit
Nit: each `|` appends its own `columnlines` entry.

`c||cc` yields `solid solid none`, so it draws a rule between the second and third columns.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB5DoubleRule ./gno.land/pkg/gnoweb/markdown/mathml/
```
with `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` as `zz_b5_test.go`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/environnements.go:156 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L156) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/environnements.go#L156) · Nit
Nit: `rowspacing` is read from any direct child of a cell, not only the `\\[len]` marker.

A `\substack` in a matrix cell passes its `rowspacing="0"` to the outer row.

<details><summary>repro</summary>

```
go test -count=1 -v -run TestB5RowspacingLeak ./gno.land/pkg/gnoweb/markdown/mathml/
```
with `tests/b5-lines-reach-rollout-catalog-defs-envs_test.go` as `zz_b5_test.go`.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/mathml.go:48 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L48) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L48) · Nit
Nit: the `recover` in `render` replaces every panic value with a fixed message, `errMaxDepth` included.

No caller can tell a depth rejection from a nil dereference.

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:94 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L94) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L94) · Nit
Nit: `UnsetAttr` and `AddProps` are exported, but the only callers are their own tests in `mathml_test.go`, so no rendering path uses them.

## gno.land/pkg/gnoweb/markdown/mathml/parse.go:212 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L212) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L212) · Nit
Nit: this `tokOpen|tokCurly` case never runs, since `GetNextToken` returns an error for every unescaped `{`.

`tokBadmacro`, `propScriptBase` and `ctxBracketed` are likewise never set or read.

## gno.land/pkg/gnoweb/markdown/mathml/parse.go:247 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L247) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/parse.go#L247) · Nit
Nit: `tokNull` is zero, so `tok.Kind&tokNull > 0` never fires.

`\left.` and `\right.` emit an empty stretchy `<mo>` where they should emit nothing.

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:3289 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3289) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3289) · Nit
Nit: `Mu` lacks `propSymUpright`, as does [`Nu`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3294), so both render italic.

Every other capital Greek letter is upright, and `greek_symbols_test.txtar` pins the italic.

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:3333 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3333) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3333) · Nit
Nit: `makeSymbol` renders every entry of the [block at 3285-3450](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3285-L3450) as `<mi>`, since none sets a `kind`.

`\coloneqq` and `\dashrightarrow` therefore lose the relation spacing of `\leq`.

## gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:543 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L543) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L543) · Nit
Nit: `(mate.Kind&t.Kind)&kind > 0` is always true.

Every caller passes a single bit, and only tokens carrying `kind` reach it.

## gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:624 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L624) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L624) · Nit
Nit: `tokNull` is zero, so `temp.Kind = tokNull` sets no mark, and no guard in `parse.go` can fire on it.

## gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:689 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L689) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L689) · Nit
Nit: `case "begin":` tests `toks[i].Value` for `begin` and `end` without checking the token is a command.

A trailing comment `x %end` sends the whole expression back to raw TeX.

## gno.land/pkg/gnoweb/markdown/mathml/variant_transform.go:379 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/variant_transform.go#L379) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/variant_transform.go#L379) · Nit
Nit: four tables are unreachable: `initial`, `looped`, `stretched` and `tailed`.

`set_variants_from_context`, the only caller of `transformByVariant`, never assigns any of the four names.

<details><summary>check</summary>

```
grep -rn '"initial"\|"looped"\|"stretched"\|"tailed"' gno.land/pkg/gnoweb/markdown --include='*.go'
```
The grep hits only `variant_transform.go`, and `transformByVariant` has one call.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/commands.go:155 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L155) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L155) · Suggestion
Suggestion: [`command_args`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/commands.go#L141) has no `\operatorname`, `\phantom`, `\kern`, `\mkern`, `\tag`, `\label` or `\eqref`, so each renders an error box and then its argument as math.

Add them here, or list them as unsupported in the docs realm.

## gno.land/pkg/gnoweb/markdown/mathml/mathml.go:54 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L54) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mathml.go#L54) · Suggestion
Suggestion: `return "", err` hands back the brace errors from `matchBracesCritical` unchanged, and those errors wrap the user's raw TeX, unescaped, in a literal `<pre>`.

Drop the `<pre>` wrapper, or escape the context, before a caller logs or displays one.

## gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go:71 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L71) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/mmlnode.go#L71) · Suggestion
Suggestion: the `i > 2` guard lets a third argument index past the end of `tagText`, so `NewMMLNode` panics on it; no caller passes three arguments today.

Name the two parameters, and the limit becomes part of the signature.

<details><summary>repro</summary>

`TestB6NewMMLNodeExtraArgs` in `tests/b6-lines-reach-catalog-empty-args_test.go` panics with index out of range.
</details>

## gno.land/pkg/gnoweb/markdown/mathml/structures.go:3 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/structures.go#L3) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/structures.go#L3) · Suggestion
Suggestion: this generic stack has two users, the `newStack[int]()` calls ([tokenize.go:521](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L521), [:568](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/tokenize.go#L568)), and `append` on a local `[]int` slice covers both.

## gno.land/pkg/gnoweb/markdown/mathml/symbols.go:3502 [gh](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3502) · [↗](../../../../../.worktrees/gno-review-4879/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L3502) · Suggestion
Suggestion: `\dotsc`, `\dotsb` and `\>` render an error box, since `symbolTable["dotso"] = symbolTable["dots"]` aliases only `dotso`, and [`space_widths`](https://github.com/alexiscolin/gno/blob/0b3987540842f0f7a875e2a344d5995fd47f40d5/gno.land/pkg/gnoweb/markdown/mathml/symbols.go#L21) lacks `>`.

Both take one line each: alias `dotsc` and `dotsb` to `dots`, and add `>` to `space_widths`.
