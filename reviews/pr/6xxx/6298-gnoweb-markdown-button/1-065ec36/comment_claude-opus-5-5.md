# Review: [#6298](https://github.com/gnolang/gno/pull/6298)
Posted: https://github.com/gnolang/gno/pull/6298#pullrequestreview-5456857517

Event: COMMENT
Verdict: REQUEST CHANGES. Two Warnings come from the branch: gnoweb turns `<gno-button>` on before any chain runs the native escape that guards sanitized text, and the in-line escape breaks a pointy link so its brackets bind to the realm's own reference definitions.
Model: claude-opus-5-5, standard review
Commit: 065ec369b321c207815f82b0c1dbf2b0f9f10b01
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6298 065ec369b`
Round: 1. 16 finders, one reflector, 14 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted, none of them above Nit.

## Body

> AI review, claude-opus-5-5, standard review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6298-gnoweb-markdown-button/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6298-gnoweb-markdown-button/1-065ec36/claims.md) · Status: REQUEST CHANGES

## gno.land/pkg/gnoweb/markdown/ext.go:96 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/ext.go#L96) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext.go#L96) · Warning [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151639)

`ExtButtons.Extend(m)` turns buttons on in every gnoweb, so user text renders as a realm-styled button until the chain runs this branch's [chain/markdown native](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gnovm/stdlibs/chain/markdown/markdown.go#L427).

<details><summary>observed output</summary>

The chain/markdown native on `chain/mainnet` is byte-identical to the merge base's. With that native swapped in and the sanitize goldens regenerated through this branch's gnoweb, 28 of 35 goldens render a live button:

```
<p>hi <a href="/r/evil$help&amp;func=Drain" class="gno-button gno-button-caution">Claim...
```

Realms already call `sanitize.Block` on user text, commondao's `renderBylaws` for one.

</details>

## gnovm/stdlibs/chain/markdown/markdown.go:427 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gnovm/stdlibs/chain/markdown/markdown.go#L427) · [↗](../../../../../.worktrees/gno-review-6298/gnovm/stdlibs/chain/markdown/markdown.go#L427) · Warning [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151651)

`escapeGnoButtonTags(line)` puts a backslash inside the pointy link `[evil](<gno-button x>)`, so goldmark drops the link and `[evil]` binds to the realm's own `[evil]:` reference or footnote.

<details><summary>observed output</summary>

`Block` and `BlockRich` on `[evil](<gno-button x>)`, rendered beside a realm line `[evil]: https://realm.example/home`:

```
out="\n\n[evil](\\<gno-button x>)\n\n"
<p><a href="https://realm.example/home" ...>evil</a>(&lt;gno-button x&gt;)</p>
bound-to-realm-ref=true idempotent=false   (8 of 8 cases)
```

The raw input binds to nothing, and the merge base's native keeps all 8 cases unbound and idempotent.

</details>

## gno.land/adr/prxxxx_gnoweb_button.md:227-230 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/adr/prxxxx_gnoweb_button.md?plain=1#L227-L230) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/adr/prxxxx_gnoweb_button.md#L227) · Nit [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151658)

Nit: this guarantee needs a chain already running the new native. The ADR names neither that ordering nor the change to `Block` and `BlockRich` output on chain.

## gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3270 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3270) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3270) · Nit [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151668)

Nit: `margin: 0` here loses to `.c-realm-view a > .tooltip:last-of-type`, whose specificity is higher. The icon inside every button keeps a 0.2em right margin.

## gno.land/pkg/gnoweb/markdown/ext_buttons.go:72 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L72) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L72) · Nit [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151679)

Nit: `len(t.href) == 0` tests the raw attribute bytes, so an href that is blank only after decoding, `href="&#32;"`, passes and renders a button linking to `%20`.

## gno.land/pkg/gnoweb/markdown/ext_buttons.go:96 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L96) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L96) · Nit [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151689)

Nit: `html.UnescapeString` decodes with text rules, not the attribute rules the godoc promises. A browser keeps `&not=` as written, and this call decodes it.

## gno.land/pkg/gnoweb/markdown/ext_buttons.go:97 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L97) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L97) · Nit [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151701)

Nit: the `strings.Map` filter drops only control characters. So `&#x2060;` alone renders a blank button.

<details><summary>observed output</summary>

U+2060, U+061C and U+180E are Unicode format characters, which `unicode.IsControl` and `StripBidiAndZeroWidth` both keep:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6298 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_label_test.go <<'GO'
package markdown

import "testing"

func TestInvisibleLabel(t *testing.T) {
	for _, l := range []string{"&#x2060;", "&#x61C;", "&#x180E;"} {
		if got := buttonLabel([]byte(l)); got != "" {
			t.Errorf("%s: label %q, want empty", l, got)
		}
	}
}
GO
go test ./gno.land/pkg/gnoweb/markdown -run TestInvisibleLabel -count=1
rm gno.land/pkg/gnoweb/markdown/zz_label_test.go
```

The test fails, since each invisible character survives as the whole label:

```
    zz_label_test.go:8: &#x2060;: label "\u2060", want empty
    zz_label_test.go:8: &#x61C;: label "\u061c", want empty
    zz_label_test.go:8: &#x180E;: label "\u180e", want empty
FAIL
```

</details>

## gno.land/pkg/gnoweb/markdown/ext_buttons_test.go:113 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/ext_buttons_test.go#L113) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext_buttons_test.go#L113) · Nit [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151714)

Nit: `benchConvert` discards the error from `weburl.Parse`, unlike every other caller in the package's tests.

## gnovm/stdlibs/chain/markdown/markdown.go:438 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gnovm/stdlibs/chain/markdown/markdown.go#L438) · [↗](../../../../../.worktrees/gno-review-6298/gnovm/stdlibs/chain/markdown/markdown.go#L438) · Nit [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151721)

Nit: this branch leaves the backslash visible and drops the tag text, which the comment above says it avoids. The [four-spaces golden](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/golden/sanitize/block-ext-delim-four-spaces.txtar) records `<p>\    <!-- raw HTML omitted --></p>`.

## gno.land/pkg/gnoweb/markdown/utils.go:105 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/utils.go#L105) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/utils.go#L105) · Suggestion [posted](https://github.com/gnolang/gno/pull/6298#discussion_r4219151731)

Suggestion: this loop stops only at a space or `>`, so each attempt on a line of `<gno-button x=` reads up to `maxButtonTagLen` bytes. I think stopping it at `<` too, as in #6297 and #6299 which carry this file, fits the ADR's one tag per attempt.

## SKIP gno.land/pkg/gnoweb/markdown/ext_buttons_test.go:55 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/ext_buttons_test.go#L55) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/ext_buttons_test.go#L55) · Missing test

Missing test: every shape in `TestButtonHostileBudget` stops at the next `<`, so raising `maxButtonTagLen` to 1 MiB leaves it green.

Not posted: PLAUSIBLE, the mutation was not run by a judge.

## SKIP gno.land/pkg/gnoweb/markdown/golden/ext_buttons/hostile_control_chars.md.txtar:2 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/golden/ext_buttons/hostile_control_chars.md.txtar#L2) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/golden/ext_buttons/hostile_control_chars.md.txtar#L2) · Missing test

Missing test: no golden feeds an href whose control byte leads or that is blank only after decoding.

Not posted: PLAUSIBLE, and the blank-href half already ships on `ext_buttons.go:72`.

## SKIP gno.land/pkg/gnoweb/markdown/golden/ext_buttons/hostile_unterminated_line.md.txtar:2 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/golden/ext_buttons/hostile_unterminated_line.md.txtar#L2) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/golden/ext_buttons/hostile_unterminated_line.md.txtar#L2) · Missing test

Missing test: the hostile-line golden covers only quoted values, and an unquoted value reads past the next `<`.

Not posted: PLAUSIBLE, and the code half ships on `utils.go:105`.

## SKIP gnovm/stdlibs/chain/markdown/markdown.gno:105 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gnovm/stdlibs/chain/markdown/markdown.gno#L105) · [↗](../../../../../.worktrees/gno-review-6298/gnovm/stdlibs/chain/markdown/markdown.gno#L105) · Nit

Nit: the native's doc lists no step for the `<gno-button` escape and still describes a line-start backslash.

Not posted: UNVERIFIED, on the finder's read only, and the line sits outside the diff.

## SKIP gno.land/pkg/gnoweb/markdown/utils.go:214 [gh](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/utils.go#L214) · [↗](../../../../../.worktrees/gno-review-6298/gno.land/pkg/gnoweb/markdown/utils.go#L214) · Nit

Nit: the copy of `utils.go` in #6299 adds `trimTagLine` after this line, so the three copies differ.

Not posted: UNVERIFIED, and it corrects the description rather than the code.
