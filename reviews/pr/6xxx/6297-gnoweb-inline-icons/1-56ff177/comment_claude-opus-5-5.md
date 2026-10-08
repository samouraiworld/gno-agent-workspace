# Review: [#6297](https://github.com/gnolang/gno/pull/6297)

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. The generated icon table strokes 55 glyphs their source leaves unstroked, and an empty heading next to any icon heading yields a duplicate id; both ship with the branch.
Model: claude-opus-5-5, standard review (finders xhigh, the other stages high)
Commit: 56ff1770722eee73f51f207f97554ea1e2ea2b00
Overview: [overview.md](../overview.md)
Open the code: [56ff177](https://github.com/gnolang/gno/tree/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown)
Round: 1. 5 finders, one reflector, 16 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 1 refuted, none of them above Nit.

## Body

- Suggestion: a heading holding only a labeled icon gets no TOC entry. [`writeNodeText`](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/utils.go#L197-L208) writes only text nodes, and [`toc.go`](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/toc.go#L91-L96) lets the next heading overwrite an untitled entry. I think the icon's label belongs in the TOC title.

  > A realm page holding `## Intro`, `## <gno-icon name="star" label="Top" />` and `## Next`, rendered by gnoweb at this head in headless Chromium from a `MockClient` fixture: Contents lists Intro and Next only.

  ![Realm page with three h2 headings, the middle one a star icon; the Contents rail lists Intro and Next](https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6297-gnoweb-inline-icons/1-56ff177/media/icon-heading-toc.png)

## gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go:203 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go#L203) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go#L203) · Warning

The generated table draws a 1.3 outline on fills and dots that 55 source icons leave unstroked, battery and lock among them. `iconHeadStroke` hands `stroke="currentColor"` down, and `addIconSymbols` never writes `stroke="none"` back onto those shapes.

> Seven of the 55, at reading size and at 96px: gnoweb's `<svg>` for each `<gno-icon />` at this head beside its `vendored.svg` symbol unchanged, in headless Chromium from a `MockClient` fixture.

![Seven icons, generated beside source, at reading size and at 96px: the battery fills and the lock, wallet and menu dots are heavier in the generated set, and the calculator keypad is one solid block](https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6297-gnoweb-inline-icons/1-56ff177/media/stroke-head-icons.png)

<details><summary>repro</summary>

Copy the round's `tests/b2-lines-reach-catalog-stroke-head.go` into `gno.land/pkg/gnoweb/markdown/b2_stroke_head_test.go`, then `go test -count=1 -run TestB2StrokeHead -v ./gno.land/pkg/gnoweb/markdown`.

Both tests fail; the shapes the source leaves unstroked come out stroked:

```
source strokes=[none none none] registry strokes=[currentColor currentColor currentColor]
--- FAIL: TestB2StrokeHeadMinimal
55 icons, 108 shapes: unstroked in the source, stroked in the registry: battery battery-75 battery-half battery-low calculator ... lock menu-horizontal menu-vertical ...
--- FAIL: TestB2StrokeHeadVendored
```

`vendored.svg` `ico-menu-horizontal` is `<g fill="currentColor"><circle r="1"/>` three times with no stroke; `icons_gen.go:307` emits the same body under `iconHeadStroke`, so each dot draws at radius 1.65. Counted per glyph, 58 of 491 differ: the 55 above plus 3 whose caps and joins turn from butt/miter to round. `TestIconTable` stays green because it compares the generator with its own output.

</details>

## SKIP gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go:202 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go#L202) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go#L202) · Warning

Of 491 generated glyphs, 58 paint a shape unlike their source. The battery bar gains an outline, and keypad, lock and wallet dots thicken.

Not posted: the same root cause as the section on line 203, closed by the same generator edit.

## gno.land/pkg/gnoweb/markdown/ext_icons.go:190 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L190) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons.go#L190) · Warning

An empty `##` fails `h.Lines().Len() > 0`, so its id `heading` never enters the fresh `newLinearIDs()`. On a page with an icon heading, the next `heading` slug repeats that id, and its TOC link jumps to the empty one.

<details><summary>repro</summary>

Save as `gno.land/pkg/gnoweb/markdown/zz_dupid_test.go`, then `go test -count=1 -run TestEmptyHeadingDuplicateID -v ./gno.land/pkg/gnoweb/markdown`.

```go
package markdown

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/yuin/goldmark/parser"
)

var dupHeadingID = regexp.MustCompile(`<h[1-6] id="([^"]*)"`)

func TestEmptyHeadingDuplicateID(t *testing.T) {
	m := newProductionLikeMarkdown()
	for _, src := range []string{
		"##\n\n## Heading\n", // control: no icon anywhere
		"##\n\n## Heading\n\n## <gno-icon name=\"star\" /> A\n",
		"##\n\n## <gno-icon name=\"star\" label=\"Top\" />\n",
	} {
		var buf bytes.Buffer
		if err := m.Convert([]byte(src), &buf, parser.WithContext(NewGnoParserContext(GnoContext{}))); err != nil {
			t.Fatal(err)
		}
		var ids []string
		seen := map[string]int{}
		for _, mm := range dupHeadingID.FindAllStringSubmatch(buf.String(), -1) {
			ids = append(ids, mm[1])
			seen[mm[1]]++
		}
		t.Logf("src=%q ids=%v", src, ids)
		for id, n := range seen {
			if n > 1 {
				t.Errorf("duplicate heading id %q (%d times) for %q", id, n, src)
			}
		}
	}
}
```

Without an icon the ids stay unique; one icon anywhere on the page duplicates `heading`:

```
src="##\n\n## Heading\n" ids=[heading heading-1]
src="##\n\n## Heading\n\n## <gno-icon name=\"star\" /> A\n" ids=[heading heading a]
    duplicate heading id "heading" (2 times)
src="##\n\n## <gno-icon name=\"star\" label=\"Top\" />\n" ids=[heading heading]
    duplicate heading id "heading" (2 times)
--- FAIL: TestEmptyHeadingDuplicateID
```

</details>

## SKIP gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go:36 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go#L36) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go#L36) · Missing test

Missing test: no test ties `icons/icons.txt` to `icons/vendored.svg`, so an `icons.txt` edit made without `make icons` passes with the old icon set.

Not posted: PLAUSIBLE on the finder's read; the mutation that removes one `icons.txt` entry and runs `TestIconTable` and `TestIconCatalog` was not run.

## SKIP gno.land/pkg/gnoweb/markdown/utils.go:93 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/utils.go#L93) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/utils.go#L93) · Missing test

Missing test: no case reaches the end-of-line guard after `=` in `scanGnoTag`, so deleting it panics the inline parser with every test green; #6298 and #6299 carry the same lines.

Not posted: PLAUSIBLE on the finder's read; the mutation deleting lines 93-95 and the two `TestScanGnoTag` cases `<gno-x a=` and `<gno-x a=  \n` were not run.

## gno.land/adr/prxxxx_gnoweb_inline_icons.md:83 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/adr/prxxxx_gnoweb_inline_icons.md#L83) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/adr/prxxxx_gnoweb_inline_icons.md#L83) · Nit

Nit: the per-icon size leaves out the label, where [`renderIcon`](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L255-L257) escapes each quote to six bytes. A label of quotes inside the tag bound roughly doubles an icon's output.

## gno.land/adr/prxxxx_gnoweb_inline_icons.md:100 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/adr/prxxxx_gnoweb_inline_icons.md#L100) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/adr/prxxxx_gnoweb_inline_icons.md#L100) · Nit

Nit: `<gno-button />` registers no parser in this tree, though this ADR line says it registers the same one. [`newGnoTagLineParser`](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L338) has one caller, and the [`utils.go` comment](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/utils.go#L53) says the same, as in #6298 and #6299.

## gno.land/adr/prxxxx_gnoweb_inline_icons.md:104 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/adr/prxxxx_gnoweb_inline_icons.md#L104) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/adr/prxxxx_gnoweb_inline_icons.md#L104) · Nit

Nit: lines 103-104 justify the line parser by a `sanitize.Block` that stopped escaping the `<gno-…>` opener. That change is absent from this tree: `gnovm/stdlibs/chain/markdown` matches master, and [line 216](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/adr/prxxxx_gnoweb_inline_icons.md#L216) calls the sanitizer unchanged.

## gno.land/pkg/gnoweb/markdown/ext_icons.go:105 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L105) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons.go#L105) · Nit

Nit: `bytes.TrimSpace` runs on the raw label, so `label="&#32;"` stays non-empty, renders `role="img"` with a blank `aria-label`, and suppresses the alone-in-link hint.

## gno.land/pkg/gnoweb/markdown/ext_icons.go:236 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L236) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons.go#L236) · Nit

Nit: `<gno-icon name=star/>` reads the name as `star/` and ends on a plain `>`, so the `write it self-closing` comment asks for the `/>` the author already wrote.

## gno.land/pkg/gnoweb/markdown/ext_icons.go:298 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L298) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons.go#L298) · Nit

Nit: the `aloneInNamedParent` walk marks every icon in the heading as `hintDone` but sets `hint` on the first alone. A link icon after it then gets no add-label hint.

<details><summary>repro</summary>

Copy `tests/b3-lines-reach-catalog-link-hint-lost.go` into `gno.land/pkg/gnoweb/markdown/zz_hint_test.go`, then `go test -count=1 -run TestB3LinkHintLostInsideHeading -v ./gno.land/pkg/gnoweb/markdown`.

The control `## Title [<gno-icon name="star" />](/r/x)` gets its hint; with an earlier icon in the heading it does not:

```
an icon-only link inside a heading got no hint: "## <gno-icon name=\"star\" /> Title [<gno-icon name=\"star\" />](/r/x)\n"
--- FAIL: TestB3LinkHintLostInsideHeading
```

</details>

## SKIP gno.land/pkg/gnoweb/markdown/ext_icons.go:287 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L287) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons.go#L287) · Nit

Nit: the `if n.hintDone` early return hands a nested link's icon the cached `hint`, so it never gets its own add-label hint.

Not posted: the same defect as the section on line 298, closed by the same edit to the walk.

## gno.land/pkg/gnoweb/markdown/ext_icons_test.go:170 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons_test.go#L170) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons_test.go#L170) · Test

Test: the cut span ends with the `aria-label` value, so `label="turn on"` matches `" on"` and `FuzzIconRender` reports an event handler on a safe render.

## gno.land/pkg/gnoweb/markdown/golden/ext_icons/nested_emphasis.md.txtar:7 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/golden/ext_icons/nested_emphasis.md.txtar#L7) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/golden/ext_icons/nested_emphasis.md.txtar#L7) · Test

Test: the golden runner registers no strikethrough extension, so this line pins literal `~~` around the icon; only `TestIconInGFM` covers the `<del>` production renders.

## gno.land/pkg/gnoweb/tools/cmd/iconset/main.go:122 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/tools/cmd/iconset/main.go#L122) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/tools/cmd/iconset/main.go#L122) · Nit

Nit: `http.Get` uses `http.DefaultClient`, which has no timeout, so a stalled connection to codeload hangs `make icons` instead of failing it.

## gno.land/pkg/gnoweb/markdown/ext_icons.go:182 [gh](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L182) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons.go#L182) · Suggestion

Suggestion: this fresh `newLinearIDs()` replaces a caller's `parser.WithIDs` generator. The `h.AttributeString("id")` check also overwrites an explicit `{#id}`, though no renderer here combines `parser.WithAttribute` with this extension.
